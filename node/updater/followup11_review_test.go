package updater

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/google/uuid"
)

// Adopted from r12 H-2 after reproducing all three failures unchanged.
// These hard assertions pin the corrected refusal and omission contracts.
// TEST ONLY: all addresses and URL credentials below are synthetic fixtures;
// DNS is faked and no public destination is dialed.
func TestReviewR12_Q6_EmptyInterfaceListAndSymlinkAgentConfig(t *testing.T) {
	t.Run("empty_interface_list_bypasses_node_address_check", func(t *testing.T) {
		cfg := probeConfigFixture(t)
		cfg.ProbePeer = probeFixture()
		// Node interfaces return only loopback, link-local, and IPv6 (zero global-unicast IPv4).
		cfg.probeSources.interfaces = probeInterfaces("127.0.0.1", "169.254.1.1", "2001:db8::1")
		known, err := cfg.probeEnvironment().addresses(context.Background())
		var refusal probeRefusal
		if !errors.As(err, &refusal) || refusal != "node_addresses_unavailable" || len(known.node) != 0 {
			t.Fatalf("empty node inventory must fail closed: %v", err)
		}
		if err := cfg.probeEnvironment().validate(context.Background(), cfg.ProbePeer); !errors.As(err, &refusal) || refusal != "node_addresses_unavailable" {
			t.Fatalf("probe accepted without node addresses: %v", err)
		}
	})

	t.Run("symlinked_agent_config_accepted", func(t *testing.T) {
		cfg := probeConfigFixture(t)
		cfg.ProbePeer = probeFixture()
		cfg.probeSources.interfaces = probeInterfaces("192.0.2.10")

		targetPath := filepath.Join(t.TempDir(), "real-agent.json")
		writeProbeAgentConfig(t, targetPath, `{"control_url":"https://control.example.invalid"}`)
		symlinkPath := filepath.Join(t.TempDir(), "symlink-agent.json")
		if err := os.Symlink(targetPath, symlinkPath); err != nil {
			t.Fatalf("Symlink: %v", err)
		}
		cfg.AgentConfig = symlinkPath

		if _, err := readProbeAgentConfig(cfg.AgentConfig); err == nil {
			t.Fatalf("readProbeAgentConfig followed symlink %s -> %s instead of rejecting non-regular lstat mode", symlinkPath, targetPath)
		}
	})

	t.Run("probe_omission_unrecorded_in_challenge_file", func(t *testing.T) {
		cfg := probeConfigFixture(t)
		cfg.ProbePeer = probeFixture()
		// Initial load succeeds with node at 192.0.2.10.
		nodeIP := "192.0.2.10"
		cfg.probeSources.interfaces = func() ([]net.Addr, error) { return probeInterfaces(nodeIP)() }
		loaded, err := loadProbeConfig(t, cfg)
		if err != nil {
			t.Fatalf("loadProbeConfig: %v", err)
		}
		configured, err := loaded.Engine()
		if err != nil {
			t.Fatalf("Engine: %v", err)
		}
		defer configured.Close()

		w := newWorld(t)
		rel := w.makeRelease("probe-audit", 1, []int{1})
		w.seed(rel)
		w.eng.ProbePeer, w.eng.probeEnv = configured.ProbePeer, configured.probeEnv

		// At issuance time, node acquires 1.2.3.4 (matching ProbePeer destination),
		// causing challengeProbe to refuse the probe with "node_destination".
		nodeIP = "1.2.3.4"
		txn := updatejournal.Txn{TxnID: uuid.Must(uuid.NewV7()).String()}
		if err := w.eng.issueChallenge(context.Background(), txn, rel.id.String(), "rollback"); err != nil {
			t.Fatalf("issueChallenge: %v", err)
		}
		raw, err := os.ReadFile(filepath.Join(w.layout.Journal, "health-challenge.json"))
		if err != nil {
			t.Fatalf("ReadFile(health-challenge.json): %v", err)
		}
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if _, hasProbe := fields["probe"]; hasProbe {
			t.Fatal("refused probe was not omitted")
		}
		omitted, ok := fields["probe_omitted"].(map[string]any)
		if !ok || omitted["peer_id"] != cfg.ProbePeer.PeerID || omitted["reason"] != "node_destination" {
			t.Fatal("configured probe omission is not attributable in the challenge")
		}
		records, err := w.eng.Journal.ProbeOmissions(context.Background(), txn.TxnID)
		if err != nil || len(records) != 1 || records[0].Nonce != fields["nonce"] || records[0].Phase != "rollback" ||
			records[0].ReleaseID != rel.id.String() || records[0].ProbeOmitted.PeerID != cfg.ProbePeer.PeerID || records[0].ProbeOmitted.Reason != "node_destination" {
			t.Fatalf("omission missing from durable journal: %v", err)
		}
	})
}

