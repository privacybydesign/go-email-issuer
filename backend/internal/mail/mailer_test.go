package mail

import (
	"testing"

	gomail "gopkg.in/mail.v2"
)

// SendEmail must surface the error from DialAndSend rather than swallowing it.
// Dialing a closed local port makes DialAndSend fail deterministically.
func TestSmtpMailerSendEmailReturnsDialError(t *testing.T) {
	sm := SmtpMailer{
		dialer: gomail.NewDialer("127.0.0.1", 1, "user", "pass"),
	}

	err := sm.SendEmail(Email{
		From:    "from@example.com",
		To:      "to@example.com",
		Subject: "subject",
		Body:    "<p>body</p>",
	})

	if err == nil {
		t.Fatal("expected SendEmail to return the DialAndSend error, got nil")
	}
}

// DummyMailer never dials anything, so it should always succeed.
func TestDummyMailerSendEmailReturnsNil(t *testing.T) {
	if err := (DummyMailer{}).SendEmail(Email{To: "to@example.com"}); err != nil {
		t.Fatalf("expected nil error from DummyMailer, got %v", err)
	}
}
