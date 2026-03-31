package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	payment_gateway "github.com/AIOZNetwork/payment/payment-gateway"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/config"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/controllers"
	crons "10.0.0.50/tuan.quang.tran/vms-v2/internal/cron"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/middlewares"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/routes"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/image"
	ip_helper "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/ip"
	client "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/job_client"
	custom_log "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/log"
	mails "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/mail"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/message"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/payment"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/response"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/storage"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/stream"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/token"
	"10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/transcribe_client"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/repositories"
	"10.0.0.50/tuan.quang.tran/vms-v2/pkg/v1/services"
	"10.0.0.50/tuan.quang.tran/vms-v2/rabbitmq"
)

// TODO: warn low on balance, delete long-uploaded media, prevent user
// upload if balance is low

var (
	appConfig      *config.AppConfig
	server         *echo.Echo
	paymentGateway *payment_gateway.SubscribeService
	cron           *crons.Cron
	debugLogger    *slog.Logger

	db            *gorm.DB
	redisUuidDb   *redis.Client
	redisConnIdDb *redis.Client

	userStatusMapping map[uuid.UUID]string
	mailHelper        mails.MailHelper
	messageHelper     message.MessageHelper
	tokenIssuer       *token.TokenIssuer
	storageHelper     storage.StorageHelper
	deserializeUser   echo.MiddlewareFunc
	authWithApiKey    echo.MiddlewareFunc
	ipHelper          ip_helper.IpHelper
	logClient         custom_log.LogClient
	paymentClient     *payment.PaymentClient
	jobClient         *client.JobClient
	transcribeClient  *transcribe_client.TranscribeClient
	thumbnailHelper   *image.ThumbnailHelper

	playerThemeRepo         models.PlayerThemeRepository
	mediaRepo               models.MediaRepository
	streamRepo              models.StreamRepository
	formatRepo              models.FormatRepository
	userRepo                models.UserRepository
	apiKeyRepo              models.ApiKeyRepository
	webhookRepo             models.WebhookRepository
	emailConnectionRepo     models.EmailConnectionRepository
	walletConnectionRepo    models.WalletConnectionRepository
	partRepo                models.PartRepository
	cdnFileRepo             models.CdnFileRepository
	cdnUsageRepo            models.CdnUsageStatisticRepository
	qualityRepo             models.QualityRepository
	watermarkRepo           models.WatermarkRepository
	mailRepo                models.MailRepository
	usageRepo               models.UsageRepository
	mediaUsageRepo          models.MediaUsageRepository
	mediaCaptionRepo        models.MediaCaptionRepository
	mediaChapterRepo        models.MediaChapterRepository
	paymentRepo             models.PaymentRepository
	liveStreamKeyRepo       models.LiveStreamKeyRepository
	liveStreamMulticastRepo models.LiveStreamMulticastRepository
	liveStreamStatisticRepo models.LiveStreamStatisticRepository
	liveStreamMediaRepo     models.LiveStreamMediaRepository
	statisticRepo           models.StatisticRepository
	ipInfoRepo              models.IpInfoRepository
	thumbnailRepo           models.ThumbnailRepository
	reportContentRepo       models.ReportContentRepository
	playlistRepo            models.PlaylistRepository
	exclusiveCodeRepo       models.ExclusiveCodeRepository

	playerThemeService         *services.PlayerThemeService
	mediaService               *services.MediaService
	userService                *services.UserService
	apiKeyService              *services.ApiKeyService
	webhookService             *services.WebhookService
	watermarkService           *services.WatermarkService
	usageService               *services.UsageService
	cdnUsageService            *services.CdnUsageService
	mailService                *services.MailService
	mediaCaptionService        *services.MediaCaptionService
	mediaChapterService        *services.MediaChapterService
	liveStreamService          *services.LiveStreamService
	liveStreamMediaService     *services.LiveStreamMediaService
	liveStreamMulticastService *services.LiveStreamMulticastService
	liveStreamStatisticService *services.LiveStreamStatisticService
	statisticService           *services.StatisticService
	playlistService            *services.PlaylistService
	reportContentService       *services.ReportContentService

	playerThemeController   *controllers.PlayerThemesController
	authController          *controllers.AuthController
	mediaController         *controllers.MediaController
	userController          *controllers.UserController
	apiKeyController        *controllers.ApiKeyController
	webhookController       *controllers.WebhookController
	watermarkController     *controllers.WatermarkController
	mediaCaptionController  *controllers.MediaCaptionController
	mediaChapterController  *controllers.MediaChapterController
	usageController         *controllers.UsageController
	liveStreamController    *controllers.LiveStreamController
	statisticController     *controllers.StatisticController
	playlistController      *controllers.PlaylistController
	reportContentController *controllers.ReportContentController
	traceController         *controllers.TraceController

	playerThemeRoute   *routes.PlayerThemesRoute
	authRoute          *routes.AuthRoute
	mediaRoute         *routes.MediaRoute
	userRoute          *routes.UserRoute
	apiKeyRoute        *routes.ApiKeyRoute
	webhookRoute       *routes.WebhookRoute
	watermarkRoute     *routes.WatermarkRoute
	usageRoute         *routes.UsageRoute
	liveStreamRoute    *routes.LiveStreamRoute
	playlistRoute      *routes.PlaylistRoute
	reportContentRoute *routes.ReportContentRoute
	traceRoute         *routes.TraceRoute
)

