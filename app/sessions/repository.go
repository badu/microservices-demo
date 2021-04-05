package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

type Repository interface {
	CreateSession(ctx context.Context, userID uuid.UUID) (*SessionDO, error)
	GetSessionByID(ctx context.Context, sessID string) (*SessionDO, error)
	DeleteSession(ctx context.Context, sessID string) error
}

type repositoryImpl struct {
	redis      *redis.Client
	prefix     string
	expiration time.Duration
}

func NewSessionRedisRepo(redis *redis.Client, prefix string, expiration time.Duration) *repositoryImpl {
	return &repositoryImpl{redis: redis, prefix: prefix, expiration: expiration}
}

func (s *repositoryImpl) CreateSession(ctx context.Context, userID uuid.UUID) (*SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.CreateSession")
	defer span.Finish()

	sess := &SessionDO{
		SessionID: uuid.NewV4().String(),
		UserID:    userID,
	}

	sessBytes, err := json.Marshal(&sess)
	if err != nil {
		return nil, errors.Wrap(err, "sessionRepo.CreateSession.json.Marshal")
	}

	if err := s.redis.SetEX(ctx, s.createKey(sess.SessionID), string(sessBytes), s.expiration).Err(); err != nil {
		return nil, errors.Wrap(err, "sessionRepo.CreateSession.redis.SetEX")
	}

	return sess, nil
}

func (s *repositoryImpl) GetSessionByID(ctx context.Context, sessID string) (*SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.GetSessionByID")
	defer span.Finish()

	result, err := s.redis.Get(ctx, s.createKey(sessID)).Result()
	if err != nil {
		return nil, errors.Wrap(err, "sessionRepo.GetSessionByID.redis.Get")
	}

	var sess SessionDO
	if err := json.Unmarshal([]byte(result), &sess); err != nil {
		return nil, errors.Wrap(err, "sessionRepo.GetSessionByID.json.Unmarshal")
	}
	return &sess, nil
}

func (s *repositoryImpl) DeleteSession(ctx context.Context, sessID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.DeleteSession")
	defer span.Finish()

	if err := s.redis.Del(ctx, s.createKey(sessID)).Err(); err != nil {
		return errors.Wrap(err, "sessionRepo.DeleteSession.redis.Del")
	}
	return nil
}

func (s *repositoryImpl) createKey(sessionID string) string {
	return fmt.Sprintf("%s: %s", s.prefix, sessionID)
}
