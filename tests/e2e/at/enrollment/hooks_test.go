//go:build e2e

package enrollment

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
)

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
		"machine_id_sha256": h.MachineID, "current_target": h.Current,
	}
}

func installerOutcomeView(out e2e.InstallerOutcome, err error) map[string]any {
	// Never publish Message, stderr, certificates or token-file contents.
	return map[string]any{
		"exit": out.Exit, "status": out.Status, "code": out.Code, "wrapper_error": err != nil,
		"node_id": out.NodeID, "incarnation_id": out.Incarnation,
		"failing_step": installerFailingStep(out),
		"retired_mode": out.RetiredMode, "retired": installerHashesView(out.RetiredHashes),
		"bound_state_before_sha256": out.BoundStateBefore, "retired_bound_state_sha256": out.RetiredBoundState,
		"before": installerHashesView(out.Before), "after": installerHashesView(out.Hashes), "services_to_restore": out.Services,
	}
}

func installerHistoryView(page api.NodeIncarnationList) []map[string]any {
	result := []map[string]any{}
	for _, item := range page.Items {
		reason, _ := item.RetireReason.Get()
		result = append(result, map[string]any{"incarnation_id": item.IncarnationId, "current": item.Current,
			"state": item.State, "retire_reason": reason, "usage": usageView(item.Usage)})
	}
	return result
}

// Allowlist the step, never persist arbitrary CLI messages or stderr.
func installerFailingStep(out e2e.InstallerOutcome) string {
	if out.Exit == 0 {
		return "none"
	}
	for _, unit := range []string{"fvpn-netbird.service", "fvpn-agent.service", "fvpn-watchdog.service", "fvpn-updater.timer"} {
		if strings.Contains(out.Message, "starting "+unit+" failed") {
			return "start " + unit
		}
	}
	for _, step := range []string{"enrollment failed", "updater seed failed", "seed baseline", "dependency"} {
		if strings.Contains(out.Message, step) {
			return step
		}
	}
	return "unclassified; private CLI diagnostic retained"
}

func (p *probe) installerDiagnostics(phase string, out e2e.InstallerOutcome) map[string]any {
	const container = "fvpn-w4-enroll-installer-node"
	result := map[string]any{"observed_at": time.Now().UTC().Format(time.RFC3339Nano)}
	logs, logErr := exec.CommandContext(p.ctx, "docker", "exec", container, "journalctl",
		"-u", "fvpn-netbird.service", "-u", "fvpn-agent.service", "-u", "fvpn-watchdog.service", "-u", "fvpn-updater.service", "--no-pager", "-n", "200", "-o", "cat").Output()
	result["journal_read_ok"] = logErr == nil
	result["netbird_socket_readonly"] = bytes.Contains(logs, []byte("listen unix /var/run/netbird.sock: bind: read-only file system"))
	result["agent_heartbeat_readonly"] = bytes.Contains(logs, []byte("mkdir /run/vpn-node: read-only file system"))
	// One scan row per phase; the initial row keeps the previous AT06 name.
	if phase == "initial" {
		p.scanStream("installer-node/journal", logs, logErr)
	} else {
		first := len(p.checks)
		p.scanStream("installer-node/"+phase+"-journal", logs, logErr)
		for i := first; i < len(p.checks); i++ {
			p.checks[i].AT = "AT07"
		}
	}
	units := map[string]any{}
	var private bytes.Buffer
	private.Write(logs)
	private.WriteString("\nCLI message (private): " + out.Message + "\n")
	for _, unit := range []string{"fvpn-netbird.service", "fvpn-agent.service", "fvpn-watchdog.service", "fvpn-updater.service", "fvpn-updater.timer"} {
		raw, err := exec.CommandContext(p.ctx, "docker", "exec", container, "systemctl", "show", unit,
			"-p", "LoadState", "-p", "ActiveState", "-p", "SubState", "-p", "Result", "-p", "ExecMainStatus").Output()
		fields := map[string]any{"read_ok": err == nil}
		for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
			key, value, ok := strings.Cut(line, "=")
			if ok {
				fields[key] = value
			}
		}
		units[unit] = fields
		private.WriteString("\n" + unit + "\n")
		private.Write(raw)
	}
	result["units"] = units
	sockets := map[string]any{}
	for _, path := range []string{"/run/fvpn-netbird/netbird.sock", "/var/run/netbird.sock"} {
		sockets[path] = exec.CommandContext(p.ctx, "docker", "exec", container, "test", "-S", path).Run() == nil
	}
	mode, modeErr := exec.CommandContext(p.ctx, "docker", "exec", container, "stat", "-c", "%a", "/run/fvpn-netbird").Output()
	sockets["directory_mode"] = strings.TrimSpace(string(mode))
	sockets["directory_stat_ok"] = modeErr == nil
	result["sockets"] = sockets
	result["private_diagnostics_saved"] = os.WriteFile(filepath.Join(filepath.Dir(p.resultPath), "installer-"+phase+".log"), private.Bytes(), 0600) == nil
	return result
}

