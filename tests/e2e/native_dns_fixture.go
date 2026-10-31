//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/streamstats"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

const fixtureProbe = "/usr/local/bin/fvpn-e2e-fixture-probe"

type FixtureProbeResult struct {
	Endpoint       *streamstats.Sample `json:"endpoint,omitempty"`
	AuthorityTTL   uint32              `json:"authority_ttl"`
	HasSOA         bool                `json:"has_soa"`
	Available      bool                `json:"available"`
	RCode          int                 `json:"rcode"`
	Types          []uint16            `json:"types"`
	TTLs           []uint32            `json:"ttls"`
	Classes        []string            `json:"address_classes"`
	ExpectedAnswer bool                `json:"expected_answer"`
	Connected      bool                `json:"connected"`
	Bytes          int                 `json:"bytes"`
	Milliseconds   int64               `json:"milliseconds"`
	Error          string              `json:"error,omitempty"`
}

func (s *Stack) InstallFixtureProbe(ctx context.Context) error {
	path := filepath.Join(s.lab.state, "fixture-probe")
	cmd := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", path, "./tests/e2e/cmd/fvpn-e2e-fixture-probe")
	cmd.Dir = s.lab.root
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build fixture probe: %w", err)
	}
	for _, container := range []string{s.lab.node, s.lab.member} {
		if err := dockerCopy(ctx, path, container+":"+fixtureProbe); err != nil {
			return err
		}
	}
	return nil
}

// StartFixtureTargets starts only a private, content-free reachability listener
// in the node namespace. It is independent of member-path success and removed
// with the node container. Call after InstallFixtureProbe and before enrollment.
func (s *Stack) StartFixtureTargets(ctx context.Context) error {
	if err := exec.CommandContext(ctx, "docker", "exec", "-d", s.lab.node, fixtureProbe, "-mode", "listen", "-addr", ":9447").Run(); err != nil {
		return fmt.Errorf("fixture target start: %w", err)
	}
	wait, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if !waitUntil(wait, 100*time.Millisecond, func() bool {
		got, e := s.fixtureProbe(wait, s.lab.node, "-mode", "connect", "-addr", "127.0.0.1:9447")
		return e == nil && got.Connected
	}) {
		return fmt.Errorf("fixture target listener unavailable")
	}
	return nil
}

// ProbeNativeDNS uses /etc/resolv.conf when resolver is empty, i.e. the native
// client's local resolver. It exports metadata only, never query contents.
func (s *Stack) ProbeNativeDNS(ctx context.Context, resolver, name string, rr uint16, tcp bool, expected string) (FixtureProbeResult, error) {
	proto := "udp"
	if tcp {
		proto = "tcp"
	}
	return s.fixtureProbe(ctx, s.lab.member, "-mode", "dns", "-addr", resolver, "-name", name, "-type", fmt.Sprint(rr), "-proto", proto, "-expect", expected)
}
func (s *Stack) fixtureProbe(ctx context.Context, container string, args ...string) (out FixtureProbeResult, err error) {
	cmdArgs := append([]string{"exec", container, fixtureProbe}, args...)
	raw, err := exec.CommandContext(ctx, "docker", cmdArgs...).Output()
	if err != nil || json.Unmarshal(raw, &out) != nil {
		return out, fmt.Errorf("fixture probe execution failed")
	}
	return out, nil
}

type NativeDNSConfiguration struct {
	ObservedNativeResolvers []string  `json:"observed_native_resolvers"`
	NativeResolversMatch    bool      `json:"native_resolvers_match"`
	Resolvers               []string  `json:"resolvers"`
	NativeDNSEnabled        bool      `json:"native_dns_enabled"`
	SystemUsesNativeStub    bool      `json:"system_uses_native_stub"`
	DocumentHashes          []string  `json:"document_hashes"`
	SameAppliedDocument     bool      `json:"same_applied_document"`
	ConfiguredAt            time.Time `json:"configured_at"`
}

