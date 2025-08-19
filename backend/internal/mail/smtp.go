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
	Body    string
}

func PrepareEmail(recipient string, link string, cfg *config.MailConfig, lang string) (*gomail.Message, error) {

	// Parse the email template
	tmpl, err := template.ParseFiles(cfg.TemplateDir + "email_" + lang + ".html")
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	// Prepare the email subject based on the language
	var subject string
	if lang == "nl" {
		subject = cfg.Subject["nl"]
	} else {
		subject = cfg.Subject["en"]
	}

	// Execute the template with the link
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, link); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Create a new message
	message := gomail.NewMessage()
	message.SetHeader("From", cfg.From)
	message.SetHeader("To", recipient)
	message.SetHeader("Subject", subject)
	message.SetBody("text/html", buf.String())

	return message, nil
}

func SendEmail(message *gomail.Message, cfg *config.MailConfig) error {

	// Set up the SMTP dialer
	dialer := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password)

	// Send the email
	if err := dialer.DialAndSend(message); err != nil {
		fmt.Println("Error:", err)
		return err
	}
	fmt.Println("Email sent successfully")
	return nil
}
