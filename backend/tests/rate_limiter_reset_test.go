package main

import (
	"backend/internal/core"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// exhaust calls Allow until it is rejected, failing the test if it never is.
func exhaust(t *testing.T, rl core.RateLimiter, key string) {
	t.Helper()
	for i := 0; i < 100; i++ {
		allow, _, err := rl.Allow(key)
		require.NoError(t, err)
		if !allow {
			return
		}
	}
	t.Fatalf("limiter for %q never blocked", key)
}

func TestInMemoryRateLimiterReset(t *testing.T) {
	policy := core.RateLimitingPolicy{Limit: 5, Window: 30 * time.Minute}
	rl := core.NewInMemoryRateLimiter(&mockClock{time: time.Now()}, policy)

	key := "email:locked@example.com"
	exhaust(t, rl, key)

	allow, _, err := rl.Allow(key)
	require.NoError(t, err)
	require.False(t, allow, "expected key to be blocked before reset")

	require.NoError(t, rl.Reset(key))

	allow, _, err = rl.Allow(key)
	require.NoError(t, err)
	require.True(t, allow, "expected key to be allowed after reset")
}

func TestInMemoryRateLimiterResetUnknownKeyIsNoOp(t *testing.T) {
	policy := core.RateLimitingPolicy{Limit: 5, Window: 30 * time.Minute}
	rl := core.NewInMemoryRateLimiter(&mockClock{time: time.Now()}, policy)
	require.NoError(t, rl.Reset("email:never-seen@example.com"))
}

func TestTotalRateLimiterResetEmail(t *testing.T) {
	clock := &mockClock{time: time.Now()}
	rl := newTestRateLimiter(clock)

	ip := "203.0.113.7"
	email := "locked@example.com"

	// Exhaust the per-email limit (10) from many different IPs so the IP limit
	// (5) is not what blocks us.
	for i := 0; i < 11; i++ {
		rl.Allow("198.51.100."+string(rune('0'+i%10)), email)
	}

	allow, _ := rl.Allow(ip, email)
	require.False(t, allow, "expected email to be rate-limited before reset")

	require.NoError(t, rl.ResetEmail(email))

	allow, _ = rl.Allow(ip, email)
	require.True(t, allow, "expected email to be allowed after reset")
}

func newRedisTestLimiter(t *testing.T, namespace string, policy core.RateLimitingPolicy) *core.RedisRateLimiter {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return core.NewRedisRateLimiter(client, namespace, policy)
}

func TestRedisRateLimiterReset(t *testing.T) {
	policy := core.RateLimitingPolicy{Limit: 5, Window: 30 * time.Minute}
	rl := newRedisTestLimiter(t, "yivi", policy)

	key := "email:locked@example.com"
	exhaust(t, rl, key)

	allow, _, err := rl.Allow(key)
	require.NoError(t, err)
	require.False(t, allow, "expected key to be blocked before reset")

	require.NoError(t, rl.Reset(key))

	allow, _, err = rl.Allow(key)
	require.NoError(t, err)
	require.True(t, allow, "expected key to be allowed after reset")
}

func TestTotalRateLimiterResetEmailRedis(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	emailPolicy := core.RateLimitingPolicy{Limit: 5, Window: 30 * time.Minute}
	ipPolicy := core.RateLimitingPolicy{Limit: 100, Window: 30 * time.Minute}
	email := core.NewRedisRateLimiter(client, "yivi", emailPolicy)
	ip := core.NewRedisRateLimiter(client, "yivi", ipPolicy)
	rl := core.NewTotalRateLimiter(email, ip)

	addr := "locked@example.com"
	for i := 0; i < 6; i++ {
		rl.Allow("203.0.113.7", addr)
	}

	allow, _ := rl.Allow("203.0.113.7", addr)
	require.False(t, allow, "expected email to be rate-limited before reset")

	require.NoError(t, rl.ResetEmail(addr))

	allow, _ = rl.Allow("203.0.113.7", addr)
	require.True(t, allow, "expected email to be allowed after reset")
}
