package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

var ErrInvalid = errors.New("invalid_or_tampered")
var ErrExpired = errors.New("expired")

// Make verification token email:timestamp:signature
func MakeToken(email string, hmac_key []byte, expiresAt int64) (string, error) {
	if strings.Contains(email, ":") {
		return "", errors.New("email_must_not_contain_colon")
	}
	msg := email + ":" + strconv.FormatInt(expiresAt, 10)

	mac := hmac.New(sha256.New, hmac_key)
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := strings.Join([]string{email, strconv.FormatInt(expiresAt, 10), sig}, ":")

	return token, nil
}

// Parse verifies signature and age, returns email and creation time.
func ParseToken(tok string, secret []byte) (email string, err error) {

	parts := strings.Split(tok, ":")
	// our token is in 3 parts
	if len(parts) != 3 {
		return "", ErrInvalid
	}
	email, expiresAtStr, sigStr := parts[0], parts[1], parts[2]

	// check expected signature vs result of new signature
	msg := email + ":" + expiresAtStr
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	expected := mac.Sum(nil)

	result, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil || !hmac.Equal(result, expected) {
		return "", ErrInvalid
	}

	// check if token is expired
	expiresAt, err := strconv.ParseInt(expiresAtStr, 10, 64)
	if err != nil {
		return "", ErrInvalid
	}

	if expiresAt < time.Now().Unix() {
		return "", ErrExpired
	}
	return email, nil
}
