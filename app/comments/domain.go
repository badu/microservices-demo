package comments

import (
	"time"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/badu/microservices-demo/app/users"
)

type CommentDO struct {
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
	Message   string     `json:"message" validate:"required,min=5,max=500"`
	Photos    []string   `json:"photos,omitempty"`
	Rating    float64    `json:"rating" validate:"required,min=0,max=10"`
	CommentID uuid.UUID  `json:"comment_id"`
	HotelID   uuid.UUID  `json:"hotel_id"`
	UserID    uuid.UUID  `json:"user_id"`
}

func (c *CommentDO) ToProto() *Comment {
	return &Comment{
		CommentID: c.CommentID.String(),
		HotelID:   c.HotelID.String(),
		UserID:    c.UserID.String(),
		Message:   c.Message,
		Photos:    c.Photos,
		Rating:    c.Rating,
		CreatedAt: timestamppb.New(*c.CreatedAt),
		UpdatedAt: timestamppb.New(*c.UpdatedAt),
	}
}

// All Comments response with pagination
type List struct {
	Comments   []*CommentDO `json:"comments"`
	TotalCount int          `json:"totalCount"`
	TotalPages int          `json:"totalPages"`
	Page       int          `json:"page"`
	Size       int          `json:"size"`
	HasMore    bool         `json:"hasMore"`
}

// All Comments response with pagination
type FullList struct {
	Comments   []*CommentFull `json:"comments"`
	TotalCount int            `json:"totalCount"`
	TotalPages int            `json:"totalPages"`
	Page       int            `json:"page"`
	Size       int            `json:"size"`
	HasMore    bool           `json:"hasMore"`
}

type FullCommentsList struct {
	UsersList    []users.UserResponse
	CommentsList List
}

func (h *List) ToProto() []*Comment {
	commentsList := make([]*Comment, 0, len(h.Comments))
	for _, hotel := range h.Comments {
		commentsList = append(commentsList, hotel.ToProto())
	}
	return commentsList
}

func (h *List) ToHotelByIDProto(userz []*users.User) []*CommentFull {
	userMap := make(map[string]*users.User, len(userz))
	for _, user := range userz {
		userMap[user.UserID] = user
	}

	commentsList := make([]*CommentFull, 0, len(h.Comments))
	for _, comm := range h.Comments {
		user := userMap[comm.UserID.String()]

		commentsList = append(commentsList, &CommentFull{
			CommentID: comm.CommentID.String(),
			HotelID:   comm.HotelID.String(),
			User: &users.User{
				UserID:    user.UserID,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Email:     user.Email,
				Avatar:    user.Avatar,
				Role:      user.Role,
			},
			Message:   comm.Message,
			Photos:    comm.Photos,
			Rating:    comm.Rating,
			CreatedAt: timestamppb.New(*comm.CreatedAt),
			UpdatedAt: timestamppb.New(*comm.UpdatedAt),
		})
	}
	return commentsList
}
