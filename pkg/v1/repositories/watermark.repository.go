package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type watermarkRepository struct {
	db *gorm.DB
}

func MustWatermarkRepository(
	db *gorm.DB, init bool,
) models.WatermarkRepository {
	if init {
		err := db.AutoMigrate(
			&models.Watermark{},
			&models.MediaWatermark{},
			&models.WaterMarkFile{},
		)
		if err != nil {
			panic(err)
		}
	}
	return &watermarkRepository{
		db: db,
	}
}

func (r *watermarkRepository) CreateUserWatermark(
	ctx context.Context,
	newWatermark *models.Watermark,
) (*models.Watermark, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateWatermark").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Create(newWatermark).Error; err != nil {
		return nil, err
	}

	return newWatermark, nil
}

func (r *watermarkRepository) CreateFile(
	ctx context.Context,
	newFile *models.WaterMarkFile,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateWatermarkFile").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Create(newFile).Error; err != nil {
		return err
	}

	return nil
}

func (r *watermarkRepository) ListAllWatermarks(
	ctx context.Context,
	filter models.GetWatermarkList,
) ([]*models.Watermark, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("ListAllWatermarks").
			Observe(time.Since(t).Seconds())
	}()

	var (
		result []*models.Watermark
		total  int64
	)

	query := r.db.WithContext(ctx).Model(&models.Watermark{}).Where(
		"user_id = ?",
		filter.UserId,
	)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order(filter.SortBy + " " + filter.Order).Offset(int(filter.Offset)).Limit(int(filter.Limit)).Find(&result).Error; err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *watermarkRepository) DeleteWatermarkById(
	ctx context.Context, userId uuid.UUID,
	watermarkId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteWatermark").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.
		WithContext(ctx).
		Select("File, File.File").
		Where(
			"id = ? and user_id = ?",
			watermarkId,
			userId,
		).Delete(&models.Watermark{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *watermarkRepository) DeleteMediaWatermarkById(
	ctx context.Context,
	watermarkId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteMediaWatermarkByWatermarkId").
			Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Where(
		"watermark_id = ?",
		watermarkId,
	).Delete(&models.MediaWatermark{}).Error; err != nil {
		return err
	}
	return nil
}

func (r *watermarkRepository) CheckIfWatermarkExistInAnyMedia(
	ctx context.Context, watermarkId uuid.UUID,
) (bool, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CheckIfWatermarkExistInAnyMedia").
			Observe(time.Since(t).Seconds())
	}()
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Media{}).
		Joins("JOIN media_watermarks ON media.id = media_watermarks.media_id").
		Where(
			"media_watermarks.watermark_id = ?",
			watermarkId,
		).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count != 0, nil
}

func (r *watermarkRepository) CreateMediaWatermark(
	ctx context.Context,
	newMediaWatermark *models.MediaWatermark,
) (*models.MediaWatermark, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateMediaWatermark").
			Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Create(newMediaWatermark).Error; err != nil {
		return nil, err
	}
	return newMediaWatermark, nil
}

func (r *watermarkRepository) GetMediaWatermark(
	ctx context.Context,
	mediaId uuid.UUID,
	watermarkId uuid.UUID,
) (*models.MediaWatermark, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaWatermark").
			Observe(time.Since(t).Seconds())
	}()
	var result models.MediaWatermark
	if err := r.db.WithContext(ctx).Where(
		"media_id = ? and watermark_id = ?",
		mediaId,
		watermarkId,
	).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *watermarkRepository) GetUserWatermarkById(
	ctx context.Context,
	userId uuid.UUID,
	watermarkId uuid.UUID,
) (*models.Watermark, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserWatermarkById").
			Observe(time.Since(t).Seconds())
	}()
	var result models.Watermark
	if err := r.db.
		WithContext(ctx).
		Preload("File").
		Preload("File.File").
		Where(
			"id = ? and user_id = ?",
			watermarkId,
			userId,
		).First(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}
