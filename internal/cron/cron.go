package crons

import (
	"context"
	"log/slog"
	"time"

	"github.com/mdobak/go-xerrors"
	"github.com/robfig/cron/v3"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
)

var logger *slog.Logger

type CronOption func(*Cron)

type Cron struct {
	cron *cron.Cron

	usageService           *services.UsageService
	mailService            *services.MailService
	apiKeyService          *services.ApiKeyService
	mediaService           *services.MediaService
	userService            *services.UserService
	cdnUsageService        *services.CdnUsageService
	statisticService       *services.StatisticService
	liveStreamMediaService *services.LiveStreamMediaService
	liveStreamService      *services.LiveStreamService
	webhookService         *services.WebhookService
	mediaCaptionService    *services.MediaCaptionService
	playerThemeService     *services.PlayerThemeService
	playlistService        *services.PlaylistService

	isUploadingWaitingMedia bool
}

func NewCron(
	usageService *services.UsageService,
	mailService *services.MailService,
	apiKeyService *services.ApiKeyService,
	mediaService *services.MediaService,
	userService *services.UserService,
	cdUsageService *services.CdnUsageService,
	statisticService *services.StatisticService,
	liveStreamMediaService *services.LiveStreamMediaService,
	livestreamService *services.LiveStreamService,
	webhookService *services.WebhookService,
	mediaCaptionService *services.MediaCaptionService,
	playerThemeService *services.PlayerThemeService,
	playlistService *services.PlaylistService,

	opts ...CronOption,
) *Cron {
	c := &Cron{
		cron: cron.New(),

		usageService:           usageService,
		mailService:            mailService,
		apiKeyService:          apiKeyService,
		mediaService:           mediaService,
		userService:            userService,
		cdnUsageService:        cdUsageService,
		statisticService:       statisticService,
		liveStreamMediaService: liveStreamMediaService,
		liveStreamService:      livestreamService,
		webhookService:         webhookService,
		mediaCaptionService:    mediaCaptionService,
		playerThemeService:     playerThemeService,
		playlistService:        playlistService,
	}

	for _, opt := range opts {
		opt(c)
	}

	if logger == nil {
		logger = slog.Default().With("group", "cron")
	}

	return c
}

func WithLogger(l *slog.Logger) CronOption {
	return func(c *Cron) {
		logger = l
	}
}

func (c *Cron) Stop() <-chan struct{} {
	return c.cron.Stop().Done()
}

func (c *Cron) Start() {
	ctx := context.Background()

	c.cron.AddFunc("@every 30s", func() {
		start := time.Now()
		defer func() {
			logger.Debug(
				"Run cron: delete pending media",
				slog.Any("runtime", time.Since(start).Seconds()),
			)
		}()
		if err := c.mediaService.DeletePendingMedia(ctx); err != nil {
			slog.Error(
				"Delete pending media error",
				slog.Any("err", xerrors.New(err)),
			)
		}

		if err := c.userService.UpdateUsersFreeBalance(ctx); err != nil {
			slog.Error(
				"Update users free balance error",
				slog.Any("err", xerrors.New(err)),
			)
		}
	})

	c.cron.AddFunc("@every 10s", func() {
		c.handleUsage(ctx)
		c.handleFailUsage(ctx)

		c.handleNotSaveLiveStreamMedia(ctx)
		c.handleWebhookRetry(ctx)
		c.updateMediasStatus(ctx)
		c.uploadWaitingMediaSource(ctx)
		c.deleteUsers(ctx)
	})

	c.cron.AddFunc("@every 15s", func() {
		c.calculateUsersUsage(ctx)
		if err := c.mediaService.GenerateMediaCaptions(ctx); err != nil {
			slog.Error("generate caption generation", slog.Any("err", err))
		}

		if err := c.mediaCaptionService.WatchCaptionsGeneration(ctx); err != nil {
			slog.Error("watch caption generation", slog.Any("err", err))
		}

		if err := c.mediaService.HandleLiveStreamMedias(ctx); err != nil {
			slog.Error(
				"Handle live stream media error",
				slog.Any("err", xerrors.New(err)),
			)
		}
	})

	// every 1 hour
	c.cron.AddFunc("0 * * * *", func() {
		c.createUsersUsage(ctx)

		c.handleLowWalletBalance(ctx)
		c.handleContentOutOfBalanceUser(ctx)

		c.deleteExpiredApiKey(ctx)
		c.deleteExpiredEmailType(ctx)
	})

	// monthly 7am
	c.cron.AddFunc("0 7 1 * *", func() {
		c.handleMonthlyReceipt(ctx)
	})

	c.cron.AddFunc("@every 1m", func() {
		c.calculateMediaView(ctx)
	})

	c.cron.AddFunc("@every 20s", func() {
		c.handleLiveStreamMediaNotStream(ctx)
	})

	c.cron.AddFunc("@every 5s", func() {
		c.handleLiveStreamView(ctx)
	})

	c.cron.AddFunc("@every 30s", func() {
		c.handleLiveStreamDuration(ctx)
	})

	c.createUsersUsage(ctx)
	c.handleUsage(ctx)
	c.cron.Start()
}

