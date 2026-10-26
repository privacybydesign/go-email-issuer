package httpapi

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"backend/internal/config"
)

// newTestAPI builds an API whose trusted-proxy list is parsed from the given
// CIDR entries. Only the fields needed by clientIP are populated.
func newTestAPI(t *testing.T, trusted ...string) *API {
	t.Helper()
	nets, err := config.ParseTrustedProxies(trusted)
	if err != nil {
		t.Fatalf("failed to parse trusted proxies %v: %v", trusted, err)
	}
	return &API{trustedProxies: nets}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		trusted    []string
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			// A spoofed X-Forwarded-For from a peer that is not a trusted proxy
			// must be ignored, so the real connection address is used.
			name:       "spoofed XFF from untrusted peer is ignored",
			trusted:    nil,
			remoteAddr: "203.0.113.5:44444",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:       "203.0.113.5",
		},
		{
			// Same, but with a configured trusted-proxy list that does not
			// include the peer.
			name:       "spoofed XFF from peer outside trusted range is ignored",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "203.0.113.5:44444",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:       "203.0.113.5",
		},
		{
			// A spoofed CF-Connecting-IP from an untrusted peer is ignored too.
			name:       "spoofed CF-Connecting-IP from untrusted peer is ignored",
			trusted:    nil,
			remoteAddr: "203.0.113.5:44444",
			headers:    map[string]string{"CF-Connecting-IP": "1.2.3.4"},
			want:       "203.0.113.5",
		},
		{
			// When the peer is a trusted proxy, X-Forwarded-For is honoured.
			name:       "trusted proxy XFF is honoured",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:55555",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:       "1.2.3.4",
		},
		{
			// A bare-IP trusted-proxy entry is honoured.
			name:       "bare IP trusted proxy XFF is honoured",
			trusted:    []string{"10.1.2.3"},
			remoteAddr: "10.1.2.3:55555",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			want:       "1.2.3.4",
		},
		{
			// When the peer is trusted, CF-Connecting-IP is honoured.
			name:       "trusted proxy CF-Connecting-IP is honoured",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:55555",
			headers:    map[string]string{"CF-Connecting-IP": "1.2.3.4"},
			want:       "1.2.3.4",
		},
		{
			// Through a chain of trusted proxies, the rightmost non-trusted
			// address is the real client — earlier (client-injected) entries
			// are discarded.
			name:       "rightmost non-trusted address wins in a proxy chain",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.0.0.1:55555",
			headers:    map[string]string{"X-Forwarded-For": "9.9.9.9, 1.2.3.4, 10.0.0.2"},
			want:       "1.2.3.4",
		},
		{
			// Trusted peer but no proxy headers: use the connection address.
			name:       "trusted proxy without headers falls back to peer",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:55555",
			headers:    nil,
			want:       "10.1.2.3",
		},
		{
			// Trusted peer with a malformed XFF value falls back to the peer.
			name:       "trusted proxy with malformed XFF falls back to peer",
			trusted:    []string{"10.0.0.0/8"},
			remoteAddr: "10.1.2.3:55555",
			headers:    map[string]string{"X-Forwarded-For": "not-an-ip"},
			want:       "10.1.2.3",
		},
		{
			// RemoteAddr without a port is handled gracefully.
			name:       "remote addr without port",
			trusted:    nil,
			remoteAddr: "203.0.113.5",
			headers:    nil,
			want:       "203.0.113.5",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestAPI(t, tc.trusted...)
			r := httptest.NewRequest(http.MethodPost, "/api/send", nil)
			r.RemoteAddr = tc.remoteAddr
			for k, v := range tc.headers {
				r.Header.Set(k, v)
			}
			if got := a.clientIP(r); got != tc.want {
				t.Errorf("clientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSpaHandler(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("index"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("app"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := spaHandler{StaticPath: dir, IndexPath: "index.html", FileServer: http.FileServer(http.Dir(dir))}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "existing file is served", path: "/app.js", wantStatus: http.StatusOK, wantBody: "app"},
		{name: "unknown path falls back to index", path: "/en/enroll", wantStatus: http.StatusOK, wantBody: "index"},
		{name: "directory falls back to index", path: "/", wantStatus: http.StatusOK, wantBody: "index"},
		{
			// os.Stat fails with ENOTDIR here and returns a nil FileInfo, which
			// used to panic on fi.IsDir().
			name:       "path under a file does not panic",
			path:       "/app.js/x",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tc.wantStatus)
			}
			if tc.wantBody != "" && w.Body.String() != tc.wantBody {
				t.Errorf("body = %q, want %q", w.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestNewAPILogsTrustedProxies(t *testing.T) {
	tests := []struct {
		name    string
		trusted []string
		want    string
	}{
		{name: "empty list warns", trusted: nil, want: "warning: trusted_proxies is empty"},
		{name: "configured list is logged", trusted: []string{"100.64.0.0/15", "127.0.0.1"}, want: "Trusting proxy headers from 100.64.0.0/15, 127.0.0.1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			prev := log.Writer()
			log.SetOutput(&buf)
			t.Cleanup(func() { log.SetOutput(prev) })

			cfg := &config.Config{}
			cfg.App.TrustedProxies = tc.trusted
			NewAPI(cfg, nil, nil, nil, nil)

			if !strings.Contains(buf.String(), tc.want) {
				t.Errorf("log = %q, want it to contain %q", buf.String(), tc.want)
			}
		})
	}
}
