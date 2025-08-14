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
func MakeToken(email string, now time.Time, secret []byte) (string, error) {
	if strings.Contains(email, ":") {
		return "", errors.New("email_must_not_contain_colon")
	}
	ts := strconv.FormatInt(now.Unix(), 10)
	msg := email + ":" + ts

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) // ~43 chars
	token := strings.Join([]string{email, ts, sig}, ":")

	return token, nil
}

// Parse verifies signature and age, returns email and creation time.
func ParseToken(tok string, maxAge time.Duration, secret []byte) (email string, created time.Time, err error) {

	parts := strings.Split(tok, ":")
	// our token is in 3 parts
	if len(parts) != 3 {
		return "", time.Time{}, ErrInvalid
	}
	email, tsStr, sigStr := parts[0], parts[1], parts[2]

	// check expected signature vs result of new signature
	msg := email + ":" + tsStr
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	expected := mac.Sum(nil)

	result, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil || !hmac.Equal(result, expected) {
		return "", time.Time{}, ErrInvalid
	}

	// check if ttl has passed (minutes)
	sec, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", time.Time{}, ErrInvalid
	}
	created = time.Unix(sec, 0).UTC()
	if maxAge > 0 && time.Since(created) > maxAge {
		return "", time.Time{}, ErrExpired
	}
	return email, created, nil
}
