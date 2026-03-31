package services

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
)

type MediaChapterService struct {
	mediaRepo        models.MediaRepository
	mediaChapterRepo models.MediaChapterRepository
	cdnFileRepo      models.CdnFileRepository

	beUrl         string
	storageHelper storage.StorageHelper
}

func NewMediaChapterService(
	mediaRepo models.MediaRepository,
	mediaChapterRepo models.MediaChapterRepository,
	cdnFileRepo models.CdnFileRepository,

	beUrl string,
	storageHelper storage.StorageHelper,
) *MediaChapterService {
	return &MediaChapterService{
		mediaRepo:        mediaRepo,
		mediaChapterRepo: mediaChapterRepo,
		cdnFileRepo:      cdnFileRepo,

		beUrl:         beUrl,
		storageHelper: storageHelper,
	}
}

func (s *MediaChapterService) Create(
	ctx context.Context,
	mediaId uuid.UUID,
	language string,
	file multipart.File,
	authInfo models.AuthenticationInfo,
) (*models.MediaChapter, error) {
	media, err := s.mediaRepo.GetUserMediaById(ctx, authInfo.User.Id, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}

		return nil, response.NewInternalServerError(err)
	}

	existed, err := s.mediaChapterRepo.GetMediaChapterByMediaIdAndLanguage(ctx, media.Id, language)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, response.NewInternalServerError(err)
	}

	if existed != nil {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf(
				"Chapter of media with id %s for %s already exists.",
				mediaId,
				strings.ToLower(models.LanToLanguageMapping[language]),
			),
		)
	}

	chapter := models.NewMediaChapter(mediaId, language)
	resp, err := s.storageHelper.Upload(ctx, mediaId.String(), file)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if err := s.mediaChapterRepo.Create(ctx, chapter); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if err := s.mediaChapterRepo.CreateFile(ctx, &models.MediaChapterFile{
		FileId:    resp.Id,
		ChapterId: chapter.Id,
		File: models.NewCdnFile(
			authInfo.User.Id,
			resp.Id,
			resp.Size,
			resp.Offset,
			1,
			models.CdnChapterType,
		),
	}); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	chapter.Url = chapter.GetUrl(media.Secret)

	return chapter, nil
}

func (s *MediaChapterService) GetMediaChapters(
	ctx context.Context,
	mediaId uuid.UUID,
	offset, limit int,
	authInfo models.AuthenticationInfo,
) ([]*models.MediaChapter, int64, error) {
	media, err := s.mediaRepo.GetUserMediaById(ctx, authInfo.User.Id, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, response.NewNotFoundError(err)
		}

		return nil, 0, response.NewInternalServerError(err)
	}

	chapters, total, err := s.mediaChapterRepo.GetMediaChaptersByMediaIdWithLimitAndOffset(
		ctx,
		media.Id,
		offset,
		limit,
	)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}

	for _, chapter := range chapters {
		chapter.Url = chapter.GetUrl(media.Secret)
	}

	return chapters, total, nil
}

func (s *MediaChapterService) DeleteMediaChapter(
	ctx context.Context,
	mediaId uuid.UUID,
	language string,
	authInfo models.AuthenticationInfo,
) error {
	media, err := s.mediaRepo.GetUserMediaById(ctx, authInfo.User.Id, mediaId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	chapter, err := s.mediaChapterRepo.GetMediaChapterByMediaIdAndLanguage(ctx, media.Id, language)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if chapter.File != nil {
		if err := s.storageHelper.Delete(ctx, &storage.Object{
			Id: chapter.File.FileId,
		}); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	if err := s.mediaChapterRepo.DeleteMediaChapterById(ctx, chapter.Id); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}
