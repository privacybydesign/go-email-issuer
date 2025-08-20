package mail

import (
	"bytes"
	"text/template"
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
