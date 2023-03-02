package sessions

import (
	"context"

	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
)

type Repository interface {
	CreateSession(ctx context.Context, userID uuid.UUID) (*SessionDO, error)
	GetSessionByID(ctx context.Context, sessID string) (*SessionDO, error)
	DeleteSession(ctx context.Context, sessID string) error
}

type ServiceImpl struct {
	repo Repository
}

func NewService(sessRepo Repository) ServiceImpl {
	return ServiceImpl{repo: sessRepo}
}

func (s *ServiceImpl) CreateSession(ctx context.Context, userID uuid.UUID) (*SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.CreateSession")
	defer span.Finish()
	return s.repo.CreateSession(ctx, userID)
}

func (s *ServiceImpl) GetSessionByID(ctx context.Context, sessID string) (*SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "ServiceImpl.GetSessionByID")
	defer span.Finish()
	return s.repo.GetSessionByID(ctx, sessID)
}

func (s *ServiceImpl) DeleteSession(ctx context.Context, sessID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "DeleteSession.GetSessionByID")
	defer span.Finish()
	return s.repo.DeleteSession(ctx, sessID)
}
