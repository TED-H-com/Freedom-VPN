//go:build e2e

package updates_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
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

// The separate record prevents a regular AT24/AT26 pass from claiming that a
// blocked installer path succeeded. Both records belong to this same live run.
func (s *updatesFollowup) installerHandoff(stack *e2e.Stack, baseline e2e.ServingBaseline) {
	h := &handoffRun{s: s, checks: map[string]bool{}}
	h.rec = *s.rec["AT24"]
	h.rec.ID = "installer-handoff"
	h.rec.Title = "D-080/D-089 systemd installer to updater hand-off"
	h.rec.Status = at.StatusBlocked
	h.rec.Scope = "Follow-up 9 installer extension to AT24/AT26; fresh node and kernel boot identity"
	h.rec.Spec = []string{"UPD02", "UPD03", "UPD05", "§13.5", "§13.6", "§13.9", "D-080", "D-089", "D-097"}
	h.rec.Expected = "Verified seed, no-work exit 0, staged upgrade with unchanged current, installer-sourced healthy apply and central promotion; tamper and downgrade refused"
	h.rec.Actual = "Installer prerequisites not reached"
	h.rec.Observations = nil
	defer func() {
		required := []string{"seed", "fresh_no_work", "fresh_check_no_updater_drift", "fresh_guard_clear", "boot_kernel", "legacy_boot_config", "staged", "tamper", "downgrade", "apply", "central"}
		all := true
		for _, name := range required {
			if !h.checks[name] {
				all = false
			}
		}
		if all {
			h.rec.Status = at.StatusPass
			h.rec.Actual = "All installer hand-off checks passed"
		}
		if err := at.Write(filepath.Join(s.r.lab.root, "docs/e2e/at/updates"), h.rec); err != nil {
			s.r.t.Error(err)
		}
		if !all {
			s.r.t.Errorf("installer hand-off incomplete: %s", h.rec.Actual)
		}
	}()
	s.r.t.Log("follow-up 9: preparing dedicated real-systemd installer node")
	if err := stack.PrepareInstallerNode(s.ctx); err != nil {
		h.rec.Actual = "devstack installer-node prerequisite: " + at.Redact(err.Error())
		return
	}
	admin := e2e.BrowserSession{Client: s.admin.http, CSRF: s.admin.csrf}
	releases := []e2e.BuiltRelease{}
	for i, spec := range []struct{ version, seq string }{{"1.1.0", "10"}, {"1.1.1", "11"}, {"1.0.9", "9"}} {
		b, err := h.buildRelease(stack, spec.version, spec.seq)
		if err != nil {
			h.rec.Actual = "installer fixture build: " + at.Redact(err.Error())
			return
		}
		if err = stack.RegisterRelease(s.ctx, admin, b); err != nil {
			h.rec.Actual = "installer release registration: " + at.Redact(err.Error())
			return
		}
		defer os.RemoveAll(b.Directory)
		path := fmt.Sprintf("/var/tmp/e2e-rel/handoff-%d", i)
		if h.run("", "mkdir", "-p", path).Exit != 0 {
			return
		}
		if err = h.copy(filepath.Join(b.Directory, "amd64.tar.zst"), path+"/artifact.tar.zst"); err != nil {
			return
		}
		if err = h.put(path+"/manifest.json", b.Manifest); err != nil {
			return
		}
		releases = append(releases, b)
		h.note("release_"+spec.seq, map[string]any{"release_id": b.ID, "manifest_sha256": b.ManifestSHA256, "architectures": []string{"amd64", "arm64"}})
	}
	ids := []uuid.UUID{baseline.Boot, baseline.Good, baseline.Bad}
	for _, b := range releases {
		ids = append(ids, b.ID)
	}
	rev, _, err := stack.ActivatePolicy(s.ctx, admin, "release_support", "e2e-installer-handoff", map[string]any{"approved_release_ids": ids, "revoked_release_ids": []string{}, "minimum_serving_version": map[string]any{"serving_release_ids": ids}})
	if err != nil {
		h.rec.Actual = "installer support policy activation: " + at.Redact(err.Error())
		return
	}
	h.note("support_policy_revision", rev)
	// The exported install spec remains authoritative for lab enrollment/network
	// inputs. Replace only release inputs; generated signing keys stay on the host.
	specOut := h.run("", "cat", "/var/tmp/e2e-installer.json")
	spec := document([]byte(specOut.Stdout))
	if spec == nil {
		h.rec.Actual = "devstack installer spec unavailable"
		return
	}
	spec.set("manifest", "/var/tmp/e2e-rel/handoff-0/manifest.json")
	spec.set("artifact", "/var/tmp/e2e-rel/handoff-0/artifact.tar.zst")
	spec.set("trust", "/var/tmp/e2e-rel/node-trust.json")
	specRaw, _ := json.Marshal(spec)
	if h.put("/var/tmp/e2e-installer.json", specRaw) != nil {
		return
	}
	name := "TEST ONLY installer handoff"
	resp, err := s.admin.call(s.ctx, "POST", s.r.lab.public+"/api/v1/admin/enrollments", api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, Name: &name}, map[string]string{"Idempotency-Key": uuid.NewString()})
	var enrollment api.EnrollmentCreated
	if err != nil || resp.Status != 201 || json.Unmarshal(resp.Body, &enrollment) != nil {
		h.rec.Actual = "installer enrollment creation failed"
		return
	}
	defer h.retire(enrollment.Enrollment.NodeId.String())
	token, err := enrollment.Token.Get()
	if err != nil || stack.StageInstallerToken(s.ctx, "installer-node", token) != nil {
		return
	}
	// Retain the exact signed archive for offline seed. The product installer
	// currently retains its envelope only; no baseline/journal data is fabricated.
	var env struct {
		Signed api.ReleaseManifest `json:"signed"`
	}
	if json.Unmarshal(releases[0].Manifest, &env) != nil {
		return
	}
	for _, a := range env.Signed.Artifacts {
		if a.Architecture == "amd64" {
			if h.run("", "mkdir", "-p", "/var/lib/vpn-node/update-staging").Exit != 0 {
				return
			}
			if h.copy(filepath.Join(releases[0].Directory, "amd64.tar.zst"), "/var/lib/vpn-node/update-staging/"+a.Sha256+".tar.zst") != nil {
				return
			}
		}
	}
	h.note("lab_adapters", []string{"Real binaries for both architectures repacked at installer root with unchanged packaged systemd units, re-signed with generated lab release key and registered through public API", "Initial signed archive retained for offline seed; no journal writes", "Runtime systemd condition delays automatic apply for stage inspection; product ExecStart unchanged"})
	// Hold the first agent start until missing-desired-state behavior is measured.
	if h.put("/run/systemd/system/fvpn-agent.service.d/fresh.conf", []byte("[Unit]\nConditionPathExists=/run/vpn-node/fresh-allow\n")) != nil {
		return
	}
	h.note("fresh_agent_barrier", "TEST ONLY runtime condition holds the agent before its first poll; no desired-state file/job is manufactured or deleted")
	installed, installErr := stack.RerunInstaller(s.ctx, "installer-node")
	h.rec.NodeIncarnation = installed.Incarnation
	h.note("install", installed)
	h.note("install_wrapper_error", installErr != nil)
	h.run("first_start_units", "systemctl", "show", "fvpn-netbird.service", "fvpn-agent.service", "fvpn-node-netbird-up.service", "-p", "Id", "-p", "ActiveState", "-p", "SubState", "-p", "Result", "-p", "ExecMainStatus")
	h.run("first_start_errors", "journalctl", "--no-pager", "-o", "cat", "-n", "35", "-u", "fvpn-netbird.service", "-u", "fvpn-agent.service", "-u", "fvpn-node-netbird-up.service")
	if h.absent(installerConfig) {
		h.rec.Actual = "installer did not produce updater configuration"
		return
	}
	node := enrollment.Enrollment.NodeId.String()
	h.central("central_before", node)
	h.run("installed_baseline", "cat", "/var/lib/vpn-node/install/baseline.json")
	h.run("installed_tree_modes", "find", "/opt/vpn-node/releases/"+releases[0].ID.String(), "-printf", "%P %m %y\n")
	h.run("seed_default_umask", installedUpdater, "seed", "--config", installerConfig, "--manifest", "/var/lib/vpn-node/update-staging/"+releases[0].ID.String()+".manifest.json", "--tree", "/opt/vpn-node/releases/"+releases[0].ID.String())
	seed := h.run("seed_packaged_umask", "sh", "-c", `umask 077; exec "$@"`, "sh", installedUpdater, "seed", "--config", installerConfig, "--manifest", "/var/lib/vpn-node/update-staging/"+releases[0].ID.String()+".manifest.json", "--tree", "/opt/vpn-node/releases/"+releases[0].ID.String())
	seeded := h.status("status_seeded")
	h.check("seed", seed.Exit == 0 && document(seeded["seeded_baseline"]).text("release_id") == releases[0].ID.String())
	cfgOut := h.run("", "cat", installerConfig)
	cfg := document([]byte(cfgOut.Stdout))
	if cfg == nil {
		return
	}
	_, hasBoot := cfg["boot_id"]
	h.note("generated_config", map[string]any{"has_boot_id": hasBoot, "result_url": cfg.text("result_url"), "meter_checkpoint": cfg.text("meter_checkpoint"), "stop_command": json.RawMessage(cfg["stop_command"]), "start_command": json.RawMessage(cfg["start_command"])})
	missing := h.absent("/var/lib/vpn-node/state/desired-state.json") && h.absent(installerPending)
	h.note("fresh_no_desired_or_pending", missing)
	h.run("fresh_oneshot", "systemctl", "start", "fvpn-updater.service")
	unit := h.run("fresh_unit_result", "systemctl", "show", "fvpn-updater.service", "-p", "ExecMainStatus", "-p", "Result", "-p", "ActiveState")
	noWork := h.run("fresh_second_run", installedUpdater, "run", "--config", installerConfig)
	h.check("fresh_no_work", missing && noWork.Exit == 0 && strings.Contains(unit.Stdout, "ExecMainStatus=0") && strings.Contains(unit.Stdout, "Result=success"))
	check := h.run("fresh_installer_check", "fvpn-node-install", "check", "--json")
	h.check("fresh_check_no_updater_drift", document([]byte(check.Stdout)) != nil && !strings.Contains(string(document([]byte(check.Stdout))["drifts"]), "unit fvpn-updater.service "))
	guard := h.absent("/var/lib/vpn-node/update-journal/forwarding.blocked")
	h.check("fresh_guard_clear", missing && guard)
	h.note("fresh_forwarding_scope", "Updater latch absence only. Other serving prerequisites may remain blocked; central forwarding reasons and failed packaged units are recorded, never interpreted as serving success.")
	kernel := h.run("kernel_boot_id", "cat", "/proc/sys/kernel/random/boot_id").Stdout
	holder := h.run("lock_holder", "cat", "/var/lib/vpn-node/update.lock")
	h.check("boot_kernel", !hasBoot && kernel != "" && document([]byte(holder.Stdout)).text("boot_id") == kernel && document(seeded["seeded_baseline"]).text("boot_id") == kernel)
	cfg.set("boot_id", uuid.NewString())
	legacyRaw, _ := json.Marshal(cfg)
	if h.put("/var/tmp/e2e-rel/legacy-updater.json", legacyRaw) != nil {
		return
	}
	legacy := h.run("legacy_boot_config", installedUpdater, "run", "--config", "/var/tmp/e2e-rel/legacy-updater.json")
	legacyHolder := h.run("legacy_lock_holder", "cat", "/var/lib/vpn-node/update.lock")
	h.check("legacy_boot_config", legacy.Exit == 0 && document([]byte(legacyHolder.Stdout)).text("boot_id") == kernel)
	h.run("release_fresh_barrier", "rm", "/run/systemd/system/fvpn-agent.service.d/fresh.conf")
	h.run("reload_after_fresh", "systemctl", "daemon-reload")
	h.run("start_agent_after_fresh", "systemctl", "start", "fvpn-agent.service")
	s.wait(20*time.Second, func() bool {
		raw := s.readJSON(`SELECT to_jsonb(count(*)) FROM node_observations WHERE node_id=$1`, enrollment.Enrollment.NodeId)
		return string(raw) != "0"
	})
	h.note("fresh_serving_gate_observation", s.readJSON(`SELECT coalesce((SELECT jsonb_build_object('member_forwarding_enabled', report_payload->'member_forwarding_enabled', 'forwarding_block_reasons', report_payload->'forwarding_block_reasons') FROM node_observations WHERE node_id=$1), 'null'::jsonb)`, enrollment.Enrollment.NodeId))
	h.run("stop_agent_after_fresh", "systemctl", "stop", "fvpn-agent.service")
	// Expose production defaults before adding only the owned lab endpoint.
	delete(cfg, "boot_id")
	cfg.set("result_url", s.r.lab.agent+"/api/v1/agent/update-results")
	cfg.set("mtls_ca", "/var/tmp/e2e-rel/ca.pem")
	cfg.set("allowed_artifact_origins", []string{s.r.lab.agent, s.r.lab.releases})
	configured, _ := json.Marshal(cfg)
	if h.put(installerConfig, configured) != nil {
		return
	}
	h.note("lab_result_endpoint", "Added result_url, agent CA and exact lab origins after recording the generated config; this is not evidence that installer configured result reporting")
	if h.put("/run/systemd/system/fvpn-updater.service.d/handoff.conf", []byte("[Unit]\nConditionPathExists=/run/vpn-node/handoff-allow\n")) != nil {
		return
	}
	if h.run("stage_barrier", "systemctl", "daemon-reload").Exit != 0 {
		return
	}
	before := h.current()
	stage := h.upgrade("upgrade", 1)
	pending := h.run("pending_before_apply", "cat", installerPending)
	h.note("current_before", before)
	h.note("current_after_stage", h.current())
	h.check("staged", stage.Exit == 0 && document([]byte(stage.Stdout)).text("status") == "staged" && pending.Exit == 0 && h.current() == before)
	// Apply authentic bytes first; failure cannot be hidden by tamper stimulus.
	h.run("units_before_apply", "systemctl", "show", "fvpn-agent.service", "fvpn-watchdog.service", "-p", "Id", "-p", "ExecMainStartTimestampMonotonic", "-p", "ActiveState", "-p", "Result")
	applied := h.run("apply", installedUpdater, "apply", "--config", installerConfig, "--pending", installerPending)
	after := h.status("status_after_apply")
	h.note("current_after_apply", h.current())
	h.note("pending_consumed_after_apply", h.absent(installerPending))
	h.run("units_after_apply", "systemctl", "show", "fvpn-agent.service", "fvpn-watchdog.service", "-p", "Id", "-p", "ExecMainStartTimestampMonotonic", "-p", "ActiveState", "-p", "Result")
	var health map[string]string
	_ = json.Unmarshal(after["health"], &health)
	h.check("apply", (updatesJournal{Health: health}).allPassed() && filepath.Base(h.current()) == releases[1].ID.String() && (applied.Exit == 0 || applied.Exit == 2) && after.text("source") == "installer" && after.text("state") == "succeeded" && after.text("current_release_id") == releases[1].ID.String() && h.absent(installerPending))
	h.run("second_run_after_apply", installedUpdater, "run", "--config", installerConfig)
	h.status("status_after_second_run")
	h.run("flush", installedUpdater, "flush", "--config", installerConfig)
	h.journalEvidence()
	central := h.central("central_after", node)
	h.check("central", installerCentralSuccess(central, releases[1].ID.String()))
	h.note("central_installer_receipts", s.readJSON(`SELECT coalesce(jsonb_agg(jsonb_build_object('payload',payload,'ack',ack)), '[]'::jsonb) FROM installer_update_results WHERE node_id=$1`, enrollment.Enrollment.NodeId))
	// Keep rejection evidence distinct from the authentic apply above.
	h.upgrade("stage_tamper", 1)
	staged := h.run("", "cat", installerPending)
	p := document([]byte(staged.Stdout))
	path := p.text("artifact_path")
	if strings.HasPrefix(path, "/var/lib/vpn-node/update-staging/") && strings.HasSuffix(path, ".tar.zst") {
		before = h.current()
		h.run("tamper_stimulus", "sh", "-c", `printf X | dd of="$1" bs=1 seek=32 count=1 conv=notrunc status=none`, "sh", path)
		rejected := h.run("apply_tampered", installedUpdater, "apply", "--config", installerConfig, "--pending", installerPending)
		rejectStatus := h.status("status_after_tamper")
		h.check("tamper", rejected.Exit != 0 && strings.Contains(rejected.Stderr, "HASH_MISMATCH") && before == h.current() && rejectStatus.text("source") == "installer")
	}
	// Upgrade itself must reject this lower sequence before a pending record.
	h.run("preserve_pending_if_any", "sh", "-c", `if [ -e "$1" ]; then mv "$1" "${1%.json}.before-downgrade.json"; fi`, "sh", installerPending)
	down := h.upgrade("downgrade", 2)
	h.note("downgrade_pending_absent", h.absent(installerPending))
	h.check("downgrade", down.Exit == 1 && strings.Contains(down.Stderr, "DOWNGRADE") && h.absent(installerPending))
	h.run("downgrade_pending_record", "sh", "-c", `if [ -e "$1" ]; then cat "$1"; mv "$1" "${1%.json}.downgrade-not-applied.json"; fi`, "sh", installerPending)
	h.rec.Actual = "See check_* observations: authentic apply, fresh-node guard, tamper and downgrade are independently measured; failed service/health prerequisites remain blocked"
}

