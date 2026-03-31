package repositories

import (
	"context"
	"fmt"
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type liveStreamKeyRepository struct {
	db *gorm.DB
}

func NewLiveStreamKeyRepository(db *gorm.DB, init bool) models.LiveStreamKeyRepository {
	if init {
		db.AutoMigrate(&models.LiveStreamKey{})
	}
	return &liveStreamKeyRepository{
		db: db,
	}
}

func (r *liveStreamKeyRepository) CreateLiveStreamKey(
	ctx context.Context,
	key *models.LiveStreamKey,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateLiveStreamKey").
			Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Create(key).Error; err != nil {
		return err
	}
	return nil
}

func (r *liveStreamKeyRepository) GetLiveStreamKeys(
	ctx context.Context,
	filter models.GetLiveStreamKeysFilter,
	userId uuid.UUID,
) ([]*models.LiveStreamKey, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamKeys").
			Observe(time.Since(t).Seconds())
	}()

	var keys []*models.LiveStreamKey

	var total int64

	query := r.db.WithContext(ctx).Model(&models.LiveStreamKey{}).Where("user_id = ?", userId)

	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("error counting total records: %w", err)
	}

	if filter.Search != "" {
		query = query.Where("name LIKE ?", "%"+filter.Search+"%")
	}

	if filter.SortBy != "" && filter.OrderBy != "" {
		query = query.Order(fmt.Sprintf("%s %s", filter.SortBy, filter.OrderBy))
	}

	query = query.Offset(filter.Offset).Limit(filter.Limit)

	if err := query.Find(&keys).Error; err != nil {
		return nil, 0, fmt.Errorf("error fetching live stream keys: %w", err)
	}

	return keys, total, nil
}

func (r *liveStreamKeyRepository) GetLiveStreamKeyByUserId(
	ctx context.Context,
	userId uuid.UUID,
) (*models.LiveStreamKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamKeyByUserId").
			Observe(time.Since(t).Seconds())
	}()

	var key models.LiveStreamKey
	err := r.db.WithContext(ctx).Where("user_id = ?", userId).First(&key).Error
	return &key, err
}

func (r *liveStreamKeyRepository) GetLiveStreamKeyById(
	ctx context.Context,
	id uuid.UUID,
) (*models.LiveStreamKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamKeyById").
			Observe(time.Since(t).Seconds())
	}()

	var lsKey models.LiveStreamKey
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&lsKey).Error; err != nil {
		return nil, err
	}
	return &lsKey, nil
}

func (r *liveStreamKeyRepository) GetByLiveStreamKey(
	ctx context.Context,
	key uuid.UUID,
) (*models.LiveStreamKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetByLiveStreamKey").
			Observe(time.Since(t).Seconds())
	}()

	var lsKey models.LiveStreamKey
	if err := r.db.Where("stream_key = ?", key).First(&lsKey).Error; err != nil {
		return nil, err
	}
	return &lsKey, nil
}

func (r *liveStreamKeyRepository) GetLiveStreamByUserIdAndName(
	ctx context.Context,
	userId uuid.UUID,
	name string,
) (*models.LiveStreamKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamByUserIdAndName").
			Observe(time.Since(t).Seconds())
	}()
	var key models.LiveStreamKey
	err := r.db.WithContext(ctx).Where("user_id = ? AND name = ?", userId, name).First(&key).Error
	return &key, err
}

func (r *liveStreamKeyRepository) GetLiveStreamKeyByStreamKey(
	ctx context.Context,
	streamKey uuid.UUID,
) (*models.LiveStreamKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamKeyByStreamKey").
			Observe(time.Since(t).Seconds())
	}()
	var key models.LiveStreamKey
	err := r.db.WithContext(ctx).Where("stream_key = ?", streamKey).First(&key).Error
	return &key, err
}

func (r *liveStreamKeyRepository) UpdateLiveStreamKey(
	ctx context.Context,
	userId uuid.UUID,
	id uuid.UUID,
	input models.UpdateLiveStreamKeyInput,
) (*models.LiveStreamKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateLiveStreamKey").
			Observe(time.Since(t).Seconds())
	}()

	var updatedKey models.LiveStreamKey
	// Ensure the Save field is included in the update
	err := r.db.WithContext(ctx).
		Model(&models.LiveStreamKey{}).
		Where("id = ? AND user_id = ?", id, userId).
		Updates(map[string]interface{}{
			"name": input.Name,
			"save": input.Save,
		}).
		First(&updatedKey).
		Error
	if err != nil {
		return nil, err
	}
	return &updatedKey, nil
}

func (r *liveStreamKeyRepository) Update(ctx context.Context, lsKey *models.LiveStreamKey) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("Update").Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Save(lsKey).Error; err != nil {
		return err
	}
	return nil
}

func (r *liveStreamKeyRepository) DeleteUserLiveStreamKey(
	ctx context.Context,
	id uuid.UUID,
	userId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteLiveStreamKey").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Where("live_stream_key_id = ?", id).Delete(&models.LiveStreamMedia{}).Error; err != nil {
			return err
		}

		if err := tx.WithContext(ctx).Where("id = ? AND user_id = ?", id, userId).Delete(&models.LiveStreamKey{}).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *liveStreamKeyRepository) DeleteAllUserLivestreamKey(
	ctx context.Context,
	userId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteAllUserLivestreamKey").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Where("user_id = ?", userId).Delete(&models.LiveStreamKey{}).Error
}
