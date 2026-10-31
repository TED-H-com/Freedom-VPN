//go:build installer

package installertest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
)

func testENR06Runtime(t *testing.T, c string) {
	t.Helper()
	const socket = "/run/fvpn-netbird/netbird.sock"
	mustExec(t, c, "test", "-S", socket)
	waitPath(t, c, "/run/vpn-node/heartbeat")
	inode := mustExec(t, c, "stat", "-c", "%i", "/run/vpn-node")
	for _, unit := range []string{"fvpn-agent", "fvpn-watchdog", "fvpn-updater", "fvpn-updater-recover", "fvpn-netbird"} {
		mustExec(t, c, "systemctl", "stop", unit)
		if unit == "fvpn-agent" {
			mustExec(t, c, "systemctl", "is-active", "fvpn-watchdog")
		}
		if unit == "fvpn-netbird" {
			mustExec(t, c, "test", "!", "-e", socket)
		}
		if got := mustExec(t, c, "stat", "-c", "%i", "/run/vpn-node"); got != inode {
			t.Fatalf("%s stop replaced shared runtime directory", unit)
		}
		mustExec(t, c, "systemctl", "start", unit)
	}
	mustExec(t, c, "systemctl", "start", "fvpn-node-netbird-up")
	mustExec(t, c, "test", "-S", socket)
	mustExec(t, c, "systemctl", "is-active", "fvpn-netbird", "fvpn-agent", "fvpn-watchdog")
	mustExec(t, c, "/usr/lib/fvpn-tests/netbird", "status", "--daemon-addr", "unix://"+socket)
	mustCheck(t, c, 0)
	logs := mustExec(t, c, "journalctl", "-u", "fvpn-netbird", "-u", "fvpn-agent", "--no-pager")
	if strings.Contains(logs, "read-only file system") {
		for _, line := range strings.Split(logs, "\n") {
			if strings.Contains(line, "read-only file system") {
				t.Log(line)
			}
		}
		t.Fatal("ENR-06 read-only-path diagnostic returned")
	}
	t.Log("ENR-06: real NetBird socket bound; enrollment -> services active -> check healthy; shared runtime survives all consumer stop/start cycles")
}

func seededStatus(t *testing.T, c, releaseID string) string {
	t.Helper()
	raw := mustExec(t, c, "/opt/vpn-node/current/fvpn-updater", "status", "--config", "/etc/vpn-node/updater.json")
	var st struct {
		Current string          `json:"current_release_id"`
		Seed    json.RawMessage `json:"seeded_baseline"`
	}
	if err := json.Unmarshal([]byte(raw), &st); err != nil || st.Current != releaseID || len(st.Seed) == 0 {
		t.Fatalf("real seeded baseline: %s, %v", raw, err)
	}
	return string(st.Seed)
}

func TestD089_SeedRefusalKeepsInstalledTreeAndJournal(t *testing.T) {
	const c = "fvpn-w5-node-installer-seed-refusal"
	startSystemd(t, c)
	stage(t, c)
	url, ca, bundle := startControl(t, c)
	writeStdin(t, c, "/run/w5-token", token1+"\n", "0600")
	r, out, code := doInstall(t, c, relIn("1", rel1.ReleaseID), url, ca, bundle, "/run/w5-token")
	if code != 0 {
		t.Fatalf("install: %d %s", code, out)
	}
	assertBaseline(t, r.Baseline, out)
	before := seededStatus(t, c, rel1.ReleaseID)
	key := sha(t, c, "/var/lib/vpn-node/identity/key.pem")
	_, out, code = doInstall(t, c, relIn("2", rel2.ReleaseID), url, ca, bundle, "")
	if code != installer.ExitFailed || !strings.Contains(out, "baseline_failed") {
		t.Fatalf("different journal seed: %d %s", code, out)
	}
	if got := strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current")); got != "releases/"+rel2.ReleaseID {
		t.Fatal("seed refusal rolled back the completed installation")
	}
	if before != seededStatus(t, c, rel1.ReleaseID) || key != sha(t, c, "/var/lib/vpn-node/identity/key.pem") {
		t.Fatal("seed refusal changed established journal or identity")
	}
	check := mustCheck(t, c, installer.ExitDrifted)
	if check.Baseline.Status != "failed" || check.Baseline.Seed == nil ||
		!strings.Contains(check.Baseline.Seed.Stderr, "seed refused: current release differs from journal") ||
		!strings.Contains(strings.Join(check.Drifts, ","), "current") {
		t.Fatalf("seed refusal diagnostics: %+v", check)
	}
	t.Log("real seed refused different journal; installed tree kept; original baseline and identity unchanged; check reports baseline/current drift")
}
