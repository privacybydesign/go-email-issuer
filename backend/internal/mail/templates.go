package mail

import (
	"fmt"
	"os"
	"path/filepath"
)

type VerifyEmailData struct {
	VerifyURL string
	Minutes   int
}

func RenderVerifyEmail(verifyURL string) (string, error) {
	path := filepath.Join("internal", "mail", "templates", "verify_email.html")
	htmlBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(string(htmlBytes), verifyURL), nil
}
