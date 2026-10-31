package httpapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/prometheus/client_golang/prometheus"
)

// TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable locks the r12 proof
// that one IPv6 /48 cannot fill the tables used by authenticated callers.
// Anonymous enrollment inserts into the anonymous ingress table and the
// anonymous class table only. An mTLS request without a verified client
// certificate inserts into none.
func TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable(t *testing.T) {
	t.Run("public_listener_5000_slash64s_on_enroll", func(t *testing.T) {
		c := config.Defaults()
		c.PublicOrigin = "https://portal.example.invalid"
		adminP := principal(auth.Admin)
		h := fixtureConfig(t, adminP, false, nil, c)

		// One request from each of MaxRatePrincipals/2 distinct /64s inside
		// 2001:db8:abcd::/48. D-083 admits only the first 64 into the two
		// anonymous tables; the rest cannot occupy an authenticated table.
		enrollBody := string(schemaEnroll())
		for subnet := 0; subnet < c.MaxRatePrincipals/2; subnet++ {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enroll", strings.NewReader(enrollBody))
			r.Header.Set("Content-Type", "application/json")
			r.RemoteAddr = fmt.Sprintf("[2001:db8:abcd:%04x::1]:40000", subnet)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
		}

		r := request(http.MethodGet, "/api/v1/admin/nodes", "", adminP)
		r.RemoteAddr = "198.51.100.77:54321"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNotImplemented {
			t.Fatalf("admin from a fresh address after a /48 enroll sweep: got %d, want 501: %s", w.Code, strings.TrimSpace(w.Body.String()))
		}
	})

	t.Run("mtls_listener_2500_slash64s_without_client_cert", func(t *testing.T) {
		c := config.Defaults()
		c.PublicOrigin = "https://portal.example.invalid"
		agentP := principal(auth.Agent)
		h := fixtureConfig(t, agentP, true, nil, c)

		// Each /64 visits every mTLS class with no client certificate.
		// Those requests are refused before any limiter insert.
		mtlsRoutes := []struct{ method, path string }{
			{http.MethodPost, "/api/v1/agent/observations"},
			{http.MethodPost, "/api/v1/agent/lease-renewals"},
			{http.MethodPost, "/api/v1/agent/usage-batches"},
			{http.MethodGet, "/api/v1/agent/desired-state"},
		}
		for subnet := 0; subnet < c.MaxRatePrincipals/len(mtlsRoutes); subnet++ {
			peerAddr := fmt.Sprintf("[2001:db8:abcd:%04x::1]:40000", subnet)
			for _, rt := range mtlsRoutes {
				r := httptest.NewRequest(rt.method, rt.path, nil)
				r.RemoteAddr = peerAddr
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
			}
		}

		leaseBody := fmt.Sprintf(`{"renewal_request_id":%q,"node_id":%q,"node_incarnation_id":%q,"current_lease_id":null,"applied_config_generation":null}`,
			"019a0000-0000-7000-8000-000000000001", agentP.NodeID, agentP.IncarnationID)
		r := request(http.MethodPost, "/api/v1/agent/lease-renewals", leaseBody, agentP)
		r.RemoteAddr = "198.51.100.77:54321"
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNotImplemented {
			t.Fatalf("certified node after an unauthenticated /48 sweep: got %d, want 501: %s", w.Code, strings.TrimSpace(w.Body.String()))
		}
	})
}

