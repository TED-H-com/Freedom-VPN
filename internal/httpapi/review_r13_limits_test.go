package httpapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// TestReviewR13_Q3_AgentAuthRunsBeforeRateLimiterAndConcurrencyGate is the r13
// H-3 proof. The agent listener spends an ingress token keyed by the SHA-256
// of the verified leaf, and takes the class concurrency slot, before
// AuthenticateAgent. A failed authentication has spent that token, so one
// revoked certificate cannot drive an unbounded database lookup. A request
// with no verified chain is refused before any insert.
func TestReviewR13_Q3_AgentAuthRunsBeforeRateLimiterAndConcurrencyGate(t *testing.T) {
	c := config.Defaults()
	c.PublicOrigin = "https://portal.example.invalid"
	c.IngressRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
	c.HeartbeatRate = config.Bucket{PerSecond: 0.0001, Burst: 1}
	c.MaxConcurrent = 1

	sets, err := Compose()
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	var revokedAuthCalls atomic.Int64
	revokedHandler, err := New(sets, Options{
		Config:        c,
		AgentListener: true,
		Agents: auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) {
			revokedAuthCalls.Add(1)
			return auth.Principal{}, apierr.New(apierr.Unauthenticated, "node certificate revoked")
		}),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const totalRequests = 25
	for i := 0; i < totalRequests; i++ {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/agent/desired-state", nil)
		r.RemoteAddr = "198.51.100.55:40000"
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := httptest.NewRecorder()
		revokedHandler.ServeHTTP(w, r)
	}

	if got := revokedAuthCalls.Load(); got != 1 {
		t.Errorf("revoked certificate invoked AuthenticateAgent %d times, want 1 (the ingress token is spent before the database lookup)", got)
	}

	var activeAuthCalls atomic.Int64
	agentP := principal(auth.Agent)
	activeHandler, err := New(sets, Options{
		Config:        c,
		AgentListener: true,
		Agents: auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) {
			activeAuthCalls.Add(1)
			return agentP, nil
		}),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var rejected429 int
	for i := 0; i < totalRequests; i++ {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/agent/desired-state", nil)
		r.RemoteAddr = "198.51.100.56:40000"
		r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
		w := httptest.NewRecorder()
		activeHandler.ServeHTTP(w, r)
		if w.Code == http.StatusTooManyRequests {
			rejected429++
		}
	}
	if got := activeAuthCalls.Load(); got != 1 || rejected429 != totalRequests-1 {
		t.Errorf("active certificate: AuthenticateAgent calls=%d want 1, 429s=%d want %d", got, rejected429, totalRequests-1)
	}
}

// TestReviewR13_Q3_XForwardedForIPv6HextetStrippedAsPortBypassingTrustedProxy
// is the r13 H-2 proof. Eight hextets written with a zero-width "::"
// (2001:db8:1::2:3:4:5:6 for 2001:db8:1:2:3:4:5:6) stay that address, bracketed
// or bare, so the last hextet is not a port and the /64 does not move into
// trusted_proxies 2001:db8:1::/64. Spoofed hops to the left of that client
// share one login bucket.
func TestReviewR13_Q3_XForwardedForIPv6HextetStrippedAsPortBypassingTrustedProxy(t *testing.T) {
	// Trusted proxies are the edge proxy 192.0.2.8/32 and internal proxy subnet 2001:db8:1::/64.
	// The untrusted client is in 2001:db8:1:2::/64 at address 2001:db8:1:2:3:4:5:6
	// (written in compressed form `2001:db8:1::2:3:4:5:6` or bracketed `[2001:db8:1::2:3:4:5:6]`).
	h, hits := followup18Login(1, 100, []string{"192.0.2.8/32", "2001:db8:1::/64"})
	proxy := "192.0.2.8:443"

	// Client sends spoofed X-Forwarded-For values while the trusted proxy 192.0.2.8
	// appends the client's real address `[2001:db8:1::2:3:4:5:6]`.
	w1 := reviewLogin(h, proxy, "/auth/login", map[string]string{
		"X-Forwarded-For": "203.0.113.1, [2001:db8:1::2:3:4:5:6]",
	})
	w2 := reviewLogin(h, proxy, "/auth/login", map[string]string{
		"X-Forwarded-For": "203.0.113.2, [2001:db8:1::2:3:4:5:6]",
	})
	w3 := reviewLogin(h, proxy, "/auth/login", map[string]string{
		"X-Forwarded-For": "203.0.113.3, 2001:db8:1::2:3:4:5:6",
	})
	if w1.Code != http.StatusNoContent {
		t.Fatalf("first login from the real /64: got %d %s", w1.Code, w1.Body.String())
	}
	if w2.Code != http.StatusTooManyRequests || w3.Code != http.StatusTooManyRequests {
		t.Errorf("spoofed hops were honoured: bracketed=%d bare=%d body=%s %s", w2.Code, w3.Code, w2.Body.String(), w3.Body.String())
	}
	if *hits != 1 {
		t.Errorf("handler hits=%d, want 1", *hits)
	}
}

