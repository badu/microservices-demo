package users

import (
	"context"
	"errors"

	"github.com/badu/bus"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/gateway/events"
	"github.com/badu/microservices-demo/app/sessions"
	"github.com/badu/microservices-demo/app/users"
	"github.com/badu/microservices-demo/pkg/logger"
)

type ServiceImpl struct {
	logger logger.Logger
}

func NewService(
	logger logger.Logger,
) ServiceImpl {
	return ServiceImpl{
		logger: logger,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, userUUID uuid.UUID) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_users_service.GetByID")
	defer span.Finish()
	event := events.NewRequireUsersGRPCClient(ctx)
	bus.Pub(event)
	event.WaitReply()
	if event.Err != nil {
		return nil, event.Err
	}

	defer event.Conn.Close()

	user, err := event.Client.GetUserByID(ctx, &users.GetByIDRequest{UserID: userUUID.String()})
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

	event := events.NewRequireSessionsGRPCClient(ctx)
	bus.Pub(event)
	event.WaitReply()
	if event.Err != nil {
		return nil, event.Err
	}

	defer event.Conn.Close()

	sessionByID, err := event.Client.GetSessionByID(ctx, &sessions.GetSessionByIDRequest{SessionID: sessionID})
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