func (c *Cron) deleteUsers(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: delete users",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()

	if err := func() error {
		users, err := c.userService.GetDeleteingUsers(ctx)
		if err != nil {
			slog.Error(
				"Delete users error",
				slog.Any("err", xerrors.New(err)),
			)
		}

		for _, user := range users {
			if err := c.liveStreamService.DeleteUserLivestreamData(ctx, user.Id); err != nil {
				return err
			}

			if err := c.playlistService.DeleteUserPlaylists(ctx, user.Id); err != nil {
				return err
			}

			if err := c.webhookService.DeleteUserWebhooks(ctx, user.Id); err != nil {
				return err
			}

			if err := c.apiKeyService.DeleteUserApiKeys(ctx, user.Id); err != nil {
				return err
			}

			if err := c.playerThemeService.DeleteUserPlayerThemes(ctx, user.Id); err != nil {
				return err
			}

			if err := c.mediaService.DeleteUserMedias(ctx, user); err != nil {
				return err
			}

			if err := c.userService.UpdateUserStatus(ctx, user.Id, models.DeletedStatus); err != nil {
				return err
			}

		}

		return nil
	}(); err != nil {
		slog.Error(
			"Delete users error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) updateMediasStatus(ctx context.Context) {
	if err := c.mediaService.UpdateMediasStatus(ctx); err != nil {
		slog.Error(
			"Update media status error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) createUsersUsage(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: create user usage",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.usageService.CreateUsersUsage(ctx); err != nil {
		slog.Error(
			"Create user usage error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) calculateMediaView(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: calculate media view",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.statisticService.CalculateMediaView(ctx); err != nil {
		slog.Error(
			"Calculate media view error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) uploadWaitingMediaSource(ctx context.Context) error {
	if c.isUploadingWaitingMedia {
		return nil
	}

	c.isUploadingWaitingMedia = true
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: upload media source",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()

	if err := c.mediaService.UploadWaitingMediaSource(ctx); err != nil {
		slog.Error(
			"Upload waiting source",
			slog.Any("err", xerrors.New(err)),
		)
	}

	c.isUploadingWaitingMedia = false

	return nil
}

func (c *Cron) calculateUsersUsage(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: calculate usage",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.usageService.CalculateUsage(ctx); err != nil {
		slog.Error(
			"Calculate usage error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) deleteExpiredEmailType(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: delete expired email",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.mailService.DeleteExpiredEmailType(ctx); err != nil {
		slog.Error(
			"Delete expired login mail type error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) deleteExpiredApiKey(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: delete expired api key",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.apiKeyService.DeleteExpiredUserApiKey(ctx); err != nil {
		slog.Error(
			"Delete expired api key error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) handleUsage(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: handle usage",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.usageService.HandleUsage(ctx); err != nil {
		slog.Error("Handle usage error", slog.Any("err", xerrors.New(err)))
	}
}

func (c *Cron) handleFailUsage(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: handle fail usage",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.usageService.HandleFailUsage(ctx); err != nil {
		slog.Error("Handle fail usage error", slog.Any("err", xerrors.New(err)))
	}
}

func (c *Cron) handleLowWalletBalance(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: handle low wallet",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.usageService.HandleLowBalanceUsers(ctx); err != nil {
		slog.Error(
			"Handle low wallet balance error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) handleContentOutOfBalanceUser(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: handle out of balance user's content",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()

	users, err := c.usageService.HandleContentOutOfBalanceUser(ctx)
	if err != nil {
		slog.Error(
			"Handle out of balance users error",
			slog.Any("err", xerrors.New(err)),
		)
	}

	for _, user := range users {
		if err := c.mediaService.DeleteUserMedias(ctx, user); err != nil {
			slog.Error(
				"Delete user media error",
				slog.Any("err", xerrors.New(err)),
			)
		}
	}
}

func (c *Cron) handleMonthlyReceipt(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: handle monthly receipt",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.usageService.HandleMonthlyReceipt(ctx); err != nil {
		slog.Error(
			"Handle monthly receipt error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) handleNotSaveLiveStreamMedia(ctx context.Context) {
	liveStreamMedias, err := c.liveStreamMediaService.GetNotSavedLiveStreamMedias(
		ctx,
	)
	if err != nil {
		slog.Error(
			"Get not saved live stream media error",
			slog.Any("err", slog.Any("err", xerrors.New(err))),
		)
	}

	if len(liveStreamMedias) == 0 {
		return
	}

	for _, liveStreamMedia := range liveStreamMedias {
		if err := c.liveStreamMediaService.UpdateMediaStatusToDeletedIfNeeded(
			ctx,
			models.DeletedStatus,
			liveStreamMedia.MediaId,
		); err != nil {
			slog.Error(
				"Update media status to deleted error",
				slog.Any("err", xerrors.New(err)),
			)
		}

		if err := c.liveStreamMediaService.UpdateLiveStreamMediaStatus(
			ctx,
			"",
			liveStreamMedia,
			models.DeletedStatus,
		); err != nil {
			slog.Error(
				"Update live stream media error",
				slog.Any("err", xerrors.New(err)),
			)
		}

	}
}

func (c *Cron) handleLiveStreamMediaNotStream(ctx context.Context) {
	liveStreamMediasStreamings, err := c.liveStreamMediaService.HandleLiveStreamMediaNotStream(
		ctx,
	)
	if err != nil {
		slog.Error(
			"Get not stream live stream media error",
			slog.Any("err", xerrors.New(err)),
		)
	}

	if len(liveStreamMediasStreamings) == 0 {
		return
	}

	for _, liveStreamMediaStreaming := range liveStreamMediasStreamings {
		if err := c.liveStreamMediaService.UpdateLiveStreamMediaStatus(
			ctx,
			"",
			liveStreamMediaStreaming,
			models.DeletedStatus,
		); err != nil {
			slog.Error(
				"Update live stream media error",
				slog.Any("err", xerrors.New(err)),
			)
		}

		if err := c.liveStreamMediaService.UpdateMediaStatusToDeletedIfNeeded(
			ctx,
			models.DeletedStatus,
			liveStreamMediaStreaming.MediaId,
		); err != nil {
			slog.Error(
				"Update media status to deleted error",
				slog.Any("err", xerrors.New(err)),
			)
		}
	}
}

func (c *Cron) handleLiveStreamView(ctx context.Context) {
	if err := c.liveStreamMediaService.UpdateLiveStreamView(ctx); err != nil {
		slog.Error(
			"Update live stream view error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) handleLiveStreamDuration(ctx context.Context) {
	if err := c.liveStreamMediaService.HandleLiveStreamDuration(ctx); err != nil {
		slog.Error(
			"Handle live stream limits error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}

func (c *Cron) handleWebhookRetry(ctx context.Context) {
	start := time.Now()
	defer func() {
		logger.Debug(
			"Run cron: handle webhook retry",
			slog.Any("runtime", time.Since(start).Seconds()),
		)
	}()
	if err := c.webhookService.HandleWebhookRetry(ctx); err != nil {
		slog.Error(
			"Handle webhook retry error",
			slog.Any("err", xerrors.New(err)),
		)
	}
}
