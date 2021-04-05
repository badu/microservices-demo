package hotels

import (
	"context"
	"encoding/json"

	"github.com/badu/microservices-demo/pkg/pagination"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/pkg/logger"
)

var (
	ErrInvalidUUID            = errors.New("invalid uuid")
	ErrInvalidDeliveryHeaders = errors.New("invalid delivery headers")
	ErrInternalServerError    = errors.New("internal server error")
	ErrInvalidImageFormat     = errors.New("invalid file format")
	ErrHotelNotFound          = errors.New("hotel not found")
)

type Service interface {
	CreateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error)
	UpdateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error)
	GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*HotelDO, error)
	GetHotels(ctx context.Context, query *pagination.Pagination) (*List, error)
	UploadImage(ctx context.Context, msg *UploadHotelImageMsg) error
	UpdateHotelImage(ctx context.Context, delivery amqp.Delivery) error
}

const (
	hotelIDHeader = "hotel_uuid"

	imagesExchange             = "images"
	uploadHotelImageRoutingKey = "upload_hotel_image"
)

type serviceImpl struct {
	repository Repository
	logger     logger.Logger
	publisher  Publisher
}

func NewService(repository Repository, logger logger.Logger, publisher Publisher) *serviceImpl {
	return &serviceImpl{repository: repository, logger: logger, publisher: publisher}
}

func (h *serviceImpl) CreateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.CreateHotel")
	defer span.Finish()

	return h.repository.CreateHotel(ctx, hotel)
}

func (h *serviceImpl) UpdateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.UpdateHotel")
	defer span.Finish()

	return h.repository.UpdateHotel(ctx, hotel)
}

func (h *serviceImpl) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.GetHotelByID")
	defer span.Finish()

	return h.repository.GetHotelByID(ctx, hotelID)
}

func (h *serviceImpl) GetHotels(ctx context.Context, query *pagination.Pagination) (*List, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.CreateHotel")
	defer span.Finish()

	return h.repository.GetHotels(ctx, query)
}

func (h *serviceImpl) UploadImage(ctx context.Context, msg *UploadHotelImageMsg) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.UploadImage")
	defer span.Finish()

	headers := make(amqp.Table, 1)
	headers[hotelIDHeader] = msg.HotelID.String()
	if err := h.publisher.Publish(
		ctx,
		imagesExchange,
		uploadHotelImageRoutingKey,
		msg.ContentType,
		headers,
		msg.Data,
	); err != nil {
		return errors.Wrap(err, "UpdateUploadedAvatar.Publish")
	}

	return nil
}

func (h *serviceImpl) UpdateHotelImage(ctx context.Context, delivery amqp.Delivery) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "service.UpdateHotelImage")
	defer span.Finish()

	var msg UpdateHotelImageMsg
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		return errors.Wrap(err, "UpdateHotelImage.json.Unmarshal")
	}

	if err := h.repository.UpdateHotelImage(ctx, msg.HotelID, msg.Image); err != nil {
		return err
	}

	return nil
}

func (h *serviceImpl) validateDeliveryHeaders(delivery amqp.Delivery) (*uuid.UUID, error) {
	h.logger.Infof("amqp.Delivery header: %-v", delivery.Headers)

	hotelUUID, ok := delivery.Headers[hotelIDHeader]
	if !ok {
		return nil, ErrInvalidDeliveryHeaders
	}
	hotelID, ok := hotelUUID.(string)
	if !ok {
		return nil, ErrInvalidUUID
	}

	parsedUUID, err := uuid.FromString(hotelID)
	if err != nil {
		return nil, errors.Wrap(err, "uuid.FromString")
	}

	return &parsedUUID, nil
}
