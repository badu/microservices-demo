package hotels

import (
	"context"

	grpcErrors "github.com/badu/microservices-demo/pkg/grpc_errors"
	"github.com/badu/microservices-demo/pkg/pagination"
	"github.com/go-playground/validator/v10"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/pkg/logger"
)

type hotelsServer struct {
	service  Service
	logger   logger.Logger
	validate *validator.Validate
}

func NewHotelsServer(service Service, logger logger.Logger, validate *validator.Validate) *hotelsServer {
	return &hotelsServer{service: service, logger: logger, validate: validate}
}

func (h *hotelsServer) CreateHotel(ctx context.Context, req *CreateHotelReq) (*CreateHotelRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsServer.CreateHotel")
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
		return nil, grpcErrors.ErrorResponse(err, "hotelsServer.GetByID")
	}

	return &CreateHotelRes{Hotel: createdHotel.ToProto()}, nil
}

func (h *hotelsServer) UpdateHotel(ctx context.Context, req *UpdateHotelReq) (*UpdateHotelRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsServer.UpdateHotel")
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

	return &UpdateHotelRes{Hotel: updatedHotel.ToProto()}, err
}

func (h *hotelsServer) GetHotelByID(ctx context.Context, req *GetByIDReq) (*GetByIDRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsServer.GetHotelByID")
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

	return &GetByIDRes{Hotel: hotel.ToProto()}, nil
}

func (h *hotelsServer) GetHotels(ctx context.Context, req *GetHotelsReq) (*GetHotelsRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsServer.GetHotels")
	defer span.Finish()

	query := pagination.NewPaginationQuery(int(req.GetSize()), int(req.GetPage()))

	hotelsList, err := h.service.GetHotels(ctx, query)
	if err != nil {
		h.logger.Errorf("service.GetHotels: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "service.GetHotels")
	}

	return &GetHotelsRes{
		TotalCount: int64(hotelsList.TotalCount),
		TotalPages: int64(hotelsList.TotalPages),
		Page:       int64(hotelsList.Page),
		Size:       int64(hotelsList.Size),
		HasMore:    hotelsList.HasMore,
		Hotels:     hotelsList.ToProto(),
	}, nil
}

func (h *hotelsServer) UploadImage(ctx context.Context, req *UploadImageReq) (*UploadImageRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotelsServer.UploadImage")
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

	return &UploadImageRes{HotelID: hotelUUID.String()}, nil
}
