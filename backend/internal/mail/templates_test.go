package mail

import (
	"strings"
	"testing"
)

// The e-mail holds one link, as the "Open Yivi" button and spelled out below
// it. The code travels in its fragment, so it never reaches a server log: on a
// phone with Yivi the app fills it in, elsewhere open.yivi.app shows it. The
// code itself is not in the text, so there is only one thing to do with it.
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

			if got := strings.Count(body, `href="`+appLink+`"`); got != 2 {
				t.Errorf("expected the button and the spelled out link to %q, found %d links", appLink, got)
			}
			if !strings.Contains(body, ">"+appLink+"</a>") {
				t.Errorf("expected the link %q spelled out in the e-mail", appLink)
			}
			if strings.Contains(body, "<strong>"+code+"</strong>") {
				t.Errorf("expected no separate code %q in the e-mail", code)
			}
			if strings.Contains(body, webLink) {
				t.Errorf("expected no web link %q in the e-mail", webLink)
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
