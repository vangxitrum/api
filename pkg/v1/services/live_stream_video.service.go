package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payment"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/stream"

	payment_gateway_model "github.com/AIOZNetwork/payment/models"
)

type LiveStreamMediaService struct {
	liveStreamMediaRepo models.LiveStreamMediaRepository
	mediaRepo           models.MediaRepository
	usageRepo           models.UsageRepository

	paymentClient     *payment.PaymentClient
	storagePath       string
	outputPath        string
	streamClient      *stream.StreamClient
	liveStreamKeyRepo models.LiveStreamKeyRepository
	redisUuidDb       *redis.Client
	redisConnIdDb     *redis.Client
}

func NewLiveStreamMediaService(
	liveStreamMediaRepo models.LiveStreamMediaRepository,
	mediaRepo models.MediaRepository,
	usageRepo models.UsageRepository,

	paymentClient *payment.PaymentClient,
	storagePath string,
	outputPath string,
	streamClient *stream.StreamClient,
	liveStreamKeyRepo models.LiveStreamKeyRepository,
	redisUuidDb *redis.Client,
	redisConnIdDb *redis.Client,
) *LiveStreamMediaService {
	return &LiveStreamMediaService{
		liveStreamMediaRepo: liveStreamMediaRepo,
		mediaRepo:           mediaRepo,
		usageRepo:           usageRepo,

		liveStreamKeyRepo: liveStreamKeyRepo,
		paymentClient:     paymentClient,
		storagePath:       storagePath,
		outputPath:        outputPath,
		streamClient:      streamClient,
		redisUuidDb:       redisUuidDb,
		redisConnIdDb:     redisConnIdDb,
	}
}

var mapStatus = map[string]bool{
	models.LiveStreamStatusStreaming: true,
	models.LiveStreamStatusCreated:   true,
}

func (s *LiveStreamMediaService) CreateLiveStreamMedia(
	ctx context.Context,
	media *models.Media,
	streamKeyId uuid.UUID,
	userId uuid.UUID,
	status string,
	input models.CreateStreamingRequest,
) (*models.LiveStreamMedia, error) {
	if status == "" {
		status = models.LiveStreamStatusStreaming
	}

	if !mapStatus[status] {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Status not allowed."),
		)
	}

	wallet, err := s.paymentClient.GetWalletByUserId(ctx, userId)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}
	if input.Save &&
		wallet.Balance.Add(wallet.FreeBalance).LessThan(models.MinBalance) {
		input.Save = false
	}

	if err := s.UpdateLiveStreamMediaObjectStatus(ctx, media, models.HiddenStatus); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	lsKey, err := s.liveStreamKeyRepo.GetLiveStreamKeyById(ctx, streamKeyId)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	liveStreamMedia := models.NewLiveStreamMedia(
		media.Id,
		streamKeyId,
		userId,
		status,
		input,
		lsKey.Type,
	)
	if err := s.liveStreamMediaRepo.Create(ctx, liveStreamMedia); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if status == models.LiveStreamStatusStreaming {
		liveStreamMedia.StreamedAt = time.Now().UTC()
	}

	_, err = s.liveStreamKeyRepo.GetLiveStreamKeyById(ctx, streamKeyId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	return liveStreamMedia, nil
}

func (s *LiveStreamMediaService) CreateVideoObject(
	ctx context.Context,
	media *models.Media,
	liveStreamKeyId uuid.UUID,
) (*models.Media, error) {
	media.Metadata = models.JsonB{
		"live_stream_id": liveStreamKeyId.String(),
	}

	if err := s.mediaRepo.Create(
		ctx,
		media,
	); err != nil {
		return nil, response.NewInternalServerError(err)
	}

	return media, nil
}

