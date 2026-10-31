//go:build installer

package installertest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
	"github.com/google/uuid"
)

func TestD089_MergedUpdaterSeedProbe(t *testing.T) {
	const name = "fvpn-w5-node-installer-seed-probe"
	startSleep(t, name)
	stage(t, name)
	out, code := dexec(t, name, "/var/tmp/w5/rel1/tree-1/fvpn-updater", "seed", "--help")
	if (code != 0 && code != 64) || !strings.Contains(out, "-manifest") || !strings.Contains(out, "-tree") {
		t.Fatalf("seed probe: exit=%d stderr=%q", code, out)
	}
	t.Logf("seed subcommand available; probe exit=%d stderr=%q", code, out)
}

func assertBaseline(t *testing.T, baseline *installer.BaselineResult, log string) {
	t.Helper()
	if baseline == nil || !strings.Contains(log, "baseline: "+baseline.Status) {
		t.Fatalf("baseline absent from result/log: %+v\n%s", baseline, log)
	}
	switch baseline.Status {
	case "seeded":
		if baseline.Seed == nil || baseline.Seed.ExitCode != 0 {
			t.Fatalf("seeded without command evidence: %+v", baseline)
		}
	default:
		t.Fatalf("baseline: %+v", baseline)
	}
	t.Logf("baseline: %s; probe exit=%d stderr=%q", baseline.Status, baseline.Probe.ExitCode, baseline.Probe.Stderr)
}

// This is deliberately a separate required scenario: a failure must not hide
// reenrollment, removal or image lifecycle evidence.
func TestD089_RealSeedThenJournaledUpgrade(t *testing.T) {
	const c = "fvpn-w5-node-installer-upgrade"
	startSystemd(t, c)
	stage(t, c)
	url, ca, bundle := startControl(t, c, "--upgrade-health")
	writeStdin(t, c, "/run/w5-token", token1+"\n", "0600")
	r, out, code := doInstall(t, c, relIn("1", rel1.ReleaseID), url, ca, bundle, "/run/w5-token", "--desired-trust", bundle)
	if code != 0 {
		t.Fatalf("install: %d %s", code, out)
	}
	assertBaseline(t, r.Baseline, out)
	assertSeedUmasks(t, c)
	enableUpgradeProbe(t, c)
	testPendingHandoff(t, c, r.Baseline)
	assertPostUpgradeReenroll(t, c, url, ca)
}

func assertSeedUmasks(t *testing.T, c string) {
	t.Helper()
	const updater = "/opt/vpn-node/current/fvpn-updater"
	const config = "/etc/vpn-node/updater.json"
	baseline := func() string {
		var status struct {
			Seed json.RawMessage `json:"seeded_baseline"`
		}
		if err := json.Unmarshal([]byte(mustExec(t, c, updater, "status", "--config", config)), &status); err != nil || len(status.Seed) == 0 {
			t.Fatalf("seeded baseline: %s %v", status.Seed, err)
		}
		return string(status.Seed)
	}
	before := baseline()
	for _, mask := range []string{"022", "077"} {
		mustExec(t, c, "sh", "-c", `umask "$1"; shift; exec "$@"`, "sh", mask,
			updater, "seed", "--config", config,
			"--manifest", "/var/lib/vpn-node/update-staging/"+rel1.ReleaseID+".manifest.json",
			"--tree", "/opt/vpn-node/releases/"+rel1.ReleaseID)
		if baseline() != before {
			t.Fatal("seed replay changed baseline identity, tree hash or timestamp")
		}
	}
	t.Log("F9-02: real standalone seed under umasks 022 and 077 passed with unchanged baseline/hash/timestamp")
}

