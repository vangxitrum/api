package config

import (
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/viper"

	"10.0.0.50/tuan.quang.tran/vms-v2/internal/models"
)

type AppConfig struct {
	Port     string `mapstructure:"PORT"`
	GrpcPort string `mapstructure:"GRPC_PORT"`

	PostgresHost     string `mapstructure:"POSTGRES_HOST"`
	PostgresPort     string `mapstructure:"POSTGRES_PORT"`
	PostgresUser     string `mapstructure:"POSTGRES_USER"`
	PostgresPassword string `mapstructure:"POSTGRES_PASSWORD"`
	PostgresDBName   string `mapstructure:"POSTGRES_DB"`

	RedisHost     string `mapstructure:"REDIS_HOST"`
	RedisPort     string `mapstructure:"REDIS_PORT"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisConnIdDB int    `mapstructure:"REDIS_CONNID_DB"`
	RedisUuidDB   int    `mapstructure:"REDIS_UUID_DB"`

	RabbitMQPort     int    `mapstructure:"RABBITMQ_PORT"         required:"true"`
	RabbitMQUser     string `mapstructure:"RABBITMQ_DEFAULT_USER" required:"true"`
	RabbitMQPassword string `mapstructure:"RABBITMQ_DEFAULT_PASS" required:"true"`
	RabbitMQHost     string `mapstructure:"RABBITMQ_HOST"         required:"true"`

	EmailFrom    string `mapstructure:"EMAIL_FROM"`
	ResendApiKey string `mapstructure:"RESEND_API_KEY"`
	TemplatesDir string `mapstructure:"TEMPLATES_DIR"`

	AccessTokenPrivateKey  string        `mapstructure:"ACCESS_TOKEN_PRIVATE_KEY"  required:"true"`
	AccessTokenPublicKey   string        `mapstructure:"ACCESS_TOKEN_PUBLIC_KEY"   required:"true"`
	RefreshTokenPrivateKey string        `mapstructure:"REFRESH_TOKEN_PRIVATE_KEY" required:"true"`
	RefreshTokenPublicKey  string        `mapstructure:"REFRESH_TOKEN_PUBLIC_KEY"  required:"true"`
	AccessTokenExpiresIn   time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRED_IN"   required:"true"`
	RefreshTokenExpiresIn  time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRED_IN"  required:"true"`
	AccessTokenMaxAge      int           `mapstructure:"ACCESS_TOKEN_MAXAGE"       required:"true"`
	RefreshTokenMaxAge     int           `mapstructure:"REFRESH_TOKEN_MAXAGE"      required:"true"`

	InputStoragePath  string `mapstructure:"INPUT_STORAGE_PATH"`
	OutputStoragePath string `mapstructure:"OUTPUT_STORAGE_PATH"`
	RecoverPath       string `mapstructure:"RECOVER_PATH"`
	RegisterId        string `mapstructure:"REGISTER_ID"`

	CdnUrl string `mapstructure:"CDN_URL"`
	HubUrl string `mapstructure:"HUB_URL"`

	BeUrl     string `mapstructure:"BE_URL"`
	FeUrl     string `mapstructure:"FE_URL"`
	PlayerUrl string `mapstructure:"PLAYER_URL"`

	BusinessAddress string `mapstructure:"BUSINESS_ADDRESS" validate:"required"`
	PassPhrase      string `mapstructure:"PASS_PHRASE"      validate:"required"`
	RpcTestnet      string `mapstructure:"RPC_TESTNET"      validate:"required"`
	EvmEthTestnet   string `mapstructure:"EVM_RPC_TESTNET"  validate:"required"`
	RpcMainnet      string `mapstructure:"RPC_MAINNET"      validate:"required"`
	EvmEthMainnet   string `mapstructure:"EVM_RPC_MAINNET"  validate:"required"`
	BurnAddress     string `mapstructure:"BURN_ADDRESS"     validate:"required"`

	OAuthToken string `mapstructure:"OAUTH_TOKEN_BOT" validate:"required"`
	ChannelID  string `mapstructure:"CHANNEL_ID"      validate:"required"`

	CostPerStorage     int64 `mapstructure:"COST_PER_STORAGE"`
	CostPerDelivery    int64 `mapstructure:"COST_PER_DELIVERY"`
	HubCostPerStorage  int64 `mapstructure:"HUB_COST_PER_STORAGE"`
	HubCostPerDelivery int64 `mapstructure:"HUB_COST_PER_DELIVERY"`

	IP2LocationApiKey string `mapstructure:"IP_2_LOCATION_API_KEY"`

	BetterStackToken string `mapstructure:"BETTER_STACK_TOKEN"`

	SlackReportBotToken  string `mapstructure:"OAUTH_TOKEN_BOT_REPORT_CONTENT"`
	SlackReportChannelID string `mapstructure:"CHANNEL_REPORT_COTENT_ID"`

	RtmpUrl       string `mapstructure:"RTMP_URL"`
	LiveServerUrl string `mapstructure:"LIVE_SERVER_URL"`

	UserConcurrentUploadingLimit int `mapstructure:"USER_CONCURRENT_UPLOADING_LIMIT"`
	UploadRateLimit              int `mapstructure:"UPLOAD_RATE_LIMIT"`
	WritesRateLimit              int `mapstructure:"WRITES_RATE_LIMIT"`
	ReadsRateLimit               int `mapstructure:"READS_RATE_LIMIT"`

	// Stream service
	StreamApiUrl  string `mapstructure:"STREAM_API_URL"`
	StreamApiUser string `mapstructure:"STREAM_API_USER"`
	StreamApiPass string `mapstructure:"STREAM_API_PASS"`

	StreamWebhookToken string `mapstructure:"STREAM_WEBHOOK_TOKEN"`

	JobServerHost string `mapstructure:"JOB_SERVER_HOST"`
	JobServerPort string `mapstructure:"JOB_SERVER_PORT"`

	TranscribeBaseUrl string `mapstructure:"TRANSCRIBE_BASE_URL"`
	TranscribeApiKey  string `mapstructure:"TRANSCRIBE_API_KEY"`
	TranscribeModelId string `mapstructure:"TRANSCRIBE_MODEL_ID"`

	AdminStatisticApiKey string `mapstructure:"ADMIN_STATISTIC_API_KEY"`

	GraylogHost        string `mapstructure:"GRAYLOG_HOST"`
	GraylogPort        int    `mapstructure:"GRAYLOG_PORT"`
	GraylogServiceName string `mapstructure:"GRAYLOG_SERVICE_NAME"`

	DemoVideoId string `mapstructure:"DEMO_VIDEO_ID"`

	DbTriggerFilePath string `mapstructure:"DB_TRIGGER_FILE_PATH"`

	AdminMailList string `mapstructure:"ADMIN_MAIL_LIST"`
}

