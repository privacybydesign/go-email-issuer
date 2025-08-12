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

// Make Verification Link Payload:Timestamp
func MakeToken(payload string, now time.Time, secret []byte) (string, error) {
	if strings.Contains(payload, ":") {
		return "", errors.New("payload_must_not_contain_colon")
	}
	ts := strconv.FormatInt(now.Unix(), 10)
	msg := payload + ":" + ts

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) // ~43 chars

	return msg + ":" + sig, nil
}

// Parse verifies signature and age, returns payload and creation time.
func ParseToken(tok string, maxAge time.Duration, now time.Time, secret []byte) (payload string, created time.Time, err error) {

	parts := strings.Split(tok, ":")
	// our token is in 3 parts
	if len(parts) != 3 {
		return "", time.Time{}, ErrInvalid
	}
	payload, tsStr, sigStr := parts[0], parts[1], parts[2]

	// check expected signature vs result of new signature
	msg := payload + ":" + tsStr
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(msg))
	expected := mac.Sum(nil)

	result, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil || !hmac.Equal(result, expected) {
		return "", time.Time{}, ErrInvalid
	}

	// check if minttl has passed (minutes)
	sec, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", time.Time{}, ErrInvalid
	}
	created = time.Unix(sec, 0).UTC()
	if maxAge > 0 && now.Sub(created) > maxAge {
		return "", time.Time{}, ErrExpired
	}
	return payload, created, nil
}
