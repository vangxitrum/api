package models

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ErrAlreadyExist = "live stream with this name already exist"
	AppName         = "live"

	LiveStreamStatusCreated   = "created"
	LiveStreamStatusStreaming = "streaming"
	LiveStreamStatusEnd       = "end"
	LiveStreamStatusUploading = "uploading"
)

var (
	LiveServerHost        string
	LiveStreamMediaHlsUrl string
)

func InitLiveStreamMedia(host string) {
	if !strings.HasSuffix(host, "/") {
		host = host + "/"
	}

	LiveServerHost = host
	LiveStreamMediaHlsUrl = fmt.Sprintf(host+"%s/index.m3u8", "%s")
}

type GetLiveStreamKeysFilter struct {
	Search  string `json:"search"   query:"search"`
	SortBy  string `json:"sort_by"  query:"sort_by"`
	OrderBy string `json:"order_by" query:"order_by"`
	Offset  int    `json:"offset"   query:"offset"`
	Limit   int    `json:"limit"    query:"limit"`
	Type    string `json:"type"     query:"type"`
}
type GetLiveStreamMediasFilter struct {
	LiveStreamKeyId uuid.UUID `json:"live_stream_key_id"`
	Search          string    `json:"search"             form:"search"`
	SortBy          string    `json:"sort_by"            form:"sort_by"`
	OrderBy         string    `json:"order_by"           form:"order_by"`
	Offset          int       `json:"offset"             form:"offset"`
	Limit           int       `json:"limit"              form:"limit"`
	Status          string    `json:"status"             form:"status"`
	MediaStatus     string    `json:"media_status"       form:"media_status"`
}

type GetStreamingsFilter struct {
	Search  string `json:"search"   query:"search"`
	SortBy  string `json:"sort_by"  query:"sort_by"`
	OrderBy string `json:"order_by" query:"order_by"`
	Offset  int    `json:"offset"   query:"offset"`
	Limit   int    `json:"limit"    query:"limit"`
}

type LiveStreamMediasResponse struct {
	LiveStreamMedias []*LiveStreamMediaResponse `json:"media"`
	Total            int64                      `json:"total"`
} //	@name	LiveStreamMediasResponse

var LiveStreamMediasSortByMap = map[string]bool{
	"created_at": true,
	"title":      true,
}

var TableInfo struct {
	DataType string `gorm:"column:data_type"`
}

type LiveStreamMediaRepository interface {
	Create(context.Context, *LiveStreamMedia) error

	GetById(context.Context, uuid.UUID) (*LiveStreamMedia, error)
	GetLiveStreamMediaByConnectionId(context.Context, string) (*LiveStreamMedia, error)
	GetCreated(context.Context, uuid.UUID, uuid.UUID) (*LiveStreamMedia, error)
	GetUserLiveStreamMediaByIdAndLiveStreamKeyId(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		uuid.UUID,
	) (*LiveStreamMedia, error)
	GetLiveStreamMedias(
		context.Context,
		uuid.UUID,
		GetLiveStreamMediasFilter,
	) ([]*LiveStreamMedia, int64, error)
	GetUserLiveStreamMediaByLiveStreamKeyId(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		GetStreamingsFilter,
	) ([]*LiveStreamMedia, int64, error)
	GetAllLiveStreamMedia(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		string,
	) ([]*LiveStreamMedia, int64, error)
	GetCdnFilesByLiveStreamMedia(context.Context, uuid.UUID) ([]*CdnFile, error)
	GetLiveStreamMediaStreaming(
		context.Context,
		uuid.UUID,
		uuid.UUID,
		uuid.UUID,
	) (*LiveStreamMedia, error)
	GetNotSavedLiveStreamMedias(context.Context) ([]*LiveStreamMedia, error)
	GetSavedLiveStreamMedias(
		context.Context,
		uuid.UUID,
		uuid.UUID,
	) ([]*LiveStreamMedia, int64, error)
	GetListLiveStreamingMedias(context.Context) ([]*LiveStreamMedia, error)

	UpdateLiveStreamName(context.Context, uuid.UUID, string) error
	UpdateLiveStreamMedia(context.Context, *LiveStreamMedia) error
	UpdateEndLiveStreamMedia(context.Context) error
	UpdateLiveStreamView(context.Context, time.Time) error

	EndLiveStream(context.Context, uuid.UUID) error
	DeleteLiveStreamMedia(context.Context, uuid.UUID, uuid.UUID) error
	DeleteLiveStreamMedias(context.Context, uuid.UUID, uuid.UUID) error
	DeleteUserLivestreamMedia(context.Context, uuid.UUID) error
}

