package mail

import (
	"bytes"
	"text/template"

	gomail "gopkg.in/mail.v2"
)

func RenderHTMLtemplate(dir string, link string) (string, error) {

	tmpl, err := template.ParseFiles(dir)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, link)
	if err != nil {
		return "", err
	}

	return buf.String(), nil

}

func ComposeEmail(e Email) *gomail.Message {
	gm := gomail.NewMessage()
	gm.SetHeader("From", e.From)
	gm.SetHeader("To", e.To)
	gm.SetHeader("Subject", e.Subject)
	gm.SetBody("text/html", e.Body)

	return gm

}
