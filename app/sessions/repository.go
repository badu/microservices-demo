package sessions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"
)

type RepositoryImpl struct {
	redis      *redis.Client
	prefix     string
	expiration time.Duration
}

func NewRepository(redis *redis.Client, prefix string, expiration time.Duration) RepositoryImpl {
	return RepositoryImpl{redis: redis, prefix: prefix, expiration: expiration}
}

func (s *RepositoryImpl) CreateSession(ctx context.Context, userID uuid.UUID) (*SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "sessions_repository.CreateSession")
	defer span.Finish()

	sess := &SessionDO{
		SessionID: uuid.NewV4().String(),
		UserID:    userID,
	}

	sessBytes, err := json.Marshal(&sess)
	if err != nil {
		return nil, errors.Join(err, errors.New("while marshalling json"))
	}

	if err := s.redis.SetEX(ctx, s.createKey(sess.SessionID), string(sessBytes), s.expiration).Err(); err != nil {
		return nil, errors.Join(err, errors.New("while saving to redis"))
	}

	return sess, nil
}

func (s *RepositoryImpl) GetSessionByID(ctx context.Context, sessID string) (*SessionDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "sessions_repository.GetSessionByID")
	defer span.Finish()

	result, err := s.redis.Get(ctx, s.createKey(sessID)).Result()
	if err != nil {
		return nil, errors.Join(err, errors.New("while reading from redis"))
	}

	var sess SessionDO
	if err := json.Unmarshal([]byte(result), &sess); err != nil {
		return nil, errors.Join(err, errors.New("while unmarshalling json"))
	}
	return &sess, nil
}

func (s *RepositoryImpl) DeleteSession(ctx context.Context, sessID string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "sessions_repository.DeleteSession")
	defer span.Finish()

	if err := s.redis.Del(ctx, s.createKey(sessID)).Err(); err != nil {
		return errors.Join(err, errors.New("while deleting from redis"))
	}
	return nil
}

func (s *RepositoryImpl) createKey(sessionID string) string {
	return fmt.Sprintf("%s: %s", s.prefix, sessionID)
}
