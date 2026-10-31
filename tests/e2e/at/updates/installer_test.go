//go:build e2e

package updates_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	e2e "github.com/TED-H-com/Freedom-VPN/tests/e2e"
	at "github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates"
	"github.com/google/uuid"
)

const installerPending = "/var/lib/vpn-node/update-staging/pending-release.json"
const installerConfig = "/etc/vpn-node/updater.json"
const installedUpdater = "/opt/vpn-node/current/fvpn-updater"

type installerCommand struct {
	Command []string `json:"command"`
	Exit    int      `json:"exit"`
	Stdout  string   `json:"stdout"`
	Stderr  string   `json:"stderr"`
}

type handoffRun struct {
	s      *updatesFollowup
	stack  *e2e.Stack
	rec    at.Record
	checks map[string]bool
}

func (h *handoffRun) note(name string, v any) {
	raw, _ := json.Marshal(v)
	h.rec.Observations = append(h.rec.Observations, at.Observation{Name: name, Detail: string(raw)})
}
func (h *handoffRun) check(name string, passed bool) {
	h.checks[name] = passed
	h.note("check_"+name, passed)
}
func (h *handoffRun) run(name string, args ...string) installerCommand {
	ctx, cancel := context.WithTimeout(h.s.ctx, 3*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", append([]string{"exec", h.s.r.lab.prefix + "-installer-node"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else {
			code = -1
		}
	}
	out := installerCommand{args, code, strings.TrimSpace(stdout.String()), strings.TrimSpace(stderr.String())}
	if name != "" {
		h.note(name, out)
	}
	return out
}
func (h *handoffRun) put(path string, raw []byte) error {
	cmd := exec.CommandContext(h.s.ctx, "docker", "exec", "-i", h.s.r.lab.prefix+"-installer-node", "sh", "-c", `umask 077; mkdir -p "$(dirname "$1")"; cat > "$1"`, "sh", path)
	cmd.Stdin = bytes.NewReader(raw)
	return cmd.Run()
}
func (h *handoffRun) copy(from, to string) error {
	return exec.CommandContext(h.s.ctx, "docker", "cp", from, h.s.r.lab.prefix+"-installer-node:"+to).Run()
}
func (h *handoffRun) status(name string) updateDocument {
	out := h.run(name, installedUpdater, "status", "--config", installerConfig)
	return document([]byte(out.Stdout))
}
func (h *handoffRun) current() string         { return h.run("", "readlink", "/opt/vpn-node/current").Stdout }
func (h *handoffRun) absent(path string) bool { return h.run("", "test", "!", "-e", path).Exit == 0 }
func (h *handoffRun) central(name, node string) updateDocument {
	resp, err := h.s.admin.call(h.s.ctx, "GET", h.s.r.lab.public+"/api/v1/admin/nodes/"+node, nil, nil)
	h.note(name+"_http", httpDetail(resp, err))
	if err != nil || resp.Status != 200 {
		return nil
	}
	doc := document(resp.Body)
	h.note(name, doc)
	return doc
}

