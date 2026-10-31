//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay"
	"reflect"
	"time"
)

type DNSQueryTrace struct {
	Started         time.Time        `json:"started"`
	Completed       time.Time        `json:"completed"`
	Responder       string           `json:"responder"`
	CacheState      string           `json:"cache_state"`
	TraceComplete   bool             `json:"trace_complete"`
	ResolverEvents  []dnsrelay.Event `json:"resolver_events"`
	AuthorityEvents []dnsrelay.Event `json:"authority_events"`
	AuthoritySerial uint32           `json:"authority_serial_after_query"`
	AuthorityReadAt time.Time        `json:"authority_read_at"`
	TTLRemaining    []uint32         `json:"answer_ttl_remaining"`
}

func (s *Stack) dnsQueryTrace(ctx context.Context, fixture string, rr uint16, started, completed time.Time, result FixtureProbeResult) *DNSQueryTrace {
	out := &DNSQueryTrace{Started: started, Completed: completed, Responder: "unattributed", CacheState: "unknown", TraceComplete: true, ResolverEvents: []dnsrelay.Event{}, AuthorityEvents: []dnsrelay.Event{}, TTLRemaining: result.TTLs}
	for _, role := range []string{"primary", "secondary", "authority"} {
		name := s.lab.prefix + "-origin-dns"
		if role == "secondary" {
			name += "-secondary"
		}
		if role == "authority" {
			name = s.lab.prefix + "-authority-dns"
		}
		raw, err := s.lab.docker(name, "cat", "/var/lib/fvpn-dns/events.jsonl")
		if err != nil {
			out.TraceComplete = false
			continue
		}
		scan := bufio.NewScanner(bytes.NewReader(raw))
		for scan.Scan() {
			var e dnsrelay.Event
			if json.Unmarshal(scan.Bytes(), &e) != nil {
				out.TraceComplete = false
				continue
			}
			if e.Case != fixture || e.Type != rr || e.Started.Before(started) || e.Started.After(completed) {
				continue
			}
			if role == "authority" {
				out.AuthorityEvents = append(out.AuthorityEvents, e)
			} else {
				out.ResolverEvents = append(out.ResolverEvents, e)
			}
		}
		if scan.Err() != nil {
			out.TraceComplete = false
		}
	}
	// Authority serial is measured separately and after the native query. It is
	// not attached retrospectively to a positive cached record that has no SOA.
	authority, e := s.fixtureProbe(ctx, s.lab.node, "-mode", "soa", "-addr", s.lab.labPrefix+".54:53")
	out.AuthorityReadAt = time.Now().UTC()
	if e == nil {
		out.AuthoritySerial = authority.AuthoritySerial
	}
	matches := []dnsrelay.Event{}
	for _, e := range out.ResolverEvents {
		if e.Success && e.RCode == result.RCode && reflect.DeepEqual(e.TTLs, result.TTLs) && reflect.DeepEqual(e.Classes, result.Classes) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		out.Responder = matches[0].Role
		out.CacheState = "unknown"
		if out.TraceComplete && len(result.Types) > 0 {
			out.CacheState = "inferred_hit_no_authority_exchange"
			for _, e := range out.AuthorityEvents {
				if !e.Started.Before(matches[0].Started) && !e.Started.After(matches[0].Completed) {
					out.CacheState = "miss_authority_exchange_observed"
				}
			}
		}
	} else if len(matches) > 1 {
		out.Responder = "multiple_matching_responses"
	} else if out.TraceComplete && len(out.ResolverEvents) == 0 {
		out.Responder = "native_stub_no_upstream_observed"
		out.CacheState = "inferred_native_stub_hit_or_unobserved_path"
	}
	return out
}
