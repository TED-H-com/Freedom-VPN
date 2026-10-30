//go:build e2e && integration

package secops

import (
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// §19 AT29/AT31, SECOPS-11: inspect the immutable product observation used by
// publication, never replace it with a harness-generated ready/healthy report.
func (p *liveProbe) publicationObservation(node uuid.UUID) map[string]any {
	var raw []byte
	err := p.db.QueryRow(p.t.Context(), `SELECT o.report_payload FROM node_observations o
JOIN nodes n ON n.id=o.node_id AND n.current_incarnation_id=o.incarnation_id AND n.last_observed_at=o.observed_at
WHERE n.id=$1 ORDER BY o.received_at DESC,o.id DESC LIMIT 1`, node).Scan(&raw)
	var report api.ObservationReport
	if err != nil || json.Unmarshal(raw, &report) != nil {
		p.t.Error("current immutable publication observation unavailable")
		return map[string]any{"available": false}
	}
	checks := []map[string]any{}
	for _, check := range report.Checks {
		checks = append(checks, map[string]any{"name": check.Name, "status": check.Status, "error_code": check.ErrorCode})
	}
	return map[string]any{"available": true, "report_id": report.ReportId, "observed_at": report.ObservedAt,
		"applied_config_generation": report.AppliedConfigGeneration, "member_forwarding_enabled": report.MemberForwardingEnabled,
		"lease": report.Lease, "checks": checks, "data_health": report.DataHealth, "safety_status": report.SafetyStatus,
		"provisioning_state": report.ProvisioningState, "forwarding_evidence_satisfied": nodes.PublicationForwardingEvidence(report, time.Now())}
}

// Compare the lease actually named in the immutable observation, not simply
// the newest lease central happened to issue (AT29 / agent follow-up 9).
func (p *liveProbe) observedLeaseGeneration(node uuid.UUID) map[string]any {
	var lease, applied, desired string
	var issued, expires time.Time
	err := p.db.QueryRow(p.t.Context(), `SELECT s.policy_generation::text,n.applied_config_generation::text,n.desired_config_generation::text,s.issued_at,s.expires_at
 FROM nodes n JOIN node_observations o ON o.node_id=n.id AND o.incarnation_id=n.current_incarnation_id AND o.observed_at=n.last_observed_at
 JOIN serving_leases s ON s.id=(o.report_payload->'lease'->>'lease_id')::uuid
 WHERE n.id=$1 ORDER BY o.received_at DESC LIMIT 1`, node).Scan(&lease, &applied, &desired, &issued, &expires)
	out := map[string]any{"read_ok": err == nil, "source": "lease ID from current immutable agent observation"}
	if err == nil {
		out["lease_generation"] = lease
		out["applied_generation"] = applied
		out["desired_generation"] = desired
		out["matches_applied"] = lease == applied
		out["issued_at"] = issued.UTC()
		out["expires_at"] = expires.UTC()
	}
	return out
}
