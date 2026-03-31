package config

import (
	"context"
	"fmt"

	// "time"

	"github.com/redis/go-redis/v9"
)

func MustConnectRedis(config *AppConfig) (*redis.Client, *redis.Client) {
	if config == nil {
		panic("config is nil")
	}

	connIdDb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
		Password: config.RedisPassword,
		DB:       config.RedisConnIdDB,
	})

	if err := connIdDb.Ping(context.Background()).Err(); err != nil {
		panic("failed to connect redis")
	}

	uuidDb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", config.RedisHost, config.RedisPort),
		Password: config.RedisPassword,
		DB:       config.RedisUuidDB,
	})

	if err := uuidDb.Ping(context.Background()).Err(); err != nil {
		panic("failed to connect redis")
	}

	return connIdDb, uuidDb
}