// The extension gates independently from the regular ATs, in the same run.
func (s *updatesFollowup) installerHandoff(stack *e2e.Stack, baseline e2e.ServingBaseline) {
	h := &handoffRun{s: s, stack: stack, checks: map[string]bool{}}
	h.rec = *s.rec["AT24"]
	h.rec.ID, h.rec.Title = "installer-handoff", "D-110/D-111/D-125/D-127 installer hand-off"
	h.rec.Status, h.rec.Scope = at.StatusBlocked, "Follow-up 10 systemd hand-off and central receipt acceptance; serving traffic proof cited from devstack 19"
	h.rec.Spec = []string{"UPD02", "UPD03", "UPD05", "§13.5", "§13.6", "§13.9", "D-080", "D-098", "D-110", "D-111", "D-125", "D-127"}
	h.rec.Expected = "Install, seed, stage and real correlated apply; central baseline/promotion/replay/correction; tamper, checkpoint and downgrade refusals"
	h.rec.Actual = "Installer prerequisites not reached"
	h.rec.Observations = nil
	defer func() {
		required := []string{"install", "seed", "fresh_no_work", "fresh_guard_clear", "boot_kernel", "legacy_boot_config", "checkpoint_failure", "failure_replay", "failure_baseline", "staged", "correlated", "apply", "restarts", "central", "job_id", "replay", "conflict", "wrong_job", "correction", "tamper", "downgrade"}
		var failed []string
		for _, k := range required {
			if !h.checks[k] {
				failed = append(failed, k)
			}
		}
		if len(failed) == 0 {
			h.rec.Status = at.StatusPass
			h.rec.Actual = "All installer hand-off and central receipt checks passed"
		} else {
			h.rec.Status = at.StatusFail
			h.rec.Actual += "; incomplete checks: " + strings.Join(failed, ", ")
		}
		if err := at.Write(filepath.Join(s.r.lab.root, "docs/e2e/at/updates"), h.rec); err != nil {
			s.r.t.Error(err)
		}
		if len(failed) > 0 {
			s.r.t.Errorf("installer hand-off: %s", h.rec.Actual)
		}
	}()
	must := func(err error) bool {
		if err != nil {
			h.rec.Actual = at.Redact(err.Error())
			return false
		}
		return true
	}
	h.note("consumed_devstack19", "docs/e2e/runs/2026-10-31/followup19/d6aaa798-6c39-49f1-8fa9-5f84f696ac91/installer-journaled-upgrade.md: serving withdrawal, five-field correlation and nonzero durable watermark; this run uses a quiet unpublished agent and separately tests central acceptance")
	s.r.t.Log("follow-up 10: systemd installer hand-off")
	if !must(stack.PrepareInstallerNode(s.ctx)) {
		return
	}
	admin := e2e.BrowserSession{Client: s.admin.http, CSRF: s.admin.csrf}
	releases, err := stack.RegisterInstallerReleases(s.ctx, admin)
	if !must(err) {
		return
	}
	for _, spec := range []struct{ version, seq string }{{"1.1.2", "12"}, {"1.1.3", "13"}} {
		dir, err := os.MkdirTemp(filepath.Join(s.r.lab.root, ".agent/tmp"), "handoff-release-")
		if !must(err) {
			return
		}
		defer os.RemoveAll(dir)
		b, err := stack.BuildInstallerRelease(s.ctx, dir, spec.version, spec.seq, "TEST-ONLY-updates-followup10", e2e.ReleaseOptions{})
		if !must(err) || !must(stack.RegisterRelease(s.ctx, admin, b)) {
			return
		}
		releases = append(releases, b)
	}
	ids := []uuid.UUID{baseline.Boot, baseline.Good, baseline.Bad}
	for i, b := range releases {
		ids = append(ids, b.ID)
		h.note(fmt.Sprintf("release_%d", 10+i), map[string]any{"release_id": b.ID, "manifest_sha256": b.ManifestSHA256, "architectures": []string{"amd64", "arm64"}})
		dir := fmt.Sprintf("/var/tmp/e2e-rel/handoff-%d", i)
		if h.run("", "mkdir", "-p", dir).Exit != 0 || !must(h.copy(filepath.Join(b.Directory, "amd64.tar.zst"), dir+"/artifact.tar.zst")) || !must(h.put(dir+"/manifest.json", b.Manifest)) {
			return
		}
	}
	_, _, err = stack.ActivatePolicy(s.ctx, admin, "release_support", "TEST ONLY installer handoff", map[string]any{"approved_release_ids": ids, "revoked_release_ids": []string{}, "minimum_serving_version": map[string]any{"serving_release_ids": ids}})
	if !must(err) || !must(stack.HoldInstallerFirstPoll(s.ctx, true)) {
		return
	}
	name := "TEST ONLY installer handoff"
	resp, err := s.admin.call(s.ctx, "POST", s.r.lab.public+"/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name}, map[string]string{"Idempotency-Key": uuid.NewString()})
	var enrollment api.EnrollmentCreated
	if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &enrollment) != nil {
		h.rec.Actual = "enrollment creation failed"
		return
	}
	node := enrollment.Enrollment.NodeId.String()
	defer h.retire(node)
	token, err := enrollment.Token.Get()
	if !must(err) || !must(stack.StageInstallerToken(s.ctx, "installer-node", token)) {
		return
	}
	installed, err := stack.RerunInstaller(s.ctx, "installer-node")
	h.note("install", installed)
	h.rec.NodeIncarnation = installed.Incarnation
	if !must(err) {
		return
	}
	socket := h.run("netbird_socket", "test", "-S", "/run/fvpn-netbird/netbird.sock")
	// The service snapshot covers consumer units; NetBird is read separately.
	netbird := h.run("netbird_service", "systemctl", "is-active", "fvpn-netbird.service")
	h.check("install", installed.Exit == 0 && socket.Exit == 0 && installed.Services["fvpn-agent.service"] == "active" && netbird.Stdout == "active")
	beforeCentral := h.central("central_before", node)
	h.note("null_central_baseline", document(beforeCentral["releases"])["current_release_id"])
	seedArgs := []string{installedUpdater, "seed", "--config", installerConfig, "--manifest", "/var/lib/vpn-node/update-staging/" + releases[0].ID.String() + ".manifest.json", "--tree", "/opt/vpn-node/releases/" + releases[0].ID.String()}
	seed := h.run("seed_default_umask", seedArgs...)
	seeded := h.status("status_seeded")
	seed077 := h.run("seed_umask077", append([]string{"sh", "-c", `umask 077; exec "$@"`, "sh"}, seedArgs...)...)
	seededAgain := h.status("status_seeded_again")
	h.check("seed", seed.Exit == 0 && seed077.Exit == 0 && bytes.Equal(seeded["seeded_baseline"], seededAgain["seeded_baseline"]) && document(seeded["seeded_baseline"]).text("release_id") == releases[0].ID.String())
	cfg := document([]byte(h.run("", "cat", installerConfig).Stdout))
	if cfg == nil {
		return
	}
	_, hasBoot := cfg["boot_id"]
	h.note("generated_config", sanitizeUpdaterConfig(cfg))
	missing := h.absent("/var/lib/vpn-node/state/desired-state.json") && h.absent(installerPending)
	h.run("fresh_oneshot", "systemctl", "start", "fvpn-updater.service")
	unit := h.run("fresh_unit", "systemctl", "show", "fvpn-updater.service", "-p", "ExecMainStatus", "-p", "Result")
	noWork := h.run("fresh_run", installedUpdater, "run", "--config", installerConfig)
	h.check("fresh_no_work", missing && noWork.Exit == 0 && strings.Contains(unit.Stdout, "ExecMainStatus=0") && strings.Contains(unit.Stdout, "Result=success"))
	h.check("fresh_guard_clear", missing && h.absent("/var/lib/vpn-node/update-journal/forwarding.blocked"))
	kernel := h.run("kernel_boot_id", "cat", "/proc/sys/kernel/random/boot_id").Stdout
	cfg.set("boot_id", uuid.NewString())
	raw, _ := json.Marshal(cfg)
	if !must(h.put("/var/tmp/e2e-rel/legacy-updater.json", raw)) {
		return
	}
	legacy := h.run("legacy_boot_config", installedUpdater, "run", "--config", "/var/tmp/e2e-rel/legacy-updater.json")
	holder := document([]byte(h.run("legacy_lock_holder", "cat", "/var/lib/vpn-node/update.lock").Stdout))
	h.check("legacy_boot_config", legacy.Exit == 0 && holder.text("boot_id") == kernel)
	delete(cfg, "boot_id")
	// The installer still leaves result_url empty. Retain that fact, then supply
	// the owned lab endpoint explicitly, as documented by devstack.
	cfg.set("result_url", s.r.lab.agent+"/api/v1/agent/update-results")
	cfg.set("mtls_ca", "/var/tmp/e2e-rel/ca.pem")
	cfg.set("allowed_artifact_origins", []string{s.r.lab.agent, s.r.lab.releases})
	raw, _ = json.Marshal(cfg)
	if !must(h.put(installerConfig, raw)) || !must(stack.HoldInstallerApply(s.ctx, true)) || !must(stack.ConfigureInstallerServing(s.ctx)) || !must(stack.HoldInstallerFirstPoll(s.ctx, false)) || !must(stack.RestoreInstallerServices(s.ctx, "installer-node", installed.Services)) {
		return
	}
	h.note("reporting_adapter", "Explicit owned result_url, CA and origins; generated installer reporting configuration is not claimed")
	ready := s.wait(60*time.Second, func() bool {
		return !h.absent("/var/lib/vpn-node/state/desired-state.json") && strings.Contains(h.run("", "cat", "/run/vpn-node/heartbeat").Stdout, "boot_id "+kernel)
	})
	h.note("agent_ready", ready)
	beat := h.run("agent_boot", "cat", "/run/vpn-node/heartbeat")
	h.check("boot_kernel", !hasBoot && kernel != "" && strings.Contains(beat.Stdout, "boot_id "+kernel) && document(seeded["seeded_baseline"]).text("boot_id") == kernel && holder.text("boot_id") == kernel)
	// Real absent-checkpoint negative before central has a release baseline.
	h.run("stop_for_checkpoint_negative", "systemctl", "stop", "fvpn-watchdog.service", "fvpn-agent.service")
	stageFailure := h.upgrade("stage_checkpoint_negative", 1)
	pendingBytes := h.run("failure_pending", "cat", installerPending).Stdout
	before := h.current()
	h.note("current_before_failure", before)
	h.run("checkpoint_before_negative", "cat", "/var/lib/vpn-node/state/meter.checkpoint")
	failed := h.run("apply_checkpoint_negative", installedUpdater, "apply", "--config", installerConfig, "--pending", installerPending)
	failedStatus := h.status("checkpoint_failure_status")
	failure := h.receipt("failure_receipt")
	h.note("current_after_failure", h.current())
	h.run("failure_latch", "cat", "/var/lib/vpn-node/update-journal/forwarding.blocked")
	h.check("checkpoint_failure", stageFailure.Exit == 0 && failed.Exit == 1 && strings.Contains(failed.Stderr, "CHECKPOINT_FAILED") && h.current() == before && failedStatus.text("state") == "failed" && failedStatus.text("stage") == "final_sample" && string(failure["usage_checkpoint"]) == "null" && string(failure["member_forwarding_enabled"]) == "false")
	// Restore the exact original pending bytes to its original path for sealed
	// replay. No changed path/request UUID can masquerade as this replay.
	if !must(h.put(installerPending, []byte(pendingBytes))) {
		return
	}
	replayFailure := h.run("sealed_failure_apply", installedUpdater, "apply", "--config", installerConfig, "--pending", installerPending)
	recoverFailure := h.run("sealed_failure_recover", installedUpdater, "recover", "--config", installerConfig)
	afterReplay := h.receipt("failure_receipt_after_recovery")
	h.check("failure_replay", replayFailure.Exit == 1 && strings.Contains(replayFailure.Stderr, "outcome=failed") && recoverFailure.Exit == 2 && bytes.Equal(marshalDoc(failure), marshalDoc(afterReplay)) && h.current() == before && !h.absent("/var/lib/vpn-node/update-journal/forwarding.blocked"))
	failureAck := h.submit("failure_submission", failure, uuid.NewString())
	failureView := h.central("central_seeded_failure", node)
	releasesView := document(failureView["releases"])
	h.check("failure_baseline", failureAck.status == 202 && string(document(beforeCentral["releases"])["current_release_id"]) == "null" && releasesView.text("current_release_id") == releases[0].ID.String() && releasesView.text("last_good_release_id") == releases[0].ID.String() && document(failureView["update"]).text("state") == "failed")
	h.run("flush_failure", installedUpdater, "flush", "--config", installerConfig)
	// Resume the real agent before a new, higher release requests a fresh latch.
	if !must(stack.RestoreInstallerServices(s.ctx, "installer-node", installed.Services)) {
		return
	}
	s.wait(20*time.Second, func() bool { return h.run("", "systemctl", "is-active", "fvpn-agent.service").Stdout == "active" })
	stage := h.upgrade("upgrade", 2)
	pending := document([]byte(h.run("pending_before_apply", "cat", installerPending).Stdout))
	h.note("current_after_stage", h.current())
	h.note("staged_pending", pending)
	h.check("staged", stage.Exit == 0 && stage.Status == "staged" && pending.text("release_id") == releases[2].ID.String() && h.current() == before)
	unitsBefore := h.run("units_before_apply", "systemctl", "show", "fvpn-agent.service", "fvpn-watchdog.service", "-p", "Id", "-p", "ExecMainStartTimestampMonotonic").Stdout
	observer := exec.CommandContext(s.ctx, "docker", "exec", s.r.lab.prefix+"-installer-node", "/usr/local/bin/fvpn-e2e-nodeconfig", "upgrade-observe")
	if !must(observer.Start()) {
		return
	}
	time.Sleep(100 * time.Millisecond)
	applied := h.run("apply", installedUpdater, "apply", "--config", installerConfig, "--pending", installerPending)
	h.run("", "touch", "/var/lib/vpn-node/e2e/upgrade-observe.done")
	observeErr := observer.Wait()
	trace := h.run("handshake_trace", "cat", "/var/lib/vpn-node/e2e/upgrade-observe.jsonl").Stdout
	checkpoint := h.correlated(trace, kernel)
	h.check("correlated", observeErr == nil && checkpoint != nil)
	after := h.status("status_after_apply")
	h.note("current_after_apply", h.current())
	unitsAfter := h.run("units_after_apply", "systemctl", "show", "fvpn-agent.service", "fvpn-watchdog.service", "-p", "Id", "-p", "ExecMainStartTimestampMonotonic").Stdout
	executable := h.run("running_executable", "sh", "-c", `p=$(systemctl show -p MainPID --value fvpn-agent.service); readlink /proc/$p/exe`).Stdout
	h.check("restarts", unitsBefore != unitsAfter && strings.Contains(executable, releases[2].ID.String()))
	success := h.receipt("success_receipt")
	var health map[string]string
	_ = json.Unmarshal(after["health"], &health)
	h.check("apply", applied.Exit == 0 && (updatesJournal{Health: health}).allPassed() && after.text("source") == "installer" && after.text("state") == "succeeded" && filepath.Base(h.current()) == releases[2].ID.String() && h.absent(installerPending) && checkpointMatches(checkpoint, success))
	claim := h.claimBoot()
	h.check("boot_kernel", h.checks["boot_kernel"] && claim == kernel && success.text("boot_id") == kernel)
	h.run("post_apply_check", "/usr/local/bin/fvpn-node-install", "check", "--json")
	h.resultAcceptance(node, releases[0], releases[2], success)
	// A separate higher release lets tamper fail before switch, independently of
	// the successful immutable job. The lower baseline must be refused at staging.
	h.upgrade("stage_tamper", 3)
	tamperPending := document([]byte(h.run("", "cat", installerPending).Stdout))
	path := tamperPending.text("artifact_path")
	if strings.HasPrefix(path, "/var/lib/vpn-node/update-staging/") && strings.HasSuffix(path, ".tar.zst") {
		before = h.current()
		h.run("tamper_stimulus", "sh", "-c", `printf X | dd of="$1" bs=1 seek=32 count=1 conv=notrunc status=none`, "sh", path)
		rejected := h.run("apply_tampered", installedUpdater, "apply", "--config", installerConfig, "--pending", installerPending)
		h.check("tamper", rejected.Exit == 1 && strings.Contains(rejected.Stderr, "HASH_MISMATCH") && h.current() == before)
	}
	h.run("preserve_pending", "sh", "-c", `if [ -e "$1" ]; then mv "$1" "${1%.json}.tamper.json"; fi`, "sh", installerPending)
	down := h.upgrade("downgrade", 0)
	h.check("downgrade", down.Exit == 1 && strings.Contains(down.Code, "DOWNGRADE") && h.absent(installerPending))
	h.rec.Actual = "Measured install, default-umask seed, final-sample negative and replay, higher candidate apply, central receipts, tamper and downgrade; see check_* deciding observations"
}

