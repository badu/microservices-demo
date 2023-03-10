package users

import (
	"context"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"

	"github.com/badu/microservices-demo/app/images"
	"github.com/badu/microservices-demo/app/sessions"
	grpcErrors "github.com/badu/microservices-demo/pkg/grpc_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	Register(ctx context.Context, user *UserDO) (*UserResponse, error)
	Login(ctx context.Context, login Login) (*UserDO, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	CreateSession(ctx context.Context, userID uuid.UUID) (string, error)
	GetSessionByID(ctx context.Context, sessionID string) (*sessions.SessionDO, error)
	GetCSRFToken(ctx context.Context, sessionID string) (string, error)
	DeleteSession(ctx context.Context, sessionID string) error
	Update(ctx context.Context, user *UserUpdate) (*UserResponse, error)
	UpdateUploadedAvatar(ctx context.Context, delivery amqp.Delivery) error
	UpdateAvatar(ctx context.Context, data *images.UpdateAvatarMsg) error
	GetUsersByIDs(ctx context.Context, userIDs []string) ([]*UserResponse, error)
}

type ServerImpl struct {
	service Service
	logger  logger.Logger
}

func NewServer(service Service, logger logger.Logger) ServerImpl {
	return ServerImpl{service: service, logger: logger}
}

func (e *ServerImpl) GetUserByID(ctx context.Context, r *GetByIDRequest) (*GetByIDResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_grpc_server.GetUserByID")
	defer span.Finish()

	userUUID, err := uuid.FromString(r.GetUserID())
	if err != nil {
		e.logger.Errorf("uuid.FromString: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "uuid.FromString")
	}

	foundUser, err := e.service.GetByID(ctx, userUUID)
	if err != nil {
		e.logger.Errorf("uuid.FromString: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "service.GetByID")
	}

	return &GetByIDResponse{User: foundUser.ToProto()}, nil
}

func (e *ServerImpl) GetUsersByIDs(ctx context.Context, req *GetUsersByIDsRequest) (*GetUsersByIDsResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_grpc_server.GetUserByID")
	defer span.Finish()

	usersByIDs, err := e.service.GetUsersByIDs(ctx, req.GetUsersIDs())
	if err != nil {
		e.logger.Errorf("service.GetUsersByIDs: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "service.GetUsersByIDs")
	}

	response := idsToUUID(usersByIDs)
	e.logger.Infof("USERS LIST RESPONSE: %v", response)

	return &GetUsersByIDsResponse{Users: response}, nil
}

func idsToUUID(users []*UserResponse) []*User {
	usersList := make([]*User, 0, len(users))
	for _, val := range users {
		usersList = append(usersList, val.ToProto())
	}

	return usersList
}
