package users

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

type RedisRepositoryImpl struct {
	client     *redis.Client
	prefix     string
	expiration time.Duration
}

func NewRedisRepository(redisConn *redis.Client, prefix string, expiration time.Duration) RedisRepositoryImpl {
	return RedisRepositoryImpl{client: redisConn, prefix: prefix, expiration: expiration}
}

func (u *RedisRepositoryImpl) SaveUser(ctx context.Context, user *UserResponse) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRedisRepository.SaveUser")
	defer span.Finish()

	userBytes, err := json.Marshal(user)
	if err != nil {
		return errors.Wrap(err, "userRedisRepository.SaveUser.json.Marshal")
	}

	if err := u.client.SetEX(ctx, u.createKey(user.UserID), string(userBytes), u.expiration).Err(); err != nil {
		return errors.Wrap(err, "userRedisRepository.SaveUser.client.SetEX")
	}

	return nil
}

func (u *RedisRepositoryImpl) GetUserByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRedisRepository.GetUserByID")
	defer span.Finish()

	result, err := u.client.Get(ctx, u.createKey(userID)).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "userRedisRepository.GetUserByID.client.Get")
	}

	var res UserResponse
	if err := json.Unmarshal(result, &res); err != nil {
		return nil, errors.Wrap(err, "userRedisRepository.GetUserByID.json.Unmarshal")
	}
	return &res, nil
}

func (u *RedisRepositoryImpl) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "userRedisRepository.DeleteUser")
	defer span.Finish()

	if err := u.client.Del(ctx, u.createKey(userID)).Err(); err != nil {
		return errors.Wrap(err, "userRedisRepository.GetUserByID.client.Del")
	}

	return nil
}

func (u *RedisRepositoryImpl) createKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s: %s", u.prefix, userID)
}
