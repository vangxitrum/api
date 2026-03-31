package services

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/grafov/m3u8"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/core"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	job_pb "10.0.0.50/tuan.quang.tran/vms-v2/internal/proto/job"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/convert"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/image"
	client "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/job_client"
	mdp_helper "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/mpd_helper"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payment"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/random"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/transcribe_client"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/repositories"
	"10.0.0.50/tuan.quang.tran/vms-v2/rabbitmq"
)

type MediaService struct {
	db                  *gorm.DB
	mediaRepo           models.MediaRepository
	streamRepo          models.StreamRepository
	formatRepo          models.FormatRepository
	partRepo            models.PartRepository
	cdnFileRepo         models.CdnFileRepository
	qualityRepo         models.QualityRepository
	watermarkRepo       models.WatermarkRepository
	userRepo            models.UserRepository
	mediaUsageRepo      models.MediaUsageRepository
	usageRepo           models.UsageRepository
	mediaCaptionRepo    models.MediaCaptionRepository
	mediaChapterRepo    models.MediaChapterRepository
	mediaPlayerRepo     models.PlayerThemeRepository
	thumbnailRepo       models.ThumbnailRepository
	cdnRepo             models.CdnFileRepository
	liveStreamMediaRepo models.LiveStreamMediaRepository

	storagePath string
	outputPath  string
	registerId  string

	userStatusMapping map[uuid.UUID]string
	storageHelper     storage.StorageHelper
	paymentClient     *payment.PaymentClient
	jobClient         *client.JobClient
	transcribeClient  *transcribe_client.TranscribeClient
	thumbnailHelper   *image.ThumbnailHelper

	userConcurrentMediaUploadingMap   map[uuid.UUID]map[uuid.UUID]int
	userConcurrentMediaUploadingLimit int

	cdnHandlerCh, qualityCh, responseCh *rabbitmq.RabbitMQ

	stopWatchMediaCh, stopWatchQualityCh, stopWatchPlaylistCh chan struct{}

	callWebhookCh *rabbitmq.RabbitMQ
}

type MediaInfoRequest struct {
	Id     string `json:"id"`
	Lang   string `json:"lang"`
	Action string `json:"action"`
}

func NewMediaService(
	db *gorm.DB,
	mediaRepo models.MediaRepository,
	streamRepo models.StreamRepository,
	formatRepo models.FormatRepository,
	partRepo models.PartRepository,
	cdnFileRepo models.CdnFileRepository,
	qualityRepo models.QualityRepository,
	watermarkRepo models.WatermarkRepository,
	userRepo models.UserRepository,
	mediaUsageRepo models.MediaUsageRepository,
	usageRepo models.UsageRepository,
	mediaCaptionRepo models.MediaCaptionRepository,
	mediaChapterRepo models.MediaChapterRepository,
	mediaPlayerRepo models.PlayerThemeRepository,
	thumbnailRepo models.ThumbnailRepository,
	cdnRepo models.CdnFileRepository,
	liveStreamMediaRepo models.LiveStreamMediaRepository,

	beUrl string,
	storagePath string,
	outputPath string,
	registerId string,

	userStatusMapping map[uuid.UUID]string,
	storageHelper storage.StorageHelper,
	paymentClient *payment.PaymentClient,
	jobClient *client.JobClient,
	transcribeClient *transcribe_client.TranscribeClient,
	thumbnailHelper *image.ThumbnailHelper,

	userConcurrentMediaUploadingLimit int,

	cdnHandlerCh, qualityCh, responseCh *rabbitmq.RabbitMQ,

	callWebhookWorker *rabbitmq.RabbitMQ,
) *MediaService {
	service := &MediaService{
		db:                  db,
		mediaRepo:           mediaRepo,
		streamRepo:          streamRepo,
		formatRepo:          formatRepo,
		partRepo:            partRepo,
		cdnFileRepo:         cdnFileRepo,
		qualityRepo:         qualityRepo,
		watermarkRepo:       watermarkRepo,
		userRepo:            userRepo,
		mediaUsageRepo:      mediaUsageRepo,
		usageRepo:           usageRepo,
		mediaCaptionRepo:    mediaCaptionRepo,
		mediaChapterRepo:    mediaChapterRepo,
		mediaPlayerRepo:     mediaPlayerRepo,
		thumbnailRepo:       thumbnailRepo,
		cdnRepo:             cdnRepo,
		liveStreamMediaRepo: liveStreamMediaRepo,

		storagePath: storagePath,
		outputPath:  outputPath,
		registerId:  registerId,

		userStatusMapping: userStatusMapping,
		storageHelper:     storageHelper,
		paymentClient:     paymentClient,
		jobClient:         jobClient,
		transcribeClient:  transcribeClient,
		thumbnailHelper:   thumbnailHelper,

		userConcurrentMediaUploadingMap: make(
			map[uuid.UUID]map[uuid.UUID]int,
		),
		userConcurrentMediaUploadingLimit: userConcurrentMediaUploadingLimit,

		cdnHandlerCh: cdnHandlerCh,
		qualityCh:    qualityCh,
		responseCh:   responseCh,

		callWebhookCh: callWebhookWorker,

		stopWatchMediaCh:    make(chan struct{}),
		stopWatchQualityCh:  make(chan struct{}),
		stopWatchPlaylistCh: make(chan struct{}),
	}

	return service
}

func (s *MediaService) newMediaServiceWithTx(tx *gorm.DB) *MediaService {
	return &MediaService{
		mediaRepo:     repositories.MustNewMediaRepository(tx, false),
		streamRepo:    repositories.MustNewStreamRepository(tx, false),
		formatRepo:    repositories.MustNewFormatRepository(tx, false),
		partRepo:      repositories.MustNewPartRepository(tx, false),
		cdnFileRepo:   repositories.MustNewCdnFileRepository(tx, false),
		qualityRepo:   repositories.MustNewQualityRepository(tx, false),
		watermarkRepo: repositories.MustWatermarkRepository(tx, false),
		userRepo:      repositories.MustNewUserRepository(tx, false),
		mediaUsageRepo: repositories.MustNewMediaUsageRepository(
			tx,
			false,
		),
		usageRepo: repositories.MustNewUsageRepository(tx, false),
		mediaCaptionRepo: repositories.MustNewMediaCaptionRepository(
			tx,
			false,
		),
		mediaChapterRepo: repositories.MustNewMediaChapterRepository(
			tx,
			false,
		),
		mediaPlayerRepo: repositories.MustNewPlayerThemeRepository(
			tx,
			false,
		),
		thumbnailRepo:       repositories.MustNewThumbnailRepository(tx, false),
		liveStreamMediaRepo: repositories.NewLiveStreamRepository(tx, false),

		storagePath: s.storagePath,
		outputPath:  s.outputPath,
		registerId:  s.registerId,

		userStatusMapping: s.userStatusMapping,
		storageHelper:     s.storageHelper,
		paymentClient:     s.paymentClient,
		jobClient:         s.jobClient,
		transcribeClient:  s.transcribeClient,
		thumbnailHelper:   s.thumbnailHelper,

		userConcurrentMediaUploadingMap:   s.userConcurrentMediaUploadingMap,
		userConcurrentMediaUploadingLimit: s.userConcurrentMediaUploadingLimit,

		cdnHandlerCh: s.cdnHandlerCh,
		qualityCh:    s.qualityCh,
		responseCh:   s.responseCh,

		callWebhookCh: s.callWebhookCh,

		stopWatchMediaCh:   s.stopWatchMediaCh,
		stopWatchQualityCh: s.stopWatchQualityCh,
	}
}

func (s *MediaService) CreateMediaObject(
	ctx context.Context,
	media *models.Media,
	authInfo models.AuthenticationInfo,
) (*models.Media, error) {
	media, err := s.createMediaObject(ctx, media)
	if err != nil {
		return nil, err
	}

	authInfo.User.MediaQualitiesConfig = media.Qualities
	if err := s.userRepo.UpdateUser(ctx, authInfo.User); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return media, nil
}

func (s *MediaService) CreateMediaObjectLiveStream(
	ctx context.Context,
	media *models.Media,
	liveStreamKeyId uuid.UUID,
) (*models.Media, error) {
	media.Metadata = models.JsonB{
		"live_stream_id": liveStreamKeyId.String(),
	}

	return s.createMediaObject(ctx, media)
}

func (s *MediaService) createMediaObject(
	ctx context.Context,
	media *models.Media,
) (*models.Media, error) {
	if media.Watermark != nil {
		if _, err := s.watermarkRepo.GetUserWatermarkById(ctx, media.UserId, media.Watermark.WatermarkId); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.NewNotFoundError(err)
			}

			return nil, response.NewInternalServerError(err)
		}
	}

	if media.Public == false {
		media.Secret = random.GenerateRandomString(32)
	}

	if player, err := s.mediaPlayerRepo.GetDefaultPlayerTheme(ctx, media.UserId); err != nil {
		return nil, response.NewInternalServerError(err)
	} else {
		if player != nil {
			media.PlayerThemeId = &player.Id
		}
	}

	if err := s.mediaRepo.Create(
		ctx,
		media,
	); err != nil {
		return nil, response.NewInternalServerError(err)
	}
	return media, nil
}

func (s *MediaService) GetMediaList(
	ctx context.Context,
	input models.GetMediaListInput,
	authInfo models.AuthenticationInfo,
) ([]*models.Media, int64, error) {
	media, total, err := s.mediaRepo.GetUserMedias(
		ctx,
		authInfo.User.Id,
		input,
	)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}

	for _, medium := range media {
		for _, caption := range medium.Captions {
			caption.Url = caption.GetUrl(models.CdnCaptionType, medium.Secret)
		}

		for _, chapter := range medium.Chapters {
			chapter.Url = chapter.GetUrl(medium.Secret)
		}
	}

	return media, total, nil
}

func (s *MediaService) GetMediaByIdAndToken(
	ctx context.Context,
	mediaId uuid.UUID,
	token string,
) (*models.Media, error) {
	media, err := s.mediaRepo.GetMediaById(
		ctx,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}

		return nil, response.NewInternalServerError(err)
	}

	if !media.Public && media.Secret != token {
		return nil, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Not found."),
		)
	}

	if media.Status == models.DeletedStatus {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Media is deleted."),
		)
	}

	if media.PlayerThemeId != nil {
		if media.PlayerTheme.Asset.FileId != nil {
			baseUrl := fmt.Sprintf(
				"%s/api/players/%s",
				models.BeUrl,
				media.PlayerThemeId,
			)
			media.PlayerTheme.Asset.LogoImageLink = baseUrl + "/logo"
		}
	}

	for _, chapter := range media.Chapters {
		chapter.Url = chapter.GetUrl(token)
	}

	for _, subtitle := range media.Captions {
		subtitle.Url = subtitle.GetUrl(models.CdnCaptionType, token)
	}

	return media, nil
}

func (s *MediaService) GetMediaDetail(
	ctx context.Context,
	mediaId uuid.UUID,
	authInfo models.AuthenticationInfo,
) (*models.Media, error) {
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		authInfo.User.Id,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}

		return nil, response.NewInternalServerError(err)
	}

	if media.Status == models.DeletedStatus {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Media is deleted.",
		)
	}

	media.Chapters = nil
	media.Captions = nil

	return media, nil
}

