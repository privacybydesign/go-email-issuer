package storage

import (
	"backend/internal/config"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
	ctx := context.Background()
	addr := fmt.Sprintf("%v:%v", cfg.Redis.Host, cfg.Redis.Port)
	options := &redis.Options{
		Addr:     addr,
		Password: cfg.Redis.Password,
	}
	client := redis.NewClient(options)
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return client, err
}

func NewRedisSentinelClient(cfg *config.Config) (*redis.Client, error) {
	ctx := context.Background()

	addr := fmt.Sprintf("%v:%v", cfg.RedisSentinel.SentinelHost, cfg.RedisSentinel.SentinelPort)
	sentinelOptions := &redis.FailoverOptions{
		MasterName:       cfg.RedisSentinel.MasterName,
		SentinelAddrs:    []string{addr},
		Username:         cfg.RedisSentinel.SentinelUsername,
		Password:         cfg.RedisSentinel.Password,
		SentinelUsername: cfg.RedisSentinel.SentinelUsername,
		SentinelPassword: cfg.RedisSentinel.Password,
	}

	client := redis.NewFailoverClient(sentinelOptions)
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis through Sentinel: %w", err)
	}

	return client, err
}
