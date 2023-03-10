package images

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"sync"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/disintegration/gift"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/pkg/logger"
)

var (
	ErrInvalidUUID            = errors.New("invalid uuid")
	ErrInvalidDeliveryHeaders = errors.New("invalid uuid")
	ErrInternalServerError    = errors.New("internal server error")
	ErrInvalidImageFormat     = errors.New("invalid image format")
)

const (
	userExchange           = "users"
	imageExchange          = "images"
	updateAvatarRoutingKey = "update_avatar_key"
	createImageRoutingKey  = "create_image_key"
	userUUIDHeader         = "user_uuid"
	resizeWidth            = 1024
	resizeHeight           = 0

	hotelsUUIDHeader      = "hotel_uuid"
	hotelsExchange        = "hotels"
	updateImageRoutingKey = "update_hotel_image_key"
)

type Repository interface {
	Create(ctx context.Context, msg *ImageDO) (*ImageDO, error)
	GetImageByID(ctx context.Context, imageID uuid.UUID) (*ImageDO, error)
}

type AWSStorage interface {
	PutObject(ctx context.Context, data []byte, fileType string) (string, error)
	GetObject(ctx context.Context, key string) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, key string) error
}

type Publisher interface {
	CreateExchangeAndQueue(exchange, queueName, bindingKey string) (*amqp.Channel, error)
	Publish(ctx context.Context, exchange, routingKey, contentType string, headers amqp.Table, body []byte) error
}

type ServiceImpl struct {
	pgRepo      Repository
	awsRepo     AWSStorage
	logger      logger.Logger
	publisher   Publisher
	resizerPool *sync.Pool
}

func NewService(pgRepo Repository, awsRepo AWSStorage, logger logger.Logger, publisher Publisher) ServiceImpl {
	resizerPool := &sync.Pool{New: func() interface{} {
		return NewImgResizer(
			gift.Resize(resizeWidth, resizeHeight, gift.LanczosResampling),
			gift.Contrast(20),
			gift.Brightness(7),
			gift.Gamma(0.5),
		)
	}}
	return ServiceImpl{pgRepo: pgRepo, awsRepo: awsRepo, logger: logger, publisher: publisher, resizerPool: resizerPool}
}

func (s *ServiceImpl) Create(ctx context.Context, delivery amqp.Delivery) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "images_service.Create")
	defer span.Finish()

	s.logger.Infof("amqp.Delivery: %-v", delivery.DeliveryTag)

	var msg UploadImageMsg
	if err := json.Unmarshal(delivery.Body, &msg); err != nil {
		return err
	}

	createdImage, err := s.pgRepo.Create(ctx, &ImageDO{
		ImageID:    msg.ImageID,
		ImageURL:   msg.ImageURL,
		IsUploaded: msg.IsUploaded,
	})
	if err != nil {
		return err
	}

	msgBytes, err := json.Marshal(createdImage)
	if err != nil {
		return errors.Join(err, errors.New("while marshalling json"))
	}

	headers := make(amqp.Table)
	headers[userUUIDHeader] = delivery.Headers[userUUIDHeader]
	if err := s.publisher.Publish(
		ctx,
		userExchange,
		updateAvatarRoutingKey,
		delivery.ContentType,
		headers,
		msgBytes,
	); err != nil {
		return errors.Join(err, errors.New("while publishing"))
	}

	return nil
}

func (s *ServiceImpl) ResizeImage(ctx context.Context, delivery amqp.Delivery) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "images_service.ResizeImage")
	defer span.Finish()

	s.logger.Infof("amqp.Delivery: %-v", delivery.DeliveryTag)

	parsedUUID, err := s.validateDeliveryHeaders(delivery)
	if err != nil {
		return err
	}

	processedImage, fileType, err := s.processImage(delivery.Body)
	if err != nil {
		return err
	}

	fileUrl, err := s.awsRepo.PutObject(ctx, processedImage, fileType)
	if err != nil {
		s.logger.Errorf("awsRepo.PutObject %-v", err)
		return err
	}

	msg := &UploadImageMsg{
		UserID:     *parsedUUID,
		ImageURL:   fileUrl,
		IsUploaded: true,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Join(err, errors.New("while marshalling json"))
	}

	headers := make(amqp.Table)
	headers[userUUIDHeader] = delivery.Headers[userUUIDHeader]
	if err := s.publisher.Publish(
		ctx,
		imageExchange,
		createImageRoutingKey,
		delivery.ContentType,
		headers,
		msgBytes,
	); err != nil {
		return errors.Join(err, errors.New("while publishing"))
	}

	return nil
}

