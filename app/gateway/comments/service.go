package comments

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_comments_service.CreateComment")
	defer span.Finish()

	ctxUser, ok := ctx.Value(users.RequestCtxUser{}).(*users.UserResponse)
	if !ok || ctxUser == nil {
		return nil, errors.Join(httpErrors.Unauthorized, errors.New("context unknown user in service"))
	}

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client creation in service"))
	}
	defer conn.Close()

	commentRes, err := client.CreateComment(
		ctx,
		&comments.CreateCommentRequest{
			HotelID: comment.HotelID.String(),
			UserID:  ctxUser.UserID.String(),
			Message: comment.Message,
			Photos:  comment.Photos,
			Rating:  comment.Rating,
		},
	)
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client responded with error in service"))
	}

	result, err := fromProto(commentRes.GetComment())
	if err != nil {
		return nil, errors.Join(err, errors.New("error transforming from proto in service"))
	}

	return result, nil
}

func (s *ServiceImpl) GetCommByID(ctx context.Context, commentID uuid.UUID) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_comments_service.GetCommByID")
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
		return nil, errors.Join(err, errors.New("grpc client creation in service"))
	}
	defer conn.Close()

	commByID, err := client.GetCommByID(ctx, &comments.GetCommentByIDRequest{CommentID: commentID.String()})
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client responded with error in service"))
	}

	result, err := fromProto(commByID.GetComment())
	if err != nil {
		return nil, errors.Join(err, errors.New("error transforming from proto in service"))
	}

	if err := s.repository.SetComment(ctx, result); err != nil {
		s.logger.Errorf("SetComment: %v", err)
	}

	return result, nil
}

func (s *ServiceImpl) UpdateComment(ctx context.Context, comment *Comment) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_comments_service.CreateComment")
	defer span.Finish()

	ctxUser, ok := ctx.Value(users.RequestCtxUser{}).(*users.UserResponse)
	if !ok || ctxUser == nil {
		return nil, errors.Join(httpErrors.Unauthorized, errors.New("context unknown user in service"))
	}

	if ctxUser.UserID != comment.UserID {
		return nil, errors.Join(httpErrors.WrongCredentials, errors.New("user is not owner of the comment in service"))
	}

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client creation in service"))
	}
	defer conn.Close()

	commRes, err := client.UpdateComment(ctx, &comments.UpdateCommentRequest{
		CommentID: comment.CommentID.String(),
		Message:   comment.Message,
		Photos:    comment.Photos,
		Rating:    comment.Rating,
	})
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client responded with error in service"))
	}

	result, err := fromProto(commRes.GetComment())
	if err != nil {
		return nil, errors.Join(err, errors.New("error transforming from proto in service"))
	}

	if err := s.repository.SetComment(ctx, result); err != nil {
		s.logger.Errorf("SetComment: %v", err)
	}

	return result, nil
}

func (s *ServiceImpl) GetByHotelID(ctx context.Context, hotelID uuid.UUID, page, size int64) (*List, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_comments_service.GetByHotelID")
	defer span.Finish()

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client creation in service"))
	}
	defer conn.Close()

	hotel, err := client.GetByHotelID(ctx, &comments.GetCommentsByHotelRequest{
		HotelID: hotelID.String(),
		Page:    page,
		Size:    size,
	})
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client responded with error in service"))
	}

	result := make([]*CommentFull, 0, len(hotel.Comments))
	for _, comment := range hotel.Comments {
		if comm, err := fromProtoToFull(comment); err != nil {
			return nil, errors.Join(err, errors.New("transforming from proto in service"))
		} else {
			result = append(result, comm)
		}
	}

	return &List{
		TotalCount: hotel.GetTotalCount(),
		TotalPages: hotel.GetTotalPages(),
		Page:       hotel.GetPage(),
		Size:       hotel.GetSize(),
		HasMore:    hotel.GetHasMore(),
		Comments:   result,
	}, nil
}
