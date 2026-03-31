package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	maps "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/map"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
)

var logger *slog.Logger

type GeneralHandler struct {
	media *models.Media

	inputPath  string
	outputPath string

	separateAudio         bool
	generateVTTCaptions   bool
	generateASSCaptions   bool
	generateThumbnail     bool
	generateWithWatermark bool

	storageHelper storage.StorageHelper

	pIdMapping     *maps.RWMap
	cancelMediaMap *maps.RWMap
}

type GeneralOption func(*GeneralHandler)

func NewGeneralHandler(
	media *models.Media,
	options ...GeneralOption,
) MediaHandler {
	handler := &GeneralHandler{
		media:          media,
		inputPath:      "input",
		outputPath:     "output",
		pIdMapping:     maps.NewRWMap(),
		cancelMediaMap: maps.NewRWMap(),
	}

	for _, option := range options {
		option(handler)
	}

	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	return handler
}

func WithWatermark() GeneralOption {
	return func(h *GeneralHandler) {
		h.generateWithWatermark = true
	}
}

func WithGenerrateThumbnail() GeneralOption {
	return func(h *GeneralHandler) {
		h.generateThumbnail = true
	}
}

func WithStorage(storageHelper storage.StorageHelper) GeneralOption {
	return func(h *GeneralHandler) {
		h.storageHelper = storageHelper
	}
}

func WithDebugLog() GeneralOption {
	return func(h *GeneralHandler) {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}
}

func WithInputPath(path string) GeneralOption {
	return func(h *GeneralHandler) {
		h.inputPath = path
	}
}

func WithOutputPath(path string) GeneralOption {
	return func(h *GeneralHandler) {
		h.outputPath = path
	}
}

func WithSeparateAudio() GeneralOption {
	return func(h *GeneralHandler) {
		h.separateAudio = true
	}
}

func WithGenerateVTTCaptions() GeneralOption {
	return func(h *GeneralHandler) {
		h.generateVTTCaptions = true
	}
}

func WithGenerateASSCaptions() GeneralOption {
	return func(h *GeneralHandler) {
		h.generateASSCaptions = true
	}
}

func (h *GeneralHandler) GetMediaInfo(
	ctx context.Context,
) (*MediaInfo, error) {
	path := h.media.GetSourcePath(h.inputPath)
	args := []string{
		"-v",
		"quiet",
		"-print_format",
		"json",
		"-show_format",
		"-show_streams",
		path,
	}

	cmd := exec.Command(
		"./ffmpegd/ffprobe",
		args...,
	)

	logger.Debug("GetMediaInfo",
		slog.Any("id", h.media.Id),
		slog.Any("path", path),
		slog.Any("cmd", cmd.String()),
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Parse the JSON output
	var info MediaInfo
	if err = json.Unmarshal(output, &info); err != nil {
		return nil, err
	}

	outPath := fmt.Sprintf("%s/%s", h.outputPath, h.media.Id)
	if err := os.MkdirAll(outPath, 0755); err != nil {
		return nil, err
	}

	masterM3U8Path := fmt.Sprintf("%s/master.m3u8", outPath)
	file, err := os.Create(masterM3U8Path)
	if err != nil {
		return nil, err
	}

	masterM3U8Header := "#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-INDEPENDENT-SEGMENTS\n\n\n"
	if _, err := file.WriteString(masterM3U8Header); err != nil {
		return nil, err
	}

	if h.generateWithWatermark {
		if err := func() error {
			watermarkPath := fmt.Sprintf(
				"%s/%s/watermark",
				h.outputPath,
				h.media.Id,
			)
			reader, err := h.storageHelper.Download(
				ctx,
				&storage.Object{
					Id:     h.media.Watermark.CdnFile.Id,
					Size:   h.media.Watermark.CdnFile.Size,
					Offset: h.media.Watermark.CdnFile.Offset,
				},
			)
			if err != nil {
				return err
			}

			file, err := os.Create(watermarkPath)
			if err != nil {
				return err
			}

			defer file.Close()

			var flag bool
			buffer := make([]byte, 1024)
			for {
				n, err := reader.Read(buffer)
				if err != nil {
					if errors.Is(err, io.EOF) {
						flag = true
					} else {
						return err
					}
				}

				if _, err := file.Write(buffer[:n]); err != nil {
					return err
				}

				if flag {
					break
				}
			}

			return nil
		}(); err != nil {
			return nil, err
		}
	}

	return &info, nil
}

func (h *GeneralHandler) GetFileInfo(
	ctx context.Context,
	mediaId uuid.UUID,
	fileType string,
) (*MediaInfo, error) {
	streamPath := fmt.Sprintf("%s/%s", h.inputPath, mediaId)
	filePath := fmt.Sprintf("%s/media.%s", streamPath, fileType)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			slog.Warn("media file not found", "id", mediaId)
			return nil, nil
		}
		return nil, fmt.Errorf("error checking media file: %v", err)
	}

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}

	cmd := exec.Command("./ffmpegd/ffprobe", args...)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed for %s: %v", filePath, err)
	}

	var info MediaInfo
	if err = json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	if len(info.Streams) == 0 {
		return nil, fmt.Errorf("no streams found in ts file")
	}

	return &info, nil
}

