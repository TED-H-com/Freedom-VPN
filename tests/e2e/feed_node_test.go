//go:build e2e

package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func (r *runner) threatFeedNode(ctx context.Context, admin *session) {
	rec := Record{ID: "threat-feed-node", Title: "Issued resolver document is applied and blocks a synthetic name", Spec: []string{"§12.5", "§19 AT22", "D-061"}, Status: StatusBlocked}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	mode, _ := r.lab.docker(r.lab.node, "stat", "-c", "%u:%a", "/etc/vpn-node/resolver-credentials.json")
	note("credential_owner_mode", strings.TrimSpace(string(mode)))
	// Record the requested runtime object only: never inline credential contents.
	cfgRaw, _ := r.lab.docker(r.lab.node, "cat", "/etc/vpn-node/agent-run.json")
	var cfg struct {
		DNS json.RawMessage `json:"dns"`
	}
	_ = json.Unmarshal(cfgRaw, &cfg)
	note("agent_run_dns", string(cfg.DNS))
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var ref api.ResolverDocumentRef
	var evidence *api.DnsPolicyEvidence
	var provisioning string
	var status int
	applied := waitUntil(wait, time.Second, func() bool {
		var desired struct {
			Signed struct {
				DNS struct {
					Ref api.ResolverDocumentRef `json:"resolver_document"`
				} `json:"dns"`
			}
		}
		raw := r.desiredRaw()
		if r.lab.verifyDesired(raw) != nil || json.Unmarshal(raw, &desired) != nil {
			return false
		}
		ref = desired.Signed.DNS.Ref
		detail, err := admin.call(wait, "GET", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), nil, nil)
		status = detail.Status
		var node struct {
			Evidence     *api.DnsPolicyEvidence `json:"dns_policy_evidence"`
			Provisioning string                 `json:"provisioning_state"`
		}
		if err != nil || status != 200 || json.Unmarshal(detail.Body, &node) != nil {
			return false
		}
		evidence, provisioning = node.Evidence, node.Provisioning
		return evidence != nil && evidence.Outcome == api.DnsPolicyEvidenceOutcomeApplied && evidence.DocumentId == ref.DocumentId && evidence.Generation == ref.Generation && evidence.ContentSha256 == ref.ContentSha256 && provisioning == "ready"
	})
	refRaw, _ := json.Marshal(ref)
	note("desired_resolver_reference", string(refRaw))
	evRaw, _ := json.Marshal(evidence)
	note("dns_policy_evidence", string(evRaw))
	note("admin_node", itoa(status))
	note("provisioning_state", provisioning)
	// Fetch with this node's actual mTLS identity and compare exact bytes, size
	// and signed digest. Rules and response bodies are never printed in evidence.
	headers, fetchErr := r.lab.docker(r.lab.node, "wget", "--server-response", "--timeout=5", "--tries=1", "-O", "/var/lib/vpn-node/e2e/resolver-response.json", "--ca-certificate=/etc/vpn-node/ca.pem", "--certificate=/var/lib/vpn-node/identity/cert.pem", "--private-key=/var/lib/vpn-node/identity/key.pem", r.lab.agent+"/api/v1/agent/resolver-documents/"+ref.DocumentId.String())
	for _, line := range strings.Split(string(headers), "\n") {
		if strings.Contains(line, "HTTP/") {
			note("resolver_document_response", strings.TrimSpace(line))
		}
	}
	body, bodyErr := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/e2e/resolver-response.json")
	sum := sha256.Sum256(body)
	verified := fetchErr == nil && bodyErr == nil && len(body) == ref.SizeBytes && hex.EncodeToString(sum[:]) == ref.ContentSha256
	note("mtls_body_matches_signed_digest_and_size", boolText(verified))
	probe, probeErr := r.lab.docker(r.lab.node, "fvpn-e2e-nodeconfig", "dns-probe")
	note("resolver_readback", strings.TrimSpace(string(probe)))
	var result struct {
		Blocked bool `json:"synthetic_blocked"`
		Rules   int  `json:"rule_count"`
		Queries bool `json:"query_log_enabled"`
		Files   bool `json:"query_log_file_enabled"`
		Stats   bool `json:"statistics_enabled"`
	}
	safe := probeErr == nil && json.Unmarshal(probe, &result) == nil && result.Blocked && result.Rules >= 10 && !result.Queries && !result.Files && !result.Stats
	r.dnsApplied = applied && verified && safe && strings.TrimSpace(string(mode)) == "0:600"
	if r.dnsApplied {
		rec.Status = StatusPass
		return
	}
	note("owner", "w2-agent-core / w3-wire-nodes: inspect issued reference, accepted DNS evidence and provisioning above; lab resolver config and HTTP readback are recorded")
}
