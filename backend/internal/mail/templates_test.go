package mail

import (
	"strings"
	"testing"
)

// The verification code travels in the fragment of the universal link, so it
// never reaches a server log. The plain code stays in the e-mail for clients
// that strip links.
func TestRenderHTMLtemplateAppLink(t *testing.T) {
	const (
		code    = "A1B2C3"
		webLink = "https://example.com/en/enroll#token:opaque"
		appLink = "https://open.yivi.app/-/email#code=A1B2C3"
	)

	for _, dir := range []string{"templates/email_en.html", "templates/email_nl.html"} {
		t.Run(dir, func(t *testing.T) {
			body, err := RenderHTMLtemplate(dir, webLink, code)
			if err != nil {
				t.Fatalf("render failed: %v", err)
			}

			if !strings.Contains(body, `href="`+appLink+`"`) {
				t.Errorf("expected an href to %q in the e-mail", appLink)
			}
			if !strings.Contains(body, "<strong>"+code+"</strong>") {
				t.Errorf("expected the plain code %q in the e-mail", code)
			}
			if !strings.Contains(body, `href="`+webLink+`"`) {
				t.Errorf("expected the existing web link %q to stay in the e-mail", webLink)
			}
		})
	}
}

func TestRenderHTMLtemplateAppLinkEscapesCode(t *testing.T) {
	body, err := RenderHTMLtemplate("templates/email_en.html", "https://example.com", `a&b"c`)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if !strings.Contains(body, `href="https://open.yivi.app/-/email#code=a%26b%22c"`) {
		t.Errorf("expected the code to be escaped in the app link, got:\n%s", body)
	}
}
