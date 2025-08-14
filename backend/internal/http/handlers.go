package httpapi

import (
	"backend/internal/core"
	"backend/internal/issue"
	"backend/internal/mail"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

func (a *API) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (a *API) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {

	// token is passed in the body as JSON from the frontend
	var req struct {
		Token string `json:"token"`
	}
	err := decodeJSON(w, r, &req)
	if err != nil || req.Token == "" {
		writeError(w, http.StatusBadRequest, "token_required")
		return
	}

	ttl := time.Duration(a.cfg.App.TTL)
	secret := []byte(a.cfg.App.Secret)

	email, created, err := core.ParseToken(req.Token, ttl, secret)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_or_expired")
		return
	}

	jwtCreator, err := issue.NewIrmaJwtCreator(a.cfg.JWT.PrivateKeyPath, a.cfg.JWT.IssuerID, a.cfg.JWT.Credential, a.cfg.JWT.Attribute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "jwt_creator_error")
		return
	}

	jwt, err := jwtCreator.CreateJwt(email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "jwt_creation_error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"jwt":             jwt,
		"irma_server_url": a.cfg.JWT.IRMAServerURL,
		"expires":         created.Add(ttl).Unix(),
	})

}

func (a *API) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Email    string `json:"email"`
		Language string `json:"language"`
	}
	var in input
	if err := decodeJSON(w, r, &in); err != nil || in.Email == "" {
		writeError(w, http.StatusBadRequest, "email_required")
		return
	}

	frontendBaseURL := a.cfg.App.FrontendBaseURL + "/" + in.Language + "/enroll/"
	secret := []byte(a.cfg.App.Secret)

	// build token
	tok, err := core.MakeToken(in.Email, time.Now(), secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_error")
		return
	}
	verifyURL := fmt.Sprintf("%s/#verify:%s", strings.TrimRight(frontendBaseURL, "/"), tok)

	// render email template and prepare the email
	message, err := mail.PrepareEmail(in.Email, a.cfg.Mail.Template, verifyURL, &a.cfg.Mail)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "email_template_error")
		return
	}
	// send the email
	err = mail.SendEmail(message, &a.cfg.Mail)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "email_send_error")
		return
	}

	// rate limit for sending emails
	if a.limiter != nil {
		ip := clientIP(r)
		allow, _ := a.limiter.Allow(ip, in.Email)
		if !allow {
			writeError(w, http.StatusTooManyRequests, "rate_limited")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message": "email_sent",
	})

}

func clientIP(r *http.Request) string {
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}
	if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
		return cf
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
