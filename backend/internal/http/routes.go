package httpapi

import (
	"backend/internal/config"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/internal/core"
	"backend/internal/mail"
	"fmt"
)

// Routes returns app's router

func Routes() http.Handler {

	mux := http.NewServeMux()

	// ------------------------------------- HEALTH ------------------------------------

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// ------------------------------------- VERIFY EMAIL TOKEN ------------------------------------
	mux.HandleFunc("GET /verify-email/", func(w http.ResponseWriter, r *http.Request) {

		tok := r.URL.Path[len("/verify-email/"):]
		if tok == "" {
			writeError(w, http.StatusBadRequest, "token_required")
			return
		}

		// load env variables from config
		cfg := config.Load()
		ttlMin, _ := strconv.Atoi(cfg.TTLMIN)
		maxAge := time.Duration(ttlMin) * time.Minute
		secret := []byte(cfg.SECRET)

		// verify: payload is the email, created is when token was issued
		email, created, err := core.ParseToken(tok, maxAge, time.Now(), secret)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_or_expired")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "verified",
			"email":   email,
			"created": created.Unix(),
			"expires": created.Add(maxAge).Unix(),
		})
	})

	// ---------------------  SEND EMAIL -------------------------------------------------
	mux.HandleFunc("POST /api/send-email", func(w http.ResponseWriter, r *http.Request) {
		type input struct {
			Email string `json:"email"`
		}
		var in input
		if err := decodeJSON(w, r, &in); err != nil || in.Email == "" {
			writeError(w, http.StatusBadRequest, "email_required")
			return
		}

		cfg := config.Load()

		baseURL := cfg.BASE_URL
		secret := []byte(cfg.SECRET)

		// build token
		tok, err := core.MakeToken(in.Email, time.Now(), secret)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token_error")
			return
		}
		verifyURL := fmt.Sprintf("%s/verify-email/%s", strings.TrimRight(baseURL, "/"), tok)

		// render email template and prepare the email
		message, err := mail.PrepareEmail(in.Email, cfg.TEMPLATE_PATH, verifyURL, &cfg)

		if err != nil {
			writeError(w, http.StatusInternalServerError, "email_template_error")
			return
		}
		// send the email
		err = mail.SendEmail(message, &cfg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "email_send_error")
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{
			"verifyURL": verifyURL,
			"message":   "email_sent",
		})

	})
	return mux

}

// --------------------------------- HELPERS -------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)

}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		return err
	}

	if err := dec.Decode(new(struct{})); !errors.Is(err, io.EOF) {
		return errors.New("body must contain a single JSON object")
	}
	return nil

}
