package validators

import (
	"net/mail"
	"strings"
)

type EmailValidator struct{}

// ParseAndValidateEmailAddress checks whether the provided input is valid.
// It accepts RFC-5322 formatted emailaddresses (i.e. "John Doe <john.doe@example.com>"), but only in lowercase.
// It returns a boolean indicating validity, a parsed emailaddress (if valid) or an error message key (if invalid).
// If valid, the second return value contains the parsed address from the RFC-5322 format (e.g. "john.doe@example.com").
// If invalid, an error message is returned in the third return value.
func (v *EmailValidator) ParseAndValidateEmailAddress(email string) (bool, *string, *string) {
	if email == "" {
		err := "email_required"
		return false, nil, &err
	}

	// We don't accept quoted emailaddresses, even if something like `"John Doe" john.doe@example.com` should be valid
	// Neither do we accept IP-address domains like `john.doe@[192.168.1.1]`
	if strings.Contains(email, "\"") || strings.Contains(email, "[") {
		err := "error_email_format"
		return false, nil, &err
	}

	if addr, err := mail.ParseAddress(email); err != nil {
		err := "error_email_format"
		return false, nil, &err
	} else {
		// We only accept lowercase emailaddresses
		if strings.ToLower(addr.Address) != addr.Address {
			err := "error_email_format_lowercase"
			return false, nil, &err
		}
		return true, &addr.Address, nil
	}
}