func init() {
	env := "debug"
	if os.Getenv("APP_ENV") != "" {
		env = os.Getenv("APP_ENV")
	}

	// config
	appConfig = config.MustNewAppConfig(
		fmt.Sprintf(
			"./%s.env",
			env,
		),
	)

	db = config.MustConnectPostgres(appConfig)
	redisConnIdDb, redisUuidDb = config.MustConnectRedis(appConfig)
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
	))

	debugLogger = slog.New(
		custom_log.NewHandler(&slog.HandlerOptions{
			Level: slog.LevelDebug,
		}, custom_log.WithLogClient(logClient)),
	)

	models.InitLiveStreamMedia(appConfig.LiveServerUrl)
	models.InitLiveStreamKey(appConfig.RtmpUrl)
	var err error
	paymentGateway, err = payment_gateway.NewSubscribeService(
		context.Background(),
		db,
		appConfig.BusinessAddress,
		appConfig.RpcMainnet,
		appConfig.EvmEthMainnet,
		appConfig.PassPhrase,
		appConfig.OAuthToken,
		appConfig.ChannelID,
		appConfig.CdnUrl,
		appConfig.BurnAddress,
		"",
		5,
	)
	if err != nil {
		log.Fatal("🚀 Could not init payment gateway ", err)
		return
	}

	// repositories
	init := true
	streamRepo = repositories.MustNewStreamRepository(
		db,
		init,
	)
	formatRepo = repositories.MustNewFormatRepository(
		db,
		init,
	)
	apiKeyRepo = repositories.MustNewApiKeyRepository(
		db,
		init,
	)
	webhookRepo = repositories.MustNewWebhookRepository(
		db,
		init,
	)

	mediaRepo = repositories.MustNewMediaRepository(
		db,
		init,
	)
	userRepo = repositories.MustNewUserRepository(
		db,
		init,
	)
	emailConnectionRepo = repositories.MustNewEmailConnectionRepository(
		db,
		init,
	)
	walletConnectionRepo = repositories.MustNewWalletConnectionRepository(
		db,
		init,
	)
	playerThemeRepo = repositories.MustNewPlayerThemeRepository(
		db,
		init,
	)
	mediaRepo = repositories.MustNewMediaRepository(
		db,
		init,
	)
	userRepo = repositories.MustNewUserRepository(
		db,
		init,
	)
	emailConnectionRepo = repositories.MustNewEmailConnectionRepository(
		db,
		init,
	)
	walletConnectionRepo = repositories.MustNewWalletConnectionRepository(
		db,
		init,
	)
	partRepo = repositories.MustNewPartRepository(
		db,
		init,
	)
	cdnFileRepo = repositories.MustNewCdnFileRepository(
		db,
		init,
	)
	cdnUsageRepo = repositories.MustNewCdnUsageRepository(
		db,
		init,
	)
	qualityRepo = repositories.MustNewQualityRepository(
		db,
		init,
	)
	watermarkRepo = repositories.MustWatermarkRepository(
		db,
		init,
	)
	mailRepo = repositories.MustNewMailRepository(
		db,
		init,
	)
	usageRepo = repositories.MustNewUsageRepository(
		db,
		init,
	)
	mediaUsageRepo = repositories.MustNewMediaUsageRepository(
		db,
		init,
	)
	mediaCaptionRepo = repositories.MustNewMediaCaptionRepository(
		db,
		init,
	)
	mediaChapterRepo = repositories.MustNewMediaChapterRepository(
		db,
		init,
	)
	paymentRepo = repositories.MustNewPaymentRepository(db, init)
	statisticRepo = repositories.MustNewStatisticRepository(
		db,
		init,
	)
	ipInfoRepo = repositories.MustNewIpInfoRepository(
		db,
		init,
	)
	thumbnailRepo = repositories.MustNewThumbnailRepository(
		db,
		init,
	)
	playlistRepo = repositories.MustNewPlaylistRepository(
		db,
		init,
	)
	exclusiveCodeRepo = repositories.MustNewExclusiveCodeRepository(
		db,
		init,
	)

	liveStreamMediaRepo = repositories.NewLiveStreamRepository(db, init)
	liveStreamKeyRepo = repositories.NewLiveStreamKeyRepository(db, init)
	reportContentRepo = repositories.MustNewReportContentRepository(db, init)
	liveStreamMulticastRepo = repositories.NewLiveStreamMulticastRepository(
		db,
		init,
	)
	liveStreamStatisticRepo = repositories.NewLiveStreamStatisticRepository(
		db,
		init,
	)

	if init {
		func() {
			if _, err := os.Stat(appConfig.DbTriggerFilePath); os.IsNotExist(
				err,
			) {
				slog.Warn("db trigger file not found")
				return
			}

			data, err := os.ReadFile(appConfig.DbTriggerFilePath)
			if err != nil {
				slog.Error("failed to read db trigger file", "error", err)
				return
			}

			if err := db.Exec(string(data)).Error; err != nil {
				slog.Error("failed to execute db trigger file", "error", err)
				return
			} else {
				slog.Info("successfully executed db trigger file")
			}
		}()
	}

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
	callWebhookChannel, err := rabbitmq.NewRabbitMQ(
		rabbitmqUrl,
		"callWebhook",
		rabbitmqOption...,
	)
	if err != nil {
		panic(err)
	}

	endLiveStreamChannel, err := rabbitmq.NewRabbitMQ(
		rabbitmqUrl,
		"endLiveStream",
		rabbitmqOption...,
	)
	if err != nil {
		panic(err)
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

	// core
	if err := os.MkdirAll(
		appConfig.InputStoragePath,
		0o755,
	); err != nil {
		panic("failed to create input storage directory")
	}

	if err := os.MkdirAll(
		appConfig.OutputStoragePath,
		0o755,
	); err != nil {
		panic("failed to create output storage directory")
	}

	userStatusMapping = make(map[uuid.UUID]string)
	mailHelper = mails.NewResendHelper(
		appConfig.ResendApiKey,
		appConfig.EmailFrom,
		appConfig.TemplatesDir,
	)
	messageHelper, err = message.NewSlackHelper(
		appConfig.SlackReportBotToken,
		appConfig.SlackReportChannelID,
	)
	if err != nil {
		panic("🚀 Could not init Message helper " + err.Error())
	}
	tokenIssuer = token.NewTokenIssuer(
		appConfig.AccessTokenPrivateKey,
		appConfig.AccessTokenPublicKey,
		appConfig.RefreshTokenPrivateKey,
		appConfig.RefreshTokenPublicKey,
		appConfig.AccessTokenMaxAge,
		appConfig.RefreshTokenMaxAge,
	)
	useCase := models.NewUseCase(
		userRepo,
		apiKeyRepo,
		webhookRepo,
	)
	deserializeUser = middlewares.DeserializeUser(
		tokenIssuer,
		userRepo,
	)
	authWithApiKey = middlewares.AuthenticateAPIKey(
		tokenIssuer,
		useCase,
	)

	storageHelper = storage.MustNewCdnHelper(
		appConfig.CdnUrl,
		appConfig.HubUrl,
		appConfig.BusinessAddress,
	)

	paymentClient = payment.NewPaymentClient(
		paymentGateway,
		userRepo,
		paymentRepo,
	)
	jobClient, err = client.NewJobClient(
		fmt.Sprintf("%s:%s", appConfig.JobServerHost, appConfig.JobServerPort),
	)
	if err != nil {
		panic(err)
	}

	transcribeClient = transcribe_client.NewTranscribeClient(
		appConfig.TranscribeBaseUrl,
		appConfig.TranscribeApiKey,
		appConfig.TranscribeModelId,
	)

	thumbnailHelper = image.NewThumbnailHelper(storageHelper)
	streamClient := stream.NewStreamClient(
		appConfig.StreamApiUrl,
		appConfig.StreamApiUser,
		appConfig.StreamApiPass,
	)

	ipHelper = ip_helper.NewIp2LocationHelper(
		appConfig.IP2LocationApiKey,
		ip_helper.WithDebug,
	)
	// liveStreamHandler := core.NewLiveStreamHandler("./public")
	response.NewTraceHelper("./trace_data")
	// services
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
		jobClient,
		transcribeClient,
		thumbnailHelper,

		appConfig.UserConcurrentUploadingLimit,

		cdnHandlerCh,
		qualityCh,
		responseCh,

		callWebhookChannel,
	)
	userService = services.NewUserService(
		userRepo,
		emailConnectionRepo,
		walletConnectionRepo,
		mailRepo,
		usageRepo,
		exclusiveCodeRepo,

		mailHelper,
		tokenIssuer,
		paymentClient,
		storageHelper,
	)
	apiKeyService = services.NewApiKeyService(apiKeyRepo)
	playerThemeService = services.NewPlayerThemeService(
		db,
		playerThemeRepo,
		cdnFileRepo,
		mediaRepo,
		usageRepo,
		storageHelper,
	)
	webhookService = services.NewWebhookService(
		webhookRepo,
		resty.New(),
		callWebhookChannel,
	)
	watermarkService = services.NewWatermarkService(
		watermarkRepo,
		cdnFileRepo,
		usageRepo,
		storageHelper,
	)
	usageService = services.NewUsageService(
		db,
		usageRepo,
		userRepo,
		cdnFileRepo,
		paymentRepo,
		mediaUsageRepo,

		appConfig.CostPerStorage,
		appConfig.CostPerDelivery,

		userStatusMapping,
		paymentClient,
		storageHelper,
		cdnUsageRepo,
		mailHelper,
		mailRepo,
	)

	playlistService = services.NewPlaylistService(
		db,
		playlistRepo,
		cdnFileRepo,
		usageRepo,
		storageHelper,
		thumbnailRepo,
		mediaRepo,
		playerThemeRepo,

		thumbnailHelper,
	)

	cdnUsageService = services.NewCdnUsageService(
		usageRepo,
		cdnFileRepo,
		cdnUsageRepo,
		storageHelper,
		mediaRepo,
	)

	mediaCaptionService = services.NewMediaCaptionService(
		mediaRepo,
		mediaCaptionRepo,
		cdnFileRepo,
		usageRepo,

		appConfig.BeUrl,
		appConfig.InputStoragePath,
		storageHelper,
		transcribeClient,
	)
	mediaChapterService = services.NewMediaChapterService(
		mediaRepo,
		mediaChapterRepo,
		cdnFileRepo,

		appConfig.BeUrl,
		storageHelper,
	)

	mailService = services.NewMailService(
		emailConnectionRepo,
		mailRepo,
	)

	liveStreamService = services.NewLiveStreamService(
		liveStreamKeyRepo,
		liveStreamMediaRepo,
		liveStreamMulticastRepo,
		mediaRepo,
		cdnFileRepo,
		storageHelper,
		endLiveStreamChannel,
		appConfig.CdnUrl,
		streamClient,
	)

	liveStreamMediaService = services.NewLiveStreamMediaService(
		liveStreamMediaRepo,
		mediaRepo,
		usageRepo,

		paymentClient,
		appConfig.InputStoragePath,
		appConfig.OutputStoragePath,
		streamClient,
		liveStreamKeyRepo,
		redisUuidDb,
		redisConnIdDb,
	)

	liveStreamMulticastService = services.NewLiveStreamMulticastService(
		liveStreamMulticastRepo,
		liveStreamKeyRepo,
	)
	liveStreamStatisticService = services.NewLiveStreamStatisticService(
		liveStreamStatisticRepo,
	)

	statisticService = services.NewStatisticService(
		statisticRepo,
		mediaRepo,
		ipInfoRepo,
		liveStreamMediaRepo,
		cdnUsageRepo,

		ipHelper,

		appConfig.AdminStatisticApiKey,
	)
	reportContentService = services.NewReportContentService(
		reportContentRepo,
		mediaRepo,
		liveStreamMediaRepo,
		messageHelper,
	)
	// controllers
	authController = controllers.NewAuthController(userService)
	mediaController = controllers.NewMediaController(
		mediaService,
		usageService,
	)
	userController = controllers.NewUserController(userService)
	playerThemeController = controllers.NewPlayerThemesController(
		playerThemeService,
		usageService,
	)
	apiKeyController = controllers.NewApiKeyController(apiKeyService)
	webhookController = controllers.NewWebhookController(webhookService)
	watermarkController = controllers.NewWatermarkController(watermarkService)
	mediaCaptionController = controllers.NewMediaCaptionController(
		mediaCaptionService,
	)
	mediaChapterController = controllers.NewMediaChapterController(
		mediaChapterService,
	)
	usageController = controllers.NewUsageController(usageService)
	liveStreamController = controllers.NewLiveStreamController(
		liveStreamService,
		liveStreamMediaService,
		liveStreamMulticastService,
		liveStreamStatisticService,
		usageService,
		mediaService,
	)
	playlistController = controllers.NewPlaylistController(
		playlistService,
		mediaService,
		usageService,
	)
	statisticController = controllers.NewStatisticController(statisticService)
	reportContentController = controllers.NewReportContentController(
		reportContentService,
	)
	playlistController = controllers.NewPlaylistController(
		playlistService,
		mediaService,
		usageService,
	)
	traceController = controllers.NewTraceController()

	// routes
	authRoute = routes.NewAuthRoute(authController)
	mediaRoute = routes.NewMediaRoute(
		mediaController,
		mediaCaptionController,
		mediaChapterController,

		authWithApiKey,
	)
	userRoute = routes.NewUserRoute(
		userController,
		deserializeUser,
		authWithApiKey,
	)
	apiKeyRoute = routes.NewApiKeyRoute(
		apiKeyController,
		authWithApiKey,
	)

	webhookRoute = routes.NewWebhookRoute(
		webhookController,
		authWithApiKey,
	)
	playerThemeRoute = routes.NewPlayerThemesRoute(
		playerThemeController,
		authWithApiKey,
	)
	watermarkRoute = routes.NewWatermarkRoute(
		watermarkController,
		authWithApiKey,
	)
	usageRoute = routes.NewUsageRoute(
		usageController,
		statisticController,
		deserializeUser,
		authWithApiKey,
	)
	liveStreamRoute = routes.NewLiveStreamRoute(
		liveStreamController,
		authWithApiKey,
	)
	playlistRoute = routes.NewPlaylistRoute(
		playlistController,
		authWithApiKey,
	)
	reportContentRoute = routes.NewReportContentRoute(
		reportContentController,
		authWithApiKey,
	)
	traceRoute = routes.NewTraceRoute(traceController)

	cron = crons.NewCron(
		usageService,
		mailService,
		apiKeyService,
		mediaService,
		userService,
		cdnUsageService,
		statisticService,
		liveStreamMediaService,
		liveStreamService,
		webhookService,
		mediaCaptionService,
		playerThemeService,
		playlistService,
	)

	server = echo.New()
	configLogger(server)
}