func (s *LiveStreamMediaService) UpdateLiveStreamMediaObjectStatus(
	ctx context.Context,
	media *models.Media,
	status string,
) error {
	media.Status = status
	if err := s.mediaRepo.UpdateMedia(ctx, media); err != nil {
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *LiveStreamMediaService) GetCreated(
	ctx context.Context,
	streamKeyId, userId uuid.UUID,
) (*models.LiveStreamMedia, error) {
	lsMedia, err := s.liveStreamMediaRepo.GetCreated(ctx, streamKeyId, userId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, response.NewInternalServerError(err)
	}
	return lsMedia, nil
}

func (s *LiveStreamMediaService) UpdateLiveStreamMedia(
	ctx context.Context,
	id, userId uuid.UUID,
	payload *models.UpdateLiveStreamMediaInput,
) error {
	liveStreamMedia, err := s.liveStreamMediaRepo.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}
	if liveStreamMedia.UserId != userId {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("You are not allowed to update this live stream media."),
		)
	}

	liveStreamMedia.Title = payload.Title

	if payload.Save {
		wallet, err := s.paymentClient.GetWalletByUserId(ctx, userId)
		if err != nil {
			return response.NewInternalServerError(err)
		}
		if wallet.Balance.Add(wallet.FreeBalance).LessThan(models.MinBalance) {
			return response.NewBadRequestError(
				"Insufficient balance to save live stream media.",
			)
		}
	}

	liveStreamMedia.UpdatedAt = time.Now().UTC()
	liveStreamMedia.Save = payload.Save
	if err := s.liveStreamMediaRepo.UpdateLiveStreamMedia(ctx, liveStreamMedia); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}
	media, err := s.mediaRepo.GetUserMediaById(
		ctx,
		userId,
		liveStreamMedia.MediaId,
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

	if payload.Title != "" {
		media.Title = payload.Title
	}

	if err := s.mediaRepo.UpdateMedia(
		ctx,
		media,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *LiveStreamMediaService) UpdateLiveStreamMediaStatus(
	ctx context.Context,
	connId string,
	liveStreamMedia *models.LiveStreamMedia,
	status string,
) error {
	liveStreamMedia.Status = status

	liveStreamMedia.UpdatedAt = time.Now().UTC()

	if status == models.LiveStreamStatusStreaming {
		liveStreamMedia.StreamedAt = time.Now().UTC()
	}

	if connId != "" {
		liveStreamMedia.ConnectionId = connId
	}

	if status == models.LiveStreamStatusEnd {
		if err := s.usageRepo.CreateLog(
			ctx,
			(&models.UsageLogBuilder{}).
				SetUserId(liveStreamMedia.UserId).
				SetLivestreamDuration(float64(time.Since(liveStreamMedia.StreamedAt).Seconds())).
				SetIsUserCost(false).
				Build(),
		); err != nil {
			slog.ErrorContext(
				ctx,
				"Create usage log error",
				slog.Any("err", err),
			)
		}
	}

	if err := s.liveStreamMediaRepo.UpdateLiveStreamMedia(ctx, liveStreamMedia); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *LiveStreamMediaService) UpdateMediaStatusToDeletedIfNeeded(
	ctx context.Context,
	status string,
	mediaId uuid.UUID,
) error {
	if status == models.DeletedStatus {
		if err := s.mediaRepo.UpdateMediaStatusById(ctx, mediaId, status); err != nil {
			return response.NewInternalServerError(err)
		}
	}
	return nil
}

func (s *LiveStreamMediaService) GetNotSavedLiveStreamMedias(
	ctx context.Context,
) ([]*models.LiveStreamMedia, error) {
	lsMedias, err := s.liveStreamMediaRepo.GetNotSavedLiveStreamMedias(ctx)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}
	return lsMedias, nil
}

func (s *LiveStreamMediaService) GetLiveStreamMediaById(
	ctx context.Context,
	id uuid.UUID,
) (*models.LiveStreamMedia, error) {
	liveStreamMedia, err := s.liveStreamMediaRepo.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}
	return liveStreamMedia, nil
}

func (s *LiveStreamMediaService) GetLiveStreamMediaByIdAndUserId(
	ctx context.Context,
	userId uuid.UUID,
	liveStreamKeyId uuid.UUID,
) (*models.LiveStreamMedia, error) {
	liveStreamMedia, err := s.liveStreamMediaRepo.GetById(ctx, liveStreamKeyId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if liveStreamMedia.UserId != userId {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("You are not allowed to access this live stream media."),
		)
	}

	return liveStreamMedia, nil
}

func (s *LiveStreamMediaService) GetLiveStreamMedias(
	ctx context.Context,
	userId uuid.UUID,
	filter models.GetLiveStreamMediasFilter,
) ([]*models.LiveStreamMedia, int64, error) {
	liveStreamMedias, total, err := s.liveStreamMediaRepo.GetLiveStreamMedias(
		ctx,
		userId,
		filter,
	)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}
	return liveStreamMedias, total, nil
}

