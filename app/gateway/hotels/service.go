package hotels

import (
	"context"

	"github.com/badu/microservices-demo/app/gateway/users"
	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/grpc_client"
	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/gateway/middlewares"
	"github.com/badu/microservices-demo/app/hotels"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	GetHotels(ctx context.Context, page, size int64) (*ListResult, error)
	GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*Hotel, error)
	UpdateHotel(ctx context.Context, hotel *Hotel) (*Hotel, error)
	CreateHotel(ctx context.Context, hotel *Hotel) (*Hotel, error)
	UploadImage(ctx context.Context, data []byte, contentType, hotelID string) error
}

type serviceImpl struct {
	logger                logger.Logger
	grpcHotelsServicePort string
	repository            Repository
	mw                    *grpc_client.ClientMiddleware
}

func NewHotelsService(logger logger.Logger, cfg *config.Config, repository Repository, tracer opentracing.Tracer) *serviceImpl {
	logger.Debugf("grpcHotelsServicePort : %s", cfg.GRPC.HotelsServicePort)
	return &serviceImpl{logger: logger, repository: repository, grpcHotelsServicePort: cfg.GRPC.HotelsServicePort, mw: grpc_client.NewClientMiddleware(logger, cfg, tracer)}
}

func (s *serviceImpl) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.GetHotelByID")
	defer span.Finish()

	ctxUser, ok := ctx.Value(middlewares.RequestCtxUser{}).(*users.UserResponse)
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

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcHotelsServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.GetHotelByID")
	}
	defer conn.Close()

	client := hotels.NewHotelsServiceClient(conn)

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

func (s *serviceImpl) UpdateHotel(ctx context.Context, hotel *Hotel) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.UpdateHotel")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcHotelsServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.UpdateHotel")
	}
	defer conn.Close()

	client := hotels.NewHotelsServiceClient(conn)

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

func (s *serviceImpl) CreateHotel(ctx context.Context, hotel *Hotel) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.CreateHotel")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcHotelsServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.CreateHotel")
	}
	defer conn.Close()

	client := hotels.NewHotelsServiceClient(conn)

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

func (s *serviceImpl) UploadImage(ctx context.Context, data []byte, contentType, hotelID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.UploadImage")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcHotelsServicePort)
	if err != nil {
		return errors.Wrap(err, "hotelsService.UploadImage")
	}
	defer conn.Close()

	client := hotels.NewHotelsServiceClient(conn)

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

func (s *serviceImpl) GetHotels(ctx context.Context, page, size int64) (*ListResult, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsService.GetHotels")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcHotelsServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.GetHotels")
	}
	defer conn.Close()

	client := hotels.NewHotelsServiceClient(conn)

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