func (h *handoffRun) upgrade(name string, index int) installerCommand {
	base := fmt.Sprintf("/var/tmp/e2e-rel/handoff-%d", index)
	return h.run(name, "fvpn-node-install", "upgrade", "--json", "--manifest", base+"/manifest.json", "--artifact", base+"/artifact.tar.zst", "--trust", "/var/tmp/e2e-rel/node-trust.json")
}
func installerCentralSuccess(doc updateDocument, releaseID string) bool {
	update := document(doc["update"])
	return document(doc["releases"]).text("current_release_id") == releaseID && update.text("source") == "installer" && string(update["batch_id"]) == "null" && update.text("state") == "succeeded"
}

// Minimum harness adapter until devstack exports a centrally registered
// installer-layout fixture. Shared BuildRelease supplies real cross-builds and
// the manifest contract; only layout and artifact digests change before signing.
func (h *handoffRun) buildRelease(stack *e2e.Stack, version, sequence string) (e2e.BuiltRelease, error) {
	dir, err := os.MkdirTemp(filepath.Join(h.s.r.lab.root, ".agent", "tmp"), "handoff-release-")
	if err != nil {
		return e2e.BuiltRelease{}, err
	}
	b, err := stack.BuildRelease(h.s.ctx, dir, version, sequence, "TEST-ONLY-installer-handoff", e2e.ReleaseOptions{})
	if err != nil {
		return b, err
	}
	var env struct {
		Signed api.ReleaseManifest `json:"signed"`
	}
	if err = json.Unmarshal(b.Manifest, &env); err != nil {
		return b, err
	}
	for i, a := range env.Signed.Artifacts {
		arch := string(a.Architecture)
		tree := filepath.Join(dir, "installer-"+arch)
		if err = os.MkdirAll(tree, 0755); err != nil {
			return b, err
		}
		if err = exec.CommandContext(h.s.ctx, "tar", "--zstd", "-xf", filepath.Join(dir, arch+".tar.zst"), "-C", tree).Run(); err != nil {
			return b, err
		}
		for _, name := range []string{"fvpn-agent", "fvpn-updater", "fvpn-watchdog"} {
			if err = os.Rename(filepath.Join(tree, "bin", name), filepath.Join(tree, name)); err != nil {
				return b, err
			}
		}
		if err = os.Remove(filepath.Join(tree, "bin")); err != nil {
			return b, err
		}
		unitDir := filepath.Join(tree, "lib", "systemd")
		if err = os.MkdirAll(unitDir, 0755); err != nil {
			return b, err
		}
		entries, err := os.ReadDir(filepath.Join(h.s.r.lab.root, "deploy", "node", "systemd"))
		if err != nil {
			return b, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(h.s.r.lab.root, "deploy", "node", "systemd", entry.Name()))
			if err != nil {
				return b, err
			}
			if err = os.WriteFile(filepath.Join(unitDir, entry.Name()), raw, 0644); err != nil {
				return b, err
			}
		}
		cmd := exec.CommandContext(h.s.ctx, "go", "build", "-trimpath", "-o", filepath.Join(tree, "fvpn-node-install"), "./packaging/installer/cmd/fvpn-node-install")
		cmd.Dir = h.s.r.lab.root
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0", "GOTOOLCHAIN=local")
		if err = cmd.Run(); err != nil {
			return b, err
		}
		var size uint64
		err = filepath.Walk(tree, func(_ string, info os.FileInfo, err error) error {
			if err == nil && info.Mode().IsRegular() {
				size += uint64(info.Size())
			}
			return err
		})
		if err != nil {
			return b, err
		}
		artifact := filepath.Join(dir, arch+".tar.zst")
		if err = release.PackageDir(tree, artifact, time.Unix(1700000000, 0).UTC()); err != nil {
			return b, err
		}
		raw, err := os.ReadFile(artifact)
		if err != nil {
			return b, err
		}
		a.Sha256 = fmt.Sprintf("%x", sha256.Sum256(raw))
		a.SizeBytes = decstr.U64(len(raw))
		a.InstalledSizeBytes = decstr.U64(size)
		env.Signed.Artifacts[i] = a
	}
	key, err := release.LoadKey(filepath.Join(h.s.r.lab.state, "release", "key.json"))
	if err != nil {
		return b, err
	}
	signed, err := json.Marshal(env.Signed)
	if err != nil {
		return b, err
	}
	_, b.Manifest, err = release.Sign(signed, key)
	if err != nil {
		return b, err
	}
	b.ManifestSHA256, err = release.EnvelopeSHA256(b.Manifest)
	return b, err
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

