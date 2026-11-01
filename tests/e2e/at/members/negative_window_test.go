//go:build e2e

package members

import (
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"testing"
	"time"
)

func negativeCacheWindow(first labstack.FixtureProbeResult) map[string]any {
	out := map[string]any{"available": false, "remaining_soa_ttl_seconds": first.AuthorityTTL, "precision_reserve_seconds": 1}
	d := first.DNS
	if d == nil || !d.TraceComplete || (d.Responder != "primary" && d.Responder != "secondary") || !first.HasSOA || first.RCode != 3 || first.AuthorityTTL <= 1 {
		return out
	}
	out["available"] = true
	out["receipt_at"] = d.Completed
	out["earliest_expiry"] = d.Completed.Add(time.Duration(first.AuthorityTTL-1) * time.Second)
	return out
}

func negativeWindowContains(first, next labstack.FixtureProbeResult) bool {
	w := negativeCacheWindow(first)
	if w["available"] != true || next.DNS == nil || !next.DNS.TraceComplete || first.DNS.Responder != next.DNS.Responder {
		return false
	}
	if !next.DNS.Started.After(first.DNS.Completed) || !next.DNS.Completed.Before(w["earliest_expiry"].(time.Time)) {
		return false
	}
	for _, req := range next.DNS.AuthorityHistory {
		if req.Start.Source == next.DNS.Responder && req.Start.Started.Before(next.DNS.Started) && (req.Completion == nil || req.Completion.Completed.After(first.DNS.Completed)) {
			return false
		}
	}
	return true
}

func TestAT20_F13NegativeWindowReservesTTLPrecision(t *testing.T) {
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	first := labstack.FixtureProbeResult{RCode: 3, HasSOA: true, AuthorityTTL: 2, DNS: &labstack.DNSQueryTrace{TraceComplete: true, Responder: "primary", Completed: at}}
	next := labstack.FixtureProbeResult{DNS: &labstack.DNSQueryTrace{TraceComplete: true, Responder: "primary", Started: at.Add(100 * time.Millisecond), Completed: at.Add(500 * time.Millisecond)}}
	if !negativeWindowContains(first, next) {
		t.Fatal("inside conservative window rejected")
	}
	next.DNS.Completed = at.Add(1228672687 * time.Nanosecond)
	if negativeWindowContains(first, next) {
		t.Fatal("F12-03 expiry uncertainty accepted")
	}
	next.DNS.Completed = at.Add(500 * time.Millisecond)
	next.DNS.Responder = "secondary"
	if negativeWindowContains(first, next) {
		t.Fatal("different cache accepted")
	}
}
