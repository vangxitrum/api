package models

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/number"
)

var (
	VideoType = "video"
	AudioType = "audio"
)

type MediaRepository interface {
	Create(context.Context, *Media) error
	CreateFile(context.Context, *MediaFile) error
	CreateMediaThumbnail(context.Context, *MediaThumbnail) error

	GetMediaById(
		context.Context, uuid.UUID,
	) (*Media, error)
	GetManyMediasByIds(
		context.Context, []uuid.UUID, uuid.UUID,
	) ([]Media, error)
	GetMediaByStatus(
		context.Context, string,
	) (*Media, error)
	GetMediasByStatus(
		context.Context, string,
	) ([]*Media, error)
	GetUserMediaById(
		context.Context, uuid.UUID, uuid.UUID,
	) (*Media, error)
	GetUserMedias(
		context.Context, uuid.UUID,
		GetMediaListInput,
	) ([]*Media, int64, error)
	GetAllUserMedias(
		context.Context, uuid.UUID,
	) ([]*Media, error)
	GetCompletedLiveStreamMedias(
		ctx context.Context,
	) ([]*Media, error)
	GetSavedLivestreamMedias(ctx context.Context) ([]*Media, error)
	GetDoneMedias(context.Context) ([]*Media, error)
	GetMediaByStatuses(context.Context, []string) ([]*Media, error)
	GetMostViewedMedia(context.Context, string, int) ([]*MediaViewData, error)

	UpdateMedia(context.Context, *Media) error
	UpdateMediaStatusById(context.Context, uuid.UUID, string) error
	UpdateMediaViewById(context.Context, uuid.UUID, int64, float64) error

	DeleteInactiveMediaPlayerTheme(context.Context, uuid.UUID) error
	DeleteMediaById(
		context.Context, uuid.UUID,
	) error
	DeleteMediaThumbnail(
		context.Context, uuid.UUID, uuid.UUID,
	) error
	DeleteMediaFiles(
		context.Context, uuid.UUID,
	) error
}

