//go:build e2e

package members

import (
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
)

func TestAT20_F12DistributionIsNotSameResolverCacheFailure(t *testing.T) {
	at := time.Now().UTC()
	first := labstack.FixtureProbeResult{Classes: []string{"other_ipv4"}, TTLs: []uint32{10}, DNS: &labstack.DNSQueryTrace{TraceComplete: true, Responder: "primary", Completed: at}}
	next := labstack.FixtureProbeResult{Classes: []string{"private_ipv4"}, TTLs: []uint32{10}, DNS: &labstack.DNSQueryTrace{TraceComplete: true, Responder: "secondary", Started: at.Add(time.Second), Completed: at.Add(time.Second)}}
	v := attributeCache(first, next)
	if !strings.Contains(v["classification"].(string), "client distribution") || v["fallback"] != "not established" {
		t.Fatal(v)
	}
	next.DNS.Responder = "primary"
	v = attributeCache(first, next)
	if !strings.Contains(v["classification"].(string), "unexplained") || v["subsecond_expiry_possible"] != false {
		t.Fatal(v)
	}
	next.DNS.Completed = at.Add(9350 * time.Millisecond)
	v = attributeCache(first, next)
	if v["subsecond_expiry_possible"] != true {
		t.Fatal("F9-ROUND precision lost", v)
	}
	next.DNS.TraceComplete = false
	v = attributeCache(first, next)
	if v["classification"] != "ambiguous" {
		t.Fatal("incomplete history attributed", v)
	}
}

func TestAT20_F12NegativePrecisionAndOutstandingCompletion(t *testing.T) {
	at := time.Now().UTC()
	first := labstack.FixtureProbeResult{RCode: 3, HasSOA: true, AuthorityTTL: 2, DNS: &labstack.DNSQueryTrace{TraceComplete: true, Responder: "primary", Completed: at}}
	next := labstack.FixtureProbeResult{RCode: 0, Types: []uint16{1}, Classes: []string{"other_ipv4"}, DNS: &labstack.DNSQueryTrace{TraceComplete: true, Responder: "primary", Started: at.Add(1200 * time.Millisecond), Completed: at.Add(1250 * time.Millisecond)}}
	next.DNS.AuthorityHistory = []dnsrelay.Request{{Start: dnsrelay.Event{Source: "primary", Started: at.Add(1210 * time.Millisecond)}}}
	v := attributeCache(first, next)
	if v["subsecond_expiry_possible"] != true || v["fetch_in_next_query"] != true || v["fallback"] != "not established" {
		t.Fatal(v)
	}
	// A request begun between the two queries can also complete before query two.
	next.DNS.AuthorityHistory = []dnsrelay.Request{{Start: dnsrelay.Event{Source: "primary", Started: at.Add(100 * time.Millisecond)}, Completion: &dnsrelay.Event{Completed: at.Add(800 * time.Millisecond)}}}
	v = attributeCache(first, next)
	if !strings.Contains(v["classification"].(string), "older outstanding completion") || v["cause_status"] != "unproven" {
		t.Fatal(v)
	}
}
