package comments

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"

	"github.com/badu/microservices-demo/app/comments"
	"github.com/badu/microservices-demo/app/gateway/users"
	httpErrors "github.com/badu/microservices-demo/pkg/http_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Repository interface {
	CommentByID(ctx context.Context, commentID uuid.UUID) (*Comment, error)
	SetComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, commentID uuid.UUID) error
}

type ServiceImpl struct {
	logger            logger.Logger
	repository        Repository
	grpcClientFactory func(ctx context.Context) (*grpc.ClientConn, comments.CommentsServiceClient, error)
}

func NewService(
	logger logger.Logger,
	repository Repository,
	grpcClientFactory func(ctx context.Context) (*grpc.ClientConn, comments.CommentsServiceClient, error),
) ServiceImpl {
	return ServiceImpl{
		logger:            logger,
		repository:        repository,
		grpcClientFactory: grpcClientFactory,
	}
}

func (s *ServiceImpl) CreateComment(ctx context.Context, comment *Comment) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.CreateComment")
	defer span.Finish()

	ctxUser, ok := ctx.Value(users.RequestCtxUser{}).(*users.UserResponse)
	if !ok || ctxUser == nil {
		return nil, errors.Wrap(httpErrors.Unauthorized, "ctx.Value user")
	}

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.CreateComment")
	}
	defer conn.Close()

	commentRes, err := client.CreateComment(
		ctx,
		&comments.CreateCommentReq{
			HotelID: comment.HotelID.String(),
			UserID:  ctxUser.UserID.String(),
			Message: comment.Message,
			Photos:  comment.Photos,
			Rating:  comment.Rating,
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "hotelsService.CreateHotel")
	}

	comm, err := FromProto(commentRes.GetComment())
	if err != nil {
		return nil, errors.Wrap(err, "FromProto")
	}

	return comm, nil
}

func (s *ServiceImpl) GetCommByID(ctx context.Context, commentID uuid.UUID) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.GetCommByID")
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

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.GetCommByID")
	}
	defer conn.Close()

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

func (s *ServiceImpl) UpdateComment(ctx context.Context, comment *Comment) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.CreateComment")
	defer span.Finish()

	ctxUser, ok := ctx.Value(users.RequestCtxUser{}).(*users.UserResponse)
	if !ok || ctxUser == nil {
		return nil, errors.Wrap(httpErrors.Unauthorized, "ctx.Value user")
	}

	if ctxUser.UserID != comment.UserID {
		return nil, errors.Wrap(httpErrors.WrongCredentials, "user is not owner")
	}

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.UpdateComment")
	}
	defer conn.Close()

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

func (s *ServiceImpl) GetByHotelID(ctx context.Context, hotelID uuid.UUID, page, size int64) (*List, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.GetByHotelID")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.GetByHotelID")
	}
	defer conn.Close()

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
		if comm, err := FullFromProto(comment); err != nil {
			return nil, errors.Wrap(err, "FullFromProto")
		} else {
			commList = append(commList, comm)
		}
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