func (h *GeneralHandler) GenerateThumbnail(
	ctx context.Context,
) error {
	return h.getMediaThumbnail(ctx)
}

func (h *GeneralHandler) getMediaThumbnail(
	_ context.Context,
) error {
	path := h.media.GetSourcePath(h.inputPath)
	outPath := fmt.Sprintf("%s/%s/thumbnail", h.outputPath, h.media.Id)
	if err := os.MkdirAll(outPath, 0755); err != nil {
		return err
	}

	var mediaStream *models.MediaStream
	for _, stream := range h.media.Streams {
		if stream.CodecType == models.StreamCodecTypeVideo {
			mediaStream = stream
			break
		}
	}

	if mediaStream == nil {
		return fmt.Errorf("media stream not found")
	}

	var width, height int32
	height = mediaStream.Height
	width = mediaStream.Width
	if height > 720 || width > 720 {
		if height < width {
			height = 720
			width = -1
		} else {
			height = -1
			width = 720

		}
	}

	args := []string{
		"-v",
		"quiet",
		"-i",
		path,
		"-vf",
		fmt.Sprintf(
			"thumbnail,scale=%d:%d:force_original_aspect_ratio=decrease",
			width,
			height,
		),
		"-frames:v",
		"1",
		"-f",
		"image2",
		fmt.Sprintf("%s/original.jpg", outPath),
	}

	cmd := exec.Command(
		"./ffmpegd/ffmpeg",
		args...,
	)

	logger.Debug("getMediaThumbnail",
		slog.Any("id", h.media.Id),
		slog.Any("path", path),
		slog.Any("cmd", cmd.String()),
	)

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (h *GeneralHandler) ConvertMediaToAudio(
	mediaPath, audioPath string,
	streamIndex int,
) error {
	args := []string{
		"-i", mediaPath,
		"-vn",
		"-acodec", "libmp3lame",
		"-b:a", "128k",
		"-ar", "44100",
		"-ac", fmt.Sprintf("%d", streamIndex),
		audioPath,
	}

	cmd := exec.Command(ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"error converting file: %v, stderr: %s",
			err,
			stderr.String(),
		)
	}

	return nil
}

func (h *GeneralHandler) ExtractCaptionFromMedia(
	fileName string,
	streamIndex int,
) error {
	args := []string{
		"-i", h.media.GetSourcePath(h.inputPath),
		"-map", fmt.Sprintf("0:%d", streamIndex),
		filepath.Join(h.outputPath, h.media.Id.String(), fileName),
	}

	cmd := exec.Command(ffmpegPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"error converting file: %v, stderr: %s",
			err,
			stderr.String(),
		)
	}

	return nil
}
