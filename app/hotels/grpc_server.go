package hotels

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	grpcErrors "github.com/badu/microservices-demo/pkg/grpc_errors"
	"github.com/badu/microservices-demo/pkg/logger"
	"github.com/badu/microservices-demo/pkg/pagination"
)

type Service interface {
	CreateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error)
	UpdateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error)
	GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*HotelDO, error)
	GetHotels(ctx context.Context, query *pagination.Pagination) (*List, error)
	UploadImage(ctx context.Context, msg *UploadHotelImageMsg) error
	UpdateHotelImage(ctx context.Context, delivery amqp.Delivery) error
}

type GRPCServer struct {
	service  Service
	logger   logger.Logger
	validate *validator.Validate
}

func NewServer(service Service, logger logger.Logger, validate *validator.Validate) GRPCServer {
	return GRPCServer{service: service, logger: logger, validate: validate}
}

func (h *GRPCServer) CreateHotel(ctx context.Context, req *CreateHotelRequest) (*CreateHotelResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_grpc_server.CreateHotel")
	defer span.Finish()

	hotel := &HotelDO{
		Name:          req.GetName(),
		Email:         req.GetEmail(),
		Country:       req.GetCountry(),
		City:          req.GetCity(),
		Description:   req.GetDescription(),
		Image:         &req.Image,
		Photos:        req.Photos,
		CommentsCount: int(req.CommentsCount),
		Latitude:      &req.Latitude,
		Longitude:     &req.Longitude,
		Location:      req.Location,
		Rating:        req.GetRating(),
	}

	if err := h.validate.StructCtx(ctx, hotel); err != nil {
		h.logger.Errorf("validate.StructCtx: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	createdHotel, err := h.service.CreateHotel(ctx, hotel)
	if err != nil {
		h.logger.Errorf("service.CreateHotel: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "GRPCServer.GetByID")
	}

	return &CreateHotelResponse{Hotel: createdHotel.ToProto()}, nil
}

func (h *GRPCServer) UpdateHotel(ctx context.Context, req *UpdateHotelRequest) (*UpdateHotelResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_grpc_server.UpdateHotel")
	defer span.Finish()

	hotelUUID, err := uuid.FromString(req.GetHotelID())
	if err != nil {
		h.logger.Errorf("uuid.FromString: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "uuid.FromString")
	}

	hotel := &HotelDO{
		HotelID:       hotelUUID,
		Name:          req.GetName(),
		Email:         req.GetEmail(),
		Country:       req.GetCountry(),
		City:          req.GetCity(),
		Description:   req.GetDescription(),
		Image:         &req.Image,
		Photos:        req.Photos,
		CommentsCount: int(req.CommentsCount),
		Latitude:      &req.Latitude,
		Longitude:     &req.Longitude,
		Location:      req.Location,
		Rating:        req.GetRating(),
	}

	if err := h.validate.StructCtx(ctx, hotel); err != nil {
		h.logger.Errorf("validate.StructCtx: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	updatedHotel, err := h.service.UpdateHotel(ctx, hotel)
	if err != nil {
		h.logger.Errorf("service.UpdateHotel: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	return &UpdateHotelResponse{Hotel: updatedHotel.ToProto()}, err
}

func (h *GRPCServer) GetHotelByID(ctx context.Context, req *GetByIDRequest) (*GetByIDResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_grpc_server.GetHotelByID")
	defer span.Finish()

	hotelUUID, err := uuid.FromString(req.GetHotelID())
	if err != nil {
		h.logger.Errorf("uuid.FromString: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "uuid.FromString")
	}

	hotel, err := h.service.GetHotelByID(ctx, hotelUUID)
	if err != nil {
		h.logger.Errorf("service.GetHotelByID: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "service.GetHotelByID")
	}

	return &GetByIDResponse{Hotel: hotel.ToProto()}, nil
}

func (h *GRPCServer) GetHotels(ctx context.Context, req *GetHotelsRequest) (*GetHotelsResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_grpc_server.GetHotels")
	defer span.Finish()

	query := pagination.NewPaginationQuery(int(req.GetSize()), int(req.GetPage()))

	hotelsList, err := h.service.GetHotels(ctx, query)
	if err != nil {
		h.logger.Errorf("service.GetHotels: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "service.GetHotels")
	}

	return &GetHotelsResponse{
		TotalCount: int64(hotelsList.TotalCount),
		TotalPages: int64(hotelsList.TotalPages),
		Page:       int64(hotelsList.Page),
		Size:       int64(hotelsList.Size),
		HasMore:    hotelsList.HasMore,
		Hotels:     hotelsList.ToProto(),
	}, nil
}

func (h *GRPCServer) UploadImage(ctx context.Context, req *UploadImageRequest) (*UploadImageResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_grpc_server.UploadImage")
	defer span.Finish()

	hotelUUID, err := uuid.FromString(req.GetHotelID())
	if err != nil {
		h.logger.Errorf("uuid.FromString: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "uuid.FromString")
	}

	if err := h.service.UploadImage(ctx, &UploadHotelImageMsg{
		HotelID:     hotelUUID,
		Data:        req.GetData(),
		ContentType: req.GetContentType(),
	}); err != nil {
		h.logger.Errorf("service.UploadImage: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "service.UploadImage")
	}

	return &UploadImageResponse{HotelID: hotelUUID.String()}, nil
}
