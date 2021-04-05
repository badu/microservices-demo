package comments

import (
	"context"

	"github.com/badu/microservices-demo/app/gateway/users"
	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/grpc_client"
	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/comments"
	"github.com/badu/microservices-demo/app/gateway/middlewares"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	CreateComment(ctx context.Context, comment *Comment) (*Comment, error)
	GetCommByID(ctx context.Context, commentID uuid.UUID) (*Comment, error)
	UpdateComment(ctx context.Context, comment *Comment) (*Comment, error)
	GetByHotelID(ctx context.Context, hotelID uuid.UUID, page, size int64) (*List, error)
}

type serviceImpl struct {
	logger                  logger.Logger
	grpcCommentsServicePort string
	repository              Repository
	mw                      *grpc_client.ClientMiddleware
}

func NewService(logger logger.Logger, cfg *config.Config, repository Repository, tracer opentracing.Tracer) *serviceImpl {
	return &serviceImpl{logger: logger, repository: repository, grpcCommentsServicePort: cfg.GRPC.CommentsServicePort, mw: grpc_client.NewClientMiddleware(logger, cfg, tracer)}
}

func (s *serviceImpl) CreateComment(ctx context.Context, comment *Comment) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.CreateComment")
	defer span.Finish()

	ctxUser, ok := ctx.Value(middlewares.RequestCtxUser{}).(*users.UserResponse)
	if !ok || ctxUser == nil {
		return nil, errors.Wrap(httpErrors.Unauthorized, "ctx.Value user")
	}

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcCommentsServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "serviceImpl.CreateComment")
	}
	defer conn.Close()

	client := comments.NewCommentsServiceClient(conn)

	commentRes, err := client.CreateComment(ctx, &comments.CreateCommentReq{
		HotelID: comment.HotelID.String(),
		UserID:  ctxUser.UserID.String(),
		Message: comment.Message,
		Photos:  comment.Photos,
		Rating:  comment.Rating,
	})
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.CreateHotel")
	}

	comm, err := FromProto(commentRes.GetComment())
	if err != nil {
		return nil, errors.Wrap(err, "FromProto")
	}

	return comm, nil
}

func (s *serviceImpl) GetCommByID(ctx context.Context, commentID uuid.UUID) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.GetCommByID")
	defer span.Finish()

	cacheComm, err := s.repository.CommentByID(ctx, commentID)
	if err != nil {
		if err != redis.Nil {
			s.logger.Errorf("CommentByID: %v", err)
		}
	}
	if cacheComm != nil {
		return cacheComm, nil
	}

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcCommentsServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "serviceImpl.GetCommByID")
	}
	defer conn.Close()

	client := comments.NewCommentsServiceClient(conn)
	commByID, err := client.GetCommByID(ctx, &comments.GetCommByIDReq{CommentID: commentID.String()})
	if err != nil {
		return nil, errors.Wrap(err, "commService.GetCommByID")
	}

	comm, err := FromProto(commByID.GetComment())
	if err != nil {
		return nil, errors.Wrap(err, "FromProto")
	}

	if err := s.repository.SetComment(ctx, comm); err != nil {
		s.logger.Errorf("SetComment: %v", err)
	}

	return comm, nil
}

func (s *serviceImpl) UpdateComment(ctx context.Context, comment *Comment) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.CreateComment")
	defer span.Finish()

	ctxUser, ok := ctx.Value(middlewares.RequestCtxUser{}).(*users.UserResponse)
	if !ok || ctxUser == nil {
		return nil, errors.Wrap(httpErrors.Unauthorized, "ctx.Value user")
	}

	if ctxUser.UserID != comment.UserID {
		return nil, errors.Wrap(httpErrors.WrongCredentials, "user is not owner")
	}

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcCommentsServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "serviceImpl.UpdateComment")
	}
	defer conn.Close()

	client := comments.NewCommentsServiceClient(conn)
	commRes, err := client.UpdateComment(ctx, &comments.UpdateCommReq{
		CommentID: comment.CommentID.String(),
		Message:   comment.Message,
		Photos:    comment.Photos,
		Rating:    comment.Rating,
	})
	if err != nil {
		return nil, errors.Wrap(err, "FromProto")
	}

	comm, err := FromProto(commRes.GetComment())
	if err != nil {
		return nil, errors.Wrap(err, "FromProto")
	}

	if err := s.repository.SetComment(ctx, comm); err != nil {
		s.logger.Errorf("SetComment: %v", err)
	}

	return comm, nil
}

func (s *serviceImpl) GetByHotelID(ctx context.Context, hotelID uuid.UUID, page, size int64) (*List, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.GetByHotelID")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.grpcCommentsServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "serviceImpl.GetByHotelID")
	}
	defer conn.Close()

	client := comments.NewCommentsServiceClient(conn)
	res, err := client.GetByHotelID(ctx, &comments.GetByHotelReq{
		HotelID: hotelID.String(),
		Page:    page,
		Size:    size,
	})
	if err != nil {
		return nil, errors.Wrap(err, "FromProto")
	}

	commList := make([]*CommentFull, 0, len(res.Comments))
	for _, comment := range res.Comments {
		comm, err := FullFromProto(comment)
		if err != nil {
			return nil, errors.Wrap(err, "FullFromProto")
		}
		commList = append(commList, comm)
	}

	return &List{
		TotalCount: res.GetTotalCount(),
		TotalPages: res.GetTotalPages(),
		Page:       res.GetPage(),
		Size:       res.GetSize(),
		HasMore:    res.GetHasMore(),
		Comments:   commList,
	}, nil
}