type Media struct {
	Id              uuid.UUID       `json:"id"                        gorm:"primaryKey;id;type:uuid"`
	UserId          uuid.UUID       `json:"user_id"                   gorm:"type:uuid"`
	JobId           string          `json:"job_id"`
	Title           string          `json:"title"`
	Description     string          `json:"description"`
	Qualities       string          `json:"qualities"`
	Source          string          `json:"source"`
	Type            string          `json:"type"`
	Public          bool            `json:"public"`
	Status          string          `json:"status"`
	Size            int64           `json:"size"`
	Secret          string          `json:"secret"`
	IsMp4           bool            `json:"is_mp4"`
	Mimetype        string          `json:"mimetype"`
	Metadata        JsonB           `json:"metadata,omitempty"        gorm:"type:jsonb"`
	Tags            string          `json:"tags,omitempty"`
	AvgFrameRate    float64         `json:"avg_frame_rate,omitempty"`
	View            int64           `json:"view"`
	WatchTime       float64         `json:"watch_time"`
	Captions        []*MediaCaption `json:"captions,omitempty"        gorm:"foreignKey:MediaId;references:Id"`
	Chapters        []*MediaChapter `json:"chapters,omitempty"        gorm:"foreignKey:MediaId;references:Id"`
	Parts           []*Part         `json:"parts,omitempty"           gorm:"foreignKey:MediaId;references:Id"`
	Streams         []*MediaStream  `json:"streams,omitempty"         gorm:"foreignKey:MediaId;references:Id"`
	MediaQualities  []*MediaQuality `json:"media_qualities,omitempty" gorm:"foreignKey:MediaId;references:Id"`
	Format          *MediaFormat    `json:"format,omitempty"          gorm:"foreignKey:MediaId;references:Id"`
	TranscodeTime   float64         `json:"transcode_time"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	PlayerThemeId   *uuid.UUID      `json:"player_theme_id,omitempty"`
	PlayerTheme     *PlayerTheme    `json:"player_theme,omitempty"    gorm:"references:Id"`
	Watermark       *MediaWatermark `json:"watermark,omitempty"       gorm:"foreignKey:MediaId;references:Id"`
	PlaylistItems   []*PlaylistItem `json:"playlist_items,omitempty"  gorm:"foreignKey:MediaId;references:Id"`
	HaveThumbnail   bool            `json:"have_thumbnail"            gorm:"-"`
	SegmentDuration int32           `json:"segment_duration"          gorm:"default:0"`
	MediaFiles      []*MediaFile    `json:"media_files,omitempty"     gorm:"foreignKey:MediaId;references:Id"`
	MediaThumbnail  *MediaThumbnail `json:"media_thumbnail,omitempty" gorm:"foreignKey:MediaId;references:Id"`
}

func NewMedia(
	userId uuid.UUID,
	mediaType string,
	title, description string,
	metadata []Metadata,
	qualityConfigs []*QualityConfig,
	segmentDuration int32,
	tags []string,
	isPublic bool,
) (*Media, error) {
	newMedia := &Media{
		Id:              uuid.New(),
		UserId:          userId,
		Title:           title,
		Description:     description,
		Tags:            strings.Join(tags, ","),
		Type:            mediaType,
		Public:          isPublic,
		Status:          NewStatus,
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
		SegmentDuration: segmentDuration,
	}

	newMedia.MediaQualities = make([]*MediaQuality, 0, len(qualityConfigs))
	for _, config := range qualityConfigs {
		newMedia.MediaQualities = append(
			newMedia.MediaQualities,
			NewQuality(newMedia.Id, config),
		)
	}

	if len(metadata) != 0 {
		newMedia.Metadata = make(map[string]any)
		for _, meta := range metadata {
			newMedia.Metadata[meta.Key] = meta.Value
		}
	}

	return newMedia, nil
}

func (v *Media) GetSourcePath(path string) string {
	return fmt.Sprintf("%s/%s/source", path, v.Id.String())
}

func (v *Media) GetSourceTempPath(path string) string {
	return fmt.Sprintf("%s/%s/source_temp", path, v.Id.String())
}

func (v *Media) GetMediaDuration() float64 {
	if v.Format != nil {
		mediaDuration, err := strconv.ParseFloat(v.Format.Duration, 64)
		if err == nil {
			return number.Round(mediaDuration, 2)
		}
	}

	return 0
}

func (v *Media) IsDone() bool {
	return v.Status == DoneStatus || v.Status == TranscribingStatus
}

func (v *Media) IsTranscoding() bool {
	return v.Status == TranscodingStatus || v.Status == WaitingStatus
}

func (v *Media) IsNew() bool {
	return v.Status == NewStatus || v.Status == HiddenStatus
}

func (v *Media) IsDeleted() bool {
	return v.Status == DeletedStatus || v.Status == DeletingStatus
}

type MediaFile struct {
	FileId  string    `json:"file_id"  gorm:"primaryKey"`
	MediaId uuid.UUID `json:"media_id" gorm:"primaryKey;type:uuid"`
	File    *CdnFile  `json:"file"     gorm:"foreignKey:FileId;references:Id"`
}

type MediaThumbnail struct {
	MediaId     uuid.UUID  `json:"media_id"     gorm:"primaryKey;type:uuid"`
	ThumbnailId uuid.UUID  `json:"thumbnail_id" gorm:"primaryKey;type:uuid"`
	Thumbnail   *Thumbnail `json:"thumbnail"    gorm:"foreignKey:ThumbnailId;references:Id"`
}

type MediaObject struct {
	Id            string           `json:"id"`
	UserId        string           `json:"user_id"`
	Title         string           `json:"title"`
	Description   string           `json:"description"`
	Metadata      []*Metadata      `json:"metadata"`
	Tags          []string         `json:"tags"`
	Qualities     []*QualityObject `json:"qualities"`
	Captions      []*MediaCaption  `json:"captions"`
	Chapters      []*MediaChapter  `json:"chapters"`
	View          int64            `json:"view"`
	PlayerTheme   *PlayerTheme     `json:"player_theme"`
	Duration      float64          `json:"duration"`
	Type          string           `json:"type"`
	Size          int64            `json:"size"`
	IsMp4         bool             `json:"is_mp4,omitempty"`
	IsPublic      bool             `json:"is_public"`
	Status        string           `json:"status"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Assets        *MediaAssets     `json:"assets"`
	PlayerThemeId string           `json:"player_theme_id"`
} //	@name	Media

