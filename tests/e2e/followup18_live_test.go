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
func TestE2E_Followup18Installer(t *testing.T) {
	if os.Getenv("FVPN_E2E_FOLLOWUP18") != "1" {
		t.Skip("set FVPN_E2E_FOLLOWUP18=1 with a fresh owned stack")
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
	r := &runner{t: t, lab: l, runID: id, sourceCommit: strings.TrimSpace(string(head)), dir: filepath.Join(l.root, "docs/e2e/runs", time.Now().UTC().Format("2006-01-02"), "followup18", id)}
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
	ruleRevision, _, err := s.ActivatePolicy(ctx, b, "domain_rule_set", "installer-dns-rules", map[string]any{"rules": []any{map[string]any{"rule_id": uuid.New(), "domain": "feed-01.example.invalid", "include_subdomains": true, "action": "block", "profile_id": nil, "reason": "admin_rule", "revision": "1"}}})
	must(err)
	r.profileID, _, err = s.ActivatePolicy(ctx, b, "dns_profile", "installer-dns", map[string]any{"profile_code": "installer-dns", "display_name": map[string]string{"zh-TW": "TEST ONLY installer resolver"}, "block_response": "nxdomain", "blocked_ttl_seconds": 5, "domain_rule_set_revision_ids": []uuid.UUID{ruleRevision}, "threat_feed_revision_ids": []string{}})
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
	r.installerMarker18(ctx, s)
	// Multiple deliberate wrapper samples otherwise exceed the packaged
	// StartLimitBurst=5 / StartLimitIntervalSec=120. Keep that safety intact.
	t.Log("waiting 121 seconds between deliberate installer restart campaigns")
	for i := 0; i < 121; i++ {
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(time.Second):
		}
	}
	r.installerUsage18(ctx, s, selected)
	r.installerServing18(ctx, s)
	r.installerCapture17(ctx, s)
	r.installerUpgrade18(ctx, s)
	r.installerReenroll17(ctx, s, b)
	// Exercise the changed ordinary entrypoint with an enrolled agent, then a
	// real process restart. Do this after installer measurements because the
	// baseline helper activates its own serving support policy.
	r.lab = l
	r.nodeID, r.incID, r.enrolledID = uuid.Nil, uuid.Nil, uuid.Nil
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

// Recreate only the exact obsolete empty marker, while services are stopped.
// The installer itself must migrate it; the lab never removes that chain.
func (r *runner) installerMarker18(ctx context.Context, s *Stack) {
	rec := Record{ID: "installer-marker-migration", Title: "D-116 legacy marker migrated by same-release install", Spec: []string{"AT07", "ENR-05", "D-116"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	services, err := s.InstallerServices(ctx, "installer-node")
	if err != nil {
		note("error", err.Error())
		return
	}
	if err = s.stopInstallerAgent(ctx); err != nil {
		note("error", err.Error())
		return
	}
	defer func() { _ = s.RestoreInstallerServices(context.WithoutCancel(ctx), "installer-node", services) }()
	c := s.installerContainer()
	before, _ := r.lab.docker(c, "nft", "-j", "list", "ruleset")
	note("before_ruleset", string(before))
	script := "delete table inet fvpn_install\nadd chain inet fvpn owner { comment \"managed-by=fvpn-node-install\"; }\n"
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", c, "nft", "-f", "-")
	cmd.Stdin = strings.NewReader(script)
	if err = cmd.Run(); err != nil {
		note("fixture", err.Error())
		return
	}
	legacy, _ := r.lab.docker(c, "nft", "-j", "list", "ruleset")
	note("legacy_ruleset", string(legacy))
	check, _ := r.lab.docker(c, "/usr/local/bin/fvpn-node-install", "check", "--json")
	note("legacy_check", string(check))
	out, err := s.RerunInstaller(ctx, "installer-node")
	noteJSON(&rec, "migration", out)
	after, _ := r.lab.docker(c, "nft", "-j", "list", "ruleset")
	note("after_ruleset", string(after))
	check, _ = r.lab.docker(c, "/usr/local/bin/fvpn-node-install", "check", "--json")
	note("migrated_check", string(check))
	if err == nil && out.Exit == 0 && strings.Contains(string(check), `"legacy_owner": false`) {
		rec.Status = StatusPass
	}
}

func (r *runner) installerUpgrade18(ctx context.Context, s *Stack) {
	rec := Record{ID: "installer-journaled-upgrade", Title: "Genuine final sample and staged installer candidate health", Spec: []string{"UPD05", "AT26", "§13.5"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	c := s.installerContainer()
	if err := s.HoldInstallerApply(ctx, true); err != nil {
		note("barrier", err.Error())
		return
	}
	before, _ := r.lab.docker(c, "readlink", "/opt/vpn-node/current")
	staged, err := s.ProbeInstallerExit(ctx, "installer-node", []string{"upgrade", "--json", "--manifest", "/var/tmp/e2e-rel/newer/manifest.json", "--artifact", "/var/tmp/e2e-rel/newer/artifact.tar.zst", "--trust", "/var/tmp/e2e-rel/newer/trust.json"})
	noteJSON(&rec, "stage", staged)
	if err != nil {
		note("stage_error", err.Error())
		return
	}
	pre, _ := r.lab.docker(c, "sh", "-c", `if [ -e /var/lib/vpn-node/state/meter.checkpoint ]; then cat /var/lib/vpn-node/state/meter.checkpoint; else echo absent; fi`)
	note("checkpoint_before", string(pre))
	raw, applyErr := r.lab.docker(c, "/opt/vpn-node/current/fvpn-updater", "apply", "--config", "/etc/vpn-node/updater.json", "--pending", "/var/lib/vpn-node/update-staging/pending-release.json")
	note("apply_process", string(raw))
	note("apply_process_success", boolText(applyErr == nil))
	status, statusErr := r.lab.docker(c, "/opt/vpn-node/current/fvpn-updater", "status", "--config", "/etc/vpn-node/updater.json")
	note("durable_result", string(status))
	checkpoint, checkpointErr := r.lab.docker(c, "cat", "/var/lib/vpn-node/state/meter.checkpoint")
	note("genuine_meter_checkpoint_present", boolText(checkpointErr == nil))
	note("checkpoint_after", string(checkpoint))
	_, copyErr := r.lab.docker(c, "/usr/local/bin/fvpn-e2e-nodeconfig", "journal-copy")
	journal, exportErr := s.ExportContainerFile(ctx, "installer-node", "/var/lib/vpn-node/e2e/journal.snapshot.db", filepath.Join(r.lab.root, ".agent/tmp", r.runID+"-installer-journal.db"))
	noteJSON(&rec, "journal_snapshot", journal)
	note("journal_copy_complete", boolText(copyErr == nil && exportErr == nil))
	after, _ := r.lab.docker(c, "readlink", "/opt/vpn-node/current")
	note("current_before", string(before))
	note("current_after", string(after))
	logs, _ := r.lab.docker(c, "journalctl", "-u", "fvpn-agent", "-u", "fvpn-updater", "--no-pager", "-n", "100")
	note("agent_updater_journal", string(logs))
	var v struct {
		State  string            `json:"state"`
		Health map[string]string `json:"health"`
	}
	decoded := json.Unmarshal(status, &v) == nil
	healthy := true
	for _, k := range []string{"processes", "identity_and_state", "internet_only", "dns_policy", "tunnel_probe"} {
		healthy = healthy && v.Health[k] == "passed"
	}
	if copyErr == nil && exportErr == nil && statusErr == nil && decoded && v.State == "succeeded" && checkpointErr == nil && !bytes.Equal(before, after) && healthy {
		rec.Status = StatusPass
	}
	note("owner", "updater / agent-core; durable result decides success, independent of apply exit")
}

func (r *runner) installerServing18(ctx context.Context, s *Stack) {
	rec := Record{ID: "installer-serving-readback", Title: "D-116 four-chain verifier and separate hookless marker after traffic restoration", Spec: []string{"ENR-05", "D-116", "AT07"}, Status: StatusFail}
	defer func() { r.record(rec) }()
	note := recPathNote(&rec)
	c := s.installerContainer()
	beat, _ := r.lab.docker(c, "cat", "/run/vpn-node/heartbeat")
	note("heartbeat", string(beat))
	for _, table := range []string{"fvpn", "fvpn_install"} {
		raw, _ := r.lab.docker(c, "nft", "-j", "list", "table", "inet", table)
		note("table_"+table, string(raw))
	}
	check, _ := r.lab.docker(c, "/usr/local/bin/fvpn-node-install", "check", "--json")
	note("installer_check", string(check))
	logs, _ := r.lab.docker(c, "journalctl", "-u", "fvpn-agent", "--no-pager", "-n", "60")
	note("agent_journal", string(logs))
	db, err := r.lab.db(ctx)
	if err != nil {
		return
	}
	defer db.Close(ctx)
	var row []byte
	err = db.QueryRow(ctx, `SELECT jsonb_build_object('received_at',received_at,'report',report_payload) FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC LIMIT 1`, r.nodeID).Scan(&row)
	note("central_observation", string(row))
	_, activeErr := r.lab.docker(c, "systemctl", "is-active", "--quiet", "fvpn-agent.service")
	note("agent_active", boolText(activeErr == nil))
	if err == nil && activeErr == nil && servingHeartbeatSince(beat, time.Now().Add(-15*time.Second)) {
		rec.Status = StatusPass
	}
}

func (r *runner) installerUsage18(ctx context.Context, s, selected *Stack) {
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
	restoreAt := time.Now()
	restoreErr := s.RestoreInstallerServices(ctx, "installer-node", again.Services)
	note("services_restored", boolText(restoreErr == nil))
	if restoreErr != nil {
		note("restoration_error", restoreErr.Error())
	}
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
	// A successful systemctl start is not a serving readback. Wait for a
	// fresh heartbeat before opening the post-wrapper native sockets.
	fresh, stop := context.WithTimeout(ctx, 45*time.Second)
	ready := waitUntil(fresh, time.Second, func() bool {
		b, e := r.lab.docker(s.installerContainer(), "cat", "/run/vpn-node/heartbeat")
		return e == nil && servingHeartbeatSince(b, restoreAt)
	})
	stop()
	note("serving_after_restore", boolText(ready))
	if restoreErr == nil && ready {
		_, transportErr = selected.StartEstablishedTransports(ctx, r.usageDevice)
	}
	note("post_wrapper_transport_restored", boolText(restoreErr == nil && ready && transportErr == nil))
	note("scope", "same-incarnation hold/release only; authorized old-incarnation backlog delivery remains unresolved")
	if produced && held.AckedThrough == initial.AckedThrough && counterErr == nil && before.Present && after.Present && after.Upload > before.Upload && after.Download > before.Download && wrapperErr == nil && again.Exit == 0 && again.Before == again.Hashes && e1 == nil && e2 == nil && bytes.Equal(first, second) && restoreErr == nil && caught && ready && transportErr == nil {
		rec.Status = StatusPass
	}
}
