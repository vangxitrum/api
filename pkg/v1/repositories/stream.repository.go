package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type StreamRepository struct {
	db *gorm.DB
}

func MustNewStreamRepository(db *gorm.DB, init bool) models.StreamRepository {
	if init {
		if err := db.AutoMigrate(&models.MediaStream{}); err != nil {
			panic(err)
		}
	}

	return &StreamRepository{
		db: db,
	}
}

func (r *StreamRepository) Create(
	ctx context.Context,
	stream *models.MediaStream,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateStream").
			Observe(time.Since(t).Seconds())
	}()
	return r.db.WithContext(ctx).
		Create(stream).Error
}

func (r *StreamRepository) GetStreamsByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) ([]*models.MediaStream, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetStreamsByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	var streams []*models.MediaStream
	if err := r.db.WithContext(ctx).
		Model(&models.MediaStream{}).
		Where("media_id = ?", mediaId).
		Find(&streams).Error; err != nil {
		return nil, err
	}

	return streams, nil
}

func (r *StreamRepository) DeleteStreamsByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteStreamsByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.MediaStream{}, "media_id = ?", mediaId).Error
}

func (r *StreamRepository) CountStreamByCodecTypeAndMediaId(
	ctx context.Context,
	codedecType string,
	mediaId uuid.UUID,
) (int, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CountStreamByCodecTypeAndMediaId").
			Observe(time.Since(now).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Table("streams").
		Where("codec_type = ? and media_id = ?", codedecType, mediaId).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return int(count), nil
}

func (r *StreamRepository) GetStreamsByCodecTypeAndMediaId(
	ctx context.Context,
	codedecType string,
	mediaId uuid.UUID,
) ([]*models.MediaStream, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetStreamByCodecTypeAndMediaId").
			Observe(time.Since(now).Seconds())
	}()

	var streams []*models.MediaStream

	if err := r.db.WithContext(ctx).
		Table("media_streams").
		Where("media_id = ? AND codec_type = ?", mediaId, codedecType).
		Find(&streams).Error; err != nil {
		return nil, err
	}
	return streams, nil
}
