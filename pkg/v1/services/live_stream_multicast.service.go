package services

import (
	"context"
	"errors"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LiveStreamMulticastService struct {
	liveStreamMulticastRepo models.LiveStreamMulticastRepository
	liveStreamKeyRepo       models.LiveStreamKeyRepository
}

func NewLiveStreamMulticastService(
	liveStreamMulticastRepo models.LiveStreamMulticastRepository,
	liveStreamKeyRepo models.LiveStreamKeyRepository,
) *LiveStreamMulticastService {
	return &LiveStreamMulticastService{
		liveStreamMulticastRepo: liveStreamMulticastRepo,
		liveStreamKeyRepo:       liveStreamKeyRepo,
	}
}

func (s *LiveStreamMulticastService) UpsertLiveStreamMulticastUrls(
	ctx context.Context,
	streamKey uuid.UUID,
	multicastUrls []string,
	authInfo models.AuthenticationInfo,
) (*models.LiveStreamMulticast, error) {

	// check userId is match with streamKey
	liveStreamKey, err := s.liveStreamKeyRepo.GetByLiveStreamKey(ctx, streamKey)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}
	if liveStreamKey.UserId != authInfo.User.Id {
		return nil, response.NewUnauthorizedError(errors.New("User does not own LiveStream Key."))
	}

	liveStreamMulticast := models.NewLiveStreamMulticast(
		liveStreamKey.Id,
		authInfo.User.Id,
		multicastUrls,
	)

	if err := s.liveStreamMulticastRepo.UpsertLiveStreamMulticast(ctx, liveStreamMulticast); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return liveStreamMulticast, nil
}

func (s *LiveStreamMulticastService) GetLiveStreamMulticastByStreamKey(
	ctx context.Context,
	streamKey uuid.UUID,
	authInfo models.AuthenticationInfo,
) (*models.LiveStreamMulticast, error) {

	liveStreamKey, err := s.liveStreamKeyRepo.GetByLiveStreamKey(ctx, streamKey)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}
	if liveStreamKey.UserId != authInfo.User.Id {
		return nil, response.NewUnauthorizedError(errors.New("User does not own LiveStream Key."))
	}

	liveStreamMulticast, err := s.liveStreamMulticastRepo.GetLiveStreamMulticastByStreamKeyId(ctx, liveStreamKey.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	return liveStreamMulticast, nil
}

func (s *LiveStreamMulticastService) DeleteLiveStreamMulticast(
	ctx context.Context,
	streamKey uuid.UUID,
	authInfo models.AuthenticationInfo,
) error {

	// check userId is match with streamKey
	liveStreamKey, err := s.liveStreamKeyRepo.GetByLiveStreamKey(ctx, streamKey)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}
	if liveStreamKey.UserId != authInfo.User.Id {
		return response.NewUnauthorizedError(errors.New("User does not own LiveStream Key."))
	}

	liveStreaming, err := s.liveStreamMulticastRepo.GetLiveStreamMulticastStreamingByStreamKeyId(ctx, liveStreamKey.Id)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return response.NewInternalServerError(err)
		}
		if liveStreaming != nil {
			return response.NewBadRequestError("You can't delete this stream because it is streaming.")
		}
	}

	err = s.liveStreamMulticastRepo.DeleteLiveStreamMulticast(ctx, liveStreamKey.Id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}
