package sessions

import (
	"context"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
)

type Service interface {
	CreateSession(ctx context.Context, userID uuid.UUID) (*SessionDO, error)
	GetSessionByID(ctx context.Context, sessID string) (*SessionDO, error)
	DeleteSession(ctx context.Context, sessID string) error
}

type serviceImpl struct {
	sessRepo Repository
}

func NewSessionUseCase(sessRepo Repository) *serviceImpl {
	return &serviceImpl{sessRepo: sessRepo}
}

func (s *serviceImpl) CreateSession(ctx context.Context, userID uuid.UUID) (*SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.CreateSession")
	defer span.Finish()
	return s.sessRepo.CreateSession(ctx, userID)
}

func (s *serviceImpl) GetSessionByID(ctx context.Context, sessID string) (*SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "serviceImpl.GetSessionByID")
	defer span.Finish()
	return s.sessRepo.GetSessionByID(ctx, sessID)
}

func (s *serviceImpl) DeleteSession(ctx context.Context, sessID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DeleteSession.GetSessionByID")
	defer span.Finish()
	return s.sessRepo.DeleteSession(ctx, sessID)
}
