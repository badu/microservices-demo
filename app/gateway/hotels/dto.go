package hotels

import (
	"time"

	uuid "github.com/satori/go.uuid"

	hotelsService "github.com/badu/microservices-demo/app/hotels"
)

type Hotel struct {
	Image         *string    `json:"image,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at"`
	CreatedAt     *time.Time `json:"created_at"`
	Longitude     *float64   `json:"longitude,omitempty"`
	Latitude      *float64   `json:"latitude,omitempty"`
	Location      string     `json:"location" validate:"required,min=10,max=250"`
	Description   string     `json:"description,omitempty" validate:"required,min=10,max=250"`
	City          string     `json:"city,omitempty" validate:"required,min=3,max=25"`
	Country       string     `json:"country,omitempty" validate:"required,min=3,max=25"`
	Email         string     `json:"email,omitempty" validate:"required,email"`
	Name          string     `json:"name" validate:"required,min=3,max=25"`
	Photos        []string   `json:"photos,omitempty"`
	Rating        float64    `json:"rating" validate:"required,min=0,max=10"`
	CommentsCount int        `json:"comments_count,omitempty"`
	HotelID       uuid.UUID  `json:"hotel_id"`
}

type ListResult struct {
	Hotels     []*Hotel `json:"hotels"`
	TotalCount int64    `json:"totalCount"`
	TotalPages int64    `json:"totalPages"`
	Page       int64    `json:"page"`
	Size       int64    `json:"size"`
	HasMore    bool     `json:"hasMore"`
}

func fromProto(v *hotelsService.Hotel) (*Hotel, error) {
	hotelUUID, err := uuid.FromString(v.GetHotelID())
	if err != nil {
		return nil, err
	}
	if err := v.CreatedAt.CheckValid(); err != nil {
		return nil, err
	}
	if err := v.UpdatedAt.CheckValid(); err != nil {
		return nil, err
	}
	createdAt := v.CreatedAt.AsTime()
	updatedAt := v.UpdatedAt.AsTime()
	return &Hotel{
		HotelID:       hotelUUID,
		Name:          v.GetName(),
		Email:         v.GetEmail(),
		Country:       v.GetCountry(),
		City:          v.GetCity(),
		Description:   v.GetDescription(),
		Location:      v.GetLocation(),
		Rating:        v.GetRating(),
		Image:         &v.Image,
		Photos:        v.GetPhotos(),
		CommentsCount: int(v.GetCommentsCount()),
		Latitude:      &v.Latitude,
		Longitude:     &v.Longitude,
		CreatedAt:     &createdAt,
		UpdatedAt:     &updatedAt,
	}, nil
}
