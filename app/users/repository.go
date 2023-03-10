package users

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/opentracing/opentracing-go"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/images"
)

const (
	createUserQuery = `INSERT INTO users (first_name, last_name, email, password, avatar, role) 
	VALUES ($1,$2,$3,$4,$5,$6) 
	RETURNING user_id, first_name, last_name, email, avatar, role, updated_at, created_at`

	getUserByIDQuery = `SELECT user_id, first_name, last_name, email, avatar, role, updated_at, created_at FROM users WHERE user_id = $1`

	getUserByEmail = `SELECT user_id, first_name, last_name, email, password, avatar, role, updated_at, created_at 
	FROM users WHERE email = $1`

	updateUserQuery = `UPDATE users 
		SET first_name = COALESCE(NULLIF($1, ''), first_name), 
	    last_name = COALESCE(NULLIF($2, ''), last_name), 
	    email = COALESCE(NULLIF($3, ''), email), 
	    role = COALESCE(NULLIF($4, '')::role, role)
		WHERE user_id = $5
	    RETURNING user_id, first_name, last_name, email, role, avatar, updated_at, created_at`

	updateAvatarQuery = `UPDATE users SET avatar = $1 WHERE user_id = $2 
	RETURNING user_id, first_name, last_name, email, role, avatar, updated_at, created_at`
)

type RepositoryImpl struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) RepositoryImpl {
	return RepositoryImpl{db: db}
}

// Create new user
func (u *RepositoryImpl) Create(ctx context.Context, user *UserDO) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_repository.Create")
	defer span.Finish()

	var created UserResponse
	if err := u.db.QueryRow(
		ctx,
		createUserQuery,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
		&user.Avatar,
		&user.Role,
	).Scan(&created.UserID, &created.FirstName, &created.LastName, &created.Email,
		&created.Avatar, &created.Role, &created.UpdatedAt, &created.CreatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("while scanning result"))
	}

	return &created, nil
}

// Get user by id
func (u *RepositoryImpl) GetByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_repository.GetByID")
	defer span.Finish()

	var res UserResponse
	if err := u.db.QueryRow(ctx, getUserByIDQuery, userID).Scan(
		&res.UserID,
		&res.FirstName,
		&res.LastName,
		&res.Email,
		&res.Avatar,
		&res.Role,
		&res.UpdatedAt,
		&res.CreatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("while scanning result"))
	}

	return &res, nil
}

func (u *RepositoryImpl) GetByEmail(ctx context.Context, email string) (*UserDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_repository.GetByEmail")
	defer span.Finish()

	var res UserDO
	if err := u.db.QueryRow(ctx, getUserByEmail, email).Scan(
		&res.UserID,
		&res.FirstName,
		&res.LastName,
		&res.Email,
		&res.Password,
		&res.Avatar,
		&res.Role,
		&res.UpdatedAt,
		&res.CreatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("while scanning result"))
	}

	return &res, nil
}

func (u *RepositoryImpl) Update(ctx context.Context, user *UserUpdate) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_repository.Update")
	defer span.Finish()

	var res UserResponse
	if err := u.db.QueryRow(ctx, updateUserQuery, &user.FirstName, &user.LastName, &user.Email, &user.Role, &user.UserID).
		Scan(
			&res.UserID,
			&res.FirstName,
			&res.LastName,
			&res.Email,
			&res.Role,
			&res.Avatar,
			&res.UpdatedAt,
			&res.CreatedAt,
		); err != nil {
		return nil, errors.Join(err, errors.New("while scanning result"))
	}

	return &res, nil
}

func (u *RepositoryImpl) UpdateAvatar(ctx context.Context, msg images.UploadedImageMsg) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_repository.UpdateUploadedAvatar")
	defer span.Finish()

	log.Printf("REPO  IMAGE: %v", msg)
	var res UserResponse
	if err := u.db.QueryRow(ctx, updateAvatarQuery, &msg.ImageURL, &msg.UserID).Scan(
		&res.UserID,
		&res.FirstName,
		&res.LastName,
		&res.Email,
		&res.Role,
		&res.Avatar,
		&res.UpdatedAt,
		&res.CreatedAt,
	); err != nil {
		return nil, errors.Join(err, errors.New("while scanning result"))
	}

	return &res, nil
}

func (u *RepositoryImpl) GetUsersByIDs(ctx context.Context, userIDs []string) ([]*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "users_repository.GetUsersByIDs")
	defer span.Finish()

	rows, err := u.db.Query(ctx, "SELECT user_id, first_name, last_name, email, avatar, role, updated_at, created_at FROM users WHERE user_id IN (SELECT UNNEST($1::uuid))", userIDs)
	if err != nil {
		return nil, errors.Join(err, errors.New("while querying"))
	}
	defer rows.Close()

	users := make([]*UserResponse, 0, len(userIDs))
	for rows.Next() {
		var res UserResponse
		if err := rows.Scan(
			&res.UserID,
			&res.FirstName,
			&res.LastName,
			&res.Email,
			&res.Avatar,
			&res.Role,
			&res.UpdatedAt,
			&res.CreatedAt,
		); err != nil {
			return nil, errors.Join(err, errors.New("while scanning result"))
		}
		users = append(users, &res)
	}

	return users, nil
}
