package comments

import (
	"time"

	"github.com/badu/microservices-demo/app/gateway/users"
	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/comments"
)

type Comment struct {
	CommentID uuid.UUID  `json:"comment_id"`
	HotelID   uuid.UUID  `json:"hotel_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Message   string     `json:"message" validate:"required,min=5,max=500"`
	Photos    []string   `json:"photos,omitempty"`
	Rating    float64    `json:"rating" validate:"required,min=0,max=10"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type List struct {
	TotalCount int64          `json:"totalCount"`
	TotalPages int64          `json:"totalPages"`
	Page       int64          `json:"page"`
	Size       int64          `json:"size"`
	HasMore    bool           `json:"hasMore"`
	Comments   []*CommentFull `json:"comments"`
}

func FromProto(comment *comments.Comment) (*Comment, error) {
	commUUID, err := uuid.FromString(comment.CommentID)
	if err != nil {
		return nil, err
	}
	userUUID, err := uuid.FromString(comment.UserID)
	if err != nil {
		return nil, err
	}
	hotelUUID, err := uuid.FromString(comment.HotelID)
	if err != nil {
		return nil, err
	}
	if err := comment.CreatedAt.CheckValid(); err != nil {
		return nil, err
	}
	if err := comment.UpdatedAt.CheckValid(); err != nil {
		return nil, err
	}
	createdAt := comment.CreatedAt.AsTime()
	updatedAt := comment.UpdatedAt.AsTime()

	return &Comment{
		CommentID: commUUID,
		HotelID:   hotelUUID,
		UserID:    userUUID,
		Message:   comment.Message,
		Photos:    comment.Photos,
		Rating:    comment.Rating,
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
	}, nil
}

type CommentFull struct {
	CommentID uuid.UUID          `json:"comment_id"`
	HotelID   uuid.UUID          `json:"hotel_id"`
	User      *users.CommentUser `json:"user"`
	Message   string             `json:"message"`
	Photos    []string           `json:"photos"`
	Rating    float64            `json:"rating"`
	CreatedAt *time.Time         `json:"createdAt"`
	UpdatedAt *time.Time         `json:"updatedAt"`
}

func FullFromProto(comm *comments.CommentFull) (*CommentFull, error) {
	commUUID, err := uuid.FromString(comm.GetCommentID())
	if err != nil {
		return nil, err
	}

	hotelUUID, err := uuid.FromString(comm.GetHotelID())
	if err != nil {
		return nil, err
	}
	if err := comm.CreatedAt.CheckValid(); err != nil {
		return nil, err
	}
	if err := comm.UpdatedAt.CheckValid(); err != nil {
		return nil, err
	}
	createdAt := comm.CreatedAt.AsTime()
	updatedAt := comm.UpdatedAt.AsTime()

	user := &users.CommentUser{}
	user.FromProto(comm.GetUser())

	return &CommentFull{
		CommentID: commUUID,
		HotelID:   hotelUUID,
		User:      user,
		Message:   comm.GetMessage(),
		Photos:    comm.GetPhotos(),
		Rating:    comm.GetRating(),
		CreatedAt: &createdAt,
		UpdatedAt: &updatedAt,
	}, nil
}
