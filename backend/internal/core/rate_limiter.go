package core

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ----------- Abstract Rate Limiter Interface -----------
type RateLimiter interface {
	Allow(key string) (allow bool, timeout time.Duration, err error)
}

type RateLimitingPolicy struct {
	Limit  int
	Window time.Duration
}

type RateLimiterEntry struct {
	Count  int
	Expiry time.Time
}

type Clock interface {
	GetTime() time.Time
}

type TotalRateLimiter struct {
	Email RateLimiter
	IP    RateLimiter
}

func NewTotalRateLimiter(email, ip RateLimiter) *TotalRateLimiter {
	return &TotalRateLimiter{Email: email, IP: ip}
}

func (l *TotalRateLimiter) Allow(ip, email string) (allow bool, timeoutRemaining time.Duration) {
	ipKey := fmt.Sprintf("ip:%s", ip)
	emailKey := fmt.Sprintf("email:%s", email)

	allowEmail, timeRemainingEmail, err := l.Email.Allow(emailKey)
	if err != nil {
		return false, 30 * time.Minute
	}

	allowIp, timeRemainingIp, err := l.IP.Allow(ipKey)
	if err != nil {
		return false, 30 * time.Minute
	}

	if !allowIp || !allowEmail {
		return false, maxDuration(timeRemainingIp, timeRemainingEmail)
	}
	return true, 0
}

// Redis rate limiter

type RedisRateLimiter struct {
	rclient   *redis.Client
	namespace string
	ctx       context.Context
	policy    RateLimitingPolicy
}

func NewRedisRateLimiter(redis *redis.Client, namespace string, policy RateLimitingPolicy) *RedisRateLimiter {
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
		log.Printf("Redis Incr failed: %v\n", err)
		return false, 0, err
	}

	if count == 1 {
		// First request: set expiry
		err = r.rclient.Expire(r.ctx, key, r.policy.Window).Err()
		if err != nil {
			log.Printf("Redis Expire failed: %v\n", err)
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

// Memory rate limiter

type InMemoryRateLimiter struct {
	memory map[string]*RateLimiterEntry
	mutex  sync.Mutex
	policy RateLimitingPolicy
	clock  Clock
}

func (r *InMemoryRateLimiter) Allow(key string) (allow bool, timeout time.Duration, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	entry, exists := r.memory[key]

	if !exists {
		entry = &RateLimiterEntry{
			Count:  0,
			Expiry: r.clock.GetTime().Add(r.policy.Window),
		}
		r.memory[key] = entry
	}

	entry.Count += 1

	if entry.Count > r.policy.Limit {
		timeUntilExpiry := entry.Expiry.Sub(r.clock.GetTime())

		if timeUntilExpiry < 0 {
			entry.Expiry = r.clock.GetTime().Add(r.policy.Window)
			entry.Count = 0
			return true, 0, nil
		}
		return false, timeUntilExpiry, nil
	}
	return true, 0, nil

}

func NewInMemoryRateLimiter(clock Clock, policy RateLimitingPolicy) *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		memory: map[string]*RateLimiterEntry{},
		mutex:  sync.Mutex{},
		policy: policy,
		clock:  clock,
	}
}

// --------------- HELPER FUNCTIONS -------------------

type SystemClock struct{}

func NewSystemClock() *SystemClock        { return &SystemClock{} }
func (c *SystemClock) GetTime() time.Time { return time.Now() }

func maxDuration(a, b time.Duration) time.Duration {
	if a >= b {
		return a
	}
	return b
}
