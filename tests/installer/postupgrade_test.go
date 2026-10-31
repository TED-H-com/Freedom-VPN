//go:build installer

package installertest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/packaging/installer"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func assertNoBarrier(t *testing.T, c string) {
	t.Helper()
	for _, path := range []string{"/var/lib/vpn-node/state/meter.checkpoint", "/var/lib/vpn-node/update-journal/forwarding.blocked"} {
		mustExec(t, c, "test", "!", "-e", path)
	}
}

func assertCorrelatedReceipt(t *testing.T, c, job string) {
	t.Helper()
	var cp struct {
		updater.UsageCheckpoint
		BarrierID string `json:"barrier_id"`
	}
	raw := mustExec(t, c, "cat", "/var/lib/vpn-node/state/meter.checkpoint")
	if err := json.Unmarshal([]byte(raw), &cp); err != nil {
		t.Fatal(err)
	}
	latch := strings.TrimSpace(mustExec(t, c, "cat", "/var/lib/vpn-node/update-journal/forwarding.blocked"))
	if uuid.Validate(latch) != nil || cp.BarrierID != latch || cp.NewMeterEpochID != "" {
		t.Fatalf("checkpoint not correlated to the real updater latch: %s", raw)
	}
	var receipt struct {
		State   string                   `json:"state"`
		Hash    string                   `json:"payload_hash"`
		Usage   *updater.UsageCheckpoint `json:"usage_checkpoint"`
		Forward bool                     `json:"member_forwarding_enabled"`
	}
	sealed := mustExec(t, c, "/usr/local/bin/fvpn-w5-control", "--upgrade-receipt", job)
	if err := json.Unmarshal([]byte(sealed), &receipt); err != nil {
		t.Fatal(err)
	}
	hash, err := canonjson.HashHex([]byte(sealed), "payload_hash")
	if err != nil || hash != receipt.Hash || receipt.State != "succeeded" || receipt.Forward || receipt.Usage == nil {
		t.Fatalf("missing authentic successful unpublished receipt: %s (%v)", sealed, err)
	}
	u := receipt.Usage
	if u.ClosedMeterEpochID != cp.ClosedMeterEpochID || u.LastDurableEventSeq != cp.LastDurableEventSeq || u.CentralAckedThrough != cp.CentralAckedThrough || uuid.Validate(u.NewMeterEpochID) != nil || u.NewMeterEpochID == u.ClosedMeterEpochID {
		t.Fatalf("receipt lost post-block accounting: checkpoint=%s receipt=%s", raw, sealed)
	}
	var ledger struct {
		Last   decstr.U64 `json:"last_event_seq"`
		Acked  decstr.U64 `json:"acked_through"`
		Epochs []struct {
			ID    string `json:"meter_epoch_id"`
			State string `json:"state"`
		} `json:"meter_epochs"`
	}
	snapshot := mustExec(t, c, "/opt/vpn-node/current/fvpn-agent", "usage", "snapshot", "-config", "/etc/vpn-node/agent.json", "-json")
	if err := json.Unmarshal([]byte(snapshot), &ledger); err != nil || ledger.Last != cp.LastDurableEventSeq || ledger.Acked < cp.CentralAckedThrough || ledger.Acked > ledger.Last {
		t.Fatalf("checkpoint watermarks differ from durable ledger: %s (%v)", snapshot, err)
	}
	closed := false
	for _, epoch := range ledger.Epochs {
		closed = closed || (epoch.ID == cp.ClosedMeterEpochID && epoch.State == "closed")
	}
	if !closed {
		t.Fatalf("receipt epoch not durably closed: %s", snapshot)
	}
	t.Logf("D-125: absent pre-block checkpoint -> real agent barrier_id byte-equal to updater latch; sealed succeeded receipt matches closed epoch and post-block durable/ACK %s/%s; distinct new-epoch note; current switched", cp.LastDurableEventSeq, cp.CentralAckedThrough)
}

