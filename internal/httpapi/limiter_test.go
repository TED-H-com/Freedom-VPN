package httpapi

import (
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAT28_IngressCannotEvictOrExhaustPrincipalBuckets(t *testing.T) {
	c := config.Defaults()
	c.MaxRatePrincipals = 3
	c.IngressRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	c.BrowserRate = config.Bucket{PerSecond: .0001, Burst: 1}
	p, other := principal(auth.Admin), principal(auth.Admin)
	sets, _ := Compose()
	h, e := New(sets, Options{Config: c, Sessions: auth.SessionAuthenticatorFunc(func(r *http.Request) (auth.Principal, error) {
		if r.Header.Get("Test-Other") == "true" {
			return other, nil
		}
		return p, nil
	})})
	if e != nil {
		t.Fatal(e)
	}
	invoke := func(ip string, other bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/v1/admin/nodes", nil)
		r.RemoteAddr = ip + ":1234"
		if other {
			r.Header.Set("Test-Other", "true")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := invoke("192.0.2.1", false); w.Code != 501 {
		t.Fatal(w.Code, w.Body.String())
	}
	for i := 2; i < 30; i++ {
		invoke(fmt.Sprintf("192.0.2.%d", i), false)
	}
	if w := invoke("192.0.2.1", false); w.Code != 429 || w.Header().Get("Retry-After") == "" || !strings.Contains(w.Body.String(), "Request rate exceeded") {
		t.Fatal("principal budget reset", w.Code, w.Body.String())
	}
	if w := invoke("192.0.2.1", true); w.Code != 501 {
		t.Fatal("ingress filled principal capacity", w.Code, w.Body.String())
	}
}
func TestAT28_BrowserRateAndConcurrencyRetryAfter(t *testing.T) {
	c := config.Defaults()
	c.AnonymousRate = config.Bucket{PerSecond: .0001, Burst: 1}
	h := BrowserRoutes(c, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/auth/session", nil))
		if i == 1 && (w.Code != 429 || w.Header().Get("Retry-After") == "") {
			t.Fatal(w.Code, w.Header())
		}
	}
	c.MaxConcurrent = 1
	c.AnonymousRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	started, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	h = BrowserRoutes(c, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-release }))
	go func() {
		defer close(done)
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/auth/login", nil))
	}()
	<-started
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/auth/session", nil))
	close(release)
	<-done
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal(w.Code, w.Header())
	}
}
