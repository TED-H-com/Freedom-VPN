//go:build e2e

package members

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Read-only snapshots accompany original mutations. No hold, alarm, member
// revocation or recovery is manufactured to populate a missing coverage row.
func (l *live) holdSnapshot(label string) {
	var raw json.RawMessage
	err := l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object(
	 'incidents',coalesce((SELECT jsonb_agg(to_jsonb(h) ORDER BY h.first_seen_at,h.id) FROM node_generation_holds h),'[]'::jsonb),
	 'members',coalesce((SELECT jsonb_agg(jsonb_build_object('member_id',m.id,'status',m.status,'admin_suspended',m.admin_suspended,'updated_at',m.updated_at,'entitlement_state',e.state,'effective_service_end',e.effective_service_end) ORDER BY m.id) FROM members m LEFT JOIN entitlements e ON e.member_id=m.id),'[]'::jsonb))`).Scan(&raw)
	l.holdSnapshots = append(l.holdSnapshots, map[string]any{"label": label, "at": time.Now().UTC(), "available": err == nil, "snapshot": raw})
}

func (l *live) recordGenerationHolds(r *record) {
	l.holdSnapshot(r.ID + " complete")
	var joined json.RawMessage
	err := l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object(
	'incidents',coalesce((SELECT jsonb_agg(to_jsonb(h) ORDER BY h.first_seen_at,h.id) FROM node_generation_holds h),'[]'::jsonb),
	'samples',coalesce((SELECT jsonb_agg((to_jsonb(s)-'applied_generation'-'lease_generation'-'reported_steps'-'reported_elapsed_ms') || jsonb_build_object('applied_generation',s.applied_generation::text,'lease_generation',s.lease_generation::text,'reported_steps',s.reported_steps::text,'reported_elapsed_ms',s.reported_elapsed_ms::text,'received_at',o.received_at,'observation',o.report_payload,'leased_member_ids',(SELECT coalesce(jsonb_agg(m.member_id ORDER BY m.member_id),'[]'::jsonb) FROM serving_lease_members m WHERE m.lease_id=s.lease_id)) ORDER BY o.received_at,s.report_id) FROM node_generation_hold_reports s JOIN node_observations o USING(node_id,incarnation_id,report_id)),'[]'::jsonb),
	'escalations',coalesce((SELECT jsonb_agg(jsonb_build_object('id',a.id,'occurred_at',a.occurred_at,'resource',a.resource,'result',a.result,'evidence',a.after_diff) ORDER BY a.occurred_at,a.id) FROM audit_events a WHERE a.action='node.generation_hold.escalated'),'[]'::jsonb),
	'revocations',coalesce((SELECT jsonb_agg(jsonb_build_object('id',a.id,'occurred_at',a.occurred_at,'action',a.action,'resource',a.resource,'operation_id',a.operation_id,'result',a.result) ORDER BY a.occurred_at,a.id) FROM audit_events a WHERE a.action IN ('device.revoke','member.grant_revoke','member.suspend','member.close','member.update','device.authorization.external_withdrawal')),'[]'::jsonb))`).Scan(&joined)
	var data struct {
		Samples []struct {
			HoldID   string                `json:"hold_id"`
			ReportID string                `json:"report_id"`
			Received time.Time             `json:"received_at"`
			Report   api.ObservationReport `json:"observation"`
		} `json:"samples"`
	}
	ok := err == nil && json.Unmarshal(joined, &data) == nil
	checks := []map[string]any{}
	for _, sample := range data.Samples {
		marker, stage, retainedSafety := false, false, false
		for _, reason := range sample.Report.ForwardingBlockReasons {
			if reason == api.ForwardingBlockReasonPolicyMissing {
				marker = true
			}
		}
		for _, e := range sample.Report.Errors {
			if e.Stage == "lease" && e.Code == api.ErrorCodeSTALEGENERATION {
				stage = true
			}
			if string(e.Code) == "TC_FOREIGN_FILTER" {
				retainedSafety = true
			}
		}
		for _, c := range sample.Report.Checks {
			code, _ := c.ErrorCode.Get()
			if c.Status == api.HealthCheckStatusFailed && (c.Name == api.NodeCheckNameQos || c.Name == api.NodeCheckNameSourceBinding) && code != api.ErrorCodeSTALEGENERATION && code != api.ErrorCodeVALIDATIONFAILED {
				retainedSafety = true
			}
		}
		cause, e := nodes.PublicationDataPathFailure(l.ctx, l.pool, sample.Report, 0, sample.Received)
		valid := marker && stage && !sample.Report.MemberForwardingEnabled && e == nil && (!retainedSafety || cause == "check_failed" || cause == "isolation_unconfirmed")
		checks = append(checks, map[string]any{"hold_id": sample.HoldID, "report_id": sample.ReportID, "policy_missing": marker, "lease_stage_stale_generation": stage, "forwarding": sample.Report.MemberForwardingEnabled, "retained_safety_failure": retainedSafety, "publication_cause_replayed": cause, "valid": valid})
		ok = ok && valid
	}
	evidence := map[string]any{"run_id": l.runID, "source_commit": l.build.Source, "case": r.ID, "joined": joined, "snapshots": l.holdSnapshots, "checks": checks,
		"interpretation":          "Node-wide hold; the observation contract has no per-member hold field. Grant or device withdrawal while members remain eligible cannot prove members_revoked. Incident closure is not service recovery.",
		"control_io_after_cancel": "not observable here", "classification_scope": "Replay original accepted reports through PublicationDataPathFailure at their receipt time; current signed-lease identity lookup. This is not a captured operation transition."}
	raw, marshalErr := json.MarshalIndent(evidence, "", "  ")
	if marshalErr != nil || writeEvidence(filepath.Join(l.root, "docs/e2e/at/members/F15-"+r.ID+"-generation-holds.json"), append(raw, '\n')) != nil {
		l.t.Fatal("hold evidence writer")
	}
	r.check("r18 M-3 accepted hold and central ID join", "retain incidents, exact reports, lease members, revocation and escalation IDs; markers and retained safety precedence when exercised", fmt.Sprintf("samples=%d; valid=%t; details=F15-%s-generation-holds.json; no matching scenario is not exercised", len(checks), ok, r.ID), "read-only own-stack durable holds/audits; replay publication classification, no new observation", ok)
}
