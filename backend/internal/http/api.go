package httpapi

import (
	"backend/internal/config"
	"backend/internal/core"
	"backend/internal/mail"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

type API struct {
	cfg            *config.Config
	limiter        *core.TotalRateLimiter
	tokenGenerator core.TokenGenerator
	tokenStorage   core.TokenStorage
	mailer         mail.Mailer
	// trustedProxies holds the parsed CIDR ranges of reverse proxies whose
	// client-IP headers we are willing to trust. See config.TrustedProxies.
	trustedProxies []*net.IPNet
}

func NewAPI(cfg *config.Config, limiter *core.TotalRateLimiter, mailer mail.Mailer, tokenGenerator core.TokenGenerator, tokenStorage core.TokenStorage) *API {
	trustedProxies, err := config.ParseTrustedProxies(cfg.App.TrustedProxies)
	if err != nil {
		// The config is validated at load time, so this should not happen; log
		// and continue with proxy headers untrusted rather than crashing.
		log.Printf("warning: ignoring trusted_proxies: %s", err)
	}
	return &API{cfg: cfg, limiter: limiter, mailer: mailer, tokenGenerator: tokenGenerator, tokenStorage: tokenStorage, trustedProxies: trustedProxies}
}

// Routes returns app's router

func (a *API) Routes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/api/health", a.handleHealthCheck).Methods("GET")
	r.HandleFunc("/api/verify", a.handleVerifyEmail).Methods("POST")
	r.HandleFunc("/api/verify-link", a.handleVerifyLink).Methods("POST")
	r.HandleFunc("/api/done", a.handleVerifyDone).Methods("GET")
	r.HandleFunc("/api/send", a.handleSendEmail).Methods("POST")

	r.HandleFunc("/api/embedded/send", a.handleSendEmail).Methods("POST")
	r.HandleFunc("/api/embedded/verify", a.handleVerifyEmail).Methods("POST")
	r.HandleFunc("/api/embedded/verify-link", a.handleVerifyLink).Methods("POST")

	r.HandleFunc("/api/admin/reset-rate-limit", a.handleResetRateLimit).Methods("POST")

	spa := spaHandler{StaticPath: "../frontend/build", IndexPath: "index.html", FileServer: http.FileServer(http.Dir("../frontend/build"))}

	r.PathPrefix("/").Handler(spa)

	return r
}

// --------------------------------- HELPERS -------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		return err
	}
	return nil
}

func writeError(w http.ResponseWriter, code int, msg string) {
	err := writeJSON(w, code, map[string]string{"error": msg})
	if err != nil {
		log.Println(err)
	}
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
