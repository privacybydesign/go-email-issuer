package core

import (
	"bytes"
	"log"
	"strings"
	"testing"
	"time"
)

// captureLog redirects the standard logger to a buffer for the duration of
// the test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

func TestAllowLogsRejections(t *testing.T) {
	const ip = "203.0.113.7"
	const email = "someone@example.com"

	tests := []struct {
		name       string
		ipLimit    int
		emailLimit int
		want       string
	}{
		{name: "ip limit", ipLimit: 1, emailLimit: 10, want: "rejected request from 203.0.113.7, ip limit reached"},
		{name: "email limit", ipLimit: 10, emailLimit: 1, want: "rejected request from 203.0.113.7, email limit reached"},
		{name: "both limits", ipLimit: 1, emailLimit: 1, want: "rejected request from 203.0.113.7, ip and email limit reached"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := NewTotalRateLimiter(
				NewInMemoryRateLimiter(NewSystemClock(), RateLimitingPolicy{Limit: tc.emailLimit, Window: time.Minute}),
				NewInMemoryRateLimiter(NewSystemClock(), RateLimitingPolicy{Limit: tc.ipLimit, Window: time.Minute}),
			)
			buf := captureLog(t)

			if allow, _ := l.Allow(ip, email); !allow {
				t.Fatal("first request should be allowed")
			}
			if buf.Len() != 0 {
				t.Errorf("allowed request should not log, got %q", buf.String())
			}

			if allow, _ := l.Allow(ip, email); allow {
				t.Fatal("second request should be rejected")
			}
			got := buf.String()
			if !strings.Contains(got, tc.want) {
				t.Errorf("log = %q, want it to contain %q", got, tc.want)
			}
			if strings.Contains(got, email) {
				t.Errorf("log should not contain the email address, got %q", got)
			}
		})
	}
}
