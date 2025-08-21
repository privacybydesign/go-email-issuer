package main

import (
	"backend/internal/issue"
	"testing"
)

func TestCreatingJwt(t *testing.T) {
	testKeyPath := "./keys/priv.pem"
	issuerId := "email_issuer"
	credential := "irma-demo.sidn-pbdf.email"
	attribute := "email"

	jwtCreator, err := issue.NewIrmaJwtCreator(testKeyPath, issuerId, credential, attribute)
	if err != nil {
		t.Fatalf("Failed to instantiate jwt creator: %v", err)
	}

	testEmail := "test@email.com"
	jwt, err := jwtCreator.CreateJwt(testEmail)

	if err != nil {
		t.Fatalf("Failed to create the jwt for the given email: %v", err)
	}

	if jwt == "" {
		t.Fatal("jwt is empty")
	}
}
