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
