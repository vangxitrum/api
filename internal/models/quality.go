package models

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	HlsQualityType  = "hls"
	Mp4QualityType  = "mp4"
	DashQualityType = "dash"
)

type QualityRepository interface {
	Create(context.Context, *MediaQuality) error
	CreateFile(context.Context, *MediaQualityFile) error

	CountQualitiesByMediaIdAndStatus(
		context.Context,
		uuid.UUID,
		string,
	) (int, error)

	GetQualityById(context.Context, uuid.UUID) (*MediaQuality, error)
	GetMp4QualityByMediaId(context.Context, uuid.UUID) (*MediaQuality, error)
	GetQualitiesByMediaId(context.Context, uuid.UUID) ([]*MediaQuality, error)
	GetQualityByPlaylistId(context.Context, string) (*MediaQuality, error)

	UpdateQuality(context.Context, *MediaQuality) error
	UpdateQualityAudioConfig(
		context.Context,
		uuid.UUID,
		*AudioConfig,
	) error

	DeleteQualityById(context.Context, uuid.UUID) error
	DeleteQualitiesByMediaId(context.Context, uuid.UUID) error
}

type MediaQuality struct {
	Id              uuid.UUID           `json:"quality_id"        gorm:"primaryKey;id;type:uuid"`
	MediaId         uuid.UUID           `json:"media_id"          gorm:"type:uuid"`
	Media           *Media              `json:"-"                 gorm:"foreignKey:MediaId;references:Id"`
	VideoPlaylistId string              `json:"video_playlist_id"`
	AudioPlaylistId string              `json:"audio_playlist_id"`
	Name            string              `json:"name"`
	Resolution      string              `json:"resolution"`
	VideoConfig     *VideoConfig        `json:"video_config"      gorm:"embedded;embeddedPrefix:media_config_"`
	AudioConfig     *AudioConfig        `json:"audio_config"      gorm:"embedded;embeddedPrefix:audio_config_"`
	Type            string              `json:"type"`
	ContainerType   string              `json:"container_type"`
	Status          string              `json:"status"`
	TranscodeTime   float64             `json:"transcode_time"    gorm:"default:0.0"`
	CreatedAt       time.Time           `json:"created_at"`
	TranscodedAt    time.Time           `json:"transcoded_at"`
	Profile         string              `json:"-"`
	VideoCodec      string              `json:"video_codec"`
	AudioCodec      string              `json:"audio_codec"`
	Bandwidth       int32               `json:"bandwidth"`
	TranscodeCost   decimal.Decimal     `json:"transcode_cost"`
	Files           []*MediaQualityFile `json:"files"             gorm:"foreignKey:MediaQualityId;references:Id"`
} //	@name	MediaQuality

type MediaQualityFile struct {
	MediaQualityId uuid.UUID `json:"media_quality_id" gorm:"primaryKey;type:uuid"`
	FileId         string    `json:"file_id"          gorm:"primaryKey"`
	File           *CdnFile  `json:"file"             gorm:"foreignKey:FileId;references:Id"`
}

type QualityObject struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Type         string `json:"type"`
	VideoCodec   string `json:"video_codec"`
	VideoBitrate int64  `json:"video_bitrate"`
	AudioCodec   string `json:"audio_codec"`
	AudioBitrate int64  `json:"audio_bitrate"`
} //	@name	QualityObject

func NewQualityObject(
	q *MediaQuality,
) *QualityObject {
	obj := &QualityObject{
		Name:   q.Resolution,
		Status: q.Status,
		Type:   q.Type,
	}

	if q.VideoConfig != nil {
		obj.VideoCodec = q.VideoConfig.Codec
		obj.VideoBitrate = q.VideoConfig.Bitrate
	}

	if q.AudioConfig != nil {
		obj.AudioCodec = q.AudioConfig.Codec
		obj.AudioBitrate = q.AudioConfig.Bitrate
	}

	return obj
}

type QualityConfig struct {
	Type          string       `json:"type"`
	ContainerType string       `json:"container_type"`
	Resolution    string       `json:"resolution"`
	VideoConfig   *VideoConfig `json:"video_config"`
	AudioConfig   *AudioConfig `json:"audio_config"`
}

