package isolation

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestD141_RuntimeProjectionAndBoundedLogs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.log")
	if got := runtimeRead(path); got.Status != "unavailable" {
		t.Fatal(got)
	}
	if err := os.WriteFile(path, []byte("before shutdown\n"), 0600); err != nil {
		t.Fatal(err)
	}
	before := runtimeRead(path)
	if err := os.WriteFile(path, []byte("new launch\n"), 0600); err != nil {
		t.Fatal(err)
	}
	after := runtimeRead(path)
	if before.Status != "available" || before.Text != "before shutdown\n" || after.Text != "new launch\n" || before.SHA256 == after.SHA256 {
		t.Fatal("independent snapshots lost pre-stop bytes")
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", runtimeReadLimit+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if got := runtimeRead(path); got.Status != "truncated" || got.Bytes != runtimeReadLimit {
		t.Fatal(got.Status, got.Bytes)
	}
	cfg := RuntimeRead{Status: "available", Text: `{"control_url":"https://127.0.0.1:9444","observation_seconds":1,"heartbeat_seconds":1,"poll_seconds":1,"datapath":{"member_ingress":"fvmb0","uplink":"fvup0","snat":"198.51.100.23"},"private":"TEST ONLY excluded"}`}
	projected := projectRuntimeConfig("/etc/vpn-node/agent-run.json", cfg)
	raw, _ := json.Marshal(projected)
	if strings.Contains(string(raw), "excluded") || projected["status"] != "available" {
		t.Fatal(string(raw))
	}
	cfg.Text = `{"datapath":`
	if got := projectRuntimeConfig("/etc/vpn-node/agent-run.json", cfg); got["status"] != "parse error" {
		t.Fatal(got)
	}
	home := "/" + "home/" + "alice/secret"
	unapproved := projectRuntimeConfig(home+"/agent-run.json", cfg)
	if unapproved["status"] != "unapproved path" || strings.Contains(unapproved["path"].(string), "alice") {
		t.Fatal(unapproved)
	}
	if err := os.WriteFile(path, []byte("path="+home+" setup_key=nbsecret \"token\":\"jwtsecret\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	redacted := runtimeRead(path)
	if strings.Contains(redacted.Text, "alice") || strings.Contains(redacted.Text, "nbsecret") || strings.Contains(redacted.Text, "jwtsecret") {
		t.Fatal(redacted.Text)
	}
}

func TestD141_RuntimeArgumentsAreAllowlisted(t *testing.T) {
	for _, tc := range []struct{ args, role string }{
		{"/usr/local/bin/fvpn-agent\x00run\x00-config\x00/etc/vpn-node/agent-run.json\x00", "fvpn-agent"},
		{"/usr/local/bin/fvpn-watchdog\x00run\x00-config\x00/etc/vpn-node/agent-run.json\x00", "fvpn-watchdog"},
		{"/usr/local/bin/fvpn-e2e-nodeconfig\x00resolver-proxy\x00", "resolver-proxy"},
		{"/usr/local/bin/fvpn-agent\x00enroll\x00-token\x00TEST ONLY excluded\x00", ""},
		{"/usr/local/bin/fvpn-agent\x00run\x00-config\x00/unapproved\x00", ""},
		{"/usr/local/bin/fvpn-agent\x00run\x00-config\x00/etc/vpn-node/agent-run.json\x00-extra\x00", ""},
	} {
		role, _, _ := approvedProcess([]byte(tc.args))
		if role != tc.role {
			t.Fatalf("role %q expected %q", role, tc.role)
		}
	}
}

func TestD141_CloneTransportAndStopReview(t *testing.T) {
	cmd := exec.Command("python3", "-B", "-m", "unittest", "-v", "clone_evidence_test.py")
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")
	out, err := cmd.CombinedOutput()
	t.Log(string(out))
	if err != nil {
		t.Fatal(err)
	}
}
