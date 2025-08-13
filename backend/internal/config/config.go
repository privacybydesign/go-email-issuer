package config

import (
	"os"
	"strconv"
)

type Config struct {
	Addr          string
	BASE_URL      string
	SECRET        string
	TTLMIN        string
	SMTP_HOST     string
	SMTP_USER     string
	SMTP_PASSWORD string
	SMTP_PORT     int
	SMTP_FROM     string
	EMAIL_SENDER  string
	EMAIL_SUBJECT string
	TEMPLATE_PATH string
}

// func to load env vars, accompanied by defaults

func Load() Config {
	return Config{
		Addr:          getenv("ADDR", ":8080"),
		BASE_URL:      getenv("BASE_URL", "localhost"),
		SECRET:        getenv("SECRET", "changeme"),
		TTLMIN:        getenv("TTLMIN", "15"),
		SMTP_HOST:     getenv("SMTP_HOST", "your.smtp.host"),
		SMTP_USER:     getenv("SMTP_USER", "user"),
		SMTP_FROM:     getenv("SMTP_FROM", "noreply@staging.yivi.app"),
		SMTP_PASSWORD: getenv("SMTP_PASSWORD", "password"),
		SMTP_PORT:     getenvInt("SMTP_PORT", 2587),
		EMAIL_SENDER:  getenv("EMAIL_SENDER", "Yivi Staging <noreply@staging.yivi.app>"),
		EMAIL_SUBJECT: getenv("EMAIL_SUBJECT", "Email Verification"),
		TEMPLATE_PATH: getenv("TEMPLATE_PATH", "./internal/mail/templates/verify_email.html"),
	}
}

func getenv(key, def string) string {

	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
