package comments

import (
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/badu/microservices-demo/app/gateway/users"

	"github.com/badu/microservices-demo/app/comments"
)

type Comment struct {
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	Message   string     `json:"message" validate:"required,min=5,max=500"`
	Photos    []string   `json:"photos,omitempty"`
	Rating    float64    `json:"rating" validate:"required,min=0,max=10"`
	CommentID uuid.UUID  `json:"comment_id"`
	HotelID   uuid.UUID  `json:"hotel_id"`
	UserID    uuid.UUID  `json:"user_id"`
}

type List struct {
	Comments   []*CommentFull `json:"comments"`
	TotalCount int64          `json:"totalCount"`
	TotalPages int64          `json:"totalPages"`
	Page       int64          `json:"page"`
	Size       int64          `json:"size"`
	HasMore    bool           `json:"hasMore"`
}

func fromProto(comment *comments.Comment) (*Comment, error) {
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
	User      *users.CommentUser `json:"user"`
	CreatedAt *time.Time         `json:"createdAt"`
	UpdatedAt *time.Time         `json:"updatedAt"`
	Message   string             `json:"message"`
	Photos    []string           `json:"photos"`
	Rating    float64            `json:"rating"`
	CommentID uuid.UUID          `json:"comment_id"`
	HotelID   uuid.UUID          `json:"hotel_id"`
}

func fromProtoToFull(comm *comments.CommentFull) (*CommentFull, error) {
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
