package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
)

type liveStreamRepository struct {
	db *gorm.DB
}

func NewLiveStreamRepository(db *gorm.DB, init bool) models.LiveStreamMediaRepository {
	if init {
		db.AutoMigrate(
			&models.LiveStreamMedia{},
		)
	}
	return &liveStreamRepository{
		db: db,
	}
}

func (r *liveStreamRepository) Create(
	ctx context.Context,
	liveStream *models.LiveStreamMedia,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("Create").Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(liveStream).Error; err != nil {
		return err
	}
	return nil
}

func (r *liveStreamRepository) GetById(
	ctx context.Context,
	id uuid.UUID,
) (*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetById").Observe(time.Since(t).Seconds())
	}()

	var liveStream models.LiveStreamMedia
	if err := r.db.WithContext(ctx).Where("id = ?", id).
		Preload("Media").
		Preload("Media.MediaQualities").
		Preload("Media.PlayerTheme").
		First(&liveStream).Error; err != nil {
		return nil, err
	}
	return &liveStream, nil
}

func (r *liveStreamRepository) GetCreated(
	ctx context.Context,
	streamKeyId, userId uuid.UUID,
) (*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCreated").Observe(time.Since(t).Seconds())
	}()

	var liveStream models.LiveStreamMedia
	if err := r.db.WithContext(ctx).
		Where("live_stream_key_id = ? AND user_id = ? AND status = ?", streamKeyId, userId, models.LiveStreamStatusCreated).
		Order("created_at ASC").
		First(&liveStream).Error; err != nil {
		return nil, err
	}
	return &liveStream, nil
}

func (r *liveStreamRepository) GetUserLiveStreamMediaByIdAndLiveStreamKeyId(
	ctx context.Context,
	userId uuid.UUID,
	liveStreamKeyId uuid.UUID,
	liveStreamMediaId uuid.UUID,
) (*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserLiveStreamMediaByIdAndLiveStreamKeyId").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMedia models.LiveStreamMedia
	if err := r.db.WithContext(ctx).
		Where("live_stream_key_id = ? AND  id= ? AND user_id = ?", liveStreamMediaId, liveStreamKeyId, userId).
		First(&liveStreamMedia).Error; err != nil {
		return nil, err
	}
	return &liveStreamMedia, nil
}

func (r *liveStreamRepository) GetLiveStreamMediaByConnectionId(
	ctx context.Context,
	connId string,
) (*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamMediaByConnectionId").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMedia models.LiveStreamMedia
	if err := r.db.WithContext(ctx).Where("connection_id = ?", connId).First(&liveStreamMedia).Error; err != nil {
		return nil, err
	}
	return &liveStreamMedia, nil
}

func (r *liveStreamRepository) GetLiveStreamByStreamKey(
	ctx context.Context,
	streamKey uuid.UUID,
) (*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamByStreamKey").
			Observe(time.Since(t).Seconds())
	}()

	var liveStream models.LiveStreamMedia
	if err := r.db.WithContext(ctx).Where("stream_key = ?", streamKey).First(&liveStream).Error; err != nil {
		return nil, err
	}
	return &liveStream, nil
}

func (r *liveStreamRepository) GetLiveStreamsByUserId(
	ctx context.Context,
	userId uuid.UUID,
) ([]*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamsByUserId").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreams []*models.LiveStreamMedia
	if err := r.db.WithContext(ctx).Where("user_id = ?", userId).Find(&liveStreams).Error; err != nil {
		return nil, err
	}
	return liveStreams, nil
}

func (r *liveStreamRepository) GetLiveStreamMedias(
	ctx context.Context,
	userId uuid.UUID,
	filter models.GetLiveStreamMediasFilter,
) ([]*models.LiveStreamMedia, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamMedias").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMedias []*models.LiveStreamMedia
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.LiveStreamMedia{}).
		Joins("LEFT JOIN media ON live_stream_media.media_id = media.id").
		Where("live_stream_media.live_stream_key_id = ? AND live_stream_media.user_id = ? AND live_stream_media.save = true AND live_stream_media.status = ?",
			filter.LiveStreamKeyId,
			userId,
			models.LiveStreamStatusEnd,
		).
		Where("media.status != ?", models.DeletedStatus)

	if filter.Search != "" {
		query = query.Where("live_stream_media.title ILIKE ?", "%"+filter.Search+"%")
	}

	if filter.SortBy != "" {
		query = query.Order(filter.SortBy + " " + filter.OrderBy)
	}

	if filter.Status != "" {
		query = query.Where("live_stream_media.status = ?", filter.Status)
	} else {
		query = query.Where("live_stream_media.status != ?", models.DeletedStatus)
	}

	if filter.MediaStatus != "" {
		query = query.Where("media.status = ?", filter.MediaStatus)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Offset(filter.Offset).Limit(filter.Limit).Preload("Media").Preload("Media.MediaQualities").Preload("Media.Format").Find(&liveStreamMedias).Error; err != nil {
		return nil, 0, err
	}

	return liveStreamMedias, total, nil
}

