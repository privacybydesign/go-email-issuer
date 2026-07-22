package httpapi

import (
	"backend/internal/core"
	"backend/internal/issue"
	"backend/internal/mail"
	"backend/internal/validators"
	"crypto/subtle"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (a *API) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	if err != nil {
		log.Printf("error: %s", err)
	}
}

type spaHandler struct {
	StaticPath string
	IndexPath  string
	FileServer http.Handler
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Join internally call path.Clean to prevent directory traversal
	path := filepath.Join(h.StaticPath, r.URL.Path)

	// check whether a file exists or is a directory at the given path
	fi, err := os.Stat(path)
	if os.IsNotExist(err) || fi.IsDir() {
		// file does not exist or path is a directory, serve index.html
		http.ServeFile(w, r, filepath.Join(h.StaticPath, h.IndexPath))
		return
	}

	if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static file
	h.FileServer.ServeHTTP(w, r)
}
func (a *API) handleVerifyDone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	decode_err := decodeJSON(w, r, &req)
	if decode_err != nil || req.Email == "" {
		writeError(w, http.StatusBadRequest, "email_required")
		return
	}

	validator := validators.EmailValidator{}
	valid, parsedAddress, errCode := validator.ParseAndValidateEmailAddress(req.Email)
	if !valid {
		writeError(w, http.StatusBadRequest, *errCode)
		return
	}

	remove_err := a.tokenStorage.RemoveToken(*parsedAddress)
	if remove_err != nil {
		writeError(w, http.StatusInternalServerError, "error_removing_token")
		return
	}

}

// handleResetRateLimit clears the rate-limit counter for a single email
// address. It is meant for operators to unblock a user who locked themselves
// out by mistake. Access requires the admin token configured in app.admin_token,
// passed as "Authorization: Bearer <token>". When no admin token is configured
// the endpoint is disabled; when set, the token must meet the minimum length
// enforced at startup (config.MinAdminTokenLength).
func (a *API) handleResetRateLimit(w http.ResponseWriter, r *http.Request) {
	if !a.authorizeAdmin(w, r) {
		return
	}

	var req struct {
		Email string `json:"email"`
	}
	if err := decodeJSON(w, r, &req); err != nil || req.Email == "" {
		writeError(w, http.StatusBadRequest, "email_required")
		return
	}

	validator := validators.EmailValidator{}
	valid, parsedAddress, errCode := validator.ParseAndValidateEmailAddress(req.Email)
	if !valid {
		writeError(w, http.StatusBadRequest, *errCode)
		return
	}

	if a.limiter == nil {
		writeError(w, http.StatusInternalServerError, "rate_limiter_not_configured")
		return
	}

	if err := a.limiter.ResetEmail(*parsedAddress); err != nil {
		writeError(w, http.StatusInternalServerError, "error_resetting_rate_limit")
		return
	}

	// Audit trail: admin reset actions are privileged, so record who was
	// unblocked and from where.
	log.Printf("admin: rate limit reset for %q from %s", *parsedAddress, a.clientIP(r))

	jserr := writeJSON(w, http.StatusOK, map[string]any{
		"message": "rate_limit_reset",
	})
	if jserr != nil {
		log.Printf("error: %s", jserr)
	}
}

// authorizeAdmin checks the bearer token against the configured admin token in
// constant time. It writes the appropriate error response and returns false
// when the request is not authorized.
func (a *API) authorizeAdmin(w http.ResponseWriter, r *http.Request) bool {
	configured := a.cfg.App.AdminToken
	if configured == "" {
		writeError(w, http.StatusForbidden, "admin_endpoint_disabled")
		return false
	}

	provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) != 1 {
		// Log rejected attempts so repeated failures (e.g. token guessing)
		// against this network-reachable route are visible in the logs.
		log.Printf("admin: rejected request with invalid token from %s", a.clientIP(r))
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	return true
}

