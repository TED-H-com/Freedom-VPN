package httpapi

import (
	"context"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/google/uuid"
)

// TestReviewR15_AnonymousReserveLeavesOneSlot is the D-083 cap. One hosting
// /48 admits at most min(max/2, 64) anonymous principal slots: 5 of 10 here,
// 64 at the production cap of 10000. Those slots are the anonymous class
// only (EnrollNode and ReceiveProviderWebhook). Login stays per IPv6 /64 on
// its own limiter. The agent bootstrap stays mTLS, keyed by the node URN.
func TestReviewR15_AnonymousReserveLeavesOneSlot(t *testing.T) {
	const maxPrincipals = 10
	prod := config.Defaults()
	if prod.MaxRatePrincipals != 10000 {
		t.Fatalf("production max_rate_principals = %d, want 10000", prod.MaxRatePrincipals)
	}
	if anonymousGroupCap(maxPrincipals) != 5 || anonymousGroupCap(prod.MaxRatePrincipals) != 64 {
		t.Fatalf("cap(10)=%d cap(10000)=%d, want 5 and 64", anonymousGroupCap(maxPrincipals), anonymousGroupCap(prod.MaxRatePrincipals))
	}
	now := time.Date(2026, 10, 30, 22, 26, 0, 0, time.UTC)
	setLimiterClock(t, func() time.Time { return now })
	l := newLimiter(maxPrincipals)
	b := config.Bucket{PerSecond: 1, Burst: 1}
	admitted := 0
	var group string
	for i := 0; i < maxPrincipals; i++ {
		ip := net.ParseIP(fmt.Sprintf("2001:db8:0:%x::", i))
		key, ok := rateKeyFromIP(ip)
		if !ok {
			t.Fatalf("rate key %v", ip)
		}
		group = anonymousGroup(key)
		if group == "" {
			t.Fatalf("no /48 group for %s", key)
		}
		if l.allowGrouped(key, group, b) == 0 {
			admitted++
		}
	}
	if admitted != 5 || len(l.entries) != 5 {
		t.Fatalf("one /48 admitted %d of %d anonymous principal slots (%d entries); want 5", admitted, maxPrincipals, len(l.entries))
	}
	other, ok := rateKeyFromIP(net.ParseIP("2001:db8:1::"))
	if !ok || anonymousGroup(other) == group {
		t.Fatal("other /48")
	}
	if d := l.allowGrouped(other, anonymousGroup(other), b); d != 0 {
		t.Fatalf("other /48 refused: %s", d)
	}
	if d := l.allowGrouped("198.51.100.10", "", b); d != 0 {
		t.Fatalf("IPv4 refused: %s", d)
	}
	if len(l.entries) != 7 {
		t.Fatalf("table %d, want 7 (5 in the hosting /48, one other /48, one IPv4)", len(l.entries))
	}
}