func (r *liveStreamRepository) GetUserLiveStreamMediaByLiveStreamKeyId(
	ctx context.Context,
	userId uuid.UUID,
	liveStreamKeyId uuid.UUID,
	payload models.GetStreamingsFilter,
) ([]*models.LiveStreamMedia, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetUserLiveStreamMediaByLiveStreamKeyId").
			Observe(time.Since(t).Seconds())
	}()
	var liveStreamMedias []*models.LiveStreamMedia
	var total int64

	query := r.db.WithContext(ctx).
		Model(&models.LiveStreamMedia{}).
		Where("live_stream_key_id = ? AND user_id = ? AND (status = ? OR status = ?)",
			liveStreamKeyId,
			userId,
			models.LiveStreamStatusStreaming,
			models.LiveStreamStatusCreated,
		).
		Order("CASE WHEN status = 'streaming' THEN 0 ELSE 1 END").
		Order(fmt.Sprintf("%s %s", payload.SortBy, payload.OrderBy)).
		Preload("Media").
		Preload("Media.MediaQualities").
		Preload("Media.MediaThumbnail")

	if payload.Search != "" {
		query = query.Where("live_stream_media.title LIKE ?", "%"+payload.Search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*models.LiveStreamMedia{}, 0, nil
	}

	if err := query.
		Offset(payload.Offset).
		Limit(payload.Limit).
		Find(&liveStreamMedias).
		Error; err != nil {
		return nil, 0, err
	}

	return liveStreamMedias, total, nil
}

func (r *liveStreamRepository) GetAllLiveStreamMedia(
	ctx context.Context,
	userId uuid.UUID,
	LiveStreamKeyId uuid.UUID,
	status string,
) ([]*models.LiveStreamMedia, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetAllLiveStreamMedia").
			Observe(time.Since(t).Seconds())
	}()
	var liveStreamMedias []*models.LiveStreamMedia
	query := r.db.WithContext(ctx).
		Where("live_stream_key_id = ? AND user_id = ? AND status = ?",
			LiveStreamKeyId,
			userId,
			status,
		)

	if err := query.Find(&liveStreamMedias).Error; err != nil {
		return nil, 0, err
	}

	return liveStreamMedias, int64(len(liveStreamMedias)), nil
}

func (r *liveStreamRepository) GetLiveStreamMediaStreaming(
	ctx context.Context,
	userId uuid.UUID,
	liveStreamKeyId, streamId uuid.UUID,
) (*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamMediaStreaming").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMedia models.LiveStreamMedia
	if err := r.db.WithContext(ctx).
		Where("id = ? AND live_stream_key_id = ? AND user_id = ? AND (status = ? OR status = ?)",
			streamId,
			liveStreamKeyId,
			userId,
			models.LiveStreamStatusStreaming,
			models.LiveStreamStatusCreated,
		).
		Preload("Media").
		Preload("Media.MediaQualities").
		First(&liveStreamMedia).Error; err != nil {
		return nil, err
	}
	return &liveStreamMedia, nil
}

func (r *liveStreamRepository) GetNotSavedLiveStreamMedias(
	ctx context.Context,
) ([]*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetNotSavedLiveStreamMedias").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMedias []*models.LiveStreamMedia
	if err := r.db.WithContext(ctx).
		Where("save = false AND status = ? ", models.LiveStreamStatusEnd).
		Find(&liveStreamMedias).Error; err != nil {
		return nil, err
	}

	return liveStreamMedias, nil
}

