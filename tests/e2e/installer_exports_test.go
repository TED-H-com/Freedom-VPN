//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func (r *runner) exportsInstaller(ctx context.Context, s *Stack, admin BrowserSession) {
	rec := Record{ID: "installer-wrappers", Title: "RerunInstaller and ControlledReinstall on the systemd installer node", Spec: []string{"§6.3", "ENR-06"}, Status: StatusFail}
	note := recPathNote(&rec)
	defer func() { r.record(rec) }()
	if err := s.PrepareInstallerNode(ctx); err != nil {
		note("prepare", err.Error())
		return
	}
	note("container", s.installerContainer())
	note("regular_node", "unchanged")
	argv, err := s.ProbeInstallerExit(ctx, "installer-node", []string{"install", "--token", "not-a-token"})
	if err != nil {
		note("exit_64_argv", err.Error())
		return
	}
	note("exit_64_argv", itoa(argv.Exit))
	if argv.Exit != 64 {
		return
	}
	if err = writeContainerFile(ctx, s.installerContainer(), "/var/tmp/e2e-rel/loose.token", "644", []byte("placeholder\n")); err != nil {
		note("loose_token", "failed")
		return
	}
	loose, err := s.ProbeInstallerExit(ctx, "installer-node", []string{"install", "--token-file", "/var/tmp/e2e-rel/loose.token"})
	if err != nil {
		note("exit_64_mode", err.Error())
		return
	}
	note("exit_64_mode", itoa(loose.Exit))
	if loose.Exit != 64 {
		return
	}
	token, nodeName, err := r.issueInstallerToken(ctx, s, admin, "e2e installer node", uuid.Nil)
	if err != nil {
		note("enrollment", err.Error())
		return
	}
	if err = s.StageInstallerToken(ctx, "installer-node", token); err != nil {
		note("token_file", "failed")
		return
	}
	token = ""
	installed, err := s.RerunInstaller(ctx, "installer-node")
	if err != nil {
		note("install", err.Error())
		return
	}
	note("install_exit", itoa(installed.Exit))
	note("install_status", emptyDash(installed.Status))
	note("install_incarnation", emptyDash(installed.Incarnation))
	if installed.Exit != 0 || (installed.Status != "installed" && installed.Status != "repaired") || installed.NodeID == "" {
		note("install_code", installed.Code)
		note("owner", "packaging/installer + deploy/node/systemd: real NetBird service startup")
		logs, _ := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "journalctl", "-u", "fvpn-netbird.service", "-u", "fvpn-agent.service", "--no-pager", "-n", "60", "-o", "cat").Output()
		note("netbird_socket_readonly", boolText(strings.Contains(string(logs), "listen unix /var/run/netbird.sock: bind: read-only file system")))
		note("agent_heartbeat_readonly", boolText(strings.Contains(string(logs), "mkdir /run/vpn-node: read-only file system")))
		note("rerun_reenroll_success", "not reached: initial install failed; no service override or NetBird stub")
		note("install_message", clipText(installed.Message, 240))
		return
	}
	again, err := s.RerunInstaller(ctx, "installer-node")
	if err != nil {
		note("again", err.Error())
		return
	}
	note("again_exit", itoa(again.Exit))
	note("again_status", emptyDash(again.Status))
	note("hashes_stable", boolText(again.Exit == 0 && again.Status == "already_current" && sameInstallerHashes(installed.Hashes, again.Hashes)))
	if again.Exit != 0 || again.Status != "already_current" || !sameInstallerHashes(installed.Hashes, again.Hashes) {
		return
	}
	if out, err := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "chmod", "0755", "/var/lib/vpn-node/state").CombinedOutput(); err != nil {
		note("drift_setup", clipText(string(out), 120))
		return
	}
	drift, err := s.ProbeInstallerExit(ctx, "installer-node", []string{"check", "--json"})
	if err != nil {
		note("exit_2", err.Error())
		return
	}
	note("exit_2", itoa(drift.Exit))
	if drift.Exit != 2 {
		return
	}
	repaired, err := s.RerunInstaller(ctx, "installer-node")
	if err != nil {
		note("repair", err.Error())
		return
	}
	mode, _ := s.fileMode(ctx, "/var/lib/vpn-node/state")
	note("repair_status", emptyDash(repaired.Status))
	note("state_mode", emptyDash(mode))
	if repaired.Exit != 0 || repaired.Status != "repaired" || mode != "700" {
		return
	}
	hold := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "sh", "-c", `flock -x /var/lib/vpn-node/update.lock sleep 30 >/dev/null 2>&1 & echo $! > /var/tmp/e2e-rel/flock.pid`)
	if out, err := hold.CombinedOutput(); err != nil {
		note("lock", clipText(string(out), 120))
		return
	}
	time.Sleep(300 * time.Millisecond)
	spec, err := s.readInstallerSpec(ctx)
	if err != nil {
		note("spec", err.Error())
		return
	}
	busy, err := s.ProbeInstallerExit(ctx, "installer-node", s.installArgs(spec)[1:])
	_, _ = exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "sh", "-c", `kill "$(cat /var/tmp/e2e-rel/flock.pid)" 2>/dev/null || true`).CombinedOutput()
	if err != nil {
		note("exit_3", err.Error())
		return
	}
	note("exit_3", itoa(busy.Exit))
	note("exit_3_code", emptyDash(busy.Code))
	if busy.Exit != 3 {
		return
	}
	if err = exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "cp", "/usr/bin/netbird", "/usr/bin/netbird.e2e-orig").Run(); err != nil {
		note("netbird_copy", "failed")
		return
	}
	badStub := "#!/bin/sh\nif [ \"${1:-}\" = version ]; then printf '%s\\n' 9.9.9; exit 0; fi\nexec /usr/bin/netbird.e2e-orig \"$@\"\n"
	if err = writeContainerFile(ctx, s.installerContainer(), "/usr/bin/netbird", "755", []byte(badStub)); err != nil {
		note("netbird_stub", "failed")
		return
	}
	dep, err := s.ProbeInstallerExit(ctx, "installer-node", s.installArgs(spec)[1:])
	_ = exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "cp", "/usr/bin/netbird.e2e-orig", "/usr/bin/netbird").Run()
	if err != nil {
		note("exit_5", err.Error())
		return
	}
	note("exit_5", itoa(dep.Exit))
	note("exit_5_code", emptyDash(dep.Code))
	if dep.Exit != 5 || dep.Code != "NETBIRD_VERSION" {
		return
	}
	upgrade, err := s.ProbeInstallerExit(ctx, "installer-node", []string{"upgrade", "--json", "--manifest", "/var/tmp/e2e-rel/newer/manifest.json", "--artifact", "/var/tmp/e2e-rel/newer/artifact.tar.zst", "--trust", "/var/tmp/e2e-rel/newer/trust.json"})
	if err != nil {
		note("upgrade", err.Error())
		return
	}
	note("upgrade_exit", itoa(upgrade.Exit))
	note("upgrade_status", emptyDash(upgrade.Status))
	if upgrade.Status != "staged" {
		note("upgrade_message", clipText(upgrade.Message, 200))
		return
	}
	if err = s.SelectInstallerRelease(ctx, "newer"); err != nil {
		note("newer", err.Error())
		return
	}
	newerRun, err := s.RerunInstaller(ctx, "installer-node")
	if err != nil {
		note("newer_install", err.Error())
		return
	}
	note("newer_status", emptyDash(newerRun.Status))
	note("current_changed", boolText(newerRun.Hashes.Current != "" && newerRun.Hashes.Current != again.Hashes.Current))
	note("identity_unchanged", boolText(sameExceptCurrent(again.Hashes, newerRun.Hashes)))
	if newerRun.Exit != 0 || newerRun.Hashes.Current == again.Hashes.Current || !sameExceptCurrent(again.Hashes, newerRun.Hashes) {
		return
	}
	down, err := s.ProbeInstallerExit(ctx, "installer-node", append([]string{"install", "--json", "--manifest", "/var/tmp/e2e-rel/older/manifest.json", "--artifact", "/var/tmp/e2e-rel/older/artifact.tar.zst", "--trust", "/var/tmp/e2e-rel/older/trust.json"}, s.installArgs(spec)[9:]...))
	if err != nil {
		note("downgrade", err.Error())
		return
	}
	note("downgrade_exit", itoa(down.Exit))
	note("downgrade_code", emptyDash(down.Code))
	if down.Exit != 1 || down.Code != "DOWNGRADE" {
		return
	}
	imagePrep, err := s.ProbeInstallerExit(ctx, "installer-node", []string{"image-prepare"})
	if err != nil {
		note("identity_present", err.Error())
		return
	}
	note("identity_present_exit", itoa(imagePrep.Exit))
	note("identity_present_code", emptyDash(imagePrep.Code))
	if imagePrep.Exit != 1 || imagePrep.Code != "identity_present" {
		return
	}
	note("exit_4", "UNSUPPORTED_PLATFORM is covered by tests/installer; this container is Ubuntu 24.04")
	beforeDB := newerRun.Hashes.UsageDB
	nodeID, err := uuid.Parse(installed.NodeID)
	if err != nil {
		note("node_id", "unreadable")
		return
	}
	if !r.revokeInstallerNode(ctx, s, admin, nodeID, note) {
		return
	}
	token, _, err = r.issueInstallerToken(ctx, s, admin, nodeName+" reenroll", nodeID)
	if err != nil {
		note("reenroll_token", err.Error())
		return
	}
	reenrolled, err := s.ControlledReinstall(ctx, "installer-node", token)
	token = ""
	if err != nil {
		note("reenroll", err.Error())
		return
	}
	note("reenroll_exit", itoa(reenrolled.Exit))
	note("reenroll_status", emptyDash(reenrolled.Status))
	note("incarnation_changed", boolText(reenrolled.Incarnation != "" && reenrolled.Incarnation != installed.Incarnation))
	note("retired_mode", emptyDash(reenrolled.RetiredMode))
	note("usage_db_same", boolText(reenrolled.Hashes.UsageDB == beforeDB))
	moved := len(reenrolled.BoundStateBefore) == 4
	for name, digest := range reenrolled.BoundStateBefore {
		moved = moved && reenrolled.RetiredBoundState[name] == digest
	}
	retired := reenrolled.RetiredHashes.Key == reenrolled.Before.Key && reenrolled.RetiredHashes.Cert == reenrolled.Before.Cert && reenrolled.RetiredHashes.NodeJSON == reenrolled.Before.NodeJSON
	note("retired_identity_hashes_match", boolText(retired))
	note("stable_node_id", boolText(reenrolled.NodeID == installed.NodeID))
	note("agent_db_moved_aside", boolText(moved))
	note("old_certificate", "ENR-02 stays the harness; this wrapper does not prove central rejects it")
	if reenrolled.Exit != 0 || reenrolled.Status != "reenrolled" || reenrolled.Incarnation == installed.Incarnation || reenrolled.RetiredMode != "700" || reenrolled.Hashes.UsageDB != beforeDB || !moved || !retired || reenrolled.NodeID != installed.NodeID || reenrolled.Hashes.UsageWAL != reenrolled.Before.UsageWAL {
		note("reenroll_message", clipText(reenrolled.Message, 240))
		return
	}
	rec.Status = StatusPass
}

