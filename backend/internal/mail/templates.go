package mail

import (
	"bytes"
	"text/template"
)

func RenderHTMLtemplate(dir string, link string, token string) (string, error) {

	tmpl, err := template.ParseFiles(dir)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Link":  link,
		"Token": token,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil

}
