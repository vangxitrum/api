package models

import (
	"context"
	"time"

	"github.com/google/uuid"
)

var RtmpUrl string

func InitLiveStreamKey(rtmpUrl string) {
	RtmpUrl = rtmpUrl
}

type LiveStreamKeyRepository interface {
	CreateLiveStreamKey(context.Context, *LiveStreamKey) error

	GetLiveStreamKeyById(context.Context, uuid.UUID) (*LiveStreamKey, error)
	GetByLiveStreamKey(context.Context, uuid.UUID) (*LiveStreamKey, error)
	GetLiveStreamKeys(
		context.Context,
		GetLiveStreamKeysFilter,
		uuid.UUID,
	) ([]*LiveStreamKey, int64, error)
	GetLiveStreamKeyByUserId(context.Context, uuid.UUID) (*LiveStreamKey, error)
	GetLiveStreamKeyByStreamKey(context.Context, uuid.UUID) (*LiveStreamKey, error)
	GetLiveStreamByUserIdAndName(context.Context, uuid.UUID, string) (*LiveStreamKey, error)

	DeleteUserLiveStreamKey(context.Context, uuid.UUID, uuid.UUID) error
	DeleteAllUserLivestreamKey(context.Context, uuid.UUID) error

	UpdateLiveStreamKey(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		UpdateLiveStreamKeyInput,
	) (*LiveStreamKey, error)
	Update(context.Context, *LiveStreamKey) error
}

type LiveStreamKey struct {
	Id                 uuid.UUID          `json:"id"                gorm:"type:uuid;primaryKey"`
	UserId             uuid.UUID          `json:"user_id"           gorm:"type:uuid"`
	Name               string             `json:"name"`
	Save               bool               `json:"save"`
	Type               string             `json:"type"`
	StreamKey          uuid.UUID          `json:"stream_key"        gorm:"unique;type:uuid"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	LiveStreamMedias   []*LiveStreamMedia `json:"live_stream_media" gorm:"foreignKey:LiveStreamKeyId;references:Id"`
	TotalSaveMedia     int64              `json:"-"                 gorm:"-"`
	TotalLiveStreaming int64              `json:"-"                 gorm:"-"`
}
type CreateLiveStreamKeyInput struct {
	Name string `json:"name"`
	Save bool   `json:"save"`
	Type string `json:"type"`
}
type UpdateLiveStreamKeyInput struct {
	Name string `json:"name"`
	Save bool   `json:"save"`
}

type GetLiveStreamKeysResponse struct {
	LiveStreamKeys []*LiveStreamKeyResponse `json:"live_stream_keys"`
	Total          int64                    `json:"total"`
}

var LiveStreamKeysSortByMap = map[string]bool{
	"created_at": true,
	"name":       true,
}

func NewLiveStreamKey(userId uuid.UUID, input CreateLiveStreamKeyInput) *LiveStreamKey {
	return &LiveStreamKey{
		Id:        uuid.New(),
		UserId:    userId,
		Name:      input.Name,
		Save:      input.Save,
		Type:      input.Type,
		StreamKey: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

type LiveStreamKeyResponse struct {
	Id                 uuid.UUID `json:"id"`
	UserId             uuid.UUID `json:"user_id"`
	Name               string    `json:"name"`
	Save               bool      `json:"save"`
	Type               string    `json:"type"`
	StreamKey          uuid.UUID `json:"stream_key"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	RtmpUrl            string    `json:"rtmp_url"`
	TotalSaveMedia     int64     `json:"total_save_media"`
	TotalLiveStreaming int64     `json:"total_live_streaming"`
}

func ConvertLiveStreamKeyToResponse(liveStreamKey *LiveStreamKey) *LiveStreamKeyResponse {
	return &LiveStreamKeyResponse{
		Id:                 liveStreamKey.Id,
		UserId:             liveStreamKey.UserId,
		Name:               liveStreamKey.Name,
		Save:               liveStreamKey.Save,
		Type:               liveStreamKey.Type,
		StreamKey:          liveStreamKey.StreamKey,
		CreatedAt:          liveStreamKey.CreatedAt,
		UpdatedAt:          liveStreamKey.UpdatedAt,
		RtmpUrl:            RtmpUrl,
		TotalSaveMedia:     liveStreamKey.TotalSaveMedia,
		TotalLiveStreaming: liveStreamKey.TotalLiveStreaming,
	}
}

func ConvertListLiveStreamKeyToResponse(
	listLiveStreamKey []*LiveStreamKey,
) []*LiveStreamKeyResponse {
	response := make([]*LiveStreamKeyResponse, len(listLiveStreamKey))
	for i, key := range listLiveStreamKey {
		response[i] = ConvertLiveStreamKeyToResponse(key)
	}
	return response
}
