package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	gtranslate "github.com/gilang-as/google-translate"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/transcribe_client"
)

type MediaCaptionService struct {
	mediaRepo        models.MediaRepository
	mediaCaptionRepo models.MediaCaptionRepository
	cdnRepo          models.CdnFileRepository
	usageRepo        models.UsageRepository

	beUrl            string
	storagePath      string
	storageHelper    storage.StorageHelper
	transcribeClient *transcribe_client.TranscribeClient
}

func NewMediaCaptionService(
	mediaRepo models.MediaRepository,
	mediaCaptionRepo models.MediaCaptionRepository,
	cdnRepo models.CdnFileRepository,
	usageRepo models.UsageRepository,

	beUrl string,
	storagePath string,
	storageHelper storage.StorageHelper,
	transcribeClient *transcribe_client.TranscribeClient,
) *MediaCaptionService {
	return &MediaCaptionService{
		mediaRepo:        mediaRepo,
		mediaCaptionRepo: mediaCaptionRepo,
		cdnRepo:          cdnRepo,
		usageRepo:        usageRepo,

		beUrl:            beUrl,
		storagePath:      storagePath,
		storageHelper:    storageHelper,
		transcribeClient: transcribeClient,
	}
}

func (s *MediaCaptionService) CreateMediaCaption(
	ctx context.Context,
	mediaId uuid.UUID,
	language string,
	description string,
	file multipart.File,
	authInfo models.AuthenticationInfo,
) (*models.MediaCaption, error) {
	media, err := s.mediaRepo.GetUserMediaById(ctx, authInfo.User.Id, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	existed, err := s.mediaCaptionRepo.GetMediaCaptionByMediaIdAndLanguage(ctx, mediaId, language)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.NewInternalServerError(err)
	}

	if existed != nil {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf(
				"Caption of media with id %s for %s already exists.",
				mediaId,
				strings.ToLower(models.LanToLanguageMapping[language]),
			),
		)
	}

	newCaption := models.NewMediaCaption(media.Id, language, false, description)
	newCaption.Status = models.DoneStatus
	resp, err := s.storageHelper.Upload(ctx, mediaId.String(), file)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if err := s.mediaCaptionRepo.Create(ctx, newCaption); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if err := s.mediaCaptionRepo.CreateFile(ctx, &models.MediaCaptionFile{
		CaptionId: newCaption.Id,
		FileId:    resp.Id,
		File: models.NewCdnFile(
			authInfo.User.Id,
			resp.Id,
			resp.Size,
			resp.Offset,
			1,
			models.CdnCaptionType,
		),
	}); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	newCaption.Url = newCaption.GetUrl(
		models.CdnCaptionType,
		media.Secret,
	)

	return newCaption, nil
}

func (s *MediaCaptionService) GetMediaCaptions(
	ctx context.Context,
	mediaId uuid.UUID,
	offset, limit int,
	authInfo models.AuthenticationInfo,
) ([]*models.MediaCaption, int64, error) {
	media, err := s.mediaRepo.GetUserMediaById(ctx, authInfo.User.Id, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, response.NewNotFoundError(err)
		}
		return nil, 0, response.NewInternalServerError(err)
	}

	subtitles, total, err := s.mediaCaptionRepo.GetMediaCaptionsByMediaIdWithOffsetAndLimit(
		ctx,
		mediaId,
		offset,
		limit,
	)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}

	for _, subtitle := range subtitles {
		subtitle.Url = subtitle.GetUrl(models.CdnCaptionType, media.Secret)
	}

	return subtitles, total, nil
}

