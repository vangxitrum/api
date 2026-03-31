package models

import (
	"context"

	"github.com/google/uuid"
	pq "github.com/lib/pq"
)

var TableLiveStreamMulticastName = "live_stream_multicasts"

type LiveStreamMulticast struct {
	Id                      uuid.UUID      `json:"id"                         gorm:"primaryKey;type:uuid"`
	LiveStreamKeyId         uuid.UUID      `json:"live_stream_key_id"         gorm:"unique;type:uuid"`
	UserId                  uuid.UUID      `json:"user_id"                    gorm:"type:text;type:uuid"`
	LiveStreamMulticastUrls pq.StringArray `json:"live_stream_multicast_urls" gorm:"type:text[]"`
	LiveStreamKey           *LiveStreamKey `json:"-"                          gorm:"foreignKey:LiveStreamKeyId;references:Id;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
} //	@name	LiveStreamMulticast

type UpsertLiveStreamMulticastInput struct {
	MulticastUrls pq.StringArray `json:"multicast_urls"`
} //	@name	UpsertLiveStreamMulticastInput

type LiveStreamMulticastRepository interface {
	UpsertLiveStreamMulticast(context.Context, *LiveStreamMulticast) error
	GetLiveStreamMulticastByStreamKeyId(context.Context, uuid.UUID) (*LiveStreamMulticast, error)
	GetLiveStreamMulticastStreamingByStreamKeyId(
		context.Context,
		uuid.UUID,
	) (*LiveStreamMulticast, error)
	DeleteLiveStreamMulticast(context.Context, uuid.UUID) error
	DeleteUserLivestreamMulticasts(context.Context, uuid.UUID) error
}

func NewLiveStreamMulticast(
	liveStreamKeyId uuid.UUID,
	userId uuid.UUID,
	multicastUrls []string,
) *LiveStreamMulticast {
	return &LiveStreamMulticast{
		Id:                      uuid.New(),
		LiveStreamKeyId:         liveStreamKeyId,
		UserId:                  userId,
		LiveStreamMulticastUrls: pq.StringArray(multicastUrls),
	}
}
