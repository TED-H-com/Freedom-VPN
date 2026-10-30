//go:build e2e

package e2e

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// Restart before members are admitted: this never substitutes a restart for
// renewal of a member lease after a desired-generation change.
func (r *runner) restartNode(ctx context.Context) {
	rec := Record{ID: "node-restart", Title: "Enrolled node resumes without an installer token", Spec: []string{"§5", "§6", "F2-05"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	conn, err := r.lab.db(ctx)
	if err != nil {
		note("database", "unavailable")
		return
	}
	defer conn.Close(ctx)
	snapshot := func() (string, error) {
		var row string
		err := conn.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(jsonb_build_object('id',id,'state',state,'consumed_at',consumed_at) ORDER BY id)::text,'[]') FROM node_enrollments WHERE node_id=$1`, r.nodeID).Scan(&row)
		return row, err
	}
	before, err := snapshot()
	if err != nil {
		note("enrollments", "query failed")
		return
	}
	identity, err := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/identity/node.json")
	if err != nil {
		return
	}
	oldBoot, _ := r.lab.docker(r.lab.node, "cat", "/etc/vpn-node/boot-id")
	tokenAbsent := !r.lab.fileNonEmpty(r.lab.node, "/var/lib/vpn-node/e2e/enrollment.token")
	note("installer_token_absent_before", boolText(tokenAbsent))
	started := time.Now()
	// Docker stop time is separate from the 30-second watchdog resume deadline.
	if err := exec.CommandContext(ctx, "docker", "restart", "--time", "5", r.lab.node).Run(); err != nil {
		note("restart", "failed")
		return
	}
	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var heartbeat, boot []byte
	var stage string
	var tableOK, accepted bool
	ok := waitUntil(wait, time.Second, func() bool {
		stage = r.lab.stage(r.lab.node, "/var/lib/vpn-node/e2e/status.json")
		heartbeat, _ = r.lab.docker(r.lab.node, "cat", "/run/vpn-node/heartbeat")
		boot, _ = r.lab.docker(r.lab.node, "cat", "/etc/vpn-node/boot-id")
		_, err := r.lab.docker(r.lab.node, "nft", "list", "table", "inet", "fvpn")
		tableOK = err == nil
		rowErr := conn.QueryRow(wait, `SELECT EXISTS(SELECT 1 FROM node_observations WHERE node_id=$1 AND incarnation_id=$2 AND received_at >= $3)`, r.nodeID, r.incID, started).Scan(&accepted)
		return stage == "enrolled" && tableOK && rowErr == nil && accepted && string(boot) != string(oldBoot) && strings.Contains(string(heartbeat), "boot_id "+strings.TrimSpace(string(boot)))
	})
	after, err := snapshot()
	sameIdentity, _ := r.lab.docker(r.lab.node, "cat", "/var/lib/vpn-node/identity/node.json")
	note("resume_within_watchdog_deadline", boolText(ok))
	note("nft_table_readable", boolText(tableOK))
	note("new_observation_same_incarnation", boolText(accepted))
	dns, dnsErr := r.lab.docker(r.lab.node, "fvpn-e2e-nodeconfig", "dns-probe")
	note("resolver_after_restart", strings.TrimSpace(string(dns)))
	note("restart_elapsed_ms", itoa(int(time.Since(started).Milliseconds())))
	note("status_stage", stage)
	note("heartbeat", strings.TrimSpace(string(heartbeat)))
	note("same_node_and_incarnation", boolText(string(identity) == string(sameIdentity)))
	note("enrollment_rows_unchanged", boolText(err == nil && before == after))
	note("installer_token_absent_after", boolText(!r.lab.fileNonEmpty(r.lab.node, "/var/lib/vpn-node/e2e/enrollment.token")))
	if ok && dnsErr == nil && err == nil && before == after && tokenAbsent && string(identity) == string(sameIdentity) {
		rec.Status = StatusPass
	}
}