// Exercise actual admission at the absolute cap and small/odd table sizes,
// including the fail-closed max=1 case required by integer division in D-083.
func TestReviewR15_AnonymousCapBoundariesAndReclamation(t *testing.T) {
	for _, tc := range []struct{ max, want int }{{1, 0}, {2, 1}, {11, 5}, {127, 63}, {128, 64}, {129, 64}, {10000, 64}} {
		t.Run(fmt.Sprintf("max_%d", tc.max), func(t *testing.T) {
			now := time.Date(2026, 10, 30, 22, 26, 0, 0, time.UTC)
			setLimiterClock(t, func() time.Time { return now })
			l := newLimiter(tc.max)
			b := config.Bucket{PerSecond: 1, Burst: 2}
			key := func(n int) string {
				k, ok := rateKeyFromIP(net.ParseIP(fmt.Sprintf("2001:db8:0:%x::1", n)))
				if !ok {
					t.Fatal("invalid fixture")
				}
				return k
			}
			group := anonymousGroup(key(0))
			for i := 0; i < tc.want; i++ {
				if d := l.allowGrouped(key(i), group, b); d != 0 {
					t.Fatalf("slot %d: %s", i, d)
				}
			}
			if d := l.allowGrouped(key(tc.want), group, b); d != time.Second || len(l.entries) != tc.want || l.groups[group] != tc.want {
				t.Fatalf("past cap: delay=%s entries=%d group=%d want=%d", d, len(l.entries), l.groups[group], tc.want)
			}
			if tc.want == 0 {
				return
			}
			// Existing keys spend their remaining token, then are limited.
			if d := l.allowGrouped(key(0), group, b); d != 0 {
				t.Fatal(d)
			}
			if d := l.allowGrouped(key(0), group, b); d != time.Second {
				t.Fatal(d)
			}
			now = now.Add(2 * time.Hour)
			if d := l.allowGrouped(key(tc.want), group, b); d != 0 || len(l.entries) != tc.want || l.groups[group] != tc.want {
				t.Fatal("idle group slot was not replaced at sparse table occupancy", d, len(l.entries))
			}
			// Global table eviction releases a group slot for a new /64.
			now = now.Add(2 * time.Hour)
			if !l.evictIdle(now) || l.groups[group] != tc.want-1 {
				t.Fatal("eviction did not release group slot")
			}
			if d := l.allowGrouped(key(tc.want+1), group, b); d != 0 || l.groups[group] != tc.want {
				t.Fatal("released group slot cannot be reused", d, l.groups[group])
			}
		})
	}
}

func TestReviewR15_AnonymousCapNeverEvictsAnotherGroup(t *testing.T) {
	now := time.Date(2026, 10, 30, 22, 26, 0, 0, time.UTC)
	setLimiterClock(t, func() time.Time { return now })
	l := newLimiter(4)
	b := config.Bucket{PerSecond: 1, Burst: 1}
	if d := l.allowGrouped("other", "other-group", b); d != 0 {
		t.Fatal(d)
	}
	now = now.Add(time.Second) // The other group's bucket is idle-full.
	for _, key := range []string{"first", "second"} {
		if d := l.allowGrouped(key, "capped-group", b); d != 0 {
			t.Fatal(d)
		}
	}
	if d := l.allowGrouped("new", "capped-group", b); d != time.Second || len(l.entries) != 3 || l.entries["other"] == nil {
		t.Fatal("at-cap group evicted another group's idle bucket", d)
	}
	now = now.Add(time.Second)
	if d := l.allowGrouped("new", "capped-group", b); d != 0 || len(l.entries) != 3 || l.groups["capped-group"] != 2 || l.entries["other"] == nil {
		t.Fatal("same-group replacement changed other group or exceeded cap", d)
	}
}

