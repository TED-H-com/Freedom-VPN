//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/nativecapture"
)

// PrepareNativeUnderlay attaches a dedicated IPv6-only bridge to the real node
// namespace and member. No host routes, DNS or nftables are edited. The bridge
// has no masquerading and no public endpoint; e2e-down owns it via node-*.
func (s *Stack) PrepareNativeUnderlay(ctx context.Context) error {
	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !waitUntil(wait, time.Second, func() bool { return s.lab.stage(s.lab.node, "/var/lib/vpn-node/e2e/status.json") == "waiting_token" }) {
		return fmt.Errorf("underlay requires a running fresh node waiting for enrollment")
	}
	network := s.lab.prefix + "-node-underlay"
	if exec.CommandContext(ctx, "docker", "network", "inspect", network).Run() == nil {
		return fmt.Errorf("underlay requires a fresh fixture")
	}
	slot := strings.TrimPrefix(s.lab.labPrefix, "172.28.")
	prefix := "2001:db8:" + slot + "::"
	steps := [][]string{
		{"network", "create", "--ipv4=false", "--ipv6", "--subnet", prefix + "/64", "--opt", "com.docker.network.bridge.enable_ip_masquerade=false", network},
		{"network", "connect", "--ip6", prefix + "40", network, s.lab.prefix + "-node-netns"},
		{"network", "connect", "--ip6", prefix + "41", network, s.lab.member},
	}
	for _, args := range steps {
		if err := exec.CommandContext(ctx, "docker", args...).Run(); err != nil {
			return fmt.Errorf("underlay network step: %w", err)
		}
	}
	// Controlled private targets live behind the exit, outside the member's
	// directly connected underlay. This avoids mistaking a client LAN route for
	// member traffic through the exit. Configure before the first topology hash.
	for _, args := range [][]string{{"link", "add", "fvfixture", "type", "dummy"}, {"addr", "add", "10.203.0.30/32", "dev", "fvfixture"}, {"-6", "addr", "add", "fd00::30/128", "dev", "fvfixture"}, {"link", "set", "fvfixture", "up"}} {
		command := append([]string{"exec", s.lab.node, "ip"}, args...)
		if err := exec.CommandContext(ctx, "docker", command...).Run(); err != nil {
			return fmt.Errorf("private fixture target: %w", err)
		}
	}
	// Interface additions on the node are deliberately done before its first
	// apply: changing topology under a serving agent must trigger containment.
	return nil
}

