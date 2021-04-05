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

type Repository interface {
	GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*Hotel, error)
	SetHotel(ctx context.Context, hotel *Hotel) error
	DeleteHotel(ctx context.Context, hotelID uuid.UUID) error
}

const (
	prefix     = "comments:"
	expiration = time.Second * 3600
)

type repositoryImpl struct {
	redisConn *redis.Client
}

func NewRepository(redisConn *redis.Client) *repositoryImpl {
	return &repositoryImpl{redisConn: redisConn}
}

// GetHotelByID
func (h *repositoryImpl) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.GetHotelByID")
	defer span.Finish()

	result, err := h.redisConn.Get(ctx, h.createKey(hotelID)).Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "repositoryImpl.GetHotelByID")
	}

	var res Hotel
	if err := json.Unmarshal(result, &res); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}

	return &res, nil
}

// SetHotel
func (h *repositoryImpl) SetHotel(ctx context.Context, hotel *Hotel) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.SetHotel")
	defer span.Finish()

	hotelBytes, err := json.Marshal(hotel)
	if err != nil {
		return errors.Wrap(err, "repositoryImpl.Marshal")
	}

	if err := h.redisConn.SetEX(ctx, h.createKey(hotel.HotelID), string(hotelBytes), expiration).Err(); err != nil {
		return errors.Wrap(err, "repositoryImpl.SetEX")
	}

	return nil
}

// DeleteHotel
func (h *repositoryImpl) DeleteHotel(ctx context.Context, hotelID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.DeleteHotel")
	defer span.Finish()

	if err := h.redisConn.Del(ctx, h.createKey(hotelID)).Err(); err != nil {
		return errors.Wrap(err, "repositoryImpl.DeleteHotel.Del")
	}

	return nil
}

func (h *repositoryImpl) createKey(hotelID uuid.UUID) string {
	return fmt.Sprintf("%s: %s", prefix, hotelID.String())
}
