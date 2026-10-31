package main

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestENR06InstallerProxyRequiresCurrentVerifiedIdentity(t *testing.T) {
	leaf := []byte("TEST ONLY unit leaf bytes")
	public := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201) })
	agent := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	marker := filepath.Join(t.TempDir(), "hold-usage")
	if err := os.WriteFile(marker, nil, 0600); err != nil {
		t.Fatal(err)
	}
	firstPoll := filepath.Join(t.TempDir(), "hold-first-poll")
	handler := installerProxyHandler(public, agent, func() (tls.Certificate, error) { return tls.Certificate{Certificate: [][]byte{leaf}}, nil }, marker, firstPoll)
	for _, tc := range []struct {
		name, method, path string
		verified           bool
		leaf               []byte
		want               int
	}{
		{"enrollment", "POST", "/api/v1/agent/enroll", false, nil, 201},
		{"anonymous runtime", "POST", "/api/v1/agent/usage-batches", false, nil, 403},
		{"enroll method", "GET", "/api/v1/agent/enroll", false, nil, 403},
		{"unverified cert", "GET", "/api/v1/agent/desired-state", false, leaf, 403},
		{"different cert", "GET", "/api/v1/agent/desired-state", true, []byte("retired unit leaf"), 403},
		{"current identity", "GET", "/api/v1/agent/desired-state", true, leaf, 204},
		{"held current usage", "POST", "/api/v1/agent/usage-batches", true, leaf, 503},
		{"held retired usage", "POST", "/api/v1/agent/usage-batches", true, []byte("retired unit leaf"), 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.leaf != nil {
				cert := &x509.Certificate{Raw: tc.leaf}
				r.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}
				if tc.verified {
					r.TLS.VerifiedChains = [][]*x509.Certificate{{cert}}
				}
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("got %d want %d", w.Code, tc.want)
			}
		})
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/api/v1/agent/usage-batches", nil)
	cert := &x509.Certificate{Raw: leaf}
	r.TLS = &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}, VerifiedChains: [][]*x509.Certificate{{cert}}}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("released usage: %d", w.Code)
	}
	if err := os.WriteFile(firstPoll, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/v1/agent/desired-state", 503},
		{"POST", "/api/v1/agent/netbird-bootstrap", 204},
		{"POST", "/api/v1/agent/usage-batches", 204},
	} {
		r.Method, r.URL.Path = tc.method, tc.path
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("first-poll %s: %d", tc.path, w.Code)
		}
	}
}
