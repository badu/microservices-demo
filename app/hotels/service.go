package hotels

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/pkg/pagination"

	"github.com/badu/microservices-demo/pkg/logger"
)

var (
	ErrInvalidUUID            = errors.New("invalid uuid")
	ErrInvalidDeliveryHeaders = errors.New("invalid delivery headers")
	ErrInternalServerError    = errors.New("internal server error")
	ErrInvalidImageFormat     = errors.New("invalid file format")
	ErrHotelNotFound          = errors.New("hotel not found")
)

const (
	hotelIDHeader = "hotel_uuid"

	imagesExchange             = "images"
	uploadHotelImageRoutingKey = "upload_hotel_image"
)

type Repository interface {
	CreateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error)
	UpdateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error)
	UpdateHotelImage(ctx context.Context, hotelID uuid.UUID, imageURL string) error
	GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*HotelDO, error)
	GetHotels(ctx context.Context, query *pagination.Pagination) (*List, error)
}

type Publisher interface {
	CreateExchangeAndQueue(exchange, queueName, bindingKey string) (*amqp.Channel, error)
	Publish(ctx context.Context, exchange, routingKey, contentType string, headers amqp.Table, body []byte) error
}

type ServiceImpl struct {
	repository Repository
	logger     logger.Logger
	publisher  Publisher
}

func NewService(repository Repository, logger logger.Logger, publisher Publisher) ServiceImpl {
	return ServiceImpl{repository: repository, logger: logger, publisher: publisher}
}

func (h *ServiceImpl) CreateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_service.CreateHotel")
	defer span.Finish()

	return h.repository.CreateHotel(ctx, hotel)
}

func (h *ServiceImpl) UpdateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_service.UpdateHotel")
	defer span.Finish()

	return h.repository.UpdateHotel(ctx, hotel)
}

func (h *ServiceImpl) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_service.GetHotelByID")
	defer span.Finish()

	return h.repository.GetHotelByID(ctx, hotelID)
}

func (h *ServiceImpl) GetHotels(ctx context.Context, query *pagination.Pagination) (*List, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_service.CreateHotel")
	defer span.Finish()

	return h.repository.GetHotels(ctx, query)
}

func (h *ServiceImpl) UploadImage(ctx context.Context, msg *UploadHotelImageMsg) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_service.UploadImage")
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
		return errors.Join(err, errors.New("while publishing image"))
	}

	return nil
}

func (h *ServiceImpl) UpdateHotelImage(ctx context.Context, delivery amqp.Delivery) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_service.UpdateHotelImage")
	defer span.Finish()

	var msg UpdateHotelImageMsg
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		return errors.Join(err, errors.New("while unmarshalling json"))
	}

	if err := h.repository.UpdateHotelImage(ctx, msg.HotelID, msg.Image); err != nil {
		return err
	}

	return nil
}

func (h *ServiceImpl) validateDeliveryHeaders(delivery amqp.Delivery) (*uuid.UUID, error) {
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
		return nil, errors.Join(err, errors.New("while reading uuid"))
	}

	return &parsedUUID, nil
}
