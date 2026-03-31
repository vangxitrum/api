package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type playlistRepository struct {
	db *gorm.DB
}

func MustNewPlaylistRepository(
	db *gorm.DB, init bool,
) models.PlaylistRepository {
	if init {
		err := db.AutoMigrate(
			&models.Playlist{},
			&models.PlaylistItem{},
			&models.PlaylistThumbnail{},
		)
		if err != nil {
			panic(err)
		}
	}
	return &playlistRepository{
		db: db,
	}
}

func (r *playlistRepository) CreatePlaylist(
	ctx context.Context,
	playlist *models.Playlist,
) (*models.Playlist, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreatePlaylist").
			Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).Create(playlist).Error; err != nil {
		return nil, err
	}

	return playlist, nil
}

func (r *playlistRepository) CreatePlaylistThumbnail(
	ctx context.Context,
	thumbnail *models.PlaylistThumbnail,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreatePlaylistThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Create(thumbnail).Error; err != nil {
		return err
	}

	return nil
}

func (r *playlistRepository) CreateFile(
	ctx context.Context,
	file *models.MediaQualityFile,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateFile").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Create(file).Error; err != nil {
		return err
	}

	return nil
}

func (r *playlistRepository) GetPlaylistById(
	ctx context.Context,
	playlistId uuid.UUID,
) (*models.Playlist, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlaylistById").
			Observe(time.Since(t).Seconds())
	}()

	playlist := &models.Playlist{}
	if err := r.db.WithContext(ctx).
		Preload("MediaItems").
		Preload("MediaItems.Media").
		Preload("MediaItems.Media.Format").
		Preload("MediaItems.Media.Chapters").
		Preload("MediaItems.Media.Captions").
		Preload("Thumbnail").
		Preload("Thumbnail.Thumbnail").
		Preload("Thumbnail.Thumbnail.Resolutions").
		Preload("Thumbnail.Thumbnail.File").
		Where("id = ?", playlistId).
		First(playlist).Error; err != nil {
		return nil, err
	}

	return playlist, nil
}

func (r *playlistRepository) GetPlaylistByIds(
	ctx context.Context,
	playlistIds []uuid.UUID,
) ([]models.Playlist, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlaylistByIds").
			Observe(time.Since(t).Seconds())
	}()

	if len(playlistIds) == 0 {
		return []models.Playlist{}, nil
	}
	var playlists []models.Playlist
	if err := r.db.WithContext(ctx).
		Where("id IN ?", playlistIds).
		Find(&playlists).Error; err != nil {
		return nil, err
	}

	return playlists, nil
}

func (r *playlistRepository) DeletePlaylistById(
	ctx context.Context,
	userId,
	playlistId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeletePlaylistById").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", playlistId, userId).
		Delete(&models.Playlist{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *playlistRepository) GetUserPlaylists(
	ctx context.Context,
	userId uuid.UUID,
	input models.PlaylistFilter,
) ([]*models.Playlist, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserPlaylists").
			Observe(time.Since(t).Seconds())
	}()

	var playlists []*models.Playlist
	var total int64
	query := r.db.WithContext(ctx).
		Model(&models.Playlist{}).
		Preload("MediaItems").
		Preload("MediaItems.Media").
		Preload("MediaItems.Media.Format").
		Preload("Thumbnail").
		Preload("Thumbnail.Thumbnail").
		Preload("Thumbnail.Thumbnail.Resolutions").
		Preload("Thumbnail.Thumbnail.File").
		Where("user_id = ?", userId)

	if input.SortBy != "" {
		query = query.Order(fmt.Sprintf("%s %s", input.SortBy, input.OrderBy))
	}

	if len(input.Metadata) > 0 {
		for _, data := range input.Metadata {
			query = query.Where("metadata->>? = ?", data.Key, data.Value)
		}
	}

	if input.Search != "" {
		query = query.Where("name ILIKE ?", "%"+input.Search+"%")
	}

	if input.PlaylistType != "" {
		query = query.Where("playlist_type = ?", input.PlaylistType)
	}

	if len(input.Tags) > 0 {
		query = query.Where(
			"tags ilike ?",
			fmt.Sprintf("%%%s%%", strings.Join(input.Tags, ",")),
		)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.Playlist{}, 0, nil
	}

	if err := query.
		Offset(input.Offset).
		Limit(input.Limit).
		Find(&playlists).
		Error; err != nil {
		return nil, 0, err
	}

	return playlists, total, nil
}

