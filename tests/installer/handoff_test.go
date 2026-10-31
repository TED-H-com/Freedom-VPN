//go:build installer

package installertest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
	"github.com/google/uuid"
)

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
	case "not seeded (updater without seed)":
		if baseline.Probe.ExitCode != 64 || !strings.Contains(baseline.Probe.Stderr, `unknown command "seed"`) || baseline.Seed != nil {
			t.Fatalf("missing seed without probe evidence: %+v", baseline)
		}
	default:
		t.Fatalf("baseline: %+v", baseline)
	}
	t.Logf("baseline: %s; probe exit=%d stderr=%q", baseline.Status, baseline.Probe.ExitCode, baseline.Probe.Stderr)
}

func testPendingHandoff(t *testing.T, c string, baseline *installer.BaselineResult) {
	t.Helper()
	const pendingPath = "/var/lib/vpn-node/update-staging/pending-release.json"
	const updater = "/opt/vpn-node/current/fvpn-updater"
	const config = "/etc/vpn-node/updater.json"
	// TEST ONLY: keep the hand-off observable before explicit apply. The
	// installed updater binary, configuration and journal remain real.
	mustExec(t, c, "systemctl", "stop", "fvpn-updater.timer")
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
	if baseline.Status == "seeded" {
		if code != 0 || view["state"] != "locally_succeeded" || view["current_release_id"] != rel2.ReleaseID {
			t.Fatalf("verified baseline must apply: exit=%d status=%+v\n%s", code, view, applyOut)
		}
	} else {
		if code != 1 || !strings.Contains(applyOut, "CHECKPOINT_FAILED") || view["state"] != "failed" || view["stage"] != "verify" {
			t.Fatalf("unseeded baseline must fail closed: exit=%d status=%+v\n%s", code, view, applyOut)
		}
		t.Log("BLOCKED: merged updater has no seed; verify/switch/restart/health success needs updater follow-up 14. No journal evidence was fabricated.")
	}
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
	// Verification fails before the baseline checkpoint even on the merged
	// updater without seed. The current tree must remain unchanged.
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