// ApplyNativeInputs changes the member's isolated underlay MTU and bounded
// family filter. TCP control remains reachable. For ipv6, IPv4 peer UDP is
// unavailable; NetBird may choose IPv6 or relay, and the observation decides.
func (s *Stack) ApplyNativeInputs(ctx context.Context, in NativeFixtureInputs) (NativeFixtureSelection, error) {
	out, err := SelectNativeFixture(in)
	if err != nil || !out.Available {
		return out, err
	}
	for _, dev := range []string{"eth0", "eth1"} {
		if err = exec.CommandContext(ctx, "docker", "exec", s.lab.member, "ip", "link", "set", "dev", dev, "mtu", fmt.Sprint(in.MTU)).Run(); err != nil {
			return out, fmt.Errorf("member underlay MTU: %w", err)
		}
	}
	_ = exec.CommandContext(ctx, "docker", "exec", s.lab.member, "nft", "delete", "table", "inet", "fvfixture").Run()
	script := "add table inet fvfixture\nadd set inet fvfixture held { type ifname; flags timeout; elements = { \"eth0\" timeout 180s, \"eth1\" timeout 180s }; }\nadd chain inet fvfixture out { type filter hook output priority 0; policy accept; }\nadd chain inet fvfixture in { type filter hook input priority 0; policy accept; }\n"
	family := ""
	if in.Underlay == "ipv4" {
		family = "ipv6"
	}
	if in.Underlay == "ipv6" {
		family = "ipv4"
	}
	if family != "" {
		for _, direction := range []struct{ chain, iface string }{{"out", "oifname"}, {"in", "iifname"}} {
			script += fmt.Sprintf("add rule inet fvfixture %s %s @held meta nfproto %s meta l4proto udp counter drop\n", direction.chain, direction.iface, family)
		}
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", s.lab.member, "nft", "-f", "-")
	cmd.Stdin = strings.NewReader(script)
	if err = cmd.Run(); err != nil {
		return out, fmt.Errorf("member family filter: %w", err)
	}
	return out, nil
}

type NativeUnderlayObservation struct {
	Path              RelayPath      `json:"path"`
	CandidateFamilies map[string]int `json:"candidate_families"`
}

// ObserveNativeUnderlay emits endpoint address families, never peer IDs/keys.
func (s *Stack) ObserveNativeUnderlay(ctx context.Context) (out NativeUnderlayObservation, err error) {
	out.CandidateFamilies = map[string]int{}
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.member, "netbird", "status", "--json").Output()
	if err != nil {
		return out, fmt.Errorf("native status unavailable")
	}
	out.Path.Relayed, out.Path.Direct = classifyNetbird(raw)
	var doc any
	if json.Unmarshal(raw, &doc) != nil {
		return out, fmt.Errorf("native status malformed")
	}
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case []any:
			for _, v := range x {
				walk(v)
			}
		case map[string]any:
			for k, v := range x {
				if strings.EqualFold(k, "iceCandidateEndpoint") {
					if endpoints, ok := v.(map[string]any); ok {
						for _, endpoint := range endpoints {
							if addr, ok := endpoint.(string); ok && addr != "" {
								family := "ipv4"
								if strings.Count(addr, ":") > 1 {
									family = "ipv6"
								}
								out.CandidateFamilies[family]++
							}
						}
					}
				}
				walk(v)
			}
		}
	}
	walk(doc)
	return out, nil
}

// SelectedNativeExit records only the selected exit's transport metadata.
// Matching uses its native overlay address in memory; addresses/keys stay private.
type SelectedNativeExit struct {
	Connected bool   `json:"connected"`
	Type      string `json:"connection_type"`
	Family    string `json:"chosen_family"`
}

func (s *Stack) ObserveSelectedNativeExit(ctx context.Context) (SelectedNativeExit, error) {
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.node, "netbird", "status", "--json").Output()
	var node struct {
		IP string `json:"netbirdIp"`
	}
	if err != nil || json.Unmarshal(raw, &node) != nil || node.IP == "" {
		return SelectedNativeExit{}, fmt.Errorf("exit native address unavailable")
	}
	raw, err = exec.CommandContext(ctx, "docker", "exec", s.lab.member, "netbird", "status", "--json").Output()
	if err != nil {
		return SelectedNativeExit{}, fmt.Errorf("member native status unavailable")
	}
	var status struct {
		Peers struct {
			Details []struct {
				IP       string `json:"netbirdIp"`
				Status   string `json:"status"`
				Type     string `json:"connectionType"`
				Endpoint struct {
					Remote string `json:"remote"`
				} `json:"iceCandidateEndpoint"`
			} `json:"details"`
		} `json:"peers"`
	}
	if json.Unmarshal(raw, &status) != nil {
		return SelectedNativeExit{}, fmt.Errorf("native status malformed")
	}
	for _, p := range status.Peers.Details {
		if strings.Split(p.IP, "/")[0] != strings.Split(node.IP, "/")[0] {
			continue
		}
		out := SelectedNativeExit{Connected: strings.EqualFold(p.Status, "Connected"), Type: p.Type, Family: "unavailable"}
		host, _, e := net.SplitHostPort(p.Endpoint.Remote)
		if e == nil {
			if ip := net.ParseIP(host); ip != nil {
				out.Family = "ipv6"
				if ip.To4() != nil {
					out.Family = "ipv4"
				}
			}
		}
		return out, nil
	}
	return SelectedNativeExit{}, fmt.Errorf("selected exit absent")
}

