package services

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
)

type WatermarkService struct {
	watermarkRepo models.WatermarkRepository
	cdnFileRepo   models.CdnFileRepository
	usageRepo     models.UsageRepository

	storageHelper storage.StorageHelper
}

func NewWatermarkService(
	watermarkRepo models.WatermarkRepository,
	cdnFileRepo models.CdnFileRepository,
	usageRepo models.UsageRepository,

	storageHelper storage.StorageHelper,
) *WatermarkService {
	return &WatermarkService{
		watermarkRepo: watermarkRepo,
		cdnFileRepo:   cdnFileRepo,
		usageRepo:     usageRepo,

		storageHelper: storageHelper,
	}
}

func (s *WatermarkService) UploadWatermark(
	ctx context.Context,
	userId uuid.UUID,
	fullFileName string,
	width, height int64,
	reader io.Reader,
) (
	*models.Watermark, error,
) {
	newWatermark := models.NewWatermark(
		userId,
		width,
		height,
		fullFileName,
	)
	watermark, err := s.watermarkRepo.CreateUserWatermark(
		ctx,
		newWatermark,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if closer, ok := reader.(io.Closer); ok {
		defer closer.Close()
	}
	resp, err := s.storageHelper.Upload(
		ctx,
		watermark.Id.String(),
		reader,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if err := s.watermarkRepo.CreateFile(
		ctx, &models.WaterMarkFile{
			WaterMarkId: watermark.Id,
			FileId:      resp.Id,
			File: models.NewCdnFile(
				userId,
				resp.Id,
				resp.Size,
				resp.Offset,
				1,
				models.CdnWatermarkType,
			),
		},
	); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if err := s.usageRepo.CreateLog(
		ctx,
		(&models.UsageLogBuilder{}).
			SetUserId(userId).
			SetStorage(resp.Size).
			SetIsUserCost(false).
			Build(),
	); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return watermark, nil
}

func (s *WatermarkService) ListAllWatermarks(
	ctx context.Context,
	params models.GetWatermarkList,
) ([]*models.Watermark, int64, error) {
	result, total, err := s.watermarkRepo.ListAllWatermarks(
		ctx, models.GetWatermarkList{
			UserId: params.UserId,
			SortBy: params.SortBy,
			Order:  params.Order,
			Offset: params.Offset,
			Limit:  params.Limit,
		},
	)
	if err != nil {
		return nil, 0, response.NewHttpError(
			http.StatusInternalServerError, err,
			"Failed to get watermark list.",
		)
	}

	return result, total, nil
}

func (s *WatermarkService) DeleteWatermarkById(
	ctx context.Context,
	userId, watermarkId uuid.UUID,
) error {
	result, err := s.watermarkRepo.CheckIfWatermarkExistInAnyMedia(
		ctx,
		watermarkId,
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if result {
		return response.NewHttpError(
			http.StatusBadRequest,
			nil,
			"Watermark is being used in media.",
		)
	}

	watermark, err := s.watermarkRepo.GetUserWatermarkById(ctx, userId, watermarkId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}

		return response.NewInternalServerError(err)
	}

	if watermark.File != nil {
		if err := s.storageHelper.Delete(
			ctx,
			&storage.Object{
				Id: watermark.File.FileId,
			},
		); err != nil {
			return response.NewInternalServerError(err)
		}
	}

	if err := s.watermarkRepo.DeleteWatermarkById(
		ctx, userId, watermarkId,
	); err != nil {
		if errors.Is(
			err,
			gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}
