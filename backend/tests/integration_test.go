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

	"github.com/stretchr/testify/require"
)

var testCfg = &config.Config{
	App: config.AppConfig{
		Addr: ":8080",
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
		Attributes: config.EmailCredentialAttributes{
			Email:       "email",
			EmailDomain: "domain",
		},
	},
}

var (
	testServer *httptest.Server
)

var testLimiter = newTestRateLimiter(&mockClock{})

var testMailer = mail.DummyMailer{}
var testToken = "TESTTK"
var testemail = "test@email.com"
var testTokenStorage = core.NewInMemoryTokenStorage()

func NewTestAPI() *httpapi.API {
	return httpapi.NewAPI(testCfg, testLimiter, testMailer, &core.StaticTokenGenerator{Token: testToken}, testTokenStorage)
}
func TestMain(m *testing.M) {
	testServer = httptest.NewServer(NewTestAPI().Routes())
	code := m.Run()
	testServer.Close()
	os.Exit(code)
}

func makeVerifyEmailRequest(t *testing.T, token string, email string) *http.Response {
	t.Helper()
	b, err := json.Marshal(map[string]string{"token": token, "email": email})
	require.NoError(t, err)

	resp, err := http.Post(testServer.URL+"/api/verify", "application/json", bytes.NewBuffer(b))
	require.NoError(t, err)
	return resp
}

func makeVerifyLinkRequest(t *testing.T, linkToken string) *http.Response {
	t.Helper()
	b, err := json.Marshal(map[string]string{"link_token": linkToken})
	require.NoError(t, err)

	resp, err := http.Post(testServer.URL+"/api/verify-link", "application/json", bytes.NewBuffer(b))
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
	body, readErr := io.ReadAll(resp.Body)
	require.NoError(t, readErr)

	closeErr := resp.Body.Close()
	require.NoError(t, closeErr)

	var m map[string]any
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

func TestHealthCheckEndpoint(t *testing.T) {
	resp, err := http.Get(testServer.URL + "/api/health")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "ok", string(body))
}

func TestVerifyEmailHappyPath(t *testing.T) {
	tokenErr := testTokenStorage.StoreToken(testemail, testToken)
	require.NoError(t, tokenErr)

	res := makeVerifyEmailRequest(t, testToken, testemail)
	resBody := readResponseBody(t, res)

	require.Equalf(t, http.StatusOK, res.StatusCode, "body: %v", resBody)

}