func (p *probe) installerRetiredCertificate() (map[string]any, bool) {
	const container = "fvpn-w4-enroll-installer-node"
	result := map[string]any{}
	allDenied := true
	// Read-only requests with the product-retired files in their original
	// container. Use direct mTLS, since the loopback adapter uses current identity.
	for _, route := range []string{"desired-state", "observations"} {
		script := `set -eu
for d in /var/lib/vpn-node/identity.retired/*; do [ -d "$d" ] && retired="$d"; done
exec curl --silent --show-error --max-time 15 --cacert /var/tmp/e2e-rel/ca.pem --cert "$retired/cert.pem" --key "$retired/key.pem" -w '\n%{http_code}' "$@"`
		args := []string{"exec", container, "sh", "-c", script, "sh"}
		if route == "observations" {
			args = append(args, "-H", "Content-Type: application/json", "--data", "{}")
		}
		args = append(args, "https://agent.example.invalid:9443/api/v1/agent/"+route)
		raw, err := exec.CommandContext(p.ctx, "docker", args...).Output()
		body, status, found := strings.Cut(string(raw), "\n")
		// JSON may contain newlines; split at the last newline from curl instead.
		if i := bytes.LastIndexByte(raw, '\n'); i >= 0 {
			body, status, found = string(raw[:i]), string(raw[i+1:]), true
		}
		var response struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		decoded := json.Unmarshal([]byte(body), &response) == nil
		denied := err == nil && found && decoded && status == "403" && response.Error.Code == "IDENTITY_REVOKED"
		allDenied = allDenied && denied
		result[route] = map[string]any{"http_status": status, "code": response.Error.Code, "decoded": decoded,
			"denied": denied}
	}
	return result, allDenied
}

func sameStrings(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func (p *probe) restoreInstaller(phase string, out e2e.InstallerOutcome, detail map[string]any) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(p.ctx), 30*time.Second)
	defer cancel()
	err := p.stack.RestoreInstallerServices(ctx, "installer-node", out.Services)
	got, readErr := p.stack.InstallerServices(ctx, "installer-node")
	ok := err == nil && readErr == nil && len(got) == 4
	for unit, want := range out.Services {
		if want == "activating" {
			want = "active"
		}
		if got[unit] != want {
			ok = false
		}
	}
	observation := map[string]any{"phase": phase, "expected": out.Services, "observed": got, "restore_ok": err == nil, "read_ok": readErr == nil}
	detail["services_after_"+phase] = observation
	p.expect("AT07", "installer_service_restoration", ok, observation)
}

// The installer consumers are stopped by RerunInstaller. The authenticated
// proxy remains live; malformed POSTs test route scope without fabricating usage.
func (p *probe) installerUsageHold() {
	detail := map[string]any{"slot": "installer-node", "scope": "non-serving endpoint scope and quiescent ledger; no counted member usage"}
	ok := false
	defer func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(p.ctx), 20*time.Second)
		defer cancel()
		released := p.stack.HoldUsageAcknowledgements(ctx, "installer-node", false) == nil
		detail["cleanup_released"] = released
		p.expect("AT07", "installer_usage_hold_release", ok && released, detail)
	}()
	before, e1 := p.stack.UsageLedgerSnapshotBytes(p.ctx, "installer-node")
	var snapshot e2e.UsageSnapshot
	decoded := json.Unmarshal(before, &snapshot) == nil
	detail["snapshot_before"], detail["snapshot_read_ok"] = snapshot, e1 == nil && decoded
	_, transportErr := p.stack.MemberTransportStack("fvpn-w4-enroll-member", "installer-node")
	detail["member_transport_export_available"] = transportErr == nil
	if p.stack.HoldUsageAcknowledgements(p.ctx, "installer-node", true) != nil {
		return
	}
	hs, hc, hr, hok := p.installerRoute("usage-batches")
	ds, _, _, dok := p.installerRoute("desired-state")
	held, e2 := p.stack.UsageLedgerSnapshotBytes(p.ctx, "installer-node")
	if p.stack.HoldUsageAcknowledgements(p.ctx, "installer-node", false) != nil {
		return
	}
	rs, rc, _, rok := p.installerRoute("usage-batches")
	after, e3 := p.stack.UsageLedgerSnapshotBytes(p.ctx, "installer-node")
	stable := e1 == nil && e2 == nil && e3 == nil && bytes.Equal(before, held) && bytes.Equal(before, after)
	detail["held_http_status"], detail["held_code"], detail["held_retryable"] = hs, hc, hr
	detail["other_route_status"], detail["released_http_status"], detail["released_code"] = ds, rs, rc
	detail["snapshot_bytes_equal"], detail["snapshot_sha256"] = stable, hash(before)
	ok = stable && decoded && transportErr == nil && hok && hs == 503 && hc == "DEPENDENCY_UNAVAILABLE" && hr && dok && ds == 200 && rok && rs == 422 && rc == "VALIDATION_FAILED"
}

func (p *probe) installerRoute(route string) (int, string, bool, bool) {
	args := []string{"exec", "fvpn-w4-enroll-installer-node", "curl", "--silent", "--show-error", "--max-time", "15",
		"--cacert", "/var/tmp/e2e-rel/ca.pem", "--cert", "/var/lib/vpn-node/identity/cert.pem", "--key", "/var/lib/vpn-node/identity/key.pem", "-w", "\n%{http_code}"}
	if route == "usage-batches" {
		args = append(args, "-H", "Content-Type: application/json", "--data", "{}")
	}
	raw, err := exec.CommandContext(p.ctx, "docker", append(args, "https://127.0.0.1:9444/api/v1/agent/"+route)...).Output()
	i := bytes.LastIndexByte(raw, '\n')
	if i < 0 {
		return 0, "", false, false
	}
	status, _ := strconv.Atoi(string(raw[i+1:]))
	var body struct {
		Error struct {
			Code      string `json:"code"`
			Retryable bool   `json:"retryable"`
		} `json:"error"`
	}
	decoded := json.Unmarshal(raw[:i], &body) == nil
	return status, body.Error.Code, body.Error.Retryable, err == nil && decoded
}
