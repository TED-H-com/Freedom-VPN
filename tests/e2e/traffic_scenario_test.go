//go:build e2e

package e2e

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// countedFlow proves admission on this node, not merely an HTTP response on
// another route. The member has no connection to the origin bridge.
func (r *runner) countedFlow(ctx context.Context, note func(string, string)) bool {
	defer func() {
		heartbeat, _ := r.lab.docker(r.lab.node, "cat", "/run/vpn-node/heartbeat")
		note("heartbeat", strings.TrimSpace(string(heartbeat)))
	}()
	ready, stopReady := context.WithTimeout(ctx, 90*time.Second)
	var before uint64
	var ok bool
	found := waitUntil(ready, time.Second, func() bool { before, ok = r.admissionBytes(); return ok && r.lab.forwardChain().Generation > 0 })
	stopReady()
	if !found {
		note("admission_counter", "unavailable after 90 seconds")
		return false
	}
	wait, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	var after uint64
	success := waitUntil(wait, 2*time.Second, func() bool {
		probe, stop := context.WithTimeout(wait, 8*time.Second)
		defer stop()
		cmd := exec.CommandContext(probe, "docker", "exec", r.lab.member, "wget", "--timeout=5", "--tries=1", "--ca-certificate=/etc/vpn-node/ca.pem", "-qO-", r.lab.releases+"/healthz")
		body, err := cmd.Output()
		after, ok = r.admissionBytes()
		return err == nil && strings.TrimSpace(string(body)) == "ok" && ok && after > before
	})
	note("controlled_origin_http_200_and_counted", boolText(success))
	note("admission_bytes_before", strconv.FormatUint(before, 10))
	note("admission_bytes_after", strconv.FormatUint(after, 10))
	return success
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
		note("traffic", "not claimed; no admission counter increment. Inspect lease generation/member count versus desired bindings.")
	}
}
