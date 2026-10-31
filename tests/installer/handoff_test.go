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
	url, ca, bundle := startControl(t, c)
	writeStdin(t, c, "/run/w5-token", token1+"\n", "0600")
	r, out, code := doInstall(t, c, relIn("1", rel1.ReleaseID), url, ca, bundle, "/run/w5-token")
	if code != 0 {
		t.Fatalf("install: %d %s", code, out)
	}
	assertBaseline(t, r.Baseline, out)
	testPendingHandoff(t, c, r.Baseline)
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
	stage := func() (installer.Result, string) {
		out, code := dexec(t, c, "/usr/local/bin/fvpn-node-install", "upgrade", "--json",
			"--manifest", relIn("2", rel2.ReleaseID).Manifest,
			"--artifact", relIn("2", rel2.ReleaseID).Artifact,
			"--trust", relIn("2", rel2.ReleaseID).Trust)
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
		if res.ManifestPath != record["manifest_path"] || sha(t, c, res.ManifestPath) != sha(t, c, relIn("2", rel2.ReleaseID).Manifest) {
			t.Fatal("retained envelope differs from verified input")
		}
		check := mustCheck(t, c, 0)
		if check.PendingRelease == nil || check.PendingRelease.RequestID != res.RequestID || check.PendingRelease.ManifestPath != res.ManifestPath {
			t.Fatalf("check lost pending record: %+v", check)
		}
		t.Logf("pending record: %s", raw)
		return res, raw
	}
	status := func() map[string]any {
		raw := mustExec(t, c, updater, "status", "--config", config)
		var view map[string]any
		if err := json.Unmarshal([]byte(raw), &view); err != nil {
			t.Fatal(err)
		}
		if view["source"] != "installer" || view["requested_release_id"] != rel2.ReleaseID || view["job_id"] == "" {
			t.Fatalf("installer status: %s", raw)
		}
		t.Logf("updater status: %s", raw)
		return view
	}
	first, raw := stage()
	if got := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current")); got != before {
		t.Fatal("staging moved current")
	}
	applyOut, code := dexec(t, c, updater, "apply", "--config", config, "--pending", pendingPath)
	view := status()
	if baseline.Status != "seeded" {
		t.Fatal("real seeded baseline required")
	}
	if code != 0 || view["state"] != "succeeded" || view["current_release_id"] != rel2.ReleaseID {
		_, checkpointCode := dexec(t, c, "test", "-f", "/var/lib/vpn-node/state/meter.checkpoint")
		t.Fatalf("verified baseline must apply: exit=%d state=%v stage=%v current=%v meter_checkpoint_present=%t\n%s", code, view["state"], view["stage"], view["current_release_id"], checkpointCode == 0, applyOut)
	}
	checked := mustCheck(t, c, 0)
	if checked.ReleaseID != rel2.ReleaseID || checked.ReleaseSource != "updater_journal" || checked.InstallRecord.ReleaseID != rel1.ReleaseID {
		t.Fatalf("post-switch check did not reconcile install record: %+v", checked)
	}
	t.Log("real seeded upgrade: verify/switch/restart/health succeeded; post-switch check healthy; install-time record retained")
	firstJob := view["job_id"]
	// Exact record replay remains the same terminal job.
	writeStdin(t, c, pendingPath, raw, "0600")
	mustExec(t, c, updater, "apply", "--config", config, "--pending", pendingPath)
	if status()["job_id"] != firstJob {
		t.Fatal("identical replay became new work")
	}
	second, _ := stage()
	if second.RequestID == first.RequestID {
		t.Fatal("second invocation reused request id")
	}
	_, _ = dexec(t, c, updater, "apply", "--config", config, "--pending", pendingPath)
	if status()["job_id"] == firstJob {
		t.Fatal("new request id did not create new work")
	}
	// Signature verification must fail before switching the current tree.
	third, _ := stage()
	current := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current"))
	envelope := mustExec(t, c, "cat", third.ManifestPath)
	tampered := strings.Replace(envelope, `"1.2.2"`, `"9.9.9"`, 1)
	if tampered == envelope {
		t.Fatal("fixture product version not found")
	}
	writeStdin(t, c, third.ManifestPath, tampered, "0600")
	out, code := dexec(t, c, updater, "apply", "--config", config, "--pending", pendingPath)
	if code != 1 || !strings.Contains(out, "HASH_MISMATCH") {
		t.Fatalf("tampered envelope: %d\n%s", code, out)
	}
	if view := status(); view["state"] != "failed" || view["stage"] != "verify" {
		t.Fatalf("tamper status: %+v", view)
	}
	if got := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current")); got != current {
		t.Fatal("tampered envelope moved current")
	}
	writeStdin(t, c, third.ManifestPath, envelope, "0600")
	mustExec(t, c, "rm", "/run/systemd/system/fvpn-updater.service.d/stage.conf")
	mustExec(t, c, "systemctl", "daemon-reload")
}
