//go:build netlab

// Package agentlab runs fvpn-agent and fvpn-watchdog inside the netlab
// container against fvpn-labcontrol. The package name is not "agent" so it
// can import nothing from node/agent's tests and does not collide with that
// import path. Helpers here are copied in miniature from tests/netlab; the
// existing lab files stay unchanged.
package agentlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	// TEST ONLY values. They are not product secrets.
	labToken    = "test-only-enrollment-token"
	labSetupKey = "test-only-setup-key"
	labDir      = "/scratch/w2"
)

func TestMain(m *testing.M) {
	if os.Getenv("FVPN_NETLAB") != "1" {
		os.Exit(m.Run())
	}
	if err := prepareLab(); err != nil {
		fmt.Fprintln(os.Stderr, "agentlab:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func prefix() string {
	if v := os.Getenv("FVPN_LAB_PREFIX"); v != "" {
		return v
	}
	return "fvpn-w2-agent-core-lab"
}

func skipLab(t *testing.T) {
	t.Helper()
	if os.Getenv("FVPN_NETLAB") != "1" {
		t.Skip("netlab: set FVPN_NETLAB=1; the container must already be up")
	}
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", prefix()).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		t.Fatalf("netlab: container %s is not running: %s", prefix(), strings.TrimSpace(string(out)))
	}
}

func prepareLab() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return err
	}
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", prefix()).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "true" {
		return fmt.Errorf("container %s is not running", prefix())
	}
	_, _, _ = dockerExec("sh", "-c", "mkdir -p "+labDir+"/state "+labDir+"/identity "+labDir+"/journal")
	if err := dockerDetach("sh", "-c", "exec fvpn-labcontrol -dir "+labDir+" -admin 127.0.0.1:9091 >"+labDir+"/labcontrol.log 2>&1"); err != nil {
		return err
	}
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, _, err := dockerExec("test", "-f", labDir+"/ready"); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, _, err := dockerExec("test", "-f", labDir+"/ready"); err != nil {
		return fmt.Errorf("labcontrol did not become ready: %s", tailLog())
	}
	urlBytes, errOut, err := dockerExec("cat", labDir+"/url")
	if err != nil {
		return fmt.Errorf("control url: %v %s", err, errOut)
	}
	if err := writeAgentConfig(strings.TrimSpace(urlBytes), 3, 1); err != nil {
		return err
	}
	if out, errOut, err := dockerExec("fvpn-agent", "enroll", "-config", labDir+"/agent.json", "-token-file", labDir+"/token"); err != nil {
		return fmt.Errorf("enroll: %v\n%s%s", err, redact(out), redact(errOut))
	}
	if _, _, err := dockerExec("test", "!", "-f", labDir+"/token"); err != nil {
		return fmt.Errorf("enrollment token file was not removed")
	}
	keyBefore, errOut, err := dockerExec("sha256sum", labDir+"/identity/key.pem")
	if err != nil || strings.TrimSpace(keyBefore) == "" {
		return fmt.Errorf("identity key missing after enroll: %v %s", err, errOut)
	}
	if _, errOut, err := dockerExecStdin([]string{"tee", labDir + "/token2"}, []byte("other-token\n")); err != nil {
		return fmt.Errorf("token2: %v %s", err, errOut)
	}
	if out, errOut, err := dockerExec("fvpn-agent", "enroll", "-config", labDir+"/agent.json", "-token-file", labDir+"/token2"); err != nil {
		return fmt.Errorf("second enroll: %v\n%s%s", err, redact(out), redact(errOut))
	}
	keyAfter, _, _ := dockerExec("sha256sum", labDir+"/identity/key.pem")
	if keyBefore != keyAfter {
		return fmt.Errorf("second enroll changed the identity key")
	}
	return nil
}

