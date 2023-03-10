package hotels

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

func NewRepository(redisClient *redis.Client) RepositoryImpl {
	return RepositoryImpl{client: redisClient}
}

func (h *RepositoryImpl) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*Hotel, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_hotels_repository.GetHotelByID")
	defer span.Finish()

	rawJson, err := h.client.Get(ctx, h.createKey(hotelID)).Bytes()
	if err != nil {
		return nil, errors.Join(err, errors.New("getting hotel from redis in repository"))
	}

	var result Hotel
	if err := json.Unmarshal(rawJson, &result); err != nil {
		return nil, errors.Join(err, errors.New("unmarshalling hotel from redis in repository"))
	}

	return &result, nil
}

func (h *RepositoryImpl) SetHotel(ctx context.Context, hotel *Hotel) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_hotels_repository.SetHotel")
	defer span.Finish()

	rawJson, err := json.Marshal(hotel)
	if err != nil {
		return errors.Join(err, errors.New("marshalling hotel to json for redis in repository"))
	}

	if err := h.client.SetEX(ctx, h.createKey(hotel.HotelID), string(rawJson), expiration).Err(); err != nil {
		return errors.Join(err, errors.New("saving hotel as json to redis in repository"))
	}

	return nil
}

func (h *RepositoryImpl) DeleteHotel(ctx context.Context, hotelID uuid.UUID) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "gateway_hotels_repository.DeleteHotel")
	defer span.Finish()

	if err := h.client.Del(ctx, h.createKey(hotelID)).Err(); err != nil {
		return errors.Join(err, errors.New("deleting hotel from redis in repository"))
	}

	return nil
}

func (h *RepositoryImpl) createKey(hotelID uuid.UUID) string {
	return fmt.Sprintf("%s: %s", prefix, hotelID.String())
}
