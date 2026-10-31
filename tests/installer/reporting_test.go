//go:build installer

package installertest

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// The installed updater loads the original rendered config and posts its own
// sealed result. No endpoint, CA, origin or TLS-client override is injected.
func assertGeneratedReporting(t *testing.T, c, job string) {
	t.Helper()
	const config = "/etc/vpn-node/updater.json"
	const updater = "/opt/vpn-node/current/fvpn-updater"
	const serverCA = "/etc/vpn-node/ca.pem"
	configHash := sha(t, c, config)
	var seal api.UpdateResult
	sealed := mustExec(t, c, "/usr/local/bin/fvpn-w5-control", "--upgrade-receipt", job)
	if err := json.Unmarshal([]byte(sealed), &seal); err != nil {
		t.Fatal(err)
	}
	accepted := "/run/w5-control/results/" + seal.ResultId.String()
	// A missing trust file must fail engine construction, not disable reporting.
	mustExec(t, c, "mv", serverCA, serverCA+".saved")
	out, code := dexec(t, c, updater, "flush", "--config", config)
	if code != 1 || !strings.Contains(out, "mtls ca") {
		t.Fatalf("missing CA: %d %s", code, out)
	}
	mustExec(t, c, "test", "!", "-e", accepted+".json")
	// The enrolled node issuer is not the HTTPS server issuer. This would fail
	// with the old renderer even if its result_url alone had been corrected.
	mustExec(t, c, "cp", "/var/lib/vpn-node/identity/ca.pem", serverCA)
	out, code = dexec(t, c, updater, "flush", "--config", config)
	if code != 1 || !strings.Contains(out, "result submit") {
		t.Fatalf("wrong CA: %d %s", code, out)
	}
	mustExec(t, c, "test", "!", "-e", accepted+".json")
	mustExec(t, c, "mv", serverCA+".saved", serverCA)
	// Test central also requires authenticated node identity at this endpoint.
	url := strings.TrimSpace(mustExec(t, c, "cat", "/run/w5-control/url"))
	status := mustExec(t, c, "curl", "--silent", "--show-error", "--cacert", serverCA, "-X", "POST", "--output", "/dev/null", "--write-out", "%{http_code}", url+"/api/v1/agent/update-results")
	if status != "401" {
		t.Fatalf("unauthenticated result: HTTP %s", status)
	}
	mustExec(t, c, updater, "flush", "--config", config)
	received := mustExec(t, c, "cat", accepted+".json")
	if strings.TrimSpace(received) != strings.TrimSpace(sealed) {
		t.Fatal("central accepted different sealed bytes")
	}
	var ack api.UpdateResultAck
	raw := mustExec(t, c, "cat", accepted+".ack.json")
	if err := json.Unmarshal([]byte(raw), &ack); err != nil || !ack.Accepted || ack.ResultId != seal.ResultId {
		t.Fatalf("central ACK: %s %v", raw, err)
	}
	mustExec(t, c, updater, "flush", "--config", config)
	if raw != mustExec(t, c, "cat", accepted+".ack.json") || configHash != sha(t, c, config) {
		t.Fatal("repeat flush changed receipt or generated config")
	}
	t.Log("F10-02: unchanged installed config -> real HTTPSink mTLS POST -> test central 202/stored ACK; exact sealed bytes and D-110 job checked; repeat flush stable; missing/wrong server CA and absent client identity refused; distinct server and node issuer CAs")
}