// The public sequence from devstack 18: seed -> stage -> apply/switch ->
// revoke -> reenroll/seed. This remains an isolated fake-control test; the
// serving campaign and same-node central reenrollment belong to devstack.
func assertPostUpgradeReenroll(t *testing.T, c, url, ca string) {
	t.Helper()
	mustExec(t, c, "systemctl", "stop", "fvpn-updater.timer", "fvpn-agent", "fvpn-watchdog")
	const root = "/opt/vpn-node/releases/"
	trees := func() string {
		return mustExec(t, c, "sh", "-c", `find "$1" -printf '%P %y %m %s %T@\n' | sort; find "$1" -type f -exec sha256sum {} + | sort`, "sh", root)
	}
	beforeTrees := trees()
	mustExec(t, c, "test", "-d", root+rel1.ReleaseID)
	usage := sha(t, c, "/var/lib/vpn-node/state/usage.db")
	inc := incarnation(t, c)
	identity := map[string]string{}
	for _, name := range []string{"key.pem", "cert.pem", "node.json"} {
		identity[name] = sha(t, c, "/var/lib/vpn-node/identity/"+name)
	}
	admin := strings.TrimSpace(mustExec(t, c, "cat", "/run/w5-control/admin"))
	mustExec(t, c, "curl", "--fail", "--silent", "--show-error", "-X", "POST", admin+"/revoke")
	assertRevoked := func(dir string) {
		t.Helper()
		out := mustExec(t, c, "curl", "--silent", "--show-error", "--cacert", ca,
			"--cert", dir+"/cert.pem", "--key", dir+"/key.pem", "--output", "/run/w5-revoked-response", "--write-out", "%{http_code}", url+"/api/v1/agent/desired-state")
		if out != "403" || !strings.Contains(mustExec(t, c, "cat", "/run/w5-revoked-response"), "IDENTITY_REVOKED") {
			t.Fatalf("old certificate was not revoked: HTTP %s", out)
		}
	}
	assertRevoked("/var/lib/vpn-node/identity")
	writeStdin(t, c, "/run/w5-token2", token2+"\n", "0600")
	out, code := dexec(t, c, "/usr/local/bin/fvpn-node-install", "reenroll", "--json", "--token-file", "/run/w5-token2")
	if code != 0 {
		// Preserve per-entry bytes/modes before the container is removed. The
		// signed reference is extracted privately, never over the live tree.
		mustExec(t, c, "mkdir", "/var/tmp/w5/seed-reference")
		mustExec(t, c, "tar", "--zstd", "--extract", "--same-permissions", "--file", relIn("2", rel2.ReleaseID).Artifact, "--directory", "/var/tmp/w5/seed-reference")
		t.Logf("resolved-current/previous inventory:\n%s", trees())
		t.Log(mustExec(t, c, "sh", "-c", `cd /var/tmp/w5/seed-reference && find . -printf '%P %y %m %s\n' | sort; find . -type f -exec sha256sum {} + | sort`))
		t.Fatalf("post-upgrade reenroll: exit=%d %s", code, out)
	}
	var result installer.Result
	if err := decodeJSON(out, &result); err != nil || result.Status != "reenrolled" {
		t.Fatalf("reenroll result: %s %v", out, err)
	}
	assertBaseline(t, result.Baseline, out)
	if result.Baseline.TreePath != root+rel2.ReleaseID || result.Baseline.ManifestPath != "/var/lib/vpn-node/update-staging/"+rel2.ReleaseID+".manifest.json" {
		t.Fatalf("seed used wrong tree/envelope: %+v", result.Baseline)
	}
	if beforeTrees != trees() || strings.TrimSpace(mustExec(t, c, "readlink", "/opt/vpn-node/current")) != "releases/"+rel2.ReleaseID {
		t.Fatal("reenroll/seed mutated release bytes, modes, mtimes or current")
	}
	if usage != sha(t, c, "/var/lib/vpn-node/state/usage.db") || incarnation(t, c) == inc || sha(t, c, "/var/lib/vpn-node/identity/key.pem") == identity["key.pem"] {
		t.Fatal("reenroll did not preserve ledger and replace identity")
	}
	retired := strings.TrimSpace(mustExec(t, c, "find", "/var/lib/vpn-node/identity.retired", "-mindepth", "1", "-maxdepth", "1", "-type", "d"))
	if strings.TrimSpace(mustExec(t, c, "stat", "-c", "%a", retired)) != "700" {
		t.Fatal("retired identity is not private")
	}
	for name, hash := range identity {
		if sha(t, c, retired+"/"+name) != hash {
			t.Fatalf("retired %s changed", name)
		}
	}
	assertRevoked(retired)
	mustExec(t, c, "test", "!", "-e", "/run/w5-token2")
	check := mustCheck(t, c, 0)
	if check.ReleaseID != rel2.ReleaseID || check.Baseline.Status != "seeded" {
		t.Fatalf("upgraded reenrollment check: %+v", check)
	}
	t.Log("F9-02/devstack18: apply umask 022 -> real revoke (403 IDENTITY_REVOKED) -> installer reenroll/seed umask 077 exit 0; candidate and previous tree bytes/modes/mtimes unchanged; ledger hash preserved; retired key/cert/node hashes preserved at 0700; check healthy")
}