func (a *API) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {
	// token is passed in the body as JSON from the frontend
	var req struct {
		Token string `json:"token"`
		Email string `json:"email"`
	}
	decode_err := decodeJSON(w, r, &req)
	if decode_err != nil || req.Token == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "token_or_email_required")
		return
	}
	// Validate and normalize the email address
	validator := validators.EmailValidator{}
	valid, parsedAddress, errCode := validator.ParseAndValidateEmailAddress(req.Email)
	if !valid {
		writeError(w, http.StatusBadRequest, *errCode)
		return
	}

	if a.tokenStorage == nil {
		http.Error(w, "token generator not configured", http.StatusInternalServerError)
		return
	}
	expectedToken, retrieve_err := a.tokenStorage.RetrieveToken(*parsedAddress)
	if retrieve_err != nil {
		writeError(w, http.StatusBadRequest, "error_token_invalid")
		return
	}

	if expectedToken != req.Token {
		writeError(w, http.StatusBadRequest, "error_invalid_token")
		return
	}

	jwtCreator, creator_err := issue.NewIrmaJwtCreator(a.cfg.JWT)
	if creator_err != nil {
		writeError(w, http.StatusInternalServerError, "jwt_creator_error")
		return
	}

	jwt, create_err := jwtCreator.CreateJwt(*parsedAddress)
	if create_err != nil {
		writeError(w, http.StatusInternalServerError, "jwt_creation_error")
		return
	}

	// The verification code is single-use: once it has been successfully
	// redeemed (i.e. a JWT was issued for it), invalidate it so it cannot be
	// used again. If we cannot invalidate it we must not hand out the JWT,
	// otherwise the code would remain reusable until it expires.
	if remove_err := a.tokenStorage.RemoveToken(*parsedAddress); remove_err != nil {
		writeError(w, http.StatusInternalServerError, "error_invalidating_token")
		return
	}

	jserr := writeJSON(w, http.StatusOK, map[string]any{
		"jwt":             jwt,
		"irma_server_url": a.cfg.JWT.IRMAServerURL,
	})
	if jserr != nil {
		log.Printf("error: %s", jserr)
	}
}

// handleVerifyLink verifies an email address from an opaque link token embedded
// in the verification link. The email is looked up server-side from the token,
// so it is never carried in the URL. On success it returns the issuance JWT and
// the resolved email address (the latter only in the POST response body, never
// in a URL) so the frontend can finish issuance and clean up the code.
func (a *API) handleVerifyLink(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LinkToken string `json:"link_token"`
	}
	decode_err := decodeJSON(w, r, &req)
	if decode_err != nil || req.LinkToken == "" {
		writeError(w, http.StatusBadRequest, "token_required")
		return
	}

	if a.tokenStorage == nil {
		http.Error(w, "token storage not configured", http.StatusInternalServerError)
		return
	}

	email, retrieve_err := a.tokenStorage.RetrieveEmailByLinkToken(req.LinkToken)
	if retrieve_err != nil {
		writeError(w, http.StatusBadRequest, "error_token_invalid")
		return
	}

	// Invalidate the link token immediately so the verification link is
	// single-use and cannot be replayed (it is a bearer credential carried in
	// a URL that may linger in history, logs, or the Referer header). A failure
	// here must not block the legitimate user, so we only log it.
	if remove_err := a.tokenStorage.RemoveLinkToken(req.LinkToken); remove_err != nil {
		log.Printf("warning: failed to invalidate link token after use: %s", remove_err)
	}

	// Re-validate and normalize the stored email defensively.
	validator := validators.EmailValidator{}
	valid, parsedAddress, errCode := validator.ParseAndValidateEmailAddress(email)
	if !valid {
		writeError(w, http.StatusBadRequest, *errCode)
		return
	}

	jwtCreator, creator_err := issue.NewIrmaJwtCreator(a.cfg.JWT)
	if creator_err != nil {
		writeError(w, http.StatusInternalServerError, "jwt_creator_error")
		return
	}

	jwt, create_err := jwtCreator.CreateJwt(*parsedAddress)
	if create_err != nil {
		writeError(w, http.StatusInternalServerError, "jwt_creation_error")
		return
	}

	jserr := writeJSON(w, http.StatusOK, map[string]any{
		"jwt":             jwt,
		"irma_server_url": a.cfg.JWT.IRMAServerURL,
		"email":           *parsedAddress,
	})
	if jserr != nil {
		log.Printf("error: %s", jserr)
	}
}

