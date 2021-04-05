package comments

import (
	"context"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/grpc_client"
	"github.com/badu/microservices-demo/pkg/pagination"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/users"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	Create(ctx context.Context, comment *CommentDO) (*CommentDO, error)
	GetByID(ctx context.Context, commentID uuid.UUID) (*CommentDO, error)
	Update(ctx context.Context, comment *CommentDO) (*CommentDO, error)
	GetByHotelID(ctx context.Context, hotelID uuid.UUID, query *pagination.Pagination) (*FullList, error)
}

type serviceImpl struct {
	repository       Repository
	logger           logger.Logger
	mw               *grpc_client.ClientMiddleware
	usersServicePort string
}

func NewService(
	repository Repository,
	logger logger.Logger,
	cfg *config.Config,
	tracer opentracing.Tracer,
) *serviceImpl {
	return &serviceImpl{repository: repository, logger: logger, usersServicePort: cfg.GRPC.UserServicePort, mw: grpc_client.NewClientMiddleware(logger, cfg, tracer)}
}

func (s *serviceImpl) Create(ctx context.Context, comment *CommentDO) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.Create")
	defer span.Finish()
	return s.repository.Create(ctx, comment)
}

func (s *serviceImpl) GetByID(ctx context.Context, commentID uuid.UUID) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.GetByID")
	defer span.Finish()
	return s.repository.GetByID(ctx, commentID)
}

func (s *serviceImpl) Update(ctx context.Context, comment *CommentDO) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.Update")
	defer span.Finish()
	return s.repository.Update(ctx, comment)
}

func (s *serviceImpl) GetByHotelID(ctx context.Context, hotelID uuid.UUID, query *pagination.Pagination) (*FullList, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.GetByHotelID")
	defer span.Finish()

	commentsList, err := s.repository.GetByHotelID(ctx, hotelID, query)
	if err != nil {
		return nil, err
	}

	uniqUserIDsMap := make(map[string]struct{}, len(commentsList.Comments))
	for _, comm := range commentsList.Comments {
		uniqUserIDsMap[comm.UserID.String()] = struct{}{}
	}

	userIDS := make([]string, 0, len(commentsList.Comments))
	for key := range uniqUserIDsMap {
		userIDS = append(userIDS, key)
	}

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.usersServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "commentsService.GetByHotelID")
	}
	defer conn.Close()

	client := users.NewUserServiceClient(conn)

	usersByIDs, err := client.GetUsersByIDs(ctx, &users.GetByIDsReq{UsersIDs: userIDS})
	if err != nil {
		return nil, err
	}

	return &FullList{
		TotalCount: commentsList.TotalCount,
		TotalPages: commentsList.TotalPages,
		Page:       commentsList.Page,
		Size:       commentsList.Size,
		HasMore:    commentsList.HasMore,
		Comments:   commentsList.ToHotelByIDProto(usersByIDs.GetUsers()),
	}, nil
}
