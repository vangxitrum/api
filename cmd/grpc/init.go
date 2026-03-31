package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/config"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	custom_log "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/log"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payment"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/repositories"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
	"10.0.0.50/tuan.quang.tran/vms-v2/rabbitmq"
)

var (
	db                *gorm.DB
	appConfig         *config.AppConfig
	storageHelper     storage.StorageHelper
	userStatusMapping map[uuid.UUID]string
	paymentClient     *payment.PaymentClient
	logClient         custom_log.LogClient

	mediaRepo           models.MediaRepository
	mediaCaptionRepo    models.MediaCaptionRepository
	streamRepo          models.StreamRepository
	formatRepo          models.FormatRepository
	partRepo            models.PartRepository
	cdnFileRepo         models.CdnFileRepository
	qualityRepo         models.QualityRepository
	watermarkRepo       models.WatermarkRepository
	usageRepo           models.UsageRepository
	mediaUsageRepo      models.MediaUsageRepository
	mediaChapterRepo    models.MediaChapterRepository
	userRepo            models.UserRepository
	playerThemeRepo     models.PlayerThemeRepository
	thumbnailRepo       models.ThumbnailRepository
	liveStreamMediaRepo models.LiveStreamMediaRepository

	mediaService *services.MediaService
)

func init() {
	env := "debug"
	if os.Getenv("APP_ENV") != "" {
		env = os.Getenv("APP_ENV")
	}

	appConfig = config.MustNewAppConfig(
		fmt.Sprintf(
			"./%s.env",
			env,
		),
	)

	db = config.MustConnectPostgres(appConfig)
	if appConfig.GraylogHost != "" && appConfig.GraylogPort != 0 {
		logClient = custom_log.NewGraylogLogger(
			"aioz-stream",
			appConfig.GraylogServiceName,
			fmt.Sprintf(
				"http://%s:%d/gelf",
				appConfig.GraylogHost,
				appConfig.GraylogPort,
			),
		)
	}

	slog.SetDefault(slog.New(
		custom_log.NewHandler(
			&slog.HandlerOptions{},
			custom_log.WithLogClient(logClient),
		),
	).With("group", "grpc"))

	storageHelper = storage.MustNewCdnHelper(
		appConfig.CdnUrl,
		appConfig.HubUrl,
		appConfig.BusinessAddress,
	)

	rabbitmqUrl := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/",
		appConfig.RabbitMQUser,
		appConfig.RabbitMQPassword,
		appConfig.RabbitMQHost,
		appConfig.RabbitMQPort,
	)
	rabbitmqOption := []rabbitmq.Option{
		rabbitmq.WithRecoverData(appConfig.RecoverPath),
		rabbitmq.WithReconnect(),
	}

	cdnHandlerCh, err := rabbitmq.NewRabbitMQ(
		rabbitmqUrl,
		"cdnHandlerCh",
		rabbitmqOption...)
	if err != nil {
		panic(err)
	}

	qualityCh, err := rabbitmq.NewRabbitMQ(
		rabbitmqUrl,
		"qualityCh",
		rabbitmqOption...)
	if err != nil {
		panic(err)
	}

	responseCh, err := rabbitmq.NewRabbitMQ(
		rabbitmqUrl,
		"responseCh",
		rabbitmqOption...)
	if err != nil {
		panic(err)
	}

	callWebhookChannel, err := rabbitmq.NewRabbitMQ(
		rabbitmqUrl,
		"callWebhook",
		rabbitmqOption...,
	)
	if err != nil {
		panic(err)
	}

	init := true
	mediaRepo = repositories.MustNewMediaRepository(
		db,
		init,
	)

	mediaCaptionRepo = repositories.MustNewMediaCaptionRepository(
		db,
		init,
	)

	cdnFileRepo = repositories.MustNewCdnFileRepository(
		db,
		init,
	)

	liveStreamMediaRepo = repositories.NewLiveStreamRepository(
		db,
		init,
	)

	mediaService = services.NewMediaService(
		db,
		mediaRepo,
		streamRepo,
		formatRepo,
		partRepo,
		cdnFileRepo,
		qualityRepo,
		watermarkRepo,
		userRepo,
		mediaUsageRepo,
		usageRepo,
		mediaCaptionRepo,
		mediaChapterRepo,
		playerThemeRepo,
		thumbnailRepo,
		cdnFileRepo,
		liveStreamMediaRepo,

		appConfig.BeUrl,
		appConfig.InputStoragePath,
		appConfig.OutputStoragePath,
		appConfig.RegisterId,

		userStatusMapping,
		storageHelper,
		paymentClient,
		nil,
		nil,
		nil,

		appConfig.UserConcurrentUploadingLimit,

		cdnHandlerCh,
		qualityCh,
		responseCh,

		callWebhookChannel,
	)
}
