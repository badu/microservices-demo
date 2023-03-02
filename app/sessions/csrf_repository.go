package sessions

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

type CsrfRepositoryImpl struct {
	redis    *redis.Client
	prefix   string
	duration time.Duration
}

func NewCSRFRepository(redis *redis.Client, prefix string, duration time.Duration) CsrfRepositoryImpl {
	return CsrfRepositoryImpl{redis: redis, prefix: prefix, duration: duration}
}

// Create csrf token
func (r *CsrfRepositoryImpl) Create(ctx context.Context, token string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CsrfRepositoryImpl.Create")
	defer span.Finish()

	if err := r.redis.SetEX(ctx, r.createKey(token), token, r.duration).Err(); err != nil {
		return errors.Wrap(err, "CsrfRepositoryImpl.Create.redis.SetEX")
	}

	return nil
}

// Check csrf token
func (r *CsrfRepositoryImpl) GetToken(ctx context.Context, token string) (string, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "CsrfRepositoryImpl.Check")
	defer span.Finish()

	token, err := r.redis.Get(ctx, r.createKey(token)).Result()
	if err != nil {
		return "", err
	}

	return token, nil
}

func (r *CsrfRepositoryImpl) createKey(token string) string {
	return fmt.Sprintf("%s: %s", r.prefix, token)
}
