package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/message"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
)

type ReportContentService struct {
	reportContentRepo models.ReportContentRepository
	mediaRepo         models.MediaRepository
	liveStreamRepo    models.LiveStreamMediaRepository
	messageHelper     message.MessageHelper
}

func NewReportContentService(
	reportContentRepo models.ReportContentRepository,
	mediaRepo models.MediaRepository,
	liveStreamRepo models.LiveStreamMediaRepository,
	messageHelper message.MessageHelper,
) *ReportContentService {
	return &ReportContentService{
		reportContentRepo: reportContentRepo,
		mediaRepo:         mediaRepo,
		liveStreamRepo:    liveStreamRepo,
		messageHelper:     messageHelper,
	}
}

func (s *ReportContentService) CreateReportContent(
	ctx context.Context,
	mediaId uuid.UUID,
	reporterIp,
	mediaType,
	description string,
	reason models.ContentReportReason,
) error {
	existingReport, err := s.reportContentRepo.GetReportByMediaAndIP(ctx, mediaId, reporterIp)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return response.NewInternalServerError(err)
	}

	if existingReport != nil {
		return nil
	}
	newReport := models.NewContentReport(mediaId, reporterIp, description, mediaType, reason)
	if err := s.reportContentRepo.CreateReport(ctx, newReport); err != nil {
		return response.NewInternalServerError(err)
	}

	mediaUrl, err := s.buildMediaUrl(ctx, mediaId, mediaType)
	if err != nil {
		return response.NewHttpError(
			http.StatusNotFound,
			fmt.Errorf("Media not found."),
			"Media not found.",
		)
	}

	report := models.ContentReport{
		MediaId:     mediaId,
		Description: description,
		Reason:      reason,
	}

	mediaReportCount, err := s.reportContentRepo.GetTotalReportsByMediaId(ctx, mediaId)
	if err != nil {
		return response.NewInternalServerError(err)
	}
	err = s.messageHelper.SendReportMessage(
		ctx,
		mediaType,
		mediaUrl,
		reporterIp,
		mediaReportCount,
		report,
	)
	if err != nil {
		slog.Error("failed to send report message", slog.Any("err", err))
	}

	return nil
}

func (s *ReportContentService) buildMediaUrl(
	ctx context.Context,
	mediaId uuid.UUID,
	mediaType string,
) (string, error) {
	if mediaType == models.VideoMediaType {
		media, err := s.mediaRepo.GetMediaById(ctx, mediaId)
		if err == nil {
			return media.GetHlsPlayerUrl(), nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", response.NewInternalServerError(err)
		}
	}

	if mediaType == models.StreamMediaType {
		return models.BuildLiveStreamPlayerUrl(mediaId), nil
	}

	return "", fmt.Errorf("Media not found.")
}

func (s *ReportContentService) GetReportContentList(
	ctx context.Context,
	filter models.GetContentReportList,
) ([]*models.ContentReport, int64, error) {
	result, total, err := s.reportContentRepo.GetListReports(ctx, filter)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}
	return result, total, nil
}

func (s *ReportContentService) UpdateReportContent(
	ctx context.Context,
	reportId uuid.UUID,
	status string,
) error {
	existingReport, err := s.reportContentRepo.GetReportById(ctx, reportId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if existingReport.Status == status {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Status is already %s.", status),
		)
	}

	if err := s.reportContentRepo.UpdateReportStatus(ctx, reportId, status); err != nil {
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *ReportContentService) DeleteReportContent(ctx context.Context, reportId uuid.UUID) error {
	existingReport, err := s.reportContentRepo.GetReportById(ctx, reportId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}
	if err := s.reportContentRepo.DeleteReport(ctx, existingReport); err != nil {
		return response.NewInternalServerError(err)
	}
	return nil
}