func (s *MediaService) GetMediaSource(
	echoCtx echo.Context,
	mediaId uuid.UUID,
	authInfo models.AuthenticationInfo,
) (int64, error) {
	ctx := echoCtx.Request().Context()
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		authInfo.User.Id,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return 0, response.NewNotFoundError(err)
		}
		return 0, response.NewInternalServerError(err)
	}

	if media.Status == models.DeletedStatus {
		return 0, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Media is deleted."),
		)
	}

	allowGetContent, err := s.allowGetContent(ctx, media.UserId)
	if err != nil {
		return 0, response.NewInternalServerError(err)
	}

	if !allowGetContent {
		return 0, response.NewPaymentFailError()
	}

	sourceFiles := make([]*models.CdnFile, 0, len(media.MediaFiles))
	for _, file := range media.MediaFiles {
		if file.File.Type == models.CdnSourceFileType {
			sourceFiles = append(sourceFiles, file.File)
		}
	}

	sort.Slice(
		sourceFiles, func(i, j int) bool {
			return sourceFiles[i].Index < sourceFiles[j].Index
		},
	)

	echoCtx.Response().Header().Set("Content-Type", media.Mimetype)
	echoCtx.Response().
		Header().
		Set("Content-Length", strconv.FormatInt(media.Size, 10))
	echoCtx.Response().
		Header().
		Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", media.Title))

	var totalBytes int64
	for _, file := range sourceFiles {
		reader, err := s.storageHelper.Download(
			ctx,
			&storage.Object{
				Id:     file.Id,
				Size:   file.Size,
				Offset: file.Offset,
			},
		)
		if err != nil {
			return 0, response.NewInternalServerError(err)
		}

		n, err := io.Copy(echoCtx.Response().Writer, reader)
		if err != nil {
			return 0, response.NewInternalServerError(err)
		}

		totalBytes += n
	}

	return totalBytes, nil
}

func (s *MediaService) GetMediaAudio(
	ctx context.Context,
	mediaId uuid.UUID,
) (io.ReadCloser, error) {
	media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if media.Status != models.DoneStatus {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Media is deleted."),
		)
	}

	audioFile, err := os.Open(
		filepath.Join(s.storagePath, media.Id.String(), "audio.mp3"),
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return audioFile, nil
}

func (s *MediaService) GetMediaChapter(
	ctx context.Context,
	mediaId uuid.UUID,
	lan string,
	token string,
) (*models.FileInfo, error) {
	media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if !media.Public && media.Secret != token {
		return nil, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Not found."),
		)
	}

	if !media.IsDone() {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Invalid media."),
		)
	}

	chapter, err := s.mediaChapterRepo.GetMediaChapterByMediaIdAndLanguage(
		ctx,
		mediaId,
		lan,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}

		return nil, response.NewInternalServerError(err)
	}

	if chapter.File == nil {
		return nil, response.NewInternalServerError(
			fmt.Errorf("Chapter file not found."),
		)
	}

	var reader io.Reader
	reader, err = s.storageHelper.Download(
		ctx,
		&storage.Object{
			Id:     chapter.File.File.Id,
			Size:   chapter.File.File.Size,
			Offset: chapter.File.File.Offset,
		},
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return (&models.FileInfoBuilder{}).
		SetMediaId(media.Id).
		SetUserId(media.UserId).
		SetReader(reader).
		SetSize(chapter.File.File.Size).
		Build(ctx), nil
}

func (s *MediaService) GetMediaCaption(
	ctx context.Context,
	mediaId uuid.UUID,
	fileType, lan string,
	token string,
) (*models.FileInfo, error) {
	media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if !media.Public && media.Secret != token {
		return nil, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Not found."),
		)
	}

	if !media.IsDone() {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Invalid media."),
		)
	}

	caption, err := s.mediaCaptionRepo.GetMediaCaptionByMediaIdAndLanguage(
		ctx,
		mediaId,
		lan,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}

		return nil, response.NewInternalServerError(err)
	}

	if caption.File == nil {
		return nil, response.NewInternalServerError(
			fmt.Errorf("Caption file not found."),
		)
	}

	if fileType == models.CdnCaptionM3u8Type {
		mediaDuration, err := strconv.ParseFloat(media.Format.Duration, 64)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		mediaDurationRound := math.Ceil(mediaDuration)
		rs := caption.GetUrl(models.CdnCaptionType, media.Secret)
		data := fmt.Sprintf(
			"#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-PLAYLIST-TYPE:VOD\n#EXT-X-MEDIA-SEQUENCE:1\n#EXT-X-TARGETDURATION:%v\n#EXTINF:%s,\n%s\n#EXT-X-ENDLIST",
			mediaDurationRound,
			media.Format.Duration,
			rs,
		)

		reader := strings.NewReader(data)

		return (&models.FileInfoBuilder{}).
			SetMediaId(media.Id).
			SetUserId(media.UserId).
			SetReader(reader).
			SetSize(int64(len(data))).
			Build(ctx), nil
	} else {
		var reader io.Reader
		reader, err = s.storageHelper.Download(
			ctx,
			&storage.Object{
				Id:     caption.File.File.Id,
				Size:   caption.File.File.Size,
				Offset: caption.File.File.Offset,
			},
		)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		return (&models.FileInfoBuilder{}).
			SetMediaId(media.Id).
			SetUserId(media.UserId).
			SetReader(reader).
			SetSize(caption.File.File.Size).
			Build(ctx), nil
	}
}

func (s *MediaService) GetMediaMp4(
	echoCtx echo.Context,
	mediaId uuid.UUID,
	token string,
) error {
	ctx := echoCtx.Request().Context()
	media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if !media.Public && media.Secret != token {
		return response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Not found."),
		)
	}

	if media.Status == models.DeletedStatus {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Media is deleted."),
		)
	}

	allowGetContent, err := s.allowGetContent(ctx, media.UserId)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if !allowGetContent {
		return response.NewPaymentFailError()
	}

	var cdnFiles []*models.CdnFile
	if media.IsMp4 {
		cdnFiles = make([]*models.CdnFile, 0, len(media.MediaFiles))
		for _, file := range media.MediaFiles {
			if file.File.Type == models.CdnSourceFileType {
				cdnFiles = append(cdnFiles, file.File)
			}
		}
	} else {
		mp4Quality, err := s.qualityRepo.GetMp4QualityByMediaId(ctx, mediaId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return response.NewNotFoundError(err)
			}

			return response.NewInternalServerError(err)
		}

		cdnFiles = make([]*models.CdnFile, 0, len(mp4Quality.Files))
		for _, file := range mp4Quality.Files {
			cdnFiles = append(cdnFiles, file.File)
		}
	}

	sort.Slice(
		cdnFiles, func(i, j int) bool {
			return cdnFiles[i].Index < cdnFiles[j].Index
		},
	)

	var totalSize int64
	for _, file := range cdnFiles {
		totalSize += file.Size
	}

	echoCtx.Response().Header().Set("Content-Type", media.Mimetype)
	echoCtx.Response().
		Header().
		Set("Content-Length", strconv.FormatInt(totalSize, 10))
	echoCtx.Response().
		Header().
		Set("Content-Disposition", "attachment; filename=media.mp4")

	var totalBytes int64

	defer func() {
		if err := s.usageRepo.CreateLog(
			ctx,
			(&models.UsageLogBuilder{}).
				SetUserId(media.UserId).
				SetDelivery(totalSize).
				SetIsUserCost(true).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx,
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}()

	for _, file := range cdnFiles {
		reader, err := s.storageHelper.Download(
			ctx,
			&storage.Object{
				Id:     file.Id,
				Size:   file.Size,
				Offset: file.Offset,
			},
		)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		n, err := io.Copy(echoCtx.Response().Writer, reader)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		totalBytes += n
	}

	return nil
}

type SaveFunc func(ctx context.Context, bytes []byte, closeReader bool) error

func (c *MediaService) GetSaveJobResourceFunc(
	ctx context.Context,
	mediaId uuid.UUID,
) (SaveFunc, error) {
	liveStreamMedia, err := c.liveStreamMediaRepo.GetById(ctx, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Newf(codes.NotFound, "media not found").Err()
		}

		return nil, status.Newf(codes.Internal, "get media error, err: %v", err).
			Err()
	}

	inputPath := liveStreamMedia.Media.GetSourceTempPath(c.storagePath)
	if err := os.MkdirAll(filepath.Dir(inputPath), os.ModePerm); err != nil {
		return nil, status.Newf(codes.Internal, "create folder error, err: %v", err).
			Err()
	}

	file, err := os.Create(inputPath)
	if err != nil {
		return nil, status.Newf(codes.Internal, "create file error, err: %v", err).
			Err()
	}

	bufferWriter := bufio.NewWriter(file)
	return func(ctx context.Context, bytes []byte, closeReader bool) error {
		if closeReader {
			defer file.Close()
			if err := bufferWriter.Flush(); err != nil {
				return fmt.Errorf("flush file: %w", err)
			}

			file.Close()
			if err := os.Rename(inputPath, liveStreamMedia.Media.GetSourcePath(c.storagePath)); err != nil {
				return fmt.Errorf("rename file: %w", err)
			}

			return nil
		}

		if len(bytes) == 0 {
			file.Close()
			return status.Newf(codes.Internal, "file is empty").Err()
		}

		_, err := bufferWriter.Write(bytes)
		if err != nil {
			defer file.Close()
			return status.Newf(codes.Internal, "write file error, err: %v", err).
				Err()
		}

		return nil
	}, nil
}

func (s *MediaService) GetMediaDashManifest(
	ctx context.Context,
	mediaId uuid.UUID,
	token string,
) (*models.FileInfo, error) {
	media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if !media.Public && media.Secret != token {
		return nil, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Not found."),
		)
	}

	if media.Status == models.DeletedStatus {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Media is deleted.",
		)
	}

	allowGetContent, err := s.allowGetContent(ctx, media.UserId)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if !allowGetContent {
		return nil, response.NewPaymentFailError()
	}

	sort.Slice(
		media.MediaQualities, func(i, j int) bool {
			if media.MediaQualities[i].Name == "" {
				return false
			}

			if media.MediaQualities[j].Name == "" {
				return true
			}

			x, err := strconv.Atoi(
				strings.TrimSuffix(media.MediaQualities[i].Name, "p"),
			)
			if err != nil {
				return false
			}

			y, err := strconv.Atoi(
				strings.TrimSuffix(media.MediaQualities[j].Name, "p"),
			)
			if err != nil {
				return true
			}

			return x < y
		},
	)

	reader, err := s.generateMdpManifest(ctx, media)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return (&models.FileInfoBuilder{}).
		SetMediaId(media.Id).
		SetUserId(media.UserId).
		SetReader(reader).
		SetSize(reader.Size()).
		Build(ctx), nil
}

func (s *MediaService) GetDemoManifest(
	ctx context.Context,
	mediaType string,
) (*models.FileInfo, error) {
	media, err := s.mediaRepo.GetMediaById(ctx, models.DemoVideoId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if media.Status == models.DeletedStatus {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Media is deleted.",
		)
	}

	allowGetContent, err := s.allowGetContent(ctx, media.UserId)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if !allowGetContent {
		return nil, response.NewPaymentFailError()
	}

	sort.Slice(
		media.MediaQualities, func(i, j int) bool {
			if media.MediaQualities[i].Name == "" {
				return false
			}

			if media.MediaQualities[j].Name == "" {
				return true
			}

			x, err := strconv.Atoi(
				strings.TrimSuffix(media.MediaQualities[i].Name, "p"),
			)
			if err != nil {
				return false
			}

			y, err := strconv.Atoi(
				strings.TrimSuffix(media.MediaQualities[j].Name, "p"),
			)
			if err != nil {
				return true
			}

			return x < y
		},
	)

	var reader *bytes.Reader
	switch mediaType {
	case models.HlsQualityType:
		reader, err = s.generateM3U8(media)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}
	case models.DashQualityType:
		reader, err = s.generateMdpManifest(ctx, media)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}
	}

	return (&models.FileInfoBuilder{}).
		SetMediaId(media.Id).
		SetUserId(media.UserId).
		SetReader(reader).
		SetSize(reader.Size()).
		Build(ctx), nil
}

