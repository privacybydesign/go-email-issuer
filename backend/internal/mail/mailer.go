package mail

import (
	"backend/internal/config"
	"bytes"
	"fmt"
	"text/template"

	gomail "gopkg.in/mail.v2"
)

type Email struct {
	From    string
	To      string
	Subject string
	Lang    string
	Link    string
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

	// set the subject with fallback
	subject, ok := sm.mcfg.Subject[e.Lang]
	if !ok {
		subject = sm.mcfg.Subject["en"]
	}
	// set body to te template with fallback
	tmpldir, ok := sm.mcfg.TemplateDir[e.Lang]
	if !ok {
		tmpldir = sm.mcfg.TemplateDir["en"]
	}
	// parse the template with the link
	tmpl, err := template.ParseFiles(tmpldir)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, e.Link); err != nil {
		return nil
	}

	// set up email message
	gm := gomail.NewMessage()
	gm.SetHeader("From", e.From)
	gm.SetHeader("To", e.To)
	gm.SetHeader("Subject", subject)
	gm.SetBody("text/html", buf.String())

	sm.dialer.DialAndSend(gm)

	return nil

}

type DummyMailer struct{}

func (dm DummyMailer) SendEmail(e Email) error {
	fmt.Printf("Sending email to %s with subject '%s' and link '%s'\n", e.To, e.Subject, e.Link)
	return nil
}
