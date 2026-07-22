package httpapi

import (
	"net/http"
	"net/http/httptest"
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
