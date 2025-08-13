package config

import (
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	App           AppConfig
	Mail          MailConfig
	Yivi          YiviConfig
	RedisSentinel RedisSentinelConfig
	Redis         RedisConfig
}
type RedisConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Password  string `json:"password"`
	Namespace string `json:"namespace"`
}

type RedisSentinelConfig struct {
	SentinelHost     string `json:"sentinel_host"`
	SentinelPort     int    `json:"sentinel_port"`
	Password         string `json:"password"`
	MasterName       string `json:"master_name"`
	SentinelUsername string `json:"sentinel_username"`
	Namespace        string `json:"namespace"`
}

type AppConfig struct {
	Addr        string
	BaseURL     string
	Secret      string
	TTL         time.Duration
	StorageType string
}

type MailConfig struct {
	Host       string
	User       string
	Password   string
	Port       int
	From       string
	SenderName string
	Subject    string
	Template   string
	UseTLS     bool
}

type YiviConfig struct {
	PrivateKeyPath string
	IssuerID       string
	CredentialType string
	Attribute      string
}

func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Addr:        getEnv("ADDR", ":8080"),
			BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
			Secret:      getEnv("SECRET", "changeme"),
			TTL:         mustParseDuration(getEnv("TTLDUR", "15m")),
			StorageType: getEnv("STORAGE_TYPE", "inmemory"),
		},
		Mail: MailConfig{
			Host:       getEnv("SMTP_HOST", "your.smtp.host"),
			User:       getEnv("SMTP_USER", "user"),
			Password:   getEnv("SMTP_PASSWORD", "password"),
			Port:       mustParseInt(getEnv("SMTP_PORT", "587")),
			From:       getEnv("SMTP_FROM", "noreply@staging.yivi.app"),
			SenderName: getEnv("EMAIL_SENDER", "Yivi Portal"),
			Subject:    getEnv("EMAIL_SUBJECT", "Verify your email"),
			Template:   getEnv("TEMPLATE_PATH", "./internal/mail/templates/verify_email.html"),
			UseTLS:     mustParseBool(getEnv("SMTP_USE_TLS", "false")),
		},
		Yivi: YiviConfig{
			PrivateKeyPath: getEnv("PRIVATE_KEY_PATH", "./internal/issue/keys/private_key.pem"),
			IssuerID:       getEnv("ISSUER_ID", "pbdf"),
			CredentialType: getEnv("CREDENTIAL_TYPE", "email"),
			Attribute:      getEnv("ATTRIBUTE", "email"),
		},
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// --- helpers ---

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func mustParseInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(fmt.Errorf("invalid int %q: %w", s, err))
	}
	return i
}

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Errorf("invalid duration %q: %w", s, err))
	}
	return d
}

func mustParseBool(s string) bool {
	switch s {
	case "1", "t", "T", "true", "TRUE", "True", "yes", "on":
		return true
	case "0", "f", "F", "false", "FALSE", "False", "no", "off":
		return false
	default:
		panic(fmt.Errorf("invalid bool %q", s))
	}
}

func validate(cfg *Config) error {
	// App
	if cfg.App.Secret == "" || cfg.App.Secret == "changeme" {
		return errors.New("SECRET must be set to a non-default value")
	}
	if cfg.App.TTL <= 0 {
		return fmt.Errorf("TTLDUR must be > 0 (got %s)", cfg.App.TTL)
	}

	// Mail
	if cfg.Mail.Host == "" {
		return errors.New("SMTP_HOST is required")
	}
	if cfg.Mail.Port <= 0 || cfg.Mail.Port > 65535 {
		return fmt.Errorf("SMTP_PORT out of range: %d", cfg.Mail.Port)
	}
	if _, err := mail.ParseAddress(cfg.Mail.From); err != nil {
		return fmt.Errorf("SMTP_FROM invalid: %w", err)
	}
	if cfg.Mail.Template != "" {
		if _, err := os.Stat(cfg.Mail.Template); err != nil {
			return fmt.Errorf("TEMPLATE_PATH not found (%s): %w", cfg.Mail.Template, err)
		}
	}

	// Yivi
	if cfg.Yivi.PrivateKeyPath != "" {
		if _, err := os.Stat(filepath.Clean(cfg.Yivi.PrivateKeyPath)); err != nil {
			return fmt.Errorf("PRIVATE_KEY_PATH not found: %w", err)
		}
	}
	if cfg.Yivi.IssuerID == "" {
		return errors.New("ISSUER_ID is required")
	}
	return nil
}