func enableUpgradeProbe(t *testing.T, c string) {
	t.Helper()
	// TEST ONLY NetBird management/relay status, matching agent-core 24's
	// UpgradeHealthFixture regression. The daemon/socket lifecycle remains
	// separately real. The candidate still runs its actual kernel verifier,
	// signed desired/lease validation and TCP dial before writing an attestation.
	mustExec(t, c, "systemctl", "stop", "fvpn-agent")
	const fixture = "/run/w5-control/netbird-status.json"
	writeStdin(t, c, fixture, `{"daemonVersion":"`+installer.NetBirdPin+`","management":{"connected":true},"relays":{"total":1,"available":1}}`, "0600")
	var cfg map[string]any
	if err := json.Unmarshal([]byte(mustExec(t, c, "cat", "/etc/vpn-node/agent.json")), &cfg); err != nil {
		t.Fatal(err)
	}
	cfg["netbird_status_file"] = fixture
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	writeStdin(t, c, "/etc/vpn-node/agent.json", string(raw), "0644")
	mustExec(t, c, "systemctl", "start", "fvpn-agent")
	waitPath(t, c, "/var/lib/vpn-node/state/desired-state.json")
	// D-125 publishes only for an actual updater block. Waiting for a quiet
	// shutdown checkpoint here would deadlock before apply creates its latch.
	assertNoBarrier(t, c)
	t.Log("TEST ONLY NetBird management/relay status; signed unpublished desired/lease and real isolated TCP probe; no checkpoint before the updater block")
}

