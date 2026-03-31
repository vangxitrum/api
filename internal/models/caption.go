package models

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MediaCaptionRepository interface {
	Create(context.Context, *MediaCaption) error
	CreateFile(context.Context, *MediaCaptionFile) error

	GetMediaCaptionsByMediaIdWithOffsetAndLimit(
		context.Context,
		uuid.UUID,
		int,
		int,
	) ([]*MediaCaption, int64, error)
	GetMediaCaptionsByMediaId(context.Context, uuid.UUID) ([]*MediaCaption, error)
	GetMediaCaptionById(context.Context, uuid.UUID) (*MediaCaption, error)
	GetMediaCaptionByMediaIdAndLanguage(context.Context, uuid.UUID, string) (*MediaCaption, error)
	GetMediaCaptionsByStatus(context.Context, string) ([]*MediaCaption, error)
	GetMediaCaptionsByStatusAndMediaId(context.Context, string, uuid.UUID) ([]*MediaCaption, error)

	SetMediaDefaultCaption(context.Context, uuid.UUID, string, bool) error

	UpdateMediaCaptionStatus(context.Context, uuid.UUID, string) error

	DeleteMediaCaptionsByMediaId(context.Context, uuid.UUID) error
	DeleteMediaCaptionById(context.Context, uuid.UUID) error
}

type MediaCaption struct {
	Id          uuid.UUID         `json:"-"           gorm:"primaryKey;type:uuid"`
	MediaId     uuid.UUID         `json:"-"`
	Media       *Media            `json:"-"           gorm:"foreignKey:MediaId;references:Id"`
	Url         string            `json:"url"         gorm:"-"`
	Language    string            `json:"language"`
	Status      string            `json:"status"`
	Description string            `json:"description"`
	IsDefault   bool              `json:"is_default"`
	TaskId      string            `json:"-"`
	CreatedAt   time.Time         `json:"-"`
	File        *MediaCaptionFile `json:"-"           gorm:"foreignKey:CaptionId;references:Id"`
} //	@name	MediaCaption

func (c *MediaCaption) GetUrl(fileType string, token string) string {
	url := fmt.Sprintf(
		"%s/api/media/%s/captions/%s?fileType=%s",
		BeUrl,
		c.MediaId,
		c.Language,
		fileType,
	)
	if token != "" {
		url += fmt.Sprintf("&token=%s", token)
	}

	return url
}

func NewMediaCaption(
	mediaId uuid.UUID,
	language string,
	isDefault bool,
	description string,
) *MediaCaption {
	return &MediaCaption{
		Id:          uuid.New(),
		MediaId:     mediaId,
		Language:    language,
		Status:      NewStatus,
		Description: description,
		IsDefault:   isDefault,
		CreatedAt:   time.Now().UTC(),
	}
}

type MediaCaptionFile struct {
	CaptionId uuid.UUID `json:"caption_id" gorm:"primaryKey;type:uuid"`
	FileId    string    `json:"file_id"    gorm:"primaryKey"`
	File      *CdnFile  `json:"-"          gorm:"foreignKey:FileId;references:Id"`
}
