package storage

import (
	"backend/internal/core"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisRateLimiter struct {
	rclient   *redis.Client
	namespace string
	ctx       context.Context
	policy    core.RateLimitingPolicy
}

func NewRedisRateLimiter(redis *redis.Client, namespace string, policy core.RateLimitingPolicy) *RedisRateLimiter {
	return &RedisRateLimiter{
		rclient:   redis,
		ctx:       context.Background(),
		policy:    policy,
		namespace: namespace,
	}
}

func (r *RedisRateLimiter) Allow(key string) (bool, time.Duration, error) {

	key = fmt.Sprintf("%s:%s", r.namespace, key)

	count, err := r.rclient.Incr(r.ctx, key).Result()
	if err != nil {
		fmt.Printf("Redis Incr failed: %v\n", err)
		return false, 0, err
	}

	if count == 1 {
		// First request: set expiry
		err = r.rclient.Expire(r.ctx, key, r.policy.Window).Err()
		if err != nil {
			fmt.Printf("Redis Expire failed: %v\n", err)
			return false, 0, err
		}
	}

	if count >= int64(r.policy.Limit) {
		timeRemaining, err := r.rclient.TTL(r.ctx, key).Result()
		if err != nil {
			return false, 0, err
		}
		return false, timeRemaining, nil
	}

	return true, 0, nil
}
