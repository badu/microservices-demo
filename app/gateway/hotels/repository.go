package hotels

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
	redisConn *redis.Client
}

func NewRepository(redisConn *redis.Client) RepositoryImpl {
	return RepositoryImpl{redisConn: redisConn}
}

func (h *RepositoryImpl) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepositoryImpl.GetHotelByID")
	defer span.Finish()

	result, err := h.redisConn.Get(ctx, h.createKey(hotelID)).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "RepositoryImpl.GetHotelByID")
	}

	var res Hotel
	if err := json.Unmarshal(result, &res); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}

	return &res, nil
}

func (h *RepositoryImpl) SetHotel(ctx context.Context, hotel *Hotel) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepositoryImpl.SetHotel")
	defer span.Finish()

	hotelBytes, err := json.Marshal(hotel)
	if err != nil {
		return errors.Wrap(err, "RepositoryImpl.Marshal")
	}

	if err := h.redisConn.SetEX(ctx, h.createKey(hotel.HotelID), string(hotelBytes), expiration).Err(); err != nil {
		return errors.Wrap(err, "RepositoryImpl.SetEX")
	}

	return nil
}

func (h *RepositoryImpl) DeleteHotel(ctx context.Context, hotelID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepositoryImpl.DeleteHotel")
	defer span.Finish()

	if err := h.redisConn.Del(ctx, h.createKey(hotelID)).Err(); err != nil {
		return errors.Wrap(err, "RepositoryImpl.DeleteHotel.Del")
	}

	return nil
}

func (h *RepositoryImpl) createKey(hotelID uuid.UUID) string {
	return fmt.Sprintf("%s: %s", prefix, hotelID.String())
}