func TestReviewR15_AnonymousCapOnEnrollAndWebhook(t *testing.T) {
	for _, path := range []string{"/api/v1/agent/enroll", "/api/v1/webhooks/example"} {
		t.Run(path, func(t *testing.T) {
			now := time.Date(2026, 10, 30, 22, 26, 0, 0, time.UTC)
			setLimiterClock(t, func() time.Time { return now })
			c := config.Defaults()
			c.MaxRatePrincipals = 10
			c.TrustedProxies = []string{"192.0.2.9/32"}
			h := fixtureConfig(t, auth.Principal{}, false, nil, c)
			invoke := func(ip string, forwarded bool) *httptest.ResponseRecorder {
				r := request(http.MethodPost, path, string(schemaEnroll()), auth.Principal{})
				r.RemoteAddr = net.JoinHostPort(ip, "443")
				if forwarded {
					r.RemoteAddr = "192.0.2.9:443"
					r.Header.Set("X-Forwarded-For", ip)
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				return w
			}
			for i := 0; i < 5; i++ {
				if w := invoke(fmt.Sprintf("2001:db8:0:%x::1", i), i%2 == 0); w.Code != http.StatusNotImplemented {
					t.Fatalf("slot %d: %d %s", i, w.Code, w.Body.String())
				}
			}
			w := invoke("2001:0db8:0000:0005:0000:0000:0000:0001", true)
			if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "1" || !strings.Contains(w.Body.String(), `"code":"RATE_LIMITED"`) {
				t.Fatalf("sixth /64: %d %s %s", w.Code, w.Header().Get("Retry-After"), w.Body.String())
			}
			for _, ip := range []string{"2001:db8:0:0::2", "2001:db8:1::1", "198.51.100.10"} {
				if w := invoke(ip, false); w.Code != http.StatusNotImplemented {
					t.Fatalf("existing /64 or unrelated principal %s refused: %d %s", ip, w.Code, w.Body.String())
				}
			}
		})
	}
}

// TestReviewR15_EvictIdleFullTableIsAShortWalk times one refused newcomer
// against a table of max_rate_principals depleted keys. The walk is the
// whole table.
func TestReviewR15_EvictIdleFullTableIsAShortWalk(t *testing.T) {
	now := time.Date(2026, 10, 30, 19, 25, 0, 0, time.UTC)
	setLimiterClock(t, func() time.Time { return now })
	const maxPrincipals = 10000
	if config.Defaults().MaxRatePrincipals != maxPrincipals {
		t.Fatalf("production max_rate_principals = %d, want %d", config.Defaults().MaxRatePrincipals, maxPrincipals)
	}
	l := newLimiter(maxPrincipals)
	dead := config.Bucket{PerSecond: 0, Burst: 1}
	for i := 0; i < maxPrincipals; i++ {
		if d := l.allow(fmt.Sprintf("depleted-%d", i), dead); d != 0 {
			t.Fatalf("fill %d: %s", i, d)
		}
	}
	if len(l.entries) != maxPrincipals {
		t.Fatalf("table %d/%d", len(l.entries), maxPrincipals)
	}
	start := time.Now()
	d := l.allow("newcomer", config.Bucket{PerSecond: 1000, Burst: 5})
	elapsed := time.Since(start)
	t.Logf("evictIdle at max_rate_principals=%d depleted keys: refused in %s (wait %s)", maxPrincipals, elapsed, d)
	if d == 0 {
		t.Fatal("newcomer was inserted into a full table of live depleted keys")
	}
	if len(l.entries) != maxPrincipals {
		t.Fatalf("refusal changed the table: %d", len(l.entries))
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("one refused newcomer walked %d depleted keys in %s", maxPrincipals, elapsed)
	}
}

// TestReviewR15_UnauthenticatedAgentDoesNotFillTheIngressTable sends agent
// requests with no verified leaf. The listener returns 401 before allow, so
// those clients cannot force evictIdle and cannot spend a node's burst.
func TestReviewR15_UnauthenticatedAgentDoesNotFillTheIngressTable(t *testing.T) {
	c := config.Defaults()
	c.IngressRate = config.Bucket{PerSecond: 0.0001, Burst: 2}
	c.MaxConcurrent = 8
	node := uuid.Must(uuid.NewV7())
	inc := uuid.Must(uuid.NewV7())
	h := reviewAgentHandler(t, c, auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) {
		return auth.Principal{Role: auth.Agent, ID: node, NodeID: node, IncarnationID: inc}, nil
	}))
	for i := 0; i < 30; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/agent/lease-renewals", strings.NewReader(`{}`))
		r.RemoteAddr = fmt.Sprintf("198.51.100.%d:40000", (i%200)+1)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated request %d status %d, want 401 before the ingress budget", i, w.Code)
		}
	}
	leaf := reviewNodeLeaf([]byte("still-admitted"), node, inc)
	if got := reviewAgentPOST(h, leaf); got.Code == http.StatusTooManyRequests || got.Code == http.StatusUnauthorized {
		t.Fatalf("a fresh leaf was refused (%d) after unauthenticated requests; those requests must not spend the node bucket or fill the table", got.Code)
	}
}