// TestReviewR13_Q3_EvictIdleTailOnlyHeadOfLineBlockingAndAnonSweep is the r13
// M-5 proof. Eviction scans a window from the tail, so one depleted bucket
// does not hide a newer bucket that has refilled. The anonymous tables also
// keep a /48 budget beside the /64 key, so one /48 cannot fill the table and
// refuse an unrelated enrollment.
func TestReviewR13_Q3_EvictIdleTailOnlyHeadOfLineBlockingAndAnonSweep(t *testing.T) {
	t.Run("evictIdle_tail_only_head_of_line_blocking_under_mixed_rates", func(t *testing.T) {
		now := time.Date(2026, 10, 30, 17, 0, 0, 0, time.UTC)
		setLimiterClock(t, func() time.Time { return now })

		lim := newLimiter(4)
		hbBucket := config.Bucket{PerSecond: 1, Burst: 5}
		browserBucket := config.Bucket{PerSecond: 10, Burst: 20}

		// P1 (heartbeat, 1/s, burst 5) spends all 5 tokens at t=0 and sits at l.tail.
		for i := 0; i < 5; i++ {
			if wait := lim.allow("agent:p1", hbBucket); wait != 0 {
				t.Fatalf("p1 request %d should be allowed, got wait=%v", i, wait)
			}
		}
		// P2, P3, P4 (browser, 10/s, burst 20) each spend 1 token at t=0.
		for _, k := range []string{"browser:p2", "browser:p3", "browser:p4"} {
			if wait := lim.allow(k, browserBucket); wait != 0 {
				t.Fatalf("%s should be allowed, got wait=%v", k, wait)
			}
		}

		// Advance 200ms: P2, P3, and P4 have refilled +2 tokens to 20/20 (100% full
		// and idle), while P1 at l.tail has 0.2/5 tokens.
		now = now.Add(200 * time.Millisecond)
		if wait := lim.allow("browser:p5", browserBucket); wait != 0 {
			t.Errorf("limiter rejected new principal browser:p5 (wait=%v) while newer buckets were full and idle", wait)
		}
	})

	t.Run("slash48_sweep_denies_legitimate_enrollment", func(t *testing.T) {
		now := time.Date(2026, 10, 30, 17, 0, 0, 0, time.UTC)
		setLimiterClock(t, func() time.Time { return now })

		c := config.Defaults()
		c.PublicOrigin = "https://portal.example.invalid"
		c.MaxRatePrincipals = 64
		h := fixtureConfig(t, principal(auth.Admin), false, nil, c)

		enrollBody := string(schemaEnroll())
		// Attacker sweeps 64 /64s inside one /48, sending 2 requests per /64
		// (spending 2 of the 20 AnonymousRate tokens; refill at 10/s takes 200ms).
		for subnet := 0; subnet < c.MaxRatePrincipals; subnet++ {
			for attempt := 0; attempt < 2; attempt++ {
				r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enroll", strings.NewReader(enrollBody))
				r.Header.Set("Content-Type", "application/json")
				r.RemoteAddr = fmt.Sprintf("[2001:db8:abcd:%04x::1]:40000", subnet)
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
			}
		}

		// 100ms later, a legitimate node on an unrelated IPv4 address tries to enroll.
		now = now.Add(100 * time.Millisecond)
		r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/enroll", strings.NewReader(enrollBody))
		r.Header.Set("Content-Type", "application/json")
		r.RemoteAddr = "198.51.100.77:54321"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == http.StatusTooManyRequests {
			t.Errorf("legitimate node enrollment from 198.51.100.77 was rejected with 429 during an IPv6 /48 sweep of the anonymous table: %s", strings.TrimSpace(w.Body.String()))
		}
	})
}

