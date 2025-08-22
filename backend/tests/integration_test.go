package main

import (
	"backend/internal/config"
	"backend/internal/core"
	httpapi "backend/internal/http"
	"backend/internal/mail"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testCfg = &config.Config{
	App: config.AppConfig{
		Addr:   ":8080",
		Secret: string(testsecret),
	},
	Mail: config.MailConfig{
		From: "noreply@example.com",
		MailTemplates: map[string]config.MailTemplate{
			"en": {
				Subject:     "Verify your email",
				TemplateDir: "../internal/mail/templates/email_en.html",
			},
		},
	},
	JWT: config.JWTConfig{
		IRMAServerURL:  "http://localhost:8000",
		PrivateKeyPath: "./keys/priv.pem",
		IssuerID:       "email-issuer",
		Credential:     "irma-demo.sidn-pbdf.email",
		Attribute:      "email",
	},
}

var (
	testServer *httptest.Server
)

var testLimiter = newTestRateLimiter(&mockClock{})

var testMailer = mail.DummyMailer{}

func NewTestAPI() *httpapi.API {
	return httpapi.NewAPI(testCfg, testLimiter, testMailer)
}
func TestMain(m *testing.M) {
	testServer = httptest.NewServer(NewTestAPI().Routes())
	code := m.Run()
	testServer.Close()
	os.Exit(code)
}

func makeVerifyEmailRequest(t *testing.T, token string) *http.Response {
	t.Helper()
	b, err := json.Marshal(map[string]string{"token": token})
	require.NoError(t, err)

	resp, err := http.Post(testServer.URL+"/api/verify", "application/json", bytes.NewBuffer(b))
	require.NoError(t, err)
	return resp
}

func makeSendEmailRequest(t *testing.T, email, language string) *http.Response {
	t.Helper()
	b, err := json.Marshal(map[string]string{"email": email, "language": language})
	require.NoError(t, err)

	resp, err := http.Post(testServer.URL+"/api/send", "application/json", bytes.NewBuffer(b))
	require.NoError(t, err)
	return resp
}

func readResponseBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

func TestHealthCheckEndpoint(t *testing.T) {
	resp, err := http.Get(testServer.URL + "/api/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(body))
}

func TestVerifyEmailHappyPath(t *testing.T) {
	testToken, err := core.MakeToken(testemail, testsecret, time.Now().Add(time.Hour).Unix())
	require.NoError(t, err)

	res := makeVerifyEmailRequest(t, testToken)
	resBody := readResponseBody(t, res)

	require.Equalf(t, http.StatusOK, res.StatusCode, "body: %v", resBody)

}

func TestVerifyEmail_InvalidAndExpired(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		resp := makeVerifyEmailRequest(t, "not-a-token")
		defer resp.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("expired", func(t *testing.T) {
		tok, err := core.MakeToken(testemail, testsecret, time.Now().Add(-time.Hour).Unix())
		require.NoError(t, err)
		resp := makeVerifyEmailRequest(t, tok)
		defer resp.Body.Close()
		require.True(t, resp.StatusCode == http.StatusBadRequest)
	})
}
func TestSendEmailEmptyData(t *testing.T) {

	res := makeSendEmailRequest(t, "", "en")
	require.Equal(t, http.StatusBadRequest, res.StatusCode)

	resBody := readResponseBody(t, res)

	require.Equal(t, resBody["error"], "email_required")

}

func TestSendEmailHappyPath(t *testing.T) {

	resp := makeSendEmailRequest(t, testemail, "en")
	if resp == nil {
		t.Fatalf("Failed to make request")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %s", resp.Status)
	}

}
