package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type PartRepository interface {
	Create(context.Context, *Part) error

	DeletePartsByMediaId(context.Context, uuid.UUID) error
}

type Part struct {
	MediaId   uuid.UUID `json:"media_id"   gorm:"primaryKey;type:uuid"`
	Hash      string    `json:"hash"       gorm:"primaryKey"`
	Index     int       `json:"index"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

func NewPart(
	mediaId, userId uuid.UUID,
	hash string,
	index int,
	size int64,
) *Part {
	return &Part{
		MediaId:   mediaId,
		Hash:      hash,
		Index:     index,
		Size:      size,
		CreatedAt: time.Now(),
	}
}
