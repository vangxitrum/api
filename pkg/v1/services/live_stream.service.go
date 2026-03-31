package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/stream"
	"10.0.0.50/tuan.quang.tran/vms-v2/rabbitmq"
)

type LiveStreamService struct {
	liveStreamKeyRepo       models.LiveStreamKeyRepository
	liveStreamMediaRepo     models.LiveStreamMediaRepository
	liveStreamMulticastRepo models.LiveStreamMulticastRepository
	mediaRepo               models.MediaRepository
	cdnFileRepo             models.CdnFileRepository
	storageHelper           storage.StorageHelper

	endLiveStreamChannel *rabbitmq.RabbitMQ

	uploaderUrl  string
	streamClient *stream.StreamClient
}

func NewLiveStreamService(
	liveStreamKeyRepo models.LiveStreamKeyRepository,
	liveStreamMediaRepo models.LiveStreamMediaRepository,
	liveStreamMulticastRepo models.LiveStreamMulticastRepository,
	mediaRepo models.MediaRepository,
	cdnFileRepo models.CdnFileRepository,
	storageHelper storage.StorageHelper,
	endLiveStreamChannel *rabbitmq.RabbitMQ,
	uploaderUrl string,
	streamClient *stream.StreamClient,
) *LiveStreamService {
	service := &LiveStreamService{
		liveStreamKeyRepo:       liveStreamKeyRepo,
		liveStreamMediaRepo:     liveStreamMediaRepo,
		liveStreamMulticastRepo: liveStreamMulticastRepo,
		mediaRepo:               mediaRepo,
		cdnFileRepo:             cdnFileRepo,
		storageHelper:           storageHelper,
		uploaderUrl:             uploaderUrl,
		endLiveStreamChannel:    endLiveStreamChannel,
		streamClient:            streamClient,
	}

	service.startEndLiveStream(context.Background())

	return service
}

func (s *LiveStreamService) CreateLiveStreamKey(
	ctx context.Context,
	authInfo models.AuthenticationInfo,
	input models.CreateLiveStreamKeyInput,
) (*models.LiveStreamKey, error) {
	liveStream := models.NewLiveStreamKey(authInfo.User.Id, input)
	if err := s.liveStreamKeyRepo.CreateLiveStreamKey(ctx, liveStream); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return liveStream, nil
}

func (s *LiveStreamService) GetLiveStreamKeyById(
	ctx context.Context,
	id uuid.UUID,
	authInfo models.AuthenticationInfo,
) (*models.LiveStreamKey, error) {
	lsKey, err := s.liveStreamKeyRepo.GetLiveStreamKeyById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if lsKey.UserId != authInfo.User.Id {
		return nil, response.NewBadRequestError(
			"You are not allowed to access this live stream key.",
		)
	}

	return lsKey, nil
}

func (s *LiveStreamService) GetLiveStreamKeyByStreamKey(
	ctx context.Context,
	streamKey uuid.UUID,
) (*models.LiveStreamKey, error) {
	lsKey, err := s.liveStreamKeyRepo.GetByLiveStreamKey(ctx, streamKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, err
	}

	lsKey.UpdatedAt = time.Now().UTC()
	if err := s.liveStreamKeyRepo.Update(context.Background(), lsKey); err != nil {
		return nil, err
	}
	return lsKey, nil
}

func (s *LiveStreamService) GetLiveStreamKeys(
	ctx context.Context,
	userId uuid.UUID,
	payload models.GetLiveStreamKeysFilter,
) ([]*models.LiveStreamKey, int64, error) {
	liveStreamKeys, total, err := s.liveStreamKeyRepo.GetLiveStreamKeys(ctx, payload, userId)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}
	for _, key := range liveStreamKeys {
		_, total, err := s.liveStreamMediaRepo.GetAllLiveStreamMedia(
			ctx,
			userId,
			key.Id,
			models.LiveStreamStatusStreaming,
		)
		if err != nil {
			return nil, 0, response.NewInternalServerError(err)
		}
		key.TotalLiveStreaming = total
		_, total, err = s.liveStreamMediaRepo.GetSavedLiveStreamMedias(ctx, userId, key.Id)
		if err != nil {
			return nil, 0, response.NewInternalServerError(err)
		}
		key.TotalSaveMedia = total
	}

	return liveStreamKeys, total, nil
}

