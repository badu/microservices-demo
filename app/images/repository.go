package images

import (
	"context"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const (
	createImageQuery = `INSERT INTO images (image_url, is_uploaded) VALUES ($1, $2) RETURNING image_id, image_url, is_uploaded, created_at`

	getImageByIDQuery = `SELECT image_id, image_url, is_uploaded, created_at, updated_at FROM images WHERE image_id = $1`
)

type RepositoryImpl struct {
	pgxPool *pgxpool.Pool
}

func NewRepository(pgxPool *pgxpool.Pool) RepositoryImpl {
	return RepositoryImpl{pgxPool: pgxPool}
}

func (r *RepositoryImpl) Create(ctx context.Context, msg *ImageDO) (*ImageDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepositoryImpl.Create")
	defer span.Finish()

	var res ImageDO
	if err := r.pgxPool.QueryRow(
		ctx,
		createImageQuery,
		msg.ImageURL,
		msg.IsUploaded,
	).Scan(&res.ImageID, &res.ImageURL, &res.IsUploaded, &res.CreatedAt); err != nil {
		return nil, errors.Wrap(err, "RepositoryImpl.Scan")
	}

	return &res, nil
}
func (r *RepositoryImpl) GetImageByID(ctx context.Context, imageID uuid.UUID) (*ImageDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "RepositoryImpl.GetImageByID")
	defer span.Finish()

	var img ImageDO
	if err := r.pgxPool.QueryRow(ctx, getImageByIDQuery, imageID).Scan(
		&img.ImageID,
		&img.ImageURL,
		&img.IsUploaded,
		&img.CreatedAt,
		&img.UpdatedAt,
	); err != nil {
		return nil, errors.Wrap(err, "RepositoryImpl.Scan")
	}

	return &img, nil
}
