package config

import (
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Config struct {
	App           AppConfig           `json:"app"`
	Mail          MailConfig          `json:"mail"`
	JWT           JWTConfig           `json:"jwt"`
	RedisSentinel RedisSentinelConfig `json:"redis_sentinel"`
	Redis         RedisConfig         `json:"redis"`
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
	Namespace        string `json:"sentinel_namespace"`
}

type AppConfig struct {
	Addr           string         `json:"addr"`
	BaseURL        string         `json:"base_url"`
	StorageType    string         `json:"storage_type"`
	UseTLS         bool           `json:"use_tls,omitempty"`
	TLSPrivKeyPath string         `json:"tls_priv_key_path,omitempty"`
	TLSCertPath    string         `json:"tls_cert_path,omitempty"`
	RateLimitCount map[string]int `json:"rate_limit_count"`
	// TrustedProxies lists the CIDR ranges (or bare IP addresses) of reverse
	// proxies that are allowed to set client-IP headers such as
	// X-Forwarded-For and CF-Connecting-IP. Proxy headers are only trusted
	// when the immediate peer (the TCP connection's remote address) falls
	// within one of these ranges. When empty, proxy headers are never trusted
	// and the connection's remote address is always used.
	TrustedProxies []string `json:"trusted_proxies,omitempty"`
	// AdminToken guards the admin endpoints (e.g. resetting a rate limit for an
	// email address). When empty, those endpoints are disabled.
	AdminToken string `json:"admin_token,omitempty"`
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

type EmailCredentialAttributes struct {
	Email       string `json:"email"`
	EmailDomain string `json:"email_domain"`
}

type JWTConfig struct {
	IRMAServerURL  string                    `json:"irma_server_url"`
	PrivateKeyPath string                    `json:"private_key_path"`
	IssuerID       string                    `json:"issuer_id"`
	CredentialType string                    `json:"credential_type"`
	Credential     string                    `json:"full_credential"`
	Attributes     EmailCredentialAttributes `json:"attributes"`
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
	//
	// The private key is mandatory: every issuance request signs a JWT with
	// it, so a missing path makes the service non-functional. Reject an empty
	// path at startup rather than letting the service boot and 500 on the
	// first issuance request.
	if cfg.JWT.PrivateKeyPath == "" {
		return errors.New("PRIVATE_KEY_PATH is required")
	}
	// Fully parse the key at startup so an unreadable or non-RSA (e.g. ECDSA)
	// key file fails fast with a clear error instead of only surfacing when
	// the first issuance request is handled.
	if _, err := LoadRSAPrivateKey(cfg.JWT.PrivateKeyPath); err != nil {
		return err
	}
	if cfg.JWT.IssuerID == "" {
		return errors.New("ISSUER_ID is required")
	}

	// Fail fast on a malformed trusted-proxy list rather than silently
	// ignoring proxy headers at runtime.
	if _, err := ParseTrustedProxies(cfg.App.TrustedProxies); err != nil {
		return err
	}

	// Admin endpoints (optional). When a token is set it is the only credential
	// guarding the admin routes, which sit on the same public router as the SPA.
	// A short token is brute-forceable over the network, so reject a weak one at
	// startup rather than accepting it silently.
	if cfg.App.AdminToken != "" && len(cfg.App.AdminToken) < MinAdminTokenLength {
		return fmt.Errorf("admin_token must be at least %d characters when set", MinAdminTokenLength)
	}
	return nil
}

// ParseTrustedProxies converts the configured list of trusted-proxy entries
// into parsed networks. Each entry may be a CIDR range (e.g. "10.0.0.0/8") or
// a bare IP address (e.g. "192.0.2.1"), which is treated as a single-host
// range. Empty entries are skipped. It returns an error describing the first
// entry that cannot be parsed.
func ParseTrustedProxies(cidrs []string) ([]*net.IPNet, error) {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, entry := range cidrs {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		// Allow bare IP addresses by promoting them to a single-host CIDR.
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				if ip.To4() != nil {
					entry += "/32"
				} else {
					entry += "/128"
				}
			}
		}
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", entry, err)
		}
		nets = append(nets, network)
	}
	return nets, nil
}

// MinAdminTokenLength is the minimum length required for app.admin_token when
// the admin endpoints are enabled.
const MinAdminTokenLength = 16

// LoadRSAPrivateKey reads and parses the PEM-encoded RSA private key at path.
// It returns a descriptive error if the file cannot be read or does not
// contain a valid RSA private key (e.g. it is empty or holds an ECDSA key).
func LoadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("could not read private key file %q: %w", path, err)
	}

	key, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid RSA private key in %q: %w", path, err)
	}

	return key, nil
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
