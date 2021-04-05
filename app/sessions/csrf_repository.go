package sessions

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

type CSRFRepository interface {
	Create(ctx context.Context, token string) error
	GetToken(ctx context.Context, token string) (string, error)
}

type csrfRepositoryImpl struct {
	redis    *redis.Client
	prefix   string
	duration time.Duration
}

func NewCsrfRepository(redis *redis.Client, prefix string, duration time.Duration) *csrfRepositoryImpl {
	return &csrfRepositoryImpl{redis: redis, prefix: prefix, duration: duration}
}

// Create csrf token
func (r *csrfRepositoryImpl) Create(ctx context.Context, token string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrfRepositoryImpl.Create")
	defer span.Finish()

	if err := r.redis.SetEX(ctx, r.createKey(token), token, r.duration).Err(); err != nil {
		return errors.Wrap(err, "csrfRepositoryImpl.Create.redis.SetEX")
	}

	return nil
}

// Check csrf token
func (r *csrfRepositoryImpl) GetToken(ctx context.Context, token string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "csrfRepositoryImpl.Check")
	defer span.Finish()

	token, err := r.redis.Get(ctx, r.createKey(token)).Result()
	if err != nil {
		return "", err
	}

	return token, nil
}

func (r *csrfRepositoryImpl) createKey(token string) string {
	return fmt.Sprintf("%s: %s", r.prefix, token)
}
