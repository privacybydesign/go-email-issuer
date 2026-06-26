package main

import (
	"backend/internal/config"
	"backend/internal/core"
	httpapi "backend/internal/http"
	"backend/internal/mail"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newAdminTestServer(t *testing.T, adminToken string, limiter *core.TotalRateLimiter) *httptest.Server {
	t.Helper()
	cfg := &config.Config{
		App:  config.AppConfig{Addr: ":8080", AdminToken: adminToken},
		Mail: config.MailConfig{From: "noreply@example.com"},
		JWT:  config.JWTConfig{IRMAServerURL: "http://localhost:8000", IssuerID: "email-issuer"},
	}
	api := httpapi.NewAPI(cfg, limiter, mail.DummyMailer{}, &core.StaticTokenGenerator{Token: "TESTTK"}, core.NewInMemoryTokenStorage())
	srv := httptest.NewServer(api.Routes())
	t.Cleanup(srv.Close)
	return srv
}

func postResetRateLimit(t *testing.T, srv *httptest.Server, token, email string) *http.Response {
	t.Helper()
	b, err := json.Marshal(map[string]string{"email": email})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/api/admin/reset-rate-limit", bytes.NewBuffer(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func TestResetRateLimitDisabledWhenNoAdminToken(t *testing.T) {
	srv := newAdminTestServer(t, "", newTestRateLimiter(&mockClock{time: time.Now()}))

	resp := postResetRateLimit(t, srv, "anything", "user@example.com")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestResetRateLimitRejectsWrongToken(t *testing.T) {
	srv := newAdminTestServer(t, "s3cret", newTestRateLimiter(&mockClock{time: time.Now()}))

	resp := postResetRateLimit(t, srv, "wrong-token", "user@example.com")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestResetRateLimitRejectsInvalidEmail(t *testing.T) {
	srv := newAdminTestServer(t, "s3cret", newTestRateLimiter(&mockClock{time: time.Now()}))

	resp := postResetRateLimit(t, srv, "s3cret", "not-an-email")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestResetRateLimitUnblocksEmail(t *testing.T) {
	limiter := newTestRateLimiter(&mockClock{time: time.Now()})
	srv := newAdminTestServer(t, "s3cret", limiter)

	email := "locked@example.com"

	// Exhaust the per-email limit (10) across enough different IPs that the
	// per-IP limit (5) is not what blocks us.
	for i := range 11 {
		limiter.Allow("198.51.100."+string(rune('0'+i%10)), email)
	}
	allow, _ := limiter.Allow("203.0.113.9", email)
	require.False(t, allow, "expected email to be rate-limited before reset")

	resp := postResetRateLimit(t, srv, "s3cret", email)
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	allow, _ = limiter.Allow("203.0.113.9", email)
	require.True(t, allow, "expected email to be allowed after admin reset")
}

func TestResetRateLimitNormalizesEmail(t *testing.T) {
	limiter := newTestRateLimiter(&mockClock{time: time.Now()})
	srv := newAdminTestServer(t, "s3cret", limiter)

	// The limiter keys on the normalized (lowercase) address; the admin request
	// uses a mixed-case form and must still hit the same entry.
	email := "locked@example.com"
	for i := range 11 {
		limiter.Allow("198.51.100."+string(rune('0'+i%10)), email)
	}
	allow, _ := limiter.Allow("203.0.113.9", email)
	require.False(t, allow)

	resp := postResetRateLimit(t, srv, "s3cret", "Locked@Example.com")
	defer func() { _ = resp.Body.Close() }()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	allow, _ = limiter.Allow("203.0.113.9", email)
	require.True(t, allow, "expected mixed-case reset to unblock the normalized email")
}
