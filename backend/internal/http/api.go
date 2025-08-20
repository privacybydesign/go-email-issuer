package httpapi

import (
	"backend/internal/config"
	"backend/internal/core"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

type API struct {
	cfg     *config.Config
	limiter *core.TotalRateLimiter
}

func NewAPIContext(cfg *config.Config, limiter *core.TotalRateLimiter) *API {
	return &API{cfg: cfg, limiter: limiter}
}

// Routes returns app's router

func (a *API) Routes() *mux.Router {

	r := mux.NewRouter()

	r.HandleFunc("/api/health", a.handleHealthCheck).Methods("GET")
	r.HandleFunc("/api/verify", a.handleVerifyEmail).Methods("POST")
	r.HandleFunc("/api/send", a.handleSendEmail).Methods("POST")

	spa := spaHandler{StaticPath: "../frontend/build", IndexPath: "index.html", FileServer: http.FileServer(http.Dir("../frontend/build"))}

	r.PathPrefix("/").Handler(spa)

	return r
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
