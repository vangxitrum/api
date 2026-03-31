package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type ThumbnailRepository struct {
	db *gorm.DB
}

func MustNewThumbnailRepository(db *gorm.DB, init bool) models.ThumbnailRepository {
	if init {
		if err := db.AutoMigrate(
			&models.Thumbnail{},
			&models.ThumbnailResolution{},
			&models.ThumbnailFile{},
		); err != nil {
			panic(err)
		}
	}

	return &ThumbnailRepository{
		db: db,
	}
}

func (r *ThumbnailRepository) CreateThumbnail(
	ctx context.Context,
	thumbnail *models.Thumbnail,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateThumbnail").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(thumbnail).Error
}

func (r *ThumbnailRepository) CreateFile(
	ctx context.Context,
	file *models.ThumbnailFile,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateFile").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(file).Error
}

func (r *ThumbnailRepository) CreateResolutions(
	ctx context.Context,
	resolutions []*models.ThumbnailResolution,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateResolutions").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(resolutions).Error
}

func (r *ThumbnailRepository) GetThumbnailByBelongToId(
	ctx context.Context,
	belongToId uuid.UUID,
) (*models.Thumbnail, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetThumbnailByBelongToId").
			Observe(time.Since(now).Seconds())
	}()

	var thumbnail models.Thumbnail
	if err := r.db.WithContext(ctx).
		Where("belongs_to_id = ?", belongToId).
		First(&thumbnail).Error; err != nil {
		return nil, err
	}

	return &thumbnail, nil
}

func (r *ThumbnailRepository) GetResolutionByThumbnailIdAndResolution(
	ctx context.Context,
	thumbnailId uuid.UUID,
	resolution string,
) (*models.ThumbnailResolution, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetResolutionByThumbnailIdAndResolution").
			Observe(time.Since(now).Seconds())
	}()

	var resolutionModel models.ThumbnailResolution
	if err := r.db.WithContext(ctx).
		Where("thumbnail_id = ? AND resolution = ?", thumbnailId, resolution).
		First(&resolutionModel).Error; err != nil {
		return nil, err
	}

	return &resolutionModel, nil
}

func (r *ThumbnailRepository) UpdateThumbnail(
	ctx context.Context,
	thumbnail *models.Thumbnail,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateThumbnail").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Save(thumbnail).Error
}

func (r *ThumbnailRepository) DeleteResolutionsByThumbnailId(
	ctx context.Context,
	thumbnailId uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteResolutionsByThumbnailId").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Where("thumbnail_id = ?", thumbnailId).
		Delete(&models.ThumbnailResolution{}).Error
}

func (r *ThumbnailRepository) DeleteThumbnailByBelongToId(
	ctx context.Context,
	belongToId uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteThumbnailByBelongToId").
			Observe(time.Since(now).Seconds())
	}()

	if err := r.db.WithContext(ctx).
		Where("thumbnail_id in (?)", r.db.Select("id").Table("thumbnails").Where("belongs_to_id = ?", belongToId)).
		Delete(&models.ThumbnailResolution{}).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Where("belongs_to_id = ?", belongToId).
		Delete(&models.Thumbnail{}).Error
}

func (r *ThumbnailRepository) DeleteThumbnailById(
	ctx context.Context,
	id uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteThumbnailById").
			Observe(time.Since(now).Seconds())
	}()

	if err := r.db.WithContext(ctx).
		Where("thumbnail_id = ?", id).
		Delete(&models.ThumbnailResolution{}).Error; err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).
		Joins("thumbnail_files", "cdn_files.id = thumbnail_files.file_id").
		Where("thumbnail_files.thumbnail_id = ?", id).
		Delete(&models.CdnFile{}).Error; err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).
		Where("thumbnail_id = ?", id).
		Delete(&models.ThumbnailFile{}).Error; err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.Thumbnail{}).Error
}