type MediaAssets struct {
	SourceUrl     string `json:"source_url"`
	ThumbnailUrl  string `json:"thumbnail_url"`
	HlsUrl        string `json:"hls_url"`
	DashUrl       string `json:"dash_url"`
	Mp4Url        string `json:"mp4_url,omitempty"`
	HlsIFrame     string `json:"hls_iframe"`
	DashIFrame    string `json:"dash_iframe"`
	HlsPlayerUrl  string `json:"hls_player_url"`
	DashPlayerUrl string `json:"dash_player_url"`
} //	@name	MediaAssets

func NewMediaObject(
	media *Media,
) *MediaObject {
	var metadata []*Metadata
	for key, value := range media.Metadata {
		metadata = append(
			metadata, &Metadata{
				Key:   key,
				Value: value.(string),
			},
		)
	}

	var tags []string
	if media.Tags != "" {
		tags = strings.Split(media.Tags, ",")
	}

	var status string
	switch media.Status {
	case TranscodingStatus, UploadedStatus, UploadingStatus, WaitingStatus:
		status = TranscodingStatus
	case TranscribingStatus:
		status = DoneStatus
	default:
		status = media.Status
	}

	obj := &MediaObject{
		Id:          media.Id.String(),
		UserId:      media.UserId.String(),
		Title:       media.Title,
		Description: media.Description,
		Metadata:    metadata,
		Chapters:    media.Chapters,
		Captions:    media.Captions,
		Tags:        tags,
		IsPublic:    media.Public,
		Status:      status,
		Type:        media.Type,
		Size:        media.Size,
		View:        media.View,
		CreatedAt:   media.CreatedAt,
		UpdatedAt:   media.UpdatedAt,
	}

	if media.Format != nil {
		mediaDuration, err := strconv.ParseFloat(media.Format.Duration, 64)
		if err == nil {
			obj.Duration = mediaDuration
		}
	}

	if media.PlayerThemeId != nil {
		obj.PlayerThemeId = media.PlayerThemeId.String()
	}

	var haveHls, haveDash bool
	for _, quality := range media.MediaQualities {
		if quality.Status == DoneStatus {
			switch quality.Type {
			case HlsQualityType:
				haveHls = true
			case DashQualityType:
				haveDash = true
			}
		}

		obj.Qualities = append(
			obj.Qualities,
			NewQualityObject(quality),
		)
	}

	sort.Slice(obj.Qualities, func(i, j int) bool {
		if obj.Qualities[i].Name == "" {
			return false
		}

		if obj.Qualities[j].Name == "" {
			return true
		}

		x, err := strconv.Atoi(strings.TrimSuffix(obj.Qualities[i].Name, "p"))
		if err != nil {
			return false
		}

		y, err := strconv.Atoi(strings.TrimSuffix(obj.Qualities[j].Name, "p"))
		if err != nil {
			return true
		}

		return x < y
	})

	if haveHls || haveDash {
		hlsPlayerUrl := fmt.Sprintf("%s/vod/hls/%s", PlayerUrl, media.Id)
		dashPlayerUrl := fmt.Sprintf("%s/vod/dash/%s", PlayerUrl, media.Id)
		baseUrl := fmt.Sprintf(AssetUrlFormat, BeUrl, media.Id)
		obj.Assets = &MediaAssets{
			SourceUrl: baseUrl + "/source",
		}

		if media.Type != AudioType {
			obj.Assets.Mp4Url = baseUrl + "/mp4"
		}

		if media.MediaThumbnail != nil {
			obj.Assets.ThumbnailUrl = baseUrl + "/thumbnail"
		}

		if haveHls {
			obj.Assets.HlsUrl = baseUrl + "/manifest.m3u8"
			obj.Assets.HlsPlayerUrl = hlsPlayerUrl
		}

		if haveDash {
			obj.Assets.DashUrl = baseUrl + "/manifest.mpd"
			obj.Assets.DashPlayerUrl = dashPlayerUrl
		}

		if media.Secret != "" {
			token := "?token=" + media.Secret
			obj.Assets.SourceUrl += token
			obj.Assets.Mp4Url += token
			if haveHls {
				obj.Assets.HlsUrl += token
				obj.Assets.HlsPlayerUrl += token
			}

			if haveDash {
				obj.Assets.DashUrl += token
				obj.Assets.DashPlayerUrl += token
			}

			if media.MediaThumbnail != nil {
				obj.Assets.ThumbnailUrl += token + "&resolution=original"
			}
		} else if media.MediaThumbnail != nil {
			obj.Assets.ThumbnailUrl += "?resolution=original"
		}

		if haveHls {
			obj.Assets.HlsIFrame = fmt.Sprintf(
				`<iframe src="%s" width="100%%" height="100%%" frameborder="0" scrolling="no" allowfullscreen="true"></iframe>`,
				obj.Assets.HlsPlayerUrl,
			)
		}
		if haveDash {
			obj.Assets.DashIFrame = fmt.Sprintf(
				`<iframe src="%s" width="100%%" height="100%%" frameborder="0" scrolling="no" allowfullscreen="true"></iframe>`,
				obj.Assets.DashPlayerUrl,
			)
		}
	} else if media.IsNew() {
		if media.MediaThumbnail != nil {
			obj.Assets = &MediaAssets{
				ThumbnailUrl: fmt.Sprintf(
					AssetUrlFormat,
					BeUrl, media.Id,
				) + "/thumbnail?resolution=original",
			}
			if media.Secret != "" {
				obj.Assets.ThumbnailUrl += "&token=" + media.Secret
			}
		} else {
			obj.Assets = &MediaAssets{}
		}
	}

	if media.PlayerTheme != nil {
		obj.PlayerTheme = media.PlayerTheme
	}

	return obj
}

