//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// This campaign keeps installer wrapper/serving/export evidence separate from
// the required journaled upgrade. A known product failure remains a failed row.
func TestE2E_Followup17Installer(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP17") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP17=1 with a fresh owned stack")
	}
	l, err := openLab()
	if err != nil {
		t.Fatal(err)
	}
	defer l.transport.CloseIdleConnections()
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Minute)
	defer cancel()
	head, _ := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	id := uuid.NewString()
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup17", id)}
	defer r.finish()
	s := &Stack{lab: l}
	admin := r.login(ctx, "e2e-admin", "admin")
	if admin == nil {
		return
	}
	b := BrowserSession{Client: admin.http, CSRF: admin.csrf}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(s.PrepareInstallerNode(ctx))
	releases, err := s.RegisterInstallerReleases(ctx, b)
	must(err)
	ids := []uuid.UUID{releases[0].ID, releases[1].ID}
	_, _, err = s.ActivatePolicy(ctx, b, "release_support", "installer-serving", map[string]any{"approved_release_ids": ids, "revoked_release_ids": []string{}, "minimum_serving_version": map[string]any{"serving_release_ids": ids}})
	must(err)
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "installer-dns", map[string]any{"profile_code": "installer-dns", "display_name": map[string]string{"zh-TW": "TEST ONLY installer resolver"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []string{}, "threat_feed_revision_ids": []string{}})
	must(err)
	r.catalogue(ctx, admin)

	rec := Record{ID: "installer-wrappers", Title: "ENR-06 fixed runtime layout, sampled hashes and restoration", Spec: []string{"§6.3", "ENR-06", "AT07"}, Status: StatusFail}
	note := recPathNote(&rec)
	for i, rel := range releases {
		noteJSON(&rec, fmt.Sprintf("registered_release_%d", i), map[string]any{"release_id": rel.ID, "manifest_sha256": rel.ManifestSHA256, "architectures": []string{"amd64", "arm64"}})
	}
	must(s.HoldInstallerFirstPoll(ctx, true))
	token, _, err := r.issueInstallerToken(ctx, s, b, "TEST ONLY installer serving", uuid.Nil)
	must(err)
	must(s.StageInstallerToken(ctx, "installer-node", token))
	token = ""
	installed, err := s.RerunInstaller(ctx, "installer-node")
	noteJSON(&rec, "initial", installed)
	if err != nil || installed.Exit != 0 {
		note("error", fmt.Sprint(err))
		r.record(rec)
		return
	}
	r.nodeID, err = uuid.Parse(installed.NodeID)
	must(err)
	r.incID, err = uuid.Parse(installed.Incarnation)
	must(err)
	// All traffic/counter helpers now select this actual systemd exit.
	selected, err := s.MemberTransportStack(l.member, "installer-node")
	must(err)
	r.lab = selected.lab
	c := s.installerContainer()
	run := func(args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "docker", append([]string{"exec", c}, args...)...).CombinedOutput()
	}
	_, noDesired := run("test", "!", "-e", "/var/lib/vpn-node/state/desired-state.json")
	noWork, noWorkErr := run("/opt/vpn-node/current/fvpn-updater", "run", "--config", "/etc/vpn-node/updater.json")
	note("first_poll_barrier_no_desired", boolText(noDesired == nil))
	note("fresh_updater_no_work", boolText(noWorkErr == nil && strings.Contains(string(noWork), "desired-state file is absent")))
	must(s.RestoreInstallerServices(ctx, "installer-node", installed.Services))
	again, err := s.RerunInstaller(ctx, "installer-node")
	noteJSON(&rec, "rerun", again)
	stable := err == nil && again.Exit == 0 && again.Status == "already_current" && again.Before == again.Hashes
	note("rerun_hashes_stable", boolText(stable))
	_, socketErr := run("test", "-S", "/run/fvpn-netbird/netbird.sock")
	_, runtimeErr := run("test", "-d", "/run/vpn-node")
	note("netbird_owned_socket", boolText(socketErr == nil))
	note("shared_runtime_survives_consumer_stop", boolText(runtimeErr == nil))
	must(s.RestoreInstallerServices(ctx, "installer-node", again.Services))
	states, err := s.InstallerServices(ctx, "installer-node")
	must(err)
	noteJSON(&rec, "restored_services", states)
	if stable && socketErr == nil && runtimeErr == nil && noDesired == nil && noWorkErr == nil {
		rec.Status = StatusPass
	}
	r.record(rec)

	must(s.ConfigureInstallerServing(ctx))
	must(s.HoldInstallerFirstPoll(ctx, false))
	_, err = run("systemctl", "restart", "fvpn-watchdog.service", "fvpn-agent.service")
	must(err)
	r.publish(ctx, admin, r.nodeID)
	r.fixtureServingMember(ctx, admin)
	r.installerUsage17(ctx, s, selected)
	r.installerCapture17(ctx, s)
	r.installerReenroll17(ctx, s, b)
	r.installerUpgrade17(ctx, s)
	// Exercise the changed ordinary entrypoint with an enrolled agent, then a
	// real process restart. Do this after installer measurements because the
	// baseline helper activates its own serving support policy.
	r.lab = l
	baseline, err := PrepareServingBaseline(ctx, admin.http, admin.csrf, false)
	must(err)
	rec = Record{ID: "regular-kernel-boot", Title: "Enrolled ordinary agent and updater share kernel boot identity", Spec: []string{"AT23", "F9-06"}, Status: StatusFail}
	noteJSON(&rec, "baseline_release_id", baseline.Boot)
	if r.enroll(ctx, admin) == uuid.Nil {
		r.record(rec)
		return
	}
	boot, bootErr := l.docker(l.node, "cat", "/proc/sys/kernel/random/boot_id")
	beat, beatErr := l.docker(l.node, "cat", "/run/vpn-node/heartbeat")
	holder, holderErr := l.docker(l.node, "cat", "/var/lib/vpn-node/update.lock")
	var claim struct {
		BootID string `json:"boot_id"`
	}
	aligned := bootErr == nil && beatErr == nil && holderErr == nil && json.Unmarshal(holder, &claim) == nil && claim.BootID == strings.TrimSpace(string(boot)) && strings.Contains(string(beat), "boot_id "+claim.BootID)
	recPathNote(&rec)("agent_heartbeat_and_updater_lock_match_kernel", boolText(aligned))
	if aligned {
		rec.Status = StatusPass
	}
	r.record(rec)
	r.restartNode(ctx)
}

