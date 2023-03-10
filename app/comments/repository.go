package comments

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/pkg/pagination"
)

const (
	createCommentQuery = `INSERT INTO comments (hotel_id, user_id, message, photos, rating) VALUES ($1, $2, $3, $4, $5) RETURNING comment_id, hotel_id, user_id, message, photos, rating, created_at, updated_at`

	getCommByIDQuery = `SELECT comment_id, hotel_id, user_id, message, photos, rating, created_at, updated_at FROM comments WHERE comment_id = $1`

	updateCommentQuery = `UPDATE comments SET message = COALESCE(NULLIF($1, ''), message), rating = $2, photos = $3
	WHERE comment_id = $4 RETURNING comment_id, hotel_id, user_id, message, photos, rating, created_at, updated_at`

	getTotalCountQuery = `SELECT count(comment_id) as total FROM comments WHERE hotel_id = $1`

	getCommentByHotelIDQuery = `SELECT comment_id, hotel_id, user_id, message, photos, rating, created_at, updated_at FROM comments WHERE hotel_id = $1 OFFSET $2 LIMIT $3`
)

type RepositoryImpl struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) RepositoryImpl {
	return RepositoryImpl{db: db}
}

func (c *RepositoryImpl) Create(ctx context.Context, comment *CommentDO) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "comments_repository.Create")
	defer span.Finish()

	var comm CommentDO
	if err := c.db.QueryRow(
		ctx,
		createCommentQuery,
		comment.HotelID,
		comment.UserID,
		comment.Message,
		comment.Photos,
		comment.Rating,
	).Scan(
		&comm.CommentID,
		&comm.HotelID,
		&comm.UserID,
		&comm.Message,
		&comm.Photos,
		&comm.Rating,
		&comm.CreatedAt,
		&comm.UpdatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("comments repository scan"))
	}

	return &comm, nil
}

func (c *RepositoryImpl) GetByID(ctx context.Context, commentID uuid.UUID) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "comments_repository.GetByID")
	defer span.Finish()

	var comm CommentDO
	if err := c.db.QueryRow(ctx, getCommByIDQuery, commentID).Scan(
		&comm.CommentID,
		&comm.HotelID,
		&comm.UserID,
		&comm.Message,
		&comm.Photos,
		&comm.Rating,
		&comm.CreatedAt,
		&comm.UpdatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("comments repository scan"))
	}

	return &comm, nil
}

func (c *RepositoryImpl) Update(ctx context.Context, comment *CommentDO) (*CommentDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "comments_repository.Update")
	defer span.Finish()

	var comm CommentDO
	if err := c.db.QueryRow(ctx, updateCommentQuery, comment.Message, comment.Rating, comment.Photos, comment.CommentID).Scan(
		&comm.CommentID,
		&comm.HotelID,
		&comm.UserID,
		&comm.Message,
		&comm.Photos,
		&comm.Rating,
		&comm.CreatedAt,
		&comm.UpdatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("comments repository scan"))
	}

	return &comm, nil
}

func (c *RepositoryImpl) GetByHotelID(ctx context.Context, hotelID uuid.UUID, query *pagination.Pagination) (*List, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "comments_repository.GetByHotelID")
	defer span.Finish()

	var totalCount int
	if err := c.db.QueryRow(ctx, getTotalCountQuery, hotelID).Scan(&totalCount); err != nil {
		return nil, errors.Join(err, errors.New("comments repository scan"))
	}

	if totalCount == 0 {
		return &List{
			TotalCount: 0,
			TotalPages: 0,
			Page:       0,
			Size:       0,
			HasMore:    false,
			Comments:   make([]*CommentDO, 0),
		}, nil
	}

	var commentsList []*CommentDO
	rows, err := c.db.Query(ctx, getCommentByHotelIDQuery, hotelID, query.GetOffset(), query.GetLimit())
	if err != nil {
		return nil, errors.Join(err, errors.New("comments repository query"))
	}
	defer rows.Close()

	for rows.Next() {
		var comm CommentDO
		if err := rows.Scan(
			&comm.CommentID,
			&comm.HotelID,
			&comm.UserID,
			&comm.Message,
			&comm.Photos,
			&comm.Rating,
			&comm.CreatedAt,
			&comm.UpdatedAt,
		); err != nil {
			return nil, errors.Join(err, errors.New("comments repository scan"))
		}
		commentsList = append(commentsList, &comm)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Join(err, errors.New("comments repository rows"))
	}

	return &List{
		TotalCount: totalCount,
		TotalPages: query.GetTotalPages(totalCount),
		Page:       query.GetPage(),
		Size:       query.GetSize(),
		HasMore:    query.GetHasMore(totalCount),
		Comments:   commentsList,
	}, nil
}