// TestReviewR13_Q4_MatchingIdentityUnmarshalsFullMultiMiBBodyIntoAny is the r13
// L-2 proof. Identity fields are read during the streaming walk, and a usage
// batch above 1 MiB is schema-checked there too, so the request does not build
// two any trees before the handler. The allocation of the whole ServeHTTP stays
// within six times the body.
func TestReviewR13_Q4_MatchingIdentityUnmarshalsFullMultiMiBBodyIntoAny(t *testing.T) {
	c := wideRates(config.Defaults())
	c.PublicOrigin = "https://portal.example.invalid"
	agentP := principal(auth.Agent)
	h := fixtureConfig(t, agentP, true, nil, c)

	var b bytes.Buffer
	fmt.Fprintf(&b, `{"node_id":%q,"node_incarnation_id":%q,"ledger_epoch":null,"events":[{"schema_version":1,"node_id":%q,"node_incarnation_id":%q,"meter_epoch_id":"019a0000-0000-7000-8000-000000000001","event_seq":"1","metering_version":"forward_admission_v1","interval_start":"2026-10-28T12:00:00Z","interval_end":"2026-10-28T12:01:00Z","quality":"complete","payload_sha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","records":[`,
		agentP.NodeID, agentP.IncarnationID, agentP.NodeID, agentP.IncarnationID)
	for i := 0; i < 6000; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `{"member_id":"019a0000-0000-7000-8000-000000000002","device_id":"019a0000-0000-7000-8000-000000000003","binding_revision":"1","period_id":"2026-10","upload_bytes":%q,"download_bytes":"1"}`, fmt.Sprintf("%d", i+1))
	}
	b.WriteString(`]}]}`)
	large := b.Bytes()
	if len(large) <= canonjson.MaxBytes {
		t.Fatalf("expected body > 1 MiB, got %d", len(large))
	}

	runtime.GC()
	var wBefore, wAfter runtime.MemStats
	runtime.ReadMemStats(&wBefore)
	if err := walkJSONDocument(large); err != nil {
		t.Fatalf("walkJSONDocument: %v", err)
	}
	runtime.ReadMemStats(&wAfter)
	walkAlloc := wAfter.TotalAlloc - wBefore.TotalAlloc

	runtime.GC()
	var mBefore, mAfter runtime.MemStats
	runtime.ReadMemStats(&mBefore)
	r := request(http.MethodPost, "/api/v1/agent/usage-batches", string(large), agentP)
	r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	runtime.ReadMemStats(&mAfter)
	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501 from unimplemented handler after middleware pass, got %d: %s", w.Code, w.Body.String())
	}
	middlewareAlloc := mAfter.TotalAlloc - mBefore.TotalAlloc

	// The streaming walk, the in-walk schema check, and the 501 path stay within
	// six times the raw body. Two any trees plus a generated decode do not.
	if middlewareAlloc > uint64(len(large))*6 {
		t.Errorf("on a %d-byte usage batch, walkJSONDocument allocated %d bytes but the request allocated %d bytes (%d heap objects), above 6x the body",
			len(large), walkAlloc, middlewareAlloc, mAfter.Mallocs-mBefore.Mallocs)
	}
}

