package repositories

import (
	"context"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LiveStreamStatisticRepository struct {
	db *gorm.DB
}

func NewLiveStreamStatisticRepository(db *gorm.DB, init bool) *LiveStreamStatisticRepository {
	if init {
		db.AutoMigrate(&models.LiveStreamStatistic{})
	}
	return &LiveStreamStatisticRepository{
		db: db,
	}
}

func (l *LiveStreamStatisticRepository) GetLiveStreamStatisticByStreamMediaId(ctx context.Context, mediaId uuid.UUID) (*models.LiveStreamStatisticResp, error) {
	var liveStreamStatistic models.LiveStreamStatisticResp
	if err := l.db.WithContext(ctx).Model(&models.LiveStreamStatistic{}).Select(`
	live_stream_statistics.id, 
	live_stream_statistics.live_stream_media_id, 
	live_stream_statistics.fps_in, 
	live_stream_statistics.fps_out, 
	live_stream_statistics.bitrate_in, 
	live_stream_statistics.bitrate_out,
	live_stream_statistics.data_transferred,
	live_stream_media.current_view,
	live_stream_media.total_view
	`).
		Joins("Join live_stream_media on live_stream_media.id = live_stream_statistics.live_stream_media_id").
		Where("live_stream_media_id = ?", mediaId).
		Scan(&liveStreamStatistic).Error; err != nil {
		return nil, err
	}
	return &liveStreamStatistic, nil
}
