package core

import (
	"fmt"
	"time"
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
		return false, max(timeRemainingIp, timeRemainingEmail)
	}
	return true, 0
}

// --------------- HELPER FUNCTIONS -------------------

type SystemClock struct{}

func NewSystemClock() *SystemClock        { return &SystemClock{} }
func (c *SystemClock) GetTime() time.Time { return time.Now() }

func max(a, b time.Duration) time.Duration {
	if a >= b {
		return a
	}
	return b
}
