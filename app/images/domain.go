package images

import (
	"time"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ImageDO model
type ImageDO struct {
	ImageID    uuid.UUID `json:"image_id"`
	ImageURL   string    `json:"image_url"`
	IsUploaded bool      `json:"is_uploaded"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Event message for upload image
type UploadImageMsg struct {
	ImageID    uuid.UUID `json:"image_id"`
	UserID     uuid.UUID `json:"user_id"`
	ImageURL   string    `json:"image_url"`
	IsUploaded bool      `json:"is_uploaded"`
}

// Event message for create image
type CreateImageMsg struct {
	ImageURL   string `json:"image_url"`
	IsUploaded bool   `json:"is_uploaded"`
}

func (i *ImageDO) ToProto() *Image {
	return &Image{
		ImageID:    i.ImageID.String(),
		ImageURL:   i.ImageURL,
		IsUploaded: i.IsUploaded,
		CreatedAt:  timestamppb.New(i.CreatedAt),
	}
}

// UpdateHotelImageMsg
type UpdateHotelImageMsg struct {
	HotelID uuid.UUID `json:"hotel_id"`
	Image   string    `json:"image,omitempty"`
}

// Event message for uploaded images
type UploadedImageMsg struct {
	ImageID    uuid.UUID `json:"image_id"`
	UserID     uuid.UUID `json:"user_id"`
	ImageURL   string    `json:"image_url"`
	IsUploaded bool      `json:"is_uploaded"`
	CreatedAt  time.Time `json:"created_at"`
}

// Event message for upload user avatar
type UpdateAvatarMsg struct {
	UserID      uuid.UUID `json:"user_id"`
	ContentType string    `json:"content_type"`
	Body        []byte
}