func (s *LiveStreamService) UpdateLiveStreamKey(
	ctx context.Context,
	userId uuid.UUID,
	id uuid.UUID,
	input models.UpdateLiveStreamKeyInput,
) (*models.LiveStreamKey, error) {
	liveStreamKey, err := s.liveStreamKeyRepo.GetLiveStreamKeyById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if liveStreamKey.UserId != userId {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("You are not allowed to update this live stream key."),
		)
	}

	updatedLiveStreamKey, err := s.liveStreamKeyRepo.UpdateLiveStreamKey(ctx, userId, id, input)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return updatedLiveStreamKey, nil
}

func (s *LiveStreamService) DeleteLiveStreamKey(
	ctx context.Context,
	id uuid.UUID,
	userId uuid.UUID,
) error {
	liveStreamKey, err := s.liveStreamKeyRepo.GetLiveStreamKeyById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	if liveStreamKey.UserId != userId {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("You are not allowed to delete this live stream key."),
		)
	}

	existingLiveStreamMedia, _, err := s.liveStreamMediaRepo.GetAllLiveStreamMedia(
		ctx,
		userId,
		id,
		models.LiveStreamStatusStreaming,
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if len(existingLiveStreamMedia) > 0 {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("You cannot delete this live stream key because it has live stream media."),
		)
	}

	existingDoneLiveStreamMedia, _, err := s.liveStreamMediaRepo.GetAllLiveStreamMedia(
		ctx,
		userId,
		id,
		models.DoneStatus,
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	for _, media := range existingDoneLiveStreamMedia {
		existingMedia, err := s.mediaRepo.GetMediaById(ctx, media.MediaId)
		if err != nil {
			return response.NewInternalServerError(err)
		}

		if !existingMedia.IsDone() {
			return response.NewHttpError(
				http.StatusBadRequest,
				fmt.Errorf("This media is not ready. You cannot delete this live stream key."),
			)
		}
	}

	if err := s.liveStreamMediaRepo.DeleteLiveStreamMedias(ctx, id, userId); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.liveStreamKeyRepo.DeleteUserLiveStreamKey(ctx, id, userId); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *LiveStreamService) startEndLiveStream(ctx context.Context) {
	messCh, err := s.endLiveStreamChannel.Consume("end_live_stream")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			for mess := range messCh {
				var lsMedia models.LiveStreamMedia
				if err := json.Unmarshal(mess.Body, &lsMedia); err != nil {
					log.Error().Err(err).Msg("Unmarshal error")
					continue
				}

				currentStateLsMedia, err := s.liveStreamMediaRepo.GetById(ctx, lsMedia.Id)
				if err != nil {
					log.Error().
						Err(err).
						Str("LiveStreamID", lsMedia.Id.String()).
						Msg("CheckAlive: EndLiveStream")
					continue
				}

				if err := s.liveStreamMediaRepo.EndLiveStream(ctx, currentStateLsMedia.Id); err != nil {
					err := fmt.Errorf("update file record id err=%s", err.Error())
					log.Error().
						Err(err).
						Str("LiveStreamID", currentStateLsMedia.Id.String()).
						Msg("CheckAlive: EndLiveStream")
					continue
				}

				if !currentStateLsMedia.Save {
					if err := s.liveStreamMediaRepo.DeleteLiveStreamMedia(ctx, currentStateLsMedia.Id, currentStateLsMedia.UserId); err != nil {
						log.Error().
							Err(err).
							Str("LiveStreamID", currentStateLsMedia.Id.String()).
							Msg("CheckAlive: EndLiveStream")
						continue
					}

					if err := s.mediaRepo.DeleteMediaById(ctx, currentStateLsMedia.MediaId); err != nil {
						log.Error().
							Err(err).
							Str("LiveStreamID", currentStateLsMedia.Id.String()).
							Msg("CheckAlive: EndLiveStream")
						continue
					}
				}
			}
		}
	}()
}

func (s *LiveStreamService) DeleteUserLivestreamData(
	ctx context.Context,
	userId uuid.UUID,
) error {
	if err := s.liveStreamMediaRepo.DeleteUserLivestreamMedia(ctx, userId); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.liveStreamMulticastRepo.DeleteUserLivestreamMulticasts(ctx, userId); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.liveStreamKeyRepo.DeleteAllUserLivestreamKey(ctx, userId); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}