func (s *MediaService) GetMediaM3U8(
	ctx context.Context,
	mediaId uuid.UUID,
	token string,
) (*models.FileInfo, error) {
	media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if !media.Public && media.Secret != token {
		return nil, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Not found."),
		)
	}

	if media.Status == models.DeletedStatus {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Media is deleted.",
		)
	}

	allowGetContent, err := s.allowGetContent(ctx, media.UserId)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if !allowGetContent {
		return nil, response.NewPaymentFailError()
	}

	sort.Slice(
		media.MediaQualities, func(i, j int) bool {
			if media.MediaQualities[i].Name == "" {
				return false
			}

			if media.MediaQualities[j].Name == "" {
				return true
			}

			x, err := strconv.Atoi(
				strings.TrimSuffix(media.MediaQualities[i].Name, "p"),
			)
			if err != nil {
				return false
			}

			y, err := strconv.Atoi(
				strings.TrimSuffix(media.MediaQualities[j].Name, "p"),
			)
			if err != nil {
				return true
			}

			return x < y
		},
	)

	reader, err := s.generateM3U8(media)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return (&models.FileInfoBuilder{}).
		SetMediaId(media.Id).
		SetUserId(media.UserId).
		SetReader(reader).
		SetSize(reader.Size()).
		Build(ctx), nil
}

func (s *MediaService) generateMdpManifest(
	ctx context.Context,
	media *models.Media,
) (*bytes.Reader, error) {
	var masterPl *mdp_helper.MPD
	var index int
	for _, q := range media.MediaQualities {
		if !(q.Status == models.DoneStatus && q.Type == models.DashQualityType) {
			continue
		}

		var playlistInfo *models.CdnFile
		for _, file := range q.Files {
			if file.File.Type == models.CdnVideoPlaylistType {
				playlistInfo = file.File
				break
			}
		}

		if playlistInfo == nil {
			continue
		}

		plFile, err := s.storageHelper.Download(
			ctx,
			&storage.Object{
				Id:     playlistInfo.Id,
				Size:   playlistInfo.Size,
				Offset: playlistInfo.Offset,
			},
		)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		defer func() {
			if closer, ok := plFile.(io.Closer); ok {
				_ = closer.Close()
			}
		}()

		dt, err := io.ReadAll(plFile)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		plData, err := mdp_helper.Unmarshal(dt)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		if masterPl == nil {
			masterPl = plData
			for _, profile := range plData.Periods {
				for _, adaptationSet := range profile.AdaptationSets {
					for _, representation := range adaptationSet.Representations {
						if adaptationSet.ContentType == models.VideoMediaType &&
							q.VideoConfig.Codec == models.H265Codec {
							representation.Codecs = models.H265VideoMpdCodec
						}
						if representation.SegmentList != nil {
							representation.SegmentList.Initialization.SourceURL = fmt.Sprintf(
								models.SegmentUrlFormat,
								models.BeUrl,
								q.Id,
								representation.SegmentList.Initialization.SourceURL,
							)
							for _, segment := range representation.SegmentList.SegmentURLs {
								segment.Media = fmt.Sprintf(
									models.SegmentUrlFormat,
									models.BeUrl,
									q.Id,
									segment.Media,
								)
							}
						}
					}
					index++
				}
			}

			continue
		}

		for _, profile := range plData.Periods {
			for _, adaptationSet := range profile.AdaptationSets {
				for _, representation := range adaptationSet.Representations {
					if representation.SegmentList != nil {
						representation.SegmentList.Initialization.SourceURL = fmt.Sprintf(
							models.SegmentUrlFormat,
							models.BeUrl,
							q.Id,
							representation.SegmentList.Initialization.SourceURL,
						)
						for _, segment := range representation.SegmentList.SegmentURLs {
							segment.Media = fmt.Sprintf(
								models.SegmentUrlFormat,
								models.BeUrl,
								q.Id,
								segment.Media,
							)
						}
					}
				}

				var flag bool
				for _, masterAdaptationSet := range masterPl.Periods[0].AdaptationSets {
					if masterAdaptationSet.ContentType == adaptationSet.ContentType {
						flag = true
						index = len(masterAdaptationSet.Representations)
						for _, representation := range adaptationSet.Representations {
							representation.ID = fmt.Sprint(index)
							masterAdaptationSet.Representations = append(
								masterAdaptationSet.Representations,
								representation,
							)
						}
					}
				}

				if !flag {
					adaptationSet.ID = fmt.Sprint(
						len(masterPl.Periods[0].AdaptationSets),
					)
					masterPl.Periods[0].AdaptationSets = append(
						masterPl.Periods[0].AdaptationSets,
						adaptationSet,
					)
				}
			}
		}
	}

	for _, q := range media.MediaQualities {
		if !(q.Status == models.DoneStatus && q.Type == models.DashQualityType) {
			continue
		}

		var playlistInfo *models.CdnFile
		for _, file := range q.Files {
			if file.File.Type == models.CdnAudioPlaylistType {
				playlistInfo = file.File
				break
			}
		}

		if playlistInfo == nil {
			continue
		}

		plFile, err := s.storageHelper.Download(
			ctx,
			&storage.Object{
				Id:     playlistInfo.Id,
				Size:   playlistInfo.Size,
				Offset: playlistInfo.Offset,
			},
		)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		defer func() {
			if closer, ok := plFile.(io.Closer); ok {
				_ = closer.Close()
			}
		}()

		dt, err := io.ReadAll(plFile)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		plData, err := mdp_helper.Unmarshal(dt)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		if masterPl == nil {
			masterPl = plData
			for _, profile := range plData.Periods {
				for _, adaptationSet := range profile.AdaptationSets {
					for _, representation := range adaptationSet.Representations {
						if representation.SegmentList != nil {
							representation.SegmentList.Initialization.SourceURL = fmt.Sprintf(
								models.SegmentUrlFormat,
								models.BeUrl,
								q.Id,
								representation.SegmentList.Initialization.SourceURL,
							)
							for _, segment := range representation.SegmentList.SegmentURLs {
								segment.Media = fmt.Sprintf(
									models.SegmentUrlFormat,
									models.BeUrl,
									q.Id,
									segment.Media,
								)
							}
						}
					}
					index++
				}
			}

			continue
		}

		for _, profile := range plData.Periods {
			for _, adaptationSet := range profile.AdaptationSets {
				for _, representation := range adaptationSet.Representations {
					if representation.SegmentList != nil {
						representation.SegmentList.Initialization.SourceURL = fmt.Sprintf(
							models.SegmentUrlFormat,
							models.BeUrl,
							q.Id,
							representation.SegmentList.Initialization.SourceURL,
						)
						for _, segment := range representation.SegmentList.SegmentURLs {
							segment.Media = fmt.Sprintf(
								models.SegmentUrlFormat,
								models.BeUrl,
								q.Id,
								segment.Media,
							)
						}
					}
				}

				adaptationSet.ID = fmt.Sprint(index)
				masterPl.Periods[0].AdaptationSets = append(
					masterPl.Periods[0].AdaptationSets,
					adaptationSet,
				)

				index++
			}
		}
	}

	if masterPl != nil {
		for _, subtitle := range media.Captions {
			adaptationSet := &mdp_helper.AdaptationSet{
				ID:               fmt.Sprint(index),
				ContentType:      "text",
				SegmentAlignment: true,
				StartWithSAP:     1,
				Lang:             subtitle.Language,
				Representations: []*mdp_helper.Representation{
					{
						ID:        fmt.Sprint(index),
						MimeType:  "text/vtt",
						Codecs:    "wvtt",
						Bandwidth: 5367,
						BaseUrl: subtitle.GetUrl(
							models.CdnCaptionType,
							media.Secret,
						),
					},
				},
			}

			adaptationSet.ID = fmt.Sprint(index)
			masterPl.Periods[0].AdaptationSets = append(
				masterPl.Periods[0].AdaptationSets,
				adaptationSet,
			)

			index++
		}
	}

	bts, err := mdp_helper.Marshal(masterPl)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return bytes.NewReader(bts), nil
}

func (s *MediaService) generateM3U8(
	media *models.Media,
) (*bytes.Reader, error) {
	var firstVariant *m3u8.Variant
	masterPl := m3u8.NewMasterPlaylist()
	masterPl.SetVersion(3)
	alternatives := make([]*m3u8.Alternative, 0)
	for _, subtitle := range media.Captions {
		alternatives = append(alternatives, &m3u8.Alternative{
			URI:      subtitle.GetUrl(models.CdnCaptionM3u8Type, media.Secret),
			Type:     "SUBTITLES",
			GroupId:  "subs",
			Name:     subtitle.Language,
			Language: subtitle.Language,
			Default:  subtitle.IsDefault,
		})
	}

	switch media.Type {
	case models.VideoType:
		var haveSharedAudio bool
		audioMapping := make(map[uuid.UUID]*models.MediaQuality)
		for _, q := range media.MediaQualities {
			if !(q.Status == models.DoneStatus && q.Type == models.HlsQualityType) {
				continue
			}

			if q.AudioConfig != nil {
				if q.VideoConfig == nil {
					haveSharedAudio = true
				}

				audioMapping[q.Id] = q
			}
		}

		var sharedDefault bool = true
		audioNameMapping := make(map[string]int)
		for _, q := range audioMapping {
			if !(q.Status == models.DoneStatus && q.Type == models.HlsQualityType) {
				continue
			}

			var playlistInfo *models.CdnFile
			for _, file := range q.Files {
				if file.File.Type == models.CdnAudioPlaylistType {
					playlistInfo = file.File
					break
				}
			}

			if playlistInfo == nil {
				return nil, response.NewInternalServerError(
					fmt.Errorf("Playlist file not found for quality %s", q.Id),
				)
			}

			groupId := "audio"
			isDefault := sharedDefault
			if q.VideoConfig != nil {
				groupId = q.Id.String()
				isDefault = true
			}

			audioName := fmt.Sprintf(
				"audio-%dKbs",
				convert.ToKbs(q.AudioConfig.Bitrate),
			)
			if index, ok := audioNameMapping[audioName]; ok {
				if index > 0 {
					audioNameMapping[audioName] = index + 1
					audioName = fmt.Sprintf("%s-%d", audioName, index-1)
				}
			}

			alternatives = append(alternatives, &m3u8.Alternative{
				URI: fmt.Sprintf(
					models.PlaylistUrlFormat,
					models.BeUrl,
					q.Id,
					playlistInfo.Offset,
					playlistInfo.Size,
					models.AudioType,
				),
				Type:     "AUDIO",
				GroupId:  groupId,
				Name:     audioName,
				Language: q.AudioConfig.Language,
				Default:  isDefault,
			})
			if sharedDefault {
				sharedDefault = false
			}
		}

		for _, q := range media.MediaQualities {
			if !(q.Status == models.DoneStatus && q.Type == models.HlsQualityType && q.VideoConfig != nil) {
				continue
			}

			var variant *m3u8.Variant
			var playlistInfo *models.CdnFile
			for _, file := range q.Files {
				if file.File.Type == models.CdnVideoPlaylistType {
					playlistInfo = file.File
					break
				}
			}

			if playlistInfo == nil {
				return nil, response.NewInternalServerError(
					fmt.Errorf("Playlist file not found for quality %s", q.Id),
				)
			}

			variant = &m3u8.Variant{}
			variant.Codecs = q.VideoCodec
			variant.Name = q.Resolution
			variant.URI = fmt.Sprintf(
				models.PlaylistUrlFormat,
				models.BeUrl,
				q.Id,
				playlistInfo.Offset,
				playlistInfo.Size,
				models.VideoType,
			)
			variant.VariantParams.Resolution = fmt.Sprintf(
				"%dx%d",
				q.VideoConfig.Width,
				q.VideoConfig.Height,
			)

			if len(media.Captions) > 0 {
				variant.Subtitles = "subs"
			}

			variant.Bandwidth = uint32(q.Bandwidth)
			if q.AudioConfig != nil && q.AudioPlaylistId != "" {
				variant.Codecs = fmt.Sprintf(
					"%s,%s",
					q.VideoCodec,
					q.AudioCodec,
				)
				variant.VariantParams.Audio = q.Id.String()
			} else if haveSharedAudio {
				variant.Codecs = fmt.Sprintf("%s,%s", q.VideoCodec, "mp4a.40.2")
				variant.VariantParams.Audio = "audio"
			}

			masterPl.Variants = append(masterPl.Variants, variant)
			if firstVariant == nil {
				firstVariant = variant
			}
		}

		if firstVariant != nil {
			firstVariant.VariantParams.Alternatives = alternatives
		}
	case models.AudioType:
		for _, q := range media.MediaQualities {
			if !(q.Status == models.DoneStatus && q.Type == models.HlsQualityType) {
				continue
			}

			var variant *m3u8.Variant
			var playlistInfo *models.CdnFile
			for _, file := range q.Files {
				if file.File.Type == models.CdnAudioPlaylistType {
					playlistInfo = file.File
					break
				}
			}

			if playlistInfo == nil {
				return nil, response.NewInternalServerError(
					fmt.Errorf("Playlist file not found for quality %s", q.Id),
				)
			}

			variant = &m3u8.Variant{}
			variant.Codecs = q.AudioCodec
			variant.Name = q.Resolution
			variant.URI = fmt.Sprintf(
				models.PlaylistUrlFormat,
				models.BeUrl,
				q.Id,
				playlistInfo.Offset,
				playlistInfo.Size,
				media.Type,
			)

			if len(media.Captions) > 0 {
				variant.Subtitles = "subs"
			}

			masterPl.Variants = append(masterPl.Variants, variant)
			if firstVariant == nil {
				firstVariant = variant
			}
		}

		if firstVariant != nil {
			firstVariant.VariantParams.Alternatives = alternatives
		}
	}

	return bytes.NewReader(masterPl.Encode().Bytes()), nil
}