func (r *playlistRepository) UpdatePlaylistById(
	ctx context.Context,
	userId,
	playlistId uuid.UUID,
	playlist *models.Playlist,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdatePlaylistById").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Model(&models.Playlist{}).
		Where("id = ? AND user_id = ?", playlistId, userId).
		Updates(playlist).Error; err != nil {
		return err
	}

	return nil
}

func (r *playlistRepository) CreatePlaylistItem(
	ctx context.Context,
	playlistItem *models.PlaylistItem,
) (*models.PlaylistItem, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreatePlaylistItem").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Create(playlistItem).Error; err != nil {
		return nil, err
	}

	return playlistItem, nil
}

func (r *playlistRepository) GetPlaylistItemById(
	ctx context.Context,
	id uuid.UUID,
) (*models.PlaylistItem, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlaylistItemById").
			Observe(time.Since(t).Seconds())
	}()

	var item models.PlaylistItem
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *playlistRepository) UpdatePlaylistItem(
	ctx context.Context,
	item *models.PlaylistItem,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdatePlaylistItem").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).
		Model(&models.PlaylistItem{}).
		Where("id = ?", item.Id).
		Updates(item)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *playlistRepository) DeletePlaylistItemById(
	ctx context.Context,
	playlistId, mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeletePlaylistItemById").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).
		Delete(&models.PlaylistItem{}, "playlist_id = ? AND id = ?", playlistId, mediaId)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *playlistRepository) GetExistsAnyMediaInPlaylists(
	ctx context.Context,
	playlistIds []uuid.UUID,
	mediaIds []uuid.UUID,
) (bool, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("ExistsAnyMediaInPlaylists").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.PlaylistItem{}).
		Where("playlist_id IN ?", playlistIds).
		Where("media_id IN ?", mediaIds).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *playlistRepository) DeletePlaylistItemByPlaylistId(
	ctx context.Context,
	playlistId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeletePlaylistItemByPlaylistId").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).
		Where("playlist_id = ?", playlistId).
		Delete(&models.PlaylistItem{})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *playlistRepository) GetLastPlaylistItemByPlaylistId(
	ctx context.Context,
	playlistId uuid.UUID,
) (*models.PlaylistItem, error) {
	var lastItem models.PlaylistItem

	result := r.db.WithContext(ctx).
		Where("playlist_id = ?", playlistId).
		Where("next_id IS NULL").
		First(&lastItem)

	if result.Error != nil {
		return nil, result.Error
	}

	return &lastItem, nil
}

func (r *playlistRepository) GetFirstPlaylistItemByPlaylistId(
	ctx context.Context,
	playlistId uuid.UUID,
) (*models.PlaylistItem, error) {
	var item models.PlaylistItem
	result := r.db.WithContext(ctx).
		Where("playlist_id = ? AND previous_id IS NULL", playlistId).
		First(&item)
	if result.Error != nil {
		return nil, result.Error
	}

	return &item, nil
}

func (r *playlistRepository) UpdatePlaylistItemsPosition(
	ctx context.Context,
	items []*models.PlaylistItem,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdatePlaylistItemsPosition").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			updates := map[string]any{
				"media_id":    item.MediaId,
				"next_id":     item.NextId,
				"previous_id": item.PreviousId,
				"updated_at":  time.Now(),
			}

			if err := tx.Model(&models.PlaylistItem{}).Where("id = ?", item.Id).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *playlistRepository) UpdatePlaylistItemPosition(
	ctx context.Context,
	item *models.PlaylistItem,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdatePlaylistItemPosition").
			Observe(time.Since(t).Seconds())
	}()

	updates := map[string]any{
		"media_id":    item.MediaId,
		"next_id":     item.NextId,
		"previous_id": item.PreviousId,
		"updated_at":  time.Now(),
	}

	result := r.db.WithContext(ctx).
		Model(&models.PlaylistItem{}).
		Where("id = ?", item.Id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *playlistRepository) GetPlaylistItemsByPlaylistId(
	ctx context.Context,
	playlistId uuid.UUID,
) ([]*models.PlaylistItem, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlaylistItemsByPlaylistId").
			Observe(time.Since(t).Seconds())
	}()

	var items []*models.PlaylistItem
	result := r.db.WithContext(ctx).
		Where("playlist_id = ?", playlistId).
		Find(&items)
	if result.Error != nil {
		return nil, result.Error
	}

	return items, nil
}

