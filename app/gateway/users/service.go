package users

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.GetByID")
	defer span.Finish()

	conn, client, err := s.grpcUsersClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.GetByID")
	}
	defer conn.Close()

	user, err := client.GetUserByID(ctx, &users.GetByIDRequest{UserID: userUUID.String()})
	if err != nil {
		return nil, errors.Wrap(err, "userService.GetUserByID")
	}

	res, err := UserFromProto(user.GetUser())
	if err != nil {
		return nil, errors.Wrap(err, "UserFromProto")
	}

	return res, nil
}

func (s *ServiceImpl) GetSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.GetSessionByID")
	defer span.Finish()

	conn, client, err := s.grpcSessionsClientFactory(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ServiceImpl.GetSessionByID")
	}
	defer conn.Close()

	sessionByID, err := client.GetSessionByID(ctx, &sessions.GetSessionByIDRequest{SessionID: sessionID})
	if err != nil {
		return nil, errors.Wrap(err, "sessClient.GetSessionByID")
	}

	sess := &Session{}
	sess, err = sess.FromProto(sessionByID.GetSession())
	if err != nil {
		return nil, errors.Wrap(err, "sess.FromProto")
	}

	return sess, nil
}