func testPendingHandoff(t *testing.T, c string, baseline *installer.BaselineResult) {
	t.Helper()
	const pendingPath = "/var/lib/vpn-node/update-staging/pending-release.json"
	const updater = "/opt/vpn-node/current/fvpn-updater"
	const config = "/etc/vpn-node/updater.json"
	// TEST ONLY: keep the hand-off observable before explicit apply. The
	// installed updater binary, configuration and journal remain real.
	mustExec(t, c, "mkdir", "-p", "/run/systemd/system/fvpn-updater.service.d")
	writeStdin(t, c, "/run/systemd/system/fvpn-updater.service.d/stage.conf", "[Service]\nExecCondition=/bin/false\n", "0644")
	mustExec(t, c, "systemctl", "daemon-reload")
	before := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current"))
	stage := func(rel crel) (installer.Result, string) {
		out, code := dexec(t, c, "/usr/local/bin/fvpn-node-install", "upgrade", "--json",
			"--manifest", rel.Manifest,
			"--artifact", rel.Artifact,
			"--trust", rel.Trust)
		var res installer.Result
		if err := decodeJSON(out, &res); err != nil || code != 0 || res.Status != "staged" {
			t.Fatalf("upgrade %d %+v %v\n%s", code, res, err, out)
		}
		raw := mustExec(t, c, "cat", pendingPath)
		var record map[string]string
		if err := json.Unmarshal([]byte(raw), &record); err != nil {
			t.Fatal(err)
		}
		for _, field := range []string{"release_id", "manifest_sha256", "artifact_path", "tree_path", "updater_command", "manifest_path", "request_id"} {
			if record[field] == "" {
				t.Fatalf("missing %s: %s", field, raw)
			}
		}
		if id, err := uuid.Parse(record["request_id"]); err != nil || id == uuid.Nil || res.RequestID != record["request_id"] {
			t.Fatalf("request id: %s", raw)
		}
		if res.ManifestPath != record["manifest_path"] || sha(t, c, res.ManifestPath) != sha(t, c, rel.Manifest) {
			t.Fatal("retained envelope differs from verified input")
		}
		check := mustCheck(t, c, 0)
		if check.PendingRelease == nil || check.PendingRelease.RequestID != res.RequestID || check.PendingRelease.ManifestPath != res.ManifestPath {
			t.Fatalf("check lost pending record: %+v", check)
		}
		t.Logf("pending record: %s", raw)
		return res, raw
	}
	status := func(releaseID string) map[string]any {
		raw := mustExec(t, c, updater, "status", "--config", config)
		var view map[string]any
		if err := json.Unmarshal([]byte(raw), &view); err != nil {
			t.Fatal(err)
		}
		if view["source"] != "installer" || view["requested_release_id"] != releaseID || view["job_id"] == "" {
			t.Fatalf("installer status: %s", raw)
		}
		t.Logf("updater status: %s", raw)
		return view
	}
	next := relIn("2", rel2.ReleaseID)
	first, raw := stage(next)
	if got := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current")); got != before {
		t.Fatal("staging moved current")
	}
	assertNoBarrier(t, c)
	// Reproduce devstack 18's direct apply followed by installer seed (077).
	applyOut, code := dexec(t, c, "sh", "-c", `umask 022; exec "$@"`, "sh", updater, "apply", "--config", config, "--pending", pendingPath)
	view := status(rel2.ReleaseID)
	if baseline.Status != "seeded" {
		t.Fatal("real seeded baseline required")
	}
	if code != 0 || view["state"] != "succeeded" || view["current_release_id"] != rel2.ReleaseID {
		_, checkpointCode := dexec(t, c, "test", "-f", "/var/lib/vpn-node/state/meter.checkpoint")
		t.Log(mustExec(t, c, "journalctl", "-u", "fvpn-agent", "--no-pager", "-n", "30"))
		t.Fatalf("verified baseline must apply: exit=%d state=%v stage=%v current=%v meter_checkpoint_present=%t\n%s", code, view["state"], view["stage"], view["current_release_id"], checkpointCode == 0, applyOut)
	}
	health, ok := view["health"].(map[string]any)
	if !ok || view["journal_stage"] != "locally_succeeded" || view["recovery_required"] != false {
		t.Fatalf("missing terminal health: %+v", view)
	}
	for _, check := range []string{"processes", "identity_and_state", "internet_only", "dns_policy", "tunnel_probe"} {
		if health[check] != "passed" {
			t.Fatalf("health %s: %v", check, health[check])
		}
	}
	if got := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current")); got != "releases/"+rel2.ReleaseID {
		t.Fatalf("candidate pointer did not switch: %s", got)
	}
	assertCorrelatedReceipt(t, c, view["job_id"].(string))
	checked := mustCheck(t, c, 0)
	if checked.ReleaseID != rel2.ReleaseID || checked.ReleaseSource != "updater_journal" || checked.InstallRecord.ReleaseID != rel1.ReleaseID {
		t.Fatalf("post-switch check did not reconcile install record: %+v", checked)
	}
	t.Log("real seeded upgrade: verify/switch/restart/health succeeded; post-switch check healthy; install-time record retained")
	assertCLIDowngrade(t, c)
	firstJob := view["job_id"]
	// Exact record replay remains the same terminal job. This unpublished
	// fixture must retain the forwarding-blocked sentinel across recovery.
	writeStdin(t, c, pendingPath, raw, "0600")
	replayOut, replayCode := dexec(t, c, updater, "apply", "--config", config, "--pending", pendingPath)
	if replayCode != 2 || !strings.Contains(replayOut, "member forwarding must stay blocked") {
		t.Fatalf("unpublished replay: %d %s", replayCode, replayOut)
	}
	mustExec(t, c, "test", "-f", "/var/lib/vpn-node/update-journal/forwarding.blocked")
	if got := status(rel2.ReleaseID); got["job_id"] != firstJob || got["state"] != "succeeded" || got["current_release_id"] != rel2.ReleaseID {
		t.Fatal("identical replay became new work")
	}
	second, _ := stage(next)
	if second.RequestID == first.RequestID {
		t.Fatal("second invocation reused request id")
	}
	restageOut, restageCode := dexec(t, c, updater, "apply", "--config", config, "--pending", pendingPath)
	if restageCode != 1 || !strings.Contains(restageOut, "VALIDATION_FAILED") {
		t.Fatalf("changed record must be refused: %d %s", restageCode, restageOut)
	}
	if got := status(rel2.ReleaseID); got["job_id"] != firstJob || got["state"] != "succeeded" {
		t.Fatal("new request id changed the terminal job")
	}
	t.Log("D-121: exact successful replay keeps job with exit 2 forwarding-blocked sentinel; changed request ID is refused without changing terminal success")
	// Signature verification must fail before switching the current tree.
	// D-110/D-121: a distinct release/manifest gives tampering its own job.
	third, _ := stage(relIn("3", rel3.ReleaseID))
	current := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current"))
	envelope := mustExec(t, c, "cat", third.ManifestPath)
	tampered := strings.Replace(envelope, `"1.2.3"`, `"9.9.9"`, 1)
	if tampered == envelope {
		t.Fatal("fixture product version not found")
	}
	writeStdin(t, c, third.ManifestPath, tampered, "0600")
	out, code := dexec(t, c, updater, "apply", "--config", config, "--pending", pendingPath)
	if code != 2 || !strings.Contains(out, "HASH_MISMATCH") || !strings.Contains(out, "member forwarding must stay blocked") {
		t.Fatalf("tampered envelope: %d\n%s", code, out)
	}
	if view := status(rel3.ReleaseID); view["state"] != "failed" || view["stage"] != "verify" {
		t.Fatalf("tamper status: %+v", view)
	}
	if got := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current")); got != current {
		t.Fatal("tampered envelope moved current")
	}
	writeStdin(t, c, third.ManifestPath, envelope, "0600")
	mustExec(t, c, "rm", "/run/systemd/system/fvpn-updater.service.d/stage.conf")
	mustExec(t, c, "systemctl", "daemon-reload")
}

