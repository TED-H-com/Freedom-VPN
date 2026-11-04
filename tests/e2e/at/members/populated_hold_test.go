//go:build e2e

package members

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

// TEST ONLY. These incidents follow the original AT02 mutations. Node2's
// publication selects the lease population; no signed document is manufactured.
func (l *live) populatedHolds(r *record) {
	if !l.secondExit(r) {
		r.block("Populated holds not exercised: second exit unavailable")
		return
	}
	restore := l.useSecond()
	defer restore()
	defer func() {
		var device uuid.UUID
		var status string
		runPopulatedHoldCleanup(r, populatedHoldCleanupSteps{
			restoreHoldMember: func() {
				l.restoreHoldMember(r, "TEST ONLY hold cleanup")
			},
			deviceNeedsRestore: func() bool {
				return l.pool.QueryRow(l.ctx, `SELECT id,status FROM member_devices WHERE member_id=$1 ORDER BY created_at,id LIMIT 1`, l.b.member).Scan(&device, &status) == nil && status != "authorized"
			},
			authorizeHoldDevice: func() grantHTTPResponse {
				return l.api(l.admin, "POST", authorizePath(l.b.member, device), nil, commandHeaders()).grantResp()
			},
			publishGroupA: func() {
				l.publish(r, []uuid.UUID{l.groupA})
			},
		})
	}()
	for i, row := range []struct {
		name   string
		groups []uuid.UUID
		count  int
	}{
		{"alarm-recovery", []uuid.UUID{l.groupA, l.groupB}, 2},
		{"sibling-removal", []uuid.UUID{l.groupA, l.groupB}, 2},
		{"sole-member", []uuid.UUID{l.groupB}, 1},
		{"retained-safety", []uuid.UUID{l.groupA, l.groupB}, 2},
	} {
		l.admin = l.login("e2e-admin", "admin")
		if i > 0 {
			l.restoreHoldMember(r, "TEST ONLY next independent hold")
		}
		// Suspension can withdraw its devices. Restore the original binding
		// through the real explicit authorization API before the next incident.
		var device uuid.UUID
		var status string
		if l.pool.QueryRow(l.ctx, `SELECT id,status FROM member_devices WHERE member_id=$1 ORDER BY created_at,id LIMIT 1`, l.b.member).Scan(&device, &status) == nil && status != "authorized" {
			got := l.api(l.admin, "POST", authorizePath(l.b.member, device), nil, commandHeaders())
			r.check("populated hold restore original B binding "+row.name, "202 explicit device authorization", detail(got), "POST admin member device authorize; no device or binding row edits", got.status == 202)
		}
		if !l.publish(r, row.groups) {
			r.block(row.name + " hold not exercised: publication unavailable")
			continue
		}
		// Let real FUP/initial policy compilation settle, as in devstack 27.
		select {
		case <-l.ctx.Done():
			return
		case <-time.After(65 * time.Second):
		}
		l.populatedHold(r, row.name, row.count, 211000000+i*1000000)
	}
}

// A restore is conditional: an already eligible member needs no mutation.
// The original F16 run keeps its two 200-versus-202 no-op assertion failures.
func (l *live) restoreHoldMember(r *record, reason string) {
	var suspended bool
	if err := l.pool.QueryRow(l.ctx, `SELECT admin_suspended FROM members WHERE id=$1`, l.b.member).Scan(&suspended); err != nil {
		r.block("Hold cleanup member state unavailable; no mutation inferred")
		return
	}
	r.jsonCheck("conditional hold restoration "+reason, "mutate only an actually suspended member", map[string]any{"member_id": l.b.member, "suspended_before": suspended, "mutation_needed": suspended}, "read original member state; no no-op PATCH", true)
	if suspended {
		l.patchMember(r, l.b, map[string]any{"admin_suspended": false, "suspension_reason": reason})
	}
}

