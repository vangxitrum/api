package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type playerThemeRepository struct {
	db *gorm.DB
}

func MustNewPlayerThemeRepository(
	db *gorm.DB, init bool,
) models.PlayerThemeRepository {
	if init {
		err := db.AutoMigrate(
			&models.PlayerTheme{})
		if err != nil {
			panic(err)
		}
	}
	return &playerThemeRepository{
		db: db,
	}
}

func (r *playerThemeRepository) CreatePlayerTheme(
	ctx context.Context,
	newPlayerTheme *models.PlayerTheme,
) (*models.PlayerTheme, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreatePlayerTheme").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Create(newPlayerTheme).Error; err != nil {
		return nil, err
	}

	return newPlayerTheme, nil
}

func (r *playerThemeRepository) GetUserPlayerThemeById(
	ctx context.Context, userId, id uuid.UUID,
) (*models.PlayerTheme, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserPlayerThemeById").
			Observe(time.Since(t).Seconds())
	}()

	var playerTheme models.PlayerTheme
	if err := r.db.WithContext(ctx).
		Preload("Asset.File").
		Where(
			"id = ? AND user_id = ?",
			id,
			userId,
		).First(&playerTheme).Error; err != nil {
		return nil, err
	}

	return &playerTheme, nil
}

func (r *playerThemeRepository) GetPlayerThemeById(
	ctx context.Context, id uuid.UUID,
) (*models.PlayerTheme, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlayerThemeById").
			Observe(time.Since(t).Seconds())
	}()

	var playerTheme models.PlayerTheme
	if err := r.db.WithContext(ctx).
		Preload("Asset.File").
		Where("id = ?", id).
		First(&playerTheme).Error; err != nil {
		return nil, err
	}

	return &playerTheme, nil
}

func (r *playerThemeRepository) DeletePlayerThemeById(
	ctx context.Context, userId, id uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeletePlayerThemeById").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Where(
		"user_id = ? AND id = ?",
		userId,
		id,
	).Delete(&models.PlayerTheme{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *playerThemeRepository) UpdatePlayerThemeById(
	ctx context.Context,
	id uuid.UUID,
	updatedPlayerTheme *models.PlayerTheme,
) (*models.PlayerTheme, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdatePlayerThemeById").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Model(&models.PlayerTheme{}).
		Where("id = ?", id).
		Updates(updatedPlayerTheme)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return updatedPlayerTheme, nil
}

func (r *playerThemeRepository) GetThemePlayerList(
	ctx context.Context,
	filter models.GetApiKeyListInput,
) ([]*models.PlayerTheme, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetThemePlayerList").
			Observe(time.Since(t).Seconds())
	}()

	var (
		result []*models.PlayerTheme
		total  int64
	)

	query := r.db.WithContext(ctx).Model(&models.PlayerTheme{}).Where(
		"user_id = ?",
		filter.UserId,
	)

	if filter.Search != "" {
		query = query.Where(
			"name ILIKE ?",
			"%"+filter.Search+"%",
		)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order(filter.SortBy + " " + filter.Order).Offset(int(filter.Offset)).Limit(int(filter.Limit)).Find(&result).Error; err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *playerThemeRepository) GetPlayerThemeList(
	ctx context.Context,
	filter models.GetThemePlayerList,
) ([]*models.PlayerTheme, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlayerThemeList").
			Observe(time.Since(t).Seconds())
	}()

	var (
		result []*models.PlayerTheme
		total  int64
	)

	query := r.db.WithContext(ctx).Model(&models.PlayerTheme{}).Where(
		"user_id = ?",
		filter.UserId,
	)

	if filter.Search != "" {
		query = query.Where(
			"name ILIKE ?",
			"%"+filter.Search+"%",
		)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Order != "" {
		query = query.Order(filter.SortBy + " " + filter.Order)
	}

	if err := query.Offset(int(filter.Offset)).Limit(int(filter.Limit)).Find(&result).Error; err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (r *playerThemeRepository) UpdatePlayerThemeAsset(
	ctx context.Context, themeId uuid.UUID,
	asset models.Asset,
) (*models.PlayerTheme, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdatePlayerThemeAsset").
			Observe(time.Since(t).Seconds())
	}()

	var playerTheme models.PlayerTheme
	if err := r.db.WithContext(ctx).Where(
		"id = ?",
		themeId,
	).First(&playerTheme).Error; err != nil {
		return nil, err
	}

	playerTheme.Asset = asset
	if err := r.db.WithContext(ctx).Save(&playerTheme).Error; err != nil {
		return nil, err
	}

	return &playerTheme, nil
}

func (r *playerThemeRepository) DeletePlayerThemeAsset(
	ctx context.Context,
	userId, themeId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeletePlayerThemeAsset").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Model(&models.PlayerTheme{}).
		Where(
			"user_id = ? AND id = ?",
			userId,
			themeId,
		).
		Updates(
			map[string]any{
				"player_asset_file_id":   gorm.Expr("NULL"),
				"player_asset_logo_link": "",
			},
		)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *playerThemeRepository) AddPlayerThemeToMediaById(
	ctx context.Context,
	themeId, mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("AddPlayerThemeToMediaById").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Model(
		&models.Media{},
	).Where(
		"id = ?",
		mediaId,
	).Updates(
		map[string]any{
			"player_theme_id": themeId,
		},
	).Error; err != nil {
		return err
	}

	return nil
}

func (r *playerThemeRepository) RemovePlayerThemeFromMedia(
	ctx context.Context,
	themeId, mediaId, userId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("RemovePlayerThemeFromMedia").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Model(
		&models.Media{},
	).Where(
		"id = ? AND player_theme_id = ? AND user_id = ?",
		mediaId,
		themeId,
		userId,
	).Updates(
		map[string]any{
			"player_theme_id": nil,
		},
	).Error; err != nil {
		return err
	}

	return nil
}

func (r *playerThemeRepository) GetActiveMediaByPlayerThemeId(
	ctx context.Context,
	userId uuid.UUID,
	themeId uuid.UUID,
) (*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetActiveMediaByPlayerThemeId").
			Observe(time.Since(t).Seconds())
	}()

	var media models.Media
	if err := r.db.WithContext(ctx).Where(
		"player_theme_id = ? AND user_id = ? AND status != ?",
		themeId, userId, models.DeletedStatus,
	).First(&media).Error; err != nil {
		return nil, err
	}

	return &media, nil
}

func (r *playerThemeRepository) GetDefaultPlayerTheme(
	ctx context.Context,
	userId uuid.UUID,
) (*models.PlayerTheme, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetDefaultPlayerTheme").
			Observe(time.Since(t).Seconds())
	}()

	var playerTheme models.PlayerTheme
	if err := r.db.WithContext(ctx).Where(
		"user_id = ? AND is_default = ?",
		userId, true,
	).First(&playerTheme).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
	}

	return &playerTheme, nil
}

func (r *playerThemeRepository) UpdateDefaultPlayerTheme(
	ctx context.Context,
	userId, themeId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateDefaultPlayerTheme").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Model(
		&models.PlayerTheme{},
	).Where(
		"user_id = ? AND id = ?",
		userId, themeId,
	).Updates(
		map[string]any{
			"is_default": false,
		},
	).Error; err != nil {
		return err
	}

	return nil
}
