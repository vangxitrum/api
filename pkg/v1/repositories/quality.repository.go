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

type QualityRepository struct {
	db *gorm.DB
}

func MustNewQualityRepository(
	db *gorm.DB,
	init bool,
) models.QualityRepository {
	if init {
		if err := db.AutoMigrate(
			&models.MediaQuality{},
			&models.MediaQualityFile{},
		); err != nil {
			panic(err)
		}
	}

	return &QualityRepository{
		db: db,
	}
}

func (r *QualityRepository) Create(
	ctx context.Context,
	quality *models.MediaQuality,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetFormatByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(quality).Error
}

func (r *QualityRepository) CreateFile(
	ctx context.Context,
	file *models.MediaQualityFile,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CreateFile").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Create(file).Error
}

func (r *QualityRepository) CountQualitiesByMediaIdAndStatus(
	ctx context.Context,
	mediaId uuid.UUID,
	status string,
) (int, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("CountQualitiesByMediaIdAndStatus").
			Observe(time.Since(t).Seconds())
	}()

	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.MediaQuality{}).
		Preload("Files").
		Preload("Files.File").
		Where("media_id = ? AND status = ?", mediaId, status).
		Count(&count).Error; err != nil {
		return 0, nil
	}

	return int(count), nil
}

func (r *QualityRepository) GetQualityById(
	ctx context.Context,
	id uuid.UUID,
) (*models.MediaQuality, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetQualityById").
			Observe(time.Since(t).Seconds())
	}()

	var quality models.MediaQuality
	if err := r.db.WithContext(ctx).
		Preload("Media").
		Preload("Files").
		Preload("Files.File").
		Where("id = ?", id).
		First(&quality).Error; err != nil {
		return nil, err
	}

	return &quality, nil
}

func (r *QualityRepository) GetMp4QualityByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) (*models.MediaQuality, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetMp4QualityByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	var quality models.MediaQuality
	if err := r.db.WithContext(ctx).
		Preload("Files").
		Preload("Files.File").
		Where("media_id = ? AND type = ?", mediaId, models.Mp4QualityType).
		First(&quality).Error; err != nil {
		return nil, err
	}

	return &quality, nil
}

func (r *QualityRepository) GetQualitiesByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) ([]*models.MediaQuality, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetQualitiesByMediaId").
			Observe(time.Since(t).Seconds())
	}()
	var qualities []*models.MediaQuality
	if err := r.db.WithContext(ctx).
		Table("media_qualities").
		Where("media_id = ?", mediaId).
		Find(&qualities).Error; err != nil {
		return nil, err
	}

	return qualities, nil
}

func (r *QualityRepository) GetQualityByPlaylistId(
	ctx context.Context,
	playlistId string,
) (*models.MediaQuality, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetQualityByPlaylistId").
			Observe(time.Since(t).Seconds())
	}()

	var quality models.MediaQuality
	if err := r.db.WithContext(ctx).
		Table("media_qualities").
		Preload("Media").
		Preload("Media.Format").
		Preload("Media.Streams").
		Preload("Media.MediaThumbnail").
		Preload("Files").
		Where("video_playlist_id = ? or audio_playlist_id = ?", playlistId, playlistId).
		First(&quality).Error; err != nil {
		return nil, err
	}

	return &quality, nil
}

func (r *QualityRepository) UpdateQuality(
	ctx context.Context,
	quality *models.MediaQuality,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateQuality").
			Observe(time.Since(t).Seconds())
	}()

	if err := r.db.WithContext(ctx).
		Updates(quality).Error; err != nil {
		return err
	}

	if quality.VideoConfig == nil {
		if err := r.db.WithContext(ctx).
			Model(&models.MediaQuality{}).
			Where("id = ?", quality.Id).
			Updates(map[string]any{
				"media_config_bitrate": nil,
				"media_config_codec":   nil,
				"media_config_width":   nil,
				"media_config_height":  nil,
				"media_config_index":   nil,
			}).Error; err != nil {
			return err
		}
	}

	if quality.AudioConfig == nil {
		if err := r.db.WithContext(ctx).
			Model(&models.MediaQuality{}).
			Where("id = ?", quality.Id).
			Updates(map[string]any{
				"audio_config_bitrate":     nil,
				"audio_config_codec":       nil,
				"audio_config_sample_rate": nil,
				"audio_config_channels":    nil,
				"audio_config_index":       nil,
				"audio_config_language":    nil,
			}).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *QualityRepository) UpdateQualityAudioConfig(
	ctx context.Context,
	qualityId uuid.UUID,
	audioConfig *models.AudioConfig,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateQualityAudioConfig").
			Observe(time.Since(t).Seconds())
	}()

	if audioConfig != nil {
		return r.db.WithContext(ctx).
			Model(&models.MediaQuality{}).
			Where("id = ?", qualityId).
			Updates(map[string]any{
				"audio_config_bitrate":     audioConfig.Bitrate,
				"audio_config_codec":       audioConfig.Codec,
				"audio_config_sample_rate": audioConfig.SampleRate,
				"audio_config_channels":    audioConfig.Channels,
				"audio_config_index":       audioConfig.Index,
				"audio_config_language":    audioConfig.Language,
			}).Error
	}

	return r.db.WithContext(ctx).
		Model(&models.MediaQuality{}).
		Where("id = ?", qualityId).
		Updates(map[string]any{
			"audio_config_bitrate":     nil,
			"audio_config_codec":       nil,
			"audio_config_sample_rate": nil,
			"audio_config_channels":    nil,
			"audio_config_index":       nil,
			"audio_config_language":    nil,
		}).Error
}

func (r *QualityRepository) DeleteQualityById(
	ctx context.Context,
	id uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteQualityById").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Select(clause.Associations).
		Delete(&models.MediaQuality{Id: id}).Error
}

func (r *QualityRepository) DeleteQualitiesByMediaId(
	ctx context.Context,
	mediaId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteQualitiesByMediaId").
			Observe(time.Since(t).Seconds())
	}()

	return r.db.WithContext(ctx).
		Delete(&models.MediaQuality{}, "media_id = ?", mediaId).Error
}
