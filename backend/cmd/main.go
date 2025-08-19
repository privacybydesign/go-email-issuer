package main

import (
	"backend/internal/config"
	"backend/internal/core"
	httpapi "backend/internal/http"
	"backend/internal/storage"
	"flag"
	"log"
	"net/http"
	"time"
)

func main() {

	// --------------------- LOAD CONFIG --------------------------

	cfgPath := flag.String("config", "config.json", "Path to the config file")
	flag.Parse()

	if *cfgPath == "" {
		log.Fatal("Please provide a config file path using the -config flag")
	}

	log.Printf("Loading configuration from %s", *cfgPath)

	cfg, err := config.LoadFromFile(*cfgPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// --------------------- SET UP SERVER --------------------------

	totalLimiter := buildTotalLimiter(cfg)

	handler := httpapi.New(cfg, totalLimiter).Routes()

	srv := &http.Server{
		Addr:              cfg.App.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", cfg.App.Addr)
	log.Fatal(srv.ListenAndServe())

}

// --------------------- Limiter Middleware --------------------------
func buildTotalLimiter(cfg *config.Config) *core.TotalRateLimiter {
	emailPolicy := core.RateLimitingPolicy{Limit: 3, Window: 30 * time.Minute}
	ipPolicy := core.RateLimitingPolicy{Limit: 10, Window: 30 * time.Minute}

	switch cfg.App.StorageType {
	case "inmemory", "memory":
		email := storage.NewInMemoryRateLimiter(core.NewSystemClock(), emailPolicy)
		ip := storage.NewInMemoryRateLimiter(core.NewSystemClock(), ipPolicy)
		return core.NewTotalRateLimiter(email, ip)

	case "redis":
		rc, err := storage.NewRedisClient(cfg)
		if err != nil {
			log.Fatalf("Error connecting to Redis: %v", err)
		}
		email := storage.NewRedisRateLimiter(rc, cfg.Redis.Namespace, emailPolicy)
		ip := storage.NewRedisRateLimiter(rc, cfg.Redis.Namespace, ipPolicy)
		return core.NewTotalRateLimiter(email, ip)

	default:
		log.Fatalf("Unsupported storage type for rate limiter: %s", cfg.App.StorageType)
		return nil
	}
}
