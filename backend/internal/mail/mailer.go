package mail

import (
	"backend/internal/config"
	"fmt"

	gomail "gopkg.in/mail.v2"
)

type Email struct {
	From    string
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	SendEmail(e Email) error
}

type SmtpMailer struct {
	mcfg   *config.MailConfig
	dialer *gomail.Dialer
}

func NewSmtpMailer(mcfg *config.MailConfig) *SmtpMailer {
	dialer := gomail.NewDialer(mcfg.Host, mcfg.Port, mcfg.User, mcfg.Password)
	return &SmtpMailer{mcfg: mcfg, dialer: dialer}
}

func (sm SmtpMailer) SendEmail(e Email) error {

	gm := gomail.NewMessage()
	gm.SetHeader("From", e.From)
	gm.SetHeader("To", e.To)
	gm.SetHeader("Subject", e.Subject)
	gm.SetBody("text/html", e.Body)

	sm.dialer.DialAndSend(gm)

	return nil

}

type DummyMailer struct{}

func (dm DummyMailer) SendEmail(e Email) error {
	fmt.Printf("Sending email to %s with subject '%s' and body '%s'\n", e.To, e.Subject, e.Body)
	return nil
}
