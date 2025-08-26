package main

import (
	"backend/internal/issue"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreatingJwt(t *testing.T) {
	jwtCreator, err := issue.NewIrmaJwtCreator(testCfg.JWT)
	if err != nil {
		t.Fatalf("Failed to instantiate jwt creator: %v", err)
	}

	testEmail := "test@email.com"
	jwt, err := jwtCreator.CreateJwt(testEmail)

	require.NoError(t, err, "Failed to create the jwt for the given email")
	require.NotEmpty(t, jwt, "jwt should not be empty")

}
