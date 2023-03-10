package users

import (
	"context"
	"errors"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc"

	"github.com/badu/microservices-demo/app/sessions"
	"github.com/badu/microservices-demo/app/users"
	"github.com/badu/microservices-demo/pkg/logger"
)

type ServiceImpl struct {
	logger                    logger.Logger
	grpcSessionsClientFactory func(ctx context.Context) (*grpc.ClientConn, sessions.AuthorizationServiceClient, error)
	grpcUsersClientFactory    func(ctx context.Context) (*grpc.ClientConn, users.UserServiceClient, error)
}

func NewService(
	logger logger.Logger,
	grpcSessionsClientFactory func(ctx context.Context) (*grpc.ClientConn, sessions.AuthorizationServiceClient, error),
	grpcUsersClientFactory func(ctx context.Context) (*grpc.ClientConn, users.UserServiceClient, error),
) ServiceImpl {
	return ServiceImpl{
		logger:                    logger,
		grpcSessionsClientFactory: grpcSessionsClientFactory,
		grpcUsersClientFactory:    grpcUsersClientFactory,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, userUUID uuid.UUID) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_users_service.GetByID")
	defer span.Finish()

	conn, client, err := s.grpcUsersClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("creating grpc client"))
	}
	defer conn.Close()

	user, err := client.GetUserByID(ctx, &users.GetByIDRequest{UserID: userUUID.String()})
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client responded with error"))
	}

	res, err := fromProto(user.GetUser())
	if err != nil {
		return nil, errors.Join(err, errors.New("transforming grpc client response"))
	}

	return res, nil
}

func (s *ServiceImpl) GetSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_users_service.GetSessionByID")
	defer span.Finish()

	conn, client, err := s.grpcSessionsClientFactory(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("creating grpc client"))
	}
	defer conn.Close()

	sessionByID, err := client.GetSessionByID(ctx, &sessions.GetSessionByIDRequest{SessionID: sessionID})
	if err != nil {
		return nil, errors.Join(err, errors.New("grpc client responded with error"))
	}

	sess := &Session{}
	sess, err = sess.FromProto(sessionByID.GetSession())
	if err != nil {
		return nil, errors.Join(err, errors.New("transforming grpc client response"))
	}

	return sess, nil
}
