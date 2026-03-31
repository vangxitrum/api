package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type MediaRepository struct {
	db *gorm.DB
}

func MustNewMediaRepository(db *gorm.DB, init bool) models.MediaRepository {
	if init {
		if err := db.AutoMigrate(
			&models.Media{},
			&models.MediaFile{},
			&models.MediaThumbnail{},
		); err != nil {
			panic(err)
		}
	}

	return &MediaRepository{
		db: db,
	}
}

func (r *MediaRepository) Create(
	ctx context.Context,
	media *models.Media,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateMedia").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(media).Error
}

func (r *MediaRepository) CreateFile(
	ctx context.Context,
	file *models.MediaFile,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateFile").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(file).Error
}

func (r *MediaRepository) CreateMediaThumbnail(
	ctx context.Context,
	thumbnail *models.MediaThumbnail,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateMediaThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(thumbnail).Error
}

func (r *MediaRepository) GetMediaById(
	ctx context.Context,
	id uuid.UUID,
) (*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaById").
			Observe(time.Since(t).Seconds())
	}()

	var media models.Media
	if err := r.db.WithContext(ctx).
		Preload("Parts", func(db *gorm.DB) *gorm.DB {
			return db.Order("index")
		}).
		Preload("Streams").
		Preload("Format").
		Preload("Watermark").
		Preload("Watermark.Watermark").
		Preload("MediaQualities").
		Preload("MediaQualities.Files").
		Preload("MediaQualities.Files.File").
		Preload("PlayerTheme").
		Preload("Captions", "media_captions.status = ?", models.DoneStatus).
		Preload("Chapters").
		Preload("MediaFiles").
		Preload("MediaFiles.File").
		Preload("MediaThumbnail").
		Preload("MediaThumbnail.Thumbnail").
		Preload("MediaThumbnail.Thumbnail.Resolutions").
		Preload("MediaThumbnail.Thumbnail.File").
		Where("id = ?", id).
		First(&media).Error; err != nil {
		return nil, err
	}

	return &media, nil
}

func (r *MediaRepository) GetManyMediasByIds(
	ctx context.Context,
	ids []uuid.UUID,
	userId uuid.UUID,
) ([]models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetManyMediasByIds").
			Observe(time.Since(t).Seconds())
	}()

	var medias []models.Media
	if err := r.db.WithContext(ctx).
		Where("id IN ? AND user_id = ?", ids, userId).
		Find(&medias).Error; err != nil {
		return nil, err
	}
	return medias, nil
}

func (r *MediaRepository) GetCompletedLiveStreamMedias(
	ctx context.Context,
) ([]*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCompletedLiveStreamMedias").
			Observe(time.Since(t).Seconds())
	}()

	var media []*models.Media
	if err := r.db.WithContext(ctx).
		Table("media v").
		Joins("JOIN live_stream_media s ON s.media_id = v.id and s.status = 'done' and s.save = true").
		Find(&media).Error; err != nil {
		return nil, err
	}

	return media, nil
}

func (r *MediaRepository) GetSavedLivestreamMedias(
	ctx context.Context,
) ([]*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCompletedLiveStreamMedias").
			Observe(time.Since(t).Seconds())
	}()

	var media []*models.Media
	if err := r.db.WithContext(ctx).
		Table("media v").
		Joins("JOIN live_stream_media s ON s.media_id = v.id and s.status = ? and s.save = true", models.LiveStreamStatusEnd).
		Preload("MediaQualities").
		Where("v.status = ?", models.HiddenStatus).
		Find(&media).Error; err != nil {
		return nil, err
	}

	return media, nil
}