// ConfigureNativeDNS reads the product-issued resolver list and both nodes'
// applied document evidence. It publishes those addresses as a TEST ONLY native
// nameserver group restricted to this member peer; admission/ACLs are unchanged.
// The filtering policy is never uploaded through an AdGuard admin mutation.
func (s *Stack) ConfigureNativeDNS(ctx context.Context, admin BrowserSession, device uuid.UUID, nodes []uuid.UUID) (out NativeDNSConfiguration, err error) {
	raw, err := s.lab.docker(s.lab.node, "cat", "/var/lib/vpn-node/state/desired-state.json")
	var desired struct {
		Signed struct {
			DNS struct {
				V4 []string `json:"resolver_ipv4"`
			} `json:"dns"`
		} `json:"signed"`
	}
	if err != nil || json.Unmarshal(raw, &desired) != nil || len(desired.Signed.DNS.V4) != 2 {
		return out, fmt.Errorf("native DNS requires two product-issued resolvers (prepare with FVPN_E2E_TWO_RESOLVERS=true)")
	}
	out.Resolvers = desired.Signed.DNS.V4
	prefix := s.lab.originIP[:strings.LastIndex(s.lab.originIP, ".")]
	if out.Resolvers[0] != prefix+".53" || out.Resolvers[1] != prefix+".54" || len(nodes) != 2 {
		return out, fmt.Errorf("native DNS resolver topology mismatch")
	}
	for _, id := range nodes {
		resp, e := s.Call(ctx, admin, "GET", "/api/v1/admin/nodes/"+id.String(), nil, nil)
		var detail api.NodeDetail
		if e != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &detail) != nil {
			return out, fmt.Errorf("native DNS node readback")
		}
		ev, e := detail.DnsPolicyEvidence.Get()
		if e != nil || ev.Outcome != api.DnsPolicyEvidenceOutcomeApplied {
			return out, fmt.Errorf("native DNS policy not applied on both nodes")
		}
		out.DocumentHashes = append(out.DocumentHashes, ev.ContentSha256)
	}
	out.SameAppliedDocument = out.DocumentHashes[0] != "" && out.DocumentHashes[0] == out.DocumentHashes[1]
	if !out.SameAppliedDocument {
		return out, fmt.Errorf("native DNS documents differ")
	}
	image, err := exec.CommandContext(ctx, "docker", "inspect", "--format={{.Image}}", s.lab.prefix+"-origin").Output()
	if err != nil {
		return out, err
	}
	if err = exec.CommandContext(ctx, "docker", "run", "-d", "--name", s.lab.prefix+"-origin-dns-secondary", "--network", s.lab.prefix+"-origin", "--ip", out.Resolvers[1], strings.TrimSpace(string(image)), "-addr", ":53", "-dns-upstream", prefix+".22:5353").Run(); err != nil {
		return out, fmt.Errorf("secondary DNS transport: %w", err)
	}
	conn, err := s.lab.db(ctx)
	if err != nil {
		return out, err
	}
	defer conn.Close(ctx)
	var peer string
	if err = conn.QueryRow(ctx, `SELECT peer_id FROM member_devices WHERE id=$1 AND status='authorized'`, device).Scan(&peer); err != nil {
		return out, fmt.Errorf("native DNS authorized member peer unavailable")
	}
	fixture := &probeFixture{lab: s.lab}
	var group struct {
		ID string `json:"id"`
	}
	if err = fixture.netbirdAPI(ctx, "POST", "/groups", map[string]any{"name": "TEST ONLY native DNS fixture", "peers": []string{peer}}, &group); err != nil {
		return out, err
	}
	var created map[string]any
	if err = fixture.netbirdAPI(ctx, "POST", "/dns/nameservers", map[string]any{"name": "TEST ONLY controlled fallback", "description": "product-issued addresses, fixture native distribution", "enabled": true, "groups": []string{group.ID}, "primary": true, "domains": []string{}, "search_domains_enabled": false, "nameservers": []any{map[string]any{"ip": out.Resolvers[0], "ns_type": "udp", "port": 53}, map[string]any{"ip": out.Resolvers[1], "ns_type": "udp", "port": 53}}}, &created); err != nil {
		return out, err
	}
	// Control names must not depend on the member's intentionally failing DNS.
	hosts := fmt.Sprintf("%s.21 netbird.example.invalid portal.example.invalid\n%s.11 idp.example.invalid\n", s.lab.labPrefix, s.lab.labPrefix)
	cmd := exec.CommandContext(ctx, "docker", "exec", "-i", s.lab.member, "sh", "-c", "cat >> /etc/hosts")
	cmd.Stdin = strings.NewReader(hosts)
	if err = cmd.Run(); err != nil {
		return out, err
	}
	if err = exec.CommandContext(ctx, "docker", "exec", s.lab.member, "netbird", "down").Run(); err != nil {
		return out, fmt.Errorf("restart native DNS client")
	}
	if err = exec.CommandContext(ctx, "docker", "exec", s.lab.member, "netbird", "up", "--disable-dns=false").Run(); err != nil {
		return out, fmt.Errorf("enable native DNS")
	}
	wait, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if !waitUntil(wait, time.Second, func() bool {
		raw, e := s.lab.docker(s.lab.member, "cat", "/var/lib/netbird/default.json")
		var native struct{ DisableDNS bool }
		out.NativeDNSEnabled = e == nil && json.Unmarshal(raw, &native) == nil && !native.DisableDNS
		raw, e = s.lab.docker(s.lab.member, "cat", "/etc/resolv.conf")
		out.SystemUsesNativeStub = false
		nameservers := 0
		if e == nil {
			for _, line := range strings.Split(string(raw), "\n") {
				fields := strings.Fields(line)
				if len(fields) == 2 && fields[0] == "nameserver" {
					nameservers++
					ip := net.ParseIP(fields[1])
					if ip != nil && (strings.HasPrefix(fields[1], "100.") || fields[1] == "127.0.0.1") {
						out.SystemUsesNativeStub = true
					}
				}
			}
		}
		out.SystemUsesNativeStub = out.SystemUsesNativeStub && nameservers == 1
		raw, e = s.lab.docker(s.lab.member, "netbird", "status", "--json")
		var status struct {
			DNS []struct {
				Servers []string `json:"servers"`
				Enabled bool     `json:"enabled"`
			} `json:"dnsServers"`
		}
		out.ObservedNativeResolvers = nil
		if e == nil && json.Unmarshal(raw, &status) == nil {
			for _, group := range status.DNS {
				if group.Enabled {
					out.ObservedNativeResolvers = append(out.ObservedNativeResolvers, group.Servers...)
				}
			}
		}
		out.NativeResolversMatch = len(out.ObservedNativeResolvers) == 2
		for _, resolver := range out.Resolvers {
			found := false
			for _, got := range out.ObservedNativeResolvers {
				found = found || got == resolver+":53"
			}
			out.NativeResolversMatch = out.NativeResolversMatch && found
		}
		return out.NativeDNSEnabled && out.SystemUsesNativeStub && out.NativeResolversMatch
	}) {
		return out, fmt.Errorf("native system resolver not installed")
	}
	out.ConfiguredAt = time.Now().UTC()
	return out, nil
}

