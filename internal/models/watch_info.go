package models

import (
	"time"

	"github.com/google/uuid"
)

var (
	WatchType      = "watch"
	ImpressionType = "impression"
)

type WatchInfo struct {
	Id             uuid.UUID `json:"id"               gorm:"id;type:uuid;primaryKey;index:idx_filter_index,priority:5"`
	SessionMediaId uuid.UUID `json:"session_media_id" gorm:"session_media_id;type:uuid;index:idx_watch_info_session_media_id;index:idx_filter_index,priority:4"`
	UserId         uuid.UUID `json:"user_id"          gorm:"user_id;index:idx_filter_index,priority:1;type:uuid"`
	Type           string    `json:"type"`
	Paused         bool      `json:"paused"`
	MediaWidth     int       `json:"media_width"      gorm:"media_width"`
	MediaHeight    int       `json:"media_height"     gorm:"media_height"`
	Retention      float64   `json:"retention"`
	WatchTime      float64   `json:"watch_time"       gorm:"watch_time;index:idx_filter_index,priority:3"`
	MediaAt        float64   `json:"media_at"`
	CreatedAt      time.Time `json:"created_at"       gorm:"created_at;index:idx_filter_index,priority:2"`
}

func NewWatchInfo(
	sessionMediaId, userId uuid.UUID,
	paused bool,
	mediaWidth, mediaHeight int,
	watchTime, mediaAt, retention float64,
	watchInfoType string,
) *WatchInfo {
	return &WatchInfo{
		Id:             uuid.New(),
		SessionMediaId: sessionMediaId,
		UserId:         userId,
		Paused:         paused,
		MediaWidth:     mediaWidth,
		MediaHeight:    mediaHeight,
		WatchTime:      watchTime,
		Retention:      retention,
		MediaAt:        mediaAt,
		Type:           watchInfoType,
		CreatedAt:      time.Now(),
	}
}

type CreateWatchInfoInput struct {
	MediaType string
	SessionId uuid.UUID
	MediaId   uuid.UUID
	Data      []*WatchInfoItem
	Type      string
}

type WatchInfoItem struct {
	Paused      bool    `json:"paused"`
	MediaWidth  int     `json:"media_width"`
	MediaHeight int     `json:"media_height"`
	MediaAt     float64 `json:"media_at"`
}

type Metrics struct {
	TotalUser                 int64   `json:"users_total"`
	TotalUsersTopUp           float64 `json:"total_users_top_up"`
	TotalUsersCharge          float64 `json:"total_users_charge"`
	TotalActiveUsers          int64   `json:"total_active_users"`
	TotalLeaveUsers           int64   `json:"total_leave_users"`
	TotalStorageFee           float64 `json:"total_storage_fee"`
	TotalStorageCapacity      int64   `json:"total_storage_capacity"`
	TotalStorageChargedUser   float64 `json:"total_storage_charged_user"`
	TotalDeliveryFee          float64 `json:"total_delivery_fee"`
	TotalDeliveryCapacity     int64   `json:"total_delivery_capacity"`
	TotalDeliveryChargedUser  float64 `json:"total_delivery_charged_user"`
	TotalTranscodeFee         float64 `json:"total_transcode_fee"`
	TotalTranscodeTime        float64 `json:"total_transcode_time"`
	TotalTranscodeChargedUser float64 `json:"total_transcode_charged_user"`
	TotalVideosCount          int64   `json:"total_videos_count"`
	TotalVideoInteractions    float64 `json:"total_video_interactions"`
	TotalTranscodeFail        int64   `json:"total_transcode_fail"`
	TotalLivestreamHours      float64 `json:"total_livestream_hours"`
	TotalLivestream           int64   `json:"total_livestream"`
}
type StatisticData struct {
	Metrics     *Metrics `json:"metrics"`
	From        int64    `json:"from"`
	To          int64    `json:"to"`
	InfraSource string   `json:"infra_source"`
}
