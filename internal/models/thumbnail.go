package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ThumbnailRepository interface {
	CreateThumbnail(context.Context, *Thumbnail) error
	CreateResolutions(context.Context, []*ThumbnailResolution) error
	CreateFile(context.Context, *ThumbnailFile) error

	GetThumbnailByBelongToId(ctx context.Context, belongToId uuid.UUID) (*Thumbnail, error)
	GetResolutionByThumbnailIdAndResolution(
		ctx context.Context,
		thumbnailId uuid.UUID,
		resolution string,
	) (*ThumbnailResolution, error)

	UpdateThumbnail(ctx context.Context, thumbnail *Thumbnail) error

	DeleteResolutionsByThumbnailId(ctx context.Context, thumbnailId uuid.UUID) error
	DeleteThumbnailByBelongToId(ctx context.Context, belongToId uuid.UUID) error
	DeleteThumbnailById(ctx context.Context, id uuid.UUID) error
}

type Thumbnail struct {
	Id          uuid.UUID              `json:"id"          gorm:"primaryKey;type:uuid"`
	Resolutions []*ThumbnailResolution `json:"resolutions" gorm:"foreignKey:ThumbnailId;references:Id"`
	File        *ThumbnailFile         `json:"file"        gorm:"foreignKey:ThumbnailId;references:Id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func NewThumbnail() *Thumbnail {
	return &Thumbnail{
		Id:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

type ThumbnailFile struct {
	ThumbnailId uuid.UUID `json:"thumbnail_id" gorm:"primaryKey;type:uuid"`
	FileId      string    `json:"file_id"      gorm:"primaryKey"`
	File        *CdnFile  `json:"file"         gorm:"foreignKey:FileId;references:Id"`
}

type ThumbnailResolution struct {
	Id          uuid.UUID  `json:"id"           gorm:"primaryKey;type:uuid"`
	ThumbnailId uuid.UUID  `json:"thumbnail_id" gorm:"type:uuid"`
	Thumbnail   *Thumbnail `json:"thumbnail"    gorm:"foreignKey:ThumbnailId;references:Id"`
	Resolution  string     `json:"resolution"`
	Size        int64      `json:"size"`
	Offset      int64      `json:"range"`
	CreatedAt   time.Time  `json:"created_at"`
}

func NewThumbnailResolution(
	thumbnailId uuid.UUID,
	resolution string,
	size int64,
	offset int64,
) *ThumbnailResolution {
	return &ThumbnailResolution{
		Id:          uuid.New(),
		ThumbnailId: thumbnailId,
		Resolution:  resolution,
		Size:        size,
		Offset:      offset,
		CreatedAt:   time.Now().UTC(),
	}
}