func MustNewAppConfig(filePath string) *AppConfig {
	var config AppConfig
	viper.SetConfigFile(filePath)
	viper.SetConfigType("env")
	if err := viper.ReadInConfig(); err != nil {
		panic("can not read config file")
	}

	if err := viper.Unmarshal(&config); err != nil {
		panic("can not unmarshal config file")
	}

	if config.BeUrl != "" {
		models.BeUrl = config.BeUrl
	}

	if config.FeUrl != "" {
		models.FeUrl = config.FeUrl
	}

	if config.PlayerUrl != "" {
		models.PlayerUrl = config.PlayerUrl
	}

	if config.UploadRateLimit != 0 {
		models.UploadRateLimit = config.UploadRateLimit
	}

	if config.WritesRateLimit != 0 {
		models.WritesRateLimit = config.WritesRateLimit
	}

	if config.ReadsRateLimit != 0 {
		models.ReadsRateLimit = config.ReadsRateLimit
	}

	if config.HubCostPerStorage != 0 {
		models.HubCostPerStorage = config.HubCostPerStorage
	}

	if config.HubCostPerDelivery != 0 {
		models.HubCostPerDelivery = config.HubCostPerDelivery
	}

	models.AdminMailList = strings.Split(config.AdminMailList, ",")
	if config.DemoVideoId != "" {
		var err error
		models.DemoVideoId, err = uuid.Parse(config.DemoVideoId)
		if err != nil {
			slog.Warn("invalid demo video id", "error", err)
		}
	}

	models.BetterStackToken = config.BetterStackToken

	return &config
}

func (c *AppConfig) LogValue() slog.Value {
	return slog.StringValue("dont log sensitive data")
}
