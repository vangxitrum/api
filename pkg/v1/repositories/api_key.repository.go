package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type apiKeyRepository struct {
	db *gorm.DB
}

func MustNewApiKeyRepository(
	db *gorm.DB, init bool,
) models.ApiKeyRepository {
	if init {
		err := db.AutoMigrate(&models.ApiKey{})
		if err != nil {
			panic(err)
		}
	}
	return &apiKeyRepository{
		db: db,
	}
}

func (r *apiKeyRepository) CreateApiKey(
	ctx context.Context, newApiKey *models.ApiKey,
) (*models.ApiKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateApiKey").Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Create(newApiKey).Error; err != nil {
		return nil, err
	}

	return newApiKey, nil
}

func (r *apiKeyRepository) GetApiKeyList(
	ctx context.Context,
	filter models.GetApiKeyListInput,
) ([]*models.ApiKey, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetApiKeyList").Observe(time.Since(t).Seconds())
	}()

	var (
		result []*models.ApiKey
		total  int64
	)

	query := r.db.WithContext(ctx).Model(&models.ApiKey{}).Where(
		"user_id = ?",
		filter.UserId,
	)

	if filter.Search != "" {
		query = query.Where(
			"name ILIKE ?",
			"%"+filter.Search+"%",
		)
	}

	if filter.Type != "" {
		query = query.Where(
			"type = ?",
			filter.Type,
		)
	}

	query = query.Where("status != ? and status != ?", models.DeletedStatus, models.ExpiredStatus)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order(filter.SortBy + " " + filter.Order).
		Offset(int(filter.Offset)).
		Limit(int(filter.Limit)).
		Find(&result).
		Error; err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *apiKeyRepository) GetApiKeyById(
	ctx context.Context, id uuid.UUID,
) (*models.ApiKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetApiKeyById").Observe(time.Since(t).Seconds())
	}()

	var apiKey models.ApiKey

	if err := r.db.WithContext(ctx).Where(
		&models.ApiKey{
			Id: id,
		},
	).First(&apiKey).Error; err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func (r *apiKeyRepository) GetApiKeyByUserId(
	ctx context.Context, id uuid.UUID,
) ([]*models.ApiKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetApiKeyByUserId").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.ApiKey

	if err := r.db.WithContext(ctx).Where(
		&models.ApiKey{
			UserId: id,
		},
	).First(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *apiKeyRepository) GetApiKeyByKey(
	ctx context.Context, key string,
) (*models.ApiKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetApiKeyByKey").
			Observe(time.Since(t).Seconds())
	}()

	var apiKey models.ApiKey
	if err := r.db.WithContext(ctx).Where(
		&models.ApiKey{
			PublicKey: key,
		},
	).First(&apiKey).Error; err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepository) GetApiKeyListBetweenTime(
	ctx context.Context, userId uuid.UUID,
	from time.Time, to time.Time,
) ([]*models.ApiKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetApiKeyListBetweenTime").
			Observe(time.Since(t).Seconds())
	}()

	var result []*models.ApiKey

	if err := r.db.WithContext(ctx).Where(
		"user_id = ? and created_at >= ? and created_at <= ?",
		userId,
		from,
		to,
	).
		Find(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *apiKeyRepository) DeleteUserApiKeyById(
	ctx context.Context, apiKeyId uuid.UUID, userId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteUserApiKeyById").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).
		Model(&models.ApiKey{}).
		Where("id = ? and user_id = ?", apiKeyId, userId).
		Update("status", models.DeletedStatus).Error; err != nil {
		return err
	}

	return nil
}

func (r *apiKeyRepository) UpdateApiKeyName(
	ctx context.Context, apiKey *models.ApiKey,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateApiKey").Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Model(&models.ApiKey{}).Where(
		"id = ?",
		apiKey.Id,
	).Update(
		"name",
		apiKey.Name,
	).Error; err != nil {
		return err
	}
	return nil
}

func (r *apiKeyRepository) GetUserApiKeyByName(
	ctx context.Context, userId uuid.UUID,
	apiKeyName string,
) (*models.ApiKey, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetApiKeyByName").
			Observe(time.Since(t).Seconds())
	}()

	var result *models.ApiKey
	if err := r.db.WithContext(ctx).Where(
		&models.ApiKey{
			UserId: userId,
			Name:   apiKeyName,
		},
	).First(&result).Error; err != nil {
		return nil, err
	}

	return result, nil
}

func (r *apiKeyRepository) DeleteExpiredApiKey(ctx context.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteExpiredApiKey").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(&models.ApiKey{}).
		Where("expired_at < ?", time.Now()).
		Update("status", models.ExpiredStatus).
		Error
}

func (r *apiKeyRepository) DeleteUserAPIKeys(
	ctx context.Context,
	userId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteUserAPIKeys").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).
		Model(&models.ApiKey{}).
		Where("user_id = ?", userId).
		Update("status", models.DeletedStatus).Error; err != nil {
		return err
	}

	return nil
}

func (r *apiKeyRepository) UpdateApiKeyLastRequestedAt(
	ctx context.Context,
	apiKeyId uuid.UUID,
	requestedAt time.Time,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateApiKeyLastUsedAt").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Model(&models.ApiKey{}).Where(
		"id = ?",
		apiKeyId,
	).Update(
		"last_requested_at",
		time.Now(),
	).Error; err != nil {
		return err
	}

	return nil
}