// TestReviewR12_Q4_ConcurrencyGate429OmitsRateLimitedMetric locks the r12
// proof that a per-class concurrency refusal increments
// fvpn_control_rate_limited_total{class,limiter="concurrency"}.
func TestReviewR12_Q4_ConcurrencyGate429OmitsRateLimitedMetric(t *testing.T) {
	c := wideRates(config.Defaults())
	c.PublicOrigin = "https://portal.example.invalid"
	c.MaxConcurrent = 1
	p := principal(auth.Admin)
	sets, err := Compose()
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	reg := prometheus.NewRegistry()
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "fvpn_control_rate_limited_total",
		Help: "TEST ONLY rate limited counter.",
	}, []string{"class", "limiter"})
	if err := reg.Register(counter); err != nil {
		t.Fatalf("Register: %v", err)
	}

	h, err := New(sets, Options{
		Config: c,
		Sessions: auth.SessionAuthenticatorFunc(func(*http.Request) (auth.Principal, error) {
			return p, nil
		}),
		RateLimited: counter,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		r1 := request(http.MethodPost, "/api/v1/admin/groups", `{"code":"a","name":"A"}`, p)
		r1.RemoteAddr = "198.51.100.10:1001"
		r1.Body = readTrackerFunc(func(b []byte) (int, error) {
			close(started)
			<-release
			return 0, io.EOF
		})
		h.ServeHTTP(httptest.NewRecorder(), r1)
	}()
	<-started

	r2 := request(http.MethodPost, "/api/v1/admin/groups", `{"code":"b","name":"B"}`, p)
	r2.RemoteAddr = "198.51.100.11:1002"
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)

	close(release)
	<-done

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on saturated concurrency gate, got %d: %s", w2.Code, w2.Body.String())
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	var concurrency float64
	for _, f := range families {
		if f.GetName() != "fvpn_control_rate_limited_total" {
			continue
		}
		for _, m := range f.GetMetric() {
			var class, limiter string
			for _, lp := range m.GetLabel() {
				switch lp.GetName() {
				case "class":
					class = lp.GetValue()
				case "limiter":
					limiter = lp.GetValue()
				}
			}
			if class == "browser" && limiter == "concurrency" {
				concurrency += m.GetCounter().GetValue()
			}
		}
	}
	if concurrency < 1 {
		t.Fatalf("concurrency gate returned 429 (%s) with Retry-After=%q but fvpn_control_rate_limited_total{class=%q,limiter=%q} stayed 0",
			strings.TrimSpace(w2.Body.String()), w2.Header().Get("Retry-After"), "browser", "concurrency")
	}
}

// TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8 locks
// the r12 proof that a body one byte above canonjson.MaxBytes is held to the
// same subset as canonjson.Canonicalize: duplicate keys, invalid UTF-8, and
// lone UTF-16 surrogates are HTTP 400.
func TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8(t *testing.T) {
	c := wideRates(config.Defaults())
	c.PublicOrigin = "https://portal.example.invalid"
	agentP := principal(auth.Agent)
	h := fixtureConfig(t, agentP, true, nil, c)

	validBatch := schemaUsageBatch(agentP)
	dupBatch := append([]byte(`{"node_id":"00000000-0000-7000-8000-000000000999",`), validBatch[1:]...)
	badUTF8Batch := bytes.Replace(validBatch, []byte(`"2026-10"`), []byte("\"2026-\xff\""), 1)
	surrogateBatch := bytes.Replace(validBatch, []byte(`"2026-10"`), []byte(`"2026-\uDEAD"`), 1)

	for _, tc := range []struct {
		name string
		doc  []byte
	}{
		{"duplicate_keys", dupBatch},
		{"invalid_utf8_byte", badUTF8Batch},
		{"lone_utf16_surrogate", surrogateBatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at1MiB := padJSONObject(t, tc.doc, canonjson.MaxBytes)
			r1 := request(http.MethodPost, "/api/v1/agent/usage-batches", string(at1MiB), agentP)
			r1.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			w1 := httptest.NewRecorder()
			h.ServeHTTP(w1, r1)
			if w1.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 at 1 MiB for %s, got %d", tc.name, w1.Code)
			}

			above1MiB := padJSONObject(t, tc.doc, canonjson.MaxBytes+1)
			r2 := request(http.MethodPost, "/api/v1/agent/usage-batches", string(above1MiB), agentP)
			r2.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			w2 := httptest.NewRecorder()
			h.ServeHTTP(w2, r2)
			if w2.Code != http.StatusBadRequest {
				t.Fatalf("%s: got HTTP %d at %d bytes, want 400: %s",
					tc.name, w2.Code, len(above1MiB), strings.TrimSpace(w2.Body.String()))
			}
		})
	}

	t.Run("integer_outside_safe_range", func(t *testing.T) {
		doc := append([]byte(`{"n":`), bytes.Repeat([]byte{'9'}, 17)...)
		doc = append(doc, '}')
		for _, n := range []int{canonjson.MaxBytes, canonjson.MaxBytes + 1} {
			body := padJSONObject(t, doc, n)
			r := request(http.MethodPost, "/api/v1/agent/usage-batches", string(body), agentP)
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("17-digit integer at %d bytes: got %d (%s)", n, w.Code, strings.TrimSpace(w.Body.String()))
			}
		}
	})

	t.Run("duplicate_keys_after_unescape", func(t *testing.T) {
		above := padJSONObject(t, []byte(`{"a":1,"\u0061":2}`), canonjson.MaxBytes+1)
		r := request(http.MethodPost, "/api/v1/agent/usage-batches", string(above), agentP)
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("unescaped duplicate key at %d bytes: got %d (%s)", len(above), w.Code, strings.TrimSpace(w.Body.String()))
		}
	})

	t.Run("subset_rules_above_canon_max", func(t *testing.T) {
		reject := []struct {
			name string
			body []byte
		}{
			{"fraction", padJSONObject(t, []byte(`{"n":1.5}`), canonjson.MaxBytes+1)},
			{"exponent", padJSONObject(t, []byte(`{"n":1e2}`), canonjson.MaxBytes+1)},
			{"negative_zero", padJSONObject(t, []byte(`{"n":-0}`), canonjson.MaxBytes+1)},
			{"leading_zero", padJSONObject(t, []byte(`{"n":01}`), canonjson.MaxBytes+1)},
			{"nested_duplicate", padJSONObject(t, []byte(`{"o":{"a":1,"a":2}}`), canonjson.MaxBytes+1)},
			{"bom", append([]byte{0xef, 0xbb, 0xbf}, padJSONObject(t, []byte(`{}`), canonjson.MaxBytes+1)...)},
		}
		for _, tc := range reject {
			r := request(http.MethodPost, "/api/v1/agent/usage-batches", string(tc.body), agentP)
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("%s at %d bytes: got %d (%s)", tc.name, len(tc.body), w.Code, strings.TrimSpace(w.Body.String()))
			}
		}
		pair := padJSONObject(t, []byte(`{"a":"\uD800\uDC00"}`), canonjson.MaxBytes+1)
		r := request(http.MethodPost, "/api/v1/agent/usage-batches", string(pair), agentP)
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == http.StatusBadRequest {
			t.Fatalf("valid surrogate pair at %d bytes was rejected: %s", len(pair), strings.TrimSpace(w.Body.String()))
		}
	})
}

// TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld verifies the Question 4 and
// 5 boundaries that already held: TCP peer keying, per-node subject buckets
// on one NAT address, IPv6 /64 sharing, Retry-After, chunked cap+1 with the
// limiter charged before the body is read, and rejection of NUL, deep
// nesting, and out-of-range integers. canonjson.MaxBytes stays the signed
// document bound.
func TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld(t *testing.T) {
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.LeaseRate = config.Bucket{PerSecond: 0.0001, Burst: 5}
	nodeA := principal(auth.Agent)
	nodeB := principal(auth.Agent)
	sets, _ := Compose()
	h, err := New(sets, Options{
		Config:        c,
		AgentListener: true,
		Agents: auth.AgentAuthenticatorFunc(func(_ context.Context, cert *x509.Certificate) (auth.Principal, error) {
			if cert.SerialNumber != nil && cert.SerialNumber.Int64() == 2 {
				return nodeB, nil
			}
			return nodeA, nil
		}),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sendRenew := func(p auth.Principal, serial int64) int {
		leaseBody := fmt.Sprintf(`{"renewal_request_id":%q,"node_id":%q,"node_incarnation_id":%q,"current_lease_id":null,"applied_config_generation":null}`,
			"019a0000-0000-7000-8000-000000000001", p.NodeID, p.IncarnationID)
		r := request(http.MethodPost, "/api/v1/agent/lease-renewals", leaseBody, p)
		r.RemoteAddr = "198.51.100.50:44321"
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{SerialNumber: big.NewInt(serial)}}}}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	for i := 0; i < 5; i++ {
		if code := sendRenew(nodeA, 1); code != http.StatusNotImplemented {
			t.Fatalf("nodeA lease renewal %d on shared NAT IP failed with %d", i+1, code)
		}
	}
	if code := sendRenew(nodeA, 1); code != http.StatusTooManyRequests {
		t.Fatalf("nodeA 6th lease renewal should be 429, got %d", code)
	}
	for i := 0; i < 5; i++ {
		if code := sendRenew(nodeB, 2); code != http.StatusNotImplemented {
			t.Fatalf("nodeB lease renewal %d on shared NAT IP failed with %d", i+1, code)
		}
	}
	if code := sendRenew(nodeB, 2); code != http.StatusTooManyRequests {
		t.Fatalf("nodeB 6th lease renewal should be 429, got %d", code)
	}

	cAnon := config.Defaults()
	cAnon.PublicOrigin = "https://portal.example.invalid"
	cAnon.AnonymousRate = config.Bucket{PerSecond: 0.0001, Burst: 10}
	hAnon := fixtureConfig(t, auth.Principal{}, false, nil, cAnon)
	for i := 1; i <= 10; i++ {
		r := request(http.MethodPost, "/api/v1/agent/enroll", string(schemaEnroll()), auth.Principal{})
		r.RemoteAddr = fmt.Sprintf("[2001:db8:1:2::%x]:12345", i)
		w := httptest.NewRecorder()
		hAnon.ServeHTTP(w, r)
		if w.Code != http.StatusNotImplemented {
			t.Fatalf("IPv6 /128 #%d in same /64 got %d (%s)", i, w.Code, w.Body.String())
		}
	}
	r11 := request(http.MethodPost, "/api/v1/agent/enroll", string(schemaEnroll()), auth.Principal{})
	r11.RemoteAddr = "[2001:db8:1:2::ff]:12345"
	w11 := httptest.NewRecorder()
	hAnon.ServeHTTP(w11, r11)
	if w11.Code != http.StatusTooManyRequests || w11.Header().Get("Retry-After") == "" {
		t.Fatalf("11th IPv6 /128 in same /64 should be 429 with Retry-After, got code=%d retry-after=%q", w11.Code, w11.Header().Get("Retry-After"))
	}

	cChunk := config.Defaults()
	cChunk.PublicOrigin = "https://portal.example.invalid"
	cChunk.UsageRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
	hChunk := fixtureConfig(t, nodeA, true, nil, cChunk)
	overCap := padJSONObject(t, schemaUsageBatch(nodeA), int(cChunk.UsageBodyBytes)+1)
	rChunk := httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", io.NopCloser(bytes.NewReader(overCap)))
	rChunk.ContentLength = -1
	rChunk.Header.Set("Content-Type", "application/json")
	rChunk.RemoteAddr = "198.51.100.88:50000"
	rChunk.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	wChunk := httptest.NewRecorder()
	hChunk.ServeHTTP(wChunk, rChunk)
	if wChunk.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("chunked cap+1 expected 413, got %d", wChunk.Code)
	}
	bodyRead := false
	rSecond := httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", readTrackerFunc(func(p []byte) (int, error) {
		bodyRead = true
		return 0, io.EOF
	}))
	rSecond.Header.Set("Content-Type", "application/json")
	rSecond.RemoteAddr = "198.51.100.88:50000"
	rSecond.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	wSecond := httptest.NewRecorder()
	hChunk.ServeHTTP(wSecond, rSecond)
	if wSecond.Code != http.StatusTooManyRequests || bodyRead {
		t.Fatalf("expected 429 before reading body, got code=%d bodyRead=%v", wSecond.Code, bodyRead)
	}

	cAdv := wideRates(config.Defaults())
	cAdv.PublicOrigin = "https://portal.example.invalid"
	hAdv := fixtureConfig(t, nodeA, true, nil, cAdv)

	for _, adv := range []struct {
		name string
		raw  []byte
	}{
		{"nul_byte_in_string", append([]byte(`{"node_id":"`), 0x00, '"', '}')},
		{"nul_byte_at_usage_cap", padJSONObject(t, append([]byte(`{"node_id":"`), 0x00, '"', '}'), int(cAdv.UsageBodyBytes))},
		{"deep_nesting_260", []byte(strings.Repeat(`{"a":`, 260) + "1" + strings.Repeat("}", 260))},
		{"deep_nesting_10005_above_canon_max", padJSONObject(t, []byte(strings.Repeat(`{"a":`, 10005)+"1"+strings.Repeat("}", 10005)), canonjson.MaxBytes+64)},
		{"huge_number_100_digits", []byte(`{"n":` + strings.Repeat("9", 100) + `}`)},
	} {
		rAdv := request(http.MethodPost, "/api/v1/agent/usage-batches", string(adv.raw), nodeA)
		rAdv.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		wAdv := httptest.NewRecorder()
		hAdv.ServeHTTP(wAdv, rAdv)
		if wAdv.Code != http.StatusBadRequest {
			t.Fatalf("adversarial body %s: expected 400, got %d (%s)", adv.name, wAdv.Code, wAdv.Body.String())
		}
	}

	overSigned := padJSONObject(t, []byte(`{"signed":{},"signatures":[]}`), canonjson.MaxBytes+1)
	if _, err := canonjson.Canonicalize(overSigned); err == nil {
		t.Fatalf("canonjson.Canonicalize accepted %d bytes (> MaxBytes)", len(overSigned))
	}
	if _, _, err := envelope.Verify(overSigned, &envelope.TrustStore{}, envelope.PurposeDesiredState, nodeA.AuthenticatedAt); err == nil {
		t.Fatalf("envelope.Verify accepted %d bytes (> MaxBytes)", len(overSigned))
	}
}

type readTrackerFunc func(p []byte) (int, error)

func (f readTrackerFunc) Read(p []byte) (int, error) { return f(p) }
func (f readTrackerFunc) Close() error               { return nil }
