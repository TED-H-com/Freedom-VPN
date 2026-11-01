//go:build e2e

package members

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Read immutable evidence from the complete run; never provoke a generation
// change to repair the restart lease.
func (l *live) nodeLeaseTimeline(nodeID uuid.UUID, since string) ([]byte, []byte, bool) {
	var leases, observations []byte
	e1 := l.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(v ORDER BY at),'[]'::jsonb) FROM (SELECT issued_at at,jsonb_build_object('issued_at',issued_at,'expires_at',expires_at,'generation',policy_generation::text,'member_grants',jsonb_array_length(envelope->'signed'->'members')) v FROM serving_leases WHERE node_id=$1 AND issued_at>=$2::timestamptz-interval '2 minutes') t`, nodeID, since).Scan(&leases)
	e2 := l.pool.QueryRow(l.ctx, `SELECT coalesce(jsonb_agg(v ORDER BY first_at),'[]'::jsonb) FROM (SELECT min(received_at) first_at,jsonb_build_object('first_at',min(received_at),'last_at',max(received_at),'observations',count(*),'generation',generation,'forwarding',forwarding,'reasons',reasons,'safety_status',safety_status,'lease_check',lease_check,'qos_check',qos_check,'source_binding_check',source_binding_check,'errors',errors) v FROM (SELECT received_at,generation,forwarding,reasons,safety_status,lease_check,qos_check,source_binding_check,errors,sum(step) OVER (ORDER BY received_at) AS grp FROM (SELECT received_at,generation,forwarding,reasons,safety_status,lease_check,qos_check,source_binding_check,errors,CASE WHEN lag(generation) OVER (ORDER BY received_at) IS DISTINCT FROM generation OR lag(forwarding) OVER (ORDER BY received_at) IS DISTINCT FROM forwarding OR lag(reasons) OVER (ORDER BY received_at) IS DISTINCT FROM reasons OR lag(safety_status) OVER (ORDER BY received_at) IS DISTINCT FROM safety_status OR lag(lease_check) OVER (ORDER BY received_at) IS DISTINCT FROM lease_check OR lag(qos_check) OVER (ORDER BY received_at) IS DISTINCT FROM qos_check OR lag(source_binding_check) OVER (ORDER BY received_at) IS DISTINCT FROM source_binding_check OR lag(errors) OVER (ORDER BY received_at) IS DISTINCT FROM errors THEN 1 ELSE 0 END AS step FROM (SELECT received_at,report_payload->'applied_config_generation' AS generation,report_payload->'member_forwarding_enabled' AS forwarding,report_payload->'forwarding_block_reasons' AS reasons,report_payload->'safety_status' AS safety_status,(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail') FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='lease' LIMIT 1) AS lease_check,(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail') FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='qos' LIMIT 1) AS qos_check,(SELECT jsonb_build_object('status',c->'status','error_code',c->'error_code','detail',c->'detail') FROM jsonb_array_elements(coalesce(report_payload->'checks','[]'::jsonb)) c WHERE c->>'name'='source_binding' LIMIT 1) AS source_binding_check,(SELECT coalesce(jsonb_agg(jsonb_build_object('stage',e->'stage','code',e->'code','message',e->'message')),'[]'::jsonb) FROM jsonb_array_elements(coalesce(report_payload->'errors','[]'::jsonb)) e) AS errors FROM node_observations WHERE node_id=$1 AND received_at>=$2::timestamptz) raw) stepped) grouped GROUP BY grp,generation,forwarding,reasons,safety_status,lease_check,qos_check,source_binding_check,errors) t`, nodeID, since).Scan(&observations)
	return leases, observations, e1 == nil && e2 == nil
}

func (l *live) restartLeaseReadback(r *record) {
	inc, err := uuid.Parse(r.Nodes["node2"])
	if err != nil {
		r.block("Second-node incarnation absent from prior record")
		return
	}
	err = l.pool.QueryRow(l.ctx, `SELECT id,current_incarnation_id FROM nodes WHERE current_incarnation_id=$1`, inc).Scan(&l.second.NodeID, &l.second.IncarnationID)
	if err != nil {
		r.block("Second-node identity unavailable")
		return
	}
	// Preserve primary failures: later readback cannot demote fail to blocked.
	leases, observations, ok := l.nodeLeaseTimeline(l.second.NodeID, r.At)
	r.check("immutable restart lease and observation timeline", "retain grant counts, deadlines and observed forwarding without modifying projections", fmt.Sprintf("leases=%s; observations=%s", leases, observations), "SELECT serving_leases and contiguous node_observations for recorded second incarnation since AT09", ok)
	restore := l.useSecond()
	actual := l.counters(uuid.Nil)
	r.check("later independent-generation kernel readback", "record later kernel state without replacing failed recovery probe", fmt.Sprintf("available=%t; generation=%s; member_accept_rules=%d", actual.Available, actual.Generation, actual.MemberAcceptRules), "read only nft table after AT20/21 policy inputs; no recovery action", actual.Available)
	restore()
}

// Read stopped storage through docker cp, never start/reconfigure a container
// to obtain evidence. A closed database is copied with any retained WAL.
func (l *live) storedLeaseReadback(node string, stopped bool) map[string]any {
	unavailable := map[string]any{"available": false}
	if !l.installProbe() {
		return unavailable
	}
	var raw []byte
	var err error
	if stopped {
		dir, e := os.MkdirTemp(filepath.Join(l.root, ".agent/tmp"), "lease-readback-")
		if e != nil {
			return unavailable
		}
		defer os.RemoveAll(dir)
		for _, suffix := range []string{"", "-wal", "-shm"} {
			_, e = l.docker("cp", l.name(node)+":/var/lib/vpn-node/state/agent.db"+suffix, filepath.Join(dir, "agent.db"+suffix))
			if suffix == "" && e != nil {
				return unavailable
			}
		}
		raw, err = exec.CommandContext(l.ctx, l.probePath(), "lease", filepath.Join(dir, "agent.db")).Output()
	} else {
		if _, e := l.docker("cp", l.probePath(), l.name(node)+":/tmp/at-members-probe"); e != nil {
			return unavailable
		}
		raw, err = l.inside(node, "/tmp/at-members-probe", "lease", "/var/lib/vpn-node/state/agent.db")
	}
	var result map[string]any
	if err != nil || json.Unmarshal(raw, &result) != nil {
		return unavailable
	}
	if !stopped {
		id, e := l.inside(node, "cat", "/var/lib/vpn-node/e2e/agent-process-id")
		if e == nil {
			result["process_id"] = strings.TrimSpace(string(id))
		}
	}
	return result
}
func (l *live) recordAgentTransition(r *record, label string, started, ended time.Time, before, after map[string]any, err error) {
	status := "unavailable"
	if before["available"] == true && after["available"] == true {
		status = "absent before transition"
		if before["present"] == true {
			status = "lost"
			if after["present"] == true {
				status = "replaced; exact prior lease survival unproven"
				if before["lease_id"] == after["lease_id"] && before["envelope_sha256"] == after["envelope_sha256"] && before["mono_sent_ns"] == after["mono_sent_ns"] {
					status = "survived unchanged"
				}
			}
		}
	}
	var receipt json.RawMessage
	if id, ok := before["process_id"].(string); ok && id != "" && filepath.Base(id) == id && id != "." {
		dir, e := os.MkdirTemp(filepath.Join(l.root, ".agent/tmp"), "exit-readback-")
		if e == nil {
			defer os.RemoveAll(dir)
			path := filepath.Join(dir, "exit.json")
			if _, e = l.docker("cp", l.name(l.nodeContainer())+":/var/lib/vpn-node/e2e/agent-exits/"+id+".json", path); e == nil {
				raw, e := os.ReadFile(path)
				if e == nil && json.Valid(raw) {
					receipt = raw
				}
			}
		}
	}
	r.jsonCheck("agent transition "+label, "record command duration and stored lease continuity", map[string]any{"parent_wait_receipt": receipt, "sigterm_to_exit": "see separate r18 M-2 direct stop if present; otherwise exact signal/exit interval not observable here", "control_io_after_cancel": "not observable here", "started_at": started, "ended_at": ended, "duration_ns": fmt.Sprint(ended.Sub(started).Nanoseconds()), "stored_lease_before": before, "stored_lease_after": after, "stored_lease_status": status, "command_ok": err == nil}, "read-only SQLite lease metadata before/after; no renewal or repair requested", err == nil && before["available"] == true && after["available"] == true)
}
