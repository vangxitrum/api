package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type WatermarkRepository interface {
	CreateUserWatermark(
		context.Context, *Watermark,
	) (*Watermark, error)
	CreateFile(
		context.Context, *WaterMarkFile,
	) error

	CheckIfWatermarkExistInAnyMedia(
		context.Context, uuid.UUID,
	) (bool, error)

	ListAllWatermarks(
		context.Context, GetWatermarkList,
	) ([]*Watermark, int64, error)

	DeleteWatermarkById(
		context.Context, uuid.UUID, uuid.UUID,
	) error

	DeleteMediaWatermarkById(
		context.Context, uuid.UUID,
	) error

	CreateMediaWatermark(
		context.Context, *MediaWatermark,
	) (*MediaWatermark, error)

	GetMediaWatermark(
		context.Context, uuid.UUID, uuid.UUID,
	) (*MediaWatermark, error)
	GetUserWatermarkById(
		context.Context, uuid.UUID, uuid.UUID,
	) (*Watermark, error)
}

type Watermark struct {
	Id            uuid.UUID      `json:"id"                    gorm:"type:uuid;primaryKey"`
	UserId        uuid.UUID      `json:"user_id"               gorm:"type:uuid"`
	WatermarkName string         `json:"watermark_name"`
	Width         int64          `json:"width"`
	Height        int64          `json:"height"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	File          *WaterMarkFile `json:"media_files,omitempty" gorm:"foreignKey:WaterMarkId;references:Id"`
}

func NewWatermark(
	userId uuid.UUID,
	width, height int64,
	watermarkName string,
) *Watermark {
	return &Watermark{
		Id:            uuid.New(),
		UserId:        userId,
		WatermarkName: watermarkName,
		Width:         width,
		Height:        height,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
}

type GetWatermarkList struct {
	UserId uuid.UUID
	SortBy string
	Order  string
	Offset uint64
	Limit  uint64
}

type WaterMarkFile struct {
	WaterMarkId uuid.UUID `json:"water_mark_id" gorm:"primaryKey;type:uuid"`
	FileId      string    `json:"file_id"       gorm:"primaryKey"`
	File        *CdnFile  `json:"file"          gorm:"foreignKey:FileId;references:Id"`
}
