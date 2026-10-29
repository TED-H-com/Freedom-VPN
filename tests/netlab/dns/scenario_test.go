//go:build netlab

package publishlab_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspolicy"
	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	nodedns "github.com/TED-H-com/Freedom-VPN/node/dns"
	"golang.org/x/net/dns/dnsmessage"
)

// Images are the dev-stack pins in deploy/dev/adguard. The lab does not retag them.
const (
	coreImage = "docker.io/coredns/coredns:1.12.0@sha256:40384aa1f5ea6bfdc77997d243aec73da05f27aed0c5e9d65bfa98933c519d97"
	aghImage  = "docker.io/adguard/adguardhome:v0.107.59@sha256:5d5e3aef39a8c8906a324f0c2da0a6123929e459b57c1e3f05e401dca5c7bf02"
)

var prefixPattern = regexp.MustCompile(`^fvpn-[a-z0-9][a-z0-9-]{0,40}$`)

// Test12_5_LabPublishAndWithdraw applies one document to two resolvers, checks a
// blocked name and an allowed name, withdraws, and checks that neither name
// still resolves (§12.4, §12.5, §12.7, AT12).
//
// The containers are unprivileged and attached to a lab bridge named with
// FVPN_LAB_PREFIX. --network none cannot be used: AdGuard Home has to reach the
// lab CoreDNS, and the test has to query the published DNS ports. Nothing is
// published except 127.0.0.1, and the upstream zone does not recurse.
func Test12_5_LabPublishAndWithdraw(t *testing.T) {
	if os.Getenv("FVPN_NETLAB") != "1" {
		t.Skip("set FVPN_NETLAB=1 to run the DNS publish lab")
	}
	prefix := os.Getenv("FVPN_LAB_PREFIX")
	if !prefixPattern.MatchString(prefix) {
		t.Fatal("FVPN_LAB_PREFIX must match ^fvpn-[a-z0-9][a-z0-9-]{0,40}$")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Fatal("docker is required for the DNS publish lab")
	}

	root := moduleRoot(t)
	state := filepath.Join(os.TempDir(), prefix+"-state")
	cleanupLab(prefix, state)
	t.Cleanup(func() { cleanupLab(prefix, state) })

	upstreamHost := prefix + "-upstream"
	cmd := exec.Command("go", "run", "./internal/dnspolicy/cmd/bootstrap",
		"--state-dir", state, "--upstream", upstreamHost+":5353")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap: %v\n%s", err, out)
	}

	docker(t, "network", "create", "--driver", "bridge", prefix+"-net")
	docker(t, "run", "-d", "--name", upstreamHost, "--network", prefix+"-net",
		"-v", filepath.Join(root, "deploy/dev/adguard/coredns")+":/etc/coredns:ro",
		coreImage, "-conf", "/etc/coredns/Corefile", "-dns.port", "5353")

	user := strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	startResolver(t, prefix, "resolver1", user, state, "18181", "15451")
	startResolver(t, prefix, "resolver2", user, state, "18182", "15452")
	for _, name := range []string{prefix + "-resolver1", prefix + "-resolver2", upstreamHost} {
		assertUnprivileged(t, name)
	}

	creds := readCreds(t, state)
	waitReady(t, "http://127.0.0.1:18181", creds)
	waitReady(t, "http://127.0.0.1:18182", creds)

	compiled, err := dnspublish.Compile(dnspublish.Input{
		ProfileCode: "standard", BlockResponse: "nxdomain", BlockedTTLSeconds: 60,
		Upstreams:    []string{upstreamHost + ":5353"},
		BootstrapDNS: []string{"127.0.0.11:53"},
		Rules: []dnspublish.Rule{
			{RuleID: "01900000-0000-7000-8000-0000000000b1", Domain: "blocked.example.com", IncludeSubdomains: true, Action: "block", Reason: "admin_rule"},
			{RuleID: "01900000-0000-7000-8000-0000000000a1", Domain: "safe-target.example.com", IncludeSubdomains: false, Action: "allow", Reason: "false_positive"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ap, err := nodedns.New([]nodedns.Target{
		{ID: "resolver1", AdminURL: "http://127.0.0.1:18181", Username: creds.Username, Password: creds.Password, DNSAddr: "127.0.0.1:15451"},
		{ID: "resolver2", AdminURL: "http://127.0.0.1:18182", Username: creds.Username, Password: creds.Password, DNSAddr: "127.0.0.1:15452"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := ap.Apply(context.Background(), "1", compiled.Document)
	if err != nil || !ev.Verified || ev.DocumentHash != compiled.Hash {
		t.Fatalf("apply evidence %+v err %v", ev, err)
	}

	addrs := []string{"127.0.0.1:15451", "127.0.0.1:15452"}
	for _, addr := range addrs {
		expectNXDOMAIN(t, addr, "blocked.example.com")
		expectNXDOMAIN(t, addr, "sub.blocked.example.com")
		expectIP(t, addr, "safe-target.example.com", "192.0.2.15")
	}
	for _, base := range []string{"http://127.0.0.1:18181", "http://127.0.0.1:18182"} {
		assertPrivacy(t, base, creds)
		assertUpstream(t, base, creds, upstreamHost+":5353")
	}
	assertNoQueryFiles(t, state)

	withdrawn, err := ap.Withdraw(context.Background(), "1", compiled.Hash)
	if err != nil || withdrawn.Verified || withdrawn.DNSOK() {
		t.Fatalf("withdraw evidence %+v err %v", withdrawn, err)
	}
	for _, addr := range addrs {
		expectNoAddress(t, addr, "blocked.example.com")
		expectNoAddress(t, addr, "safe-target.example.com")
	}
	assertNoQueryFiles(t, state)
}

func startResolver(t *testing.T, prefix, which, user, state, httpPort, dnsPort string) {
	t.Helper()
	docker(t, "run", "-d",
		"--name", prefix+"-"+which,
		"--network", prefix+"-net",
		"--user", user,
		"-p", "127.0.0.1:"+httpPort+":8080/tcp",
		"-p", "127.0.0.1:"+dnsPort+":5353/udp",
		"-p", "127.0.0.1:"+dnsPort+":5353/tcp",
		"-v", filepath.Join(state, which, "conf")+":/opt/adguardhome/conf",
		"-v", filepath.Join(state, which, "work")+":/opt/adguardhome/work",
		aghImage,
		"--no-check-update", "--no-permcheck",
		"-c", "/opt/adguardhome/conf/AdGuardHome.yaml",
		"-w", "/opt/adguardhome/work",
	)
}

func assertUnprivileged(t *testing.T, name string) {
	t.Helper()
	priv := strings.TrimSpace(string(docker(t, "inspect", "-f", "{{.HostConfig.Privileged}}", name)))
	netMode := strings.TrimSpace(string(docker(t, "inspect", "-f", "{{.HostConfig.NetworkMode}}", name)))
	if priv != "false" || netMode == "host" || netMode == "none" {
		t.Fatalf("%s privileged=%s network=%s", name, priv, netMode)
	}
}

type labCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func readCreds(t *testing.T, state string) labCreds {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(state, "credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	var c labCreds
	if err := json.Unmarshal(raw, &c); err != nil || c.Username == "" || c.Password == "" {
		t.Fatal("resolver credentials were not written")
	}
	return c
}

func waitReady(t *testing.T, base string, c labCreds) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, base+"/control/status", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.SetBasicAuth(c.Username, c.Password)
		resp, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			last = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("resolver %s did not become ready: %v", base, last)
}

func expectNXDOMAIN(t *testing.T, addr, name string) {
	t.Helper()
	for _, proto := range []string{"udp", "tcp"} {
		resp, err := query(proto, addr, name)
		if err != nil || resp == nil || !resp.IsNXDOMAIN() {
			rcode := ""
			if resp != nil {
				rcode = resp.RCode.String()
			}
			t.Fatalf("%s %s blocked probe rcode %s err %v", proto, addr, rcode, err)
		}
	}
}

func expectIP(t *testing.T, addr, name, ip string) {
	t.Helper()
	for _, proto := range []string{"udp", "tcp"} {
		resp, err := query(proto, addr, name)
		if err != nil || resp == nil || !resp.IsSuccess() || !hasIP(resp, ip) {
			t.Fatalf("%s %s allowed probe was not %s (err %v)", proto, addr, ip, err)
		}
	}
}

func expectNoAddress(t *testing.T, addr, name string) {
	t.Helper()
	for _, proto := range []string{"udp", "tcp"} {
		resp, err := query(proto, addr, name)
		if err == nil && resp != nil && resp.IsSuccess() && hasAddress(resp) {
			t.Fatalf("%s %s still returned an address after withdraw", proto, addr)
		}
	}
}

func query(proto, addr, name string) (*dnspolicy.DNSResponse, error) {
	return dnspolicy.NewDNSClient(3*time.Second).QueryProto(context.Background(), proto, addr, name, dnsmessage.TypeA)
}

func hasIP(resp *dnspolicy.DNSResponse, ip string) bool {
	for _, ans := range resp.Answers {
		if ans.IP.String() == ip {
			return true
		}
	}
	return false
}

func hasAddress(resp *dnspolicy.DNSResponse) bool {
	for _, ans := range resp.Answers {
		if (ans.Type == dnsmessage.TypeA || ans.Type == dnsmessage.TypeAAAA) && ans.IP.IsValid() {
			return true
		}
	}
	return false
}

func assertPrivacy(t *testing.T, base string, c labCreds) {
	t.Helper()
	ql := getJSON(t, base+"/control/querylog/config", c)
	stats := getJSON(t, base+"/control/stats/config", c)
	if ql["enabled"] != false || stats["enabled"] != false {
		t.Fatal("query log or statistics is enabled")
	}
	if file, ok := ql["file_enabled"]; ok && file != false {
		t.Fatal("query log file is enabled")
	}
	logBody := getBody(t, base+"/control/querylog", c)
	statsBody := getBody(t, base+"/control/stats", c)
	for _, body := range []string{logBody, statsBody} {
		if strings.Contains(body, "blocked.example.com") || strings.Contains(body, "safe-target.example.com") {
			t.Fatal("resolver API returned a probe name")
		}
	}
}

func assertUpstream(t *testing.T, base string, c labCreds, upstream string) {
	t.Helper()
	info := getJSON(t, base+"/control/dns_info", c)
	raw, _ := json.Marshal(info["upstream_dns"])
	if !strings.Contains(string(raw), upstream) {
		t.Fatalf("upstream was not applied at %s", base)
	}
	fallback, _ := json.Marshal(info["fallback_dns"])
	if strings.Contains(string(fallback), "dns.google") || strings.Contains(string(fallback), "1.1.1.1") || strings.Contains(string(fallback), "8.8.8.8") {
		t.Fatal("fallback DNS is set")
	}
}

func assertNoQueryFiles(t *testing.T, state string) {
	t.Helper()
	for _, which := range []string{"resolver1", "resolver2"} {
		work := filepath.Join(state, which, "work")
		_ = filepath.WalkDir(work, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := strings.ToLower(d.Name())
			if !strings.Contains(name, "querylog") && !strings.Contains(name, "stats") {
				return nil
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if strings.Contains(string(body), "blocked.example.com") || strings.Contains(string(body), "safe-target.example.com") {
				t.Fatalf("probe name stored in %s", d.Name())
			}
			return nil
		})
	}
}

func getJSON(t *testing.T, url string, c labCreds) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(getBody(t, url, c)), &out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
	return out
}

func getBody(t *testing.T, url string, c labCreds) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return string(body)
}

func docker(t *testing.T, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func cleanupLab(prefix, state string) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	for _, name := range []string{prefix + "-resolver1", prefix + "-resolver2", prefix + "-upstream"} {
		_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
	}
	_ = exec.CommandContext(ctx, "docker", "network", "rm", prefix+"-net").Run()
	_ = os.RemoveAll(state)
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