func writeAgentConfig(controlURL string, stale, interval int) error {
	cfg := map[string]any{
		"state_dir":                 labDir + "/state",
		"identity_dir":              labDir + "/identity",
		"journal_dir":               labDir + "/journal",
		"lock_path":                 labDir + "/update.lock",
		"control_url":               controlURL,
		"server_ca_file":            labDir + "/ca.pem",
		"trust_bundle":              labDir + "/trust.json",
		"heartbeat_path":            labDir + "/heartbeat",
		"datapath":                  map[string]string{"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release":             "0.0.0-test",
		"release_id":                "019a0000-0000-7000-8000-0000000000c1",
		"renew_lead_seconds":        1,
		"heartbeat_seconds":         1,
		"watchdog_stale_seconds":    stale,
		"watchdog_interval_seconds": interval,
		"poll_seconds":              1,
		"observation_seconds":       1,
		"test_mode":                 true,
		"test_exempt_v4":            []string{"203.0.113.0/24"},
		"extra_denied_v4":           []string{"192.0.2.200/32"},
		"host_v4":                   []string{"192.0.2.33/32", "192.0.2.37/32", "192.168.0.1/32"},
		"host_reserve_bps":          5000000,
		"netbird_backend":           "kernel",
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	_, errOut, err := dockerExecStdin([]string{"tee", labDir + "/agent.json"}, append(raw, '\n'))
	if err != nil {
		return fmt.Errorf("agent.json: %v %s", err, errOut)
	}
	return nil
}

func dockerExec(args ...string) (string, string, error) {
	cmd := exec.Command("docker", append([]string{"exec", prefix()}, args...)...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func dockerExecStdin(args []string, stdin []byte) (string, string, error) {
	cmd := exec.Command("docker", append([]string{"exec", "-i", prefix()}, args...)...)
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func dockerDetach(args ...string) error {
	cmd := exec.Command("docker", append([]string{"exec", "-d", prefix()}, args...)...)
	var errb bytes.Buffer
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, errb.String())
	}
	return nil
}

func mustExec(t *testing.T, args ...string) string {
	t.Helper()
	out, errOut, err := dockerExec(args...)
	if err != nil {
		t.Fatalf("docker exec %s: %v\n%s%s", strings.Join(args, " "), err, redact(out), redact(errOut))
	}
	return out
}

func writeScratch(t *testing.T, path string, body []byte) {
	t.Helper()
	out, errOut, err := dockerExecStdin([]string{"tee", path}, body)
	if err != nil {
		t.Fatalf("tee %s: %v\n%s%s", path, err, out, errOut)
	}
}

func readScratch(t *testing.T, path string) string {
	t.Helper()
	out, errOut, err := dockerExec("cat", path)
	if err != nil {
		t.Fatalf("cat %s: %v %s", path, err, errOut)
	}
	return out
}

func scratchExists(path string) bool {
	_, _, err := dockerExec("test", "-f", path)
	return err == nil
}

func tailLog() string {
	out, errOut, err := dockerExec("sh", "-c", "tail -n 60 "+labDir+"/agent.log "+labDir+"/labcontrol.log 2>/dev/null || true")
	if err != nil {
		return errOut
	}
	return redact(out)
}

func redact(s string) string {
	s = strings.ReplaceAll(s, labToken, "[redacted]")
	s = strings.ReplaceAll(s, labSetupKey, "[redacted]")
	return s
}

func assertNoSecrets(t *testing.T) {
	t.Helper()
	if !scratchExists(labDir + "/agent.log") {
		return
	}
	body := readScratch(t, labDir+"/agent.log")
	if strings.Contains(body, labToken) || strings.Contains(body, labSetupKey) {
		t.Fatal("agent log contains an enrollment token or setup key")
	}
}

type probeCase struct {
	Name      string `json:"name"`
	NS        string `json:"ns"`
	Proto     string `json:"proto"`
	Dst       string `json:"dst,omitempty"`
	Port      int    `json:"port,omitempty"`
	Bind      string `json:"bind,omitempty"`
	Expect    string `json:"expect"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
	Src       string `json:"src,omitempty"`
	SrcMAC    string `json:"src_mac,omitempty"`
	DstMAC    string `json:"dst_mac,omitempty"`
	Iface     string `json:"iface,omitempty"`
	Size      int    `json:"size,omitempty"`
}

type probeResult struct {
	Name     string `json:"name"`
	Expect   string `json:"expect"`
	Observed string `json:"observed"`
	OK       bool   `json:"ok"`
}

func inetCase(expect string) probeCase {
	return probeCase{Name: "inet", NS: "fv-a1", Proto: "tcp", Dst: "203.0.113.10", Port: 80, Expect: expect, TimeoutMS: 800}
}

func runProbe(t *testing.T, cases []probeCase) []probeResult {
	t.Helper()
	body, err := json.Marshal(map[string]any{"cases": cases})
	if err != nil {
		t.Fatal(err)
	}
	writeScratch(t, labDir+"/cases.json", body)
	out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/probe.py", labDir+"/cases.json")
	if err != nil && !strings.Contains(err.Error(), "exit status 2") {
		t.Fatalf("probe: %v\n%s%s", err, redact(out), redact(errOut))
	}
	var doc struct {
		Results []probeResult `json:"results"`
	}
	if jerr := json.Unmarshal([]byte(out), &doc); jerr != nil {
		t.Fatalf("probe json: %v\n%s", jerr, redact(out))
	}
	return doc.Results
}

// probeOK reports whether the Internet probe matched expect. expect is
// "allow" or "deny". probe.py stores the socket result in Observed
// ("connected", "timeout", ...) and sets OK from that expectation.
func probeOK(t *testing.T, expect string) (bool, string) {
	t.Helper()
	res := runProbe(t, []probeCase{inetCase(expect)})
	if len(res) != 1 {
		return false, ""
	}
	return res[0].OK, res[0].Observed
}

func waitProbe(t *testing.T, expect string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var obs string
	for time.Now().Before(deadline) {
		ok, got := probeOK(t, expect)
		obs = got
		if ok {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Logf("last probe expect %s observed %s", expect, obs)
	return false
}

func admin(t *testing.T, path, jsonBody string) {
	t.Helper()
	data := "b''"
	if jsonBody != "" {
		data = fmt.Sprintf("b%s", strconvQuote(jsonBody))
	}
	script := fmt.Sprintf(`import urllib.request
req = urllib.request.Request("http://127.0.0.1:9091%s", data=%s, method="POST")
urllib.request.urlopen(req, timeout=5).read()
`, path, data)
	writeScratch(t, labDir+"/admin.py", []byte(script))
	mustExec(t, "python3", labDir+"/admin.py")
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func stopProc(name, pidFile string) {
	_, _, _ = dockerExec("sh", "-c", fmt.Sprintf(
		"if [ -f %s ]; then kill $(cat %s) 2>/dev/null || true; sleep 0.2; kill -9 $(cat %s) 2>/dev/null || true; rm -f %s; fi; pkill -x %s 2>/dev/null || true",
		pidFile, pidFile, pidFile, pidFile, name,
	))
}

func startProc(t *testing.T, bin, pidFile, logFile string) {
	t.Helper()
	stopProc(bin, pidFile)
	script := fmt.Sprintf("echo $$ > %s; exec %s run -config %s/agent.json >>%s 2>&1", pidFile, bin, labDir, logFile)
	if err := dockerDetach("sh", "-c", script); err != nil {
		t.Fatal(err)
	}
}

func startAgent(t *testing.T) {
	t.Helper()
	startProc(t, "fvpn-agent", labDir+"/agent.pid", labDir+"/agent.log")
}
func startWatch(t *testing.T) {
	t.Helper()
	startProc(t, "fvpn-watchdog", labDir+"/watchdog.pid", labDir+"/watchdog.log")
}
func stopAgent() { stopProc("fvpn-agent", labDir+"/agent.pid") }
func stopWatch() { stopProc("fvpn-watchdog", labDir+"/watchdog.pid") }

func flushConntrack(t *testing.T) {
	t.Helper()
	_, _, _ = dockerExec("conntrack", "-F")
	_, _, _ = dockerExec("conntrack", "-F", "-f", "ipv6")
}

func debugPlane(t *testing.T) string {
	t.Helper()
	out, _, _ := dockerExec("nft", "list", "table", "inet", "fvpn")
	return redact(out + "\n" + tailLog())
}
