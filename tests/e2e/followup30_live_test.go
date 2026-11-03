//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestE2E_Followup30SurvivorHold(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP30") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP30=1 on a fresh owned stack")
	}
	if os.Getenv("FVPN_E2E_REPRODUCE_TRACE26") != "" {
		t.Fatal("node tracer forbidden in follow-up 30 measured rows")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Minute)
	defer cancel()
	head, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	runDir := filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup30", id)
	r := &runner{
		t:            t,
		lab:          l,
		runID:        id,
		sourceCommit: strings.TrimSpace(string(head)),
		dir:          runDir,
	}
	defer r.finish()
	defer r.exportCentral26(context.WithoutCancel(ctx))

	s := &Stack{lab: l}
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	must(s.EnableEnforcementTiming(ctx))
	must(s.PrepareNativeUnderlay(ctx))
	must(s.InstallFixtureProbe(ctx))
	must(s.StartFixtureTargets(ctx))
	rev, _, err := s.ActivatePolicy(ctx, b, "domain_rule_set", "followup30-rules", map[string]any{
		"rules": []any{
			map[string]any{
				"rule_id":            uuid.New(),
				"domain":             "feed-01.example.invalid",
				"include_subdomains": true,
				"action":             "block",
				"profile_id":         nil,
				"reason":             "admin_rule",
				"revision":           "1",
			},
		},
	})
	must(err)
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "followup30-profile", map[string]any{
		"profile_code":                 "followup30",
		"display_name":                 map[string]string{"zh-TW": "TEST ONLY followup30"},
		"block_response":               "nxdomain",
		"blocked_ttl_seconds":          5,
		"domain_rule_set_revision_ids": []uuid.UUID{rev},
		"threat_feed_revision_ids":     []string{},
	})
	must(err)
	r.catalogue(ctx, admin)
	_, err = PrepareServingBaseline(ctx, admin.http, admin.csrf, false)
	must(err)
	if r.enroll(ctx, admin) == uuid.Nil {
		return
	}
	r.desired(ctx)
	r.publish(ctx, admin, r.nodeID)
	r.fixtureServingMember(ctx, admin)
	r.survivorHold30(ctx, s, admin)
}