func (r *MediaRepository) GetMostViewedMedia(
	ctx context.Context,
	mediaType string,
	limit int,
) ([]*models.MediaViewData, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMostViewMedias").
			Observe(time.Since(t).Seconds())
	}()

	var medias []*models.MediaViewData
	switch mediaType {
	case models.VideoMediaType:
		if err := r.db.WithContext(ctx).
			Table("media").
			Select("media.id, media.title, view").
			Limit(limit).
			Order("view DESC").
			Where("status = ?", models.DoneStatus).
			Find(&medias).Error; err != nil {
			return nil, err
		}
	case models.StreamMediaType:
		if err := r.db.WithContext(ctx).
			Table("live_stream_media").
			Select("live_stream_media.id, live_stream_media.title , total_view").
			Limit(limit).
			Order("total_view DESC").
			Find(&medias).Error; err != nil {
			return nil, err
		}
	}

	return medias, nil
}

func (r *MediaRepository) GetMediaByStatus(
	ctx context.Context,
	status string,
) (*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaByStatus").
			Observe(time.Since(t).Seconds())
	}()

	var media models.Media
	if err := r.db.WithContext(ctx).
		Preload("Parts").
		Preload("MediaQualities").
		Preload("Streams").
		Preload("Format").
		Where("status = ?", status).
		Order("created_at").
		First(&media).Error; err != nil {
		return nil, err
	}

	return &media, nil
}

func (r *MediaRepository) GetMediaByStatuses(
	ctx context.Context,
	statuses []string,
) ([]*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaByStatuses").
			Observe(time.Since(t).Seconds())
	}()

	var media []*models.Media
	if err := r.db.WithContext(ctx).
		Preload("Parts").
		Preload("MediaQualities").
		Preload("Streams").
		Preload("Format").
		Where("status in (?)", statuses).
		Order("created_at").
		Find(&media).Error; err != nil {
		return nil, err
	}

	return media, nil
}

func (r *MediaRepository) GetMediasByStatus(
	ctx context.Context,
	status string,
) ([]*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMediaByStatus").
			Observe(time.Since(t).Seconds())
	}()

	var media []*models.Media
	if err := r.db.WithContext(ctx).
		Table("media m").
		Preload("MediaQualities").
		Preload("Streams").
		Preload("Format").
		Where("status = ?", status).
		Order("created_at").
		Find(&media).Error; err != nil {
		return nil, err
	}

	return media, nil
}

func (r *MediaRepository) GetUserMediaById(
	ctx context.Context,
	userId uuid.UUID,
	mediaId uuid.UUID,
) (*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserMediaById").
			Observe(time.Since(t).Seconds())
	}()

	var media models.Media
	if err := r.db.WithContext(ctx).
		Preload("Parts", func(db *gorm.DB) *gorm.DB {
			return db.Order("index")
		}).
		Preload("Streams").
		Preload("Format").
		Preload("Watermark").
		Preload("Watermark.Watermark").
		Preload("MediaQualities").
		Preload("PlayerTheme").
		Preload("Captions", "media_captions.status = ?", models.DoneStatus).
		Preload("Chapters").
		Preload("MediaFiles").
		Preload("MediaFiles.File").
		Preload("MediaThumbnail").
		Preload("MediaThumbnail.Thumbnail").
		Preload("MediaThumbnail.Thumbnail.Resolutions").
		Preload("MediaThumbnail.Thumbnail.File").
		Where("user_id = ? AND id = ?", userId, mediaId).
		First(&media).Error; err != nil {
		return nil, err
	}

	return &media, nil
}

func (r *MediaRepository) GetAllUserMedias(
	ctx context.Context,
	userId uuid.UUID,
) ([]*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetAllUserMedias").
			Observe(time.Since(t).Seconds())
	}()

	var media []*models.Media
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userId).
		Find(&media).Error; err != nil {
		return nil, err
	}

	return media, nil
}

