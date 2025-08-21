package main

import (
	"backend/internal/core"
	"testing"
	"time"
)

func TestRateLimiterWithSameIPandEmail(t *testing.T) {
	clock := &mockClock{time: time.Now()}
	rl := newTestRateLimiter(clock)

	ip := "127.0.0.1"
	email := "test@email.com"

	// First 5 should pass
	for i := 0; i < 5; i++ {
		allow, _ := rl.Allow(ip, email)
		if !allow {
			t.Fatalf("unexpected fail at attempt %d", i+1)
		}
	}

	// 6th should fail
	allow, timeout := rl.Allow(ip, email)
	if allow {
		t.Fatal("expected to fail at 6th attempt")
	}
	if timeout <= 0 {
		t.Fatal("expected a positive timeout")
	}
}
func TestRateLimiterWithDifferentEmails(t *testing.T) {
	clock := &mockClock{time: time.Now()}
	rl := newTestRateLimiter(clock)
	ip := "127.0.0.1"
	emails := []string{
		"test@email.com",
		"test1@email.com",
		"test2@email.com",
		"test3@email.com",
		"test4@email.com",
		"test5@email.com",
	}

	// 5 different emails with the same ip allowed
	for i, email := range emails {
		allow, _ := rl.Allow(ip, email)
		if !allow {
			t.Fatalf("unexpected fail at attempt %d", i+1)
		}
	}

	// 6th email from the same ip should fail
	allow, _ := rl.Allow(ip, emails[5])
	if !allow {
		t.Fatal("expected to fail at 6th attempt")
	}

}

func TestRateLimiterWindowReset(t *testing.T) {
	clock := &mockClock{time: time.Now()}
	rl := newTestRateLimiter(clock)

	ip := "0.0.0.0"
	email := "test@email.com"

	for i := 0; i < 5; i++ {
		rl.Allow(ip, email)
	}

	clock.IncTime(31 * time.Minute)

	allow, timeout := rl.Allow(ip, email)
	if !allow {
		t.Fatalf("expected the new window, got timeout %v minutes", timeout.Minutes())
	}
	if timeout > 0 {
		t.Fatal("timeout should not be bigger than 0")

	}

}

func TestRateLimiterDifferentIPs(t *testing.T) {
	clock := &mockClock{time: time.Now()}
	rl := newTestRateLimiter(clock)

	ips := []string{
		"127.0.0.1",
		"127.0.0.2",
		"127.0.0.3",
		"127.0.0.4",
		"127.0.0.5",
	}
	email := "test@email.com"

	for _, ip := range ips {
		allow, _ := rl.Allow(ip, email)
		if !allow {
			t.Fatal("expected to for the first 3 attempts to pass")
		}
	}

	// 4th with same ip should fail
	allow, _ := rl.Allow("127.0.0.6", email)
	if allow {
		t.Fatal("expected the 4th attempt to succeed but it failed")
	}

}

func newTestRateLimiter(clock core.Clock) *core.TotalRateLimiter {
	ipPolicy := core.RateLimitingPolicy{
		Window: 30 * time.Minute,
		Limit:  5,
	}
	emailPolicy := core.RateLimitingPolicy{
		Window: 30 * time.Minute,
		Limit:  5,
	}

	email := core.NewInMemoryRateLimiter(clock, emailPolicy)
	ip := core.NewInMemoryRateLimiter(clock, ipPolicy)
	testRateLimiter := core.NewTotalRateLimiter(email, ip)

	return testRateLimiter

}

type mockClock struct {
	time time.Time
}

func (c *mockClock) GetTime() time.Time {
	return c.time
}

func (c *mockClock) IncTime(time time.Duration) {
	c.time = c.time.Add(time)
}