func configLogger(server *echo.Echo) {
	ExcludedPaths := map[string]bool{
		"/metrics": true,
	}

	server.Use(
		middleware.RequestLoggerWithConfig(
			middleware.RequestLoggerConfig{
				LogStatus:   true,
				LogURI:      true,
				LogURIPath:  true,
				LogError:    true,
				LogMethod:   true,
				LogLatency:  true,
				HandleError: true,
				LogValuesFunc: func(
					c echo.Context,
					v middleware.RequestLoggerValues,
				) error {
					if ExcludedPaths[c.Path()] {
						return nil
					}

					d := v.Latency
					var latencyString string
					switch {
					case d >= time.Second:
						latencyString = fmt.Sprintf(
							"%03ds",
							int64(d.Seconds()),
						)
					case d >= time.Millisecond:
						latencyString = fmt.Sprintf(
							"%03dms",
							int64(d.Milliseconds()),
						)
					case d >= time.Microsecond:
						latencyString = fmt.Sprintf(
							"%03dµs",
							int64(d.Microseconds()),
						)
					default:
						latencyString = fmt.Sprintf(
							"%03dns",
							d.Nanoseconds(),
						)
					}

					var method string
					switch v.Method {
					case http.MethodGet:
						method = custom_log.GetColor
					case http.MethodPost:
						method = custom_log.PostColor
					case http.MethodPut:
						method = custom_log.PutColor
					case http.MethodPatch:
						method = custom_log.PatchColor
					case http.MethodDelete:
						method = custom_log.DeleteColor
					default:
						return nil
					}

					var status string
					switch {
					case v.Status >= 200 && v.Status < 400:
						status = fmt.Sprintf(custom_log.SuccessColor, v.Status)
					case v.Status >= 400 && v.Status < 500:
						status = fmt.Sprintf(custom_log.FailColor, v.Status)
					default:
						status = fmt.Sprintf(custom_log.ErrorColor, v.Status)
					}

					slog.Default().With("group", "api").InfoContext(
						c.Request().Context(),
						fmt.Sprintf(
							"%s status:%s latency:%s url:%s",
							method,
							status,
							latencyString,
							v.URIPath,
						),
					)

					return nil
				},
			},
		),
	)
}