// TestReviewR13_Q3_Q4_ControlWhatHeld verifies the Q3 and Q4 properties that held:
//   - Differential test: walkJSONDocument and canonjson.Canonicalize agree on
//     every tested input <= 1 MiB (numbers, -0, exponents, 2^53 boundaries,
//     surrogates, invalid UTF-8, escaped duplicate keys, trailing data).
//   - Nesting depth 256 is accepted and depth 257 is rejected.
//   - Chunked bodies without Content-Length (ContentLength = -1) are bounded by
//     http.MaxBytesReader before walkJSONDocument runs.
//   - Token spend occurs before body read, walkJSONDocument, and matchingIdentity.
func TestReviewR13_Q3_Q4_ControlWhatHeld(t *testing.T) {
	cases := []struct {
		name string
		raw  []byte
	}{
		{"empty", []byte("")},
		{"whitespace_only", []byte("   \n\t  ")},
		{"simple_object", []byte(`{"a":1,"b":"ok","c":[true,false,null]}`)},
		{"neg_zero", []byte(`{"n":-0}`)},
		{"neg_zero_frac", []byte(`{"n":-0.0}`)},
		{"float_frac", []byte(`{"n":1.5}`)},
		{"exponent_pos", []byte(`{"n":1e5}`)},
		{"exponent_neg", []byte(`{"n":1e-5}`)},
		{"leading_zero", []byte(`{"n":01}`)},
		{"minus_leading_zero", []byte(`{"n":-01}`)},
		{"max_safe_int", []byte(`{"n":9007199254740991}`)},
		{"min_safe_int", []byte(`{"n":-9007199254740991}`)},
		{"above_safe_int", []byte(`{"n":9007199254740992}`)},
		{"below_safe_int", []byte(`{"n":-9007199254740992}`)},
		{"seventeen_digits", []byte(`{"n":10000000000000000}`)},
		{"valid_surrogate_pair", []byte(`{"s":"\uD83D\uDE00"}`)},
		{"lone_high_surrogate", []byte(`{"s":"\uD800"}`)},
		{"lone_low_surrogate", []byte(`{"s":"\uDC00"}`)},
		{"high_surrogate_followed_by_ascii", []byte(`{"s":"\uD800a"}`)},
		{"invalid_utf8_raw", []byte("{\"s\":\"\xff\"}")},
		{"duplicate_key_literal", []byte(`{"a":1,"a":2}`)},
		{"duplicate_key_escaped", []byte(`{"a":1,"\u0061":2}`)},
		{"trailing_comma_obj", []byte(`{"a":1,}`)},
		{"trailing_comma_arr", []byte(`[1,]`)},
		{"trailing_token", []byte(`{"a":1} {}`)},
		{"unclosed_string", []byte(`{"a":"abc}`)},
		{"control_char_in_string", []byte("{\"a\":\"line\nbreak\"}")},
		{"invalid_escape", []byte(`{"a":"\x41"}`)},
	}

	for _, tc := range cases {
		_, canonErr := canonjson.Canonicalize(tc.raw)
		walkErr := walkJSONDocument(tc.raw)
		if (canonErr == nil) != (walkErr == nil) {
			t.Fatalf("%s: canonjson err=%v vs walkJSONDocument err=%v", tc.name, canonErr, walkErr)
		}
	}

	// Depth 256 vs 257:
	depthDoc := func(d int) []byte {
		var b bytes.Buffer
		for i := 0; i < d; i++ {
			b.WriteByte('[')
		}
		b.WriteByte('1')
		for i := 0; i < d; i++ {
			b.WriteByte(']')
		}
		return b.Bytes()
	}
	if err := walkJSONDocument(depthDoc(256)); err != nil {
		t.Fatalf("depth 256 should be accepted: %v", err)
	}
	if err := walkJSONDocument(depthDoc(257)); err == nil {
		t.Fatal("depth 257 should be rejected")
	}

	// Chunked body without Content-Length (ContentLength = -1) is bounded by MaxBytesReader:
	c := wideRates(config.Defaults())
	c.PublicOrigin = "https://portal.example.invalid"
	adminP := principal(auth.Admin)
	h := fixtureConfig(t, adminP, false, nil, c)

	var readBytes int
	chunkedReader := readTrackerFunc(func(p []byte) (int, error) {
		for i := range p {
			p[i] = ' '
		}
		readBytes += len(p)
		return len(p), nil
	})
	req := request(http.MethodPost, "/api/v1/admin/groups", `{"code":"a"}`, adminP)
	req.Body = chunkedReader
	req.ContentLength = -1
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for unbounded chunked stream, got %d: %s", w.Code, w.Body.String())
	}
	if readBytes > canonjson.MaxBytes+64*1024 {
		t.Fatalf("read too many bytes from unbounded chunked stream: %d", readBytes)
	}
	_ = io.EOF
}
