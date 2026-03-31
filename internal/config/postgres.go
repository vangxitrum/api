package config

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"golang.org/x/net/context"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	custom_log "10.0.0.50/tuan.quang.tran/vms-v2/internal/utils/log"
)

func MustConnectPostgres(config *AppConfig) *gorm.DB {
	if config == nil {
		panic("config is nil")
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		config.PostgresHost,
		config.PostgresUser,
		config.PostgresPassword,
		config.PostgresDBName,
		config.PostgresPort,
	)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             1000 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,  // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false, // Don't include params in the SQL log
			Colorful:                  true,  // Enable color
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		panic("failed to connect database")
	}

	sqlDB, err := db.DB()
	if err != nil {
		panic("failed to get sql.DB")
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)
	if err := sqlDB.Ping(); err != nil {
		panic("failed to ping database")
	}

	if err := db.Callback().Query().After("gorm:query").Register("captureQueryOnError", captureQueryOnError); err != nil {
		slog.Error("failed to register captureQueryOnError callback")
	}

	fmt.Println("Connected Successfully to the database.")

	return db
}

func captureQueryOnError(db *gorm.DB) {
	if db.Error != nil && !errors.Is(db.Error, gorm.ErrRecordNotFound) {
		query := db.Statement.SQL.String()
		ctx := db.Statement.Context
		attrs, ok := ctx.Value(custom_log.SlogFieldsKey).([]slog.Attr)
		if ok {
			attrs = append(attrs, slog.Any("query", query))
		} else {
			attrs = []slog.Attr{slog.Any("query", query)}
		}

		ctx = context.WithValue(ctx, custom_log.SlogFieldsKey, attrs)
		slog.ErrorContext(ctx, "db error info")
	}
}
