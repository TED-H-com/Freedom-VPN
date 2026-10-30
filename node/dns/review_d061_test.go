package dns

import (
	"context"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestReviewD061_AdminClientRefusesOffOriginRedirect(t *testing.T) {
	// TODO(review): newHTTPClient leaves CheckRedirect nil, so a host-local
	// admin URL can 307 the DNS write to another origin. Load-time checks
	// only see the configured URL. SIGHUP rebuilds this same client.
	// 127.0.0.2 stands in for a remote host so the proof stays on loopback.
	t.Run("different host", func(t *testing.T) {
		redirectFollows(t, "127.0.0.2:0")
	})
	t.Run("same host other port", func(t *testing.T) {
		redirectFollows(t, "127.0.0.1:0")
	})
}

func redirectFollows(t *testing.T, remoteAddr string) {
	t.Helper()
	var hits atomic.Int32
	var authHeader, body atomic.Value
	remote := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		authHeader.Store(r.Header.Get("Authorization"))
		raw, _ := io.ReadAll(r.Body)
		body.Store(string(raw))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	ln, err := net.Listen("tcp", remoteAddr)
	if err != nil {
		t.Fatal(err)
	}
	remote.Listener = ln
	remote.Start()
	t.Cleanup(remote.Close)
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, remote.URL+r.URL.RequestURI(), http.StatusTemporaryRedirect)
	}))
	t.Cleanup(local.Close)
	applier, err := New([]Target{{
		ID:       "lab",
		AdminURL: local.URL,
		Username: "e2e",
		Password: "pw-d061-redirect",
		DNSAddr:  "127.0.0.1:53",
	}})
	if err != nil {
		t.Fatal(err)
	}
	const secret = "pw-d061-redirect"
	const marker = "review-d061-rule"
	err = applier.call(context.Background(), applier.targets[0], http.MethodPost, "/control/filtering/set_rules", map[string]any{"rules": []string{marker}}, nil)
	auth, _ := authHeader.Load().(string)
	sent, _ := body.Load().(string)
	if hits.Load() != 0 || err == nil {
		t.Fatalf("TODO(review): resolver admin client followed a 307 off the configured origin: hits=%d err=%v authorization=%q credential_forwarded=%t body=%q",
			hits.Load(), err, auth, basicPassword(auth) == secret, sent)
	}
}

func basicPassword(header string) string {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil || !strings.HasPrefix(header, "Basic ") {
		return ""
	}
	_, pass, _ := strings.Cut(string(raw), ":")
	return pass
}
