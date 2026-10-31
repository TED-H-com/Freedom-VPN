//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay"
)

func TestAT20_EarlierAuthorityCompletionAcrossQueries(t *testing.T) {
	base := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	at := func(ms int) time.Time { return base.Add(time.Duration(ms) * time.Millisecond) }
	result := FixtureProbeResult{RCode: 0, Types: []uint16{1}, TTLs: []uint32{9}, Classes: []string{"other_ipv4"}}
	old := dnsrelay.Event{RequestID: "old", Role: "authority", Source: "primary", Case: "cache", Type: 1, Started: at(0)}
	reply := old
	reply.Completed, reply.Success = at(3500), true
	current := dnsrelay.Event{Role: "primary", Case: "cache", Type: 1, Started: at(3000), Completed: at(3100), Success: true, TTLs: result.TTLs, Classes: result.Classes}
	trace := DNSQueryTrace{Started: at(2990), Completed: at(3110), TraceComplete: true, CacheState: "unknown"}
	trace.addHistory("authority", "cache", 1, []dnsrelay.Request{{Start: old}})
	trace.ResolverEvents = []dnsrelay.Event{current}
	trace.attribute(result)
	if trace.CacheState != "unknown_earlier_authority_request_inflight" || len(trace.OutstandingAtRead) != 1 || len(trace.OutstandingAtStart) != 1 {
		t.Fatalf("lost pending fetch: %+v", trace)
	}
	// Next query sees the same identity's late reply, even though its start and
	// completion are both outside the new query. This is the F9-INFLIGHT shape.
	next := DNSQueryTrace{Started: at(4000), Completed: at(4110), TraceComplete: true, CacheState: "unknown"}
	next.addHistory("authority", "cache", 1, []dnsrelay.Request{{Start: old, Completion: &reply}})
	current.Started, current.Completed = at(4010), at(4100)
	next.ResolverEvents = []dnsrelay.Event{current}
	next.attribute(result)
	if len(next.AuthorityEvents) != 1 || next.AuthorityEvents[0].RequestID != "old" || len(next.OutstandingAtRead) != 0 || next.CacheState != "inferred_hit_no_authority_exchange" {
		t.Fatalf("lost between-query completion: %+v", next)
	}
	// A different resolver's fetch is never a primary miss or proof of fallback.
	other := old
	other.RequestID, other.Source, other.Started = "secondary", "secondary", at(4020)
	next.addHistory("authority", "cache", 1, []dnsrelay.Request{{Start: other}})
	next.attribute(result)
	if next.CacheState != "inferred_hit_no_authority_exchange" {
		t.Fatal(next.CacheState)
	}
	next.TraceComplete = false
	next.CacheState = "unknown"
	next.attribute(result)
	if next.CacheState != "unknown" {
		t.Fatal("incomplete history inferred cache hit")
	}
}
