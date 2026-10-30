package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/prometheus/client_golang/prometheus"
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
	c.LoginRate = config.Bucket{PerSecond: .0001, Burst: 1}
	h := BrowserRoutes(c, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/auth/session", nil))
		if i == 1 && (w.Code != 429 || w.Header().Get("Retry-After") == "") {
			t.Fatal(w.Code, w.Header())
		}
	}
	c.MaxConcurrent = 1
	c.LoginRate = config.Bucket{PerSecond: 1000, Burst: 1000}
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

func TestAT31_D060_LoginAddressBurstAndMetric(t *testing.T) {
	c := config.Defaults()
	// Keep default Burst=40 and freeze refill within the test window so the 41st
	// login/callback request is deterministically inside one refill window.
	c.LoginRate = config.Bucket{PerSecond: 0.0001, Burst: config.Defaults().LoginRate.Burst}
	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fvpn_control_rate_limited_total",
		Help: "TEST ONLY rate limited counter.",
	}, []string{"class", "limiter"})
	if err := reg.Register(counter); err != nil {
		t.Fatal(err)
	}
	h := BrowserRoutesWithMetrics(c, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), counter)

	const signIns = 20
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, signIns)
	for i := 0; i < signIns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			for _, path := range []string{"/auth/login", "/auth/callback", "/auth/session"} {
				req := httptest.NewRequest("GET", path, nil)
				req.RemoteAddr = "198.51.100.10:40000"
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK {
					errs <- fmt.Errorf("sign-in %d %s: got HTTP %d, want 200", idx, path, rec.Code)
					return
				}
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}

	// The 20 concurrent sign-ins consumed 20 (/auth/login) + 20 (/auth/callback) = 40
	// tokens from the login address bucket. The 41st login request inside one refill
	// window must be refused with HTTP 429, Retry-After, and RATE_LIMITED.
	req41 := httptest.NewRequest("GET", "/auth/login", nil)
	req41.RemoteAddr = "198.51.100.10:40000"
	rec41 := httptest.NewRecorder()
	h.ServeHTTP(rec41, req41)
	if rec41.Code != http.StatusTooManyRequests || rec41.Header().Get("Retry-After") == "" || !strings.Contains(rec41.Body.String(), `"code":"RATE_LIMITED"`) {
		t.Fatalf("41st login request: status=%d headers=%v body=%s", rec41.Code, rec41.Header(), rec41.Body.String())
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var got float64
	for _, f := range families {
		if f.GetName() == "fvpn_control_rate_limited_total" {
			for _, m := range f.GetMetric() {
				got = m.GetCounter().GetValue()
			}
		}
	}
	if got != 1 {
		t.Fatalf("rate_limited_total{class=login,limiter=address} = %v, want 1", got)
	}
}

func TestAnonymousAddressBurstUsesIPv6Prefix(t *testing.T) {
	c := config.Defaults()
	c.IngressRate = config.Bucket{PerSecond: 1000, Burst: 1000}
	// Unauthenticated routes key the class bucket on the connection address.
	// Hold every class bucket to one token so the assertion does not depend on
	// which classified rate the route selects.
	slow := config.Bucket{PerSecond: 0.0001, Burst: 1}
	c.AnonymousRate = slow
	c.AgentRate = slow
	c.BrowserRate = slow
	c.MaxRatePrincipals = 100
	sets, err := Compose()
	if err != nil {
		t.Fatal(err)
	}
	h, err := New(sets, Options{Config: c})
	if err != nil {
		t.Fatal(err)
	}
	post := func(remote string, header map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enroll", strings.NewReader(`{}`))
		req.RemoteAddr = remote
		req.Header.Set("Content-Type", "application/json")
		for k, v := range header {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	first := post("[2001:db8:9:1::1]:443", nil)
	if first.Code == http.StatusTooManyRequests {
		t.Fatal(first.Code, first.Body.String())
	}
	same := post("[2001:db8:9:1::2]:443", map[string]string{"X-Forwarded-For": "198.51.100.9"})
	if same.Code != http.StatusTooManyRequests || same.Header().Get("Retry-After") == "" || !strings.Contains(same.Body.String(), "RATE_LIMITED") {
		t.Fatal("same /64", same.Code, same.Header().Get("Retry-After"), same.Body.String())
	}
	bare := post("2001:db8:9:1::3:443", nil)
	if bare.Code != http.StatusTooManyRequests {
		t.Fatal("unbracketed address in the same /64", bare.Code, bare.Body.String())
	}
	other := post("[2001:db8:9:2::1]:443", map[string]string{"X-Forwarded-For": "2001:db8:9:1::1"})
	if other.Code == http.StatusTooManyRequests {
		t.Fatal("different /64 or forwarded header was trusted", other.Code, other.Body.String())
	}
	if post("192.0.2.10:9", nil).Code == http.StatusTooManyRequests {
		t.Fatal("ipv4 shared the ipv6 prefix bucket")
	}
	if post("192.0.2.10:10", nil).Code != http.StatusTooManyRequests {
		t.Fatal("ipv4 did not keep a per-address bucket")
	}

	// Both the ingress key and the anonymous key land in one map, so one
	// parsed address needs two principal slots. A refused address must not take either.
	c.MaxRatePrincipals = 2
	h, err = New(sets, Options{Config: c})
	if err != nil {
		t.Fatal(err)
	}
	if w := post("not-an-address", nil); w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Fatal("unparseable", w.Code, w.Header().Get("Retry-After"), w.Body.String())
	}
	if w := post("192.0.2.40:9", nil); w.Code == http.StatusTooManyRequests {
		t.Fatal("unparseable address occupied a principal slot", w.Code, w.Body.String())
	}
}
