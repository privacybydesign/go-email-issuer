package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	App           AppConfig
	Mail          MailConfig
	JWT           JWTConfig
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
	Addr            string       `json:"addr"`
	BaseURL         string       `json:"base_url"`
	FrontendBaseURL string       `json:"frontend_base_url"`
	Secret          string       `json:"secret"`
	TTL             JSONDuration `json:"ttl"`
	StorageType     string       `json:"storage_type"`
}
type MailTemplate struct {
	Subject     string `json:"mail_subject"`
	TemplateDir string `json:"mail_template_dir"`
}

type MailConfig struct {
	Host          string                  `json:"mail_host"`
	User          string                  `json:"mail_user"`
	Password      string                  `json:"mail_password"`
	Port          int                     `json:"mail_port"`
	From          string                  `json:"mail_from"`
	SenderName    string                  `json:"mail_sender_name"`
	UseTLS        bool                    `json:"mail_use_tls"`
	MailTemplates map[string]MailTemplate `json:"mail_templates"`
}

type JWTConfig struct {
	IRMAServerURL  string `json:"irma_server_url"`
	PrivateKeyPath string `json:"private_key_path"`
	IssuerID       string `json:"issuer_id"`
	CredentialType string `json:"credential_type"`
	Credential     string `json:"full_credential"`
	Attribute      string `json:"attribute"`
}

func LoadFromFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("failed to close file: %v", err)
		}
	}()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// --- helpers ---

func validate(cfg *Config) error {
	// App
	if cfg.App.Secret == "" || cfg.App.Secret == "changeme" {
		return errors.New("SECRET must be set to a non-default value")
	}
	if cfg.App.TTL <= 0 {
		return fmt.Errorf("TTLDUR must be > 0 (got %s)", time.Duration(cfg.App.TTL))
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

	// Yivi issuance session JWT
	if cfg.JWT.PrivateKeyPath != "" {
		if _, err := os.Stat(filepath.Clean(cfg.JWT.PrivateKeyPath)); err != nil {
			return fmt.Errorf("PRIVATE_KEY_PATH not found: %w", err)
		}
	}
	if cfg.JWT.IssuerID == "" {
		return errors.New("ISSUER_ID is required")
	}
	return nil
}

type JSONDuration time.Duration

func (d *JSONDuration) UnmarshalJSON(b []byte) error {
	// Try string: "15m", "1h30m", etc.
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		dd, err := time.ParseDuration(s)
		if err != nil {
			return fmt.Errorf("invalid duration %q: %w", s, err)
		}
		*d = JSONDuration(dd)
		return nil
	}
	return fmt.Errorf("invalid duration: %s", string(b))
}
