package models

import (
	"context"

	"github.com/google/uuid"
)

type LiveStreamStatistic struct {
	Id                uuid.UUID `json:"id"                   gorm:"primaryKey;type:uuid"`
	LiveStreamMediaId uuid.UUID `json:"live_stream_media_id" gorm:"UniqueIndex;type:uuid"`
	FpsIn             int16     `json:"fps_in"`
	FpsOut            int16     `json:"fps_out"`
	BitrateIn         float64   `json:"bitrate_in"`
	BitrateOut        float64   `json:"bitrate_out"`
	DataTransferred   float64   `json:"data_transferred"`
} //	@name	LiveStreamStatistic

type LiveStreamStatisticResp struct {
	Id                uuid.UUID `json:"id"                   gorm:"primaryKey;type:uuid"`
	LiveStreamMediaId uuid.UUID `json:"live_stream_media_id" gorm:"UniqueIndex;type:uuid"`
	FpsIn             int16     `json:"fps_in"`
	FpsOut            int16     `json:"fps_out"`
	BitrateIn         float64   `json:"bitrate_in"`
	BitrateOut        float64   `json:"bitrate_out"`
	DataTransferred   float64   `json:"data_transferred"`
	CurrentView       int64     `json:"current_view"`
	TotalView         int64     `json:"total_view"`
} //	@name	LiveStreamStatisticResp

type LiveStreamStatisticRepository interface {
	GetLiveStreamStatisticByStreamMediaId(
		ctx context.Context,
		streamMediaId uuid.UUID,
	) (*LiveStreamStatisticResp, error)
}