// ConstrainSelectedNativePeer1280 adds a TEST ONLY /128 route on the selected
// sender, never a global MTU/MSS or product change. ApplyNativeInputs deliberately
// remains asymmetric. This control models known path MTU without inventing PTB.
// The returned cleanup removes only the new route; an existing host route is
// refused by `route add`, never replaced. It requires this fixture's IPv6 link.
func (s *Stack) ConstrainSelectedNativePeer1280(ctx context.Context) (func(context.Context) error, error) {
	addresses, err := s.nativeAddresses(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := exec.CommandContext(ctx, "docker", "exec", s.lab.node, "netbird", "status", "--json").Output()
	if err != nil {
		return nil, fmt.Errorf("sender status unavailable")
	}
	p, err := nativecapture.Select(raw, addresses["member"])
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(p.Endpoint.Remote)
	expected := "2001:db8:" + strings.TrimPrefix(s.lab.labPrefix, "172.28.") + "::41"
	if err != nil || host != expected || !strings.EqualFold(p.Status, "Connected") || p.Type != "P2P" {
		return nil, fmt.Errorf("selected sender is not the fixture IPv6 member endpoint")
	}
	raw, err = exec.CommandContext(ctx, "docker", "exec", s.lab.node, "ip", "-j", "-6", "route", "get", host).Output()
	var routes []struct {
		Dev string `json:"dev"`
	}
	if err != nil || json.Unmarshal(raw, &routes) != nil || len(routes) != 1 || !regexp.MustCompile(`^eth[0-9]+$`).MatchString(routes[0].Dev) {
		return nil, fmt.Errorf("selected sender underlay route unavailable")
	}
	route := []string{"-6", "route", "add", host + "/128", "dev", routes[0].Dev, "mtu", "1280"}
	if err = exec.CommandContext(ctx, "docker", append([]string{"exec", s.lab.node, "ip"}, route...)...).Run(); err != nil {
		return nil, fmt.Errorf("add isolated peer route: %w", err)
	}
	return func(clean context.Context) error {
		return exec.CommandContext(clean, "docker", "exec", s.lab.node, "ip", "-6", "route", "del", host+"/128", "dev", routes[0].Dev, "mtu", "1280").Run()
	}, nil
}

// NativeRouter1280 is a TEST ONLY routed underlay. Existing member-only and
// known-PMTU controls are independent. Both bridge links have matching MTUs.
type NativeRouter1280 struct {
	stack     *Stack
	Container string `json:"container"`
	Member4   string `json:"member_ipv4"`
	Member6   string `json:"member_ipv6"`
	Router4   string `json:"router_ipv4"`
	Router6   string `json:"router_ipv6"`
}

// PrepareNativeRouter1280 adds no node interface and changes no host networking.
// The new member link is 1280; router ingress and node links remain 1500. Routes
// are specific to fixture peers and contain no MTU metric. e2e-down owns cleanup.
func (s *Stack) PrepareNativeRouter1280(ctx context.Context) (*NativeRouter1280, error) {
	if s.lab.node != s.lab.prefix+"-node" || s.lab.member != s.lab.prefix+"-member" {
		return nil, fmt.Errorf("router requires the canonical prepared native node/member pair")
	}
	slot := strings.TrimPrefix(s.lab.labPrefix, "172.28.")
	left6 := "2001:db8:" + slot + "::"
	right6 := "2001:db8:" + slot + ":23::"
	network4, router4, member4, err := nativeRouterIPv4(slot)
	if err != nil {
		return nil, err
	}
	h := &NativeRouter1280{stack: s, Container: s.lab.prefix + "-node-router", Member4: member4, Router4: router4, Member6: right6 + "3", Router6: right6 + "2"}
	network := s.lab.prefix + "-node-router-link"
	image, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Image}}", s.lab.node).Output()
	if err != nil {
		return nil, err
	}
	steps := [][]string{
		{"network", "create", "--ipv6", "--subnet", network4, "--subnet", right6 + "/64", "--opt", "com.docker.network.bridge.enable_ip_masquerade=false", "--opt", "com.docker.network.driver.mtu=1280", network},
		{"run", "-d", "--name", h.Container, "--privileged", "--network", s.lab.prefix, "--ip", s.lab.labPrefix + ".60", "--entrypoint", "sleep", strings.TrimSpace(string(image)), "infinity"},
		{"network", "connect", "--ip6", left6 + "60", s.lab.prefix + "-node-underlay", h.Container},
		{"network", "connect", "--ip", h.Router4, "--ip6", h.Router6, network, h.Container},
		{"network", "connect", "--ip", h.Member4, "--ip6", h.Member6, network, s.lab.member},
		{"exec", h.Container, "sysctl", "-w", "net.ipv4.ip_forward=1", "net.ipv6.conf.all.forwarding=1"},
		{"exec", s.lab.node, "ip", "route", "add", h.Member4 + "/32", "via", s.lab.labPrefix + ".60"},
		{"exec", s.lab.node, "ip", "-6", "route", "add", h.Member6 + "/128", "via", left6 + "60"},
		{"exec", s.lab.member, "ip", "route", "add", s.lab.labPrefix + ".40/32", "via", h.Router4},
		{"exec", s.lab.member, "ip", "-6", "route", "add", left6 + "40/128", "via", h.Router6},
	}
	for i, args := range steps {
		if err = exec.CommandContext(ctx, "docker", args...).Run(); err != nil {
			return nil, fmt.Errorf("router step %d: %w", i, err)
		}
	}
	return h, nil
}