func (r *runner) installerUsage17(ctx context.Context, s, selected *Stack) {
	rec := Record{ID: "installer-usage", Title: "ENR-05 same installer slot: counted traffic, held ACKs and CLI ledger", Spec: []string{"§10.5", "ENR-05"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	if !r.flowCounted {
		note("error", "no counted installer member path")
		return
	}
	if err := s.HoldUsageAcknowledgements(ctx, "installer-node", true); err != nil {
		note("hold", err.Error())
		return
	}
	defer func() { _ = s.HoldUsageAcknowledgements(context.WithoutCancel(ctx), "installer-node", false) }()
	initial, err := s.UsageLedgerSnapshot(ctx, "installer-node")
	if err != nil {
		note("snapshot", err.Error())
		return
	}
	before, err := selected.CountedMemberTraffic(ctx, r.usageDevice)
	if err != nil {
		note("counter", err.Error())
		return
	}
	if _, err = selected.StartEstablishedTransports(ctx, r.usageDevice); err != nil {
		note("transport", err.Error())
		return
	}
	defer func() { _ = selected.StopEstablishedTransports(context.WithoutCancel(ctx)) }()
	wait, cancel := context.WithTimeout(ctx, 150*time.Second)
	var held UsageSnapshot
	produced := waitUntil(wait, 2*time.Second, func() bool {
		held, err = s.UsageLedgerSnapshot(wait, "installer-node")
		return err == nil && held.UnackedCount > initial.UnackedCount
	})
	cancel()
	after, counterErr := selected.CountedMemberTraffic(ctx, r.usageDevice)
	noteJSON(&rec, "counters_before", before)
	noteJSON(&rec, "counters_after", after)
	noteJSON(&rec, "initial", initial)
	noteJSON(&rec, "held", held)
	_ = selected.StopEstablishedTransports(ctx)
	// Sample only after the wrapper quiesces SQLite, then restore its prior state.
	again, wrapperErr := s.RerunInstaller(ctx, "installer-node")
	noteJSON(&rec, "serving_rerun", again)
	first, e1 := s.UsageLedgerSnapshotBytes(ctx, "installer-node")
	second, e2 := s.UsageLedgerSnapshotBytes(ctx, "installer-node")
	note("quiescent_snapshot_byte_equal", boolText(e1 == nil && e2 == nil && bytes.Equal(first, second)))
	restoreErr := s.RestoreInstallerServices(ctx, "installer-node", again.Services)
	note("services_restored", boolText(restoreErr == nil))
	if err = s.HoldUsageAcknowledgements(ctx, "installer-node", false); err != nil {
		note("release", err.Error())
		return
	}
	target, _ := strconv.ParseUint(held.LastEventSeq, 10, 64)
	wait, cancel = context.WithTimeout(ctx, 150*time.Second)
	defer cancel()
	var released UsageSnapshot
	caught := waitUntil(wait, 2*time.Second, func() bool {
		released, err = s.UsageLedgerSnapshot(wait, "installer-node")
		ack, _ := strconv.ParseUint(released.AckedThrough, 10, 64)
		return err == nil && ack >= target
	})
	noteJSON(&rec, "released", released)
	transportErr := error(nil)
	if restoreErr == nil {
		_, transportErr = selected.StartEstablishedTransports(ctx, r.usageDevice)
	}
	note("post_wrapper_transport_restored", boolText(restoreErr == nil && transportErr == nil))
	note("scope", "same-incarnation hold/release only; authorized old-incarnation backlog delivery remains unresolved")
	if produced && held.AckedThrough == initial.AckedThrough && counterErr == nil && before.Present && after.Present && after.Upload > before.Upload && after.Download > before.Download && wrapperErr == nil && again.Exit == 0 && again.Before == again.Hashes && e1 == nil && e2 == nil && bytes.Equal(first, second) && restoreErr == nil && caught && transportErr == nil {
		rec.Status = StatusPass
	}
}

func (r *runner) installerCapture17(ctx context.Context, s *Stack) {
	rec := Record{ID: "capture-and-boot", Title: "Kernel identity and complete on-disk capture export", Spec: []string{"AT23", "ISO01", "F9-06"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	c := s.installerContainer()
	kernel, ke := r.lab.docker(c, "cat", "/proc/sys/kernel/random/boot_id")
	beat, be := r.lab.docker(c, "cat", "/run/vpn-node/heartbeat")
	regular, re := r.lab.docker(s.lab.prefix+"-node", "cat", "/etc/vpn-node/boot-id")
	regularKernel, rke := r.lab.docker(s.lab.prefix+"-node", "cat", "/proc/sys/kernel/random/boot_id")
	note := recPathNote(&rec)
	aligned := ke == nil && be == nil && re == nil && rke == nil && bytes.Equal(regularKernel, regular) && strings.Contains(string(beat), "boot_id "+strings.TrimSpace(string(kernel)))
	note("agent_boot_equals_kernel_and_lab", boolText(aligned))
	// Capture bounded local loopback traffic, independent of node capture tools.
	origin := s.lab.prefix + "-origin"
	cmd := exec.CommandContext(ctx, "docker", "exec", origin, "sh", "-c", `timeout 15 tcpdump -U -nn -i lo -c 2 -w /tmp/e2e-origin.pcap 'tcp port 443' >/tmp/e2e-capture.log 2>&1 & pid=$!; sleep 1; wget -q --no-check-certificate -O /dev/null https://127.0.0.1/healthz; wait "$pid"`)
	captureErr := cmd.Run()
	path := filepath.Join(s.lab.root, ".agent", "tmp", "followup17-origin.pcap")
	capture, exportErr := s.ExportContainerFile(ctx, "origin", "/tmp/e2e-origin.pcap", path)
	noteJSON(&rec, "origin_pcap", capture)
	// The independent large fixture exceeds the old diagnostic ceiling.
	_, createErr := r.lab.docker(origin, "sh", "-c", "dd if=/dev/zero of=/tmp/e2e-large-export bs=1048576 count=3 status=none")
	large, largeErr := s.ExportContainerFile(ctx, "origin", "/tmp/e2e-large-export", filepath.Join(s.lab.root, ".agent", "tmp", "followup17-large-export"))
	noteJSON(&rec, "large_export", large)
	_, overflowErr := r.lab.docker(origin, "cat", "/tmp/e2e-large-export")
	note("diagnostic_overflow_explicit", boolText(overflowErr != nil))
	if aligned && captureErr == nil && exportErr == nil && createErr == nil && largeErr == nil && large.Bytes == "3145728" && overflowErr != nil {
		rec.Status = StatusPass
	}
}

func (r *runner) installerReenroll17(ctx context.Context, s *Stack, admin BrowserSession) {
	rec := Record{ID: "installer-reenroll", Title: "Controlled reinstall, retained identity and retired certificate refusal", Spec: []string{"ENR-06", "AT07", "§6.3"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	if !r.revokeInstallerNode(ctx, s, admin, r.nodeID, note) {
		return
	}
	// Stop the first poll of the replacement until retained bytes are sampled.
	if err := s.HoldInstallerFirstPoll(ctx, true); err != nil {
		note("barrier", err.Error())
		return
	}
	token, _, err := r.issueInstallerToken(ctx, s, admin, "TEST ONLY installer replacement", r.nodeID)
	if err != nil {
		note("enrollment", err.Error())
		return
	}
	out, err := s.ControlledReinstall(ctx, "installer-node", token)
	token = ""
	noteJSON(&rec, "reenroll", out)
	if err != nil {
		note("wrapper", err.Error())
	}
	retired, retiredErr := s.retiredDirectory(ctx)
	oldHTTP := map[string]string{}
	if retiredErr == nil && retired != "" {
		for label, endpoint := range map[string]string{"direct": "https://agent.example.invalid:9443", "proxy": "https://127.0.0.1:9444"} {
			cmd := exec.CommandContext(ctx, "docker", "exec", s.installerContainer(), "curl", "--silent", "--show-error", "--max-time", "20", "--cacert", "/var/tmp/e2e-rel/ca.pem", "--cert", retired+"/cert.pem", "--key", retired+"/key.pem", "-o", "/dev/null", "-w", "%{http_code}", endpoint+"/api/v1/agent/desired-state")
			raw, e := cmd.Output()
			oldHTTP[label] = strings.TrimSpace(string(raw))
			if e != nil {
				oldHTTP[label] = "transport-error"
			}
		}
	}
	noteJSON(&rec, "retired_certificate_http", oldHTTP)
	moved := len(out.BoundStateBefore) == 4
	for name, digest := range out.BoundStateBefore {
		moved = moved && out.RetiredBoundState[name] == digest
	}
	kept := out.Before.UsageDB == out.Hashes.UsageDB && out.Before.UsageWAL == out.Hashes.UsageWAL
	identity := out.Before.Key == out.RetiredHashes.Key && out.Before.Cert == out.RetiredHashes.Cert && out.Before.NodeJSON == out.RetiredHashes.NodeJSON
	note("usage_retained", boolText(kept))
	note("bound_state_retired", boolText(moved))
	if err == nil && out.Exit == 0 && out.Status == "reenrolled" && out.NodeID == r.nodeID.String() && out.Incarnation != r.incID.String() && out.RetiredMode == "700" && kept && moved && identity && oldHTTP["direct"] == "403" && oldHTTP["proxy"] == "403" {
		rec.Status = StatusPass
	}
	if err = s.ConfigureInstallerServing(ctx); err == nil {
		err = s.HoldInstallerFirstPoll(ctx, false)
	}
	if err == nil {
		err = s.RestoreInstallerServices(ctx, "installer-node", out.Services)
	}
	note("restoration", boolText(err == nil))
}

func (r *runner) installerUpgrade17(ctx context.Context, s *Stack) {
	rec := Record{ID: "installer-journaled-upgrade", Title: "Required journaled installer upgrade; no synthetic meter checkpoint", Spec: []string{"UPD05", "AT26", "§13.5"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	if err := s.HoldInstallerApply(ctx, true); err != nil {
		note("barrier", err.Error())
		return
	}
	before, _ := r.lab.docker(s.installerContainer(), "readlink", "/opt/vpn-node/current")
	staged, err := s.ProbeInstallerExit(ctx, "installer-node", []string{"upgrade", "--json", "--manifest", "/var/tmp/e2e-rel/newer/manifest.json", "--artifact", "/var/tmp/e2e-rel/newer/artifact.tar.zst", "--trust", "/var/tmp/e2e-rel/newer/trust.json"})
	noteJSON(&rec, "stage", staged)
	if err != nil {
		note("stage_error", err.Error())
		return
	}
	pending := "/var/lib/vpn-node/update-staging/pending-release.json"
	_, absent := r.lab.docker(s.installerContainer(), "test", "!", "-e", "/var/lib/vpn-node/state/meter.checkpoint")
	note("meter_checkpoint_absent", boolText(absent == nil))
	raw, applyErr := r.lab.docker(s.installerContainer(), "/opt/vpn-node/current/fvpn-updater", "apply", "--config", "/etc/vpn-node/updater.json", "--pending", pending)
	note("apply_process", strings.TrimSpace(string(raw)))
	note("apply_process_success", boolText(applyErr == nil))
	status, statusErr := r.lab.docker(s.installerContainer(), "/opt/vpn-node/current/fvpn-updater", "status", "--config", "/etc/vpn-node/updater.json")
	if json.Valid(status) {
		noteJSON(&rec, "durable_result", json.RawMessage(status))
	}
	after, _ := r.lab.docker(s.installerContainer(), "readlink", "/opt/vpn-node/current")
	note("current_unchanged", boolText(bytes.Equal(before, after)))
	note("owner", "agent-core meter checkpoint; later service restart and candidate health remain unverified")
	var v struct {
		State  string            `json:"state"`
		Health map[string]string `json:"health"`
	}
	decoded := json.Unmarshal(status, &v) == nil
	healthy := true
	for _, check := range []string{"processes", "identity_and_state", "internet_only", "dns_policy", "tunnel_probe"} {
		healthy = healthy && v.Health[check] == "passed"
	}
	if statusErr == nil && decoded && v.State == "succeeded" && !bytes.Equal(before, after) && healthy {
		rec.Status = StatusPass
	}
}