func TestUPD05_StoppedProbeFailsRealHealth(t *testing.T) {
	const c = "fvpn-w5-node-installer-probe-failure"
	startSystemd(t, c)
	stage(t, c)
	url, ca, bundle := startControl(t, c, "--upgrade-health")
	writeStdin(t, c, "/run/w5-token", token1+"\n", "0600")
	_, out, code := doInstall(t, c, relIn("1", rel1.ReleaseID), url, ca, bundle, "/run/w5-token", "--desired-trust", bundle)
	if code != 0 {
		t.Fatalf("install: %d %s", code, out)
	}
	enableUpgradeProbe(t, c)
	admin := strings.TrimSpace(mustExec(t, c, "cat", "/run/w5-control/admin"))
	mustExec(t, c, "curl", "--fail", "--silent", "--show-error", "-X", "POST", admin+"/stop-probe")
	mustExec(t, c, "mkdir", "-p", "/run/systemd/system/fvpn-updater.service.d")
	writeStdin(t, c, "/run/systemd/system/fvpn-updater.service.d/stage.conf", "[Service]\nExecCondition=/bin/false\n", "0644")
	mustExec(t, c, "systemctl", "daemon-reload")
	next := relIn("2", rel2.ReleaseID)
	mustExec(t, c, "/usr/local/bin/fvpn-node-install", "upgrade", "--json", "--manifest", next.Manifest, "--artifact", next.Artifact, "--trust", next.Trust)
	const updater = "/opt/vpn-node/current/fvpn-updater"
	const config = "/etc/vpn-node/updater.json"
	out, code = dexec(t, c, updater, "apply", "--config", config, "--pending", "/var/lib/vpn-node/update-staging/pending-release.json")
	var status struct {
		State     string            `json:"state"`
		Stage     string            `json:"stage"`
		Current   string            `json:"current_release_id"`
		Recovery  bool              `json:"recovery_required"`
		Candidate map[string]string `json:"candidate_health_json"`
	}
	raw := mustExec(t, c, updater, "status", "--config", config)
	if err := json.Unmarshal([]byte(raw), &status); err != nil {
		t.Fatal(err)
	}
	if code != 1 || !strings.Contains(out, "ROLLBACK_FAILED") || status.State != "failed" || status.Stage != "rollback" || status.Current != rel1.ReleaseID || !status.Recovery {
		t.Fatalf("stopped probe: exit=%d %s\n%s", code, out, raw)
	}
	if status.Candidate["tunnel_probe"] != "failed" || status.Candidate["internet_only"] != "passed" || status.Candidate["dns_policy"] != "passed" || status.Candidate["processes"] != "passed" {
		t.Fatalf("probe failure was not isolated to real TCP reachability: %s", raw)
	}
	mustExec(t, c, "test", "-f", "/var/lib/vpn-node/update-journal/forwarding.blocked")
	t.Log("real stopped TCP listener: candidate tunnel_probe failed; processes/isolation/DNS passed; restored health also failed; ROLLBACK_FAILED exit 1 with old release current and recovery_required")
}
