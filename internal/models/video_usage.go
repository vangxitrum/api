package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type MediaUsageRepository interface {
	Create(context.Context, *MediaUsage) error

	GetMediaUsageByMediaId(context.Context, uuid.UUID) (*MediaUsage, error)
	GetUserMediaUsages(
		ctx context.Context,
		userId uuid.UUID,
	) ([]*MediaUsage, error)

	DeleteMediaUsageByMediaId(context.Context, uuid.UUID) error
}

type MediaUsage struct {
	Id        uuid.UUID       `json:"id"         gorm:"primaryKey;type:uuid"`
	MediaId   uuid.UUID       `json:"media_id"   gorm:"type:uuid;references:Id"`
	Media     *Media          `json:"media"      gorm:"foreignKey:MediaId"`
	Duration  float64         `json:"duration"`
	Cost      decimal.Decimal `json:"cost"`
	CreatedAt time.Time       `json:"created_at" gorm:"timestamp"`
}

func NewMediaUsage(mediaId uuid.UUID, duration float64, cost decimal.Decimal) *MediaUsage {
	return &MediaUsage{
		Id:        uuid.New(),
		MediaId:   mediaId,
		Cost:      cost,
		Duration:  duration,
		CreatedAt: time.Now(),
	}
}
