package models

import (
	"context"

	"github.com/google/uuid"
)

type StreamRepository interface {
	Create(context.Context, *MediaStream) error

	GetStreamsByMediaId(context.Context, uuid.UUID) ([]*MediaStream, error)

	DeleteStreamsByMediaId(context.Context, uuid.UUID) error

	CountStreamByCodecTypeAndMediaId(context.Context, string, uuid.UUID) (int, error)

	GetStreamsByCodecTypeAndMediaId(context.Context, string, uuid.UUID) ([]*MediaStream, error)
}

var (
	StreamCodecTypeVideo    = "video"
	StreamCodecTypeAudio    = "audio"
	StreamCodecTypeSubtitle = "subtitle"
)

type MediaStream struct {
	Id                 uuid.UUID   `json:"stream_id"                      gorm:"primaryKey;id;type:uuid"`
	MediaId            uuid.UUID   `json:"media_id"                       gorm:"type:uuid;references:Id"`
	Media              *Media      `json:"media"                          gorm:"foreignKey:MediaId"`
	Index              int         `json:"index"`
	TypeIndex          int32       `json:"type_index"`
	CodecName          string      `json:"codec_name"`
	CodecLongName      string      `json:"codec_long_name"`
	Profile            string      `json:"profile,omitempty"`
	CodecType          string      `json:"codec_type"`
	CodecTagString     string      `json:"codec_tag_string"`
	CodecTag           string      `json:"codec_tag"`
	Width              int32       `json:"width,omitempty"`
	Height             int32       `json:"height,omitempty"`
	CodedWidth         int         `json:"coded_width,omitempty"`
	CodedHeight        int         `json:"coded_height,omitempty"`
	ClosedCaptions     int         `json:"closed_captions,omitempty"`
	FilmGrain          int         `json:"film_grain,omitempty"`
	HasBFrames         int         `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string      `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string      `json:"display_aspect_ratio,omitempty"`
	PixFmt             string      `json:"pix_fmt,omitempty"`
	Level              int         `json:"level,omitempty"`
	ColorRange         string      `json:"color_range,omitempty"`
	ColorSpace         string      `json:"color_space,omitempty"`
	ColorTransfer      string      `json:"color_transfer,omitempty"`
	ColorPrimaries     string      `json:"color_primaries,omitempty"`
	ChromaLocation     string      `json:"chroma_location,omitempty"`
	Refs               int         `json:"refs,omitempty"`
	RFrameRate         string      `json:"r_frame_rate"`
	AvgFrameRate       string      `json:"avg_frame_rate"`
	TimeBase           string      `json:"time_base"`
	StartPts           int         `json:"start_pts"`
	StartTime          string      `json:"start_time"`
	ExtradataSize      int         `json:"extradata_size,omitempty"`
	SampleFmt          string      `json:"sample_fmt,omitempty"`
	SampleRate         string      `json:"sample_rate,omitempty"`
	Channels           int         `json:"channels,omitempty"`
	ChannelLayout      string      `json:"channel_layout,omitempty"`
	BitsPerSample      int         `json:"bits_per_sample,omitempty"`
	BitRate            string      `json:"bit_rate,omitempty"`
	DurationTs         int         `json:"duration_ts,omitempty"`
	Duration           string      `json:"duration,omitempty"`
	Disposition        Disposition `json:"disposition"                    gorm:"embedded;embeddedPrefix:disposition_"`
	Tags               Tag         `json:"tags"                           gorm:"embedded;embeddedPrefix:tag_"`
}

type Disposition struct {
	Default         int `json:"default"`
	Dub             int `json:"dub"`
	Original        int `json:"original"`
	Comment         int `json:"comment"`
	Lyrics          int `json:"lyrics"`
	Karaoke         int `json:"karaoke"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
	VisualImpaired  int `json:"visual_impaired"`
	CleanEffects    int `json:"clean_effects"`
	AttachedPic     int `json:"attached_pic"`
	TimedThumbnails int `json:"timed_thumbnails"`
	Captions        int `json:"captions"`
	Descriptions    int `json:"descriptions"`
	Metadata        int `json:"metadata"`
	Dependent       int `json:"dependent"`
	StillImage      int `json:"still_image"`
}

type Tag struct {
	Title                    string `json:"title"`
	Language                 string `json:"language"`
	BPS                      string `json:"BPS"`
	DURATION                 string `json:"DURATION"`
	NUMBEROFFRAMES           string `json:"NUMBER_OF_FRAMES"`
	NUMBEROFBYTES            string `json:"NUMBER_OF_BYTES"`
	STATISTICSWRITINGAPP     string `json:"_STATISTICS_WRITING_APP"`
	STATISTICSWRITINGDATEUTC string `json:"_STATISTICS_WRITING_DATE_UTC"`
	STATISTICSTAGS           string `json:"_STATISTICS_TAGS"`
}
