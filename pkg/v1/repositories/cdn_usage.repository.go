package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type CdnUsageRepository struct {
	db *gorm.DB
}

func MustNewCdnUsageRepository(db *gorm.DB, init bool) models.CdnUsageStatisticRepository {
	if init {
		if err := db.AutoMigrate(&models.CdnUsageStatistic{}); err != nil {
			panic(err)
		}
	}

	return &CdnUsageRepository{
		db: db,
	}
}

func (r *CdnUsageRepository) CreateCdnUsageStatistic(
	ctx context.Context, statistic *models.CdnUsageStatistic,
) (*models.CdnUsageStatistic, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateCdnUsageStatistic").
			Observe(time.Since(now).Seconds())
	}()

	if err := r.db.WithContext(ctx).Create(statistic).Error; err != nil {
		return nil, err
	}

	return statistic, nil
}

func (r *CdnUsageRepository) GetCdnUsageStatistic(
	ctx context.Context, cursor time.Time,
) (*models.CdnUsageStatistic, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnUsageStatistic").
			Observe(time.Since(now).Seconds())
	}()
	var statistic models.CdnUsageStatistic

	if err := r.db.WithContext(ctx).
		Where("created_at = ?", cursor).
		First(&statistic).Error; err != nil {
		return nil, err
	}
	return &statistic, nil
}

func (r *CdnUsageRepository) UpdateTotalStorage(
	ctx context.Context,
	id uuid.UUID,
	totalStorage int64,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateTotalStorage").
			Observe(time.Since(now).Seconds())
	}()

	if err := r.db.WithContext(ctx).Model(&models.CdnUsageStatistic{}).
		Where("id = ?", id).
		Update("total_storage", totalStorage).Error; err != nil {
		return err
	}

	return nil
}

func (r *CdnUsageRepository) UpdateDeliveryAndTranscode(
	ctx context.Context,
	id uuid.UUID,
	totalDelivery int64,
	transcode float64,
	totalTranscode decimal.Decimal,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateDeliveryAndTranscode").
			Observe(time.Since(now).Seconds())
	}()

	if err := r.db.WithContext(ctx).Model(&models.CdnUsageStatistic{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"transcode":       transcode,
			"total_delivery":  totalDelivery,
			"total_transcode": totalTranscode,
		}).Error; err != nil {
		return err
	}

	return nil
}

func (r *CdnUsageRepository) UpdateCdnUsage(
	ctx context.Context,
	statistic *models.CdnUsageStatistic,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateCdnUsage").
			Observe(time.Since(now).Seconds())
	}()

	if err := r.db.WithContext(ctx).Save(statistic).Error; err != nil {
		return err
	}

	return nil
}