type LiveStreamMedia struct {
	Id              uuid.UUID      `json:"id"                 gorm:"type:uuid;primaryKey"`
	LiveStreamKeyId uuid.UUID      `json:"live_stream_key_id" gorm:"type:uuid"`
	LiveStreamKey   *LiveStreamKey `json:"-"                  gorm:"foreignKey:LiveStreamKeyId;references:Id"`
	UserId          uuid.UUID      `json:"user_id"            gorm:"type:uuid"`
	ConnectionId    string         `json:"connection_id"`
	Title           string         `json:"title"`
	ThumbnailUrl    string         `json:"thumbnail_url"`
	Save            bool           `json:"save"`
	Type            string         `json:"type"`
	MediaId         uuid.UUID      `json:"media_id"           gorm:"type:uuid"`
	Media           *Media         `json:"media"              gorm:"foreignKey:MediaId;references:Id"`
	Status          string         `json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	StreamedAt      time.Time      `json:"streamed_at"`
	CurrentView     int64          `json:"current_view"`
	TotalView       int64          `json:"total_view"`
}

type LiveStreamMediaResponse struct {
	Id              uuid.UUID        `json:"id"`
	LiveStreamKeyId uuid.UUID        `json:"live_stream_key_id"`
	UserId          uuid.UUID        `json:"user_id"`
	Title           string           `json:"title"`
	Duration        int64            `json:"duration"`
	Assets          LiveStreamAssets `json:"assets"`
	Save            bool             `json:"save"`
	Type            string           `json:"type"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Status          string           `json:"status"`
	Media           *MediaObject     `json:"media"`
	Qualities       []string         `json:"qualities"`
	FrameRate       int              `json:"frame_rate"`
	AudioBitrate    int              `json:"audio_bitrate"`
	CurrentView     int64            `json:"current_view"`
	TotalView       int64            `json:"total_view"`
} //	@name	LiveStreamMediaResponse

type LiveStreamAnalytics struct {
	LiveStreamMediaId uuid.UUID `json:"live_stream_media_id"`
	CurrentView       int64     `json:"current_view"`
	TotalView         int64     `json:"total_view"`
}

type CreateStreamingRequest struct {
	Title string `json:"title"`
	Save  bool   `json:"save"`
	// Qualities of the media (default: 1080p, 720p,  360p, allow:2160p, 1440p, 1080p, 720p,  360p, 240p, 144p)
	Qualities []*QualityConfig `json:"qualities" form:"qualities"`
}

type UpdateLiveStreamMediaInput struct {
	StreamId  uuid.UUID        `json:"stream_id"`
	Title     string           `json:"title"`
	Save      bool             `json:"save"`
	Qualities []*QualityConfig `json:"qualities" form:"qualities"`
}

type LiveStreamAssets struct {
	ThumbnailUrl string `json:"thumbnail_url"`
	HlsUrl       string `json:"hls_url"`
	IFrame       string `json:"iframe"`
	PlayerUrl    string `json:"player_url"`
}

type CreateLiveStreamMediaInput struct {
	Title     string   `json:"title"`
	Save      bool     `json:"save"`
	Qualities []string `json:"qualities"`
}

type HLSMuxerItem struct {
	Path        string    `json:"path"`
	Created     time.Time `json:"created"`
	LastRequest time.Time `json:"lastRequest"`
	BytesSent   int64     `json:"bytesSent"`
}

type HLSMuxerResponse struct {
	ItemCount int            `json:"itemCount"`
	PageCount int            `json:"pageCount"`
	Items     []HLSMuxerItem `json:"items"`
}

type RTMPListConnectionResponse struct {
	ItemCount int                         `json:"itemCount"`
	PageCount int                         `json:"pageCount"`
	Items     []LiveStreamWebhookResponse `json:"items"`
}

type LiveStreamWebhookResponse struct {
	ID            string `json:"id"`
	Created       string `json:"created"`
	RemoteAddr    string `json:"remoteAddr"`
	State         string `json:"state"`
	Path          string `json:"path"`
	Query         string `json:"query"`
	BytesReceived int64  `json:"bytesReceived"`
	BytesSent     int64  `json:"bytesSent"`
	StreamKey     string `json:"streamKey"`
}

type LiveStreamRecordResponse struct {
	Name     string `json:"name"`
	Segments []struct {
		Start string `json:"start"`
	} `json:"segments"`
}
type AnalyticsResult struct {
	LiveStreamMediaID string `gorm:"column:live_stream_media_id"`
	CurrentView       int    `gorm:"column:current_view"`
	TotalView         int    `gorm:"column:total_view"`
}