func (s *MediaService) GetDemoMedia(
	ctx context.Context,
) (*models.Media, error) {
	media, err := s.mediaRepo.GetMediaById(
		ctx,
		models.DemoVideoId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}

		return nil, response.NewInternalServerError(err)
	}

	var token string
	if media.Status == models.DeletedStatus {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Media is deleted."),
		)
	}

	if media.PlayerThemeId != nil {
		if media.PlayerTheme.Asset.FileId != nil {
			baseUrl := fmt.Sprintf(
				"%s/api/players/%s",
				models.BeUrl,
				media.PlayerThemeId,
			)
			media.PlayerTheme.Asset.LogoImageLink = baseUrl + "/logo"
		}
	}

	for _, chapter := range media.Chapters {
		chapter.Url = chapter.GetUrl(token)
	}

	for _, subtitle := range media.Captions {
		subtitle.Url = subtitle.GetUrl(models.CdnCaptionType, token)
	}

	return media, nil
}

func (s *MediaService) GetMediaThumbnail(
	ctx context.Context,
	mediaId uuid.UUID,
	resolution string,
	token string,
) (*models.FileInfo, error) {
	media, err := s.mediaRepo.GetMediaById(
		ctx,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if !media.Public && media.Secret != token {
		return nil, response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Not found."),
		)
	}

	if media.Status == models.DeletedStatus {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Media is deleted",
		)
	}

	if media.MediaThumbnail == nil {
		return nil, response.NewInternalServerError(
			fmt.Errorf("Thumbnail not found."),
		)
	}

	var thumbnailResolution *models.ThumbnailResolution
	for _, res := range media.MediaThumbnail.Thumbnail.Resolutions {
		if res.Resolution == resolution {
			thumbnailResolution = res
			break
		}
	}

	if thumbnailResolution == nil {
		return nil, response.NewNotFoundError(
			fmt.Errorf("Thumbnail resolution not found."),
		)
	}

	redirectUrl, expiredAt, err := s.storageHelper.GetLink(
		ctx,
		&storage.Object{
			Id:     media.MediaThumbnail.Thumbnail.File.FileId,
			Size:   thumbnailResolution.Size,
			Offset: thumbnailResolution.Offset,
		},
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	var reader io.Reader
	if redirectUrl == "" {
		reader, err = s.storageHelper.Download(
			ctx,
			&storage.Object{
				Id:     media.MediaThumbnail.Thumbnail.File.FileId,
				Size:   thumbnailResolution.Size,
				Offset: thumbnailResolution.Offset,
			},
		)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}
	}

	return (&models.FileInfoBuilder{}).
		SetMediaId(media.Id).
		SetUserId(media.UserId).
		SetRedirectUrl(redirectUrl).
		SetReader(reader).
		SetExpiredAt(expiredAt).
		SetSize(thumbnailResolution.Size).
		Build(ctx), nil
}

func (s *MediaService) GetMediaContent(
	ctx context.Context,
	belongsToId uuid.UUID,
	offset, size int64,
	index int,
	filename, contentType string,
) (*models.FileInfo, error) {
	var fileType string
	if strings.Contains(filename, ".m3u8") {
		if contentType == models.VideoType {
			fileType = models.CdnVideoPlaylistType
		} else if contentType == models.AudioType {
			fileType = models.CdnAudioPlaylistType
		}
	} else {
		if contentType == models.VideoType {
			fileType = models.CdnVideoContentType
		} else if contentType == models.AudioType {
			fileType = models.CdnAudioContentType
		}
	}

	quality, err := s.qualityRepo.GetQualityById(ctx, belongsToId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}

		return nil, response.NewInternalServerError(err)
	}

	allowGetContent, err := s.allowGetContent(ctx, quality.Media.UserId)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if !allowGetContent {
		return nil, response.NewPaymentFailError()
	}

	var fileInfo *models.CdnFile

	if len(quality.Files) == 0 {
		return nil, response.NewInternalServerError(
			fmt.Errorf("File not found."),
		)
	}

	for _, file := range quality.Files {
		if file.File.Type == fileType && file.File.Index == index {
			fileInfo = file.File
			break
		}
	}

	if fileInfo == nil {
		return nil, response.NewInternalServerError(errors.New("file not found"))
	}

	if fileType != models.CdnVideoPlaylistType &&
		fileType != models.CdnAudioPlaylistType {
		redirectUrl, expiredAt, err := s.storageHelper.GetLink(
			ctx,
			&storage.Object{
				Id:     fileInfo.Id,
				Size:   size,
				Offset: offset,
			},
		)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		var mimeType string
		if contentType == models.VideoType {
			if quality.ContainerType == models.Mp4ContainerType {
				mimeType = "video/mp4"
			} else if quality.ContainerType == models.MpegtsContainerType {
				mimeType = "video/mp2t"
			}
		} else if contentType == models.AudioType {
			mimeType = "audio/mp4"
		}

		var reader io.Reader
		if redirectUrl == "" {
			reader, err = s.storageHelper.Download(
				ctx,
				&storage.Object{
					Id:     fileInfo.Id,
					Size:   size,
					Offset: offset,
				},
			)
			if err != nil {
				return nil, response.NewInternalServerError(err)
			}

			return (&models.FileInfoBuilder{}).
				SetMediaId(quality.MediaId).
				SetUserId(fileInfo.CreatedBy).
				SetReader(reader).
				SetSize(size).
				SetMimeType(mimeType).
				Build(ctx), nil
		}

		return (&models.FileInfoBuilder{}).
			SetMediaId(quality.MediaId).
			SetUserId(fileInfo.CreatedBy).
			SetSize(size).
			SetMimeType(mimeType).
			SetRedirectUrl(redirectUrl).
			SetExpiredAt(expiredAt).
			Build(ctx), nil
	} else {
		reader, err := s.storageHelper.Download(
			ctx,
			&storage.Object{
				Id:     fileInfo.Id,
				Size:   size,
				Offset: offset,
			},
		)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		buffer := new(bytes.Buffer)
		if _, err := io.Copy(buffer, reader); err != nil {
			return nil, response.NewInternalServerError(err)
		}

		playlist, _, err := m3u8.DecodeFrom(buffer, true)
		if err != nil {
			return nil, response.NewInternalServerError(err)
		}

		playlistData, ok := playlist.(*m3u8.MediaPlaylist)
		if !ok {
			return nil, response.NewInternalServerError(
				fmt.Errorf("Invalid playlist."),
			)
		}

		if playlistData.Map != nil {
			playlistData.Map.URI = fmt.Sprintf(
				models.SegmentUrlFormat,
				models.BeUrl,
				quality.Id.String(),
				playlistData.Map.URI,
			)
		}

		for _, segment := range playlistData.Segments {
			if segment == nil {
				break
			}

			segment.URI = fmt.Sprintf(
				models.SegmentUrlFormat,
				models.BeUrl,
				quality.Id.String(),
				segment.URI,
			)

			// NOTE: we will add segment extension
			// for older videos for new player compatible
			var ext string
			switch quality.ContainerType {
			case models.MpegtsContainerType:
				ext = ".ts"
			case models.Mp4ContainerType:
				ext = ".m4s"
			}

			parts := strings.Split(segment.URI, "?")
			extRegex := regexp.MustCompile(`\.[a-zA-Z0-9]+$`)
			if !extRegex.MatchString(parts[0]) {
				parts[0] = parts[0] + ext
				segment.URI = strings.Join(parts, "?")
			}
		}

		playlistDataBytes := playlistData.Encode().Bytes()
		newReader := bytes.NewReader(playlistDataBytes)

		return (&models.FileInfoBuilder{}).
			SetMediaId(quality.MediaId).
			SetUserId(fileInfo.CreatedBy).
			SetReader(newReader).
			SetSize(int64(len(playlistDataBytes))).
			SetMimeType("application/vnd.apple.mpegurl").
			Build(ctx), nil
	}
}

func (s *MediaService) UpdateMediaInfo(
	ctx context.Context,
	mediaId uuid.UUID,
	input models.UpdateMediaInfoInput,
	authInfo models.AuthenticationInfo,
) error {
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		authInfo.User.Id,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if input.Title != nil {
		media.Title = *input.Title
	}

	if input.Description != nil {
		media.Description = *input.Description
	}

	if input.IsPublic != nil {
		media.Public = *input.IsPublic
		if !media.Public {
			media.Secret = random.GenerateRandomString(16)
		} else {
			media.Secret = ""
		}
	}

	if input.PlayerId != nil {
		_, err := s.mediaPlayerRepo.GetPlayerThemeById(ctx, *input.PlayerId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return response.NewNotFoundError(err)
			}

			return response.NewInternalServerError(err)
		}

		media.PlayerThemeId = input.PlayerId
	}

	if input.Tags != nil {
		var tags string
		if len(input.Tags) > 0 {
			tags = strings.Join(input.Tags, ",")
		}

		media.Tags = tags
	}

	if input.Metadata != nil {
		media.Metadata = make(map[string]any)
		for _, data := range input.Metadata {
			media.Metadata[data.Key] = data.Value
		}
	}

	if err := s.mediaRepo.UpdateMedia(
		ctx,
		media,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *MediaService) CalculateMediaCost(
	ctx context.Context,
	mediaDuration float64,
	mediaType string,
	qualities []string,
	authInfo models.AuthenticationInfo,
) (decimal.Decimal, bool, error) {
	totalCost := decimal.Zero
	for _, quality := range qualities {
		if mediaType == models.AudioType {
			quality = models.AudioType
		}

		transcodeCost, err := s.calculateMediaCost(quality, mediaDuration)
		if err != nil {
			return decimal.Zero, false, response.NewInternalServerError(err)
		}

		totalCost = totalCost.Add(transcodeCost)
	}

	wallet, err := s.paymentClient.GetWalletByUserId(ctx, authInfo.User.Id)
	if err != nil {
		return decimal.Zero, false, response.NewInternalServerError(err)
	}

	return totalCost, wallet.Balance.GreaterThan(totalCost), nil
}