func (s *MediaCaptionService) SetMediaDefaultCaption(
	ctx context.Context,
	mediaId uuid.UUID,
	language string,
	isDefault bool,
	authInfo models.AuthenticationInfo,
) error {
	_, err := s.mediaRepo.GetUserMediaById(ctx, authInfo.User.Id, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if _, err := s.mediaCaptionRepo.GetMediaCaptionByMediaIdAndLanguage(
		ctx,
		mediaId,
		language,
	); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if err := s.mediaCaptionRepo.SetMediaDefaultCaption(ctx, mediaId, language, isDefault); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *MediaCaptionService) DeleteMediaCaption(
	ctx context.Context,
	mediaId uuid.UUID,
	language string,
	authInfo models.AuthenticationInfo,
) error {
	_, err := s.mediaRepo.GetUserMediaById(ctx, authInfo.User.Id, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	caption, err := s.mediaCaptionRepo.GetMediaCaptionByMediaIdAndLanguage(
		ctx,
		mediaId,
		language,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if caption.File != nil {
		if err := s.storageHelper.Delete(ctx, &storage.Object{
			Id: caption.File.FileId,
		}); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	if err := s.mediaCaptionRepo.DeleteMediaCaptionById(ctx, caption.Id); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *MediaCaptionService) WatchCaptionsGeneration(ctx context.Context) error {
	captions, err := s.mediaCaptionRepo.GetMediaCaptionsByStatus(ctx, models.NewStatus)
	if err != nil {
		return err
	}

	for _, caption := range captions {
		if err := func() error {
			reader, err := s.transcribeClient.GetTaskResult(ctx, caption.TaskId)
			if err != nil {
				if errors.Is(err, transcribe_client.GenerateFailError) {
					if err := s.mediaCaptionRepo.UpdateMediaCaptionStatus(
						ctx,
						caption.Id,
						models.FailStatus,
					); err != nil {
						return err
					}
				}

				return err
			}

			data, err := io.ReadAll(reader)
			if err != nil {
				return err
			}

			resp, err := s.storageHelper.Upload(
				ctx,
				caption.MediaId.String(),
				bytes.NewReader(data),
			)
			if err != nil {
				return err
			}

			totalSize := resp.Size
			if err := s.mediaCaptionRepo.CreateFile(
				ctx,
				&models.MediaCaptionFile{
					CaptionId: caption.Id,
					FileId:    resp.Id,
					File: models.NewCdnFile(
						caption.Media.UserId,
						resp.Id,
						resp.Size,
						resp.Offset,
						1,
						models.CdnCaptionType,
					),
				},
			); err != nil {
				return err
			}

			if err := s.mediaCaptionRepo.UpdateMediaCaptionStatus(
				ctx,
				caption.Id,
				models.DoneStatus,
			); err != nil {
				return err
			}

			mediaCaptions, err := s.mediaCaptionRepo.GetMediaCaptionsByStatusAndMediaId(
				ctx,
				models.NewStatus,
				caption.MediaId,
			)
			if err != nil {
				return err
			}

			if len(mediaCaptions) == 0 {
				if err := os.Remove(filepath.Join(s.storagePath, caption.MediaId.String(), "audio.mp3")); err != nil {
					slog.Error("remove audio file", slog.Any("error", err))
				}
			}

			for _, lang := range models.DefaultCaptionLanguage {
				newCaption := models.NewMediaCaption(
					caption.MediaId,
					lang,
					false,
					caption.Description,
				)
				newCaption.Status = models.DoneStatus

				translatedData := bytes.NewBuffer(nil)
				pattern := `^\d{2}:\d{2}:\d{2}\.\d{3}\s*-->\s*\d{2}:\d{2}:\d{2}\.\d{3}$`
				re, err := regexp.Compile(pattern)
				if err != nil {
					return err
				}

				for i, line := range strings.Split(string(data), "\n") {
					if len(strings.Trim(line, " ")) == 0 {
						translatedData.WriteString("\n")
						continue
					}

					if re.MatchString(line) || i == 0 {
						translatedData.WriteString(line + "\n")
						continue
					}

					result, err := gtranslate.Translator(gtranslate.Translate{
						Text: line,
						To:   lang,
					})
					if err != nil {
						slog.Error("translate text", slog.Any("err", err), slog.Any("text", line))
						continue
					}

					for i := 0; i < len(result.Text); {
						rune, size := utf8.DecodeRuneInString(result.Text[i:])
						i += size
						translatedData.WriteRune(rune)
					}

					translatedData.WriteString("\n")
				}

				if err := s.mediaCaptionRepo.Create(ctx, newCaption); err != nil {
					return err
				}

				file, err := s.storageHelper.Upload(
					ctx,
					caption.MediaId.String(),
					bytes.NewReader(translatedData.Bytes()),
				)
				if err != nil {
					return err
				}

				totalSize += file.Size
				if err := s.mediaCaptionRepo.CreateFile(
					ctx,
					&models.MediaCaptionFile{
						CaptionId: newCaption.Id,
						FileId:    resp.Id,
						File: models.NewCdnFile(
							caption.Media.UserId,
							file.Id,
							file.Size,
							file.Offset,
							1,
							models.CdnCaptionType,
						),
					},
				); err != nil {
					return err
				}
			}

			if err := s.usageRepo.CreateLog(
				ctx,
				(&models.UsageLogBuilder{}).
					SetUserId(caption.Media.UserId).
					SetStorage(totalSize).
					SetIsUserCost(false).
					Build(),
			); err != nil {
				return err
			}

			return nil
		}(); err != nil {
			slog.Error("watch captions generation", slog.Any("error", err))
		}
	}

	return nil
}