func NewLiveStreamMedia(
	mediaId, streamKey, userId uuid.UUID,
	status string,
	input CreateStreamingRequest,
	typeMedia string,
) *LiveStreamMedia {
	return &LiveStreamMedia{
		Id:              uuid.New(),
		MediaId:         mediaId,
		LiveStreamKeyId: streamKey,
		UserId:          userId,
		Title:           input.Title,
		ThumbnailUrl:    "",
		Save:            input.Save,
		Type:            typeMedia,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		Status:          status,
	}
}

func NewLiveStreamMediaWithId(
	liveStreamMediaId uuid.UUID,
	mediaId, streamKey, userId uuid.UUID,
	status string,
	input CreateStreamingRequest,
	typeMedia string,
) *LiveStreamMedia {
	return &LiveStreamMedia{
		Id:              liveStreamMediaId,
		MediaId:         mediaId,
		LiveStreamKeyId: streamKey,
		UserId:          userId,
		Title:           input.Title,
		ThumbnailUrl:    "",
		Save:            input.Save,
		Type:            typeMedia,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		Status:          status,
	}
}

func ConvertLiveStreamMediasToResponse(
	liveStreamMedias []*LiveStreamMedia,
	total int64,
) *LiveStreamMediasResponse {
	lsvs := make([]*LiveStreamMediaResponse, 0)
	for _, lsv := range liveStreamMedias {
		lsvs = append(lsvs, ConvertLiveStreamMediaToResponse(lsv))
	}

	return &LiveStreamMediasResponse{
		LiveStreamMedias: lsvs,
		Total:            total,
	}
}

func BuildLiveStreamPlayerUrl(liveStreamId uuid.UUID) string {
	playerUrl := fmt.Sprintf("%s/live/%s", PlayerUrl, liveStreamId)
	return playerUrl
}

func CalculateLiveStreamDuration(liveStreamMedia *LiveStreamMedia) int64 {
	if liveStreamMedia.StreamedAt.IsZero() {
		return 0
	}

	return time.Now().UTC().Sub(liveStreamMedia.StreamedAt).Milliseconds()
}

func ConvertLiveStreamMediaToResponse(liveStreamMedia *LiveStreamMedia) *LiveStreamMediaResponse {
	var media *MediaObject
	var duration int64
	var thumbnailUrl string
	if liveStreamMedia.Media != nil {
		media = NewMediaObject(liveStreamMedia.Media)
		if media.Assets != nil {
			thumbnailUrl = fmt.Sprintf(
				AssetUrlFormat,
				BeUrl, media.Id,
			) + "/thumbnail?resolution=original"
		}
	}

	playerUrl := fmt.Sprintf("%s/live/%s", PlayerUrl, liveStreamMedia.Id)

	iFrame := fmt.Sprintf(
		`<iframe src="%s" width="100%%" height="100%%" frameborder="0" scrolling="no" allowfullscreen="true"></iframe>`,
		playerUrl,
	)

	frameRate := 0
	audioBitrate := 0

	duration = CalculateLiveStreamDuration(liveStreamMedia)
	var qualities []string
	if liveStreamMedia.Media != nil {
		qualities = make([]string, 0, len(liveStreamMedia.Media.MediaQualities))
		for _, quality := range liveStreamMedia.Media.MediaQualities {
			if (quality.VideoConfig != nil || quality.AudioConfig != nil) &&
				!slices.Contains(qualities, quality.Resolution) {
				qualities = append(qualities, quality.Resolution)
			}
		}
	}

	return &LiveStreamMediaResponse{
		Id:              liveStreamMedia.Id,
		LiveStreamKeyId: liveStreamMedia.LiveStreamKeyId,
		UserId:          liveStreamMedia.UserId,
		Title:           liveStreamMedia.Title,
		Assets: LiveStreamAssets{
			HlsUrl:       fmt.Sprintf(LiveStreamMediaHlsUrl, liveStreamMedia.Id),
			IFrame:       iFrame,
			PlayerUrl:    playerUrl,
			ThumbnailUrl: thumbnailUrl,
		},
		Save:         liveStreamMedia.Save,
		Type:         liveStreamMedia.Type,
		Duration:     duration,
		CreatedAt:    liveStreamMedia.CreatedAt,
		UpdatedAt:    liveStreamMedia.UpdatedAt,
		Status:       liveStreamMedia.Status,
		Qualities:    qualities,
		FrameRate:    frameRate,
		AudioBitrate: audioBitrate,
		Media:        media,
		CurrentView:  liveStreamMedia.CurrentView,
		TotalView:    liveStreamMedia.TotalView,
	}
}
