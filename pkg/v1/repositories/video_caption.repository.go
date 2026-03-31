package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type MediaCaptionRepository struct {
	db *gorm.DB
}

func MustNewMediaCaptionRepository(db *gorm.DB, init bool) models.MediaCaptionRepository {
	if init {
		if err := db.AutoMigrate(
			&models.MediaCaption{},
			&models.MediaCaptionFile{},
		); err != nil {
			panic("failed to auto migrate media subtitle model")
		}
	}

	return &MediaCaptionRepository{
		db: db,
	}
}

func (r *MediaCaptionRepository) Create(
	ctx context.Context,
	subtitle *models.MediaCaption,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateCaption").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).Create(subtitle).Error
}

func (r *MediaCaptionRepository) CreateFile(
	ctx context.Context,
	file *models.MediaCaptionFile,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateFile").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).Create(file).Error
}

func (r *MediaCaptionRepository) GetMediaCaptionsByMediaIdWithOffsetAndLimit(
	ctx context.Context,
	mediaId uuid.UUID,
	offset, limit int,
) ([]*models.MediaCaption, int64, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCaption").
			Observe(time.Since(now).Seconds())
	}()

	var subtitles []*models.MediaCaption
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.MediaCaption{}).
		Where("media_id = ? and status = ?", mediaId, models.DoneStatus).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.MediaCaption{}, 0, nil
	}

	if err := r.db.WithContext(ctx).
		Where("media_id = ?", mediaId).
		Offset(offset).
		Limit(limit).
		Find(&subtitles).Error; err != nil {
		return nil, 0, err
	}

	return subtitles, total, nil
}

func (r *MediaCaptionRepository) GetMediaCaptionsByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) ([]*models.MediaCaption, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaCaptionsByMediaId").
			Observe(time.Since(now).Seconds())
	}()

	var subtitles []*models.MediaCaption
	if err := r.db.WithContext(ctx).
		Preload("File").
		Preload("File.File").
		Where("media_id = ?", mediaId).
		Find(&subtitles).Error; err != nil {
		return nil, err
	}

	return subtitles, nil
}

func (r *MediaCaptionRepository) GetMediaCaptionById(
	ctx context.Context,
	id uuid.UUID,
) (*models.MediaCaption, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCaption").
			Observe(time.Since(now).Seconds())
	}()

	var subtitle models.MediaCaption
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&subtitle).Error; err != nil {
		return nil, err
	}

	return &subtitle, nil
}

func (r *MediaCaptionRepository) GetMediaCaptionByMediaIdAndLanguage(
	ctx context.Context,
	mediaId uuid.UUID,
	language string,
) (*models.MediaCaption, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCaption").
			Observe(time.Since(now).Seconds())
	}()

	var subtitle models.MediaCaption
	if err := r.db.WithContext(ctx).
		Preload("File").
		Preload("File.File").
		Where("media_id = ? AND language = ?", mediaId, language).
		First(&subtitle).Error; err != nil {
		return nil, err
	}

	return &subtitle, nil
}

func (r *MediaCaptionRepository) GetMediaCaptionsByStatus(
	ctx context.Context,
	status string,
) ([]*models.MediaCaption, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaCaptionsByStatus").
			Observe(time.Since(now).Seconds())
	}()

	var captions []*models.MediaCaption
	if err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Preload("Media").
		Find(&captions).Error; err != nil {
		return nil, err
	}

	return captions, nil
}

func (r *MediaCaptionRepository) GetMediaCaptionsByStatusAndMediaId(
	ctx context.Context,
	status string,
	mediaId uuid.UUID,
) ([]*models.MediaCaption, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCaptionGetMediaCaptionsByStatusAndMediaId").
			Observe(time.Since(now).Seconds())
	}()

	var captions []*models.MediaCaption
	if err := r.db.WithContext(ctx).
		Where("status = ? and media_id = ?", status, mediaId).
		Find(&captions).Error; err != nil {
		return nil, err
	}

	return captions, nil
}

func (r *MediaCaptionRepository) SetMediaDefaultCaption(
	ctx context.Context,
	mediaId uuid.UUID,
	language string,
	isDefault bool,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("SetDefaultCaption").
			Observe(time.Since(now).Seconds())
	}()

	if err := r.db.WithContext(ctx).
		Model(&models.MediaCaption{}).
		Where("media_id = ?", mediaId).
		Update("is_default", false).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&models.MediaCaption{}).
		Where("media_id = ? AND language = ?", mediaId, language).
		Update("is_default", isDefault).Error
}

func (r *MediaCaptionRepository) UpdateMediaCaptionStatus(
	ctx context.Context,
	id uuid.UUID,
	status string,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateCaptionStatus").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(&models.MediaCaption{}).
		Where("id = ?", id).
		Update("status", status).Error
}

func (r *MediaCaptionRepository) DeleteMediaCaptionsByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteCaption").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.MediaCaption{}, "media_id = ?", mediaId).Error
}

func (r *MediaCaptionRepository) DeleteMediaCaptionById(
	ctx context.Context,
	id uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteCaption").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Select(clause.Associations).
		Delete(&models.MediaCaption{Id: id}).Error
}
