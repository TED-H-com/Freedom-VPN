//go:build e2e

package e2e

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

// countedFlow proves admission on this node, not merely an HTTP response on
// another route. The member has no connection to the origin bridge.
func (r *runner) countedFlow(ctx context.Context, note func(string, string)) bool {
	r.trafficOwner = "w2-agent-core: device admission counters absent after bounded readiness poll; see accepted serving checks"
	defer func() {
		heartbeat, _ := r.lab.docker(r.lab.node, "cat", "/run/vpn-node/heartbeat")
		note("heartbeat", strings.TrimSpace(string(heartbeat)))
	}()
	ready, stopReady := context.WithTimeout(ctx, 90*time.Second)
	var ok bool
	found := waitUntil(ready, time.Second, func() bool { _, ok = r.admissionBytes(); return ok && r.lab.forwardChain().Generation > 0 })
	stopReady()
	if !found {
		note("admission_counter", "unavailable after 90 seconds")
		return false
	}
	if !r.selectExit(ctx, note) {
		r.trafficOwner = "w3-devstack-e2e: native client exit selection; see route and NetBird network evidence"
		note("owner", r.trafficOwner)
		return false
	}
	return r.tracedFetch(ctx, note)
}

func (r *runner) admissionBytes() (uint64, bool) {
	raw, err := r.lab.docker(r.lab.node, "nft", "-j", "list", "table", "inet", "fvpn")
	if err != nil {
		return 0, false
	}
	st, err := enforce.ParseState(raw)
	if err != nil {
		return 0, false
	}
	suffix := strings.ReplaceAll(r.usageDevice.String(), "-", "")
	var total uint64
	for _, key := range []string{"u_" + suffix, "d_" + suffix} {
		c, ok := st.Counters[key]
		if !ok {
			return 0, false
		}
		total += c.Bytes
	}
	return total, true
}

func (r *runner) noteAdmissionGate(ctx context.Context, admin *session, rec *Record) {
	note := recPathNote(rec)
	mode, _ := r.lab.docker(r.lab.node, "stat", "-c", "%u:%a", "/etc/vpn-node/updater-recover.json")
	note("persistent_boot_recovery_config_owner_mode", strings.TrimSpace(string(mode)))
	detail, err := admin.call(ctx, "GET", r.lab.public+"/api/v1/admin/nodes/"+r.nodeID.String(), nil, nil)
	note("admin_node", httpDetail(detail, err))
	if err == nil && detail.Status == 200 {
		noteNode(rec, detail.Body)
	}
	conn, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer conn.Close(ctx)
	// The heartbeat can precede its asynchronous central projection. Wait for
	// one coherent generation snapshot before recording the stored rows.
	alignedCtx, stop := context.WithTimeout(ctx, 30*time.Second)
	aligned := waitUntil(alignedCtx, time.Second, func() bool {
		var same bool
		return conn.QueryRow(alignedCtx, `SELECT n.desired_config_generation=n.applied_config_generation AND n.applied_config_generation=(SELECT s.policy_generation FROM serving_leases s WHERE s.node_id=n.id ORDER BY s.issued_at DESC LIMIT 1) FROM nodes n WHERE n.id=$1`, r.nodeID).Scan(&same) == nil && same
	})
	stop()
	note("stored_lease_matches_applied_and_desired_generation", boolText(aligned))
	var raw []byte
	err = conn.QueryRow(ctx, `SELECT jsonb_build_object(
 'desired_generation',n.desired_config_generation::text,
 'applied_generation',n.applied_config_generation::text,
 'capacity',n.capacity,
 'latest_lease',(SELECT jsonb_build_object('generation',s.policy_generation::text,'issued_at',s.issued_at,'expires_at',s.expires_at,'member_count',jsonb_array_length(s.envelope->'signed'->'members')) FROM serving_leases s WHERE s.node_id=n.id ORDER BY s.issued_at DESC LIMIT 1),
 'lease_count',(SELECT count(*) FROM serving_leases s WHERE s.node_id=n.id),
 'observation',(SELECT jsonb_build_object('checks',o.report_payload->'checks','member_forwarding_enabled',o.report_payload->'member_forwarding_enabled','forwarding_block_reasons',o.report_payload->'forwarding_block_reasons') FROM node_observations o WHERE o.node_id=n.id ORDER BY received_at DESC LIMIT 1)) FROM nodes n WHERE n.id=$1`, r.nodeID).Scan(&raw)
	if err == nil {
		note("serving_gate_rows", string(raw))
	} else {
		note("serving_gate_rows", "query failed")
	}
	policies, bindings, intent, generation := r.desiredMemberView()
	note("desired_member_rate_policies", itoa(policies))
	note("desired_device_bindings", itoa(bindings))
	note("desired_generation", generation)
	note("desired_publication_intent", intent)
	if !r.flowCounted {
		note("traffic", "not claimed; no admission counter increment. Inspect route selection, tunnel captures and named rule counters.")
	}
}

// noteReadyLeaseDelay records the observed gap between the first ready
// observation and the next serving lease whose members array is non-empty.
// D-067 is a product renew. This does not restart the agent to force one.
func (r *runner) noteReadyLeaseDelay(ctx context.Context, rec *Record) {
	note := recPathNote(rec)
	if r.nodeID == uuid.Nil {
		note("ready_to_populated_lease", "node absent")
		return
	}
	conn, err := r.lab.db(ctx)
	if err != nil {
		note("ready_to_populated_lease", "database unavailable")
		return
	}
	defer conn.Close(ctx)
	var readyAt, populatedAt sql.NullTime
	var latestMembers sql.NullInt32
	err = conn.QueryRow(ctx, `
SELECT
  (SELECT min(received_at) FROM node_observations WHERE node_id=$1 AND report_payload->>'provisioning_state'='ready'),
  (SELECT min(s.issued_at) FROM serving_leases s
    WHERE s.node_id=$1
      AND s.issued_at >= (SELECT min(received_at) FROM node_observations WHERE node_id=$1 AND report_payload->>'provisioning_state'='ready')
      AND jsonb_typeof(s.envelope->'signed'->'members')='array'
      AND jsonb_array_length(s.envelope->'signed'->'members') > 0),
  (SELECT jsonb_array_length(s.envelope->'signed'->'members') FROM serving_leases s
    WHERE s.node_id=$1 AND jsonb_typeof(s.envelope->'signed'->'members')='array'
    ORDER BY s.issued_at DESC LIMIT 1)`, r.nodeID).Scan(&readyAt, &populatedAt, &latestMembers)
	if err != nil {
		note("ready_to_populated_lease", "query failed")
		return
	}
	if readyAt.Valid {
		note("provisioning_ready_at", readyAt.Time.UTC().Format(time.RFC3339Nano))
	} else {
		note("provisioning_ready_at", "not observed")
	}
	if populatedAt.Valid {
		note("populated_lease_at", populatedAt.Time.UTC().Format(time.RFC3339Nano))
	} else {
		note("populated_lease_at", "not observed")
	}
	if latestMembers.Valid {
		note("latest_lease_member_count", itoa(int(latestMembers.Int32)))
	} else {
		note("latest_lease_member_count", "not observed")
	}
	if readyAt.Valid && populatedAt.Valid {
		note("ready_to_populated_lease_seconds", strconv.FormatFloat(populatedAt.Time.Sub(readyAt.Time).Seconds(), 'f', 3, 64))
		return
	}
	note("ready_to_populated_lease_seconds", "not observed")
	note("ready_to_populated_lease_owner", "w2-node-agent / w1-leases: D-067 renews within 60s of ready; the lab does not restart the agent to force a populated members lease")
}
