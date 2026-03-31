package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type FormatRepository struct {
	db *gorm.DB
}

func MustNewFormatRepository(db *gorm.DB, init bool) models.FormatRepository {
	if init {
		if err := db.AutoMigrate(&models.MediaFormat{}); err != nil {
			panic(err)
		}
	}

	return &FormatRepository{
		db: db,
	}
}

func (r *FormatRepository) Create(ctx context.Context, format *models.MediaFormat) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateFormat").
			Observe(time.Since(t).Seconds())
	}()
	return r.db.WithContext(ctx).
		Create(format).Error
}

func (r *FormatRepository) GetFormatByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) (*models.MediaFormat, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetFormatByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	var format models.MediaFormat
	if err := r.db.WithContext(ctx).
		Where("media_id = ?", mediaId).
		Find(&format).Error; err != nil {
		return nil, err
	}

	return &format, nil
}

func (r *FormatRepository) DeleteFormatByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteFormatByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.MediaFormat{}, "media_id = ?", mediaId).Error
}
