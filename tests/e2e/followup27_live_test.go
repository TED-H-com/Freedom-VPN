//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestE2E_Followup27(t *testing.T) {
	mode := os.Getenv("FVPN_E2E_FOLLOWUP27")
	if mode == "" {
		t.Skip("set FVPN_E2E_FOLLOWUP27=hold|native on a fresh owned stack")
	}
	if mode != "hold" && mode != "native" {
		t.Fatal("invalid follow-up 27 mode")
	}
	if os.Getenv("FVPN_E2E_REPRODUCE_TRACE26") != "" {
		t.Fatal("node tracer forbidden in follow-up 27 measured rows")
	}
	t.Setenv("FVPN_E2E_FOLLOWUP26", "1")
	TestE2E_Followup26Native(t)
}

func (r *runner) populatedHold27(ctx context.Context, s *Stack, admin *session) {
	rec := Record{ID: "populated-hold", Title: "TEST ONLY two-member stale-generation hold, alarm, single escalation and recovery", Spec: []string{"AT13", "AT02", "D-126", "§16.3"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	defer func() {
		out, e := s.GenerationHoldSnapshot(context.WithoutCancel(ctx), r.nodeID)
		if e == nil {
			noteJSON(&rec, "central", out)
		} else {
			recPathNote(&rec)("snapshot_error", "unavailable")
		}
	}()
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	scale, err := s.ScaleNativeMembers(ctx, b, r.planRevID, "hold27", 1, 1)
	noteJSON(&rec, "second_member", scale)
	if err != nil {
		recPathNote(&rec)("second_member_error", err.Error())
		return
	}
	// Initial FUP projection is allowed to settle; never force a generation.
	select {
	case <-ctx.Done():
		return
	case <-time.After(65 * time.Second):
	}
	if !r.servingGate26(ctx, "two-member-serving") {
		return
	}
	db, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer db.Close(ctx)
	var base, oldLease string
	var before json.RawMessage
	err = db.QueryRow(ctx, `SELECT s.policy_generation::text,s.id::text,jsonb_build_object('report_id',o.report_id,'lease_id',s.id,'generation',s.policy_generation::text,'expires_at',s.expires_at,'member_count',(SELECT count(DISTINCT member_id) FROM serving_lease_members WHERE lease_id=s.id))
 FROM nodes n JOIN LATERAL(SELECT * FROM node_observations WHERE node_id=n.id AND incarnation_id=n.current_incarnation_id ORDER BY received_at DESC,id DESC LIMIT 1)o ON true
 JOIN serving_leases s ON s.id::text=o.report_payload->'lease'->>'lease_id'
 WHERE n.id=$1 AND o.member_forwarding_enabled AND s.policy_generation=o.applied_generation AND (SELECT count(DISTINCT member_id) FROM serving_lease_members WHERE lease_id=s.id)>=2 AND s.expires_at>clock_timestamp()+interval '90 seconds'`, r.nodeID).Scan(&base, &oldLease, &before)
	if err != nil {
		recPathNote(&rec)("populated_prerequisite", "unavailable")
		return
	}
	noteJSON(&rec, "before", before)
	arm, err := s.ArmMatchingLeaseHold(ctx, "node", base, 45*time.Second)
	noteJSON(&rec, "arm", arm)
	if err != nil {
		return
	}
	defer func() {
		clean, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		ev, e := s.MatchingLeaseHold(clean, "node", arm.ID)
		if e == nil {
			noteJSON(&rec, "proxy", ev)
		}
		if e == nil && ev.ReleasedAt == nil {
			_ = s.ReleaseMatchingLeaseHold(clean, "node", arm.ID)
		}
	}()
	path := r.lab.public + "/api/v1/admin/nodes/" + r.nodeID.String()
	detail, err := admin.call(ctx, "GET", path, nil, nil)
	if err != nil || detail.Status != 200 {
		return
	}
	patch, err := admin.call(ctx, "PATCH", path, map[string]any{"capacity": map[string]any{"down_bps": 210000000, "up_bps": 210000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}}, map[string]string{"If-Match": detail.Header.Get("ETag"), "Idempotency-Key": uuid.NewString()})
	noteJSON(&rec, "command", map[string]any{"at": time.Now().UTC(), "status": patch.Status, "accepted": err == nil && patch.Status == 202})
	if err != nil || patch.Status != 202 {
		return
	}
	var incident string
	var samples []json.RawMessage
	wait, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	matched := waitUntil(wait, time.Second, func() bool {
		raw, e := s.GenerationHoldSnapshot(wait, r.nodeID)
		if e != nil {
			return false
		}
		var snap struct {
			Incidents []struct {
				ID     string     `json:"id"`
				First  time.Time  `json:"first_seen_at"`
				End    *time.Time `json:"ended_at"`
				Reason *string    `json:"end_reason"`
			} `json:"incidents"`
			Samples []struct {
				Hold  string `json:"hold_id"`
				Lease string `json:"lease_id"`
			} `json:"samples"`
		}
		if json.Unmarshal(raw, &snap) != nil {
			return false
		}
		for _, v := range snap.Samples {
			if v.Lease == oldLease {
				incident = v.Hold
			}
		}
		if incident != "" && len(samples) < 120 {
			samples = append(samples, raw)
		}
		for _, h := range snap.Incidents {
			if h.ID == incident && h.End != nil && h.Reason != nil && *h.Reason == "generation_matched" {
				return true
			}
		}
		return false
	})
	noteJSON(&rec, "incident_id", incident)
	// Keep bounded polling facts separately; the final consistent snapshot
	// contains every original accepted report and incident sample.
	states := []map[string]any{}
	for _, raw := range samples {
		var snap map[string]json.RawMessage
		_ = json.Unmarshal(raw, &snap)
		states = append(states, map[string]any{"sampled_at": snap["sampled_at"], "incidents": snap["incidents"]})
	}
	noteJSON(&rec, "polls", states)
	if !matched {
		recPathNote(&rec)("product_owner", "agent-core / wire-nodes: no matching populated hold recovery; inspect exact receipts")
		return
	}
	var proof json.RawMessage
	err = db.QueryRow(ctx, `SELECT jsonb_build_object('incident_id',h.id,'first_seen_at',h.first_seen_at,'last_seen_at',h.last_seen_at,'ended_at',h.ended_at,'end_reason',h.end_reason,'duration_ms',(extract(epoch FROM h.ended_at-h.first_seen_at)*1000)::bigint::text,'audit_count',(SELECT count(*) FROM audit_events a WHERE a.action='node.generation_hold.escalated' AND a.after_diff->>'incident_id'=h.id::text),'alarm_reports',(SELECT count(*) FROM node_generation_hold_reports r WHERE r.hold_id=h.id AND r.summary_valid AND r.reported_alarm)) FROM node_generation_holds h WHERE h.id=$1 AND h.ended_at>h.first_seen_at+interval '30 seconds' AND h.end_reason='generation_matched'`, incident).Scan(&proof)
	if err != nil {
		return
	}
	noteJSON(&rec, "proof", proof)
	var p struct {
		Audits int `json:"audit_count"`
		Alarms int `json:"alarm_reports"`
	}
	if json.Unmarshal(proof, &p) != nil || p.Audits != 1 || p.Alarms < 1 {
		recPathNote(&rec)("product_owner", "agent-core / wire-nodes: alarm or single escalation missing")
		return
	}
	// The durable local lease must equal a genuine forwarded response bytewise.
	ev, err := s.MatchingLeaseHold(ctx, "node", arm.ID)
	if err != nil {
		return
	}
	local, err := s.LocalLeaseReadback(ctx, "node")
	if err != nil {
		recPathNote(&rec)("local_read_error", "unavailable")
		return
	}
	exact := false
	for _, e := range ev.Events {
		if e.Kind == "matching_lease_forwarded" && e.LeaseID == local.LeaseID && e.SHA256 == local.SHA256 {
			exact = true
		}
	}
	noteJSON(&rec, "released_local_lease", local)
	noteJSON(&rec, "exact_forwarded_body", exact)
	if exact && r.servingGate26(ctx, "matching-lease-serving") {
		rec.Status = StatusPass
	}
}
