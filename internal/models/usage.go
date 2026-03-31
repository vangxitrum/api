package models

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Usage struct {
	Id                 uuid.UUID       `json:"id"                            gorm:"primaryKey;type:uuid"`
	UserId             uuid.UUID       `json:"user_id"                       gorm:"type:uuid"`
	User               *User           `json:"user"                          gorm:"foreignKey:UserId;references:Id"`
	Storage            int64           `json:"storage"`
	Transcode          float64         `json:"transcode"`
	Delivery           int64           `json:"delivery"`
	StorageCost        decimal.Decimal `json:"total"`
	DeliveryCost       decimal.Decimal `json:"delivery_cost"`
	TranscodeCost      decimal.Decimal `json:"transcode_cost"`
	LivestreamDuration float64         `json:"livestream_duration,omitempty"`
	TotalCost          decimal.Decimal `json:"total_cost"`
	Status             string          `json:"status"`
	CreatedAt          time.Time       `json:"created_at"                    gorm:"timestamp"`
}

type UsageRepository interface {
	Create(context.Context, *Usage) error
	CreateLog(context.Context, *UsageLog) error

	GetUsageLogs(context.Context, time.Time) ([]*UsageLog, error)
	GetUserLatestUsage(context.Context, uuid.UUID, time.Time) (*Usage, error)
	GetLastHourUsagesByStatus(context.Context, string) ([]*Usage, error)
	GetUserUsagesByStatus(context.Context, uuid.UUID, string) ([]*Usage, error)
	GetLastHourUsages(context.Context) ([]*Usage, error)
	GetUserUsageByTimeRange(context.Context, uuid.UUID, TimeRange) (decimal.Decimal, error)
	GetUserLastHourUsage(context.Context, uuid.UUID) (decimal.Decimal, error)
	GetUsageLogsByCursor(context.Context, time.Time) ([]*UsageLog, error)
	GetUserIdsByTimeRange(context.Context, TimeRange) ([]uuid.UUID, error)
	GetUserTotalCostByTimeRange(context.Context, uuid.UUID, TimeRange) (*UsageBilling, error)
	GetUserDebt(ctx context.Context, userId uuid.UUID) (decimal.Decimal, error)
	GetUserTocalCostByDate(
		ctx context.Context,
		userId uuid.UUID,
		date time.Time,
	) (decimal.Decimal, error)

	UpdateUsage(context.Context, *Usage) error

	DeleteUsageLogsByCursor(context.Context, time.Time) error
}

func NewUsage(userId uuid.UUID, storage int64, transcode float64, delivery int64) *Usage {
	return &Usage{
		Id:          uuid.New(),
		UserId:      userId,
		Storage:     storage,
		Transcode:   transcode,
		Delivery:    delivery,
		Status:      PendingStatus,
		StorageCost: decimal.Zero,
		CreatedAt:   time.Now().UTC().Truncate(time.Hour),
	}
}

type UsageLog struct {
	Id                 uuid.UUID       `json:"id"                            gorm:"primaryKey"`
	UserId             uuid.UUID       `json:"user_id"`
	Storage            int64           `json:"storage"`
	Transcode          float64         `json:"transcode"`
	Delivery           int64           `json:"delivery"`
	TranscodeCost      decimal.Decimal `json:"transcode_cost"`
	LivestreamDuration float64         `json:"livestream_duration,omitempty"`
	IsUserCost         bool            `json:"is_user_cost"`
	CreatedAt          time.Time       `json:"created_at"`
}

type UsageLogBuilder struct {
	log UsageLog
}

func (b *UsageLogBuilder) SetUserId(userId uuid.UUID) *UsageLogBuilder {
	b.log.UserId = userId
	return b
}

func (b *UsageLogBuilder) SetStorage(storage int64) *UsageLogBuilder {
	b.log.Storage = storage
	return b
}

func (b *UsageLogBuilder) SetTranscode(transcode float64) *UsageLogBuilder {
	b.log.Transcode = transcode
	return b
}

func (b *UsageLogBuilder) SetTranscodeCost(transcodeCost decimal.Decimal) *UsageLogBuilder {
	b.log.TranscodeCost = transcodeCost
	return b
}

func (b *UsageLogBuilder) SetDelivery(delivery int64) *UsageLogBuilder {
	b.log.Delivery = delivery
	return b
}

func (b *UsageLogBuilder) SetLivestreamDuration(duration float64) *UsageLogBuilder {
	b.log.LivestreamDuration = duration
	return b
}

func (b *UsageLogBuilder) SetIsUserCost(isUserCost bool) *UsageLogBuilder {
	b.log.IsUserCost = isUserCost
	return b
}

func (b *UsageLogBuilder) Build() *UsageLog {
	b.log.Id = uuid.New()
	b.log.CreatedAt = time.Now().UTC().Truncate(time.Second * 15)
	return &b.log
}

func NewUsageLog(
	userId uuid.UUID,
	storage int64,
	transcode float64,
	transcodeCost decimal.Decimal,
	delivery int64,
	liveStreamDuration float64,
	isUserCost bool,
) *UsageLog {
	return &UsageLog{
		Id:            uuid.New(),
		UserId:        userId,
		Storage:       storage,
		Transcode:     transcode,
		TranscodeCost: transcodeCost,
		Delivery:      delivery,
		CreatedAt:     time.Now().UTC().Truncate(time.Second * 15),
		IsUserCost:    isUserCost,
	}
}

type UsageData struct {
	UserId    uuid.UUID `json:"user_id"`
	Storage   int64     `json:"storage"`
	Transcode float64   `json:"transcode"`
	Delivery  float64   `json:"delivery"`
}

type UsageBilling struct {
	DeliveryCost  decimal.Decimal `json:"delivery_cost"`
	TranscodeCost decimal.Decimal `json:"transcode_cost"`
	StorageCost   decimal.Decimal `json:"storage_cost"`
	TotalCost     decimal.Decimal `json:"total_cost"`
}

type DataUsage struct {
	UserId    uuid.UUID `json:"user_id"`
	Delivery  int64     `json:"delivery"`
	CreatedAt time.Time `json:"created_at"`
}
