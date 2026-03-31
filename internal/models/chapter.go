package models

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type MediaChapterRepository interface {
	Create(context.Context, *MediaChapter) error
	CreateFile(context.Context, *MediaChapterFile) error

	GetMediaChaptersByMediaIdWithLimitAndOffset(
		context.Context,
		uuid.UUID,
		int,
		int,
	) ([]*MediaChapter, int64, error)
	GetMediaChaptersByMediaId(context.Context, uuid.UUID) ([]*MediaChapter, error)

	GetMediaChapterById(context.Context, uuid.UUID) (*MediaChapter, error)
	GetMediaChapterByMediaIdAndLanguage(context.Context, uuid.UUID, string) (*MediaChapter, error)

	DeleteMediaChaptersByMediaId(context.Context, uuid.UUID) error
	DeleteMediaChapterById(context.Context, uuid.UUID) error
}

type MediaChapter struct {
	Id        uuid.UUID         `json:"-"        gorm:"primaryKey;type:uuid"`
	MediaId   uuid.UUID         `json:"-"`
	Media     *Media            `json:"-"        gorm:"foreignKey:MediaId"`
	Url       string            `json:"url"      gorm:"-"`
	Language  string            `json:"language"`
	CreatedAt time.Time         `json:"-"`
	File      *MediaChapterFile `json:"-"        gorm:"foreignKey:ChapterId;references:Id"`
} //	@name	MediaChapter

func (v *MediaChapter) GetUrl(token string) string {
	url := fmt.Sprintf(
		"%s/api/media/%s/chapters/%s",
		BeUrl,
		v.MediaId,
		v.Language,
	)

	if token != "" {
		url += fmt.Sprintf("?token=%s", token)
	}

	return url
}

func NewMediaChapter(mediaId uuid.UUID, language string) *MediaChapter {
	return &MediaChapter{
		Id:        uuid.New(),
		MediaId:   mediaId,
		Language:  language,
		CreatedAt: time.Now().UTC(),
	}
}

type MediaChapterFile struct {
	ChapterId uuid.UUID `json:"chapter_id" gorm:"primaryKey;type:uuid"`
	FileId    string    `json:"file_id"    gorm:"primaryKey"`
	File      *CdnFile  `json:"file"       gorm:"foreignKey:FileId;references:Id"`
}
