package comments

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

const (
	prefix     = "comments:"
	expiration = time.Second * 3600
)

type RepositoryImpl struct {
	client *redis.Client
}

func NewRepository(redisConn *redis.Client) RepositoryImpl {
	return RepositoryImpl{client: redisConn}
}

func (c *RepositoryImpl) CommentByID(ctx context.Context, commentID uuid.UUID) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepositoryImpl.CommentByID")
	defer span.Finish()

	result, err := c.client.Get(ctx, c.createKey(commentID)).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "RepositoryImpl.CommentByID.Get")
	}

	var res Comment
	if err := json.Unmarshal(result, &res); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	return &res, nil
}

func (c *RepositoryImpl) SetComment(ctx context.Context, comment *Comment) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepositoryImpl.SetComment")
	defer span.Finish()

	commBytes, err := json.Marshal(comment)
	if err != nil {
		return errors.Wrap(err, "RepositoryImpl.Marshal")
	}

	if err := c.client.SetEX(ctx, c.createKey(comment.CommentID), string(commBytes), expiration).Err(); err != nil {
		return errors.Wrap(err, "RepositoryImpl.SetComment.SetEX")
	}

	return nil
}

func (c *RepositoryImpl) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepositoryImpl.DeleteComment")
	defer span.Finish()

	if err := c.client.Del(ctx, c.createKey(commentID)).Err(); err != nil {
		return errors.Wrap(err, "RepositoryImpl.DeleteComment.Del")
	}

	return nil
}

func (c *RepositoryImpl) createKey(commID uuid.UUID) string {
	return fmt.Sprintf("%s: %s", prefix, commID.String())
}