func NewMediaObjects(
	media []*Media,
) []*MediaObject {
	var mediaObjects []*MediaObject
	for _, media := range media {
		mediaObjects = append(
			mediaObjects,
			NewMediaObject(media),
		)
	}

	return mediaObjects
}

type Metadata struct {
	Key   string `json:"key"   gorm:"primaryKey"`
	Value string `json:"value"`
} //	@name	Metadata

type MediaTag struct {
	MediaId uuid.UUID `json:"-"   gorm:"primaryKey"`
	Tag     string    `json:"tag" gorm:"primaryKey"`
}

type GetMediaListInput struct {
	OrderBy  string
	SortBy   string
	Status   []string
	Offset   int
	Limit    int
	Type     string
	Metadata []Metadata
	Tags     []string
	Search   string
}

type GetStatisticMediasInput struct {
	Limit  int
	Offset int
	From   time.Time
	To     time.Time
	Type   string
}

type UpdateMediaInfoInput struct {
	Description *string    `json:"description"`
	Metadata    []Metadata `json:"metadata"`
	Tags        []string   `json:"tags"`
	Title       *string    `json:"title"`
	IsPublic    *bool      `json:"is_public"`
	PlayerId    *uuid.UUID `json:"player_id"`
}

type JsonB map[string]any

func (j JsonB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JsonB) Scan(value any) error {
	return json.Unmarshal(value.([]byte), j)
}

