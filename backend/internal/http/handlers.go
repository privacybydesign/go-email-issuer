package httpapi

import (
	"backend/internal/issue"
	"backend/internal/mail"
	"fmt"
	"log"
	"net"
	"net/http"
	netmail "net/mail"
	"os"
	"path/filepath"
	"strings"
)

func (a *API) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("ok"))
	if err != nil {
		log.Printf("error: %s", err)
	}
}

type spaHandler struct {
	StaticPath string
	IndexPath  string
	FileServer http.Handler
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Join internally call path.Clean to prevent directory traversal
	path := filepath.Join(h.StaticPath, r.URL.Path)

	// check whether a file exists or is a directory at the given path
	fi, err := os.Stat(path)
	if os.IsNotExist(err) || fi.IsDir() {
		// file does not exist or path is a directory, serve index.html
		http.ServeFile(w, r, filepath.Join(h.StaticPath, h.IndexPath))
		return
	}

	if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static file
	h.FileServer.ServeHTTP(w, r)
}
func (a *API) handleVerifyDone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	decode_err := decodeJSON(w, r, &req)
	if decode_err != nil || req.Email == "" {
		writeError(w, http.StatusBadRequest, "email_required")
		return
	}

	remove_err := a.tokenStorage.RemoveToken(req.Email)
	if remove_err != nil {
		writeError(w, http.StatusInternalServerError, "error_removing_token")
		return
	}

}

func (a *API) handleVerifyEmail(w http.ResponseWriter, r *http.Request) {

	// token is passed in the body as JSON from the frontend
	var req struct {
		Token string `json:"token"`
		Email string `json:"email"`
	}
	decode_err := decodeJSON(w, r, &req)
	if decode_err != nil || req.Token == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "token_or_email_required")
		return
	}
	if a.tokenStorage == nil {
		http.Error(w, "token generator not configured", http.StatusInternalServerError)
		return
	}
	expectedToken, retrieve_err := a.tokenStorage.RetrieveToken(req.Email)
	if retrieve_err != nil {
		writeError(w, http.StatusBadRequest, "error_token_invalid")
		return
	}

	if expectedToken != req.Token {
		writeError(w, http.StatusBadRequest, "error_invalid_token")
		return
	}

	jwtCreator, creator_err := issue.NewIrmaJwtCreator(a.cfg.JWT)
	if creator_err != nil {
		writeError(w, http.StatusInternalServerError, "jwt_creator_error")
		return
	}

	jwt, create_err := jwtCreator.CreateJwt(req.Email)
	if create_err != nil {
		writeError(w, http.StatusInternalServerError, "jwt_creation_error")
		return
	}

	jserr := writeJSON(w, http.StatusOK, map[string]any{
		"jwt":             jwt,
		"irma_server_url": a.cfg.JWT.IRMAServerURL,
	})
	if jserr != nil {
		log.Printf("error: %s", jserr)
	}

}

func (a *API) handleSendEmail(w http.ResponseWriter, r *http.Request) {
	type input struct {
		Email    string `json:"email"`
		Language string `json:"language"`
	}

	var in input
	if err := decodeJSON(w, r, &in); err != nil || in.Email == "" {
		writeError(w, http.StatusBadRequest, "email_required")
		return
	}

	if _, err := netmail.ParseAddress(in.Email); err != nil {
		writeError(w, http.StatusBadRequest, "error_email_format")
		return
	}

	// render email template and prepare the email
	mailTmpl, ok := a.cfg.Mail.MailTemplates[in.Language]
	if !ok {
		mailTmpl = a.cfg.Mail.MailTemplates["en"]
	}

	if a.tokenGenerator == nil {
		http.Error(w, "token generator not configured", http.StatusInternalServerError)
		return
	}
	tok := a.tokenGenerator.GenerateToken()

	err := a.tokenStorage.StoreToken(in.Email, tok)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error_storing_token")
		return
	}
	baseURL := strings.TrimSuffix(a.cfg.App.BaseURL, "/")
	verifyURL := fmt.Sprintf("%s/%s/enroll#verify:%s:%s", baseURL, in.Language, in.Email, tok)

	tmplStr, err := mail.RenderHTMLtemplate(mailTmpl.TemplateDir, verifyURL, tok)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "template_render_error")
		return
	}

	emData := mail.Email{From: a.cfg.Mail.From, To: in.Email,
		Subject:    mailTmpl.Subject,
		Body:       tmplStr,
		SenderName: a.cfg.Mail.SenderName,
	}

	// rate limit for sending emails
	if a.limiter != nil {
		ip := clientIP(r)
		allow, _ := a.limiter.Allow(ip, in.Email)
		if !allow {
			writeError(w, http.StatusTooManyRequests, "error_ratelimit")
			return
		}
	}

	err = a.mailer.SendEmail(emData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error_sending_email")
		return
	}

	jserr := writeJSON(w, http.StatusOK, map[string]any{
		"message": "email_sent",
	})
	if jserr != nil {
		log.Printf("error: %s", jserr)
	}

}

func clientIP(r *http.Request) string {
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}
	if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
		return cf
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
