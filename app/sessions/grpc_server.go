package sessions

import (
	"context"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
	"google.golang.org/grpc/status"

	grpcErrors "github.com/badu/microservices-demo/pkg/grpc_errors"
	"github.com/badu/microservices-demo/pkg/logger"
)

type Service interface {
	CreateSession(ctx context.Context, userID uuid.UUID) (*SessionDO, error)
	GetSessionByID(ctx context.Context, sessID string) (*SessionDO, error)
	DeleteSession(ctx context.Context, sessID string) error
}

type ServerImpl struct {
	logger      logger.Logger
	service     Service
	csrfService CSRFService
}

func NewServer(logger logger.Logger, service Service, csrfService CSRFService) ServerImpl {
	return ServerImpl{logger: logger, service: service, csrfService: csrfService}
}

func (s *ServerImpl) CreateSession(ctx context.Context, r *CreateSessionRequest) (*CreateSessionResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrf_grpc_server.CreateSession")
	defer span.Finish()

	userUUID, err := uuid.FromString(r.UserID)
	if err != nil {
		s.logger.Errorf("uuid.FromString: %v", err)
		return nil, status.Errorf(grpcErrors.ParseGRPCErrStatusCode(err), "uuid.FromString: %v", err)
	}
	sess, err := s.service.CreateSession(ctx, userUUID)
	if err != nil {
		s.logger.Errorf("service.CreateSession: %v", err)
		return nil, status.Errorf(grpcErrors.ParseGRPCErrStatusCode(err), "service.CreateSession: %v", err)
	}

	return &CreateSessionResponse{Session: s.sessionJSONToProto(sess)}, nil
}

func (s *ServerImpl) GetSessionByID(ctx context.Context, r *GetSessionByIDRequest) (*GetSessionByIDResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrf_grpc_server.GetSessionByID")
	defer span.Finish()

	sess, err := s.service.GetSessionByID(ctx, r.SessionID)
	if err != nil {
		s.logger.Errorf("service.GetSessionByID: %v", err)
		return nil, status.Errorf(grpcErrors.ParseGRPCErrStatusCode(err), "service.GetSessionByID: %v", err)
	}

	return &GetSessionByIDResponse{Session: s.sessionJSONToProto(sess)}, nil
}

func (s *ServerImpl) DeleteSession(ctx context.Context, r *DeleteSessionRequest) (*DeleteSessionResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrf_grpc_server.DeleteSession")
	defer span.Finish()

	if err := s.service.DeleteSession(ctx, r.SessionID); err != nil {
		return nil, status.Errorf(grpcErrors.ParseGRPCErrStatusCode(err), "service.DeleteSession: %v", err)
	}

	return &DeleteSessionResponse{SessionID: r.SessionID}, nil
}

func (s *ServerImpl) CreateCsrfToken(ctx context.Context, r *CreateCsrfTokenRequest) (*CreateCsrfTokenResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrf_grpc_server.CreateCsrfToken")
	defer span.Finish()

	token, err := s.csrfService.GetCSRFToken(ctx, r.GetCsrfTokenInput().GetSessionID())
	if err != nil {
		return nil, status.Errorf(grpcErrors.ParseGRPCErrStatusCode(err), "csrfService.CreateCsrfToken: %v", err)
	}

	return &CreateCsrfTokenResponse{CsrfToken: &CsrfToken{Token: token}}, nil
}

func (s *ServerImpl) CheckCsrfToken(ctx context.Context, r *CheckCsrfTokenRequest) (*CheckCsrfTokenResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrf_grpc_server.CheckCsrfToken")
	defer span.Finish()

	isValid, err := s.csrfService.ValidateCSRFToken(ctx, r.GetCsrfTokenCheck().GetSessionID(), r.GetCsrfTokenCheck().GetToken())
	if err != nil {
		return nil, status.Errorf(grpcErrors.ParseGRPCErrStatusCode(err), "csrfService.CheckToken: %v", err)
	}

	return &CheckCsrfTokenResponse{CheckResult: &CheckResult{Result: isValid}}, nil
}

func (s *ServerImpl) sessionJSONToProto(sess *SessionDO) *Session {
	return &Session{
		UserID:    sess.UserID.String(),
		SessionID: sess.SessionID,
	}
}
