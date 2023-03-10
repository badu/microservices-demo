package comments

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
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_comments_repository.CommentByID")
	defer span.Finish()

	rawComment, err := c.client.Get(ctx, c.createKey(commentID)).Bytes()
	if err != nil {
		return nil, errors.Join(err, errors.New("repository redis client getting comment by id"))
	}

	var result Comment
	if err := json.Unmarshal(rawComment, &result); err != nil {
		return nil, errors.Join(err, errors.New("repository unmarshalling json from redis"))
	}
	return &result, nil
}

func (c *RepositoryImpl) SetComment(ctx context.Context, comment *Comment) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_comments_repository.SetComment")
	defer span.Finish()

	jsonComment, err := json.Marshal(comment)
	if err != nil {
		return errors.Join(err, errors.New("repository marshalling json for redis"))
	}

	if err := c.client.SetEX(ctx, c.createKey(comment.CommentID), string(jsonComment), expiration).Err(); err != nil {
		return errors.Join(err, errors.New("repository storing comment in redis"))
	}

	return nil
}

func (c *RepositoryImpl) DeleteComment(ctx context.Context, commentID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_comments_repository.DeleteComment")
	defer span.Finish()

	if err := c.client.Del(ctx, c.createKey(commentID)).Err(); err != nil {
		return errors.Join(err, errors.New("repository deleting comment from redis"))
	}

	return nil
}

func (c *RepositoryImpl) createKey(commID uuid.UUID) string {
	return fmt.Sprintf("%s: %s", prefix, commID.String())
}
