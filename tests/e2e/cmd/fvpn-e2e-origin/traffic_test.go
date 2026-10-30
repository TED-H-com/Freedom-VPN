package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSyntheticTrafficProbeRequiresAuthAndExactID(t *testing.T) {
	p := &trafficProbe{}
	mux := http.NewServeMux()
	mux.Handle("GET /", http.NotFoundHandler())
	p.register(mux, "TEST ONLY generated-at-runtime equivalent")
	id := uuid.NewString()
	call := func(method, secret, body string) int {
		req := httptest.NewRequest(method, "/e2e/traffic", strings.NewReader(body))
		req.Header.Set("X-E2E-Secret", secret)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w.Code
	}
	if call("PUT", "", `{"id":"`+id+`"}`) != 401 || call("GET", "", "") != 401 {
		t.Fatal("unauthenticated probe")
	}
	if call("PUT", "TEST ONLY generated-at-runtime equivalent", `{"id":"`+id+`"}`) != 204 {
		t.Fatal("arm")
	}
	req := httptest.NewRequest("GET", "/healthz", nil)
	req.RemoteAddr = "192.0.2.20:12345"
	p.observe(req)
	req.Header.Set("X-E2E-Probe", uuid.NewString())
	p.observe(req)
	if p.count != 0 {
		t.Fatal("unrelated requests recorded")
	}
	req.Header.Set("X-E2E-Probe", id)
	p.observe(req)
	if p.count != 1 || p.source != "192.0.2.20" {
		t.Fatal("synthetic arrival missing")
	}
}
