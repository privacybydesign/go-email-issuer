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

	// Block once the window count exceeds the limit. This matches the
	// in-memory backend (entry.Count > Limit): a policy of Limit=N allows N
	// requests per window and blocks the (N+1)-th. Using > rather than >=
	// keeps both backends in agreement when storage_type is switched.
	if count > int64(r.policy.Limit) {
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

	now := r.clock.GetTime()
	entry, exists := r.memory[key]

	// Start a fresh window if the key is new or its window has already elapsed.
	// Doing the reset here (rather than lazily, only once the limit was already
	// exceeded) makes the current request count toward the new window and keeps
	// the behaviour independent of whether the janitor has run yet.
	if !exists || !entry.Expiry.After(now) {
		entry = &RateLimiterEntry{
			Count:  0,
			Expiry: now.Add(r.policy.Window),
		}
		r.memory[key] = entry
	}

	entry.Count += 1

	// Block the (Limit+1)-th request within the window. This matches the Redis
	// backend (count > Limit), so a policy of Limit=N allows N requests per
	// window on both backends.
	if entry.Count > r.policy.Limit {
		return false, entry.Expiry.Sub(now), nil
	}
	return true, 0, nil

}

// Cleanup evicts every entry whose window has already elapsed. Without it the
// memory map grows unbounded, since each distinct key (ip:<addr> / email:<addr>)
// adds an entry that is otherwise only ever overwritten, never removed. It is
// safe for concurrent use; the janitor goroutine calls it periodically and tests
// can call it directly to force deterministic eviction.
func (r *InMemoryRateLimiter) Cleanup() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := r.clock.GetTime()
	for key, entry := range r.memory {
		if !entry.Expiry.After(now) {
			delete(r.memory, key)
		}
	}
}

// Len reports the number of tracked keys. Useful for observing memory growth and
// eviction (primarily in tests).
func (r *InMemoryRateLimiter) Len() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return len(r.memory)
}

// StartJanitor launches a background goroutine that evicts expired entries every
// interval, bounding memory use under churn. It returns a stop function that
// terminates the goroutine. Production code starts it once at construction;
// tests that need deterministic behaviour drive Cleanup directly instead.
func (r *InMemoryRateLimiter) StartJanitor(interval time.Duration) (stop func()) {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				r.Cleanup()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
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