func (r *playlistRepository) GetPlaylistItemCount(
	ctx context.Context,
	playlistId uuid.UUID,
) (int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlaylistItemCount").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	result := r.db.WithContext(ctx).
		Model(&models.PlaylistItem{}).
		Where("playlist_id = ?", playlistId).
		Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}

	return count, nil
}

func (r *playlistRepository) GetPlaylistItemMediaById(
	ctx context.Context,
	playlistId uuid.UUID,
) ([]*models.PlaylistItem, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlaylistItemMediaById").
			Observe(time.Since(t).Seconds())
	}()

	var items []*models.PlaylistItem
	result := r.db.WithContext(ctx).
		Preload("Media").
		Preload("Media.MediaQualities").
		Preload("Media.Format", func(db *gorm.DB) *gorm.DB {
			return db.Select("media_id, duration")
		}).
		Where("playlist_id = ?", playlistId).
		Find(&items)

	return items, result.Error
}

func (r *playlistRepository) GetPlaylistItemMediaByIdWithFilter(
	ctx context.Context,
	playlistId uuid.UUID,
	filter *models.PlaylistItemFilter,
) ([]*models.PlaylistItem, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlaylistItemMediaByIdWithFilter").
			Observe(time.Since(t).Seconds())
	}()

	query := r.db.WithContext(ctx).
		Preload("Media").
		Preload("Media.MediaQualities").
		Joins("LEFT JOIN media ON playlist_items.media_id = media.id").
		Joins("LEFT JOIN media_formats ON media.id = media_formats.media_id").
		Where("playlist_items.playlist_id = ?", playlistId)

	if filter != nil && filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where(
			r.db.Where("LOWER(media.title) LIKE LOWER(?)", searchTerm),
		)
	}

	if filter != nil && filter.SortBy != "" {
		orderStr := "ASC"
		if filter.OrderBy == "desc" {
			orderStr = "DESC"
		}

		switch filter.SortBy {
		case "title":
			query = query.Order(fmt.Sprintf("media.title %s", orderStr))
		case "duration":
			orderClause := fmt.Sprintf(`
             CASE
                WHEN media_formats.duration IS NOT NULL AND media_formats.duration ~ '^[0-9]+\.?[0-9]*$'
                THEN CAST(media_formats.duration AS FLOAT)
                ELSE 0
             END %s`, orderStr)
			query = query.Order(orderClause)
		case "created_at":
			query = query.Order(
				fmt.Sprintf("playlist_items.created_at %s", orderStr),
			)
		default:
			query = query.Order("playlist_items.created_at ASC")
		}
	} else {
		query = query.Order("playlist_items.created_at ASC")
	}

	var items []*models.PlaylistItem
	err := query.Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, err
}

func (r *playlistRepository) DeletePlaylistThumbnail(
	ctx context.Context,
	playlistId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeletePlaylistThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).
		Where("playlist_id = ?", playlistId).
		Delete(&models.PlaylistThumbnail{}).Error; err != nil {
		return err
	}

	return nil
}

func (r *playlistRepository) GetPlaylistItemCounts(
	ctx context.Context,
	playlistIds []uuid.UUID,
) (map[uuid.UUID]int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetPlaylistItemCounts").
			Observe(time.Since(t).Seconds())
	}()

	if len(playlistIds) == 0 {
		return make(map[uuid.UUID]int64), nil
	}

	type Result struct {
		PlaylistId uuid.UUID
		Count      int64
	}

	var results []Result
	err := r.db.WithContext(ctx).
		Model(&models.PlaylistItem{}).
		Select("playlist_id, COUNT(*) as count").
		Where("playlist_id IN ?", playlistIds).
		Group("playlist_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	countsMap := make(map[uuid.UUID]int64)
	for _, result := range results {
		countsMap[result.PlaylistId] = result.Count
	}

	for _, id := range playlistIds {
		if _, exists := countsMap[id]; !exists {
			countsMap[id] = 0
		}
	}

	return countsMap, nil
}

func (r *playlistRepository) CheckMediaExistsInPlaylists(
	ctx context.Context,
	playlistIds []uuid.UUID,
	mediaId uuid.UUID,
) ([]*models.PlaylistItem, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CheckMediaExistsInPlaylists").
			Observe(time.Since(t).Seconds())
	}()

	var items []*models.PlaylistItem

	err := r.db.WithContext(ctx).
		Where("playlist_id IN ?", playlistIds).
		Where("media_id = ?", mediaId).
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}