func (r *runner) survivorHold30(ctx context.Context, s *Stack, admin *session) {
	rec := Record{
		ID:     "survivor-hold-continuation",
		Title:  "TEST ONLY two-member hold, real sibling suspension advancing generation, accepted post-command survivor hold report, and matching recovery",
		Spec:   []string{"AT02", "AT13", "D-102", "D-126", "§16.3"},
		Status: StatusFail,
	}
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
	scale, err := s.ScaleNativeMembers(ctx, b, r.planRevID, "hold30", 1, 1)
	noteJSON(&rec, "second_member", scale)
	if err != nil || scale.Users != 1 || scale.SimultaneousAuthorized < 1 {
		recPathNote(&rec)("second_member_error", "unavailable")
		return
	}
	survivorSess, err := s.Login(ctx, "e2e-member", "member")
	if err != nil || survivorSess.MemberID == "" {
		recPathNote(&rec)("survivor_login_error", "unavailable")
		return
	}
	siblingSess, err := s.Login(ctx, "hold30-u01", "member")
	if err != nil || siblingSess.MemberID == "" || siblingSess.MemberID == survivorSess.MemberID {
		recPathNote(&rec)("sibling_login_error", "unavailable")
		return
	}
	survivorID := survivorSess.MemberID
	siblingID := siblingSess.MemberID
	noteJSON(&rec, "members", map[string]any{
		"survivor_member_id": survivorID,
		"sibling_member_id":  siblingID,
	})

	// Let initial FUP and grant projection settle before arming.
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
	waitBase, cancelBase := context.WithTimeout(ctx, 60*time.Second)
	defer cancelBase()
	ready := waitUntil(waitBase, time.Second, func() bool {
		e := db.QueryRow(waitBase, `SELECT s.policy_generation::text,s.id::text,jsonb_build_object(
			'node_id',n.id,
			'incarnation_id',o.incarnation_id,
			'report_id',o.report_id,
			'received_at',o.received_at,
			'lease_id',s.id,
			'generation',s.policy_generation::text,
			'desired_generation',n.desired_config_generation::text,
			'applied_generation',o.applied_generation::text,
			'expires_at',s.expires_at,
			'members',(SELECT jsonb_agg(member_id ORDER BY member_id) FROM serving_lease_members WHERE lease_id=s.id)
		)
		FROM nodes n
		JOIN LATERAL(SELECT * FROM node_observations WHERE node_id=n.id AND incarnation_id=n.current_incarnation_id ORDER BY received_at DESC,id DESC LIMIT 1) o ON true
		JOIN serving_leases s ON s.id::text=o.report_payload->'lease'->>'lease_id'
		WHERE n.id=$1
		  AND n.desired_config_generation=n.applied_config_generation
		  AND n.applied_config_generation=o.applied_generation
		  AND o.member_forwarding_enabled
		  AND o.received_at>clock_timestamp()-interval '15 seconds'
		  AND s.policy_generation=o.applied_generation
		  AND (SELECT count(DISTINCT member_id) FROM serving_lease_members WHERE lease_id=s.id)=2
		  AND EXISTS(SELECT 1 FROM serving_lease_members WHERE lease_id=s.id AND member_id=$2)
		  AND EXISTS(SELECT 1 FROM serving_lease_members WHERE lease_id=s.id AND member_id=$3)
		  AND s.expires_at>clock_timestamp()+interval '150 seconds'`,
			r.nodeID, survivorID, siblingID).Scan(&base, &oldLease, &before)
		return e == nil
	})
	noteJSON(&rec, "before", before)
	if !ready {
		recPathNote(&rec)("populated_prerequisite", "unavailable")
		return
	}

	localBefore, err := s.LocalLeaseReadback(ctx, "node")
	noteJSON(&rec, "retained_local_lease", localBefore)
	if err != nil || localBefore.LeaseID != oldLease {
		recPathNote(&rec)("local_prerequisite", "mismatch")
		return
	}

	arm, err := s.ArmMatchingLeaseHoldSurvivor(ctx, "node", base, survivorID, 90*time.Second)
	noteJSON(&rec, "arm", arm)
	if err != nil {
		return
	}
	defer func() {
		clean, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		ev, e := s.MatchingLeaseHold(clean, "node", arm.ID)
		if e == nil && ev.ReleasedAt == nil {
			_ = s.ReleaseMatchingLeaseHold(clean, "node", arm.ID)
			ev, e = s.MatchingLeaseHold(clean, "node", arm.ID)
		}
		if e == nil {
			noteJSON(&rec, "proxy", ev)
		}
	}()

	nodePath := r.lab.public + "/api/v1/admin/nodes/" + r.nodeID.String()
	detail, err := admin.call(ctx, "GET", nodePath, nil, nil)
	if err != nil || detail.Status != 200 {
		return
	}
	patch, err := admin.call(ctx, "PATCH", nodePath, map[string]any{
		"capacity": map[string]any{
			"down_bps":            210000000,
			"up_bps":              210000000,
			"member_down_bps_cap": 100000000,
			"member_up_bps_cap":   100000000,
		},
	}, map[string]string{
		"If-Match":        detail.Header.Get("ETag"),
		"Idempotency-Key": uuid.NewString(),
	})
	noteJSON(&rec, "command", map[string]any{
		"at":       time.Now().UTC(),
		"status":   patch.Status,
		"accepted": err == nil && patch.Status == 202,
		"body":     json.RawMessage(patch.Body),
	})
	if err != nil || patch.Status != 202 {
		return
	}

	var incident, entryReportID, alarmReportID, escalationAuditID, initialAppliedGen string
	waitEntry, cancelEntry := context.WithTimeout(ctx, 45*time.Second)
	defer cancelEntry()
	entered := waitUntil(waitEntry, 500*time.Millisecond, func() bool {
		return db.QueryRow(waitEntry, `SELECT h.id::text, s.report_id::text, s.applied_generation::text
			FROM node_generation_holds h
			JOIN node_generation_hold_reports s ON s.hold_id=h.id
			WHERE h.node_id=$1 AND s.lease_id=$2 AND h.first_seen_at>=$3 AND h.ended_at IS NULL
			ORDER BY s.reported_steps ASC, s.report_id ASC LIMIT 1`,
			r.nodeID, oldLease, arm.ArmedAt).Scan(&incident, &entryReportID, &initialAppliedGen) == nil
	})
	if !entered {
		recPathNote(&rec)("entry_error", "no accepted incident for retained populated lease")
		return
	}

	waitAlarm, cancelAlarm := context.WithTimeout(ctx, 30*time.Second)
	defer cancelAlarm()
	alarmed := waitUntil(waitAlarm, 500*time.Millisecond, func() bool {
		e1 := db.QueryRow(waitAlarm, `SELECT report_id::text FROM node_generation_hold_reports
			WHERE hold_id=$1 AND summary_valid AND reported_alarm
			ORDER BY report_id ASC LIMIT 1`, incident).Scan(&alarmReportID)
		e2 := db.QueryRow(waitAlarm, `SELECT id::text FROM audit_events
			WHERE action='node.generation_hold.escalated' AND after_diff->>'incident_id'=$1
			ORDER BY occurred_at ASC, id ASC LIMIT 1`, incident).Scan(&escalationAuditID)
		return e1 == nil && e2 == nil
	})
	if !alarmed {
		recPathNote(&rec)("alarm_error", "alarm report or single escalation audit missing before sibling suspension")
		return
	}

	// Real sibling suspension while the hold is open.
	memberPath := r.lab.public + "/api/v1/admin/members/" + siblingID
	memberDetail, err := admin.call(ctx, "GET", memberPath, nil, nil)
	if err != nil || memberDetail.Status != 200 {
		recPathNote(&rec)("sibling_get_error", "unavailable")
		return
	}
	revStartedAt := time.Now().UTC()
	revPatch, err := admin.call(ctx, "PATCH", memberPath, map[string]any{
		"admin_suspended":   true,
		"suspension_reason": "TEST ONLY D-102 survivor hold continuation proof",
	}, map[string]string{
		"If-Match":        memberDetail.Header.Get("ETag"),
		"Idempotency-Key": uuid.NewString(),
	})
	revCompletedAt := time.Now().UTC()
	noteJSON(&rec, "member_revocation", map[string]any{
		"member_id":    siblingID,
		"started_at":   revStartedAt,
		"completed_at": revCompletedAt,
		"status":       revPatch.Status,
		"body":         json.RawMessage(revPatch.Body),
	})
	if err != nil || revPatch.Status != 202 {
		return
	}

	// Wait for an accepted post-command survivor hold report at the advanced generation while the incident stays open.
	var postReportID, postAppliedGen, postSteps string
	var postReceivedAt time.Time
	waitPost, cancelPost := context.WithTimeout(ctx, 25*time.Second)
	defer cancelPost()
	survivorHeld := waitUntil(waitPost, 500*time.Millisecond, func() bool {
		return db.QueryRow(waitPost, `SELECT s.report_id::text, s.applied_generation::text, s.reported_steps::text, o.received_at
			FROM node_generation_holds h
			JOIN node_generation_hold_reports s ON s.hold_id=h.id
			JOIN node_observations o ON o.node_id=s.node_id AND o.incarnation_id=s.incarnation_id AND o.report_id=s.report_id
			WHERE h.id=$1
			  AND h.ended_at IS NULL
			  AND o.received_at>$2
			  AND s.applied_generation>$3::numeric
			  AND s.summary_valid
			  AND s.reported_alarm
			  AND EXISTS(
			    SELECT 1 FROM serving_lease_members lm
			    JOIN members m ON m.id=lm.member_id
			    WHERE lm.lease_id=s.lease_id
			      AND m.id=$4
			      AND m.status NOT IN ('suspended','closed')
			      AND NOT m.admin_suspended
			  )
			  AND EXISTS(
			    SELECT 1 FROM members m
			    WHERE m.id=$5 AND m.admin_suspended
			  )
			ORDER BY o.received_at ASC, s.report_id ASC LIMIT 1`,
			incident, revCompletedAt, initialAppliedGen, survivorID, siblingID).Scan(&postReportID, &postAppliedGen, &postSteps, &postReceivedAt) == nil
	})
	noteJSON(&rec, "survivor_hold_observed", survivorHeld)
	if !survivorHeld {
		recPathNote(&rec)("survivor_hold_error", "no accepted post-command survivor hold report at advanced generation before incident closure")
		return
	}

	// Release the hold explicitly and verify clean generation_matched recovery at the continued generation.
	releaseRequestedAt := time.Now().UTC()
	if err := s.ReleaseMatchingLeaseHold(ctx, "node", arm.ID); err != nil {
		recPathNote(&rec)("release_error", err.Error())
		return
	}
	noteJSON(&rec, "release_requested_at", releaseRequestedAt)

	var recoveryReportID, recoveryLeaseID string
	waitEnd, cancelEnd := context.WithTimeout(ctx, 35*time.Second)
	defer cancelEnd()
	recovered := waitUntil(waitEnd, 500*time.Millisecond, func() bool {
		var endReason string
		if db.QueryRow(waitEnd, `SELECT end_reason FROM node_generation_holds WHERE id=$1 AND ended_at IS NOT NULL`, incident).Scan(&endReason) != nil || endReason != "generation_matched" {
			return false
		}
		return db.QueryRow(waitEnd, `SELECT o.report_id::text, o.report_payload->'lease'->>'lease_id'
			FROM node_observations o
			WHERE o.node_id=$1
			  AND o.incarnation_id=$2
			  AND o.member_forwarding_enabled
			  AND o.applied_generation=$3::numeric
			  AND o.received_at>$4
			ORDER BY o.received_at ASC, o.id ASC LIMIT 1`,
			r.nodeID, r.incID, postAppliedGen, postReceivedAt).Scan(&recoveryReportID, &recoveryLeaseID) == nil
	})
	if !recovered {
		recPathNote(&rec)("recovery_error", "incident did not close generation_matched with serving report at continued generation")
		return
	}

	var auditCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE action='node.generation_hold.escalated' AND after_diff->>'incident_id'=$1`, incident).Scan(&auditCount); err != nil || auditCount != 1 {
		recPathNote(&rec)("audit_count_error", "expected exactly 1 escalation audit for incident")
		return
	}

	ev, err := s.MatchingLeaseHold(ctx, "node", arm.ID)
	if err != nil {
		return
	}
	localAfter, err := s.LocalLeaseReadback(ctx, "node")
	if err != nil {
		recPathNote(&rec)("local_read_error", "unavailable")
		return
	}
	exact := false
	for _, e := range ev.Events {
		if e.Kind == "matching_lease_forwarded" && e.Generation == postAppliedGen && e.LeaseID == localAfter.LeaseID && e.SHA256 == localAfter.SHA256 {
			exact = true
		}
	}
	noteJSON(&rec, "released_local_lease", localAfter)
	noteJSON(&rec, "exact_forwarded_body", exact)
	noteJSON(&rec, "proof_ids", map[string]any{
		"node_id":                       r.nodeID,
		"incarnation_id":                r.incID,
		"arm_id":                        arm.ID,
		"survivor_member_id":            survivorID,
		"sibling_member_id":             siblingID,
		"retained_lease_id":             oldLease,
		"base_generation":               base,
		"initial_target_generation":     initialAppliedGen,
		"continued_generation":          postAppliedGen,
		"incident_id":                   incident,
		"entry_hold_report_id":          entryReportID,
		"alarm_hold_report_id":          alarmReportID,
		"escalation_audit_id":           escalationAuditID,
		"escalation_audit_count":        auditCount,
		"post_command_hold_report_id":   postReportID,
		"post_command_reported_steps":   postSteps,
		"post_command_report_received":  postReceivedAt,
		"recovery_serving_report_id":    recoveryReportID,
		"recovery_survivor_lease_id":    recoveryLeaseID,
		"local_lease_matches_forwarded": exact,
	})
	if exact && localAfter.LeaseID == recoveryLeaseID && r.servingGate26(ctx, "post-continuation-serving") {
		rec.Status = StatusPass
	}
}
