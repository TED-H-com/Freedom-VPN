//go:build e2e

package e2e

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	tx, err := db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)
	queries := []struct{ name, sql string }{
		{"desired", `SELECT coalesce(jsonb_agg(jsonb_build_object('generation',generation::text,'signed_at',signed_at,'binding_revision',binding_revision::text) ORDER BY generation),'[]') FROM desired_states WHERE node_id=$1`},
		{"leases", `SELECT coalesce(jsonb_agg(jsonb_build_object('lease_id',s.id,'generation',s.policy_generation::text,'authorization_generation',s.authorization_generation::text,'issued_at',s.issued_at,'expires_at',s.expires_at,'signed_expires_at',s.envelope->'signed'->'expires_at','members',coalesce((SELECT jsonb_agg(jsonb_build_object('member_id',m.member_id,'entitlement_revision',m.entitlement_revision::text,'valid_from',m.valid_from,'expires_at',m.expires_at)) FROM serving_lease_members m WHERE m.lease_id=s.id),'[]')) ORDER BY s.issued_at,s.id),'[]') FROM serving_leases s WHERE s.node_id=$1`},
		{"observations", `SELECT coalesce(jsonb_agg(jsonb_build_object('report_id',o.report_id,'received_at',o.received_at,'observed_at',o.observed_at,'applied_config_generation',o.report_payload->'applied_config_generation','member_forwarding_enabled',o.member_forwarding_enabled,'forwarding_block_reasons',o.report_payload->'forwarding_block_reasons','lease',o.report_payload->'lease','provisioning_state',o.provisioning_state,'checks',coalesce((SELECT jsonb_agg(jsonb_build_object('name',c->'name','status',c->'status','error_code',c->'error_code','observed_at',c->'observed_at')) FROM jsonb_array_elements(o.report_payload->'checks') c),'[]'),'errors',coalesce((SELECT jsonb_agg(jsonb_build_object('code',e->'code','stage',e->'stage','observed_at',e->'observed_at')) FROM jsonb_array_elements(o.report_payload->'errors') e),'[]')) ORDER BY o.received_at,o.id),'[]') FROM node_observations o WHERE o.node_id=$1`},
	}
	noteJSON(&rec, "node_id", r.nodeID)
	for _, q := range queries {
		var raw []byte
		if tx.QueryRow(ctx, q.sql, r.nodeID).Scan(&raw) != nil {
			recPathNote(&rec)("export_error", q.name)
			return
		}
		noteJSON(&rec, q.name, json.RawMessage(raw))
	}
	if tx.Commit(ctx) != nil {
		return
	}
	private := filepath.Join(r.lab.root, ".agent/tmp/followup26", r.runID, "producer.jsonl")
	receipt, e := (&Stack{lab: r.lab}).ExportContainerFile(ctx, "node", "/var/lib/vpn-node/e2e/enforcement-timing.jsonl", private)
	noteJSON(&rec, "producer_receipt", receipt)
	if e == nil {
		f, e := os.Open(private)
		if e != nil {
			return
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 4096), 4<<20)
		rows := []map[string]json.RawMessage{}
		for scanner.Scan() {
			var original map[string]json.RawMessage
			if json.Unmarshal(scanner.Bytes(), &original) != nil {
				return
			}
			row := map[string]json.RawMessage{}
			for _, key := range []string{"run_id", "boot_id", "sequence", "source", "anchor", "lease", "renewal", "baseline", "heartbeat"} {
				if v, ok := original[key]; ok {
					row[key] = v
				}
			}
			if v, ok := original["timing"]; ok {
				var timing map[string]json.RawMessage
				if json.Unmarshal(v, &timing) != nil {
					return
				}
				delete(timing, "commands")
				raw, _ := json.Marshal(timing)
				row["timing"] = raw
			}
			rows = append(rows, row)
		}
		if scanner.Err() != nil {
			return
		}
		noteJSON(&rec, "producer_timing", rows)
	} else {
		recPathNote(&rec)("producer_export_error", "unavailable")
	}
	noteJSON(&rec, "exported_at", time.Now().UTC())
	rec.Status = StatusPass
}

func (r *runner) servingGate26(ctx context.Context, id string, captures ...*NativeCapture) bool {
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
	counterReadFailed := false
	sample := 0
	ready := waitUntil(wait, time.Second, func() bool {
		if len(captures) > 0 {
			sample++
			if captures[0].Sample(wait, fmt.Sprintf("%s-%d", id, sample)) != nil {
				return false
			}
		}
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
		if e != nil {
			counterReadFailed = true
			return true
		} // End this gate; never replace a failed counter read.
		beat, e2 := r.lab.docker(r.lab.node, "cat", "/run/vpn-node/heartbeat")
		return e == nil && e2 == nil && counters.Present && counters.Generation == a.Generation && servingHeartbeatSince(beat, start)
	})
	ready = ready && !counterReadFailed
	noteJSON(&rec, "counter_read_failed", counterReadFailed)
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
	sample := 0
	withdrawn := waitUntil(wait, time.Second, func() bool {
		sample++
		if c.Sample(wait, fmt.Sprintf("withdrawal-%d", sample)) != nil {
			return false
		}
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
	if !r.servingGate26(ctx, "regrant-serving", c) {
		return
	}
	r.nativeTransfer26(ctx, s, &rec, c)
}

// Replay the exact live NFT listing through the unchanged verifier, with empty
// TC fixture reads to isolate the NFT decision. This is not a runtime Apply or
// a forged agent observation. The actual report and writer timing remain separate.
func (r *runner) traceRejection26(ctx context.Context, k *kernelTrace, rec *Record) {
	raw, err := r.lab.docker(r.lab.node, "nft", "-j", "list", "ruleset")
	if err != nil {
		return
	}
	path := filepath.Join(r.lab.root, ".agent/tmp/followup26", r.runID, "traced-node-ruleset.raw")
	if os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	_, we := f.Write(raw)
	ce := f.Close()
	if we != nil || ce != nil {
		return
	}
	_, err = enforce.VerifyServing(ctx, traceListing26{string(raw)}, enforce.Model{Datapath: enforce.Datapath{MemberIngress: "fvmb0", Uplink: "fvup0"}}, nil)
	var check *enforce.CheckError
	got := map[string]any{"read_at": time.Now().UTC(), "bytes": fmt.Sprint(len(raw)), "sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "private_retained": true, "scope": "offline exact NFT listing replay; empty TC fixture isolates NFT rejection; runtime observation is separate"}
	if errors.As(err, &check) {
		got["error_code"] = string(check.Reason)
		got["deciding_table_is_tracer"] = strings.Contains(check.Detail, "table "+k.table+" ")
		got["deciding_hook_is_prerouting"] = strings.Contains(check.Detail, "hook prerouting ")
		got["deciding_hook_is_ingress"] = strings.Contains(check.Detail, "hook ingress ")
		got["deciding_family_is_netdev"] = strings.Contains(check.Detail, "family netdev ")
		got["deciding_statement_is_mangle"] = strings.Contains(check.Detail, "statement mangle")
	}
	noteJSON(rec, "tracer_verifier_replay", got)
}
