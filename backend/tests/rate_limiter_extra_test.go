package main

import (
	"backend/internal/core"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestInMemoryRateLimiterEviction verifies that expired entries are actually
// removed from the map, rather than lingering forever (the memory leak).
func TestInMemoryRateLimiterEviction(t *testing.T) {
	clock := &mockClock{time: time.Now()}
	policy := core.RateLimitingPolicy{Window: 30 * time.Minute, Limit: 5}
	rl := core.NewInMemoryRateLimiter(clock, policy)

	for i := range 100 {
		_, _, _ = rl.Allow(fmt.Sprintf("ip:%d", i))
	}
	if got := rl.Len(); got != 100 {
		t.Fatalf("expected 100 tracked keys, got %d", got)
	}

	// Before the window elapses, cleanup must keep every live entry.
	rl.Cleanup()
	if got := rl.Len(); got != 100 {
		t.Fatalf("expected entries to survive before the window elapses, got %d", got)
	}

	// Once the window has elapsed, cleanup must drop every stale key.
	clock.IncTime(31 * time.Minute)
	rl.Cleanup()
	if got := rl.Len(); got != 0 {
		t.Fatalf("expected all expired keys to be evicted, got %d", got)
	}

	// A request after eviction starts a fresh window and is allowed again.
	allow, _, err := rl.Allow("ip:0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allow {
		t.Fatal("expected request after eviction to be allowed")
	}
	if got := rl.Len(); got != 1 {
		t.Fatalf("expected 1 tracked key after re-use, got %d", got)
	}
}

// TestInMemoryThresholdBoundary pins the exact boundary: a policy of Limit=N
// allows N requests within the window and blocks the (N+1)-th.
func TestInMemoryThresholdBoundary(t *testing.T) {
	clock := &mockClock{time: time.Now()}
	policy := core.RateLimitingPolicy{Window: 30 * time.Minute, Limit: 3}
	rl := core.NewInMemoryRateLimiter(clock, policy)

	for i := 1; i <= 3; i++ {
		allow, _, err := rl.Allow("key")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allow {
			t.Fatalf("request %d within the limit should be allowed", i)
		}
	}

	allow, timeout, err := rl.Allow("key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allow {
		t.Fatal("request beyond the limit should be blocked")
	}
	if timeout <= 0 {
		t.Fatalf("expected a positive timeout when blocked, got %v", timeout)
	}
}

// TestInMemoryWindowResetCountsCurrentRequest checks the post-expiry reset: the
// request that opens a new window is itself counted, so the new window allows
// exactly Limit requests (not Limit+1).
func TestInMemoryWindowResetCountsCurrentRequest(t *testing.T) {
	clock := &mockClock{time: time.Now()}
	policy := core.RateLimitingPolicy{Window: 30 * time.Minute, Limit: 2}
	rl := core.NewInMemoryRateLimiter(clock, policy)

	// Exhaust the first window: 2 allowed, 3rd blocked.
	_, _, _ = rl.Allow("key")
	_, _, _ = rl.Allow("key")
	if allow, _, _ := rl.Allow("key"); allow {
		t.Fatal("expected 3rd request to be blocked in the first window")
	}

	// Move into a new window.
	clock.IncTime(31 * time.Minute)

	// The new window must allow exactly Limit requests, with the first
	// (window-opening) request counted.
	if allow, _, _ := rl.Allow("key"); !allow {
		t.Fatal("expected 1st request of the new window to be allowed")
	}
	if allow, _, _ := rl.Allow("key"); !allow {
		t.Fatal("expected 2nd request of the new window to be allowed")
	}
	if allow, _, _ := rl.Allow("key"); allow {
		t.Fatal("expected 3rd request of the new window to be blocked")
	}
}

// TestRedisAndInMemoryAgreeOnThreshold runs the same request sequence against
// both backends and asserts they make identical allow/block decisions, guarding
// against the off-by-one regression where Redis used >= and in-memory used >.
func TestRedisAndInMemoryAgreeOnThreshold(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = client.Close() }()

	const limit = 5
	policy := core.RateLimitingPolicy{Window: 30 * time.Minute, Limit: limit}

	clock := &mockClock{time: time.Now()}
	inMem := core.NewInMemoryRateLimiter(clock, policy)
	redisRL := core.NewRedisRateLimiter(client, "test", policy)

	allowedInMem := 0
	allowedRedis := 0
	for i := 1; i <= limit+3; i++ {
		memAllow, _, memErr := inMem.Allow("key")
		if memErr != nil {
			t.Fatalf("in-memory error at request %d: %v", i, memErr)
		}
		redisAllow, _, redisErr := redisRL.Allow("key")
		if redisErr != nil {
			t.Fatalf("redis error at request %d: %v", i, redisErr)
		}

		if memAllow != redisAllow {
			t.Fatalf("backends disagree at request %d: in-memory=%v redis=%v", i, memAllow, redisAllow)
		}
		if memAllow {
			allowedInMem++
		}
		if redisAllow {
			allowedRedis++
		}
	}

	if allowedInMem != limit {
		t.Fatalf("expected in-memory to allow exactly %d requests, allowed %d", limit, allowedInMem)
	}
	if allowedRedis != limit {
		t.Fatalf("expected redis to allow exactly %d requests, allowed %d", limit, allowedRedis)
	}
}