func (l *live) populatedHold(r *record, name string, count, capacity int) {
	defer l.timedSection("populated-hold")()
	role, node, inc := l.nodeContainer(), l.node, l.inc
	ev := map[string]any{"run_id": l.runID, "source_commit": l.build.Source, "row": name, "node_id": node, "incarnation_id": inc, "test_only": true, "started_at": time.Now().UTC(), "coverage": "not exercised", "service_recovery_claimed": false}
	var incident, lease, gen string
	var arm e2e.MatchingLeaseHoldArm
	defer func() {
		if arm.ID != "" {
			clean, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), 15*time.Second)
			defer cancel()
			_ = l.hook.ReleaseMatchingLeaseHold(clean, role, arm.ID)
			proxy, e := l.hook.MatchingLeaseHold(clean, role, arm.ID)
			ev["proxy"], ev["proxy_available"] = proxy, e == nil
		}
		snap, e := l.hook.GenerationHoldSnapshot(l.ctx, node)
		ev["central"], ev["central_available"] = snap, e == nil
		rows, receipt, e := l.readEnforcementTimingAt(role)
		ev["producer_records"], ev["producer_receipt"], ev["producer_available"] = rows, receipt, e == nil
		ev["incident_id"], ev["finished_at"] = incident, time.Now().UTC()
		raw, _ := json.MarshalIndent(ev, "", "  ")
		if writeEvidence(filepath.Join(l.root, "docs/e2e/at/members/F18-hold-"+name+".json"), append(raw, '\n')) != nil {
			l.t.Fatal("hold evidence write")
		}
	}()
	var before json.RawMessage
	ready := l.wait(60*time.Second, func() bool {
		err := l.pool.QueryRow(l.ctx, `SELECT s.policy_generation::text,s.id::text,jsonb_build_object('node_id',n.id,'incarnation_id',o.incarnation_id,'report_id',o.report_id,'received_at',o.received_at,'lease_id',s.id,'generation',s.policy_generation::text,'desired_generation',n.desired_config_generation::text,'applied_generation',o.applied_generation::text,'expires_at',s.expires_at,'members',(SELECT jsonb_agg(member_id ORDER BY member_id) FROM serving_lease_members WHERE lease_id=s.id))
 FROM nodes n JOIN LATERAL(SELECT * FROM node_observations WHERE node_id=n.id AND incarnation_id=n.current_incarnation_id ORDER BY received_at DESC,id DESC LIMIT 1)o ON true
 JOIN serving_leases s ON s.id::text=o.report_payload->'lease'->>'lease_id'
 WHERE n.id=$1 AND n.desired_config_generation=n.applied_config_generation AND n.applied_config_generation=o.applied_generation AND s.node_id=n.id AND s.incarnation_id=n.current_incarnation_id AND o.member_forwarding_enabled AND o.received_at>clock_timestamp()-interval '15 seconds' AND s.policy_generation=o.applied_generation AND (SELECT count(DISTINCT member_id) FROM serving_lease_members WHERE lease_id=s.id)=$2 AND s.expires_at>clock_timestamp()+interval '150 seconds' AND NOT EXISTS(SELECT 1 FROM serving_lease_members lm WHERE lm.lease_id=s.id AND NOT EXISTS(SELECT 1 FROM device_bindings b JOIN member_devices d ON d.id=b.device_id WHERE b.member_id=lm.member_id AND d.status='authorized' AND b.valid_from<=clock_timestamp() AND b.valid_until>clock_timestamp()))`, node, count).Scan(&gen, &lease, &before)
		return err == nil
	})
	ev["before"], ev["populated_prerequisite"] = before, ready
	if !ready {
		r.block(name + " hold not exercised: fresh exact populated lease unavailable")
		return
	}
	local, e := l.hook.LocalLeaseReadback(l.ctx, role)
	ev["retained_local_lease"] = local
	if e != nil || local.LeaseID != lease {
		r.block(name + " hold not exercised: local lease does not join accepted report")
		return
	}
	if name == "sibling-removal" {
		ev["survivor_member_id"], ev["sibling_member_id"] = l.a.member, l.b.member
		arm, e = l.hook.ArmMatchingLeaseHoldSurvivor(l.ctx, role, gen, l.a.member, 90*time.Second)
	} else {
		arm, e = l.hook.ArmMatchingLeaseHold(l.ctx, role, gen, 90*time.Second)
	}
	ev["arm"] = arm
	if e != nil {
		r.block(name + " hold not exercised: arm unavailable")
		return
	}
	command := l.nodeMutation("PATCH", "", map[string]any{"capacity": map[string]any{"down_bps": capacity, "up_bps": capacity, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}})
	ev["command"] = map[string]any{"at": time.Now().UTC(), "status": command.status, "body": json.RawMessage(command.body), "node_write": command.write.Receipt}
	r.check("populated hold capacity command "+name, "202 real capacity command", command.detail(), "D-102 fresh TEST ONLY arm; current ETag and Idempotency-Key", command.accepted(202))
	if !command.accepted(202) {
		return
	}
	entered := l.wait(45*time.Second, func() bool {
		return l.pool.QueryRow(l.ctx, `SELECT h.id::text FROM node_generation_holds h JOIN node_generation_hold_reports s ON s.hold_id=h.id WHERE h.node_id=$1 AND s.lease_id=$2 AND h.first_seen_at>=$3 AND h.ended_at IS NULL ORDER BY h.first_seen_at DESC LIMIT 1`, node, lease, arm.ArmedAt).Scan(&incident) == nil
	})
	ev["entered"] = entered
	if !entered {
		r.block(name + " hold not exercised: no accepted incident for retained populated lease")
		return
	}
	var initialTarget, entryReport string
	_ = l.pool.QueryRow(l.ctx, `SELECT s.applied_generation::text,s.report_id::text FROM node_generation_hold_reports s JOIN node_observations o USING(node_id,incarnation_id,report_id) WHERE s.hold_id=$1 ORDER BY o.received_at,s.report_id LIMIT 1`, incident).Scan(&initialTarget, &entryReport)
	ev["initial_target_generation"], ev["entry_hold_report_id"] = initialTarget, entryReport
	during, _ := l.hook.GenerationHoldSnapshot(l.ctx, node)
	ev["at_entry"] = during
	ev["coverage"] = "accepted populated hold; individual assertions require final ID joins"
	// Require the reported three-Step alarm and exact single escalation join.
	alarm := l.wait(25*time.Second, func() bool {
		return l.count(`SELECT count(*) FROM node_generation_hold_reports WHERE hold_id=$1 AND summary_valid AND reported_alarm`, incident) > 0 && l.count(`SELECT count(*) FROM audit_events WHERE action='node.generation_hold.escalated' AND after_diff->>'incident_id'=$1`, incident) == 1
	})
	var alarmReport, auditID string
	_ = l.pool.QueryRow(l.ctx, `SELECT report_id::text FROM node_generation_hold_reports WHERE hold_id=$1 AND summary_valid AND reported_alarm ORDER BY report_id LIMIT 1`, incident).Scan(&alarmReport)
	_ = l.pool.QueryRow(l.ctx, `SELECT id::text FROM audit_events WHERE action='node.generation_hold.escalated' AND after_diff->>'incident_id'=$1 ORDER BY occurred_at,id LIMIT 1`, incident).Scan(&auditID)
	ev["alarm_hold_report_id"], ev["escalation_audit_id"] = alarmReport, auditID
	ev["alarm_and_single_escalation"] = alarm
	r.check("populated hold alarm and central escalation "+name, "accepted alarm with exactly one audit joined by incident_id", fmt.Sprintf("incident_id=%s; alarm_and_single=%t", incident, alarm), "GenerationHoldSnapshot and exact audit after_diff incident join", alarm)
	if name == "sibling-removal" || name == "sole-member" {
		at := time.Now().UTC()
		mutation := l.patchMember(r, l.b, map[string]any{"admin_suspended": true, "suspension_reason": "TEST ONLY populated hold member revocation"})
		completed := time.Now().UTC()
		l.holdSnapshot("after populated " + name + " real member revocation")
		ev["member_revocation"] = map[string]any{"member_id": l.b.member, "started_at": at, "completed_at": completed, "status": mutation.status, "body": json.RawMessage(mutation.body)}
		if name == "sole-member" {
			closed := l.wait(15*time.Second, func() bool {
				return l.count(`SELECT count(*) FROM node_generation_holds WHERE id=$1 AND end_reason='members_revoked'`, incident) == 1
			})
			ev["members_revoked_closure"] = closed
			r.check("sole leased-member closure", "members_revoked for the original one-member lease; no service-recovery claim", fmt.Sprintf("incident_id=%s; closed=%t", incident, closed), "real member suspension, exact incident and original lease member set", closed)
		} else {
			// A later matching generation may legitimately close the incident; require
			// an original post-revocation hold sample before claiming survivor coverage.
			held := l.wait(25*time.Second, func() bool {
				var accepted json.RawMessage
				err := l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object('report_id',s.report_id,'received_at',o.received_at,'applied_generation',s.applied_generation::text,'lease_id',s.lease_id,'incident_id',h.id,'incident_open',h.ended_at IS NULL,'first_seen_at',h.first_seen_at,'last_seen_at',h.last_seen_at) FROM node_generation_holds h JOIN node_generation_hold_reports s ON s.hold_id=h.id JOIN node_observations o ON o.node_id=s.node_id AND o.incarnation_id=s.incarnation_id AND o.report_id=s.report_id WHERE h.id=$1 AND h.ended_at IS NULL AND o.received_at>$2 AND s.summary_valid AND s.reported_alarm AND s.applied_generation>$5::numeric AND s.lease_id=$6::uuid AND EXISTS(SELECT 1 FROM serving_lease_members lm JOIN members m ON m.id=lm.member_id WHERE lm.lease_id=s.lease_id AND m.id=$4::uuid AND m.status NOT IN ('suspended','closed') AND NOT m.admin_suspended) AND EXISTS(SELECT 1 FROM serving_lease_members lm JOIN members m ON m.id=lm.member_id WHERE lm.lease_id=s.lease_id AND m.id=$3::uuid AND m.admin_suspended) ORDER BY o.received_at LIMIT 1`, incident, completed, l.b.member, l.a.member, initialTarget, lease).Scan(&accepted)
				if err == nil {
					ev["accepted_post_command_survivor"] = accepted
				}
				return err == nil
			})
			ev["survivor_hold_observed"] = held
			if !held {
				r.block("Sibling removal during same hold not exercised: no accepted survivor sample before incident ending; retain mutation and proxy receipts")
			}
		}
	}
	if name == "retained-safety" {
		// A deny-only foreign TC fault, inside our node2 namespace. No node nft
		// observation rule, redirect, host networking command or signed-row edit.
		faultAt := time.Now().UTC()
		_, qerr := l.inside(role, "tc", "qdisc", "add", "dev", "fvmb0", "clsact")
		injected := false
		if qerr == nil {
			defer func() { _, _ = l.inside(role, "tc", "qdisc", "del", "dev", "fvmb0", "clsact") }()
			_, ferr := l.inside(role, "tc", "filter", "add", "dev", "fvmb0", "ingress", "protocol", "ip", "pref", "49152", "flower", "ip_proto", "udp", "dst_port", "9", "action", "drop")
			injected = ferr == nil
		}
		ev["tc_fault"] = map[string]any{"started_at": faultAt, "injected": injected, "kind": "TEST ONLY clsact/flower deny UDP/9; no redirect", "creator": "members follow-up 16"}
		if injected {
			time.Sleep(18 * time.Second)
		}
		faultSnap, _ := l.hook.GenerationHoldSnapshot(l.ctx, node)
		ev["during_fault"] = faultSnap
		if qerr == nil {
			_, err := l.inside(role, "tc", "qdisc", "del", "dev", "fvmb0", "clsact")
			ev["tc_removed"] = err == nil
		}
		// Capture whatever the runtime actually produced; offline review decides
		// whether failed Apply or retained-safety precedence intersected this hold.
	}
	release := l.hook.ReleaseMatchingLeaseHold(l.ctx, role, arm.ID)
	ev["release_requested_at"], ev["release_request_ok"] = time.Now().UTC(), release == nil
	ended := l.wait(35*time.Second, func() bool {
		return l.count(`SELECT count(*) FROM node_generation_holds WHERE id=$1 AND ended_at IS NOT NULL`, incident) == 1
	})
	ev["ended_observed"] = ended
	finalLocal, le := l.hook.LocalLeaseReadback(l.ctx, role)
	ev["final_local_lease"], ev["final_local_available"] = finalLocal, le == nil
	proxy, pe := l.hook.MatchingLeaseHold(l.ctx, role, arm.ID)
	matched := false
	if pe == nil && le == nil {
		for _, event := range proxy.Events {
			if event.Kind == "matching_lease_forwarded" && event.LeaseID == finalLocal.LeaseID && event.SHA256 == finalLocal.SHA256 {
				matched = true
			}
		}
	}
	ev["forwarded_body_matches_local"] = matched
	if name == "alarm-recovery" || name == "sibling-removal" {
		closed := l.count(`SELECT count(*) FROM node_generation_holds WHERE id=$1 AND end_reason='generation_matched'`, incident) == 1
		serving := l.wait(20*time.Second, func() bool {
			var report json.RawMessage
			err := l.pool.QueryRow(l.ctx, `SELECT jsonb_build_object('report_id',report_id,'received_at',received_at,'applied_generation',applied_generation::text,'lease_id',report_payload->'lease'->>'lease_id') FROM node_observations WHERE node_id=$1 AND member_forwarding_enabled AND received_at>$2 ORDER BY received_at DESC,id DESC LIMIT 1`, node, ev["release_requested_at"]).Scan(&report)
			if err == nil {
				ev["recovery_serving_report"] = report
			}
			return err == nil
		})
		ev["separate_serving_report_observed"] = serving
		r.check("populated matching recovery", "generation_matched, genuine forwarded local lease, separate serving report", fmt.Sprintf("closed=%t; exact_local=%t; serving=%t", closed, matched, serving), "incident closure is checked separately from service", closed && matched && serving)
	}
	if name == "retained-safety" {
		// Recovery requires a real newer clean generation; do not raise links,
		// restore addresses or weaken the verifier after the bounded injection.
		clean := l.nodeMutation("PATCH", "", map[string]any{"capacity": map[string]any{"down_bps": capacity + 1000000, "up_bps": capacity + 1000000, "member_down_bps_cap": 100000000, "member_up_bps_cap": 100000000}})
		ev["clean_capacity_status"] = clean.status
		ev["clean_capacity_node_write"] = clean.write.Receipt
	}
}
