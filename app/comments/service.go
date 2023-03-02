package comments

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"

	"github.com/badu/microservices-demo/pkg/pagination"

	"github.com/badu/microservices-demo/app/users"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Repository interface {
	Create(ctx context.Context, comment *CommentDO) (*CommentDO, error)
	GetByID(ctx context.Context, commentID uuid.UUID) (*CommentDO, error)
	Update(ctx context.Context, comment *CommentDO) (*CommentDO, error)
	GetByHotelID(ctx context.Context, hotelID uuid.UUID, query *pagination.Pagination) (*List, error)
}

type ServiceImpl struct {
	repository        Repository
	logger            logger.Logger
	grpcClientFactory func(ctx context.Context) (*grpc.ClientConn, users.UserServiceClient, error)
}

func NewService(
	repository Repository,
	logger logger.Logger,
	grpcClientFactory func(ctx context.Context) (*grpc.ClientConn, users.UserServiceClient, error),
) ServiceImpl {
	return ServiceImpl{repository: repository, logger: logger, grpcClientFactory: grpcClientFactory}
}

func (s *ServiceImpl) Create(ctx context.Context, comment *CommentDO) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.Create")
	defer span.Finish()
	return s.repository.Create(ctx, comment)
}

func (s *ServiceImpl) GetByID(ctx context.Context, commentID uuid.UUID) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.GetByID")
	defer span.Finish()
	return s.repository.GetByID(ctx, commentID)
}

func (s *ServiceImpl) Update(ctx context.Context, comment *CommentDO) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.Update")
	defer span.Finish()
	return s.repository.Update(ctx, comment)
}

func (s *ServiceImpl) GetByHotelID(ctx context.Context, hotelID uuid.UUID, query *pagination.Pagination) (*FullList, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.GetByHotelID")
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

	conn, client, err := s.grpcClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.GetByHotelID")
	}
	defer conn.Close()

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
