package repositories

import (
	"context"
	"time"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/metrics"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type liveStreamMulticastRepository struct {
	db *gorm.DB
}

func NewLiveStreamMulticastRepository(db *gorm.DB, init bool) models.LiveStreamMulticastRepository {
	if init {
		db.AutoMigrate(&models.LiveStreamMulticast{})
	}
	return &liveStreamMulticastRepository{
		db: db,
	}
}

func (l *liveStreamMulticastRepository) UpsertLiveStreamMulticast(
	ctx context.Context,
	liveStreamMulticast *models.LiveStreamMulticast,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpsertLiveStreamMulticast").
			Observe(time.Since(t).Seconds())
	}()

	if err := l.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "live_stream_key_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"live_stream_multicast_urls"}),
	}).Create(&liveStreamMulticast).Error; err != nil {
		return err
	}

	return nil
}

func (l *liveStreamMulticastRepository) DeleteLiveStreamMulticast(
	c context.Context,
	streamKeyId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteLiveStreamMulticast").
			Observe(time.Since(t).Seconds())
	}()

	if err := l.db.WithContext(c).Where("live_stream_key_id = ?", streamKeyId).Delete(&models.LiveStreamMulticast{}).Error; err != nil {
		return err
	}

	return nil
}

func (l *liveStreamMulticastRepository) DeleteUserLivestreamMulticasts(
	ctx context.Context,
	userId uuid.UUID,
) error {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("DeleteUserLivestreamMulticasts").
			Observe(time.Since(t).Seconds())
	}()

	if err := l.db.WithContext(ctx).Where("user_id = ?", userId).Delete(&models.LiveStreamMulticast{}).Error; err != nil {
		return err
	}

	return nil
}

func (l *liveStreamMulticastRepository) GetLiveStreamMulticastByStreamKeyId(
	c context.Context,
	streamKeyId uuid.UUID,
) (*models.LiveStreamMulticast, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("GetLiveStreamMulticastByStreamKeyId").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMulticast models.LiveStreamMulticast
	if err := l.db.WithContext(c).Where("live_stream_key_id = ?", streamKeyId).First(&liveStreamMulticast).Error; err != nil {
		return nil, err
	}
	return &liveStreamMulticast, nil
}

func (l *liveStreamMulticastRepository) GetLiveStreamMulticastStreamingByStreamKeyId(
	c context.Context,
	streamKeyId uuid.UUID,
) (*models.LiveStreamMulticast, error) {
	t := time.Now().UTC()
	defer func() {
		metrics.DbMetricsIns.DbSum.WithLabelValues("UpdateLiveStreamMulticast").
			Observe(time.Since(t).Seconds())
	}()

	var liveStreamMulticast models.LiveStreamMulticast

	err := l.db.WithContext(c).Model(&models.LiveStreamMulticast{}).
		Joins("JOIN live_stream_media ON live_stream_media.live_stream_key_id = live_stream_multicasts.live_stream_key_id").
		Where("live_stream_media.status = ? AND live_stream_multicasts.live_stream_key_id = ?", "streaming", streamKeyId).
		First(&liveStreamMulticast).Error
	if err != nil {
		return nil, err
	}
	return &liveStreamMulticast, err
}
