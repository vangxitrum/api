package core

import (
	"context"
	"fmt"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

type MediaHandler interface {
	GetMediaInfo(ctx context.Context) (*MediaInfo, error)
	GenerateThumbnail(ctx context.Context) error
	ConvertMediaToAudio(inputPath, outputPath string, streamIndex int) error
	ExtractCaptionFromMedia(captionFileName string, streamIndex int) error
}

var MediaQualityTranscodingConfigs = map[string]MediaQualityTranscodingConfig{
	"144p": {
		Width:        256,
		Height:       144,
		Bitrate:      400,
		MaxBitrate:   600,
		Crf:          23,
		TimeoutRatio: 100,
	},
	"240p": {
		Width:        426,
		Height:       240,
		Bitrate:      800,
		MaxBitrate:   1000,
		Crf:          23,
		TimeoutRatio: 100,
	},
	"360p": {
		Width:        640,
		Height:       360,
		Bitrate:      1500,
		MaxBitrate:   2000,
		Crf:          23,
		IsDefault:    true,
		TimeoutRatio: 100,
	},
	"480p": {
		Width:        854,
		Height:       480,
		Bitrate:      2000,
		MaxBitrate:   2500,
		Crf:          23,
		TimeoutRatio: 100,
	},
	"720p": {
		Width:        1280,
		Height:       720,
		Bitrate:      3000,
		MaxBitrate:   4000,
		Crf:          20,
		IsDefault:    true,
		TimeoutRatio: 100,
	},
	"1080p": {
		Width:        1920,
		Height:       1080,
		Bitrate:      6000,
		MaxBitrate:   8000,
		Crf:          18,
		IsDefault:    true,
		TimeoutRatio: 100,
	},
	"1440p": {
		Width:        2560,
		Height:       1440,
		Bitrate:      8000,
		MaxBitrate:   10000,
		Crf:          16,
		TimeoutRatio: 100,
	},
	"2160p": {
		Width:        3840,
		Height:       2160,
		Bitrate:      12000,
		MaxBitrate:   14000,
		Crf:          14,
		TimeoutRatio: 100,
	},
}

var (
	VTT = "vtt"
	ASS = "ass"

	CancelTranscodeError = fmt.Errorf("cancel")
)

type MediaQualityTranscodingConfig struct {
	Width        int
	Height       int
	Bitrate      int64
	MaxBitrate   int64
	Crf          int64
	IsDefault    bool
	TimeoutRatio float64
	ProfileIDC   string // Profile Indication Code (e.g., "42" for Baseline)
	ProfileIOP   string // Profile Compatibility (e.g., "E0")
	LevelIDC     string // Level Indication Code (e.g., "1E" for Level 3.0)
}

type MediaInfo struct {
	Streams []*models.MediaStream `json:"streams,omitempty"`
	Format  *models.MediaFormat   `json:"format,omitempty"`
}
