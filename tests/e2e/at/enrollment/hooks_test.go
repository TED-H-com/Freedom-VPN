//go:build e2e

package enrollment

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

const usageInstallerGap = "ENR-05: #266 HoldUsageAcknowledgements, UsageLedgerSnapshot and MemberTransportStack accept node/node2/node-clone, while ControlledReinstall accepts only installer-node. Owner devstack: export the hold, snapshot and counted member transport on that same installer slot, including its proxy/config paths and service restoration. Owner usage/installer: supply an authorized old-incarnation backlog delivery protocol; packaging/installer/reenroll.go explicitly leaves retained old events unuploaded. No central-once or cross-reinstall usage claim."

const installerStartupGap = "ENR-06: #266 real installer startup prerequisite failed. Owner packaging/installer and deploy/node/systemd (installer follow-up 3): fix NetBird socket write permissions and shared RuntimeDirectory lifecycle, then devstack rerun the wrappers and restore prior service state after hash sampling. No working forwarding baseline; successful rerun and ControlledReinstall are not reached."

// This is a narrow availability/snapshot proof on the regular node. It must
// never satisfy counted_unacked_usage: the installer uses a different slot.
func (p *probe) usageHoldScope() {
	detail := map[string]any{
		"slot": "node", "source_pr": 266,
		"mechanism": "network-side mTLS proxy marker /var/lib/vpn-node/e2e/hold-usage; only POST /api/v1/agent/usage-batches returns retryable 503 before control; central ledger epoch unchanged",
		"scope":     "endpoint scope and read-only snapshot only; no counted traffic or reinstall",
	}
	status := "blocked"
	defer func() { p.note("AT07", "usage_hold_endpoint_scope", status, detail) }()
	if err := p.stack.Service(p.ctx, "stop", ""); err != nil {
		detail["gap"] = "devstack Service(stop) prerequisite unavailable for stable snapshots"
		return
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		released := p.stack.HoldUsageAcknowledgements(ctx, "node", false) == nil
		resumed := p.stack.Service(ctx, "start", "/usr/local") == nil
		detail["cleanup_hold_released"], detail["cleanup_agent_resumed"] = released, resumed
		if !released || !resumed {
			status = "blocked"
			detail["gap"] = "devstack usage hold cleanup or service resume failed"
		}
	}()
	before, err := p.stack.UsageLedgerSnapshotBytes(p.ctx, "node")
	if err != nil {
		detail["gap"] = "devstack UsageLedgerSnapshotBytes prerequisite unavailable"
		return
	}
	var snap e2e.UsageSnapshot
	if json.Unmarshal(before, &snap) != nil {
		detail["gap"] = "devstack usage snapshot is not UsageSnapshot JSON"
		return
	}
	detail["snapshot_before"] = snap
	detail["hold_requested_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	if err = p.stack.HoldUsageAcknowledgements(p.ctx, "node", true); err != nil {
		detail["gap"] = "devstack hold request failed"
		return
	}
	detail["hold_applied_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	heldStatus, heldCode, heldRetryable, heldOK := p.usageRoute(true)
	otherStatus, _, _, otherOK := p.usageRoute(false)
	detail["held_status"], detail["held_code"], detail["held_retryable"] = heldStatus, heldCode, heldRetryable
	detail["desired_state_status"] = otherStatus
	held, heldErr := p.stack.UsageLedgerSnapshotBytes(p.ctx, "node")
	detail["release_requested_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	if err = p.stack.HoldUsageAcknowledgements(p.ctx, "node", false); err != nil {
		detail["gap"] = "devstack release request failed"
		return
	}
	detail["released_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	releasedStatus, releasedCode, _, releasedOK := p.usageRoute(true)
	detail["released_status"], detail["released_code"] = releasedStatus, releasedCode
	after, afterErr := p.stack.UsageLedgerSnapshotBytes(p.ctx, "node")
	stable := heldErr == nil && afterErr == nil && bytes.Equal(before, held) && bytes.Equal(before, after)
	detail["paused_snapshot_bytes_unchanged"] = stable
	detail["snapshot_sha256"] = hash(before)
	if heldOK && heldStatus == 503 && heldCode == "DEPENDENCY_UNAVAILABLE" && heldRetryable &&
		otherOK && otherStatus == 200 && releasedOK && releasedStatus == 422 && releasedCode == "VALIDATION_FAILED" && stable {
		status = "pass"
	} else {
		detail["gap"] = "devstack endpoint-scope or snapshot expectation not observed; see typed observations"
	}
}

func (p *probe) usageRoute(post bool) (status int, code string, retryable, ok bool) {
	args := []string{"exec", "fvpn-w4-enroll-node", "wget", "-q", "-S", "-O", "-", "--content-on-error", "--timeout=15",
		"--ca-certificate=/etc/vpn-node/ca.pem", "--certificate=/var/lib/vpn-node/identity/cert.pem", "--private-key=/var/lib/vpn-node/identity/key.pem"}
	path := "/api/v1/agent/desired-state"
	if post {
		args = append(args, "--header=Content-Type: application/json", "--post-data={}")
		path = "/api/v1/agent/usage-batches"
	}
	cmd := exec.CommandContext(p.ctx, "docker", append(args, "https://127.0.0.1:9444"+path)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run() // HTTP errors are the observation; never discard their body.
	for _, line := range strings.Split(stderr.String(), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.HasPrefix(fields[0], "HTTP/") {
			status, _ = strconv.Atoi(fields[1])
		}
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			Retryable bool   `json:"retryable"`
		} `json:"error"`
	}
	decode := json.Unmarshal(stdout.Bytes(), &body)
	return status, body.Error.Code, body.Error.Retryable, status != 0 && decode == nil
}

func installerHashesView(h e2e.InstallerHashes) map[string]string {
	return map[string]string{
		"key_sha256": h.Key, "cert_sha256": h.Cert, "node_json_sha256": h.NodeJSON,
		"usage_db_sha256": h.UsageDB, "usage_wal_sha256": h.UsageWAL,
		"machine_id_sha256": h.MachineID, "current": h.Current,
	}
}

func installerOutcomeView(out e2e.InstallerOutcome, err error) map[string]any {
	// Never publish Message, stderr, certificates or token-file contents.
	return map[string]any{
		"exit": out.Exit, "status": out.Status, "code": out.Code, "wrapper_error": err != nil,
		"node_id": out.NodeID, "incarnation_id": out.Incarnation,
		"before": installerHashesView(out.Before), "after": installerHashesView(out.Hashes),
	}
}

func (p *probe) installerHooks() {
	gap := installerStartupGap
	detail := map[string]any{"source_pr": 266, "slot": "installer-node", "container": "fvpn-w4-enroll-installer-node"}
	defer func() {
		p.note("AT07", "counted_unacked_usage", "blocked", map[string]any{"gap": usageInstallerGap, "slot_probe": detail["slot_probe"], "counted_traffic_started": false, "controlled_reinstall_run": false})
		for _, name := range []string{"installer_rerun", "identity_ledger_preserved", "controlled_reinstall", "old_certificate_denied", "historical_usage_retained"} {
			p.note("AT07", name, "blocked", map[string]any{"gap": gap, "prerequisite": detail,
				"usage_gap": usageInstallerGap, "check": name})
		}
	}()
	// These exports reject the slot before any mutation or command. A common
	// name on Stack does not make the two fixture families composable.
	holdErr := p.stack.HoldUsageAcknowledgements(p.ctx, "installer-node", true)
	_, snapErr := p.stack.UsageLedgerSnapshot(p.ctx, "installer-node")
	_, transportErr := p.stack.MemberTransportStack("fvpn-w4-enroll-member", "installer-node")
	detail["slot_probe"] = map[string]any{
		"hold_unknown_node_slot":      holdErr != nil && holdErr.Error() == "unknown node slot",
		"snapshot_unknown_node_slot":  snapErr != nil && snapErr.Error() == "unknown node slot",
		"transport_unknown_node_slot": transportErr != nil && transportErr.Error() == "unknown node slot",
	}
	if holdErr == nil {
		_ = p.stack.HoldUsageAcknowledgements(p.ctx, "installer-node", false)
	}
	if err := p.stack.PrepareInstallerNode(p.ctx); err != nil {
		gap = "ENR-06: PrepareInstallerNode failed before invocation. Owner devstack: restore the dedicated systemd fixture; diagnostic content withheld."
		return
	}
	detail["image_digest"] = dockerText(p.ctx, "inspect", "--format={{.Image}}", "fvpn-w4-enroll-installer-node")
	token, ok := p.issue("installer-initial", 900)
	if !ok {
		gap = "ENR-06: initial installer token API prerequisite failed"
		return
	}
	if err := p.stack.StageInstallerToken(p.ctx, "installer-node", token.token); err != nil {
		gap = "ENR-06: StageInstallerToken failed; owner devstack"
		return
	}
	detail["install_started_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	installed, err := p.stack.RerunInstaller(p.ctx, "installer-node")
	detail["install_finished_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	detail["initial_install"] = installerOutcomeView(installed, err)
	logs, logErr := exec.CommandContext(p.ctx, "docker", "exec", "fvpn-w4-enroll-installer-node", "journalctl",
		"-u", "fvpn-netbird.service", "-u", "fvpn-agent.service", "--no-pager", "-n", "100", "-o", "cat").Output()
	detail["journal_read_ok"] = logErr == nil
	detail["netbird_socket_readonly"] = bytes.Contains(logs, []byte("listen unix /var/run/netbird.sock: bind: read-only file system"))
	detail["agent_heartbeat_readonly"] = bytes.Contains(logs, []byte("mkdir /run/vpn-node: read-only file system"))
	p.scanStream("installer-node/journal", logs, logErr)
	if node, parseErr := uuid.Parse(installed.NodeID); parseErr == nil {
		detail["projection_after_install"] = p.identityProjection(node)
		if inc, incErr := uuid.Parse(installed.Incarnation); incErr == nil {
			p.oneIncarnation("AT07", node, inc, token.enrollment.EnrollmentId)
		}
	}
	// Wrapper success alone cannot prove forwarding. The wrapper stops the
	// services for hashing, and no installer-slot resume/traffic export exists.
	if err == nil && installed.Exit == 0 {
		gap = "ENR-06: initial install returned success, but #266 wrappers leave agent/watchdog/updater stopped after hashing and expose no installer-slot forwarding/resume helper. Owner devstack: restore prior service state and expose same-slot traffic before rerun/reinstall assertions."
	}
}
