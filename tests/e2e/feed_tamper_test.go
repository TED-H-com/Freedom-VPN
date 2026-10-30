//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func (r *runner) threatFeedTamper(ctx context.Context, admin *session) {
	rec := Record{ID: "threat-feed-tamper", Title: "Corrupted resolver response retains node last-good and recovers", Spec: []string{"§12.5", "§19 AT22"}, Status: StatusBlocked}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	note("positive_node_delivery", boolText(r.dnsApplied))
	if !r.dnsApplied {
		note("owner", "w2-agent-core / w3-wire-nodes: positive resolver delivery prerequisite")
		return
	}
	before, _ := r.lab.docker(r.lab.node, "fvpn-e2e-nodeconfig", "dns-probe")
	note("last_good_before", strings.TrimSpace(string(before)))
	marker := filepath.Join(r.lab.state, "node", "var", "e2e", "corrupt-resolver")
	if err := atomicHook(marker, []byte("*\n")); err != nil {
		rec.Status = StatusFail
		note("fault_hook", "could not arm")
		return
	}
	defer os.Remove(marker)
	// Refresh the approved source through its authenticated lab origin. The
	// production worker owns cadence, compilation and signed-reference issuance.
	raw, err := os.ReadFile(filepath.Join(r.lab.state, "releases", "feeds", "acceptance.txt"))
	if err != nil {
		rec.Status = StatusFail
		return
	}
	raw = append(raw, []byte("0.0.0.0 feed-11.example.invalid\n")...)
	if err = r.lab.putObject(ctx, "feeds/acceptance.txt", raw); err != nil {
		rec.Status = StatusFail
		note("feed_upload", "failed")
		return
	}
	note("source_change", "approved source adds one synthetic rule; wait for its configured 300-second refresh cadence")
	poll := func(timeout time.Duration, match func(*api.DnsPolicyEvidence) bool) bool {
		wait, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return waitUntil(wait, time.Second, func() bool {
			resp, e := admin.call(wait, "GET", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), nil, nil)
			var node struct {
				Evidence *api.DnsPolicyEvidence `json:"dns_policy_evidence"`
			}
			return e == nil && resp.Status == 200 && json.Unmarshal(resp.Body, &node) == nil && node.Evidence != nil && match(node.Evidence)
		})
	}
	var failed *api.DnsPolicyEvidence
	rejected := poll(360*time.Second, func(ev *api.DnsPolicyEvidence) bool {
		code, _ := ev.ErrorCode.Get()
		if ev.Outcome == api.DnsPolicyEvidenceOutcomeFailed && code == api.ErrorCodeHASHMISMATCH {
			failed = ev
			return true
		}
		return false
	})
	fault, _ := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/e2e/resolver-fault.json")
	var injection struct {
		Injected   bool   `json:"fault_injected"`
		DocumentID string `json:"document_id"`
	}
	_ = json.Unmarshal(fault, &injection)
	note("fault_injected", boolText(injection.Injected))
	note("transport_fault", strings.TrimSpace(string(fault)))
	evRaw, _ := json.Marshal(failed)
	note("rejected_evidence", string(evRaw))
	after, _ := r.lab.docker(r.lab.node, "fvpn-e2e-nodeconfig", "dns-probe")
	note("last_good_during_fault", strings.TrimSpace(string(after)))
	same := strings.TrimSpace(string(before)) == strings.TrimSpace(string(after)) && len(before) > 0
	note("last_good_unchanged", boolText(same))
	if err = os.Remove(marker); err != nil {
		rec.Status = StatusFail
		note("fault_hook", "could not disarm")
		return
	}
	var recovered *api.DnsPolicyEvidence
	ok := poll(90*time.Second, func(ev *api.DnsPolicyEvidence) bool {
		if failed != nil && ev.DocumentId == failed.DocumentId && ev.Outcome == api.DnsPolicyEvidenceOutcomeApplied {
			recovered = ev
			return true
		}
		return false
	})
	evRaw, _ = json.Marshal(recovered)
	note("recovered_evidence", string(evRaw))
	probe, _ := r.lab.docker(r.lab.node, "fvpn-e2e-nodeconfig", "dns-probe")
	note("resolver_after_recovery", strings.TrimSpace(string(probe)))
	var result struct {
		Count   int  `json:"rule_count"`
		Blocked bool `json:"synthetic_blocked"`
	}
	valid := json.Unmarshal(probe, &result) == nil && result.Count >= 11 && result.Blocked
	if injection.Injected && rejected && same && ok && valid {
		rec.Status = StatusPass
		return
	}
	note("owner", "w2-agent-core / w3-wire-nodes / w3-dns: inspect transport fault, projected rejection and recovery; no desired state or cache was forged")
}
