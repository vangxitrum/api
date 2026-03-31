package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type CdnFileRepository struct {
	db *gorm.DB
}

func MustNewCdnFileRepository(
	db *gorm.DB,
	init bool,
) models.CdnFileRepository {
	if init {
		if err := db.AutoMigrate(&models.CdnFile{}); err != nil {
			panic(err)
		}
	}

	return &CdnFileRepository{
		db: db,
	}
}

func (r *CdnFileRepository) Create(
	ctx context.Context,
	cdnFile *models.CdnFile,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateCdnFile").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(cdnFile).Error
}

func (r *CdnFileRepository) GetCdnFiles(
	ctx context.Context,
) ([]*models.CdnFile, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnFiles").
			Observe(time.Since(now).Seconds())
	}()

	var cdnFiles []*models.CdnFile
	if err := r.db.WithContext(ctx).
		Find(&cdnFiles).Error; err != nil {
		return nil, err
	}

	return cdnFiles, nil
}

func (r *CdnFileRepository) GetCdnFileByFileId(
	ctx context.Context,
	id uuid.UUID,
) (*models.CdnFile, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnFileById").
			Observe(time.Since(now).Seconds())
	}()

	var cdnFile models.CdnFile
	if err := r.db.WithContext(ctx).
		Where("file_id = ?", id).
		First(&cdnFile).Error; err != nil {
		return nil, err
	}

	return &cdnFile, nil
}

func (r *CdnFileRepository) GetCdnFilesByBelongsToId(
	ctx context.Context,
	belongsToId uuid.UUID,
) ([]*models.CdnFile, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnFilesByBelongsToId").
			Observe(time.Since(now).Seconds())
	}()

	var cdnFiles []*models.CdnFile
	if err := r.db.WithContext(ctx).
		Where("belongs_to_id = ?", belongsToId).
		Find(&cdnFiles).Error; err != nil {
		return nil, err
	}

	return cdnFiles, nil
}

func (r *CdnFileRepository) GetCdnFileByBelongsToIdAndType(
	ctx context.Context,
	id uuid.UUID,
	fileType string,
) (*models.CdnFile, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnFileByBelongsToIdAndType").
			Observe(time.Since(now).Seconds())
	}()

	var cdnFile models.CdnFile
	if err := r.db.WithContext(ctx).
		Where("belongs_to_id = ? and type = ?", id, fileType).
		First(&cdnFile).Error; err != nil {
		return nil, err
	}

	return &cdnFile, nil
}

func (r *CdnFileRepository) GetCdnFilesByBelongsToIdAndType(
	ctx context.Context,
	id uuid.UUID,
	fileType string,
) ([]*models.CdnFile, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnFilesByBelongsToIdAndType").
			Observe(time.Since(now).Seconds())
	}()

	var cdnFiles []*models.CdnFile
	if err := r.db.WithContext(ctx).
		Where("belongs_to_id = ? and type = ?", id, fileType).
		Find(&cdnFiles).Error; err != nil {
		return nil, err
	}

	return cdnFiles, nil
}

func (r *CdnFileRepository) GetCdnFileByBelongsToIdAndIndexAndType(
	ctx context.Context,
	id uuid.UUID,
	index int,
	fileType string,
) (*models.CdnFile, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnFileByBelongsToIdAndIndex").
			Observe(time.Since(now).Seconds())
	}()

	var cdnFile models.CdnFile
	if err := r.db.WithContext(ctx).
		Where("belongs_to_id = ? and index = ? and type = ?", id, index, fileType).
		First(&cdnFile).Error; err != nil {
		return nil, err
	}

	return &cdnFile, nil
}

func (r *CdnFileRepository) GetUserTotalStorage(
	ctx context.Context,
	userId uuid.UUID,
) (int64, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserTotalStorage").
			Observe(time.Since(now).Seconds())
	}()

	var totalStorage int64
	if err := r.db.WithContext(ctx).
		Model(&models.CdnFile{}).
		Where("created_by = ? and  type in ?", userId, models.BilledCdnTypes).
		Select("coalesce(sum(size),0) as size").
		Row().
		Scan(&totalStorage); err != nil {
		return 0, err
	}

	return totalStorage, nil
}

func (r *CdnFileRepository) GetMediaTotalStorage(
	ctx context.Context,
	mediaId uuid.UUID,
) (int64, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaTotalStorage").
			Observe(time.Since(now).Seconds())
	}()

	var totalStorage int64
	if err := r.db.WithContext(ctx).
		Raw(`
		select coalesce(sum("size"),0) as sum
		from (
			(select cf."size"
			from media v
			join cdn_files cf  on v.id = cf.belongs_to_id and v.id = ?)
			union
			(select cf."size"
			from media v
			join streams s on v.id = s.media_id and v.id = ?
			join cdn_files cf on s.id = cf.belongs_to_id)
			union
			(
			select cf."size"
			from media v
			join qualities q on v.id = q.media_id and v.id = ?
			join cdn_files cf  on q.id = cf.belongs_to_id)
		)
	`, mediaId, mediaId, mediaId).
		Scan(&totalStorage).Error; err != nil {
		return 0, err
	}

	return totalStorage, nil
}

func (r *CdnFileRepository) GetCdnFilesByType(
	ctx context.Context,
	fileType string,
) ([]*models.CdnFile, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnFilesByType(").
			Observe(time.Since(now).Seconds())
	}()

	var cdnFiles []*models.CdnFile
	if err := r.db.WithContext(ctx).
		Where("type = ?", fileType).
		Find(&cdnFiles).Error; err != nil {
		return nil, err
	}

	return cdnFiles, nil
}

func (r *CdnFileRepository) UpdateCdnFile(
	ctx context.Context,
	cdnFile *models.CdnFile,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateCdnFile").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Save(cdnFile).Error
}

func (r *CdnFileRepository) DeleteCdnFileByFileId(
	ctx context.Context,
	fileId string,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteCdnFileById").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Where("id = ?", fileId).
		Delete(&models.CdnFile{}).Error
}

func (r *CdnFileRepository) DeleteCdnFilesByBelongsToId(
	ctx context.Context,
	belongsToId uuid.UUID,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteCdnFilesByBelongsToId").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.CdnFile{}, "belongs_to_id = ?", belongsToId).Error
}

func (r *CdnFileRepository) DeleteCdnFilesByBelongsToIdAndType(
	ctx context.Context,
	belongsToId uuid.UUID,
	fileType string,
) error {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteCdnFilesByBelongsToIdAndType").
			Observe(time.Since(now).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.CdnFile{}, "belongs_to_id = ? and type = ?", belongsToId, fileType).Error
}

func (r *CdnFileRepository) GetTotalSizeStorage(
	ctx context.Context,
) (int64, error) {
	now := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetTotalSizeStorage").
			Observe(time.Since(now).Seconds())
	}()

	var totalStorage int64
	if err := r.db.WithContext(ctx).Model(&models.CdnFile{}).
		Select("coalesce(sum(size),0) as size").
		Row().
		Scan(&totalStorage); err != nil {
		return 0, err
	}
	return totalStorage, nil
}
