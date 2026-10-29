//go:build e2e

package isolation

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type attempt struct {
	name, via, result string
}

func (s *stack) exerciseServing(t *testing.T, nodeIPs []string) exitProbe {
	t.Helper()
	var probe exitProbe
	probe.members = s.memberCount()
	probe.ipv6 = s.ipv6Fact()
	probe.reboot = "not run: entrypoint-node.sh enrolls again from the token file on every start, so a container restart is not a reboot of an enrolled node"
	probe.updater = "not run: this acceptance run does not publish a release batch"
	targets := []struct {
		name, ip string
		port     int
	}{
		{"rfc1918-10", "10.255.255.1", 80},
		{"rfc1918-172", "172.16.255.1", 80},
		{"rfc1918-192", "192.168.255.1", 80},
		{"cgnat", "100.64.255.1", 80},
		{"metadata", "169.254.169.254", 80},
		{"node-dummy", "192.0.2.20", 9},
	}
	for _, ip := range nodeIPs {
		if ip == "" || ip == "192.0.2.20" {
			continue
		}
		targets = append(targets, struct {
			name, ip string
			port     int
		}{"node-" + ip, ip, 22})
	}
	var attempts []attempt
	for _, target := range targets {
		via, result := s.tcpProbe(target.ip, target.port)
		attempts = append(attempts, attempt{target.name, via, result})
		if via != "fvnb0" {
			continue
		}
		probe.protectedVia++
		switch result {
		case "connected", "refused":
			probe.protectedAllows++
			if result == "refused" {
				probe.protectedRefuse++
			}
		}
	}
	if s.originIP != "" {
		via, result := s.tcpProbe(s.originIP, 443)
		probe.internet = via + " " + result
		attempts = append(attempts, attempt{"origin", via, result})
	} else {
		probe.internet = "origin address absent"
	}
	probe.lines = formatAttempts(attempts)
	if probe.protectedAllows > 0 {
		probe.fail = "a protected destination was delivered through fvnb0"
		t.Errorf("%s", probe.fail)
		return probe
	}
	probe.flap, probe.rate = s.flapDuringProbe(t)
	if strings.Contains(probe.flap, "allow") {
		probe.fail = probe.flap
		t.Errorf("%s", probe.fail)
		return probe
	}
	probe.dns = s.dnsFact()
	probe.agentKill = s.killAgent()
	if strings.Contains(probe.agentKill, "allow") {
		probe.fail = probe.agentKill
		t.Errorf("%s", probe.fail)
		return probe
	}
	probe.netbird = s.restartNetbird()
	if strings.Contains(probe.netbird, "allow") {
		probe.fail = probe.netbird
		t.Errorf("%s", probe.fail)
	}
	return probe
}

func (s *stack) tcpProbe(ip string, port int) (via, result string) {
	route, err := s.docker(s.member, "ip", "route", "get", ip)
	text := strings.TrimSpace(string(route))
	if err != nil || !strings.Contains(text, "dev fvnb0") {
		return "not_fvnb0", clip(oneLine(text), 160)
	}
	script := fmt.Sprintf(
		`err=$(timeout 0.4 bash -c 'echo >/dev/tcp/%s/%d' 2>&1 >/dev/null); rc=$?; if [ "$rc" -eq 0 ]; then echo connected; elif printf '%%s' "$err" | grep -qi refused; then echo refused; else echo denied; fi`,
		ip, port,
	)
	out, err := s.docker(s.member, "bash", "-c", script)
	got := strings.TrimSpace(string(out))
	switch {
	case strings.Contains(got, "connected"):
		return "fvnb0", "connected"
	case strings.Contains(got, "refused"):
		return "fvnb0", "refused"
	case err != nil && got == "":
		return "fvnb0", "denied"
	default:
		return "fvnb0", "denied"
	}
}

func (s *stack) flapDuringProbe(t *testing.T) (flap, rate string) {
	t.Helper()
	script := `rm -f /tmp/at10.hits /tmp/at10.count; start=$(date +%s); end=$((start+3)); n=0; while [ "$(date +%s)" -lt "$end" ]; do n=$((n+1)); if timeout 0.05 bash -c 'echo >/dev/tcp/10.255.255.1/80' >/dev/null 2>&1; then echo allow >> /tmp/at10.hits; fi; echo "$n" > /tmp/at10.count; done`
	if err := s.dockerDetach(s.member, "bash", "-c", script); err != nil {
		return "probe loop did not start", "0"
	}
	time.Sleep(400 * time.Millisecond)
	_, _ = s.docker(s.node, "ip", "link", "del", "fvatif")
	if _, err := s.docker(s.node, "ip", "link", "add", "fvatif", "type", "dummy"); err != nil {
		return "dummy interface was not created", "0"
	}
	_, _ = s.docker(s.node, "ip", "addr", "add", "198.18.0.10/32", "dev", "fvatif")
	_, _ = s.docker(s.node, "ip", "link", "set", "fvatif", "up")
	via, result := s.tcpProbe("198.18.0.10", 80)
	_, _ = s.docker(s.node, "ip", "link", "del", "fvatif")
	time.Sleep(3 * time.Second)
	countRaw, _ := s.docker(s.member, "cat", "/tmp/at10.count")
	hitsRaw, _ := s.docker(s.member, "cat", "/tmp/at10.hits")
	count := strings.TrimSpace(string(countRaw))
	if count == "" {
		count = "0"
	}
	flap = "dummy fvatif 198.18.0.10/32 " + via + " " + result
	if via == "fvnb0" && (result == "connected" || result == "refused") {
		flap = "allow during interface change: " + flap
	}
	if strings.TrimSpace(string(hitsRaw)) != "" {
		flap = "allow during continuous probe"
	}
	return flap, count + " attempts over about 3s"
}

