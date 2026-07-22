package validators

import (
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeResolver is a deterministic stub for the DNS deliverability check so the
// tests don't depend on real DNS. Domains present in mx/ips resolve; any other
// domain returns a definitive "not found" answer. forceTemporary makes every
// lookup return a transient error to exercise the fail-open path.
type fakeResolver struct {
	mx             map[string][]*net.MX
	ips            map[string][]net.IP
	forceTemporary bool
}

func (f fakeResolver) LookupMX(name string) ([]*net.MX, error) {
	if f.forceTemporary {
		return nil, &net.DNSError{Err: "server misbehaving", Name: name, IsTemporary: true}
	}
	if v, ok := f.mx[name]; ok {
		return v, nil
	}
	return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
}

func (f fakeResolver) LookupIP(host string) ([]net.IP, error) {
	if f.forceTemporary {
		return nil, &net.DNSError{Err: "server misbehaving", Name: host, IsTemporary: true}
	}
	if v, ok := f.ips[host]; ok {
		return v, nil
	}
	return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
}

// newTestValidator returns a validator whose DNS check treats example.com and
// gmail.com as deliverable domains and everything else as non-existent.
func newTestValidator() EmailValidator {
	return EmailValidator{
		Resolver: fakeResolver{
			mx: map[string][]*net.MX{
				"example.com": {{Host: "mail.example.com", Pref: 10}},
				"gmail.com":   {{Host: "gmail-smtp-in.l.google.com", Pref: 5}},
			},
		},
	}
}

func Test_ParseAndValidateEmailAddress_Given_ValidRfc5322Address_Should_ReturnPlainEmailAddress(t *testing.T) {
	ev := newTestValidator()

	testCases := []string{
		"John Doe <john.doe@example.com>", // full name with angle brackets
		"   john.doe@example.com",         // leading whitespaces
		"\t\tjohn.doe@example.com",        // leading tabs
		"john.doe@example.com   ",         // trailing whitespaces
		"john.doe@example.com\t",          // trailing tabs
	}

	for _, tc := range testCases {
		valid, parsedAddress, err := ev.ParseAndValidateEmailAddress(tc)

		require.True(t, valid)
		require.Nil(t, err)
		require.Equal(t, "john.doe@example.com", *parsedAddress)
	}
}

func Test_ParseAndValidateEmailAddress_Given_AddressWithTag_Should_ReturnPlainEmailAddress(t *testing.T) {
	ev := newTestValidator()

	testCases := []string{
		"John Doe <john.doe+tag@example.com>", // full address with plus tag
		"john.doe+tag@example.com",            // with plus tag
	}

	for _, tc := range testCases {
		valid, parsedAddress, err := ev.ParseAndValidateEmailAddress(tc)

		require.True(t, valid)
		require.Nil(t, err)
		require.Equal(t, "john.doe+tag@example.com", *parsedAddress)
	}
}

func Test_ParseAndValidateEmailAddress_Given_EmailAddressesWithQuotesOrIpDomains_Should_ReturnError(t *testing.T) {
	ev := newTestValidator()

	testCases := []string{
		"\"John Doe\" john.doe@example.com", // valid RFC-5322 format, but we don't accept any quotes
		"\"john.doe\"@example.com",          // quoted
		"\"john@doe\"@example.com",          // quoted
		"john.doe@[192.168.1.1]",            // with IP address domain
	}

	for _, tc := range testCases {
		valid, parsedAddress, err := ev.ParseAndValidateEmailAddress(tc)

		require.False(t, valid)
		require.Nil(t, parsedAddress)
		require.Equal(t, "error_email_format", *err)
	}
}

func Test_ParseAndValidateEmailAddress_Given_AddressWithUppercase_Should_NormalizeToLowercase(t *testing.T) {
	ev := newTestValidator()

	testCases := []struct {
		input    string
		expected string
	}{
		{"John Doe <John.Doe@Example.com>", "john.doe@example.com"},
		{"   John.Doe@Example.com", "john.doe@example.com"},
		{"\t\tJohn.Doe@Example.com", "john.doe@example.com"},
		{"John.Doe@Example.com   ", "john.doe@example.com"},
		{"John.Doe@Example.com\t", "john.doe@example.com"},
		{"John.Doe+tag@Example.com", "john.doe+tag@example.com"},
	}

	for _, tc := range testCases {
		valid, parsedAddress, err := ev.ParseAndValidateEmailAddress(tc.input)

		require.True(t, valid)
		require.Nil(t, err)
		require.Equal(t, tc.expected, *parsedAddress)
	}
}

func Test_ParseAndValidateEmailAddress_Given_EmptyAddress_Should_ReturnEmailRequired(t *testing.T) {
	ev := newTestValidator()

	valid, parsedAddress, err := ev.ParseAndValidateEmailAddress("")

	require.False(t, valid)
	require.Nil(t, parsedAddress)
	require.Equal(t, "email_required", *err)
}

// Addresses that are grammatically well-formed but whose domain is clearly
// malformed (e.g. a single-character or numeric TLD) must be rejected up front
// as a format error, without needing a DNS lookup.
func Test_ParseAndValidateEmailAddress_Given_MalformedDomain_Should_ReturnFormatError(t *testing.T) {
	ev := newTestValidator()

	testCases := []string{
		"janjansen@gmail.c",    // single-character TLD (the issue example)
		"janjansen@gmail",      // no TLD at all
		"janjansen@gmail.",     // empty TLD
		"janjansen@.com",       // empty second-level label
		"janjansen@gmail.123",  // purely numeric TLD
		"janjansen@-gmail.com", // label starting with a hyphen
	}

	for _, tc := range testCases {
		valid, parsedAddress, err := ev.ParseAndValidateEmailAddress(tc)

		require.Falsef(t, valid, "expected %q to be rejected", tc)
		require.Nil(t, parsedAddress)
		require.Equalf(t, "error_email_format", *err, "for input %q", tc)
	}
}

// Addresses whose domain is syntactically fine but does not exist (no MX and no
// A/AAAA record) must be rejected with a dedicated error so the user can
// correct the typo instead of silently never receiving the verification email.
func Test_ParseAndValidateEmailAddress_Given_NonExistentDomain_Should_ReturnUnknownDomainError(t *testing.T) {
	ev := newTestValidator()

	testCases := []string{
		"janjansen@gemail.com",          // the issue example: a plausible typo of gmail.com
		"janjansen@nonexistent.example", // domain not present in the resolver
	}

	for _, tc := range testCases {
		valid, parsedAddress, err := ev.ParseAndValidateEmailAddress(tc)

		require.Falsef(t, valid, "expected %q to be rejected", tc)
		require.Nil(t, parsedAddress)
		require.Equalf(t, "error_email_unknown_domain", *err, "for input %q", tc)
	}
}

// A domain without an MX record but with an A/AAAA record is still able to
// receive mail (RFC 5321 fallback) and must be accepted.
func Test_ParseAndValidateEmailAddress_Given_DomainWithOnlyARecord_Should_BeValid(t *testing.T) {
	ev := EmailValidator{
		Resolver: fakeResolver{
			ips: map[string][]net.IP{
				"a-only.example": {net.ParseIP("203.0.113.10")},
			},
		},
	}

	valid, parsedAddress, err := ev.ParseAndValidateEmailAddress("john.doe@a-only.example")

	require.True(t, valid)
	require.Nil(t, err)
	require.Equal(t, "john.doe@a-only.example", *parsedAddress)
}

// Transient DNS failures (timeouts, no network) must not block legitimate
// users: the deliverability check fails open.
func Test_ParseAndValidateEmailAddress_Given_TransientDnsError_Should_FailOpen(t *testing.T) {
	ev := EmailValidator{Resolver: fakeResolver{forceTemporary: true}}

	valid, parsedAddress, err := ev.ParseAndValidateEmailAddress("john.doe@example.com")

	require.True(t, valid)
	require.Nil(t, err)
	require.Equal(t, "john.doe@example.com", *parsedAddress)
}

func Test_isNotFound(t *testing.T) {
	require.True(t, isNotFound(&net.DNSError{IsNotFound: true}))
	require.False(t, isNotFound(&net.DNSError{IsTemporary: true}))
	require.False(t, isNotFound(errors.New("some other error")))
	require.False(t, isNotFound(nil))
}