// SetDNSFixturePhase replaces only the owned synthetic authority zone. Resolver
// caches are deliberately left alone. Never call this with production names.
func (s *Stack) SetDNSFixturePhase(phase string) error {
	if phase != "initial" && phase != "private" {
		return fmt.Errorf("unknown DNS fixture phase")
	}
	raw, err := os.ReadFile(filepath.Join(s.lab.root, "deploy/devstack/coredns/example.invalid.zone"))
	if err != nil {
		return err
	}
	text := strings.ReplaceAll(string(raw), "172.28.30.", s.lab.labPrefix+".")
	prefix := s.lab.originIP[:strings.LastIndex(s.lab.originIP, ".")]
	text = strings.ReplaceAll(text, "192.0.3.", prefix+".")
	if phase == "private" {
		text = strings.Replace(text, ". 1001 3600 600 86400 5", ". 1002 3600 600 86400 5", 1)
		text += "absent 5 IN A " + s.lab.originIP + "\n"
		text = strings.Replace(text, "cache 10 IN A "+s.lab.originIP, "cache 10 IN A 10.203.0.30", 1)
		text = strings.Replace(text, "rebind 5 IN A "+s.lab.originIP, "rebind 5 IN A 10.203.0.30", 1)
		text = strings.Replace(text, "rebind 5 IN AAAA 2001:db8:30::10", "rebind 5 IN AAAA fd00::30", 1)
	}
	return os.WriteFile(filepath.Join(s.lab.state, "render/coredns/example.invalid.zone"), []byte(text), 0600)
}
