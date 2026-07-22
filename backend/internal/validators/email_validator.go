package validators

import (
	"errors"
	"net"
	"net/mail"
	"strings"
)

// Resolver abstracts the DNS lookups used to verify that a domain is able to
// receive email. It is satisfied by the standard net package and can be
// stubbed in tests so they don't depend on real DNS.
type Resolver interface {
	LookupMX(name string) ([]*net.MX, error)
	LookupIP(host string) ([]net.IP, error)
}

type netResolver struct{}

func (netResolver) LookupMX(name string) ([]*net.MX, error) { return net.LookupMX(name) }
func (netResolver) LookupIP(host string) ([]net.IP, error)  { return net.LookupIP(host) }

type EmailValidator struct {
	// Resolver performs the DNS deliverability check. When nil, a real DNS
	// resolver is used; tests can inject a stub.
	Resolver Resolver
}

// ParseAndValidateEmailAddress checks whether the provided input is valid.
// It accepts RFC-5322 formatted emailaddresses (i.e. "John Doe <john.doe@example.com>").
// Any uppercase characters in the address are normalized to lowercase.
// Beyond the RFC-5322 grammar it also rejects addresses whose domain is
// clearly invalid (e.g. a malformed or single-character TLD such as
// "gmail.c") or whose domain cannot receive email because it has no MX or
// A/AAAA records (e.g. a typo like "gemail.com"). This prevents the user from
// silently never receiving the verification email.
// It returns a boolean indicating validity, a parsed emailaddress (if valid) or an error message key (if invalid).
// If valid, the second return value contains the parsed address from the RFC-5322 format (e.g. "john.doe@example.com").
// If invalid, an error message is returned in the third return value.
func (v *EmailValidator) ParseAndValidateEmailAddress(email string) (bool, *string, *string) {
	if email == "" {
		return invalid("email_required")
	}

	// We don't accept quoted emailaddresses, even if something like `"John Doe" john.doe@example.com` should be valid
	// Neither do we accept IP-address domains like `john.doe@[192.168.1.1]`
	if strings.Contains(email, "\"") || strings.Contains(email, "[") {
		return invalid("error_email_format")
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return invalid("error_email_format")
	}

	normalized := strings.ToLower(addr.Address)

	// After a successful parse the address always contains exactly one '@'.
	domain := normalized[strings.LastIndex(normalized, "@")+1:]
	if !isValidDomainSyntax(domain) {
		return invalid("error_email_format")
	}

	if !v.domainCanReceiveMail(domain) {
		return invalid("error_email_unknown_domain")
	}

	return true, &normalized, nil
}

func invalid(code string) (bool, *string, *string) {
	c := code
	return false, nil, &c
}

// isValidDomainSyntax performs a syntactic sanity check on the domain part of
// an email address. It requires at least one dot-separated label plus a TLD,
// valid characters per label, and a TLD of at least two characters that is not
// purely numeric. This rejects obvious typos such as "gmail.c" without needing
// a DNS lookup.
func isValidDomainSyntax(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		// A deliverable domain always has at least a second-level domain and a TLD.
		return false
	}

	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			if !isDomainLabelChar(label[i]) {
				return false
			}
		}
	}

	// The TLD must be at least two characters and must not be purely numeric.
	// (Hyphens/digits are allowed to keep punycode TLDs such as "xn--p1ai" valid.)
	tld := labels[len(labels)-1]
	if len(tld) < 2 {
		return false
	}
	allDigits := true
	for i := 0; i < len(tld); i++ {
		if tld[i] < '0' || tld[i] > '9' {
			allDigits = false
			break
		}
	}
	return !allDigits
}

// isDomainLabelChar reports whether c is allowed in a (lowercased) DNS label:
// an ASCII letter, digit or hyphen.
func isDomainLabelChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-'
}

// domainCanReceiveMail verifies that the domain has a mail exchanger, falling
// back to an A/AAAA record as allowed by RFC 5321. Transient DNS failures
// (timeouts, no network) fail open so legitimate users are never blocked; only
// a definitive "no such host"/no-records answer rejects the address.
func (v *EmailValidator) domainCanReceiveMail(domain string) bool {
	r := v.Resolver
	if r == nil {
		r = netResolver{}
	}

	mx, err := r.LookupMX(domain)
	if err != nil && !isNotFound(err) {
		return true // transient failure: fail open
	}
	if len(mx) > 0 {
		return true
	}

	// No MX record found; fall back to the A/AAAA record (RFC 5321 §5.1).
	ips, err := r.LookupIP(domain)
	if err != nil && !isNotFound(err) {
		return true // transient failure: fail open
	}
	return len(ips) > 0
}

// isNotFound reports whether the DNS error is a definitive "host not found"
// answer (NXDOMAIN / no records) as opposed to a transient failure.
func isNotFound(err error) bool {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.IsNotFound
	}
	return false
}
