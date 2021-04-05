package hotels

import (
	"time"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HotelDO model
type HotelDO struct {
	HotelID       uuid.UUID  `json:"hotel_id"`
	Name          string     `json:"name" validate:"required,min=3,max=25"`
	Email         string     `json:"email,omitempty" validate:"required,email"`
	Country       string     `json:"country,omitempty" validate:"required,min=3,max=25"`
	City          string     `json:"city,omitempty" validate:"required,min=3,max=25"`
	Description   string     `json:"description,omitempty" validate:"required,min=10,max=250"`
	Location      string     `json:"location" validate:"required,min=10,max=250"`
	Rating        float64    `json:"rating" validate:"required,min=0,max=10"`
	Image         *string    `json:"image,omitempty"`
	Photos        []string   `json:"photos,omitempty"`
	CommentsCount int        `json:"comments_count,omitempty"`
	Latitude      *float64   `json:"latitude,omitempty"`
	Longitude     *float64   `json:"longitude,omitempty"`
	CreatedAt     *time.Time `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

func (h *HotelDO) GetImage() string {
	var image string
	if h.Image != nil {
		image = *h.Image
	}
	return image
}

func (h *HotelDO) GetLatitude() float64 {
	var lat float64
	if h.Latitude != nil {
		lat = *h.Latitude
	}
	return lat
}

func (h *HotelDO) GetLongitude() float64 {
	var lon float64
	if h.Longitude != nil {
		lon = *h.Longitude
	}
	return lon
}

func (h *HotelDO) ToProto() *Hotel {
	return &Hotel{
		HotelID:       h.HotelID.String(),
		Name:          h.Name,
		Email:         h.Email,
		Country:       h.Country,
		City:          h.City,
		Description:   h.Description,
		Image:         h.GetImage(),
		Photos:        h.Photos,
		CommentsCount: int64(h.CommentsCount),
		Latitude:      h.GetLatitude(),
		Longitude:     h.GetLongitude(),
		Location:      h.Location,
		CreatedAt:     timestamppb.New(*h.CreatedAt),
		UpdatedAt:     timestamppb.New(*h.UpdatedAt),
	}
}

// All Hotels response with pagination
type List struct {
	TotalCount int        `json:"totalCount"`
	TotalPages int        `json:"totalPages"`
	Page       int        `json:"page"`
	Size       int        `json:"size"`
	HasMore    bool       `json:"hasMore"`
	Hotels     []*HotelDO `json:"comments"`
}

// ToProto
func (h *List) ToProto() []*Hotel {
	hotelsList := make([]*Hotel, 0, len(h.Hotels))
	for _, hotel := range h.Hotels {
		hotelsList = append(hotelsList, hotel.ToProto())
	}
	return hotelsList
}

// UpdateHotelImageMsg
type UpdateHotelImageMsg struct {
	HotelID uuid.UUID `json:"hotel_id"`
	Image   string    `json:"image,omitempty"`
}

// UpdateHotelImageMsg
type UploadHotelImageMsg struct {
	HotelID     uuid.UUID `json:"hotel_id"`
	Data        []byte    `json:"date"`
	ContentType string    `json:"content_type"`
}
