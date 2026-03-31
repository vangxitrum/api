package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type MediaUsageRepository struct {
	db *gorm.DB
}

func MustNewMediaUsageRepository(db *gorm.DB, init bool) models.MediaUsageRepository {
	if init {
		if err := db.AutoMigrate(&models.MediaUsage{}); err != nil {
			panic(err)
		}
	}

	return &MediaUsageRepository{
		db: db,
	}
}

func (r *MediaUsageRepository) Create(
	ctx context.Context,
	mediaUsage *models.MediaUsage,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateMediaUsage").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(mediaUsage).Error
}

func (r *MediaUsageRepository) GetUserMediaUsages(
	ctx context.Context,
	userId uuid.UUID,
) ([]*models.MediaUsage, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserMediaUsages").
			Observe(time.Since(now).Seconds())
	}()

	var mediaUsages []*models.MediaUsage
	if err := r.db.WithContext(ctx).
		Joins("JOIN media ON media_usages.media_id = media.id and media.status = ?", models.TranscodingStatus).
		Where("media.user_id = ?", userId).
		Find(&mediaUsages).Error; err != nil {
		return nil, err
	}

	return mediaUsages, nil
}

func (r *MediaUsageRepository) GetMediaUsageByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) (*models.MediaUsage, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaUsageByMediaId").
			Observe(time.Since(now).Seconds())
	}()

	var mediaUsage models.MediaUsage
	if err := r.db.WithContext(ctx).
		Where("media_id = ?", mediaId).
		First(&mediaUsage).Error; err != nil {
		return nil, err
	}

	return &mediaUsage, nil
}

func (r *MediaUsageRepository) DeleteMediaUsageByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteMediaUsageByMediaId").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.MediaUsage{}, "media_id = ?", mediaId).Error
}