type MediaInfo struct {
	Id                   string         `json:"id,omitempty"`
	Title                string         `json:"title,omitempty"`
	UserId               string         `json:"user_id,omitempty"`
	Status               string         `json:"status,omitempty"`
	HaveAudio            bool           `json:"have_audio,omitempty"`
	Mimetype             string         `json:"mimetype,omitempty"`
	Size                 int64          `json:"size,omitempty"`
	AvgFrameRate         float64        `json:"avg_frame_rate,omitempty"`
	Width                int32          `json:"width,omitempty"`
	Height               int32          `json:"height,omitempty"`
	MediaIndex           int32          `json:"media_index,omitempty"`
	AudioIndex           int32          `json:"audio_index,omitempty"`
	Watermark            *Watermark     `json:"watermark,omitempty"`
	Qualities            []*QualityInfo `json:"qualities,omitempty"`
	MediaBitrate         int32          `json:"media_bitrate,omitempty"`
	AudioBitrate         int32          `json:"audio_bitrate,omitempty"`
	Duration             float64        `json:"duration,omitempty"`
	StartTime            float64        `json:"start_time,omitempty"`
	CdnFiles             []*CdnFileInfo `json:"cdn_files,omitempty"`
	ForceServerTranscode bool           `json:"force_server_transcode,omitempty"`
}

type CdnFileInfo struct {
	Id     string `json:"id"`
	Type   string `json:"type"`
	Offset int64  `json:"offset"`
	Size   int64  `json:"size"`
	Index  int    `json:"index"`
}

type QualityInfo struct {
	Id                 string         `json:"id,omitempty"`
	Name               string         `json:"name,omitempty"`
	Status             string         `json:"status,omitempty"`
	Type               string         `json:"type,omitempty"`
	TranscodedByServer bool           `json:"transcoded_by_server,omitempty"`
	Profile            string         `json:"profile,omitempty"`
	CdnFiles           []*CdnFileInfo `json:"cdn_files,omitempty"`
}

func NewHandlerMediaRequest(
	media *Media,
	haveAudio bool,
	width, height, mediaBitrate, audioBitrate, mediaIndex, audioIndex int,
	avgFrameRate, duration, startTime float64,
	forceServerTranscode bool,
) *MediaInfo {
	newMedia := &MediaInfo{
		Id:                   media.Id.String(),
		Title:                media.Title,
		UserId:               media.UserId.String(),
		HaveAudio:            haveAudio,
		AudioIndex:           int32(audioIndex),
		MediaIndex:           int32(mediaIndex),
		AvgFrameRate:         avgFrameRate,
		Size:                 media.Size,
		Mimetype:             media.Mimetype,
		Width:                int32(width),
		Height:               int32(height),
		MediaBitrate:         int32(mediaBitrate),
		AudioBitrate:         int32(audioBitrate),
		Duration:             duration,
		StartTime:            startTime,
		ForceServerTranscode: forceServerTranscode,
	}

	newMedia.Qualities = make([]*QualityInfo, 0, len(media.MediaQualities))
	for _, quality := range media.MediaQualities {
		newMedia.Qualities = append(
			newMedia.Qualities,
			&QualityInfo{
				Id:     quality.Id.String(),
				Name:   quality.Name,
				Type:   quality.Type,
				Status: NewStatus,
			},
		)
	}

	return newMedia
}

func (v *Media) GetHlsManifestUrl() string {
	hlsBase := fmt.Sprintf("%s/api/media/%s/manifest.m3u8", BeUrl, v.Id)
	if v.Secret != "" {
		return hlsBase + "?token=" + v.Secret
	}
	return hlsBase
}

func (v *Media) GetThumbnailUrl() string {
	thumbnailBase := fmt.Sprintf("%s/api/media/%s", BeUrl, v.Id) + "/thumbnail"
	if v.Secret != "" {
		return thumbnailBase + "?token=" + v.Secret + "&resolution=original"
	}
	return thumbnailBase + "?resolution=original"
}

func (v *Media) GetHlsPlayerUrl() string {
	playerUrl := fmt.Sprintf("%s/vod/hls/%s", PlayerUrl, v.Id)
	if v.Secret != "" {
		return playerUrl + "?token=" + v.Secret
	}
	return playerUrl
}

type MediaViewData struct {
	Id    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	View  int64     `json:"view"`
}