func (s *stack) killAgent() string {
	out, err := s.docker(s.node, "pidof", "fvpn-agent")
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if err != nil || len(fields) == 0 {
		return "fvpn-agent pid absent"
	}
	pid := fields[0]
	for _, c := range pid {
		if c < '0' || c > '9' {
			return "fvpn-agent pid absent"
		}
	}
	before := s.forwardChain()
	if _, err := s.docker(s.node, "bash", "-c", "kill -9 "+pid); err != nil {
		return "SIGKILL failed"
	}
	time.Sleep(300 * time.Millisecond)
	after := s.forwardChain()
	via, result := s.tcpProbe("10.255.255.1", 80)
	text := fmt.Sprintf("pid %s generation %d -> %d policy %s probe %s %s", pid, before.Generation, after.Generation, after.Policy, via, result)
	if via == "fvnb0" && (result == "connected" || result == "refused") {
		return "allow after SIGKILL: " + text
	}
	if after.Generation < 1 || after.Policy == "accept" {
		return "ruleset dropped after SIGKILL: " + text
	}
	return text
}

func (s *stack) restartNetbird() string {
	if _, err := s.docker(s.node, "test", "-s", "/var/lib/vpn-node/identity/setup.key"); err != nil {
		return "setup key file is not on the node, so netbird up cannot be repeated"
	}
	down, err := s.docker(s.node, "netbird", "down")
	if err != nil {
		return "netbird down failed: " + clip(string(down), 120)
	}
	via, result := s.tcpProbe("10.255.255.1", 80)
	up, upErr := s.docker(s.node, "netbird", "up",
		"--setup-key-file", "/var/lib/vpn-node/identity/setup.key",
		"--management-url", "https://netbird.example.invalid:443",
		"--interface-name", "fvnb0", "--disable-dns", "--disable-ipv6")
	text := "down then probe " + via + " " + result
	if upErr != nil {
		text += "; up failed " + clip(string(up), 120)
	}
	if via == "fvnb0" && (result == "connected" || result == "refused") {
		return "allow while NetBird was down: " + text
	}
	return text
}

func (s *stack) memberCount() string {
	if _, err := s.docker(s.member, "true"); err != nil {
		return "member container is not running"
	}
	out, err := exec.Command("docker", "ps", "--filter", "name=^"+s.prefix+"-", "--format", "{{.Names}}").Output()
	if err != nil {
		return "container list failed"
	}
	n := 0
	for _, name := range strings.Fields(string(out)) {
		if strings.Contains(name, "member") {
			n++
		}
	}
	return fmt.Sprintf("%d member container", n)
}

func (s *stack) nodeIPs() []string {
	out, err := exec.Command("docker", "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}", s.node).Output()
	if err != nil {
		return nil
	}
	return strings.Fields(strings.TrimSpace(string(out)))
}

func (s *stack) ipv6Fact() string {
	out, err := s.docker(s.member, "ip", "-6", "addr", "show")
	if err != nil {
		return "member ipv6 address list failed"
	}
	var globals []string
	for _, line := range strings.Split(string(out), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "inet6 ") && !strings.Contains(trim, "fe80:") && !strings.Contains(trim, "::1/") {
			globals = append(globals, strings.Fields(trim)[1])
		}
	}
	if len(globals) == 0 {
		return "member has no global ipv6 address; the dev stack lab network is ipv4"
	}
	return "member global ipv6 " + strings.Join(globals, ",")
}

func (s *stack) dnsFact() string {
	return "designated resolver reachability is scored only when a route uses fvnb0; see resolver_ipv4"
}

func (s *stack) dockerDetach(container string, args ...string) error {
	cmd := exec.Command("docker", append([]string{"exec", "-d", container}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v %s", err, clip(string(out), 160))
	}
	return nil
}

func formatAttempts(items []attempt) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item.name+" "+item.via+" "+item.result)
	}
	return lines
}