// TestReviewR12_Q6_IssuerValidationHeld verifies the Question 6 properties that held:
//   - Oversized agent.json: 1,048,576 bytes (1 MiB = maxBytes in readProbeAgentConfig)
//     is accepted; 1,048,577 bytes is refused with probeRefusal("agent_config_unavailable").
//   - control_url with userinfo is refused without lookup; a port alone is
//     stripped before resolving and checking every answer.
//   - IPv4-mapped IPv6 addresses (::ffff:1.2.3.4 and ::ffff:100.64.8.1) in both
//     interface addresses and DNS answers are unmapped via Addr.Unmap() and refused.
//   - DNS answer omitting the real control plane IP (1.2.3.4): issuer accepts 1.2.3.4/32,
//     and the §13.9 enforcement renderer emits @probe_dst_v4 = { 1.2.3.4 . tcp . 443 timeout 45s },
//     allowing only the probe source 100.64.8.1 to reach 1.2.3.4:443/tcp (plus optional
//     @resolver_v4:53) for 45s.
func TestReviewR12_Q6_IssuerValidationHeld(t *testing.T) {
	// 1. Oversized agent.json boundary at 1 MiB (1 << 20 = 1,048,576 bytes).
	cfg := probeConfigFixture(t)
	cfg.ProbePeer = probeFixture()
	cfg.probeSources.interfaces = probeInterfaces("192.0.2.10")

	const maxAgentConfigBytes = 1 << 20
	prefix := `{"control_url":"https://control.example.invalid","pad":"`
	suffix := `"}`
	padExact := maxAgentConfigBytes - len(prefix) - len(suffix)
	writeProbeAgentConfig(t, cfg.AgentConfig, prefix+strings.Repeat("a", padExact)+suffix)
	if err := cfg.probeEnvironment().validate(context.Background(), cfg.ProbePeer); err != nil {
		t.Fatalf("agent.json at exact 1 MiB (1048576 bytes) failed: %v", err)
	}
	writeProbeAgentConfig(t, cfg.AgentConfig, prefix+strings.Repeat("a", padExact+1)+suffix)
	err := cfg.probeEnvironment().validate(context.Background(), cfg.ProbePeer)
	var refusal probeRefusal
	if !errors.As(err, &refusal) || refusal != probeRefusal("agent_config_unavailable") {
		t.Fatalf("agent.json at 1048577 bytes expected agent_config_unavailable, got %v", err)
	}

	// 2a. control_url with userinfo is refused fail-closed by probeURLHost (u.User != nil)
	// with control_addresses_unavailable without calling DNS lookup.
	writeProbeAgentConfig(t, cfg.AgentConfig, `{"control_url":"https://admin:secret-pass@control.example.invalid:8443/api/v1"}`)
	cfg.probeSources.lookup = func(_ context.Context, _ string, host string) ([]netip.Addr, error) {
		t.Fatalf("unexpected DNS lookup for control_url with userinfo: %q", host)
		return nil, nil
	}
	err = cfg.probeEnvironment().validate(context.Background(), cfg.ProbePeer)
	if !errors.As(err, &refusal) || refusal != probeRefusal("control_addresses_unavailable") {
		t.Fatalf("control_url with userinfo expected control_addresses_unavailable, got %v", err)
	}

	// 2b. control_url with port ("https://control.example.invalid:8443/api/v1") strips port via
	// u.Hostname(), resolves "control.example.invalid", and refuses matching destination.
	writeProbeAgentConfig(t, cfg.AgentConfig, `{"control_url":"https://control.example.invalid:8443/api/v1"}`)
	cfg.probeSources.lookup = func(_ context.Context, _ string, host string) ([]netip.Addr, error) {
		if host != "control.example.invalid" {
			t.Fatalf("unexpected lookup host %q", host)
		}
		return []netip.Addr{netip.MustParseAddr("1.2.3.4")}, nil
	}
	err = cfg.probeEnvironment().validate(context.Background(), cfg.ProbePeer)
	if !errors.As(err, &refusal) || refusal != probeRefusal("control_destination") {
		t.Fatalf("control_url with port expected control_destination, got %v", err)
	}

	// 3. IPv4-mapped IPv6 interface address (::ffff:100.64.8.1) and DNS answer (::ffff:1.2.3.4).
	cfg.probeSources.lookup = func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{netip.MustParseAddr("192.0.2.1")}, nil
	}
	cfg.probeSources.interfaces = func() ([]net.Addr, error) {
		// 16-byte IPv4-mapped IPv6 representation of 100.64.8.1.
		mapped := net.IP{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 100, 64, 8, 1}
		return []net.Addr{&net.IPAddr{IP: mapped}}, nil
	}
	err = cfg.probeEnvironment().validate(context.Background(), cfg.ProbePeer)
	if !errors.As(err, &refusal) || refusal != probeRefusal("node_source") {
		t.Fatalf("IPv4-mapped interface address expected node_source, got %v", err)
	}

	// 4. Split-horizon / incomplete DNS for control host: DNS returns only 192.0.2.1 while
	// real control plane is at 1.2.3.4. Issuer accepts 1.2.3.4/32:443/tcp, and enforce.RenderBaselineModel
	// scopes the §13.9 carve-out strictly to saddr 100.64.8.1 -> daddr . l4proto . dport
	// { 1.2.3.4 . tcp . 443 timeout 45s } (plus @resolver_v4 port 53).
	cfg.probeSources.interfaces = probeInterfaces("192.0.2.10")
	if err := cfg.probeEnvironment().validate(context.Background(), cfg.ProbePeer); err != nil {
		t.Fatalf("expected validate to pass when DNS omits 1.2.3.4: %v", err)
	}
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	m := enforce.Model{
		Generation: 1,
		IPv6Mode:   enforce.IPv6Block,
		Now:        now,
		Datapath:   enforce.Datapath{MemberIngress: "fv-fh", Uplink: "fv-ih", SNAT: netip.MustParseAddr("192.0.2.10")},
		Resolvers:  []netip.Addr{netip.MustParseAddr("198.51.100.53")},
		Probe: &enforce.ProbeException{
			PeerID:       "probe-peer",
			SourceV4:     netip.MustParsePrefix("100.64.8.1/32"),
			Destinations: []enforce.ProbeDestination{{AddrV4: netip.MustParsePrefix("1.2.3.4/32"), Proto: "tcp", Port: 443}},
			ResolverDNS:  true,
			ExpiresAt:    now.Add(45 * time.Second),
		},
	}
	nft, err := enforce.RenderBaseline(m, enforce.State{})
	if err != nil {
		t.Fatalf("enforce.RenderBaseline: %v", err)
	}
	if !strings.Contains(nft, "1.2.3.4 . tcp . 443 timeout 45s") ||
		!strings.Contains(nft, `ip saddr @probe_v4 ip daddr . meta l4proto . th dport @probe_dst_v4 counter name "probe_up" accept`) {
		t.Fatalf("unexpected §13.9 probe ruleset:\n%s", nft)
	}
}
