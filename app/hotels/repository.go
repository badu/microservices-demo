package hotels

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/pkg/pagination"
)

const (
	getHotelByIDQuery = `SELECT hotel_id, email, name, location, description, comments_count, 
       	country, city, ((coordinates::POINT)[0])::decimal, ((coordinates::POINT)[1])::decimal, rating, photos, image, created_at, updated_at 
		FROM hotels WHERE hotel_id = $1`

	updateHotelQuery = `UPDATE hotels 
		SET email = COALESCE(NULLIF($1, ''), email), name = $2, location = $3, description = $4, 
	 	country = $5, city = $6, coordinates = ST_GeomFromEWKT($7)
		WHERE hotel_id = $8
	    RETURNING hotel_id, email, name, location, description, comments_count, 
       	country, city, ((coordinates::POINT)[0])::decimal, ((coordinates::POINT)[1])::decimal, rating, photos, image, created_at, updated_at`

	createHotelQuery = `INSERT INTO hotels (name, location, description, image, photos, coordinates, email, country, city, rating) 
	VALUES ($1, $2, $3, $4, $5, ST_GeomFromEWKT($6), $7, $8, $9, $10) RETURNING hotel_id, created_at, updated_at`

	getTotalHotelsCountQuery = `SELECT COUNT(*) as total FROM hotels`

	getHotelsQuery = `SELECT hotel_id, email, name, location, description, comments_count, 
       	country, city, ((coordinates::POINT)[0])::decimal, ((coordinates::POINT)[1])::decimal, rating, photos, image, created_at, updated_at 
       	FROM hotels OFFSET $1 LIMIT $2`
)

type RepositoryImpl struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) RepositoryImpl {
	return RepositoryImpl{db: db}
}

func (h *RepositoryImpl) CreateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_repository.CreateHotel")
	defer span.Finish()

	point := GeneratePointToGeoFromFloat64(*hotel.Latitude, *hotel.Longitude)

	var res HotelDO
	if err := h.db.QueryRow(
		ctx,
		createHotelQuery,
		hotel.Name,
		hotel.Location,
		hotel.Description,
		hotel.Image,
		hotel.Photos,
		point,
		hotel.Email,
		hotel.Country,
		hotel.City,
		hotel.Rating,
	).Scan(&res.HotelID, &res.CreatedAt, &res.UpdatedAt); err != nil {
		return nil, errors.Join(err, errors.New("while scanning"))
	}

	hotel.HotelID = res.HotelID
	hotel.CreatedAt = res.CreatedAt
	hotel.UpdatedAt = res.UpdatedAt

	return hotel, nil
}

func (h *RepositoryImpl) UpdateHotel(ctx context.Context, hotel *HotelDO) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_repository.UpdateHotel")
	defer span.Finish()

	point := GeneratePointToGeoFromFloat64(*hotel.Latitude, *hotel.Longitude)
	var res HotelDO
	if err := h.db.QueryRow(
		ctx,
		updateHotelQuery,
		hotel.Email,
		hotel.Name,
		hotel.Location,
		hotel.Description,
		hotel.Country,
		hotel.City,
		point,
		hotel.HotelID,
	).Scan(
		&res.HotelID,
		&res.Email,
		&res.Name,
		&res.Location,
		&res.Description,
		&res.CommentsCount,
		&res.Country,
		&res.City,
		&res.Latitude,
		&res.Longitude,
		&res.Rating,
		&res.Photos,
		&res.Image,
		&res.CreatedAt,
		&res.UpdatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("while scanning"))
	}

	return &res, nil
}

func (h *RepositoryImpl) GetHotelByID(ctx context.Context, hotelID uuid.UUID) (*HotelDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_repository.GetHotelByID")
	defer span.Finish()

	var hotel HotelDO
	if err := h.db.QueryRow(ctx, getHotelByIDQuery, hotelID).Scan(
		&hotel.HotelID,
		&hotel.Email,
		&hotel.Name,
		&hotel.Location,
		&hotel.Description,
		&hotel.CommentsCount,
		&hotel.Country,
		&hotel.City,
		&hotel.Latitude,
		&hotel.Longitude,
		&hotel.Rating,
		&hotel.Photos,
		&hotel.Image,
		&hotel.CreatedAt,
		&hotel.UpdatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("while scanning"))
	}

	return &hotel, nil
}

func (h *RepositoryImpl) GetHotels(ctx context.Context, query *pagination.Pagination) (*List, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_repository.GetHotels")
	defer span.Finish()

	var total int
	if err := h.db.QueryRow(ctx, getTotalHotelsCountQuery).Scan(&total); err != nil {
		return nil, errors.Join(err, errors.New("while scanning total"))
	}
	if total == 0 {
		return &List{
			TotalCount: total,
			TotalPages: 0,
			Page:       0,
			Size:       0,
			HasMore:    false,
			Hotels:     make([]*HotelDO, 0),
		}, nil
	}

	rows, err := h.db.Query(ctx, getHotelsQuery, query.GetOffset(), query.GetLimit())
	if err != nil {
		return nil, errors.Join(err, errors.New("while running query"))
	}
	defer rows.Close()

	hotels := make([]*HotelDO, 0, query.GetLimit())
	for rows.Next() {
		var hotel HotelDO
		if err := rows.Scan(
			&hotel.HotelID,
			&hotel.Email,
			&hotel.Name,
			&hotel.Location,
			&hotel.Description,
			&hotel.CommentsCount,
			&hotel.Country,
			&hotel.City,
			&hotel.Latitude,
			&hotel.Longitude,
			&hotel.Rating,
			&hotel.Photos,
			&hotel.Image,
			&hotel.CreatedAt,
			&hotel.UpdatedAt,
		); err != nil {
			return nil, errors.Join(err, errors.New("while scanning"))
		}
		hotels = append(hotels, &hotel)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Join(err, errors.New("while rows"))
	}

	log.Printf("HOTELS: %-v", hotels)

	return &List{
		TotalCount: total,
		TotalPages: query.GetTotalPages(total),
		Page:       query.GetPage(),
		Size:       query.GetSize(),
		HasMore:    query.GetHasMore(total),
		Hotels:     hotels,
	}, nil
}

func (h *RepositoryImpl) UpdateHotelImage(ctx context.Context, hotelID uuid.UUID, imageURL string) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, "hotels_repository.UpdateHotelImage")
	defer span.Finish()

	updateHotelImageQuery := `UPDATE hotels SET image = $1 WHERE hotel_id = $2`

	result, err := h.db.Exec(ctx, updateHotelImageQuery, imageURL, hotelID)
	if err != nil {
		return errors.Join(err, errors.New("while performing update"))
	}

	if result.RowsAffected() == 0 {
		return errors.Join(ErrHotelNotFound, errors.New("no rows were affected while performing update"))
	}

	return nil
}

func GeneratePointToGeoFromFloat64(latitude float64, longitude float64) string {
	return fmt.Sprintf("SRID=4326;POINT(%f %f)", latitude, longitude)
}