func (r *MediaRepository) GetUserMedias(
	ctx context.Context,
	userId uuid.UUID,
	input models.GetMediaListInput,
) ([]*models.Media, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserMedias").
			Observe(time.Since(t).Seconds())
	}()

	var media []*models.Media
	var total int64
	query := r.db.WithContext(ctx).
		Model(models.Media{}).
		Preload("MediaQualities").
		Preload("Format").
		Preload("MediaThumbnail").
		Where("user_id = ?", userId).
		Order(fmt.Sprintf("%s %s", input.SortBy, input.OrderBy))

	if len(input.Status) > 0 {
		query = query.Where("status IN ?", input.Status)
	} else {
		query = query.Where("status != ? AND status != ?", models.DeletedStatus, models.HiddenStatus)
	}

	if input.Type != "" {
		query = query.Where("type = ?", input.Type)
	}

	if input.Search != "" {
		query = query.Where(
			"title ilike ?",
			fmt.Sprintf("%%%s%%", input.Search),
		)
	}

	if len(input.Metadata) > 0 {
		for _, data := range input.Metadata {
			query = query.Where("metadata->>? = ?", data.Key, data.Value)
		}
	}

	if len(input.Tags) > 0 {
		query = query.Where(
			"tags ilike ?",
			fmt.Sprintf("%%%s%%", strings.Join(input.Tags, ",")),
		)
	}

	if err := query.
		Count(&total).
		Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.Media{}, 0, nil
	}

	if err := query.
		Offset(input.Offset).
		Limit(input.Limit).
		Find(&media).
		Error; err != nil {
		return nil, 0, err
	}

	return media, total, nil
}

func (r *MediaRepository) UpdateMedia(
	ctx context.Context,
	media *models.Media,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateMedia").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Save(media).Error
}

func (r *MediaRepository) GetDoneMedias(
	ctx context.Context,
) ([]*models.Media, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateMediasStatus").
			Observe(time.Since(t).Seconds())
	}()

	var media []*models.Media

	if err := r.db.WithContext(ctx).
		Model(models.Media{}).
		Table("media m").
		Where("m.status = ?", models.TranscodingStatus).
		Where(
			"EXISTS (SELECT 1 FROM media_qualities mq WHERE mq.media_id = m.id HAVING BOOL_AND(mq.status in (?, ?)))",
			models.DoneStatus,
			models.FailStatus,
		).
		Preload("MediaQualities").
		Preload("Streams").
		Find(&media).Error; err != nil {
		return nil, err
	}

	return media, nil
}

func (r *MediaRepository) UpdateMediaStatusById(
	ctx context.Context,
	mediaId uuid.UUID,
	status string,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateMediaStatusById").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(models.Media{}).
		Where("id = ?", mediaId).
		Update("status", status).Error
}

func (r *MediaRepository) UpdateMediaViewById(
	ctx context.Context,
	mediaId uuid.UUID,
	view int64,
	watchTime float64,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateMediaViewById").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Model(models.Media{}).
		Where("id = ?", mediaId).
		Updates(map[string]any{
			"view":       gorm.Expr("view + ?", view),
			"watch_time": gorm.Expr("watch_time + ?", watchTime),
		}).Error
}

func (r *MediaRepository) DeleteMediaById(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteMediaById").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(models.Media{}, "id = ?", mediaId).Error
}

func (r *MediaRepository) DeleteMediaThumbnail(
	ctx context.Context,
	mediaId, thumbnailId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteMediaThumbnail").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Select(clause.Associations).
		Where("media_id = ? AND thumbnail_id = ?", mediaId, thumbnailId).
		Delete(&models.MediaThumbnail{}).Error
}

func (r *MediaRepository) DeleteMediaFiles(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteMediaFile").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Table("media_files").
		Where("media_id = ?", mediaId).
		Delete(&models.MediaFile{}).Error
}

func (r *MediaRepository) DeleteInactiveMediaPlayerTheme(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteInactiveMediaPlayerTheme").
			Observe(time.Since(t).Seconds())
	}()
	if err := r.db.WithContext(ctx).
		Model(models.Media{}).
		Where("id = ?", mediaId).
		Updates(map[string]any{"player_theme_id": nil}).Error; err != nil {
		return err
	}
	return nil
}
