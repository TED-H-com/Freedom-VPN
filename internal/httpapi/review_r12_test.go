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

// TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable proves that a single
// IPv6 /48 allocation (which contains 65,536 distinct /64 prefixes) can fill
// the shared pre-authentication `ingress` limiter map (MaxRatePrincipals = 10,000,
// 1-hour eviction) by sending 1 request from each of 5,000 /64 prefixes to the
// anonymous enrollment endpoint (`/api/v1/agent/enroll`), because each anonymous
// request inserts TWO keys into `ingress` (`"ingress:anonymous:"+peer` at line 309
// and `"anonymous:anonymous:"+peer` at line 353). Once `ingress` reaches
// MaxRatePrincipals, every legitimate mTLS node or browser user connecting from
// a new IP address on ANY route class (`heartbeat`, `lease`, `usage`, `agent`,
// `browser`) is rejected with HTTP 429 RATE_LIMITED for up to 1 hour.
func TestReviewR12_Q4_IPv6Slash48ExhaustsSharedIngressTable(t *testing.T) {
	// TODO(review): control owner — isolate unauthenticated/anonymous rate-limit
	// state from authenticated/mTLS classes and do not let a /48 sweep lock out
	// new mTLS node or browser addresses for an hour.
	t.Run("public_listener_5000_slash64s_on_enroll", func(t *testing.T) {
		c := config.Defaults()
		c.PublicOrigin = "https://portal.example.invalid"
		adminP := principal(auth.Admin)
		h := fixtureConfig(t, adminP, false, nil, c)

		// Sweep 5,000 distinct /64 subnets inside one IPv6 /48 (2001:db8:abcd::/48)
		// against the unauthenticated /api/v1/agent/enroll endpoint (1 request per /64).
		// Each request inserts TWO entries into `ingress`:
		//   1. "ingress:anonymous:" + peer (middleware.go:309)
		//   2. "anonymous:anonymous:" + peer (middleware.go:353)
		enrollBody := string(schemaEnroll())
		for subnet := 0; subnet < c.MaxRatePrincipals/2; subnet++ {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enroll", strings.NewReader(enrollBody))
			r.Header.Set("Content-Type", "application/json")
			r.RemoteAddr = fmt.Sprintf("[2001:db8:abcd:%04x::1]:40000", subnet)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
		}

		// Now a legitimate authenticated admin connects from a new IPv4 address.
		r := request(http.MethodGet, "/api/v1/admin/nodes", "", adminP)
		r.RemoteAddr = "198.51.100.77:54321"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("TODO(review): 5,000 /64s from one IPv6 /48 on /api/v1/agent/enroll filled shared ingress map (%d entries) and locked out authenticated admin from 198.51.100.77 with 429: %s",
				c.MaxRatePrincipals, strings.TrimSpace(w.Body.String()))
		}
	})

	t.Run("mtls_listener_2500_slash64s_without_client_cert", func(t *testing.T) {
		c := config.Defaults()
		c.PublicOrigin = "https://portal.example.invalid"
		agentP := principal(auth.Agent)
		h := fixtureConfig(t, agentP, true, nil, c)

		// On the mTLS listener, `ingress.allow("ingress:"+ingressClass+":"+peer)` runs at
		// middleware.go:309 BEFORE verifying `r.TLS` / client certificates at line 330.
		// Sweeping 2,500 /64s across the 4 mTLS route classes (heartbeat, lease, usage, agent)
		// without any client certificate inserts 10,000 keys into `ingress`.
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

		// Now a legitimate mTLS node with a verified client certificate connects from 198.51.100.77.
		r := request(http.MethodPost, "/api/v1/agent/lease-renewals", string(schemaRenew(agentP)), agentP)
		r.RemoteAddr = "198.51.100.77:54321"
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("TODO(review): 2,500 unauthenticated /64s from one IPv6 /48 filled mTLS ingress map (%d entries) before client-cert check and locked out mTLS node 198.51.100.77 with 429: %s",
				c.MaxRatePrincipals, strings.TrimSpace(w.Body.String()))
		}
	})
}

// TestReviewR12_Q4_ConcurrencyGate429OmitsRateLimitedMetric proves that when the
// per-class concurrency gate `gates[class]` in httpapi.New (middleware.go:364-370)
// rejects a request with HTTP 429 RATE_LIMITED ("Request concurrency exceeded."),
// it does not increment Options.RateLimited (`fvpn_control_rate_limited_total`),
// unlike BrowserRoutesWithMetrics (middleware.go:676) and contrary to
// docs/control/README.md:84 and docs/control/AUTH.md:125.
func TestReviewR12_Q4_ConcurrencyGate429OmitsRateLimitedMetric(t *testing.T) {
	// TODO(review): control owner — increment Options.RateLimited on concurrency-gate
	// 429 responses in httpapi.New (middleware.go:367).
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

	// gates["browser"] is acquired at middleware.go:365 before reading r.Body at line 398.
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
	var total float64
	for _, f := range families {
		if f.GetName() == "fvpn_control_rate_limited_total" {
			for _, m := range f.GetMetric() {
				total += m.GetCounter().GetValue()
			}
		}
	}
	if total == 0 {
		t.Fatalf("TODO(review): concurrency gate returned 429 (%s) with Retry-After=%q but fvpn_control_rate_limited_total remained 0",
			strings.TrimSpace(w2.Body.String()), w2.Header().Get("Retry-After"))
	}
}

// TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8 proves
// that after follow-up 16 (`checkJSONDocument` in internal/httpapi/middleware.go:505-530),
// bodies up to canonjson.MaxBytes (1 MiB) enforce RFC 8785 / canonjson rules
// (rejecting duplicate object keys, invalid UTF-8 bytes, lone UTF-16 surrogates,
// and non-integer numbers with 400 MALFORMED_REQUEST), whereas padding the exact
// same payload by 1 byte to `canonjson.MaxBytes + 1` (1,048,577 bytes) switches
// to `json.NewDecoder.Decode(&doc)` into `json.RawMessage`, which accepts duplicate
// keys, invalid UTF-8 bytes, and lone surrogates without returning 400.
func TestReviewR12_Q5_AboveCanonMaxBytesAcceptsDuplicateKeysAndInvalidUTF8(t *testing.T) {
	// TODO(review): control owner — enforce duplicate-key, valid-UTF-8, surrogate,
	// and depth checks consistently in checkJSONDocument above canonjson.MaxBytes.
	c := wideRates(config.Defaults())
	c.PublicOrigin = "https://portal.example.invalid"
	agentP := principal(auth.Agent)
	h := fixtureConfig(t, agentP, true, nil, c)

	validBatch := schemaUsageBatch(agentP)
	// Inject a duplicate "node_id" key at the start of the object (first key has a random UUID,
	// second key has the certificate's agentP.NodeID).
	dupBatch := append([]byte(`{"node_id":"00000000-0000-7000-8000-000000000999",`), validBatch[1:]...)

	// Inject invalid UTF-8 byte (0xff) inside period_id string.
	badUTF8Batch := bytes.Replace(validBatch, []byte(`"2026-10"`), []byte("\"2026-\xff\""), 1)

	// Inject lone UTF-16 surrogate (\uDEAD) inside period_id string.
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
				t.Fatalf("TODO(review): %s rejected with 400 at %d bytes (1 MiB) but returned HTTP %d at %d bytes (1 MiB + 1): %s",
					tc.name, len(at1MiB), w2.Code, len(above1MiB), strings.TrimSpace(w2.Body.String()))
			}
		})
	}
}

// TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld verifies and measures the
// properties from Questions 4 and 5 that held:
//   - TCP peer keying (`r.RemoteAddr`), ignoring X-Forwarded-For / X-Real-IP.
//   - Two mTLS nodes sharing one NAT IP get independent post-auth subject buckets
//     (burst 5 each on heartbeat, burst 5 each on lease, burst 10 each on usage)
//     up to the shared pre-auth IngressRate (burst 60).
//   - IPv6 /64 keying: multiple /128 addresses in the same /64 share one bucket
//     (total burst 10 on anonymous).
//   - Retry-After header rounds up sub-second wait times to at least 1 second.
//   - Exact byte boundaries (cap-1, cap, cap+1) across all classes (browser, agent,
//     heartbeat, lease, usage, webhook, anonymous), chunked transfer encoding with
//     ContentLength = -1, NUL byte rejection, and limiter token charged before body read.
func TestReviewR12_Q4_Q5_LimiterAndBodyBoundariesHeld(t *testing.T) {
	// 1. Two mTLS nodes sharing one NAT IP on lease renewal (burst 5 per node, shared ingress burst 60).
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
		r.RemoteAddr = "198.51.100.50:44321" // Shared NAT IP
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
	// Node B on the exact same NAT IP still gets its full burst of 5.
	for i := 0; i < 5; i++ {
		if code := sendRenew(nodeB, 2); code != http.StatusNotImplemented {
			t.Fatalf("nodeB lease renewal %d on shared NAT IP failed with %d", i+1, code)
		}
	}
	if code := sendRenew(nodeB, 2); code != http.StatusTooManyRequests {
		t.Fatalf("nodeB 6th lease renewal should be 429, got %d", code)
	}

	// 2. IPv6 /128 hosts inside the same /64 share one bucket (burst 10 on anonymous).
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

	// 3. Chunked transfer with ContentLength = -1 at usage cap (8 MiB) and cap + 1,
	// and proof that rate-limiter token is spent before reading the body.
	cChunk := config.Defaults()
	cChunk.PublicOrigin = "https://portal.example.invalid"
	cChunk.UsageRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
	hChunk := fixtureConfig(t, nodeA, true, nil, cChunk)
	overCap := padJSONObject(t, schemaUsageBatch(nodeA), int(cChunk.UsageBodyBytes)+1)
	rChunk := httptest.NewRequest(http.MethodPost, "/api/v1/agent/usage-batches", io.NopCloser(bytes.NewReader(overCap)))
	rChunk.ContentLength = -1 // chunked / unknown length
	rChunk.Header.Set("Content-Type", "application/json")
	rChunk.RemoteAddr = "198.51.100.88:50000"
	rChunk.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	wChunk := httptest.NewRecorder()
	hChunk.ServeHTTP(wChunk, rChunk)
	if wChunk.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("chunked cap+1 expected 413, got %d", wChunk.Code)
	}
	// Because the limiter ran BEFORE reading the 8 MiB+1 body, the single Burst=1 token
	// is already spent; a second request immediately gets 429 without reading its body.
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

	// 4. Adversarial bodies (NUL byte, deep nesting, huge number) and signed-document
	// bound (canonjson.MaxBytes = 1 MiB).
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

	// Signed-document path (canonjson.Canonicalize / envelope.Verify) still refuses > canonjson.MaxBytes.
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
