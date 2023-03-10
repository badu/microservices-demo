package hotels

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_hotels_service.GetHotelByID")
	defer span.Finish()

	ctxUser, ok := ctx.Value(users.RequestCtxUser{}).(*users.UserResponse)
	if !ok || ctxUser == nil {
		return nil, errors.Join(httpErrors.Unauthorized, errors.New("current user not present in context in service"))
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
		return nil, errors.Join(err, errors.New("creating hotels grpc client in service"))
	}
	defer conn.Close()

	hotelByID, err := client.GetHotelByID(ctx, &hotels.GetByIDRequest{HotelID: hotelID.String()})
	if err != nil {
		return nil, errors.Join(err, errors.New("hotels grpc client returned an error in service"))
	}

	fromProto, err := fromProto(hotelByID.GetHotel())
	if err != nil {
		return nil, errors.Join(err, errors.New("transforming hotel from gprc response"))
	}

	if err := s.repository.SetHotel(ctx, fromProto); err != nil {
		s.logger.Errorf("SetHotel: %v", err)
	}

	return fromProto, nil
}

func (s *ServiceImpl) UpdateHotel(ctx context.Context, hotel *Hotel) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_hotels_service.UpdateHotel")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("creating hotels grpc client in service"))
	}
	defer conn.Close()

	hotelRes, err := client.UpdateHotel(ctx, &hotels.UpdateHotelRequest{
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
		return nil, errors.Join(err, errors.New("hotel grpc client responded with error"))
	}

	fromProto, err := fromProto(hotelRes.GetHotel())
	if err != nil {
		return nil, errors.Join(err, errors.New("reading hotel grpc response"))
	}

	if err := s.repository.SetHotel(ctx, fromProto); err != nil {
		s.logger.Errorf("SetHotel: %v", err)
	}

	return fromProto, nil
}

func (s *ServiceImpl) CreateHotel(ctx context.Context, hotel *Hotel) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_hotels_service.CreateHotel")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("creating gprc client for hotels"))
	}
	defer conn.Close()

	hotelRes, err := client.CreateHotel(ctx, &hotels.CreateHotelRequest{
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
		return nil, errors.Join(err, errors.New("grpc client responded with error"))
	}

	fromProto, err := fromProto(hotelRes.GetHotel())
	if err != nil {
		return nil, errors.Join(err, errors.New("transforming grpc client response"))
	}

	return fromProto, nil
}

func (s *ServiceImpl) UploadImage(ctx context.Context, data []byte, contentType, hotelID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_hotels_service.UploadImage")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return errors.Join(err, errors.New("creating grpc client in service"))
	}
	defer conn.Close()

	_, err = client.UploadImage(ctx, &hotels.UploadImageRequest{
		HotelID:     hotelID,
		Data:        data,
		ContentType: contentType,
	})
	if err != nil {
		return errors.Join(err, errors.New("grpc client responded with error"))
	}

	return nil
}

func (s *ServiceImpl) GetHotels(ctx context.Context, page, size int64) (*ListResult, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_hotels_service.GetHotels")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("creating hotels grpc client in service"))
	}
	defer conn.Close()

	hotelsRes, err := client.GetHotels(ctx, &hotels.GetHotelsRequest{
		Page: page,
		Size: size,
	})
	if err != nil {
		s.logger.Errorf("error : %v", err)
		return nil, errors.Join(err, errors.New("reading hotels in service"))
	}

	hotelsList := make([]*Hotel, 0, len(hotelsRes.Hotels))
	for _, v := range hotelsRes.Hotels {
		hotel, err := fromProto(v)
		if err != nil {
			return nil, errors.Join(err, errors.New("converting from proto in service"))
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