func (r *runner) issueInstallerToken(ctx context.Context, s *Stack, admin BrowserSession, name string, nodeID uuid.UUID) (string, string, error) {
	req := api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name}
	if nodeID != uuid.Nil {
		id := openapi_types.UUID(nodeID)
		req.NodeId = &id
	}
	created, err := s.Call(ctx, admin, "POST", "/api/v1/admin/enrollments", req, map[string]string{"Idempotency-Key": uuid.NewString()})
	if err != nil || created.Status != 201 {
		return "", name, fmt.Errorf("enrollment status %d", created.Status)
	}
	var body api.EnrollmentCreated
	if json.Unmarshal(created.Body, &body) != nil {
		return "", name, fmt.Errorf("enrollment response")
	}
	token, err := body.Token.Get()
	if err != nil || token == "" {
		return "", name, fmt.Errorf("enrollment token absent")
	}
	return token, name, nil
}

func (r *runner) revokeInstallerNode(ctx context.Context, s *Stack, admin BrowserSession, id uuid.UUID, note func(string, string)) bool {
	detail, err := s.Call(ctx, admin, "GET", "/api/v1/admin/nodes/"+id.String(), nil, nil)
	if err != nil || detail.Status != 200 {
		note("revoke_get", itoa(detail.Status))
		return false
	}
	rev, err := s.Call(ctx, admin, "POST", "/api/v1/admin/nodes/"+id.String()+"/revoke", api.ReasonRequest{Reason: "e2e installer reenroll"}, map[string]string{
		"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header.Get("ETag"),
	})
	note("revoke", itoa(rev.Status))
	return err == nil && rev.Status == 202
}

