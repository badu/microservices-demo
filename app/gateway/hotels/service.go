package hotels

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"

	"github.com/badu/microservices-demo/app/gateway/users"
	"github.com/badu/microservices-demo/app/hotels"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Repository interface {
	GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*Hotel, error)
	SetHotel(ctx context.Context, hotel *Hotel) error
	DeleteHotel(ctx context.Context, hotelID uuid.UUID) error
}

type ServiceImpl struct {
	logger            logger.Logger
	repository        Repository
	grpcClientFactory func(ctx context.Context) (*grpc.ClientConn, hotels.HotelsServiceClient, error)
}

func NewService(
	logger logger.Logger,
	repository Repository,
	grpcClientFactory func(ctx context.Context) (*grpc.ClientConn, hotels.HotelsServiceClient, error),
) ServiceImpl {
	return ServiceImpl{logger: logger, repository: repository, grpcClientFactory: grpcClientFactory}
}

func (s *ServiceImpl) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.GetHotelByID")
	defer span.Finish()

	ctxUser, ok := ctx.Value(users.RequestCtxUser{}).(*users.UserResponse)
	if !ok || ctxUser == nil {
		return nil, errors.Wrap(httpErrors.Unauthorized, "ctx.Value user")
	}

	cacheHotel, err := s.repository.GetHotelByID(ctx, hotelID)
	if err != nil {
		if err != redis.Nil {
			s.logger.Errorf("GetHotelByID: %v", err)
		}
	}
	if cacheHotel != nil {
		return cacheHotel, nil
	}

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.GetHotelByID")
	}
	defer conn.Close()

	hotelByID, err := client.GetHotelByID(ctx, &hotels.GetByIDReq{HotelID: hotelID.String()})
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.GetHotelByID")
	}

	fromProto, err := HotelFromProto(hotelByID.GetHotel())
	if err != nil {
		return nil, errors.Wrap(err, "HotelFromProto")
	}

	if err := s.repository.SetHotel(ctx, fromProto); err != nil {
		s.logger.Errorf("SetHotel: %v", err)
	}

	return fromProto, nil
}

func (s *ServiceImpl) UpdateHotel(ctx context.Context, hotel *Hotel) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.UpdateHotel")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.UpdateHotel")
	}
	defer conn.Close()

	hotelRes, err := client.UpdateHotel(ctx, &hotels.UpdateHotelReq{
		HotelID:       hotel.HotelID.String(),
		Name:          hotel.Name,
		Email:         hotel.Email,
		Country:       hotel.Country,
		City:          hotel.City,
		Description:   hotel.Description,
		Location:      hotel.Location,
		Rating:        hotel.Rating,
		Image:         *hotel.Image,
		Photos:        hotel.Photos,
		CommentsCount: int64(hotel.CommentsCount),
		Latitude:      *hotel.Latitude,
		Longitude:     *hotel.Longitude,
	})
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.UpdateHotel")
	}

	fromProto, err := HotelFromProto(hotelRes.GetHotel())
	if err != nil {
		return nil, errors.Wrap(err, "HotelFromProto")
	}

	if err := s.repository.SetHotel(ctx, fromProto); err != nil {
		s.logger.Errorf("SetHotel: %v", err)
	}

	return fromProto, nil
}

func (s *ServiceImpl) CreateHotel(ctx context.Context, hotel *Hotel) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.CreateHotel")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.CreateHotel")
	}
	defer conn.Close()

	hotelRes, err := client.CreateHotel(ctx, &hotels.CreateHotelReq{
		Name:          hotel.Name,
		Email:         hotel.Email,
		Country:       hotel.Country,
		City:          hotel.City,
		Description:   hotel.Description,
		Location:      hotel.Location,
		Rating:        hotel.Rating,
		Image:         *hotel.Image,
		Photos:        hotel.Photos,
		CommentsCount: int64(hotel.CommentsCount),
		Latitude:      *hotel.Latitude,
		Longitude:     *hotel.Longitude,
	})
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.CreateHotel")
	}

	fromProto, err := HotelFromProto(hotelRes.GetHotel())
	if err != nil {
		return nil, errors.Wrap(err, "HotelFromProto")
	}

	return fromProto, nil
}

func (s *ServiceImpl) UploadImage(ctx context.Context, data []byte, contentType, hotelID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.UploadImage")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return errors.Wrap(err, "ServiceImpl.UploadImage")
	}
	defer conn.Close()

	_, err = client.UploadImage(ctx, &hotels.UploadImageReq{
		HotelID:     hotelID,
		Data:        data,
		ContentType: contentType,
	})
	if err != nil {
		return errors.Wrap(err, "hotelsService.UploadImage")
	}

	return nil
}

func (s *ServiceImpl) GetHotels(ctx context.Context, page, size int64) (*ListResult, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.GetHotels")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.GetHotels")
	}
	defer conn.Close()

	hotelsRes, err := client.GetHotels(ctx, &hotels.GetHotelsReq{
		Page: page,
		Size: size,
	})
	if err != nil {
		s.logger.Errorf("error : %v", err)
		return nil, errors.Wrap(err, "hotelsService.GetHotels")
	}

	hotelsList := make([]*Hotel, 0, len(hotelsRes.Hotels))
	for _, v := range hotelsRes.Hotels {
		hotel, err := HotelFromProto(v)
		if err != nil {
			return nil, errors.Wrap(err, "hotelsService.HotelFromProto")
		}
		hotelsList = append(hotelsList, hotel)
	}

	return &ListResult{
		TotalCount: hotelsRes.GetTotalCount(),
		TotalPages: hotelsRes.GetTotalPages(),
		Page:       hotelsRes.GetPage(),
		Size:       hotelsRes.GetSize(),
		HasMore:    hotelsRes.GetHasMore(),
		Hotels:     hotelsList,
	}, nil
}
