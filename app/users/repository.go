package users

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/badu/microservices-demo/app/images"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

type Repository interface {
	Create(ctx context.Context, user *UserDO) (*UserResponse, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	GetByEmail(ctx context.Context, email string) (*UserDO, error)
	Update(ctx context.Context, user *UserUpdate) (*UserResponse, error)
	UpdateAvatar(ctx context.Context, msg images.UploadedImageMsg) (*UserResponse, error)
	GetUsersByIDs(ctx context.Context, userIDs []string) ([]*UserResponse, error)
}

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

type repositoryImpl struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *repositoryImpl {
	return &repositoryImpl{db: db}
}

// Create new user
func (u *repositoryImpl) Create(ctx context.Context, user *UserDO) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.Create")
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
		return nil, errors.Wrap(err, "Scan")
	}

	return &created, nil
}

// Get user by id
func (u *repositoryImpl) GetByID(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.GetByID")
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
		return nil, errors.Wrap(err, "Scan")
	}

	return &res, nil
}

// GetByEmail
func (u *repositoryImpl) GetByEmail(ctx context.Context, email string) (*UserDO, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.GetByEmail")
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
		return nil, errors.Wrap(err, "Scan")
	}

	return &res, nil
}

// Update
func (u *repositoryImpl) Update(ctx context.Context, user *UserUpdate) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.Update")
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
		return nil, errors.Wrap(err, "Scan")
	}

	return &res, nil
}

func (u *repositoryImpl) UpdateAvatar(ctx context.Context, msg images.UploadedImageMsg) (*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.UpdateUploadedAvatar")
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
		return nil, errors.Wrap(err, "Scan")
	}

	return &res, nil
}

func (u *repositoryImpl) GetUsersByIDs(ctx context.Context, userIDs []string) ([]*UserResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "repositoryImpl.GetUsersByIDs")
	defer span.Finish()

	placeholders := CreateSQLPlaceholders(len(userIDs))
	query := fmt.Sprintf("SELECT user_id, first_name, last_name, email, avatar, role, updated_at, created_at FROM users WHERE user_id IN (%v)", placeholders)

	args := make([]interface{}, len(userIDs))
	for i, id := range userIDs {
		args[i] = id
	}

	rows, err := u.db.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "db.Query")
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
			return nil, errors.Wrap(err, "db.Query")
		}
		users = append(users, &res)
	}

	return users, nil
}

// CreateSQLPlaceholders Generate postgres $ placeholders
func CreateSQLPlaceholders(length int) string {
	var builder strings.Builder
	for i := 0; i < length; i++ {
		if i < length-1 {
			builder.WriteString(`,`)
		}
		builder.WriteString(`$` + strconv.Itoa(i+1))
	}
	return builder.String()
}
