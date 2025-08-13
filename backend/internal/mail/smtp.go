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

func PrepareEmail(recipient string, templatePath string, link string, cfg *config.Config) (*gomail.Message, error) {

	// Parse the email template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, link); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Create a new message
	message := gomail.NewMessage()
	message.SetHeader("From", cfg.SMTP_FROM)
	message.SetHeader("To", recipient)
	message.SetHeader("Subject", cfg.EMAIL_SUBJECT)
	message.SetBody("text/html", buf.String())

	return message, nil
}

func SendEmail(message *gomail.Message, cfg *config.Config) error {

	// Set up the SMTP dialer
	dialer := gomail.NewDialer(cfg.SMTP_HOST, cfg.SMTP_PORT, cfg.SMTP_USER, cfg.SMTP_PASSWORD)

	// Send the email
	if err := dialer.DialAndSend(message); err != nil {
		fmt.Println("Error:", err)
		return err
	}
	fmt.Println("Email sent successfully to", message.GetHeader("To"))
	return nil
}
