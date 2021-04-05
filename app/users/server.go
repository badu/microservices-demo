package users

import (
	"context"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	grpcErrors "github.com/badu/microservices-demo/pkg/grpc_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type serverImpl struct {
	service Service
	logger  logger.Logger
}

func NewServer(service Service, logger logger.Logger) *serverImpl {
	return &serverImpl{service: service, logger: logger}
}

func (e *serverImpl) GetUserByID(ctx context.Context, r *GetByIDRequest) (*GetByIDResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serverImpl.GetUserByID")
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

func (e *serverImpl) GetUsersByIDs(ctx context.Context, req *GetByIDsReq) (*GetByIDsRes, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serverImpl.GetUserByID")
	defer span.Finish()

	usersByIDs, err := e.service.GetUsersByIDs(ctx, req.GetUsersIDs())
	if err != nil {
		e.logger.Errorf("service.GetUsersByIDs: %v", err)
		return nil, grpcErrors.ErrorResponse(err, "service.GetUsersByIDs")
	}

	response := idsToUUID(usersByIDs)
	e.logger.Infof("USERS LIST RESPONSE: %v", response)

	return &GetByIDsRes{Users: response}, nil
}

func idsToUUID(users []*UserResponse) []*User {
	usersList := make([]*User, 0, len(users))
	for _, val := range users {
		usersList = append(usersList, val.ToProto())
	}

	return usersList
}
