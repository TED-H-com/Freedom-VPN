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

func TestE2E_Followup26Native(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP26") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP26=1 on a fresh owned kernel stack")
	}
	t.Setenv("FVPN_E2E_FOLLOWUP25", "1")
	t.Setenv("FVPN_E2E_FOLLOWUP25_ARM", "kernel")
	t.Setenv("FVPN_E2E_FOLLOWUP25_FAMILY", os.Getenv("FVPN_E2E_FOLLOWUP26_FAMILY"))
	TestE2E_Followup25Native(t)
}

// All rows, not just the latest projection. SQL projects content-free fields;
// free-form check detail and error messages never enter the public artifact.
func (r *runner) exportCentral26(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	rec := Record{ID: "central-record", Title: "Accepted central serving history before teardown", Status: StatusFail}
	defer func() { r.record(rec) }()
	db, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer db.Close(ctx)
	queries := []struct{ name, sql string }{
		{"desired", `SELECT coalesce(jsonb_agg(jsonb_build_object('generation',generation::text,'signed_at',signed_at,'binding_revision',binding_revision::text) ORDER BY generation),'[]') FROM desired_states WHERE node_id=$1`},
		{"leases", `SELECT coalesce(jsonb_agg(jsonb_build_object('lease_id',s.id,'generation',s.policy_generation::text,'authorization_generation',s.authorization_generation::text,'issued_at',s.issued_at,'expires_at',s.expires_at,'signed_expires_at',s.envelope->'signed'->'expires_at','members',coalesce((SELECT jsonb_agg(jsonb_build_object('member_id',m.member_id,'entitlement_revision',m.entitlement_revision::text,'valid_from',m.valid_from,'expires_at',m.expires_at)) FROM serving_lease_members m WHERE m.lease_id=s.id),'[]')) ORDER BY s.issued_at,s.id),'[]') FROM serving_leases s WHERE s.node_id=$1`},
		{"observations", `SELECT coalesce(jsonb_agg(jsonb_build_object('report_id',o.report_id,'received_at',o.received_at,'observed_at',o.observed_at,'applied_config_generation',o.report_payload->'applied_config_generation','member_forwarding_enabled',o.member_forwarding_enabled,'forwarding_block_reasons',o.report_payload->'forwarding_block_reasons','lease',o.report_payload->'lease','provisioning_state',o.provisioning_state,'checks',coalesce((SELECT jsonb_agg(jsonb_build_object('name',c->'name','status',c->'status','error_code',c->'error_code','observed_at',c->'observed_at')) FROM jsonb_array_elements(o.report_payload->'checks') c),'[]'),'errors',coalesce((SELECT jsonb_agg(jsonb_build_object('code',e->'code','stage',e->'stage','observed_at',e->'observed_at')) FROM jsonb_array_elements(o.report_payload->'errors') e),'[]')) ORDER BY o.received_at,o.id),'[]') FROM node_observations o WHERE o.node_id=$1`},
	}
	noteJSON(&rec, "node_id", r.nodeID)
	for _, q := range queries {
		var raw []byte
		if db.QueryRow(ctx, q.sql, r.nodeID).Scan(&raw) != nil {
			recPathNote(&rec)("export_error", q.name)
			return
		}
		noteJSON(&rec, q.name, json.RawMessage(raw))
	}
	noteJSON(&rec, "exported_at", time.Now().UTC())
	rec.Status = StatusPass
}

