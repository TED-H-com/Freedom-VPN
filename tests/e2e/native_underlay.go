//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// PrepareNativeUnderlay attaches a dedicated IPv6-only bridge to the real node
// namespace and member. No host routes, DNS or nftables are edited. The bridge
// has no masquerading and no public endpoint; e2e-down owns it via node-*.
func (s *Stack) PrepareNativeUnderlay(ctx context.Context) error {
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
