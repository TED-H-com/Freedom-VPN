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

// TestReviewR15_AnonymousReserveLeavesOneSlot records the capacity one
// hosting /48 can take after anonymousGroupReserve replaced the half cap.
// Production max_rate_principals is 10000, so the same rule holds 9999
// anonymous principal slots. Login is a separate /64 limiter and is not
// in this table. The anonymous class is enrollNode and ReceiveProviderWebhook.
func TestReviewR15_AnonymousReserveLeavesOneSlot(t *testing.T) {
	const maxPrincipals = 10
	if anonymousGroupReserve != 1 {
		t.Fatalf("anonymous reserve is %d, this proof is written for 1", anonymousGroupReserve)
	}
	prod := config.Defaults()
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
		if l.allowGrouped(key, group, b) == 0 {
			admitted++
		}
	}
	half := maxPrincipals / 2
	prodHold := prod.MaxRatePrincipals - anonymousGroupReserve
	if admitted > half {
		t.Fatalf("TODO(review): anonymousGroupReserve=%d lets one hosting /48 hold %d of %d anonymous principal slots (production max_rate_principals %d holds %d). The previous cap was half the table (%d here, %d in production). Those slots are the anonymous class only: EnrollNode and ReceiveProviderWebhook. Login is keyed per IPv6 /64 on its own limiter, and the agent bootstrap is mTLS keyed by the node URN. One /48 can therefore take every enrollment and provider-webhook slot but one, and every other /48 shares the single leftover. Recommend the previous half cap (max/2) or a small absolute cap such as 64.",
			anonymousGroupReserve, admitted, maxPrincipals, prod.MaxRatePrincipals, prodHold, half, prod.MaxRatePrincipals/2)
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
		t.Fatalf("TODO(review): one refused newcomer walked %d depleted keys in %s", maxPrincipals, elapsed)
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