func (h *handoffRun) upgrade(name string, index int) e2e.InstallerOutcome {
	base := fmt.Sprintf("/var/tmp/e2e-rel/handoff-%d", index)
	out, err := h.stack.ProbeInstallerExit(h.s.ctx, "installer-node", []string{"upgrade", "--json", "--manifest", base + "/manifest.json", "--artifact", base + "/artifact.tar.zst", "--trust", "/var/tmp/e2e-rel/node-trust.json"})
	h.note(name, out)
	if err != nil {
		h.note(name+"_error", at.Redact(err.Error()))
	}
	return out
}
func installerCentralSuccess(doc updateDocument, releaseID string) bool {
	u := document(doc["update"])
	return document(doc["releases"]).text("current_release_id") == releaseID && u.text("source") == "installer" && string(u["batch_id"]) == "null" && u.text("state") == "succeeded"
}

func testInstallerEvidence(t *testing.T) {
	t.Helper()
	good := document([]byte(`{"releases":{"current_release_id":"new"},"update":{"source":"installer","batch_id":null,"state":"succeeded"}}`))
	if !installerCentralSuccess(good, "new") {
		t.Fatal("accepted installer summary rejected")
	}
	for _, raw := range []string{
		`{"releases":{"current_release_id":"old"},"update":{"source":"installer","batch_id":null,"state":"succeeded"}}`,
		`{"releases":{"current_release_id":"new"},"update":{"source":"central","batch_id":null,"state":"succeeded"}}`,
		`{"releases":{"current_release_id":"new"},"update":{"source":"installer","state":"succeeded"}}`,
		`{"releases":{"current_release_id":"new"},"update":{"source":"installer","batch_id":null,"state":"failed"}}`,
		`{"releases":{"current_release_id":"new"},"update":null}`,
	} {
		if installerCentralSuccess(document([]byte(raw)), "new") {
			t.Fatal("incomplete installer evidence passed")
		}
	}
}

// Keep this early installer node outside the regular all-nodes batch snapshots.
// Retirement is through the API with ETag and read-back, never a database edit.
func (h *handoffRun) retire(node string) {
	view := h.central("before_fixture_retirement", node)
	resp, err := h.s.admin.call(h.s.ctx, "DELETE", h.s.r.lab.public+"/api/v1/admin/nodes/"+node, nil, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": `"` + view.text("resource_version") + `"`})
	h.note("fixture_retirement_http", httpDetail(resp, err))
	retired := h.central("fixture_retired", node)
	if err != nil || resp.Status != 202 || retired.text("lifecycle_state") != "retired" {
		h.s.r.t.Fatal("installer fixture retirement not confirmed; refusing polluted fleet snapshots")
	}
}
