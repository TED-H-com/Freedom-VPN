//go:build e2e

package e2e

import (
	"context"
	"reflect"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay"
)

type DNSQueryTrace struct {
	Started    time.Time `json:"started"`
	Completed  time.Time `json:"completed"`
	Responder  string    `json:"responder"`
	CacheState string    `json:"cache_state"`
	// TraceComplete describes readable, contiguous lifecycle receipts, not the
	// absence of in-flight requests or proof of an internal AdGuard cache state.
	TraceComplete           bool                 `json:"trace_complete"`
	ResolverEvents          []dnsrelay.Event     `json:"resolver_events"`
	AuthorityEvents         []dnsrelay.Event     `json:"authority_events"`
	AuthorityHistory        []dnsrelay.Request   `json:"authority_history"`
	OutstandingAtStart      []string             `json:"authority_outstanding_at_query_start"`
	OutstandingAtCompletion []string             `json:"authority_outstanding_at_query_completion"`
	OutstandingAtRead       []string             `json:"authority_outstanding_at_trace_read"`
	TraceReadAt             map[string]time.Time `json:"trace_read_at"`
	AuthoritySerial         uint32               `json:"authority_serial_after_query"`
	AuthorityReadAt         time.Time            `json:"authority_read_at"`
	TTLRemaining            []uint32             `json:"answer_ttl_remaining"`
}

func (s *Stack) dnsQueryTrace(ctx context.Context, fixture string, rr uint16, started, completed time.Time, result FixtureProbeResult) *DNSQueryTrace {
	out := &DNSQueryTrace{Started: started, Completed: completed, Responder: "unattributed", CacheState: "unknown", TraceComplete: fixture != "", ResolverEvents: []dnsrelay.Event{}, AuthorityEvents: []dnsrelay.Event{}, AuthorityHistory: []dnsrelay.Request{}, OutstandingAtStart: []string{}, OutstandingAtCompletion: []string{}, OutstandingAtRead: []string{}, TraceReadAt: map[string]time.Time{}, TTLRemaining: result.TTLs}
	for _, role := range []string{"primary", "secondary", "authority"} {
		name := s.lab.prefix + "-origin-dns"
		if role == "secondary" {
			name += "-secondary"
		}
		if role == "authority" {
			name = s.lab.prefix + "-authority-dns"
		}
		raw, err := s.lab.docker(name, "cat", "/var/lib/fvpn-dns/events.jsonl")
		out.TraceReadAt[role] = time.Now().UTC()
		if err != nil {
			out.TraceComplete = false
			continue
		}
		history, complete := dnsrelay.History(raw)
		out.TraceComplete = out.TraceComplete && complete
		out.addHistory(role, fixture, rr, history)
	}
	// This separate post-query sample is never the cached positive A's serial.
	authority, err := s.fixtureProbe(ctx, s.lab.node, "-mode", "soa", "-addr", s.lab.labPrefix+".54:53")
	out.AuthorityReadAt = time.Now().UTC()
	if err == nil {
		out.AuthoritySerial = authority.AuthoritySerial
	}
	out.attribute(result)
	return out
}

func outstandingAt(r dnsrelay.Request, at time.Time) bool {
	return !r.Start.Started.After(at) && (r.Completion == nil || r.Completion.Completed.After(at))
}

func (out *DNSQueryTrace) addHistory(role, fixture string, rr uint16, history []dnsrelay.Request) {
	for _, r := range history {
		e := r.Start
		if e.Case != fixture || e.Type != rr {
			continue
		}
		if e.Role != role {
			out.TraceComplete = false
			continue
		}
		if role == "authority" {
			// Keep the whole matching sequence, including prior completed requests and
			// requests still outstanding after a client has abandoned a query.
			out.AuthorityHistory = append(out.AuthorityHistory, r)
			if r.Completion != nil {
				out.AuthorityEvents = append(out.AuthorityEvents, *r.Completion)
			}
			if outstandingAt(r, out.Started) {
				out.OutstandingAtStart = append(out.OutstandingAtStart, e.RequestID)
			}
			if outstandingAt(r, out.Completed) {
				out.OutstandingAtCompletion = append(out.OutstandingAtCompletion, e.RequestID)
			}
			if r.Completion == nil {
				out.OutstandingAtRead = append(out.OutstandingAtRead, e.RequestID)
			}
		} else if !e.Started.Before(out.Started) && !e.Started.After(out.Completed) {
			if r.Completion != nil {
				out.ResolverEvents = append(out.ResolverEvents, *r.Completion)
			} else {
				out.TraceComplete = false
			}
		}
	}
}

func (out *DNSQueryTrace) attribute(result FixtureProbeResult) {
	matches := []dnsrelay.Event{}
	for _, e := range out.ResolverEvents {
		if e.Success && !e.Completed.After(out.Completed) && e.RCode == result.RCode && reflect.DeepEqual(e.TTLs, result.TTLs) && reflect.DeepEqual(e.Classes, result.Classes) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		match := matches[0]
		out.Responder = match.Role
		if out.TraceComplete && len(result.Types) > 0 {
			out.CacheState = "inferred_hit_no_authority_exchange"
			for _, r := range out.AuthorityHistory {
				if r.Start.Source != match.Role {
					continue
				}
				if !r.Start.Started.Before(match.Started) && !r.Start.Started.After(match.Completed) {
					// A start proves a fetch, even if it outlives this client. It does not
					// prove which fetch supplied a response, or that the client fell back.
					out.CacheState = "fetch_authority_request_observed"
					return
				}
				if r.Start.Started.Before(match.Started) && outstandingAt(r, match.Started) {
					out.CacheState = "unknown_earlier_authority_request_inflight"
				}
			}
		}
	} else if len(matches) > 1 {
		out.Responder = "multiple_matching_responses"
	} else if out.TraceComplete && len(out.ResolverEvents) == 0 {
		out.Responder = "native_stub_no_upstream_observed"
		out.CacheState = "inferred_native_stub_hit_or_unobserved_path"
	}
}
