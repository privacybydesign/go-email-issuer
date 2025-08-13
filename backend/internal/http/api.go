package httpapi

import (
	"backend/internal/config"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type API struct {
	cfg *config.Config
	mux *http.ServeMux
}

func New(cfg *config.Config) *API {
	a := &API{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
	return a
}

// Routes returns app's router

func (a *API) Routes() http.Handler {

	a.mux.HandleFunc("GET /healthz", a.handleHealthCheck)

	a.mux.HandleFunc("GET /verify-email/", a.handleVerifyEmail)

	a.mux.HandleFunc("POST /api/send-email", a.handleSendEmail)

	return a.mux

}

// --------------------------------- HELPERS -------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)

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
