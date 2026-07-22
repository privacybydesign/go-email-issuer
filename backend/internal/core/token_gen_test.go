package core

import (
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

func countDigits(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

func isAllowedChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func TestGenerateToken_Properties(t *testing.T) {
	tg := NewRandomTokenGenerator()

	const iterations = 1000
	for range iterations {
		token, err := tg.GenerateToken()
		require.NoError(t, err)

		// Check each code is 6 characters
		if len(token) != 6 {
			t.Fatalf("expected length 6, got %d: %q", len(token), token)
		}

		// Check if they're all part of the allowed charset
		for _, r := range token {
			if !isAllowedChar(r) {
				t.Fatalf("token contains invalid character %q in %q", string(r), token)
			}
			// Ensure uppercase only
			if unicode.IsLetter(r) && !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZ", r) {
				t.Fatalf("token contains non-uppercase letter %q in %q", string(r), token)
			}
		}

		// Assert at least 2 digits
		d := countDigits(token)
		if d < 2 {
			t.Fatalf("expected at least 2 digits, got %d in %q", d, token)
		}
	}
}

func TestGenerateToken_FillUsesFullCharset(t *testing.T) {
	// Regression test for the fill-loop entropy bug: the non-digit fill
	// positions must draw from the full A-Z charset, not just A-J.
	// Previously the bound was len(digits) (10), so charset[0..9] == "ABCDEFGHIJ"
	// and the 16 letters K-Z were unreachable.
	//
	// Across many generations every uppercase letter (including K-Z) should
	// appear. With ~2.5 fill positions per token and each letter having a
	// ~1/36 chance per fill position, the expected count per letter over this
	// many iterations is in the hundreds, so missing any letter is
	// astronomically unlikely unless the charset is restricted again.
	tg := NewRandomTokenGenerator()

	const iterations = 10000
	seenLetters := make(map[rune]struct{})
	for range iterations {
		token, err := tg.GenerateToken()
		require.NoError(t, err)
		for _, r := range token {
			if r >= 'A' && r <= 'Z' {
				seenLetters[r] = struct{}{}
			}
		}
	}

	// The crux of the regression: at least one K-Z letter must show up.
	sawBackHalf := false
	for r := 'K'; r <= 'Z'; r++ {
		if _, ok := seenLetters[r]; ok {
			sawBackHalf = true
			break
		}
	}
	if !sawBackHalf {
		t.Fatalf("no letters from K-Z appeared in %d tokens; fill loop is restricted to a sub-range of the charset", iterations)
	}

	// Stronger guard: every uppercase letter A-Z should be reachable.
	for r := 'A'; r <= 'Z'; r++ {
		if _, ok := seenLetters[r]; !ok {
			t.Fatalf("letter %q never appeared in %d tokens; charset is not fully reachable (seen %d of 26 letters)", string(r), iterations, len(seenLetters))
		}
	}
}

func TestGenerateToken_BasicUniquenessSanity(t *testing.T) {
	// This is a sanity check, NOT a cryptographic test.
	// It can theoretically fail by chance, but with these params it should be extremely unlikely.
	tg := NewRandomTokenGenerator()

	const n = 500
	seen := make(map[string]struct{}, n)

	for range n {
		token, err := tg.GenerateToken()
		require.NoError(t, err)
		seen[token] = struct{}{}
	}

	// If randomness is totally broken, you'd see a tiny number of uniques.
	// With a proper generator, you should get almost all unique values here.
	if len(seen) < n-5 {
		t.Fatalf("expected near-unique tokens; got %d unique out of %d", len(seen), n)
	}
}