func (s *MediaService) UpdateMediaQualities(
	ctx context.Context,
	mediaId uuid.UUID,
	newQualities []*models.QualityConfig,
	authInfo models.AuthenticationInfo,
) error {
	media, err := s.mediaRepo.GetUserMediaById(ctx, authInfo.User.Id, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if err := s.qualityRepo.DeleteQualitiesByMediaId(ctx, media.Id); err != nil {
		return response.NewInternalServerError(err)
	}

	mediaQualities := make([]*models.MediaQuality, 0, len(newQualities))
	for _, config := range newQualities {
		mediaQualities = append(
			mediaQualities,
			models.NewQuality(media.Id, config),
		)
	}

	media.MediaQualities = mediaQualities

	if err := s.mediaRepo.UpdateMedia(ctx, media); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *MediaService) UpdateMediaSetting(
	ctx context.Context,
	mediaId uuid.UUID,
	isPublic bool,
	authInfo models.AuthenticationInfo,
) error {
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		authInfo.User.Id,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if media.Public == isPublic {
		return nil
	}

	media.Public = isPublic
	if !isPublic {
		media.Secret = random.GenerateRandomString(16)
	} else {
		media.Secret = ""
	}

	if err := s.mediaRepo.UpdateMedia(
		ctx,
		media,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *MediaService) DeleteUserMedias(
	ctx context.Context,
	user *models.User,
) error {
	media, err := s.mediaRepo.GetAllUserMedias(ctx, user.Id)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	for _, media := range media {
		if media.Status != models.DeletedStatus &&
			media.Status != models.DeletingStatus &&
			media.Status != models.TranscodingStatus {
			if err := s.DeleteMedia(
				ctx, media.Id, models.AuthenticationInfo{
					User: user,
				},
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *MediaService) DeleteMedia(
	ctx context.Context,
	mediaId uuid.UUID,
	authInfo models.AuthenticationInfo,
) error {
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		authInfo.User.Id,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if media.Status == models.DeletedStatus {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("media is already deleted."),
		)
	}

	if media.IsTranscoding() {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("media is being processed."),
		)
	}

	mediaStatus := media.Status
	media.Status = models.DeletingStatus
	if err := s.mediaRepo.UpdateMedia(ctx, media); err != nil {
		return response.NewInternalServerError(err)
	}

	switch mediaStatus {
	case models.NewStatus, models.HiddenStatus:
		if err := s.deleteMediaResource(media); err != nil {
			return response.NewInternalServerError(err)
		}
	default:
		if err := s.deleteMediaResource(media); err != nil {
			return response.NewInternalServerError(err)
		}

		if err := s.deleteMediaTranscodingContent(ctx, media); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	if media.Watermark != nil {
		if err := s.watermarkRepo.DeleteMediaWatermarkById(
			ctx,
			media.Watermark.Id,
		); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	media.MediaQualities = nil
	media.Streams = nil
	media.Chapters = nil
	media.Captions = nil
	media.Format = nil
	media.MediaThumbnail = nil
	media.MediaFiles = nil
	media.Status = models.DeletedStatus
	if err := s.mediaRepo.UpdateMedia(ctx, media); err != nil {
		return err
	}

	if media.PlayerThemeId != nil {
		if err := s.mediaRepo.DeleteInactiveMediaPlayerTheme(
			ctx,
			media.Id,
		); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	if userMap, ok := s.userConcurrentMediaUploadingMap[media.UserId]; ok {
		delete(userMap, mediaId)
	}

	return nil
}

func (s *MediaService) DeletePendingMedia(
	ctx context.Context,
) error {
	if err := func() error {
		media, err := s.mediaRepo.GetMediasByStatus(ctx, models.NewStatus)
		if err != nil {
			return err
		}

		for _, media := range media {
			if time.Now().UTC().Sub(media.CreatedAt) > models.MaxMediaPendingTime {
				if err := s.deleteMediaResource(media); err != nil {
					return err
				}

				if err := s.mediaRepo.UpdateMediaStatusById(ctx, media.Id, models.DeletedStatus); err != nil {
					return err
				}

			}
		}

		return nil
	}(); err != nil {
		slog.ErrorContext(
			ctx,
			"Delete pending media error",
			slog.Any("err", err),
		)
	}

	if err := filepath.WalkDir(s.storagePath, func(path string, info os.DirEntry, err error) error {
		if info.IsDir() {
			mediaId, err := uuid.Parse(info.Name())
			if err != nil {
				return nil
			}

			media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					_, newErr := func() (bool, error) {
						stat, err := os.Stat(path)
						if err != nil {
							return false, err
						}

						if stat.ModTime().Add(models.MaxNotFoundPendingTime).Before(time.Now().UTC()) {
							if err := os.RemoveAll(path); err != nil {
								return false, err
							}

							return true, nil
						}

						return false, nil
					}()
					if newErr != nil {
						slog.Error("delete pending media ", slog.Any("err", newErr), slog.Any("mediaId", mediaId))
					}

					return nil
				} else {
					return err
				}
			}

			if media == nil || media.Status == models.DoneStatus || media.Status == models.DeletedStatus || media.Status == models.FailStatus {
				if media != nil && media.Status == models.DoneStatus {
					for _, quality := range media.MediaQualities {
						if quality.Status == models.NewStatus {
							return nil
						}
					}
				}

				if err := os.RemoveAll(path); err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		slog.ErrorContext(ctx, "Delete pending media error", slog.Any("err", err))
	}

	if err := filepath.WalkDir(
		s.outputPath, func(path string, info os.DirEntry, err error) error {
			if info.IsDir() {
				mediaId, err := uuid.Parse(info.Name())
				if err != nil {
					return nil
				}

				media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
				if err != nil {
					if err == gorm.ErrRecordNotFound {
						return nil
					} else {
						return err
					}
				}

				if media == nil || media.Status == models.DoneStatus || media.Status == models.DeletedStatus || media.Status == models.FailStatus {
					if err := os.RemoveAll(path); err != nil {
						return err
					}
				}
			}

			return nil
		},
	); err != nil {
		slog.ErrorContext(ctx, "Delete pending media error", slog.Any("err", err))
	}

	return nil
}

func (s *MediaService) UploadThumbnail(
	ctx context.Context,
	mediaId uuid.UUID,
	reader multipart.File,
	authInfo models.AuthenticationInfo,
) error {
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		authInfo.User.Id,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if media.Status == models.DeletedStatus ||
		media.Status == models.DeletingStatus {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Invalid media status."),
		)
	}

	mimetype.SetLimit(1024 * 1024)
	kind, err := mimetype.DetectReader(reader)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if valid, ok := models.ValidImageTypes[kind.String()]; !ok || !valid {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Invalid file type (allow: jpeg, png)."),
		)
	}

	if media.MediaThumbnail != nil {
		if err := s.storageHelper.Delete(
			ctx,
			&storage.Object{
				Id: media.MediaThumbnail.Thumbnail.File.FileId,
			},
		); err != nil {
			return response.NewInternalServerError(err)
		}

		if err := s.mediaRepo.DeleteMediaThumbnail(
			ctx,
			media.Id,
			media.MediaThumbnail.ThumbnailId,
		); err != nil {
			return response.NewInternalServerError(err)
		}

		media.MediaThumbnail = nil
	}

	newThumbnail, totalSize, err := s.thumbnailHelper.GenerateThumbnail(
		ctx,
		media.UserId,
		reader,
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.mediaRepo.CreateMediaThumbnail(ctx, &models.MediaThumbnail{
		MediaId:     media.Id,
		Thumbnail:   newThumbnail,
		ThumbnailId: newThumbnail.Id,
	}); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.usageRepo.CreateLog(
		ctx,
		(&models.UsageLogBuilder{}).
			SetUserId(media.UserId).
			SetStorage(totalSize).
			Build(),
	); err != nil {
		return response.NewInternalServerError(err)
	}

	media.UpdatedAt = time.Now().UTC()
	if err := s.mediaRepo.UpdateMedia(ctx, media); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *MediaService) DeleteMediaThumbnail(
	ctx context.Context,
	mediaId uuid.UUID,
	authInfo models.AuthenticationInfo,
) error {
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		authInfo.User.Id,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if media.IsDeleted() {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Invalid media status."),
		)
	}

	if media.MediaThumbnail != nil {
		if err := s.storageHelper.Delete(
			ctx,
			&storage.Object{
				Id: media.MediaThumbnail.Thumbnail.File.FileId,
			},
		); err != nil {
			return response.NewInternalServerError(err)
		}

		if err := s.mediaRepo.DeleteMediaThumbnail(
			ctx,
			media.Id,
			media.MediaThumbnail.ThumbnailId,
		); err != nil {
			return response.NewInternalServerError(err)
		}

		media.MediaThumbnail = nil
	}

	return nil
}

func (s *MediaService) UploadPart(
	ctx context.Context,
	mediaId uuid.UUID,
	mediaSize int64,
	newPart *models.Part,
	reader multipart.File,
	authInfo models.AuthenticationInfo,
) error {
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		authInfo.User.Id,
		mediaId,
	)
	if err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound,
		) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if media.Status != models.NewStatus {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Invalid media status."),
		)
	}

	if media.Size == 0 {
		media.Size = mediaSize
		if err := s.mediaRepo.UpdateMedia(
			ctx,
			media,
		); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	var totalSize int64
	for _, part := range media.Parts {
		if part.Hash == newPart.Hash && part.Index == newPart.Index {
			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("Part is already uploaded."),
			)
		}

		totalSize += part.Size
	}

	if totalSize+newPart.Size > models.MaxMediaSize {
		if err := s.deleteMediaResource(media); err != nil {
			slog.Error(
				"delete media error",
				slog.Any("mediaId", mediaId),
				slog.Any("err", err),
			)
		}

		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Media size is too large."),
		)
	}

	path := fmt.Sprintf(
		"%s/%s/parts",
		s.storagePath,
		media.Id,
	)
	if err := os.MkdirAll(
		path,
		0o755,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	dst, err := os.Create(
		fmt.Sprintf(
			"%s/%s/parts/%s-%d",
			s.storagePath,
			mediaId,
			newPart.Hash,
			newPart.Index,
		),
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return response.NewInternalServerError(err)
	}

	if _, err := io.Copy(
		dst,
		reader,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.partRepo.Create(
		ctx,
		newPart,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	if userMap, ok := s.userConcurrentMediaUploadingMap[authInfo.User.Id]; ok {
		if _, ok := userMap[mediaId]; !ok {
			if len(userMap) >= s.userConcurrentMediaUploadingLimit {
				return response.NewHttpError(
					http.StatusBadRequest,
					fmt.Errorf("Too many concurrent media uploading."),
				)
			}

			userMap[mediaId] = 1
		}
	} else {
		s.userConcurrentMediaUploadingMap[authInfo.User.Id] = map[uuid.UUID]int{
			mediaId: 1,
		}
	}

	return nil
}

func (s *MediaService) UploadMediaComplete(
	ctx context.Context,
	mediaId uuid.UUID,
	authInfo models.AuthenticationInfo,
) error {
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		mediaService := s.newMediaServiceWithTx(tx)
		media, err := mediaService.mediaRepo.GetUserMediaById(
			ctx,
			authInfo.User.Id,
			mediaId,
		)
		if err != nil {
			if errors.Is(
				err,
				gorm.ErrRecordNotFound,
			) {
				return response.NewNotFoundError(err)
			}

			return response.NewInternalServerError(err)
		}

		if len(media.Parts) == 0 {
			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("Invalid media."),
			)
		}

		firstPart, err := os.Open(
			fmt.Sprintf(
				"%s/%s/parts/%s-%d",
				mediaService.storagePath,
				mediaId,
				media.Parts[0].Hash,
				media.Parts[0].Index,
			),
		)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		mimetype.SetLimit(1024 * 1024)
		kind, err := mimetype.DetectReader(firstPart)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		media.Mimetype = kind.String()
		var supported bool
		switch media.Type {
		case models.VideoType:
			supported = models.SupportedVideoMimetypeMapping[kind.String()]
		case models.AudioType:
			supported = models.SupportedAudioMimetypeMapping[kind.String()]
		}

		if !supported {
			if err := mediaService.deleteMediaResource(media); err != nil {
				slog.Error(
					"delete media error",
					slog.Any("mediaId", mediaId),
					slog.Any("err", err),
				)
			}

			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("Invalid media type: %s.", kind.String()),
				"Invalid media type.",
			)
		}

		if !media.IsNew() {
			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("Invalid media status"),
			)
		}

		var totalSize int64
		for _, part := range media.Parts {
			totalSize += part.Size
		}

		if totalSize != media.Size {
			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("Invalid media size"),
			)
		}

		file, err := os.Create(media.GetSourcePath(s.storagePath))
		if err != nil {
			return response.NewInternalServerError(err)
		}

		defer file.Close()

		if userMap, ok := mediaService.userConcurrentMediaUploadingMap[authInfo.User.Id]; ok {
			delete(userMap, mediaId)
		}

		for _, part := range media.Parts {
			path := fmt.Sprintf(
				"%s/%s/parts/%s-%d",
				mediaService.storagePath,
				mediaId,
				part.Hash,
				part.Index,
			)
			partFile, err := os.Open(path)
			if err != nil {
				return response.NewInternalServerError(err)
			}

			defer partFile.Close()

			if _, err := io.Copy(
				file,
				partFile,
			); err != nil {
				return response.NewInternalServerError(err)
			}

			if err := os.Remove(path); err != nil {
				return response.NewInternalServerError(err)
			}
		}

		file.Close()
		if err := mediaService.completeMedia(ctx, media, authInfo); err != nil {
			return err
		}

		return nil
	}); err != nil {
		if err := s.mediaRepo.UpdateMediaStatusById(ctx, mediaId, models.FailStatus); err != nil {
			slog.Error("update media status error", slog.Any("err", err))
		}

		return err
	}

	return nil
}

