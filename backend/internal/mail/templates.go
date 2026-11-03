package mail

import (
	"bytes"
	"net/url"
	"text/template"
)

// appLinkBase is the universal link the Yivi app claims. The code goes in the
// fragment, which browsers do not send to the server, so it stays out of logs.
const appLinkBase = "https://open.yivi.app/-/email"

func RenderHTMLtemplate(dir string, link string, token string) (string, error) {

	tmpl, err := template.ParseFiles(dir)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		"Link":    link,
		"AppLink": appLinkBase + "#code=" + url.QueryEscape(token),
		"Token":   token,
	})
	if err != nil {
		return "", err
	}

	return buf.String(), nil

}