func (s *LiveStreamMediaService) GetLiveStreamMediaStreamings(
	ctx context.Context,
	liveStreamKeyId uuid.UUID,
	payload models.GetStreamingsFilter,
	userId uuid.UUID,
) ([]*models.LiveStreamMedia, int64, error) {
	liveStreamMedias, total, err := s.liveStreamMediaRepo.GetUserLiveStreamMediaByLiveStreamKeyId(
		ctx,
		userId,
		liveStreamKeyId,
		payload,
	)
	if err != nil {
		return nil, 0, response.NewInternalServerError(err)
	}

	return liveStreamMedias, total, nil
}

func (s *LiveStreamMediaService) GetLiveStreamMediaStreaming(
	ctx context.Context,
	userId uuid.UUID,
	liveStreamKeyId uuid.UUID,
	streamId uuid.UUID,
) (*models.LiveStreamMedia, error) {
	lsv, err := s.liveStreamMediaRepo.GetLiveStreamMediaStreaming(
		ctx,
		userId,
		liveStreamKeyId,
		streamId,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	return lsv, nil
}

func (s *LiveStreamMediaService) DeleteLiveStreamMedia(
	ctx context.Context,
	userId uuid.UUID,
	liveStreamMedia *models.LiveStreamMedia,
) error {
	if liveStreamMedia.UserId != userId {
		return response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("You are not allowed to delete this live stream media."),
		)
	}

	if err := s.liveStreamMediaRepo.DeleteLiveStreamMedia(ctx, liveStreamMedia.Id, userId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	_, err := s.liveStreamKeyRepo.GetLiveStreamKeyById(
		ctx,
		liveStreamMedia.LiveStreamKeyId,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.NewNotFoundError(err)
		}
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *LiveStreamMediaService) UpdateEndLiveStreamMedia(
	ctx context.Context,
) error {
	if err := s.liveStreamMediaRepo.UpdateEndLiveStreamMedia(ctx); err != nil {
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *LiveStreamMediaService) HandleLiveStreamDuration(
	ctx context.Context,
) error {
	liveStreamings, err := s.liveStreamMediaRepo.GetListLiveStreamingMedias(ctx)
	if err != nil {
		slog.Error(
			"Error fetching live streamings",
			slog.Any("err", err),
		)
		return err
	}

	if len(liveStreamings) == 0 {
		return nil
	}

	for _, liveStreaming := range liveStreamings {
		liveStreamDetails, err := s.liveStreamMediaRepo.GetById(
			ctx,
			liveStreaming.Id,
		)
		if err != nil {
			slog.Error(
				"Error fetching live stream details",
				slog.Any("stream_id", liveStreaming.Id),
				slog.Any("err", err),
			)
			return err
		}

		if err := s.checkLiveStreamingDuration(ctx, liveStreamDetails); err != nil {
			slog.Error(
				"Error checking duration for stream",
				slog.Any("stream_id", liveStreamDetails.Id),
				slog.Any("err", err),
			)
			return err
		}
	}
	return nil
}

func (s *LiveStreamMediaService) checkLiveStreamingDuration(
	ctx context.Context,
	liveStreamDetails *models.LiveStreamMedia,
) error {
	muxerPath, err := s.streamClient.GetHLSMuxerInfo(
		ctx,
		liveStreamDetails.Id.String(),
	)
	if err != nil {
		slog.Error(
			"Error fetching HLS muxer info",
			slog.Any("err", err),
		)
		return err
	}

	if muxerPath == "" {
		return nil
	}

	if err := s.liveStreamMediaRepo.EndLiveStream(ctx, liveStreamDetails.Id); err != nil {
		slog.Error(
			"Error ending live stream",
			slog.Any("err", err),
		)
		return err
	}

	if err := s.streamClient.KickRTMPConnection(ctx, liveStreamDetails.ConnectionId); err != nil {
		slog.Error(
			"Error kicking RTMP connection",
			slog.Any("err", err),
		)
		return err
	}

	if liveStreamDetails.Save {
		if err := s.UpdateLiveStreamMediaStatus(ctx, "", liveStreamDetails, models.LiveStreamStatusEnd); err != nil {
			slog.Error(
				"Error updating live stream status",
				slog.Any("err", err),
			)
			return err
		}
	} else {
		if err := s.UpdateMediaStatusToDeletedIfNeeded(ctx, models.DeletedStatus, liveStreamDetails.MediaId); err != nil {
			slog.Error(
				"Error deleting record by path",
				slog.Any("err", err),
			)
			return err
		}
	}

	return nil
}

func (s *LiveStreamMediaService) UpdateLiveStreamView(
	ctx context.Context,
) error {
	timeRange := time.Now().UTC().Add(-2 * time.Minute)
	if err := s.liveStreamMediaRepo.UpdateLiveStreamView(ctx, timeRange); err != nil {
		return response.NewInternalServerError(err)
	}
	return nil
}

func (s *LiveStreamMediaService) GetUserLiveStreamMediaByIdAndLiveStreamKeyId(
	ctx context.Context,
	userId uuid.UUID,
	liveStreamKeyId uuid.UUID,
	liveStreamMediaId uuid.UUID,
) (*models.LiveStreamMedia, error) {
	liveStreamMedia, err := s.liveStreamMediaRepo.GetUserLiveStreamMediaByIdAndLiveStreamKeyId(
		ctx,
		userId,
		liveStreamKeyId,
		liveStreamMediaId,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if liveStreamMedia.UserId != userId {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("You are not allowed to access this live stream media."),
		)
	}

	if liveStreamMedia.Status != models.LiveStreamStatusCreated {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf(
				"You can only delete live stream media when it's created.",
			),
		)
	}

	return liveStreamMedia, nil
}

func (s *LiveStreamMediaService) HandleNewLiveStreamMediaByConnId(
	ctx context.Context,
	webhookResp *models.LiveStreamWebhookResponse,
	streamPath uuid.UUID,
	connId string,
) error {
	parsedUUIDStreamKey, err := uuid.Parse(webhookResp.StreamKey)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	liveStreamKey, err := s.liveStreamKeyRepo.GetByLiveStreamKey(
		ctx,
		parsedUUIDStreamKey,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := s.streamClient.KickRTMPConnection(ctx, webhookResp.ID); err != nil {
				slog.Error(
					"Error kicking RTMP connection",
					slog.Any("err", err),
				)
				return err
			}
		}
		return response.NewInternalServerError(err)
	}

	newMedia, err := models.NewMedia(
		liveStreamKey.UserId,
		liveStreamKey.Type,
		liveStreamKey.Name,
		"",
		nil,
		models.DefaultVideoConfig,
		models.DefaultSegmentDuration,
		nil,
		true,
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	newMedia.Status = models.HiddenStatus
	if _, err := s.CreateVideoObject(
		ctx,
		newMedia,
		liveStreamKey.Id,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	save := true
	wallet, err := s.paymentClient.GetWalletByUserId(ctx, liveStreamKey.UserId)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if wallet.Balance.Add(wallet.FreeBalance).LessThan(models.MinBalance) {
		save = false
	}

	liveStreamMedia := models.NewLiveStreamMediaWithId(
		streamPath,
		newMedia.Id,
		liveStreamKey.Id,
		liveStreamKey.UserId,
		models.LiveStreamStatusStreaming,
		models.CreateStreamingRequest{
			Title: liveStreamKey.Name,
			Save:  save,
		},
		liveStreamKey.Type,
	)

	if err := s.liveStreamMediaRepo.Create(ctx, liveStreamMedia); err != nil {
		return response.NewInternalServerError(err)
	}

	if err := s.UpdateLiveStreamMediaStatus(ctx, connId, liveStreamMedia, models.LiveStreamStatusStreaming); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *LiveStreamMediaService) GetConnectLiveStreamMediaIdByConnId(
	ctx context.Context,
	connId string,
) error {
	liveStreamMediaPath, err := s.streamClient.GetStreamPathWithStreamType(
		ctx,
		connId,
		models.StreamConnectPath,
	)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	parsedUUIDPath, err := uuid.Parse(liveStreamMediaPath.Path)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if parsedUUIDPath == uuid.Nil {
		return response.NewNotFoundError(fmt.Errorf("Stream not found."))
	}

	liveStreamMedia, err := s.liveStreamMediaRepo.GetById(ctx, parsedUUIDPath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			if err := s.HandleNewLiveStreamMediaByConnId(ctx, liveStreamMediaPath, parsedUUIDPath, connId); err != nil {
				return response.NewInternalServerError(err)
			}
			return nil
		}
		return response.NewInternalServerError(err)
	}

	if err := s.UpdateLiveStreamMediaStatus(ctx, connId, liveStreamMedia, models.LiveStreamStatusStreaming); err != nil {
		return response.NewInternalServerError(err)
	}

	return nil
}

func (s *LiveStreamMediaService) HandleDisconnectLiveStreamMediaIdByConnId(
	ctx context.Context,
	connId string,
) error {
	liveStreamMedia, err := s.GetDisconnectLiveStreamMediaIdByConnId(
		ctx,
		connId,
	)
	if err != nil {
		return err
	}

	if err := s.UpdateLiveStreamMediaStatus(
		ctx,
		"",
		liveStreamMedia,
		models.LiveStreamStatusEnd,
	); err != nil {
		return response.NewInternalServerError(err)
	}

	// If error when delete in redis, we still delete in db and skip redis. Redis will be clean up later by TTL.
	_, err = s.redisUuidDb.Del(ctx, liveStreamMedia.Id.String(), connId).
		Result()
	if err != nil {
		slog.Error(
			"Error deleting from Redis",
			slog.Any("err", err),
		)
	}

	return nil
}

func (s *LiveStreamMediaService) GetDisconnectLiveStreamMediaIdByConnId(
	ctx context.Context,
	connId string,
) (*models.LiveStreamMedia, error) {
	liveStreamMediaPath, err := s.streamClient.GetStreamPathWithStreamType(
		ctx,
		connId,
		models.StreamDisconnectPath,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if liveStreamMediaPath != nil {
		return nil, response.NewNotFoundError(
			fmt.Errorf("Stream is still active."),
		)
	}

	liveStreamMedia, err := s.liveStreamMediaRepo.GetLiveStreamMediaByConnectionId(
		ctx,
		connId,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NewNotFoundError(err)
		}
		return nil, response.NewInternalServerError(err)
	}

	if liveStreamMedia.Status != models.LiveStreamStatusStreaming {
		return nil, response.NewHttpError(
			http.StatusBadRequest,
			fmt.Errorf("Live stream media is not streaming."),
		)
	}

	return liveStreamMedia, nil
}

func (s *LiveStreamMediaService) HandleLiveStreamMediaNotStream(
	ctx context.Context,
) ([]*models.LiveStreamMedia, error) {
	liveStreamMedia, err := s.liveStreamMediaRepo.GetListLiveStreamingMedias(
		ctx,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}

	if len(liveStreamMedia) == 0 {
		return liveStreamMedia, nil
	}

	var invalidStreamingMedia []*models.LiveStreamMedia
	for _, video := range liveStreamMedia {
		identityServer, err := s.redisUuidDb.Get(ctx, video.ConnectionId).
			Result()
		if err != nil {
			invalidStreamingMedia = append(invalidStreamingMedia, video)
			continue
		}

		isExist, err := s.redisConnIdDb.SIsMember(ctx, identityServer, video.ConnectionId).
			Result()
		if !isExist || err != nil {
			invalidStreamingMedia = append(invalidStreamingMedia, video)
			_ = s.redisUuidDb.Del(ctx, video.Id.String(), video.ConnectionId).
				Err()
		}
	}

	return invalidStreamingMedia, nil
}

func (s *LiveStreamMediaService) GetListLiveStreamingMedias(
	ctx context.Context,
) ([]*models.LiveStreamMedia, error) {
	liveStreamMedias, err := s.liveStreamMediaRepo.GetListLiveStreamingMedias(
		ctx,
	)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}
	return liveStreamMedias, nil
}

func (s *LiveStreamMediaService) GetWalletByUserId(
	ctx context.Context,
	userId uuid.UUID,
) (*payment_gateway_model.Wallet, error) {
	wallet, err := s.paymentClient.GetWalletByUserId(ctx, userId)
	if err != nil {
		return nil, response.NewInternalServerError(err)
	}
	return wallet, nil
}

func (s *LiveStreamMediaService) EnsureEnoughBalanceIfSave(ctx context.Context, userId uuid.UUID, save bool) error {
	if !save {
		return nil
	}

	wallet, err := s.GetWalletByUserId(ctx, userId)
	if err != nil {
		return response.NewInternalServerError(err)
	}

	if wallet.Balance.LessThan(models.MinBalance) {
		return response.NewBadRequestError("Insufficient balance to save live stream media.")
	}

	return nil
}

func (s *LiveStreamMediaService) CheckLiveStreamIsExist(ctx context.Context, streamKeyId uuid.UUID, userId uuid.UUID) bool {
	lsMedia, err := s.liveStreamMediaRepo.GetCreated(ctx, streamKeyId, userId)
	if err != nil || lsMedia == nil {
		return false
	}

	return true
}
