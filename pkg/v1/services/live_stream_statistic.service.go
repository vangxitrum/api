package services

import (
	"context"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"github.com/google/uuid"
)

type LiveStreamStatisticService struct {
	liveStreamStatisticRepo models.LiveStreamStatisticRepository
}

func NewLiveStreamStatisticService(
	liveStreamStatisticRepo models.LiveStreamStatisticRepository,
) *LiveStreamStatisticService {
	return &LiveStreamStatisticService{
		liveStreamStatisticRepo: liveStreamStatisticRepo,
	}
}

func (s *LiveStreamStatisticService) GetStatisticByStreamMediaId(
	ctx context.Context,
	streamMediaId uuid.UUID,
) (*models.LiveStreamStatisticResp, error) {
	liveStreamStatistic, err := s.liveStreamStatisticRepo.GetLiveStreamStatisticByStreamMediaId(ctx, streamMediaId)
	if err != nil {
		return nil, err
	}
	return liveStreamStatistic, nil
}