func (s *ServiceImpl) GetImageByID(ctx context.Context, imageID uuid.UUID) (*ImageDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "images_service.GetImageByID")
	defer span.Finish()

	imgByID, err := s.pgRepo.GetImageByID(ctx, imageID)
	if err != nil {
		return nil, err
	}

	return imgByID, nil
}

func (s *ServiceImpl) ProcessHotelImage(ctx context.Context, delivery amqp.Delivery) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "images_service.Create")
	defer span.Finish()

	s.logger.Infof("amqp.Delivery: %-v", delivery.DeliveryTag)

	uuidHeader, err := s.extractUUIDHeader(delivery, hotelsUUIDHeader)
	if err != nil {
		return err
	}

	processedImage, fileType, err := s.processImage(delivery.Body)
	if err != nil {
		return err
	}

	fileUrl, err := s.awsRepo.PutObject(ctx, processedImage, fileType)
	if err != nil {
		s.logger.Errorf("awsRepo.PutObject %-v", err)
		return err
	}

	msg := &UpdateHotelImageMsg{
		HotelID: *uuidHeader,
		Image:   fileUrl,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return errors.Join(err, errors.New("while marshalling json"))
	}

	headers := make(amqp.Table)
	headers[hotelsUUIDHeader] = delivery.Headers[hotelsUUIDHeader]
	if err := s.publisher.Publish(
		ctx,
		hotelsExchange,
		updateImageRoutingKey,
		delivery.ContentType,
		headers,
		msgBytes,
	); err != nil {
		return errors.Join(err, errors.New("while publishing"))
	}

	return nil
}

func (s *ServiceImpl) validateDeliveryHeaders(delivery amqp.Delivery) (*uuid.UUID, error) {
	s.logger.Infof("amqp.Delivery header: %-v", delivery.Headers)

	userUUID, ok := delivery.Headers[userUUIDHeader]
	if !ok {
		return nil, ErrInvalidDeliveryHeaders
	}
	userID, ok := userUUID.(string)
	if !ok {
		return nil, ErrInvalidUUID
	}

	parsedUUID, err := uuid.FromString(userID)
	if err != nil {
		return nil, errors.Join(err, errors.New("while reading uuid"))
	}

	return &parsedUUID, nil
}

func (s *ServiceImpl) extractUUIDHeader(delivery amqp.Delivery, key string) (*uuid.UUID, error) {
	s.logger.Infof("amqp.Delivery header: %-v", delivery.Headers)

	uid, ok := delivery.Headers[key]
	if !ok {
		return nil, ErrInvalidDeliveryHeaders
	}
	userID, ok := uid.(string)
	if !ok {
		return nil, ErrInvalidUUID
	}

	parsedUUID, err := uuid.FromString(userID)
	if err != nil {
		return nil, errors.Join(err, errors.New("while reading uuid"))
	}

	return &parsedUUID, nil
}

func (s *ServiceImpl) processImage(img []byte) ([]byte, string, error) {
	src, imageType, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		return nil, "", err
	}

	imgResizer, ok := s.resizerPool.Get().(*ImgResizer)
	if !ok {
		return nil, "", ErrInternalServerError
	}
	defer s.resizerPool.Put(imgResizer)
	imgResizer.Buffer.Reset()

	dst := image.NewNRGBA(imgResizer.Gift.Bounds(src.Bounds()))
	imgResizer.Gift.Draw(dst, src)

	switch imageType {
	case "png":
		err = png.Encode(imgResizer.Buffer, dst)
		if err != nil {
			return nil, "", err
		}
	case "jpeg":
		err = jpeg.Encode(imgResizer.Buffer, dst, nil)
		if err != nil {
			return nil, "", err
		}
	case "jpg":
		err = jpeg.Encode(imgResizer.Buffer, dst, nil)
		if err != nil {
			return nil, "", err
		}
	case "gif":
		err = gif.Encode(imgResizer.Buffer, dst, nil)
		if err != nil {
			return nil, "", err
		}
	default:
		return nil, "", ErrInvalidImageFormat
	}

	return imgResizer.Buffer.Bytes(), imageType, nil
}
