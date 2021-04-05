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

type Repository interface {
	CommentByID(ctx context.Context, commentID uuid.UUID) (*Comment, error)
	SetComment(ctx context.Context, comment *Comment) error
	DeleteComment(ctx context.Context, commentID uuid.UUID) error
}

const (
	prefix     = "comments:"
	expiration = time.Second * 3600
)

type repositoryImpl struct {
	client *redis.Client
}

func NewRepository(redisConn *redis.Client) *repositoryImpl {
	return &repositoryImpl{client: redisConn}
}

func (c *repositoryImpl) CommentByID(ctx context.Context, commentID uuid.UUID) (*Comment, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.CommentByID")
	defer span.Finish()

	result, err := c.client.Get(ctx, c.createKey(commentID)).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "repositoryImpl.CommentByID.Get")
	}

	var res Comment
	if err := json.Unmarshal(result, &res); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	return &res, nil
}

func (c *repositoryImpl) SetComment(ctx context.Context, comment *Comment) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.SetComment")
	defer span.Finish()

	commBytes, err := json.Marshal(comment)
	if err != nil {
		return errors.Wrap(err, "repositoryImpl.Marshal")
	}

	if err := c.client.SetEX(ctx, c.createKey(comment.CommentID), string(commBytes), expiration).Err(); err != nil {
		return errors.Wrap(err, "repositoryImpl.SetComment.SetEX")
	}

	return nil
}

func (c *repositoryImpl) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.DeleteComment")
	defer span.Finish()

	if err := c.client.Del(ctx, c.createKey(commentID)).Err(); err != nil {
		return errors.Wrap(err, "repositoryImpl.DeleteComment.Del")
	}

	return nil
}

func (c *repositoryImpl) createKey(commID uuid.UUID) string {
	return fmt.Sprintf("%s: %s", prefix, commID.String())
}
