package main

import (
	"backend/internal/core"
	"testing"
	"time"
)

// test to create token

var testemail = "test@email.com"
var testsecret = []byte("TEST_SECRET")

func TestMakeToken(t *testing.T) {
	token, err := core.MakeToken(testemail, testsecret, time.Now().Add(time.Hour).Unix())
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
}

func TestParseTokenHappyPath(t *testing.T) {
	expiresAt := time.Now().Add(time.Hour).Unix()
	token, err := core.MakeToken(testemail, testsecret, expiresAt)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	email, err := core.ParseToken(token, testsecret)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if email != testemail {
		t.Fatalf("expected email %s, got %s", testemail, email)
	}
}

func TestParseMalformedToken(t *testing.T) {
	_, err := core.ParseToken("bad:token:struct", testsecret)

	if err == nil {
		t.Fatal("expected to throw error for malformed token")
	}

}

func TestParseExpiredToken(t *testing.T) {
	token, _ := core.MakeToken(testemail, testsecret, time.Now().Add(-time.Hour).Unix())
	email, err := core.ParseToken(token, testsecret)
	if err == nil {
		t.Fatalf("expected error for expired token, got email %s", email)
	}
}
