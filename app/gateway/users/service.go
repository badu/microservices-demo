package users

import (
	"context"

	"github.com/badu/microservices-demo/pkg/config"
	"github.com/badu/microservices-demo/pkg/grpc_client"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/sessions"
	"github.com/badu/microservices-demo/app/users"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	GetByID(ctx context.Context, userUUID uuid.UUID) (*UserResponse, error)
	GetSessionByID(ctx context.Context, sessionID string) (*Session, error)
}

type serviceImpl struct {
	mw               *grpc_client.ClientMiddleware
	usersServicePort string
	authServicePort  string
	logger           logger.Logger
}

func NewService(
	logger logger.Logger,
	cfg *config.Config,
	tracer opentracing.Tracer,
) *serviceImpl {
	return &serviceImpl{usersServicePort: cfg.GRPC.UserServicePort, authServicePort: cfg.GRPC.SessionServicePort, mw: grpc_client.NewClientMiddleware(logger, cfg, tracer), logger: logger}
}

func (s *serviceImpl) GetByID(ctx context.Context, userUUID uuid.UUID) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.GetByID")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.usersServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "userService.GetUserByID")
	}
	defer conn.Close()

	client := users.NewUserServiceClient(conn)

	user, err := client.GetUserByID(ctx, &users.GetByIDRequest{UserID: userUUID.String()})
	if err != nil {
		return nil, errors.Wrap(err, "userService.GetUserByID")
	}

	res, err := UserFromProtoRes(user.GetUser())
	if err != nil {
		return nil, errors.Wrap(err, "UserFromProtoRes")
	}

	return res, nil
}

func (s *serviceImpl) GetSessionByID(ctx context.Context, sessionID string) (*Session, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.GetSessionByID")
	defer span.Finish()

	conn, err := grpc_client.NewGRPCClientServiceConn(ctx, s.mw, s.authServicePort)
	if err != nil {
		return nil, errors.Wrap(err, "userService.GetUserByID")
	}
	defer conn.Close()

	client := sessions.NewAuthorizationServiceClient(conn)

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
