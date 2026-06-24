package config

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// baseConfig returns a Config that passes validation except for the JWT
// private key, which each test sets up itself.
func baseConfig(keyPath string) *Config {
	return &Config{
		Mail: MailConfig{
			Host: "smtp.example.com",
			Port: 587,
			From: "noreply@example.com",
		},
		JWT: JWTConfig{
			PrivateKeyPath: keyPath,
			IssuerID:       "email-issuer",
		},
	}
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

func validRSAKeyPEM(t *testing.T) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func ecdsaKeyPEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ECDSA key: %v", err)
	}
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal ECDSA key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}

func TestValidateValidRSAKey(t *testing.T) {
	path := writeTempFile(t, "priv.pem", validRSAKeyPEM(t))
	if err := validate(baseConfig(path)); err != nil {
		t.Fatalf("expected valid RSA key to pass validation, got: %v", err)
	}
}

func TestValidateEmptyKeyPath(t *testing.T) {
	err := validate(baseConfig(""))
	if err == nil {
		t.Fatal("expected validation to fail for empty private key path, got nil")
	}
	if !strings.Contains(err.Error(), "PRIVATE_KEY_PATH is required") {
		t.Fatalf("expected PRIVATE_KEY_PATH required error, got: %v", err)
	}
}

func TestValidateEmptyKeyFile(t *testing.T) {
	path := writeTempFile(t, "empty.pem", nil)
	err := validate(baseConfig(path))
	if err == nil {
		t.Fatal("expected validation to fail for empty key file, got nil")
	}
	if !strings.Contains(err.Error(), "invalid RSA private key") {
		t.Fatalf("expected descriptive RSA key error, got: %v", err)
	}
}

func TestValidateECDSAKey(t *testing.T) {
	path := writeTempFile(t, "ec.pem", ecdsaKeyPEM(t))
	err := validate(baseConfig(path))
	if err == nil {
		t.Fatal("expected validation to fail for ECDSA key, got nil")
	}
	if !strings.Contains(err.Error(), "invalid RSA private key") {
		t.Fatalf("expected descriptive RSA key error, got: %v", err)
	}
}

func TestValidateMissingKeyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.pem")
	err := validate(baseConfig(path))
	if err == nil {
		t.Fatal("expected validation to fail for missing key file, got nil")
	}
	if !strings.Contains(err.Error(), "could not read private key file") {
		t.Fatalf("expected descriptive read error, got: %v", err)
	}
}

func TestLoadRSAPrivateKeyValid(t *testing.T) {
	path := writeTempFile(t, "priv.pem", validRSAKeyPEM(t))
	key, err := LoadRSAPrivateKey(path)
	if err != nil {
		t.Fatalf("expected to load valid RSA key, got: %v", err)
	}
	if key == nil {
		t.Fatal("expected non-nil RSA key")
	}
}
