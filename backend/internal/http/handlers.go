package httpapi

import (
	"backend/internal/core"
	"backend/internal/issue"
	"backend/internal/mail"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (a *API) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (a *API) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {

	tok := r.URL.Path[len("/verify-email/"):]
	if tok == "" {
		writeError(w, http.StatusBadRequest, "token_required")
		return
	}

	ttl := a.cfg.App.TTL
	secret := []byte(a.cfg.App.Secret)

	// verify: payload is the email, created is when token was issued
	email, created, err := core.ParseToken(tok, ttl, time.Now(), secret)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_or_expired")
		return
	}

	jwtCreator, err := issue.NewIrmaJwtCreator(a.cfg.Yivi.PrivateKeyPath, a.cfg.Yivi.IssuerID, a.cfg.Yivi.CredentialType, a.cfg.Yivi.Attribute)
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
		"status":   "verified",
		"email":    email,
		"yivi-jwt": jwt,
		"created":  created.Unix(),
		"expires":  created.Add(ttl).Unix(),
	})

}

func (a *API) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Email string `json:"email"`
	}
	var in input
	if err := decodeJSON(w, r, &in); err != nil || in.Email == "" {
		writeError(w, http.StatusBadRequest, "email_required")
		return
	}

	baseURL := a.cfg.App.BaseURL
	secret := []byte(a.cfg.App.Secret)

	// build token
	tok, err := core.MakeToken(in.Email, time.Now(), secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token_error")
		return
	}
	verifyURL := fmt.Sprintf("%s/verify-email/%s", strings.TrimRight(baseURL, "/"), tok)

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

	writeJSON(w, http.StatusAccepted, map[string]any{
		"verifyURL": verifyURL,
		"message":   "email_sent",
	})

}