func (r *runner) servingGate26(ctx context.Context, id string) bool {
	rec := Record{ID: id, Title: "Populated lease and serving prerequisite before positive measurement", Status: StatusFail}
	defer func() { r.record(rec) }()
	start := time.Now().UTC()
	noteJSON(&rec, "started_at", start)
	db, err := r.lab.db(ctx)
	if err != nil {
		return false
	}
	defer db.Close(ctx)
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var accepted []byte
	ready := waitUntil(wait, time.Second, func() bool {
		err = db.QueryRow(wait, `SELECT jsonb_build_object('report_id',o.report_id,'received_at',o.received_at,'observed_at',o.observed_at,'generation',o.applied_generation::text,'lease_id',s.id,'expires_at',s.expires_at,'member_count',cardinality(s.member_grant_refs))
 FROM nodes n JOIN LATERAL (SELECT * FROM node_observations WHERE node_id=n.id AND incarnation_id=n.current_incarnation_id ORDER BY received_at DESC,id DESC LIMIT 1) o ON true
 JOIN serving_leases s ON s.id::text=o.report_payload->'lease'->>'lease_id' AND s.node_id=n.id AND s.incarnation_id=n.current_incarnation_id
 WHERE n.id=$1 AND n.desired_config_generation=n.applied_config_generation AND o.applied_generation=n.applied_config_generation AND s.policy_generation=o.applied_generation
 AND o.member_forwarding_enabled AND o.report_payload->'forwarding_block_reasons'='[]'::jsonb AND o.received_at>clock_timestamp()-interval '15 seconds' AND s.expires_at>clock_timestamp()
 AND EXISTS (SELECT 1 FROM device_bindings b JOIN member_devices d ON d.id=b.device_id JOIN serving_lease_members m ON m.lease_id=s.id AND m.member_id=b.member_id WHERE b.device_id=$2 AND d.status='authorized' AND b.valid_from<=clock_timestamp() AND b.valid_until>clock_timestamp() AND m.valid_from<=clock_timestamp() AND m.expires_at>clock_timestamp())`, r.nodeID, r.usageDevice).Scan(&accepted)
		if err != nil {
			return false
		}
		var a struct {
			Generation string `json:"generation"`
		}
		if json.Unmarshal(accepted, &a) != nil {
			return false
		}
		var local struct {
			Signed struct {
				Generation string `json:"config_generation"`
			} `json:"signed"`
		}
		if json.Unmarshal(r.desiredRaw(), &local) != nil || local.Signed.Generation != a.Generation {
			return false
		}
		if _, ok := readProjectedBinding(r.desiredRaw(), r.usageDevice); !ok {
			return false
		}
		counters, e := (&Stack{lab: r.lab}).DeviceByteCounters(wait, r.usageDevice)
		noteJSON(&rec, "last_counters", counters)
		beat, e2 := r.lab.docker(r.lab.node, "cat", "/run/vpn-node/heartbeat")
		return e == nil && e2 == nil && counters.Present && counters.Generation == a.Generation && servingHeartbeatSince(beat, start)
	})
	noteJSON(&rec, "ready", ready)
	noteJSON(&rec, "completed_at", time.Now().UTC())
	if len(accepted) > 0 {
		noteJSON(&rec, "accepted", json.RawMessage(accepted))
	}
	if ready {
		rec.Status = StatusPass
	}
	return ready
}

// D-077 restore fence input, as used by members. No member validation or
// projector-owned device/binding row is modified by this TEST ONLY fixture.
func (r *runner) regrant26(ctx context.Context, admin *session) {
	rec := Record{ID: "regrant-session", Title: "F14-01 first D-077 re-grant under native session capture", Spec: []string{"AT02", "D-077", "F14-01"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	if !r.flowCounted {
		recPathNote(&rec)("prerequisite", "counted initial path unavailable; no re-grant attempted")
		return
	}
	s := &Stack{lab: r.lab}
	c, err := s.StartNativeCapture(ctx)
	if err != nil {
		return
	}
	defer r.finishNativeCapture22(ctx, &rec, c)
	if c.Sample(ctx, "before-fence") != nil {
		return
	}
	db, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer db.Close(ctx)
	var member uuid.UUID
	if db.QueryRow(ctx, `SELECT member_id FROM member_devices WHERE id=$1`, r.usageDevice).Scan(&member) != nil {
		return
	}
	noteJSON(&rec, "fence_started_at", time.Now().UTC())
	if _, err = db.Exec(ctx, `INSERT INTO device_authorization_fences(device_id,fenced_at,observed_group_ids) VALUES($1,clock_timestamp(),'{}')`, r.usageDevice); err != nil {
		return
	}
	wait, cancel := context.WithTimeout(ctx, 90*time.Second)
	withdrawn := waitUntil(wait, time.Second, func() bool {
		var yes bool
		return db.QueryRow(wait, `SELECT EXISTS(SELECT 1 FROM device_authorization_observations WHERE device_id=$1 AND NOT observed_authorized)`, r.usageDevice).Scan(&yes) == nil && yes
	})
	cancel()
	noteJSON(&rec, "withdrawn", withdrawn)
	noteJSON(&rec, "withdrawal_observed_at", time.Now().UTC())
	if !withdrawn {
		return
	}
	if c.Sample(ctx, "withdrawn-before-authorize") != nil {
		return
	}
	path := r.lab.public + "/api/v1/admin/members/" + member.String() + "/devices/" + r.usageDevice.String() + "/authorize"
	key := uuid.NewString()
	noteJSON(&rec, "authorize_started_at", time.Now().UTC())
	response, e := admin.call(ctx, "POST", path, nil, map[string]string{"Idempotency-Key": key})
	recPathNote(&rec)("authorize", httpDetail(response, e))
	if e != nil || response.Status != 202 {
		return
	}
	replay, e := admin.call(ctx, "POST", path, nil, map[string]string{"Idempotency-Key": key})
	var first, again struct {
		ID uuid.UUID `json:"operation_id"`
	}
	replayOK := e == nil && replay.Status == 202 && json.Unmarshal(response.Body, &first) == nil && json.Unmarshal(replay.Body, &again) == nil && first.ID != uuid.Nil && first.ID == again.ID
	noteJSON(&rec, "same_operation_replay", replayOK)
	if !replayOK {
		return
	}
	if c.Sample(ctx, "after-authorize") != nil {
		return
	}
	if !r.servingGate26(ctx, "regrant-serving") {
		return
	}
	r.nativeTransfer26(ctx, s, &rec, c)
}