// Select forces native peer UDP through the new member link, with bounded rules.
// It does not reconnect or transfer. Call the consumer's original down/up next.
func (h *NativeRouter1280) Select(ctx context.Context, family string) error {
	if family != "ipv4" && family != "ipv6" {
		return fmt.Errorf("router family must be ipv4 or ipv6")
	}
	s := h.stack
	_, err := s.ApplyNativeInputs(ctx, NativeFixtureInputs{Underlay: family, MTU: 1500, Protocol: "tcp", Platform: "linux"})
	if err != nil {
		return err
	}
	other := "ipv4"
	if family == "ipv4" {
		other = "ipv6"
	}
	script := "add set inet fvfixture routed { type ifname; flags timeout; elements = { \"eth2\" timeout 180s }; }\n"
	for _, d := range []struct{ chain, iface string }{{"out", "oifname"}, {"in", "iifname"}} {
		script += fmt.Sprintf("add rule inet fvfixture %s %s @held meta l4proto udp counter drop\n", d.chain, d.iface)
		script += fmt.Sprintf("add rule inet fvfixture %s %s @routed meta nfproto %s meta l4proto udp counter drop\n", d.chain, d.iface, other)
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", s.lab.member, "nft", "-f", "-")
	cmd.Stdin = strings.NewReader(script)
	return cmd.Run()
}

// StartCapture includes both router interfaces in the ready/stop evidence.
func (h *NativeRouter1280) StartCapture(ctx context.Context) (*NativeCapture, error) {
	return h.stack.startNativeCapture(ctx, map[string]string{"router": h.Container})
}

// Preserve the default lane's .128/28 and reserve a disjoint /28 per port slot.
func nativeRouterIPv4(slot string) (network, router, member string, err error) {
	n, e := strconv.Atoi(slot)
	if e != nil || n < 30 || n > 39 {
		return "", "", "", fmt.Errorf("unknown native lab slot")
	}
	base := ((n - 30 + 8) % 16) * 16
	return fmt.Sprintf("192.0.2.%d/28", base), fmt.Sprintf("192.0.2.%d", base+2), fmt.Sprintf("192.0.2.%d", base+3), nil
}
