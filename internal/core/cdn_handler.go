package core

// import (
// 	"context"
// 	"fmt"
// 	"io"
// 	"log/slog"
// 	"os"
// 	"strconv"
// 	"strings"
// 	"time"
//
// 	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
// 	maps "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/map"
// 	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
// )
//
// type CdnHandler struct {
// 	media     *models.Media
// 	mediaPath string
//
// 	inputPath  string
// 	outputPath string
//
// 	separateAudio         bool
// 	generateVTTSubtitles  bool
// 	generateASSSubtitles  bool
// 	generateThumbnail     bool
// 	generateWithWatermark bool
//
// 	storageHelper storage.StorageHelper
//
// 	pIdMapping     *maps.RWMap
// 	cancelMediaMap *maps.RWMap
// }
//
// type CdnOption func(*CdnHandler)
//
// func NewCdnHandler(
// 	media *models.Media,
// 	storageHelper storage.StorageHelper,
// 	options ...CdnOption,
// ) MediaHandler {
// 	ext := models.MimetypeMapping[media.Mimetype]
// 	handler := &CdnHandler{
// 		media:          media,
// 		inputPath:      "input",
// 		outputPath:     "output",
// 		storageHelper:  storageHelper,
// 		pIdMapping:     maps.NewRWMap(),
// 		cancelMediaMap: maps.NewRWMap(),
// 	}
//
// 	for _, option := range options {
// 		option(handler)
// 	}
//
// 	handler.mediaPath = fmt.Sprintf("%s/%s/media.%s", handler.inputPath, media.Id, ext)
// 	if logger == nil {
// 		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
// 			Level: slog.LevelInfo,
// 		}))
// 	}
//
// 	return handler
// }
//
// func WithCdnInputPath(inputPath string) CdnOption {
// 	return func(h *CdnHandler) {
// 		h.inputPath = inputPath
// 	}
// }
//
// func WithCdnOutputPath(outputPath string) CdnOption {
// 	return func(h *CdnHandler) {
// 		h.outputPath = outputPath
// 	}
// }
//
// func WithCdnDebugLog() CdnOption {
// 	return func(h *CdnHandler) {
// 		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
// 			Level: slog.LevelDebug,
// 		}))
// 	}
// }
//
// func (h *CdnHandler) ConvertMediaToMp4(ctx context.Context) error {
// 	return nil
// }
//
// func (h *CdnHandler) CancelTranscode(ctx context.Context) (chan struct{}, error) {
// 	return nil, nil
// }
//
// func (h *CdnHandler) GetMediaInfo(ctx context.Context) (*MediaInfo, error) {
// 	return nil, nil
// }
//
// func (h *CdnHandler) GenerateThumbnail(ctx context.Context) error {
// 	return nil
// }
//
// func (h *CdnHandler) HandleMedia(ctx context.Context) error {
// 	source, err := os.Open(h.mediaPath)
// 	if err != nil {
// 		return err
// 	}
//
// 	resp, err := h.storageHelper.UploadRaw(h.media.Id.String(), h.media.Size, source)
// 	if err != nil {
// 		return err
// 	}
//
// 	sourceFileId := resp.Id
// 	h.media.CdnFiles = append(
// 		h.media.CdnFiles,
// 		models.NewCdnFile(
// 			h.media.UserId,
// 			h.media.Id,
// 			resp.Id,
// 			resp.Size,
// 			resp.Offset,
// 			1,
// 			models.CdnSourceFileType,
// 		),
// 	)
//
// 	masterM3U8Path := fmt.Sprintf("%s/%s/master.m3u8", h.outputPath, h.media.Id)
// 	file, err := os.Open(masterM3U8Path)
// 	if err != nil {
// 		return err
// 	}
//
// 	defer file.Close()
//
// 	haveAudio := false
// 	var mediaStream *models.Stream
// 	for _, stream := range h.media.Streams {
// 		if stream.CodecType == models.StreamCodecTypeAudio {
// 			haveAudio = true
// 		}
//
// 		if stream.CodecType == models.StreamCodecTypeMedia {
// 			mediaStream = stream
// 		}
// 	}
//
// 	var masterM3U8Data string
// 	for _, quality := range h.media.MediaQualities {
// 		quality.Status = models.FailStatus
// 		config := MediaQualityTranscodingConfigs[quality.Name]
// 		transcodeProfile := &storage.TranscodeProfile{
// 			Version:              1,
// 			Encoder:              storage.H264Encoder,
// 			OutputVwidth:         config.Width,
// 			OutputVheight:        config.Height,
// 			OutputVmaxRate:       int(config.Bitrate * 1000),
// 			OutputTranscodeAudio: haveAudio,
// 			OutputStructType:     storage.SegmentStructType,
// 			OutputContainer:      storage.TsContainer,
// 			SegmentNameTemplate:  "segment%03d.ts",
// 			SegmentDuration:      6,
// 			PlaylistName:         "playlist.m3u8",
// 		}
//
// 		if haveAudio {
// 			transcodeProfile.OutputAbitrate = 44100
// 			transcodeProfile.OutputAcodec = storage.AccCodec
// 		}
//
// 		resp, err := h.storageHelper.Transcode(resp.Id, transcodeProfile)
// 		if err != nil {
// 			return err
// 		}
//
// 		fileId := resp.FileRecordId
// 		var status string
// 		for {
// 			status, err = h.storageHelper.GetTranscodeStatus(fileId)
// 			if err != nil {
// 				return err
// 			}
//
// 			if status != storage.TranscodingStatus {
// 				break
// 			} else {
// 				time.Sleep(3 * time.Second)
// 			}
// 		}
//
// 		if status != storage.TranscodedStatus {
// 			if err := h.storageHelper.Delete(&storage.Object{
// 				Id: fileId,
// 			}); err != nil {
// 				slog.Error("failed to delete transcoded file", slog.Any("err", err))
// 			}
//
// 			if err := h.storageHelper.Delete(&storage.Object{
// 				Id: sourceFileId,
// 			}); err != nil {
// 				slog.Error("failed to delete source file", slog.Any("err", err))
// 			}
// 		}
//
// 		fileRecord, err := h.storageHelper.GetFileRecord(fileId)
// 		if err != nil {
// 			return err
// 		}
//
// 		zipHeaders, err := h.storageHelper.GetZipHeader(fileId)
// 		if err != nil {
// 			return err
// 		}
//
// 		var playlistFileSize int64
// 		var playlistCdnFile *models.CdnFile
// 		for _, file := range zipHeaders.File {
// 			if file.Name == "playlist.m3u8" {
// 				playlistFileSize = int64(file.UncompressedSize64)
// 				playlistCdnFile = models.NewCdnFile(
// 					h.media.UserId,
// 					quality.Id,
// 					fileId,
// 					int64(playlistFileSize),
// 					int64(file.Offset),
// 					1,
// 					models.CdnMediaPlaylistType,
// 				)
// 				h.media.CdnFiles = append(
// 					h.media.CdnFiles,
// 					playlistCdnFile,
// 				)
//
// 				break
// 			}
// 		}
//
// 		h.media.CdnFiles = append(
// 			h.media.CdnFiles,
// 			models.NewCdnFile(
// 				h.media.UserId,
// 				quality.Id,
// 				fileId,
// 				fileRecord.Size-playlistFileSize,
// 				0,
// 				1,
// 				models.CdnMediaContentType,
// 			),
// 		)
//
// 		var frameRate float64
// 		frameData := strings.Split(mediaStream.AvgFrameRate, "/")
// 		if len(frameData) == 1 {
// 			var err error
// 			frameRate, err = strconv.ParseFloat(frameData[0], 64)
// 			if err != nil {
// 				return err
// 			}
// 		} else {
// 			numerator, err := strconv.ParseFloat(frameData[0], 64)
// 			if err != nil {
// 				return err
// 			}
//
// 			denominator, err := strconv.ParseFloat(frameData[1], 64)
// 			if err != nil {
// 				return err
// 			}
//
// 			frameRate = numerator / denominator
// 		}
//
// 		rs := fmt.Sprintf(
// 			"hostUrl/api/media/vod/%s/playlist.m3u8?range=%d,%d&index=1",
// 			quality.Id.String(),
// 			playlistCdnFile.Offset,
// 			playlistCdnFile.Size,
// 		)
//
// 		if haveAudio {
// 			masterM3U8Data += fmt.Sprintf(
// 				"#EXT-X-STREAM-INF:BANDWIDTH=%d,CODECS=\"%s.%s\",BANDWIDTH=%d,RESOLUTION=%dx%d,FRAME-RATE=%v,AUDIO=\"stereo\",SUBTITLES=\"subs\"\n%s\n",
// 				config.Bitrate*1000,
// 				"h264",
// 				"aac",
// 				config.Bitrate*1000,
// 				config.Width,
// 				config.Height,
// 				frameRate,
// 				rs,
// 			)
// 		} else {
// 			masterM3U8Data += fmt.Sprintf(
// 				"#EXT-X-STREAM-INF:BANDWIDTH=%d,CODECS=\"%s\",BANDWIDTH=%d,RESOLUTION=%dx%d,FRAME-RATE=%v,SUBTITLES=\"subs\"\n%s\n",
// 				config.Bitrate*1000,
// 				"h264",
// 				config.Bitrate*1000,
// 				config.Width,
// 				config.Height,
// 				frameRate,
// 				rs,
// 			)
// 		}
//
// 		quality.Status = models.DoneStatus
// 	}
//
// 	if _, err := file.WriteString(masterM3U8Data); err != nil {
// 		return err
// 	}
//
// 	if _, err := file.Seek(0, io.SeekStart); err != nil {
// 		return err
// 	}
//
// 	resp, err = h.storageHelper.Upload(h.media.Id.String(), file)
// 	if err != nil {
// 		return err
// 	}
//
// 	h.media.CdnFiles = append(
// 		h.media.CdnFiles,
// 		models.NewCdnFile(
// 			h.media.UserId,
// 			h.media.Id,
// 			resp.Id,
// 			resp.Size,
// 			resp.Offset,
// 			1,
// 			models.CdnM3u8FileType,
// 		),
// 	)
//
// 	return nil
// }
