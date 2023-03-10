package images

import (
	"context"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
	"google.golang.org/grpc/status"

	"github.com/badu/microservices-demo/pkg/config"
	grpcErrors "github.com/badu/microservices-demo/pkg/grpc_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	ResizeImage(ctx context.Context, delivery amqp.Delivery) error
	ProcessHotelImage(ctx context.Context, delivery amqp.Delivery) error
	Create(ctx context.Context, delivery amqp.Delivery) error
	GetImageByID(ctx context.Context, imageID uuid.UUID) (*ImageDO, error)
}

type ImageServer struct {
	cfg     *config.Config
	logger  logger.Logger
	service Service
}

func NewImageServer(cfg *config.Config, logger logger.Logger, service Service) ImageServer {
	return ImageServer{cfg: cfg, logger: logger, service: service}
}

func (i *ImageServer) GetImageByID(ctx context.Context, req *GetByIDRequest) (*GetByIDResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "images_grpc_server.GetImageByID")
	defer span.Finish()

	imageUUID, err := uuid.FromString(req.GetImageID())
	if err != nil {
		i.logger.Errorf("uuid.FromString: %v", err)
		return nil, status.Errorf(grpcErrors.ParseGRPCErrStatusCode(err), "uuid.FromString: %v", err)
	}

	imageByID, err := i.service.GetImageByID(ctx, imageUUID)
	if err != nil {
		i.logger.Errorf("uuid.FromString: %v", err)
		return nil, status.Errorf(grpcErrors.ParseGRPCErrStatusCode(err), "imageServer.GetImageByID: %v", err)
	}

	return &GetByIDResponse{Image: imageByID.ToProto()}, nil
}
