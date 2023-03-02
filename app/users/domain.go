package users

import (
	"fmt"
	"strings"
	"time"

	uuid "github.com/satori/go.uuid"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/badu/microservices-demo/pkg/postgres"
)

type UserDO struct {
	Role      *Role               `json:"role"`
	CreatedAt *time.Time          `json:"created_at"`
	UpdatedAt *time.Time          `json:"updated_at"`
	FirstName string              `json:"first_name" validate:"required,min=3,max=25"`
	LastName  string              `json:"last_name" validate:"required,min=3,max=25"`
	Email     string              `json:"email" validate:"required,email"`
	Password  string              `json:"password" validate:"required,min=6,max=250"`
	Avatar    postgres.NullString `json:"avatar" validate:"max=250" swaggertype:"string"`
	UserID    uuid.UUID           `json:"user_id"`
}

type UserResponse struct {
	Role      *Role               `json:"role"`
	CreatedAt *time.Time          `json:"created_at"`
	UpdatedAt *time.Time          `json:"updated_at"`
	FirstName string              `json:"first_name" validate:"required,min=3,max=25"`
	LastName  string              `json:"last_name" validate:"required,min=3,max=25"`
	Email     string              `json:"email" validate:"required,email"`
	Avatar    postgres.NullString `json:"avatar" validate:"max=250" swaggertype:"string"`
	UserID    uuid.UUID           `json:"user_id"`
}

type UserUpdate struct {
	Role      *Role     `json:"role"`
	FirstName string    `json:"first_name" validate:"omitempty,min=3,max=25" swaggertype:"string"`
	LastName  string    `json:"last_name" validate:"omitempty,min=3,max=25" swaggertype:"string"`
	Email     string    `json:"email" validate:"omitempty,email" swaggertype:"string"`
	Avatar    string    `json:"avatar" validate:"max=250" swaggertype:"string"`
	UserID    uuid.UUID `json:"user_id"`
}

type Login struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6,max=250"`
}

type Role string

const (
	RoleGuest  Role = "guest"
	RoleMember Role = "member"
	RoleAdmin  Role = "admin"
)

func (e *Role) ToString() string {
	return string(*e)
}

func (e *Role) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = Role(s)
	case string:
		*e = Role(s)
	default:
		return fmt.Errorf("unsupported scan type for Role: %T", src)
	}
	return nil
}

// Hash user password with bcrypt
func (u *UserDO) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// Compare user password and payload
func (u *UserDO) ComparePasswords(password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err != nil {
		return err
	}
	return nil
}

// Sanitize user password
func (u *UserDO) SanitizePassword() {
	u.Password = ""
}

// Prepare user for register
func (u *UserDO) PrepareCreate() error {
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	u.Password = strings.TrimSpace(u.Password)

	if err := u.HashPassword(); err != nil {
		return err
	}
	return nil
}

func (r *UserResponse) ToProto() *User {
	return &User{
		UserID:    r.UserID.String(),
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Email:     r.Email,
		Avatar:    r.Avatar.String,
		Role:      r.Role.ToString(),
		CreatedAt: timestamppb.New(*r.CreatedAt),
		UpdatedAt: timestamppb.New(*r.UpdatedAt),
	}
}