func (r *liveStreamRepository) GetSavedLiveStreamMedias(
	ctx context.Context,
	userId uuid.UUID,
	liveStreamKeyId uuid.UUID,
) ([]*models.LiveStreamMedia, int64, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetSavedLiveStreamMedias").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMedias []*models.LiveStreamMedia
	query := r.db.WithContext(ctx).
		Joins("JOIN media ON live_stream_media.media_id = media.id").
		Where("live_stream_media.live_stream_key_id = ? AND live_stream_media.user_id = ? AND live_stream_media.save = true AND live_stream_media.status = ?",
			liveStreamKeyId,
			userId,
			models.LiveStreamStatusEnd,
		).
		Where("media.status != ?", models.DeletedStatus)

	if err := query.Find(&liveStreamMedias).Error; err != nil {
		return nil, 0, err
	}

	return liveStreamMedias, int64(len(liveStreamMedias)), nil
}

func (r *liveStreamRepository) GetLiveStreamByUserIdAndTitle(
	ctx context.Context,
	userId uuid.UUID,
	title string,
) (*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamByUserIdAndTitle").
			Observe(time.Since(t).Seconds())
	}()

	var liveStream models.LiveStreamMedia
	if err := r.db.WithContext(ctx).Where("user_id = ? AND title = ?", userId, title).First(&liveStream).Error; err != nil {
		return nil, err
	}
	return &liveStream, nil
}

func (r *liveStreamRepository) UpdateLiveStreamName(
	ctx context.Context,
	id uuid.UUID,
	newName string,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateLiveStreamName").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Model(&models.LiveStreamMedia{}).Where("id = ?", id).Updates(models.LiveStreamMedia{Title: newName, UpdatedAt: time.Now().UTC()}).Error; err != nil {
		return err
	}
	return nil
}

func (r *liveStreamRepository) EndLiveStream(ctx context.Context, id uuid.UUID) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("EndLiveStream").Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).Model(&models.LiveStreamMedia{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"updated_at": time.Now().UTC(),
			"status":     models.LiveStreamStatusEnd,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *liveStreamRepository) GetListLiveStreamingMedias(
	ctx context.Context,
) ([]*models.LiveStreamMedia, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetListLiveStreamingMedias").
			Observe(time.Since(t).Seconds())
	}()

	var media []*models.LiveStreamMedia
	if err := r.db.WithContext(ctx).Where("status = ?", models.LiveStreamStatusStreaming).Find(&media).Error; err != nil {
		return nil, err
	}
	return media, nil
}

func (r *liveStreamRepository) GetCdnFilesByLiveStreamMedia(
	ctx context.Context,
	liveStreamId uuid.UUID,
) ([]*models.CdnFile, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetCdnFilesByLiveStreamMedia").
			Observe(time.Since(t).Seconds())
	}()

	var cdnFiles []*models.CdnFile
	if err := r.db.WithContext(ctx).Where("belongs_to_id = ?", liveStreamId).Find(&cdnFiles).Error; err != nil {
		return nil, err
	}
	return cdnFiles, nil
}

func (r *liveStreamRepository) UpdateLiveStreamMedia(
	ctx context.Context,
	liveStreamMedia *models.LiveStreamMedia,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateLiveStreamMedia").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Save(liveStreamMedia).Error; err != nil {
		return err
	}
	return nil
}

func (z *liveStreamRepository) UpdateEndLiveStreamMedia(ctx context.Context) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateEndLiveStreamMedia").
			Observe(time.Since(t).Seconds())
	}()

	oneMinuteAgo := time.Now().UTC().Add(-1 * time.Minute)

	if err := z.db.WithContext(ctx).
		Model(&models.LiveStreamMedia{}).
		Where("status = ? AND updated_at < ?", models.LiveStreamStatusStreaming, oneMinuteAgo).
		Update("status", models.LiveStreamStatusEnd).
		Error; err != nil {
		return err
	}

	return nil
}

func (r *liveStreamRepository) DeleteLiveStreamMedia(
	ctx context.Context,
	id, userID uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteLiveStreamMedia").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMedia models.LiveStreamMedia
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&liveStreamMedia).Error; err != nil {
		return err
	}

	if err := r.db.WithContext(ctx).Delete(&liveStreamMedia).Error; err != nil {
		return err
	}

	return nil
}