func TestWrongTokenFails(t *testing.T) {
	tokenErr := testTokenStorage.StoreToken(testemail, testToken)
	require.NoError(t, tokenErr)

	makeSendEmailRequest(t, testemail, "en")

	resp := makeVerifyEmailRequest(t, "ABCDEF", testemail)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

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
func TestSendAndVerifyWithUppercaseEmail(t *testing.T) {
	uppercaseEmail := "Test@Email.Com"

	resp := makeSendEmailRequest(t, uppercaseEmail, "en")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Verify using the same uppercase email — should match via normalization
	res := makeVerifyEmailRequest(t, testToken, uppercaseEmail)
	resBody := readResponseBody(t, res)
	require.Equalf(t, http.StatusOK, res.StatusCode, "body: %v", resBody)
}

func TestSendEmailEmptyData(t *testing.T) {

	res := makeSendEmailRequest(t, "", "en")
	require.Equal(t, http.StatusBadRequest, res.StatusCode)

	resBody := readResponseBody(t, res)

	require.Equal(t, resBody["error"], "email_required")

}

// TestVerifyLinkHappyPath verifies that a valid opaque link token resolves to
// the stored email server-side and yields an issuance JWT — without the email
// ever being supplied by the client.
func TestVerifyLinkHappyPath(t *testing.T) {
	linkToken := "opaque-link-token-happy"
	require.NoError(t, testTokenStorage.StoreLinkToken(linkToken, testemail))

	res := makeVerifyLinkRequest(t, linkToken)
	resBody := readResponseBody(t, res)

	require.Equalf(t, http.StatusOK, res.StatusCode, "body: %v", resBody)
	require.NotEmpty(t, resBody["jwt"])
	require.Equal(t, testemail, resBody["email"])
	require.NotEmpty(t, resBody["irma_server_url"])
}

// TestVerifyLinkTokenIsSingleUse guards against replay: the opaque link token is
// a bearer credential carried in a URL (which may linger in browser history,
// logs, or a Referer header), so a second verification with the same token must
// fail rather than succeed again (issue #44 follow-up).
func TestVerifyLinkTokenIsSingleUse(t *testing.T) {
	linkToken := "opaque-link-token-single-use"
	require.NoError(t, testTokenStorage.StoreLinkToken(linkToken, testemail))

	// First use succeeds.
	first := makeVerifyLinkRequest(t, linkToken)
	require.Equal(t, http.StatusOK, first.StatusCode)

	// Second use of the same token is rejected — it was invalidated on first use.
	second := makeVerifyLinkRequest(t, linkToken)
	require.Equal(t, http.StatusBadRequest, second.StatusCode)

	resBody := readResponseBody(t, second)
	require.Equal(t, "error_token_invalid", resBody["error"])
}

func TestVerifyLinkUnknownTokenFails(t *testing.T) {
	res := makeVerifyLinkRequest(t, "this-token-was-never-stored")
	require.Equal(t, http.StatusBadRequest, res.StatusCode)

	resBody := readResponseBody(t, res)
	require.Equal(t, "error_token_invalid", resBody["error"])
}

func TestVerifyLinkEmptyTokenFails(t *testing.T) {
	res := makeVerifyLinkRequest(t, "")
	require.Equal(t, http.StatusBadRequest, res.StatusCode)

	resBody := readResponseBody(t, res)
	require.Equal(t, "token_required", resBody["error"])
}

// TestSendEmailStoresLinkTokenWithoutEmailInUrl is a regression guard for issue
// #44: the send flow must create a link-token mapping so that the verification
// link can carry an opaque token instead of the email address.
func TestSendEmailStoresLinkTokenWithoutEmailInUrl(t *testing.T) {
	freshStorage := core.NewInMemoryTokenStorage()
	freshMailer := &capturingMailer{}
	api := httpapi.NewAPI(testCfg, testLimiter, freshMailer, &core.StaticTokenGenerator{Token: testToken}, freshStorage)
	srv := httptest.NewServer(api.Routes())
	defer srv.Close()

	b, err := json.Marshal(map[string]string{"email": testemail, "language": "en"})
	require.NoError(t, err)
	resp, err := http.Post(srv.URL+"/api/send", "application/json", bytes.NewBuffer(b))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Exactly one opaque link token was stored, mapping back to the email.
	require.Len(t, freshStorage.LinkTokenMap, 1)
	for linkToken, email := range freshStorage.LinkTokenMap {
		require.Equal(t, testemail, email)
		require.NotContains(t, linkToken, "@", "link token must not embed the email")

		// The opaque token round-trips through the verify-link endpoint.
		res := makeVerifyLinkRequestTo(t, srv, linkToken)
		require.Equal(t, http.StatusOK, res.StatusCode)
	}

	// The rendered email must carry the verification link, and that link must
	// contain the opaque token but never the email address (issue #44).
	require.NotNil(t, freshMailer.last)
	body := freshMailer.last.Body
	require.Contains(t, body, "/enroll#token:", "email must contain the opaque verification link")
	require.NotContains(t, body, "#verify:", "email must not use the old email-bearing link format")
	require.NotContains(t, body, testemail, "the email address must not appear anywhere in the email body/link")
}

func makeVerifyLinkRequestTo(t *testing.T, srv *httptest.Server, linkToken string) *http.Response {
	t.Helper()
	b, err := json.Marshal(map[string]string{"link_token": linkToken})
	require.NoError(t, err)
	resp, err := http.Post(srv.URL+"/api/verify-link", "application/json", bytes.NewBuffer(b))
	require.NoError(t, err)
	return resp
}

// capturingMailer records the last email it was asked to send so tests can make
// assertions about the rendered body (e.g. that it does not leak the address).
type capturingMailer struct {
	last *mail.Email
}

func (m *capturingMailer) SendEmail(e mail.Email) error {
	m.last = &e
	return nil
}
