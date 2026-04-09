package httpapi

import (
	"backend/internal/issue"
	"backend/internal/mail"
	"backend/internal/validators"
	"fmt"
	"log"
	"net"
	"net/http"
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

	validator := validators.EmailValidator{}
	valid, parsedAddress, errCode := validator.ParseAndValidateEmailAddress(req.Email)
	if !valid {
		writeError(w, http.StatusBadRequest, *errCode)
		return
	}

	remove_err := a.tokenStorage.RemoveToken(*parsedAddress)
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
	// Validate and normalize the email address
	validator := validators.EmailValidator{}
	valid, parsedAddress, errCode := validator.ParseAndValidateEmailAddress(req.Email)
	if !valid {
		writeError(w, http.StatusBadRequest, *errCode)
		return
	}

	if a.tokenStorage == nil {
		http.Error(w, "token generator not configured", http.StatusInternalServerError)
		return
	}
	expectedToken, retrieve_err := a.tokenStorage.RetrieveToken(*parsedAddress)
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

	jwt, create_err := jwtCreator.CreateJwt(*parsedAddress)
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
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "email_required")
		return
	}

	// Validate email address format
	validator := validators.EmailValidator{}
	valid, parsedAddress, errCode := validator.ParseAndValidateEmailAddress(in.Email)
	if !valid {
		writeError(w, http.StatusBadRequest, *errCode)
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
	tok, err := a.tokenGenerator.GenerateToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error_generating_token")
		return
	}

	// Make sure we use the parsed email address from the validator, to ensure that we e.g. trim any whitespace and remove any full name from the RFC-5322 format
	err = a.tokenStorage.StoreToken(*parsedAddress, tok)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "error_storing_token")
		return
	}
	baseURL := strings.TrimSuffix(a.cfg.App.BaseURL, "/")
	verifyURL := fmt.Sprintf("%s/%s/enroll#verify:%s:%s", baseURL, in.Language, *parsedAddress, tok)

	tmplStr, err := mail.RenderHTMLtemplate(mailTmpl.TemplateDir, verifyURL, tok)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "template_render_error")
		return
	}

	// For sending the email, we can used the unparsed email address from the input, since the mailer will use the parsed email address from the validator as the "To" address. This allows us to e.g. keep the full name in the "To" field if the user provided an RFC-5322 formatted email address.
	emData := mail.Email{From: a.cfg.Mail.From, To: in.Email,
		Subject: mailTmpl.Subject,
		Body:    tmplStr,
	}

	// rate limit for sending emails
	if a.limiter != nil {
		ip := clientIP(r)
		allow, _ := a.limiter.Allow(ip, *parsedAddress)
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
