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

type MediaChapterRepository struct {
	db *gorm.DB
}

func MustNewMediaChapterRepository(db *gorm.DB, init bool) models.MediaChapterRepository {
	if init {
		if err := db.AutoMigrate(
			&models.MediaChapter{},
			&models.MediaChapterFile{},
		); err != nil {
			panic("failed to auto migrate media subtitle model")
		}
	}

	return &MediaChapterRepository{
		db: db,
	}
}

func (r *MediaChapterRepository) Create(
	ctx context.Context,
	subtitle *models.MediaChapter,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateChapter").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).Create(subtitle).Error
}

func (r *MediaChapterRepository) CreateFile(
	ctx context.Context,
	file *models.MediaChapterFile,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateFile").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).Create(file).Error
}

func (r *MediaChapterRepository) GetMediaChaptersByMediaIdWithLimitAndOffset(
	ctx context.Context,
	mediaId uuid.UUID,
	offset, limit int,
) ([]*models.MediaChapter, int64, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaChaptersByMediaIdWithLimitAndOffset").
			Observe(time.Since(now).Seconds())
	}()

	var subtitles []*models.MediaChapter
	var total int64
	if err := r.db.WithContext(ctx).
		Model(&models.MediaChapter{}).
		Where("media_id = ?", mediaId).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.MediaChapter{}, 0, nil
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

func (r *MediaChapterRepository) GetMediaChaptersByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) ([]*models.MediaChapter, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaChaptersByMediaId").
			Observe(time.Since(now).Seconds())
	}()

	var subtitles []*models.MediaChapter
	if err := r.db.WithContext(ctx).
		Preload("File").
		Preload("File.File").
		Where("media_id = ?", mediaId).
		Find(&subtitles).Error; err != nil {
		return nil, err
	}

	return subtitles, nil
}

func (r *MediaChapterRepository) GetMediaChapterById(
	ctx context.Context,
	id uuid.UUID,
) (*models.MediaChapter, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetChapter").
			Observe(time.Since(now).Seconds())
	}()

	var subtitle models.MediaChapter
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&subtitle).Error; err != nil {
		return nil, err
	}

	return &subtitle, nil
}

func (r *MediaChapterRepository) GetMediaChapterByMediaIdAndLanguage(
	ctx context.Context,
	mediaId uuid.UUID,
	language string,
) (*models.MediaChapter, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetChapter").
			Observe(time.Since(now).Seconds())
	}()

	var subtitle models.MediaChapter
	if err := r.db.WithContext(ctx).
		Preload("File").
		Preload("File.File").
		Where("media_id = ? AND language = ?", mediaId, language).
		First(&subtitle).Error; err != nil {
		return nil, err
	}

	return &subtitle, nil
}

func (r *MediaChapterRepository) DeleteMediaChaptersByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteChapter").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.MediaChapter{}, "media_id = ?", mediaId).Error
}

func (r *MediaChapterRepository) DeleteMediaChapterById(
	ctx context.Context,
	id uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteChapter").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Select(clause.Associations).
		Delete(&models.MediaChapter{Id: id}).Error
}
