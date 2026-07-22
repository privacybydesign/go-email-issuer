package core

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateLinkTokenIsOpaqueAndUnique(t *testing.T) {
	a, err := GenerateLinkToken()
	require.NoError(t, err)
	b, err := GenerateLinkToken()
	require.NoError(t, err)

	require.NotEmpty(t, a)
	require.NotEqual(t, a, b, "link tokens must be unique")

	// The token must be URL-safe (base64url, no padding) so it can be carried
	// verbatim in a URL fragment without escaping.
	require.NotContains(t, a, "+")
	require.NotContains(t, a, "/")
	require.NotContains(t, a, "=")

	// It must not look like an email address: the whole point is that no email
	// is ever embedded in the link.
	require.NotContains(t, a, "@")
	require.GreaterOrEqual(t, len(a), 32)
}

func TestInMemoryLinkTokenRoundTrip(t *testing.T) {
	s := NewInMemoryTokenStorage()
	const linkToken = "opaque-link-token"
	const email = "user@example.com"

	// Unknown token must error.
	_, err := s.RetrieveEmailByLinkToken(linkToken)
	require.Error(t, err)

	// Store then retrieve.
	require.NoError(t, s.StoreLinkToken(linkToken, email))
	got, err := s.RetrieveEmailByLinkToken(linkToken)
	require.NoError(t, err)
	require.Equal(t, email, got)

	// Remove then it must be gone.
	require.NoError(t, s.RemoveLinkToken(linkToken))
	_, err = s.RetrieveEmailByLinkToken(linkToken)
	require.Error(t, err)

	// Removing a non-existent token is an error.
	require.Error(t, s.RemoveLinkToken("does-not-exist"))
}

func TestInMemoryLinkTokenIndependentFromCode(t *testing.T) {
	// The link-token map and the email->code map must not interfere.
	s := NewInMemoryTokenStorage()
	require.NoError(t, s.StoreToken("user@example.com", "ABC123"))
	require.NoError(t, s.StoreLinkToken("link", "user@example.com"))

	code, err := s.RetrieveToken("user@example.com")
	require.NoError(t, err)
	require.Equal(t, "ABC123", code)

	email, err := s.RetrieveEmailByLinkToken("link")
	require.NoError(t, err)
	require.Equal(t, "user@example.com", email)

	// A link token is not a valid email key and vice versa.
	_, err = s.RetrieveToken("link")
	require.Error(t, err)
	_, err = s.RetrieveEmailByLinkToken("user@example.com")
	require.Error(t, err)

	require.False(t, strings.Contains("link", "@"))
}