func (r *liveStreamRepository) DeleteLiveStreamMedias(
	ctx context.Context,
	liveStreamKeyId uuid.UUID,
	userId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteLiveStreamMedias").
			Observe(time.Since(t).Seconds())
	}()

	result := r.db.WithContext(ctx).
		Where("live_stream_key_id = ? AND user_id = ?", liveStreamKeyId, userId).
		Delete(&models.LiveStreamMedia{})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *liveStreamRepository) DeleteUserLivestreamMedia(
	ctx context.Context,
	userId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteUserLivestreamMedia").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).Where("user_id = ?", userId).Delete(&models.LiveStreamMedia{}).Error; err != nil {
		return err
	}

	return nil
}

func (r *liveStreamRepository) UpdateLiveStreamView(
	ctx context.Context,
	timeRange time.Time,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateLiveStreamView").
			Observe(time.Since(t).Seconds())
	}()

	selectQuery := fmt.Sprintf(`
	WITH latest_watch AS (
		SELECT
			wi.session_media_id,
			lv.id AS live_stream_media_id,
			wi.created_at,
			wi.watch_time,
			ROW_NUMBER() OVER (PARTITION BY wi.session_media_id ORDER BY wi.created_at DESC) AS rn
		FROM live_stream_media lv
		INNER JOIN session_media sm ON lv.id = sm.media_id
		INNER JOIN watch_infos wi ON sm.id = wi.session_media_id
		WHERE wi.watch_time > %v
		AND lv.status = '%s'
	)
	SELECT
		live_stream_media_id,
		SUM(CASE
			WHEN created_at >= '%s'
			THEN 1
			ELSE 0
		END) AS current_view,
		SUM(1) AS total_view
	FROM latest_watch
	WHERE rn = 1
	GROUP BY live_stream_media_id`,
		models.MinWatchTime,
		models.LiveStreamStatusStreaming,
		timeRange.Format(time.RFC3339),
	)

	var analyticsResults []models.AnalyticsResult
	if err := r.db.WithContext(ctx).Raw(selectQuery).Scan(&analyticsResults).Error; err != nil {
		return err
	}

	if len(analyticsResults) == 0 {
		return nil
	}

	checkTypeQuery := `
	SELECT data_type
	FROM information_schema.columns
	WHERE table_name = 'live_stream_media' AND column_name = 'id'
	LIMIT 1
	`

	if err := r.db.WithContext(ctx).Raw(checkTypeQuery).Scan(&models.TableInfo).Error; err != nil {
		return err
	}

	updateQuery := `
	UPDATE live_stream_media
	SET
		current_view = CASE
	`

	for _, result := range analyticsResults {
		if models.TableInfo.DataType == "uuid" {
			updateQuery += fmt.Sprintf("WHEN id = '%s'::uuid THEN %d\n",
				result.LiveStreamMediaID, result.CurrentView)
		} else {
			updateQuery += fmt.Sprintf("WHEN id::text = '%s' THEN %d\n",
				result.LiveStreamMediaID, result.CurrentView)
		}
	}

	updateQuery += `
		ELSE current_view
		END,
		total_view = CASE
	`

	for _, result := range analyticsResults {
		if models.TableInfo.DataType == "uuid" {
			updateQuery += fmt.Sprintf("WHEN id = '%s'::uuid THEN %d\n",
				result.LiveStreamMediaID, result.TotalView)
		} else {
			updateQuery += fmt.Sprintf("WHEN id::text = '%s' THEN %d\n",
				result.LiveStreamMediaID, result.TotalView)
		}
	}

	updateQuery += `
		ELSE total_view
		END
	WHERE
	`

	if models.TableInfo.DataType == "uuid" {
		updateQuery += "id IN ("
		for i, result := range analyticsResults {
			if i > 0 {
				updateQuery += ", "
			}
			updateQuery += fmt.Sprintf("'%s'::uuid", result.LiveStreamMediaID)
		}
		updateQuery += ")"
	} else {
		updateQuery += "id::text IN ("
		for i, result := range analyticsResults {
			if i > 0 {
				updateQuery += ", "
			}
			updateQuery += fmt.Sprintf("'%s'", result.LiveStreamMediaID)
		}
		updateQuery += ")"
	}

	if err := r.db.WithContext(ctx).Exec(updateQuery).Error; err != nil {
		return err
	}

	return nil
}