func (c *QualityConfig) IsValid(mediaType string) (string, bool) {
	if _, ok := ValidQualityTypes[c.Type]; !ok {
		return "Invalid quality type.", false
	}

	if validContainerTypes, ok := ValidContainerType[c.Type]; !ok {
		return "Invalid quality type.", false
	} else {
		if _, ok := validContainerTypes[c.ContainerType]; !ok {
			return "Invalid container type.", false
		}
	}

	if c.Resolution == "" {
		return "Resolution cannot be empty.", false
	}

	if message, ok := c.AudioConfig.IsValid(); !ok {
		return message, ok
	}

	if mediaType == VideoType && c.VideoConfig != nil {
		if _, ok := ValidMediaQualities[c.Resolution]; !ok {
			return "Invalid resolution.", false
		}

		maxBitrate, ok := MaxVideoQualityBitrates[c.Resolution]
		if !ok {
			return "Invalid resolution.", false
		}

		if c.VideoConfig.Bitrate > maxBitrate {
			return fmt.Sprintf(
				"Video bitrate exceeds maximum allowed for %s resolution: %d",
				c.Resolution,
				maxBitrate,
			), false
		}

		if message, ok := c.VideoConfig.IsValid(); !ok {
			return message, ok
		}
	}

	return "", true
}

type VideoConfig struct {
	Codec   string `json:"codec"`
	Bitrate int64  `json:"bitrate"`
	Index   int32  `json:"index"`
	Width   int32  `json:"-"`
	Height  int32  `json:"-"`
}

func (c *VideoConfig) Hash() string {
	if c == nil {
		return ""
	}

	return fmt.Sprintf(
		"%s-%d-%d-%d-%d",
		c.Codec,
		c.Bitrate,
		c.Width,
		c.Height,
		c.Index,
	)
}

func (c *VideoConfig) IsValid() (string, bool) {
	if c == nil {
		return "", true
	}

	if _, ok := ValidVideoCodecs[c.Codec]; !ok {
		return "Invalid video codec.", false
	}

	if c.Bitrate <= 0 {
		return "Invalid video bitrate", false
	}

	if c.Index < 0 {
		return "Invalid video index", false
	}

	return "", true
}

type AudioConfig struct {
	Codec      string `json:"codec"`
	Bitrate    int64  `json:"bitrate"`
	SampleRate int32  `json:"sample_rate"`
	Channels   string `json:"channels"`
	Index      int32  `json:"index"`
	Language   string `json:"language"`
}

func (c *AudioConfig) Hash() string {
	if c == nil {
		return ""
	}

	return fmt.Sprintf(
		"%s-%d-%d-%s-%d-%s",
		c.Codec,
		c.Bitrate,
		c.SampleRate,
		c.Channels,
		c.Index,
		c.Language,
	)
}

func (c *AudioConfig) IsValid() (string, bool) {
	if c == nil {
		return "", true
	}

	if _, ok := ValidAudioCodecs[c.Codec]; !ok {
		return "Invalid audio codec.", false
	}

	if c.Bitrate <= 0 {
		return "Invalid audio bitrate.", false
	}

	if c.Bitrate > MaxAudioBitrate {
		return fmt.Sprintf(
			"Audio bitrate exceeds maximum allowed: %d",
			MaxAudioBitrate,
		), false
	}

	if _, ok := ValidSampleRates[c.SampleRate]; !ok {
		return "Invalid audio sample rate.", false
	}

	if c.Index < 0 {
		return "Invalid audio index.", false
	}

	if _, ok := ValidAudioChannels[c.Channels]; !ok {
		return "Invalid audio channels.", false
	}

	return "", true
}

func NewQuality(
	mediaId uuid.UUID,
	qualityConfig *QualityConfig,
) *MediaQuality {
	newQuality := &MediaQuality{
		Id:            uuid.New(),
		MediaId:       mediaId,
		Type:          qualityConfig.Type,
		ContainerType: qualityConfig.ContainerType,
		Resolution:    qualityConfig.Resolution,
		Status:        WaitingStatus,
		CreatedAt:     time.Now().UTC(),
	}

	if qualityConfig.VideoConfig != nil {
		newQuality.VideoConfig = qualityConfig.VideoConfig
	}

	if qualityConfig.AudioConfig != nil {
		newQuality.AudioConfig = qualityConfig.AudioConfig
	}

	return newQuality
}