func (s *MediaService) HandleLiveStreamMedias(
	ctx context.Context,
) error {
	savedLivestreamMedias, err := s.mediaRepo.GetSavedLivestreamMedias(ctx)
	if err != nil {
		return err
	}

	for _, media := range savedLivestreamMedias {
		if _, err := os.Stat(media.GetSourcePath(s.storagePath)); os.IsNotExist(
			err,
		) {
			continue
		}

		if err := s.completeMedia(ctx, media, models.AuthenticationInfo{
			User: &models.User{
				Id: media.UserId,
			},
		}); err != nil {
			slog.Error(
				"error complete media",
				slog.Any("mediaId", media.Id),
				slog.Any("err", err),
			)
		}
	}

	return nil
}

func (s *MediaService) completeMedia(
	ctx context.Context,
	media *models.Media,
	authInfo models.AuthenticationInfo,
) error {
	if len(media.MediaQualities) == 0 {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Missing media qualities."),
		)
	}

	media.IsMp4 = media.Mimetype == "video/mp4"
	media.Status = models.WaitingStatus
	handler := core.NewGeneralHandler(
		media,
		core.WithInputPath(s.storagePath),
		core.WithOutputPath(s.outputPath),
	)
	info, err := handler.GetMediaInfo(ctx)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if info.Format.Duration == "" {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Missing media's duration information."),
		)
	}

	if media.Size == 0 {
		if info.Format.Size == "" {
			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("Missing media's size information."),
			)
		}

		media.Size, err = strconv.ParseInt(info.Format.Size, 10, 64)
		if err != nil {
			return response.NewInternalServerError(err)
		}
	}

	mediaDuration, err := strconv.ParseFloat(info.Format.Duration, 64)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	indexMap := make(map[string]int32)
	var haveAudio bool
	var colorFormat string
	var videoHeight, videoWidth int32
	var videoStreamCount, audioStreamCount int32
	audioChannelMap := make(map[int32]string)
	startTimeMapping := make(map[string]float64)
	for _, stream := range info.Streams {
		index, ok := indexMap[stream.CodecType]
		if !ok {
			index = 0
		} else {
			index++
		}

		stream.Id = uuid.New()
		stream.MediaId = media.Id
		stream.TypeIndex = index
		if err := s.streamRepo.Create(ctx, stream); err != nil {
			return response.NewInternalServerError(err)
		}

		if stream.CodecType == models.StreamCodecTypeVideo ||
			stream.CodecType == models.StreamCodecTypeAudio {
			startTimeMapping[fmt.Sprintf("%s:%d", stream.CodecType, index)], err = strconv.ParseFloat(
				stream.StartTime,
				32,
			)
			if err != nil {
				return response.NewInternalServerError(err)
			}
		}

		indexMap[stream.CodecType] = index
		if stream.CodecType == models.StreamCodecTypeAudio {
			haveAudio = true
			audioStreamCount++
			audioChannelMap[stream.TypeIndex] = fmt.Sprint(stream.Channels)
		}

		if stream.CodecType == models.StreamCodecTypeVideo {
			videoStreamCount++
			colorFormat = stream.PixFmt
			videoHeight = stream.Height
			videoWidth = stream.Width
			if stream.AvgFrameRate != "" {
				frameData := strings.Split(stream.AvgFrameRate, "/")
				if len(frameData) == 1 {
					var err error
					media.AvgFrameRate, err = strconv.ParseFloat(
						frameData[0],
						64,
					)
					if err != nil {
						return response.NewInternalServerError(err)
					}
				} else {
					numerator, err := strconv.ParseFloat(frameData[0], 64)
					if err != nil {
						return response.NewInternalServerError(err)
					}

					denominator, err := strconv.ParseFloat(frameData[1], 64)
					if err != nil {
						return response.NewInternalServerError(err)
					}

					if denominator != 0 {
						media.AvgFrameRate = numerator / denominator
					}
				}
			}
		}
	}

	info.Format.Id = uuid.New()
	info.Format.MediaId = media.Id
	if err := s.formatRepo.Create(ctx, info.Format); err != nil {
		return response.NewInternalServerError(err)
	}

	var mp4Quality *models.MediaQuality
	totalCost := decimal.Zero
	validQualities := make([]*models.MediaQuality, 0, len(media.MediaQualities))
	haveVideo := false
	for _, quality := range media.MediaQualities {
		if quality.VideoConfig == nil {
			if quality.AudioConfig != nil &&
				quality.AudioConfig.Index > audioStreamCount {
				quality.Status = models.FailStatus
				if err := s.qualityRepo.UpdateQuality(ctx, quality); err != nil {
					return response.NewInternalServerError(err)
				}

				continue
			}

			if media.Type == models.VideoType {
				validQualities = append(validQualities, quality)
				continue
			}
		}

		resolution := "audio"
		if media.Type == models.VideoType {
			maxSize := min(videoHeight, videoWidth)
			if (quality.VideoConfig.Height > maxSize && quality.VideoConfig.Codec == models.H265Codec) ||
				(quality.VideoConfig.Index > videoStreamCount) ||
				(quality.VideoConfig.Height > models.MaxH264Resolution && quality.VideoConfig.Codec == models.H264Codec) ||
				(quality.VideoConfig.Height > models.MaxH265Resolution && quality.VideoConfig.Codec == models.H265Codec) {
				quality.Status = models.FailStatus
				if err := s.qualityRepo.UpdateQuality(ctx, quality); err != nil {
					return response.NewInternalServerError(err)
				}

				continue
			}

			resolution = quality.Resolution
			haveVideo = true
		}

		transcodeCost, err := s.calculateMediaCost(
			resolution,
			mediaDuration,
		)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		quality.TranscodeCost = transcodeCost
		if !media.IsMp4 && media.Type == models.VideoType {
			if (mp4Quality == nil && quality.VideoConfig != nil) ||
				(quality.VideoConfig != nil && quality.VideoConfig.Width > mp4Quality.VideoConfig.Height) {
				mp4Quality = models.NewQuality(
					media.Id,
					&models.QualityConfig{
						Type:          models.Mp4QualityType,
						ContainerType: models.Mp4ContainerType,
						Resolution:    quality.Resolution,
					},
				)
				mp4Quality.VideoConfig = quality.VideoConfig
			}
		}

		totalCost = totalCost.Add(transcodeCost)
		validQualities = append(validQualities, quality)
	}

	if len(validQualities) == 0 ||
		(!haveVideo && media.Type == models.VideoType) {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("No valid video qualities found."),
		)
	}

	media.MediaQualities = validQualities
	if err := s.paymentClient.CreateTransaction(
		ctx, authInfo.User.Id, totalCost, models.PaymentTypeBill, media.Id,
	); err != nil {
		if err == payment.InsufficientBalanceError {
			if err := s.DeleteMedia(ctx, media.Id, authInfo); err != nil {
				slog.Error(
					"delete media error",
					slog.Any("mediaId", media.Id),
					slog.Any("err", err),
				)
			}

			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("Not enough balance, media deleted."),
			)
		} else {
			return response.NewInternalServerError(err)
		}
	}

	if mp4Quality != nil {
		mp4Quality.AudioConfig = &models.AudioConfig{
			Codec:      models.AacCodec,
			Bitrate:    128_000,
			SampleRate: 44100,
			Channels:   models.StereoChannel,
		}

		if err := s.qualityRepo.Create(ctx, mp4Quality); err != nil {
			return response.NewInternalServerError(err)
		}

		media.MediaQualities = append(media.MediaQualities, mp4Quality)
	}

	if err := s.partRepo.DeletePartsByMediaId(
		ctx,
		media.Id,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	req := &job_pb.RegisterJobRequest{
		MediaId:    media.Id.String(),
		RegisterId: s.registerId,
		PlaylistConfigs: make(
			[]*job_pb.PlaylistConfig,
			0,
			len(media.MediaQualities),
		),
	}
	for _, q := range media.MediaQualities {
		if q.Type != models.Mp4QualityType {
			if q.VideoConfig != nil {
				req.PlaylistConfigs = append(
					req.PlaylistConfigs,
					&job_pb.PlaylistConfig{
						Name:       q.Id.String(),
						Resolution: q.Resolution,
						Type:       "segment",
						SegmentConfig: &job_pb.SegmentConfig{
							Duration:      int32(media.SegmentDuration),
							SegmentType:   q.Type,
							ContainerType: q.ContainerType,
						},
						VideoConfig: &job_pb.VideoConfig{
							Codec:        q.VideoConfig.Codec,
							Bitrate:      q.VideoConfig.Bitrate,
							Size:         media.Size,
							Index:        q.VideoConfig.Index,
							Width:        videoWidth,
							Height:       videoHeight,
							ColourFormat: colorFormat,
							StartTime: float32(
								startTimeMapping[fmt.Sprintf("%s:%d", models.StreamCodecTypeVideo, q.VideoConfig.Index)],
							),
						},
					},
				)
			}

			if q.AudioConfig != nil {
				if haveAudio {
					if q.AudioConfig.Channels == "" {
						q.AudioConfig.Channels = audioChannelMap[q.AudioConfig.Index]
					} else {
						userChannels, err := strconv.ParseInt(q.AudioConfig.Channels, 10, 64)
						if err != nil {
							return response.NewInternalServerError(err)
						}

						ac, ok := audioChannelMap[q.AudioConfig.Index]
						if ok {
							audioChannels, err := strconv.ParseInt(ac, 10, 64)
							if err != nil {
								return response.NewInternalServerError(err)
							}

							if userChannels > audioChannels {
								return response.NewBadRequestError("Invalid audio channel.")
							}
						} else {
							return response.NewBadRequestError("Invalid audio index.")
						}

					}

					req.PlaylistConfigs = append(
						req.PlaylistConfigs,
						&job_pb.PlaylistConfig{
							Name:       q.Id.String(),
							Resolution: q.Resolution,
							Type:       "segment",
							SegmentConfig: &job_pb.SegmentConfig{
								Duration:      int32(media.SegmentDuration),
								SegmentType:   q.Type,
								ContainerType: q.ContainerType,
							},
							AudioConfig: &job_pb.AudioConfig{
								Codec:      q.AudioConfig.Codec,
								Bitrate:    q.AudioConfig.Bitrate,
								SampleRate: q.AudioConfig.SampleRate,
								Channels:   q.AudioConfig.Channels,
								Index:      q.AudioConfig.Index,
								Language:   q.AudioConfig.Language,
								StartTime: float32(
									startTimeMapping[fmt.Sprintf("%s:%d", models.StreamCodecTypeAudio, q.AudioConfig.Index)],
								),
							},
						},
					)
				} else {
					if q.VideoConfig != nil {
						q.AudioConfig = nil
					} else {
						if err := s.qualityRepo.DeleteQualityById(ctx, q.Id); err != nil {
							return response.NewInternalServerError(err)
						}
					}
				}
			}
		} else {
			if q.VideoConfig != nil {
				plConfig := &job_pb.PlaylistConfig{
					Name:       q.Id.String(),
					Resolution: q.Resolution,
					Type:       "mp4",
					SegmentConfig: &job_pb.SegmentConfig{
						Duration:      int32(media.SegmentDuration),
						SegmentType:   q.Type,
						ContainerType: q.ContainerType,
					},
					VideoConfig: &job_pb.VideoConfig{
						Codec:        q.VideoConfig.Codec,
						Bitrate:      q.VideoConfig.Bitrate,
						Size:         media.Size,
						Index:        q.VideoConfig.Index,
						Width:        q.VideoConfig.Width,
						Height:       q.VideoConfig.Height,
						ColourFormat: colorFormat,
					},
				}

				if haveAudio {
					plConfig.AudioConfig = &job_pb.AudioConfig{
						Codec:      q.AudioConfig.Codec,
						Bitrate:    q.AudioConfig.Bitrate,
						SampleRate: q.AudioConfig.SampleRate,
						Channels:   q.AudioConfig.Channels,
						Index:      q.AudioConfig.Index,
						Language:   q.AudioConfig.Language,
					}
				}

				req.PlaylistConfigs = append(req.PlaylistConfigs, plConfig)
			}
		}
	}

	job, err := s.jobClient.RegisterJob(ctx, req)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	media.JobId = job.Id
	for _, pl := range job.Playlists {
		for _, q := range media.MediaQualities {
			if pl.Name == q.Id.String() {
				if pl.VideoConfig != nil {
					q.VideoPlaylistId = pl.Id
				}

				if pl.AudioConfig != nil {
					q.AudioPlaylistId = pl.Id
				}
			}

			if err := s.qualityRepo.UpdateQuality(ctx, q); err != nil {
				return response.NewInternalServerError(err)
			}
		}
	}

	media.MediaQualities = nil
	if err := s.mediaRepo.UpdateMedia(
		ctx,
		media,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.callWebhookCh.Publish(
		models.WebhookNotification{
			Type:      models.EventFileReceived,
			EmittedAt: time.Now().UTC(),
			MediaId:   media.Id,
			Qualities: media.Qualities,
			Title:     media.Title,
			Status:    media.Status,
			MediaType: models.VideoMediaType,
			UserId:    media.UserId,
		},
	); err != nil {
		slog.Error("publish webhook error", slog.Any("error", err))
	}

	return nil
}

func (s *MediaService) GetMediaById(
	ctx context.Context, mediaId uuid.UUID,
) (*models.Media, error) {
	return s.mediaRepo.GetMediaById(ctx, mediaId)
}

func (s *MediaService) GenerateMediaCaptions(
	ctx context.Context,
) error {
	// audioFiles, err := s.cdnFileRepo.GetCdnFilesByType(ctx, models.CdnAudioType)
	// if err != nil {
	// 	return err
	// }

	// for _, audioFile := range audioFiles {
	// 	presignedUrl, err := s.s3Client.GetPresignURL(ctx, audioFile.FileId)
	// 	if err != nil {
	// 		return err
	// 	}
	//
	// 	if presignedUrl != "" {
	// 		for language, code := range models.DefaultCaptionLanguage {
	// 			taskId, err := s.transcribeClient.CreateTask(
	// 				ctx,
	// 				audioFile.BelongsToId,
	// 				language,
	// 				presignedUrl,
	// 			)
	// 			if err != nil {
	// 				return err
	// 			}
	//
	// 			newCaption := models.NewMediaCaption(audioFile.BelongsToId, code, false, "gen")
	// 			newCaption.TaskId = taskId
	// 			if err := s.mediaCaptionRepo.Create(ctx, newCaption); err != nil {
	// 				return err
	// 			}
	// 		}
	// 	}
	//
	// 	if err := s.cdnFileRepo.DeleteCdnFileByFileId(ctx, audioFile.FileId); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (s *MediaService) UpdateMediasStatus(
	ctx context.Context,
) error {
	doneMedias, err := s.mediaRepo.GetDoneMedias(ctx)
	if err != nil {
		return err
	}

	existedMap := make(map[uuid.UUID]bool)
	for _, media := range doneMedias {
		mediaStatus := models.FailStatus
		for _, quality := range media.MediaQualities {
			if quality.Status == models.DoneStatus {
				mediaStatus = models.DoneStatus
			}
		}

		if err := s.mediaRepo.UpdateMediaStatusById(
			ctx,
			media.Id,
			mediaStatus,
		); err != nil {
			return err
		}

		if !existedMap[media.Id] {
			switch mediaStatus {
			case models.DoneStatus:
				var haveAudio bool
				for _, stream := range media.Streams {
					if stream.CodecType == models.StreamCodecTypeAudio {
						haveAudio = true
					}
				}

				chunkSize := 100 * 1024 * 1024 // 100 MB
				file, err := os.Open(media.GetSourcePath(s.storagePath))
				if err != nil {
					return err
				}

				defer file.Close()
				var index int = 1
				for {
					chunk := make([]byte, chunkSize)
					n, err := file.Read(chunk)
					if err != nil && err != io.EOF {
						return err
					}

					if n == 0 {
						break
					}

					resp, err := s.storageHelper.Upload(
						ctx,
						media.Id.String(),
						bytes.NewReader(chunk[:n]),
					)
					if err != nil {
						return err
					}

					if err := s.mediaRepo.CreateFile(ctx, &models.MediaFile{
						FileId:  resp.Id,
						MediaId: media.Id,
						File: models.NewCdnFile(
							media.UserId,
							resp.Id,
							resp.Size,
							resp.Offset,
							index,
							models.CdnSourceFileType,
						),
					}); err != nil {
						return err
					}

					index++
				}

				if err := s.usageRepo.CreateLog(
					ctx,
					(&models.UsageLogBuilder{}).
						SetUserId(media.UserId).
						SetStorage(media.Size).
						SetIsUserCost(true).
						Build(),
				); err != nil {
					return err
				}

				if haveAudio {
					if err := func() error {
						handler := core.NewGeneralHandler(
							media,
						)
						inputPath := media.GetSourcePath(s.storagePath)
						outputPath := filepath.Join(filepath.Dir(inputPath), "audio.mp3")
						if err := handler.ConvertMediaToAudio(inputPath, outputPath, 0); err != nil {
							return err
						}

						audioFile, err := os.Open(outputPath)
						if err != nil {
							return err
						}

						defer audioFile.Close()

						taskId, err := s.transcribeClient.CreateTask(
							ctx,
							media.Id,
							models.DefaultLanguage,
							fmt.Sprintf("%s/api/videos/%s/audio.mp3", models.BeUrl, media.Id.String()),
						)
						if err != nil {
							return err
						}

						newCaption := models.NewMediaCaption(
							media.Id,
							models.DefaultLanguage2Word,
							false,
							"gen",
						)
						newCaption.Status = models.NewStatus
						newCaption.TaskId = taskId
						if err := s.mediaCaptionRepo.Create(ctx, newCaption); err != nil {
							return err
						}

						if err = s.handleAudioFile(ctx, media); err != nil {
							return err
						}

						return nil
					}(); err != nil {
						slog.Error(
							"transcribe audio error",
							slog.Any("err", err),
						)
					}
				}

				if err := s.callWebhookCh.Publish(
					models.WebhookNotification{
						Type:      models.EventEncodingFinished,
						EmittedAt: time.Now().UTC(),
						MediaId:   media.Id,
						Qualities: media.Qualities,
						Title:     media.Title,
						Status:    mediaStatus,
						MediaType: models.VideoMediaType,
						UserId:    media.UserId,
					},
				); err != nil {
					slog.Error("publish webhook error", slog.Any("error", err))
				}
			case models.FailStatus:
				if err := s.callWebhookCh.Publish(
					models.WebhookNotification{
						Type:      models.EventEncodingFailed,
						EmittedAt: time.Now().UTC(),
						MediaId:   media.Id,
						Qualities: media.Qualities,
						Title:     media.Title,
						Status:    mediaStatus,
						MediaType: models.VideoMediaType,
						UserId:    media.UserId,
					},
				); err != nil {
					slog.Error("publish webhook error", slog.Any("error", err))
				}

			}
		}
	}

	return nil
}

func (s *MediaService) handlePlaylist(
	ctx context.Context,
	pl *job_pb.Playlist,
) error {
	quality, err := s.qualityRepo.GetQualityByPlaylistId(ctx, pl.Id)
	if err != nil {
		return err
	}

	if quality.Status == models.DoneStatus ||
		quality.Status == models.FailStatus {
		return nil
	}

	switch pl.Status {
	case models.DoneStatus:
		var isVideoPlaylist bool
		if quality.VideoPlaylistId == pl.Id {
			isVideoPlaylist = true
			quality.VideoCodec = pl.VideoCodec
		}

		if quality.AudioPlaylistId == pl.Id {
			quality.AudioCodec = pl.AudioCodec
		}

		if pl.VideoConfig != nil {
			quality.Bandwidth = pl.VideoConfig.Bandwidth
		} else if pl.AudioConfig != nil {
			quality.Bandwidth = pl.AudioConfig.Bandwidth
		}

		totalStorage := int64(0)
		if len(pl.Files) == 0 {
			return fmt.Errorf("playlist %s has no files", pl.Id)
		}

		existingFile := make(map[string]bool)
		for _, file := range quality.Files {
			if _, ok := existingFile[file.FileId]; !ok {
				existingFile[file.FileId] = true
			}
		}

		for _, file := range pl.Files {
			if _, ok := existingFile[file.Id]; ok {
				continue
			}

			if err := s.qualityRepo.CreateFile(ctx, &models.MediaQualityFile{
				MediaQualityId: quality.Id,
				FileId:         file.Id,
				File: models.NewCdnFile(
					quality.Media.UserId,
					file.Id,
					file.Size,
					file.Offset,
					int(file.Index),
					file.Type,
				),
			}); err != nil {
				return err
			}

			totalStorage += file.Size
		}

		if quality.VideoConfig != nil && isVideoPlaylist {
			transcodeLogs := (&models.UsageLogBuilder{}).
				SetUserId(quality.Media.UserId).
				SetTranscode(quality.Media.GetMediaDuration()).
				SetTranscodeCost(quality.TranscodeCost).
				SetIsUserCost(true).
				Build()
			transcodeLogs.CreatedAt = quality.CreatedAt
			if err := s.usageRepo.CreateLog(ctx, transcodeLogs); err != nil {
				return err
			}
		}

		if err := s.usageRepo.CreateLog(
			ctx,
			(&models.UsageLogBuilder{}).
				SetUserId(quality.Media.UserId).
				SetStorage(totalStorage).
				SetIsUserCost(true).
				Build(),
		); err != nil {
			return response.NewInternalServerError(err)
		}

		if quality.Media.MediaThumbnail == nil &&
			quality.Media.Type == models.VideoType {
			if err := func() error {
				handler := core.NewGeneralHandler(
					quality.Media,
					core.WithInputPath(s.storagePath),
					core.WithOutputPath(s.outputPath),
				)
				if err := handler.GenerateThumbnail(ctx); err != nil {
					return err
				}

				reader, err := os.Open(
					fmt.Sprintf(
						"%s/%s/thumbnail/original.jpg",
						s.outputPath,
						quality.Media.Id,
					),
				)
				if err != nil {
					return err
				}

				defer reader.Close()

				newThumbnail, totalSize, err := s.thumbnailHelper.GenerateThumbnail(
					ctx,
					quality.Media.UserId,
					reader,
				)
				if err := s.mediaRepo.CreateMediaThumbnail(
					ctx,
					&models.MediaThumbnail{
						MediaId:     quality.Media.Id,
						Thumbnail:   newThumbnail,
						ThumbnailId: newThumbnail.Id,
					},
				); err != nil {
					return err
				}

				if err := s.usageRepo.CreateLog(
					ctx,
					(&models.UsageLogBuilder{}).
						SetUserId(quality.Media.UserId).
						SetStorage(totalSize).
						SetIsUserCost(false).
						Build(),
				); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				slog.Error("generate thumbnail", "err", err)
			}
		}

		shouldBreak := false
		switch quality.Media.Type {
		case models.VideoType:
			if quality.Type != models.Mp4QualityType {
				if (quality.VideoPlaylistId != "" && quality.VideoCodec == "") ||
					(quality.AudioPlaylistId != "" && quality.AudioCodec == "") {
					shouldBreak = true
				}
			}
		case models.AudioType:
			if quality.AudioPlaylistId != "" && quality.AudioCodec == "" {
				shouldBreak = true
			}
		}

		if shouldBreak {
			break
		}

		quality.Status = models.DoneStatus
		quality.TranscodedAt = time.Now().UTC()
	case models.FailStatus:
		for _, file := range pl.Files {
			if err := s.storageHelper.Delete(
				ctx, &storage.Object{
					Id: file.Id,
				},
			); err != nil {
				return err
			}
		}

		if err := s.paymentClient.CreateTransaction(
			ctx, quality.Media.UserId,
			quality.TranscodeCost,
			models.PaymentTypeRefund,
			quality.Media.Id,
		); err != nil {
			return err
		}

		quality.TranscodedAt = time.Now().UTC()
		quality.Status = models.FailStatus
	default:
		return nil
	}

	if err := s.qualityRepo.UpdateQuality(ctx, quality); err != nil {
		return err
	}

	if (quality.Status == models.DoneStatus || quality.Status == models.FailStatus) &&
		quality.Type != models.Mp4QualityType {
		if err := s.callWebhookCh.Publish(
			models.WebhookNotification{
				Type:      models.EventPartialFinished,
				EmittedAt: time.Now().UTC(),
				MediaId:   quality.MediaId,
				Qualities: quality.Resolution,
				Title:     quality.Media.Title,
				MediaType: models.VideoMediaType,
				UserId:    quality.Media.UserId,
			},
		); err != nil {
			slog.Error("publish webhook error", slog.Any("error", err))
		}
	}

	return nil
}

func (s *MediaService) StopCron() {
	close(s.stopWatchPlaylistCh)
	close(s.stopWatchQualityCh)
	close(s.stopWatchMediaCh)
}

func (s *MediaService) StartWatchPlaylist(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				transcodingMedia, err := s.mediaRepo.GetMediaByStatuses(
					ctx,
					[]string{
						models.TranscodingStatus,
						models.WaitingStatus,
					},
				)
				if err != nil {
					slog.Error(
						"get transcoding videos error",
						slog.Any("error", err),
					)
				}

				for _, media := range transcodingMedia {
					if media.JobId == "" {
						slog.Warn(
							"media job id is empty, skipping",
							slog.Any("mediaId", media.Id),
						)
						continue
					}

					job, err := s.jobClient.GetJobDetail(
						ctx,
						media.JobId,
						s.registerId,
					)
					if err != nil {
						slog.Error(
							"get job by id error",
							slog.Any("jobId", media.JobId),
							slog.Any("error", err),
						)
					}

					if (job.Status == models.ProcessingStatus ||
						job.Status == models.DoneStatus ||
						job.Status == models.FailStatus) &&
						media.Status == models.WaitingStatus {
						if err := s.callWebhookCh.Publish(
							models.WebhookNotification{
								Type:      models.EventEncodingStarted,
								EmittedAt: time.Now().UTC(),
								MediaId:   media.Id,
								Title:     media.Title,
								Status:    models.TranscodingStatus,
								MediaType: models.VideoMediaType,
								UserId:    media.UserId,
							},
						); err != nil {
							slog.Error(
								"publish webhook error",
								slog.Any("error", err),
							)
						}

						if err := s.mediaRepo.UpdateMediaStatusById(ctx, media.Id, models.TranscodingStatus); err != nil {
							slog.Error(
								"update media status error",
								slog.Any("error", err),
							)
						}
					}

					for _, pl := range job.Playlists {
						if err := s.handlePlaylist(ctx, pl); err != nil {
							slog.Error(
								"update playlist status error",
								slog.Any("error", err),
							)
						}
					}
				}
			case <-s.stopWatchPlaylistCh:
				return

			}
		}
	}()
}