// Copy only a quiescent updater journal, then issue read-only SQL. No identity
// database or key is copied, and no product journal row is edited.
func (h *handoffRun) journalEvidence() {
	path := filepath.Join(h.s.r.t.TempDir(), "installer-journal.db")
	source := h.s.r.lab.prefix + "-installer-node:/var/lib/vpn-node/update-journal/journal.db"
	if exec.CommandContext(h.s.ctx, "docker", "cp", source, path).Run() != nil {
		h.note("journal_copy", "failed")
		return
	}
	_ = exec.CommandContext(h.s.ctx, "docker", "cp", source+"-wal", path+"-wal").Run()
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro&_pragma=query_only(1)")
	if err != nil {
		h.note("journal_read", "failed")
		return
	}
	defer db.Close()
	var raw string
	err = db.QueryRow(`SELECT coalesce(json_group_array(json_object('job_id',job_id,'release_id',release_id,'phase',phase,'outcome',outcome,'error_code',error_code,'claim',json(intent_json))),'[]') FROM jobs`).Scan(&raw)
	if err == nil {
		h.note("journal_jobs", json.RawMessage(raw))
	}
	err = db.QueryRow(`SELECT coalesce(json_group_array(json_object('stage',stage,'error_code',error_code,'error_stage',error_stage,'finished_at',finished_at)),'[]') FROM txns`).Scan(&raw)
	if err == nil {
		h.note("journal_transactions", json.RawMessage(raw))
	}
	var payload []byte
	if db.QueryRow(`SELECT payload FROM result_outbox ORDER BY seq DESC LIMIT 1`).Scan(&payload) == nil && json.Valid(payload) {
		h.note("immutable_result_payload", json.RawMessage(payload))
		if h.put("/var/tmp/e2e-rel/result.json", payload) != nil {
			return
		}
		// Keyed exact replay supplies the deciding HTTP validation error if the
		// product sender records a rejection; it never manufactures a result.
		key := uuid.NewString()
		args := []string{"curl", "--silent", "--show-error", "--max-time", "20", "--cacert", "/var/tmp/e2e-rel/ca.pem", "--cert", "/var/lib/vpn-node/identity/cert.pem", "--key", "/var/lib/vpn-node/identity/key.pem", "-H", "Content-Type: application/json", "-H", "Idempotency-Key: " + key, "--data-binary", "@/var/tmp/e2e-rel/result.json", "-w", "\nHTTP %{http_code}\n", h.s.r.lab.agent + "/api/v1/agent/update-results"}
		h.run("result_keyed_submission", args...)
		h.run("result_keyed_replay", args...)
	}
}