func TestE2E_InstallerExports(t *testing.T) {
	if os.Getenv("FVPN_E2E_INSTALLER") != "1" {
		t.Skip("set FVPN_E2E_INSTALLER=1 with an owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()
	head, _ := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	r := &runner{t: t, lab: l, runID: uuid.NewString(), sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup15", "installer-focus")}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	r.exportsInstaller(ctx, &Stack{lab: l}, BrowserSession{Client: admin.http, CSRF: admin.csrf})
	r.finish()
}

func TestENR06InstallerOutcomeRetainsFailureWithStagedStatus(t *testing.T) {
	out := parseInstallerOutcome([]byte(`{"status":"staged","incarnation_id":"fixture"}`), []byte("updater_failed: apply did not finish\n"), 1)
	if out.Exit != 1 || out.Status != "staged" || out.Code != "updater_failed" {
		t.Fatalf("lost failure: %+v", out)
	}
	for _, tc := range []struct {
		exit int
		code string
	}{{1, "DOWNGRADE"}, {2, "drift"}, {3, "busy"}, {4, "UNSUPPORTED_PLATFORM"}, {5, "NETBIRD_VERSION"}, {64, "usage"}} {
		out := parseInstallerOutcome(nil, []byte("Created symlink unit.\n"+tc.code+": fixture failure\n"), tc.exit)
		if out.Exit != tc.exit || out.Code != tc.code || strings.Contains(out.Message, "symlink") {
			t.Fatalf("lost CLI outcome: %+v", out)
		}
	}
}

func TestENR06InstallerHashesNameTheirDigests(t *testing.T) {
	raw, err := json.Marshal(InstallerHashes{Key: strings.Repeat("a", 64), Cert: "absent"})
	if err != nil || !strings.Contains(string(raw), `"key_sha256"`) || !strings.Contains(string(raw), `"cert_sha256":"absent"`) || strings.Contains(string(raw), `"Key"`) {
		t.Fatalf("ambiguous digest labels: %s, %v", raw, err)
	}
}