func (a *API) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Email    string `json:"email"`
		Language string `json:"language"`
	}

	var in input
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "email_required")
		return
	}

	// Validate email address format
	validator := validators.EmailValidator{}
	valid, parsedAddress, errCode := validator.ParseAndValidateEmailAddress(in.Email)
	if !valid {
		writeError(w, http.StatusBadRequest, *errCode)
		return
	}

	// render email template and prepare the email
	mailTmpl, ok := a.cfg.Mail.MailTemplates[in.Language]
	if !ok {
		mailTmpl = a.cfg.Mail.MailTemplates["en"]
	}

	if a.tokenGenerator == nil {
		http.Error(w, "token generator not configured", http.StatusInternalServerError)
		return
	}
	tok, err := a.tokenGenerator.GenerateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error_generating_token")
		return
	}

	// Make sure we use the parsed email address from the validator, to ensure that we e.g. trim any whitespace and remove any full name from the RFC-5322 format
	err = a.tokenStorage.StoreToken(*parsedAddress, tok)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error_storing_token")
		return
	}

	// Generate an opaque link token and store a reverse mapping to the email
	// address. The verification link carries only this token; the email is
	// resolved server-side, so it never appears in the URL (and therefore not
	// in browser history, server logs, or the Referer header) — see issue #44.
	linkTok, err := core.GenerateLinkToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error_generating_token")
		return
	}
	if err = a.tokenStorage.StoreLinkToken(linkTok, *parsedAddress); err != nil {
		writeError(w, http.StatusInternalServerError, "error_storing_token")
		return
	}

	baseURL := strings.TrimSuffix(a.cfg.App.BaseURL, "/")
	verifyURL := fmt.Sprintf("%s/%s/enroll#token:%s", baseURL, in.Language, linkTok)

	tmplStr, err := mail.RenderHTMLtemplate(mailTmpl.TemplateDir, verifyURL, tok)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "template_render_error")
		return
	}

	// For sending the email, we can used the unparsed email address from the input, since the mailer will use the parsed email address from the validator as the "To" address. This allows us to e.g. keep the full name in the "To" field if the user provided an RFC-5322 formatted email address.
	emData := mail.Email{From: a.cfg.Mail.From, To: in.Email,
		Subject: mailTmpl.Subject,
		Body:    tmplStr,
	}

	// rate limit for sending emails
	if a.limiter != nil {
		ip := a.clientIP(r)
		allow, _ := a.limiter.Allow(ip, *parsedAddress)
		if !allow {
			writeError(w, http.StatusTooManyRequests, "error_ratelimit")
			return
		}
	}

	err = a.mailer.SendEmail(emData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error_sending_email")
		return
	}

	jserr := writeJSON(w, http.StatusOK, map[string]any{
		"message": "email_sent",
	})
	if jserr != nil {
		log.Printf("error: %s", jserr)
	}

}

// clientIP determines the originating client's IP address for rate-limiting.
//
// Proxy-supplied headers (X-Forwarded-For, CF-Connecting-IP) can be set to any
// value by an HTTP client, so trusting them unconditionally lets a caller spoof
// their IP and bypass the IP-based rate limit. We therefore only consult those
// headers when the immediate peer (the TCP connection's remote address) is a
// configured trusted proxy. Otherwise, and whenever the headers yield no usable
// address, we fall back to the connection's remote address.
func (a *API) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	peerIP := net.ParseIP(host)
	if peerIP == nil || !a.isTrustedProxy(peerIP) {
		// The direct peer is not a trusted proxy: ignore proxy headers.
		return host
	}

	// CF-Connecting-IP holds the single originating client IP set by Cloudflare.
	if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
		if net.ParseIP(cf) != nil {
			return cf
		}
	}

	// X-Forwarded-For is a comma-separated chain "client, proxy1, proxy2, ...".
	// Walk it right-to-left and return the first address that is not itself a
	// trusted proxy. That is the closest we can get to the real client while
	// discarding any values a caller may have injected before the request
	// reached our trusted proxies.
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ipStr := strings.TrimSpace(parts[i])
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if a.isTrustedProxy(ip) {
				continue
			}
			return ipStr
		}
	}

	return host
}

// isTrustedProxy reports whether ip falls within one of the configured
// trusted-proxy CIDR ranges.
func (a *API) isTrustedProxy(ip net.IP) bool {
	for _, network := range a.trustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
