//go:build e2e

package members

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/dnsrelay"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
)

const dnsCacheContract = "docs/dns/BACKEND.md §2.1.5: F9-HIT / F9-OTHER / F9-ROUND / F9-INFLIGHT"

func (l *live) cacheSettings() map[string]any {
	out := map[string]any{}
	for _, role := range []string{"adguard", "node2-adguard"} {
		cmd := exec.CommandContext(l.ctx, "docker", "exec", "-i", l.name(role), "/tmp/at-members-probe", "http", "/control/dns_info")
		cmd.Stdin = strings.NewReader(l.secret("adguard.password"))
		raw, err := cmd.Output()
		var full map[string]json.RawMessage
		ok := err == nil && json.Unmarshal(raw, &full) == nil
		safe := map[string]any{"available": ok, "read_at": time.Now().UTC()}
		for _, k := range []string{"cache_size", "cache_ttl_min", "cache_ttl_max", "cache_optimistic", "upstream_mode", "upstream_timeout", "fallback_dns", "blocking_mode", "blocked_response_ttl"} {
			if v, has := full[k]; has {
				safe[k] = v
			}
		}
		out[role] = safe
	}
	return out
}

// Attribution never changes the original assertion's result. A unique responder
// and complete lifecycle receipts are prerequisites, not a private cache flag.
func attributeCache(first, next labstack.FixtureProbeResult) map[string]any {
	out := map[string]any{"contract": dnsCacheContract, "cause_status": "unproven", "classification": "ambiguous", "fallback": "not established", "assertion_scope": "native client distribution; same-resolver assertion only with a unique matching responder"}
	a, b := first.DNS, next.DNS
	if a == nil || b == nil || !a.TraceComplete || !b.TraceComplete {
		return out
	}
	unique := func(s string) bool { return s == "primary" || s == "secondary" }
	if !unique(a.Responder) || !unique(b.Responder) {
		return out
	}
	out["initial_responder"], out["next_responder"] = a.Responder, b.Responder
	out["elapsed_query_completion_ms"] = b.Completed.Sub(a.Completed).Milliseconds()
	if a.Responder != b.Responder {
		out["classification"] = "client distribution across independent caches (F9-OTHER); fallback requires failure/timeout plus secondary receipts"
		return out
	}
	out["assertion_scope"] = "same uniquely matched resolver"
	ttl := first.AuthorityTTL
	if len(first.TTLs) > 0 {
		ttl = first.TTLs[0]
	}
	if ttl > 0 {
		// A receipt TTL is whole seconds; completion of a slow query is not its start.
		boundary := a.Completed.Add(time.Duration(ttl)*time.Second - time.Second)
		out["subsecond_expiry_possible"] = !b.Completed.Before(boundary)
		out["earliest_whole_second_expiry"] = boundary
	}
	late := []any{}
	fetch := false
	for _, request := range b.AuthorityHistory {
		if request.Start.Source != b.Responder {
			continue
		}
		if !request.Start.Started.Before(b.Started) && !request.Start.Started.After(b.Completed) {
			fetch = true
		}
		if request.Start.Started.Before(b.Started) && request.Completion != nil && request.Completion.Completed.After(a.Completed) && !request.Completion.Completed.After(b.Completed) {
			late = append(late, request)
		}
	}
	out["older_outstanding_completions"] = late
	out["fetch_in_next_query"] = fetch
	same := reflect.DeepEqual(first.Classes, next.Classes)
	switch {
	case len(late) > 0:
		out["classification"] = "same-resolver answer with older outstanding completion (F9-INFLIGHT candidate); insertion order not directly observed"
	case fetch:
		out["classification"] = "same-resolver miss/refetch; inspect F9-ROUND, eviction, invalidation and cache key; not fallback"
	case same && len(first.Types) == 0:
		out["classification"] = "negative response without new authority fetch; cache state remains unknown (local synthesis is separate)"
	case same:
		out["classification"] = "same-resolver inferred hit without new authority fetch (F9-HIT)"
	default:
		out["classification"] = "changed same-resolver answer without new fetch; unexplained pending full lifecycle/invalidation/cache-key evidence"
	}
	return out
}

func (l *live) cacheArtifact(value any) {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		l.t.Fatal(err)
	}
	if err = writeEvidence(filepath.Join(l.root, "docs/e2e/at/members/F15-AT20-cache.json"), append(raw, '\n')); err != nil {
		l.t.Fatal(err)
	}
}

// Both resolver histories include prior/completing requests; a successful
// relay write is never called proof of client receipt. No query retry here.
func (l *live) dnsLifecycle(label string) map[string]any {
	out := map[string]any{}
	for role, slot := range map[string]string{"primary": "origin-dns", "secondary": "origin-dns-secondary", "authority": "authority-dns"} {
		dir := filepath.Join(l.root, ".agent/tmp/f15-dns", l.runID, label)
		if os.MkdirAll(dir, 0700) != nil {
			out[role] = map[string]any{"available": false}
			continue
		}
		path := filepath.Join(dir, role+".jsonl")
		f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			out[role] = map[string]any{"available": false}
			continue
		}
		hash := sha256.New()
		cmd := exec.CommandContext(l.ctx, "docker", "exec", l.name(slot), "cat", "/var/lib/fvpn-dns/events.jsonl")
		cmd.Stdout = io.MultiWriter(f, hash)
		e = cmd.Run()
		closeErr := f.Close()
		raw, readErr := os.ReadFile(path)
		history, complete := dnsrelay.History(raw)
		pending := []string{}
		for _, q := range history {
			if q.Completion == nil {
				pending = append(pending, q.Start.RequestID)
			}
		}
		out[role] = map[string]any{"available": e == nil && closeErr == nil && readErr == nil, "complete": complete, "read_at": time.Now().UTC(), "bytes": fmt.Sprint(len(raw)), "sha256": fmt.Sprintf("%x", hash.Sum(nil)), "history": history, "outstanding_at_read": pending}
	}
	return out
}

func (l *live) originalDNSArtifact(r *record, original *labstack.FixtureProbeResult, capture json.RawMessage, before, after map[string]any) {
	value := map[string]any{"run_id": l.runID, "source_commit": l.build.Source, "original_query": original, "native_capture": capture, "lifecycle_before": before, "lifecycle_after": after, "cause_status": "unproven", "historical_F12_04": "5,001 ms SERVFAIL remains failed and unattributed; this is the new original query, never a replacement"}
	raw, e := json.MarshalIndent(value, "", "  ")
	if e != nil || writeEvidence(filepath.Join(l.root, "docs/e2e/at/members/F15-F12-04-original-dns.json"), append(raw, '\n')) != nil {
		l.t.Fatal("write original DNS evidence")
	}
	r.jsonCheck("F12-04 original query evidence", "original one-shot RR query and both resolver lifecycles retained; capture completion is separate", value, "StartNativeDNSCapture after two-resolver configuration, before readiness and original RR/listener/cache sequence", original != nil && before != nil && after != nil && capture != nil)
}
