package storage

import (
	"backend/internal/core"
	"sync"
	"time"
)

type InMemoryRateLimiter struct {
	memory map[string]*core.RateLimiterEntry
	mutex  sync.Mutex
	policy core.RateLimitingPolicy
	clock  core.Clock
}

func (r *InMemoryRateLimiter) Allow(key string) (allow bool, timeout time.Duration, err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	entry, exists := r.memory[key]

	if !exists {
		r.memory[key] = &core.RateLimiterEntry{
			Count:  0,
			Expiry: r.clock.GetTime().Add(r.policy.Window),
		}
		entry = r.memory[key]
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

func NewInMemoryRateLimiter(clock core.Clock, policy core.RateLimitingPolicy) *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		memory: map[string]*core.RateLimiterEntry{},
		mutex:  sync.Mutex{},
		policy: policy,
		clock:  clock,
	}
}
