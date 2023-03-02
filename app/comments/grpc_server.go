package comments

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/pkg/pagination"

	"github.com/badu/microservices-demo/pkg/config"
	grpcErrors "github.com/badu/microservices-demo/pkg/grpc_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	Create(ctx context.Context, comment *CommentDO) (*CommentDO, error)
	GetByID(ctx context.Context, commentID uuid.UUID) (*CommentDO, error)
	Update(ctx context.Context, comment *CommentDO) (*CommentDO, error)
	GetByHotelID(ctx context.Context, hotelID uuid.UUID, query *pagination.Pagination) (*FullList, error)
}

type GRPCServer struct {
	service  Service
	logger   logger.Logger
	cfg      *config.Config
	validate *validator.Validate
}

func NewServer(service Service, logger logger.Logger, cfg *config.Config, validate *validator.Validate) GRPCServer {
	return GRPCServer{service: service, logger: logger, cfg: cfg, validate: validate}
}

func (c *GRPCServer) CreateComment(ctx context.Context, req *CreateCommentReq) (*CreateCommentRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GRPCServer.CreateComment")
	defer span.Finish()

	comm, err := c.protoToModel(req)
	if err != nil {
		c.logger.Errorf("validate.StructCtx: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	if err := c.validate.StructCtx(ctx, comm); err != nil {
		c.logger.Errorf("validate.StructCtx: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	createdComm, err := c.service.Create(ctx, comm)
	if err != nil {
		c.logger.Errorf("service.Create: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	c.logger.Infof("CREATED: %-v", createdComm)

	return &CreateCommentRes{Comment: createdComm.ToProto()}, nil
}

func (c *GRPCServer) GetCommByID(ctx context.Context, req *GetCommByIDReq) (*GetCommByIDRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GRPCServer.GetCommByID")
	defer span.Finish()

	commUUID, err := uuid.FromString(req.GetCommentID())
	if err != nil {
		c.logger.Errorf("uuid.FromString: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	comm, err := c.service.GetByID(ctx, commUUID)
	if err != nil {
		c.logger.Errorf("service.GetByID: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	return &GetCommByIDRes{Comment: comm.ToProto()}, nil
}

func (c *GRPCServer) UpdateComment(ctx context.Context, req *UpdateCommReq) (*UpdateCommRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GRPCServer.UpdateComment")
	defer span.Finish()

	commUUID, err := uuid.FromString(req.GetCommentID())
	if err != nil {
		return nil, err
	}

	comm := &CommentDO{
		CommentID: commUUID,
		Message:   req.GetMessage(),
		Photos:    req.GetPhotos(),
		Rating:    req.GetRating(),
	}

	if err := c.validate.StructCtx(ctx, comm); err != nil {
		c.logger.Errorf("validate.StructCtx: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	updatedComm, err := c.service.Update(ctx, comm)
	if err != nil {
		c.logger.Errorf("service.Update: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	return &UpdateCommRes{Comment: updatedComm.ToProto()}, nil
}

func (c *GRPCServer) GetByHotelID(ctx context.Context, req *GetByHotelReq) (*GetByHotelRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "GRPCServer.GetByHotelID")
	defer span.Finish()

	hotelUUID, err := uuid.FromString(req.GetHotelID())
	if err != nil {
		c.logger.Errorf("uuid.FromString: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	query := pagination.NewPaginationQuery(int(req.GetSize()), int(req.GetPage()))

	commentsList, err := c.service.GetByHotelID(ctx, hotelUUID, query)
	if err != nil {
		c.logger.Errorf("service.GetByHotelID: %v", err)
		return nil, grpcErrors.ErrorResponse(err, err.Error())
	}

	return &GetByHotelRes{
		TotalCount: int64(commentsList.TotalCount),
		TotalPages: int64(commentsList.TotalPages),
		Page:       int64(commentsList.Page),
		Size:       int64(commentsList.Size),
		HasMore:    commentsList.HasMore,
		Comments:   commentsList.Comments,
	}, nil
}

func (c *GRPCServer) protoToModel(req *CreateCommentReq) (*CommentDO, error) {
	hotelUUID, err := uuid.FromString(req.GetHotelID())
	if err != nil {
		return nil, err
	}
	userUUID, err := uuid.FromString(req.GetUserID())
	if err != nil {
		return nil, err
	}

	return &CommentDO{
		HotelID: hotelUUID,
		UserID:  userUUID,
		Message: req.GetMessage(),
		Photos:  req.GetPhotos(),
		Rating:  req.GetRating(),
	}, nil
}
