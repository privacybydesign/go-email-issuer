package validators

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParseAndValidateEmailAddress_Given_ValidRfc5322Address_Should_ReturnPlainEmailAddress(t *testing.T) {
	ev := EmailValidator{}

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
	ev := EmailValidator{}

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
	ev := EmailValidator{}

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

func Test_ParseAndValidateEmailAddress_Given_AddressWithUppercase_Should_ReturnError(t *testing.T) {
	ev := EmailValidator{}

	testCases := []string{
		"John Doe <John.Doe@Example.com>", // full name with angle brackets
		"   John.Doe@Example.com",         // leading whitespaces
		"\t\tJohn.Doe@Example.com",        // leading tabs
		"John.Doe@Example.com   ",         // trailing whitespaces
		"John.Doe@Example.com\t",          // trailing tabs
		"John.Doe+tag@Example.com",        // with plus tag
	}

	for _, tc := range testCases {
		valid, parsedAddress, err := ev.ParseAndValidateEmailAddress(tc)

		require.False(t, valid)
		require.Nil(t, parsedAddress)
		require.Equal(t, "error_email_format_lowercase", *err)
	}
}
