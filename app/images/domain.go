package images

import (
	"time"

	uuid "github.com/satori/go.uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ImageDO model
type ImageDO struct {
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ImageURL   string    `json:"image_url"`
	ImageID    uuid.UUID `json:"image_id"`
	IsUploaded bool      `json:"is_uploaded"`
}

// Event message for upload image
type UploadImageMsg struct {
	ImageURL   string    `json:"image_url"`
	ImageID    uuid.UUID `json:"image_id"`
	UserID     uuid.UUID `json:"user_id"`
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

type UpdateHotelImageMsg struct {
	Image   string    `json:"image,omitempty"`
	HotelID uuid.UUID `json:"hotel_id"`
}

// Event message for uploaded images
type UploadedImageMsg struct {
	CreatedAt  time.Time `json:"created_at"`
	ImageURL   string    `json:"image_url"`
	ImageID    uuid.UUID `json:"image_id"`
	UserID     uuid.UUID `json:"user_id"`
	IsUploaded bool      `json:"is_uploaded"`
}

// Event message for upload user avatar
type UpdateAvatarMsg struct {
	ContentType string `json:"content_type"`
	Body        []byte
	UserID      uuid.UUID `json:"user_id"`
}