func (s *MediaService) deleteMediaResource(
	media *models.Media,
) error {
	if err := os.RemoveAll(fmt.Sprintf("%s/%s", s.storagePath, media.Id)); err != nil &&
		!os.IsNotExist(err) {
		return err
	}

	ctx := context.Background()
	if err := s.partRepo.DeletePartsByMediaId(ctx, media.Id); err != nil {
		return err
	}

	if media.MediaThumbnail != nil {
		if err := s.storageHelper.Delete(ctx, &storage.Object{
			Id: media.MediaThumbnail.Thumbnail.File.FileId,
		}); err != nil {
			return err
		}

		if err := s.mediaRepo.DeleteMediaThumbnail(ctx, media.Id, media.MediaThumbnail.ThumbnailId); err != nil {
			return err
		}

		media.MediaThumbnail = nil
	}

	chapters, err := s.mediaChapterRepo.GetMediaChaptersByMediaId(ctx, media.Id)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	for _, chapter := range chapters {
		if err := s.storageHelper.Delete(
			ctx,
			&storage.Object{
				Id: chapter.File.File.Id,
			},
		); err != nil {
			return response.NewInternalServerError(err)
		}

		if err := s.mediaChapterRepo.DeleteMediaChapterById(ctx, chapter.Id); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	captions, err := s.mediaCaptionRepo.GetMediaCaptionsByMediaId(ctx, media.Id)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	for _, caption := range captions {
		if caption.Status != models.DoneStatus {
			continue
		}

		if caption.File != nil && caption.File.File != nil {
			if err := s.storageHelper.Delete(
				ctx,
				&storage.Object{
					Id: caption.File.File.Id,
				},
			); err != nil {
				return response.NewInternalServerError(err)
			}
		}

		if err := s.mediaCaptionRepo.DeleteMediaCaptionById(ctx, caption.Id); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	return nil
}

func (s *MediaService) deleteMediaTranscodingContent(
	ctx context.Context,
	media *models.Media,
) error {
	if err := s.streamRepo.DeleteStreamsByMediaId(ctx, media.Id); err != nil {
		return err
	}

	if err := s.formatRepo.DeleteFormatByMediaId(ctx, media.Id); err != nil {
		return err
	}

	for _, file := range media.MediaFiles {
		if err := s.storageHelper.Delete(
			ctx,
			&storage.Object{
				Id: file.File.Id,
			},
		); err != nil {
			return err
		}
	}

	if err := s.mediaRepo.DeleteMediaFiles(ctx, media.Id); err != nil {
		return err
	}

	for _, quality := range media.MediaQualities {
		if err := s.deleteMediaQuality(ctx, quality); err != nil {
			return err
		}
	}

	if err := os.RemoveAll(fmt.Sprintf("%s/%s", s.outputPath, media.Id)); err != nil {
		return err
	}

	return nil
}

func (s *MediaService) calculateMediaCost(
	quality string,
	mediaDuration float64,
) (decimal.Decimal, error) {
	if mediaDuration <= 0 {
		return decimal.Zero, fmt.Errorf("invalid media duration")
	}

	var rs decimal.Decimal

	price, ok := models.MediaQualitiesTranscodePriceMapping[quality]
	if ok {
		rs = rs.Add(
			decimal.NewFromFloat(price).
				Mul(decimal.NewFromFloat(mediaDuration / 60)).
				Mul(decimal.NewFromFloat(math.Pow10(18))),
		).Round(0)
	}

	return rs, nil
}

func (s *MediaService) deleteMediaQuality(
	ctx context.Context,
	quality *models.MediaQuality,
) error {
	for _, file := range quality.Files {
		if err := s.storageHelper.Delete(
			ctx,
			&storage.Object{
				Id: file.FileId,
			},
		); err != nil {
			return err
		}
	}

	if err := s.qualityRepo.DeleteQualityById(ctx, quality.Id); err != nil {
		return err
	}

	return nil
}

func (s *MediaService) allowGetContent(
	ctx context.Context,
	userId uuid.UUID,
) (bool, error) {
	userStatus, ok := s.userStatusMapping[userId]
	if ok {
		return userStatus == models.ActiveStatus, nil
	}

	user, err := s.userRepo.GetUserById(ctx, userId)
	if err != nil {
		return false, err
	}

	s.userStatusMapping[userId] = user.Status
	return user.Status == models.ActiveStatus, nil
}

func (s *MediaService) handleAudioFile(
	ctx context.Context,
	media *models.Media,
) error {
	handler := core.NewGeneralHandler(
		media,
		core.WithInputPath(s.storagePath),
		core.WithOutputPath(s.outputPath),
	)

	if _, err := os.Stat(media.GetSourcePath(s.storagePath)); os.IsNotExist(
		err,
	) {
		return err
	}

	captionStreams, err := s.streamRepo.GetStreamsByCodecTypeAndMediaId(
		ctx,
		models.StreamCodecTypeSubtitle,
		media.Id,
	)
	if err != nil {
		return err
	}

	for _, stream := range captionStreams {
		language, ok := models.LanguageMapping[stream.Tags.Language]
		if !ok {
			language = "auto"
		}

		captionFileName := fmt.Sprintf(
			"%s/%s/%s_%d.%s",
			media.Id,
			models.StreamCodecTypeSubtitle,
			language,
			stream.Index,
			models.CaptionFormat,
		)
		if err := handler.ExtractCaptionFromMedia(captionFileName, stream.Index); err != nil {
			return err
		}

		captionFile, err := os.Open(
			filepath.Join(s.outputPath, media.Id.String(), captionFileName),
		)
		if err != nil {
			slog.Error(
				"failed to open caption file",
				slog.Any("caption_path", captionFileName),
				slog.Any("error", err),
			)
			return err
		}

		defer captionFile.Close()

		newCaption := models.NewMediaCaption(
			media.Id,
			stream.Tags.Language,
			false,
			models.CaptionFormat,
		)
		newCaption.Status = models.DoneStatus
		resp, err := s.storageHelper.Upload(
			ctx,
			newCaption.Id.String(),
			captionFile,
		)
		if err != nil {
			slog.Error(
				"failed to upload caption file",
				slog.Any("caption_path", captionFileName),
				slog.Any("error", err),
			)
			return err
		}

		if err := s.mediaCaptionRepo.Create(ctx, newCaption); err != nil {
			slog.Error(
				"failed to create media caption",
				slog.Any("media_id", media.Id.String()),
				slog.Any("error", err),
			)
			return err
		}

		if err := s.mediaCaptionRepo.CreateFile(
			ctx,
			&models.MediaCaptionFile{
				FileId:    resp.Id,
				CaptionId: newCaption.Id,
				File: models.NewCdnFile(
					media.UserId,
					resp.Id,
					resp.Size,
					resp.Offset,
					1,
					models.CdnCaptionType,
				),
			},
		); err != nil {
			slog.Error(
				"failed to create media caption file",
				slog.Any("media_id", media.Id.String()),
				slog.Any("error", err),
			)
		}

		if err := os.Remove(filepath.Join(filepath.Join(s.outputPath, media.Id.String(), captionFileName))); err != nil {
			slog.Error(
				"failed to delete caption file after upload",
				slog.Any("caption_path", captionFileName),
				slog.Any("error", err),
			)
			return err
		}
	}

	return nil
}

func (s *MediaService) UploadWaitingMediaSource(ctx context.Context) error {
	waitingMedias, err := s.mediaRepo.GetMediasByStatus(
		ctx,
		models.WaitingStatus,
	)
	if err != nil {
		return err
	}

	var errs []error
	for _, media := range waitingMedias {
		if err := func() error {
			file, err := os.Open(media.GetSourcePath(s.storagePath))
			if err != nil {
				return err
			}

			defer file.Close()
			if err := s.jobClient.UploadMediaResource(ctx, media.JobId, media.Id.String(), media.Size, file); err != nil {
				return err
			}

			if err := s.mediaRepo.UpdateMediaStatusById(ctx, media.Id, models.TranscodingStatus); err != nil {
				return err
			}

			return nil
		}(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
