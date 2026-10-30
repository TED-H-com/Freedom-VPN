//go:build netlab

package netlab

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// TestForeign_TableBeforeFvpn proves the follow-up 10 tolerance rule on a
// live ruleset. An earlier accept does not stop the inet fvpn drop. A
// prerouting payload rewrite is in the bytes VerifyServing reads, serving
// is not reported, and the agent leaves the foreign table in place.
func TestForeign_TableBeforeFvpn(t *testing.T) {
	skipLab(t)
	t.Cleanup(func() {
		_, _, _ = dockerExec("nft", "delete", "table", "inet", "fvobs")
	})
	_, _, _ = dockerExec("nft", "delete", "table", "inet", "fvobs")

	now := time.Now().UTC()
	m := functionalModel(now, 81)
	applyAt(t, m)
	flushConntrack(t)

	mustExec(t, "nft", "add", "table", "inet", "fvobs")
	mustExec(t, "nft", "add", "chain", "inet", "fvobs", "early", "{ type filter hook forward priority -200 ; policy accept ; }")
	mustExec(t, "nft", "add", "rule", "inet", "fvobs", "early", "accept")

	if _, stderr, code := runLab(t, m, "verify"); code != 0 {
		t.Fatalf("accept-only verify exit %d\n%s", code, stderr)
	}

	var obs IngressObservation
	uplink := captureDuring(t, "root", "fv-ih", "udp port 39998", func() {
		obs = injectPort(t, "fv-fh", "100.64.9.9", "203.0.113.10", "39998")
	})
	ctunku := counterDelta(obs, "ctunku").Packets
	uplinkN := 0
	if strings.TrimSpace(uplink) != "" {
		uplinkN = 1
	}
	if ctunku < 1 || uplinkN != 0 {
		t.Fatalf("accept-only chain stopped the fvpn drop: ctunku=%d uplink=%q", ctunku, uplink)
	}

	mustExec(t, "nft", "add", "chain", "inet", "fvobs", "pre", "{ type filter hook prerouting priority -200 ; policy accept ; }")
	mustExec(t, "nft", "add", "rule", "inet", "fvobs", "pre", "ip saddr 100.64.9.9 ip saddr set 100.64.1.1")

	_, stderr, code := runLab(t, m, "verify")
	if code != 3 {
		t.Fatalf("mangle verify exit %d, want 3\n%s", code, stderr)
	}
	for _, part := range []string{"ISOLATION_POLICY_INVALID", "table fvobs", "family inet", "hook prerouting", "priority -200", "statement mangle"} {
		if !strings.Contains(stderr, part) {
			t.Fatalf("verdict %q missing %q", stderr, part)
		}
	}

	_, applyErr, applyCode := runLab(t, m, "apply")
	if applyCode != 3 {
		t.Fatalf("mangle apply exit %d, want 3\n%s", applyCode, applyErr)
	}
	if !strings.Contains(applyErr, "ISOLATION_POLICY_INVALID") || !strings.Contains(applyErr, "statement mangle") {
		t.Fatalf("apply verdict: %s", applyErr)
	}
	shape := rulesetShape(t)
	if !strings.Contains(shape, "fvpn_baseline=1") || strings.Contains(shape, "fvpn_generation=81") {
		t.Fatalf("serving ruleset left installed:\n%s", shape)
	}
	if _, errOut, err := dockerExec("nft", "list", "table", "inet", "fvobs"); err != nil {
		t.Fatalf("apply removed inet fvobs: %v\n%s", err, errOut)
	}

	mustExec(t, "nft", "delete", "table", "inet", "fvobs")
	applyAt(t, m)
	if _, stderr, code := runLab(t, m, "verify"); code != 0 {
		t.Fatalf("recovery verify exit %d\n%s", code, stderr)
	}
	if strings.Contains(rulesetShape(t), "fvpn_baseline=1") {
		t.Fatal("recovery left the baseline in place")
	}

	line := "FOREIGN accept_ctunku=" + strconv.FormatUint(ctunku, 10) + " accept_uplink=0 verdict=ISOLATION_POLICY_INVALID mangle_exit=3 table_kept=1 recovered=1"
	t.Log("SUMMARY " + line)
	noteSummary(line)
}

func injectPort(t *testing.T, iface, src, dst, port string) IngressObservation {
	t.Helper()
	out, errOut, err := dockerExec("fvpn-netlab", "inject", "-iface", iface, "-src", src, "-dst", dst, "-port", port)
	if err != nil {
		t.Fatalf("inject: %v\n%s%s", err, out, errOut)
	}
	var obs IngressObservation
	if jerr := json.Unmarshal([]byte(out), &obs); jerr != nil {
		t.Fatalf("inject json: %v\n%s", jerr, out)
	}
	return obs
}

func runLab(t *testing.T, m enforce.Model, cmd string) (stdout, stderr string, code int) {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err = dockerExecStdin([]string{"fvpn-netlab", cmd}, raw)
	if err == nil {
		return stdout, stderr, 0
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("%s: %v\n%s%s", cmd, err, stdout, stderr)
	}
	return stdout, stderr, exit.ExitCode()
}
