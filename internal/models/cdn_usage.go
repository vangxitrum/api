package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CdnUsageStatistic struct {
	Id                      uuid.UUID       `json:"id"                                   gorm:"type:uuid;primaryKey"`
	RemainCredit            decimal.Decimal `json:"remain_credit"`
	TotalDelivery           int64           `json:"total_delivery"`
	TotalStorage            int64           `json:"total_storage"`
	Transcode               float64         `json:"transcode"`
	TotalTranscode          decimal.Decimal `json:"total_transcode"`
	CdnDeliveryBytes        int64           `json:"cdn_delivery_bytes"`
	CdnDeliveryCredit       decimal.Decimal `json:"cdn_delivery_credit"`
	CdnStorageBytes         int64           `json:"cdn_storage_bytes"`
	CdnStorageCredit        decimal.Decimal `json:"cdn_storage_credit"`
	CdnTranscodeCredit      decimal.Decimal `json:"cdn_transcode_credit"`
	TotalLiveStreamDuration float64         `json:"total_live_stream_duration,omitempty"`
	CreatedAt               time.Time       `json:"created_at"`
}
type CdnUsageStatisticRepository interface {
	CreateCdnUsageStatistic(context.Context, *CdnUsageStatistic) (*CdnUsageStatistic, error)
	GetCdnUsageStatistic(context.Context, time.Time) (*CdnUsageStatistic, error)

	UpdateTotalStorage(context.Context, uuid.UUID, int64) error
	UpdateDeliveryAndTranscode(context.Context, uuid.UUID, int64, float64, decimal.Decimal) error
	UpdateCdnUsage(
		context.Context,
		*CdnUsageStatistic,
	) error
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

func NewCdnUsageStatistic(
	metaData *CdnUsageStatistic,
) *CdnUsageStatistic {
	return &CdnUsageStatistic{
		Id:                 uuid.New(),
		RemainCredit:       metaData.RemainCredit,
		TotalStorage:       metaData.TotalStorage,
		TotalDelivery:      metaData.TotalDelivery,
		TotalTranscode:     metaData.TotalTranscode,
		Transcode:          metaData.Transcode,
		CdnDeliveryBytes:   metaData.CdnDeliveryBytes,
		CdnDeliveryCredit:  metaData.CdnDeliveryCredit,
		CdnStorageBytes:    metaData.CdnStorageBytes,
		CdnStorageCredit:   metaData.CdnStorageCredit,
		CdnTranscodeCredit: metaData.CdnTranscodeCredit,
		CreatedAt:          time.Now().UTC().Truncate(24 * time.Hour),
	}
}
