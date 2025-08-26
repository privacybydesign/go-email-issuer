package httpapi

import (
	"backend/internal/config"
	"backend/internal/core"
	"backend/internal/mail"
	"backend/internal/storage"
	"log"
	"net/http"
	"time"
)

func buildTotalLimiter(cfg *config.Config) *core.TotalRateLimiter {
	emailPolicy := core.RateLimitingPolicy{Limit: cfg.App.EmailLimit, Window: 30 * time.Minute}
	ipPolicy := core.RateLimitingPolicy{Limit: cfg.App.IPLimit, Window: 30 * time.Minute}

	switch cfg.App.StorageType {
	case "inmemory", "memory":
		email := core.NewInMemoryRateLimiter(core.NewSystemClock(), emailPolicy)
		ip := core.NewInMemoryRateLimiter(core.NewSystemClock(), ipPolicy)
		return core.NewTotalRateLimiter(email, ip)

	case "redis":
		rc, err := storage.NewRedisClient(cfg)
		if err != nil {
			log.Fatalf("Error connecting to Redis: %v", err)
		}
		email := core.NewRedisRateLimiter(rc, cfg.Redis.Namespace, emailPolicy)
		ip := core.NewRedisRateLimiter(rc, cfg.Redis.Namespace, ipPolicy)
		return core.NewTotalRateLimiter(email, ip)
	case "redis_sentinel":
		sc, err := storage.NewRedisSentinelClient(cfg)
		if err != nil {
			log.Fatalf("Error connecting to Redis Sentinel: %v", err)
		}
		email := core.NewRedisRateLimiter(sc, cfg.Redis.Namespace, emailPolicy)
		ip := core.NewRedisRateLimiter(sc, cfg.Redis.Namespace, ipPolicy)
		return core.NewTotalRateLimiter(email, ip)

	default:
		log.Fatalf("Unsupported storage type for rate limiter: %s", cfg.App.StorageType)
		return nil
	}
}

type Server struct {
	cfg    *config.Config
	server *http.Server
}

func NewServer(cfg *config.Config) *Server {
	totalLimiter := buildTotalLimiter(cfg)
	smtpMailer := mail.NewSmtpMailer(&cfg.Mail)

	router := NewAPI(cfg, totalLimiter, smtpMailer)

	s := &Server{
		cfg: cfg,
		server: &http.Server{
			Addr:              cfg.App.Addr,
			Handler:           router.Routes(),
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
	return s
}

func (s *Server) ListenAndServe() error {
	if !s.cfg.App.UseTLS {
		log.Printf("Running without TLS")
		return s.server.ListenAndServe()
	}
	log.Printf("Running with TLS")
	return s.server.ListenAndServeTLS(s.cfg.App.TLSCertPath, s.cfg.App.TLSPrivKeyPath)
}
