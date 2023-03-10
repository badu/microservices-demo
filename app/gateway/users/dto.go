package users

import (
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/sessions"

	"github.com/badu/microservices-demo/app/users"
)

type UserResponse struct {
	Role      *string    `json:"role"`
	Avatar    *string    `json:"avatar" validate:"max=250" swaggertype:"string"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	FirstName string     `json:"first_name" validate:"required,min=3,max=25"`
	LastName  string     `json:"last_name" validate:"required,min=3,max=25"`
	Email     string     `json:"email" validate:"required,email"`
	UserID    uuid.UUID  `json:"user_id"`
}

func fromProto(user *users.User) (*UserResponse, error) {
	userUUID, err := uuid.FromString(user.GetUserID())
	if err != nil {
		return nil, err
	}

	if err := user.CreatedAt.CheckValid(); err != nil {
		return nil, err
	}
	if err := user.UpdatedAt.CheckValid(); err != nil {
		return nil, err
	}
	createdAt := user.CreatedAt.AsTime()
	updatedAt := user.UpdatedAt.AsTime()

	return &UserResponse{
		UserID:    userUUID,
		FirstName: user.GetFirstName(),
		LastName:  user.GetLastName(),
		Email:     user.GetEmail(),
		Role:      &user.Role,
		Avatar:    &user.Avatar,
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
	}, nil
}

type CommentUser struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Role      string `json:"role"`
}

func (u *CommentUser) FromProto(user *users.User) {
	u.UserID = user.UserID
	u.FirstName = user.FirstName
	u.LastName = user.LastName
	u.Email = user.Email
	u.Avatar = user.Avatar
	u.Role = user.Role
}

type Session struct {
	SessionID string    `json:"session_id"`
	UserID    uuid.UUID `json:"user_id"`
}

func (s *Session) FromProto(session *sessions.Session) (*Session, error) {
	userUUID, err := uuid.FromString(session.GetUserID())
	if err != nil {
		return nil, err
	}
	s.UserID = userUUID
	s.SessionID = session.GetSessionID()
	return s, nil
}
