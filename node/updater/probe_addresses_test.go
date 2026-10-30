package updater

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/google/uuid"
)

func probeInterfaces(addrs ...string) func() ([]net.Addr, error) {
	return func() ([]net.Addr, error) {
		var out []net.Addr
		for _, addr := range addrs {
			out = append(out, &net.IPNet{IP: net.ParseIP(addr), Mask: net.CIDRMask(24, 32)})
		}
		return out, nil
	}
}

func TestUPD_D063_ProbeAddressRefusals(t *testing.T) {
	// TEST ONLY: public stand-ins exercise exclusions, never network traffic.
	for _, tc := range []struct {
		name, reason string
		change       func(*testing.T, *Config)
	}{
		{"node_destination", "node_destination", func(_ *testing.T, c *Config) {
			c.probeSources.interfaces = probeInterfaces("192.0.2.1", "1.2.3.4")
		}},
		{"node_overlay_source", "node_source", func(_ *testing.T, c *Config) {
			c.probeSources.interfaces = probeInterfaces("192.0.2.1", "100.64.8.1")
		}},
		{"node_private_source", "node_source", func(_ *testing.T, c *Config) {
			c.ProbePeer.SourceV4 = "10.9.8.7/32"
			c.probeSources.interfaces = probeInterfaces("10.9.8.7")
		}},
		{"control_literal", "control_destination", func(t *testing.T, c *Config) {
			writeProbeAgentConfig(t, c.AgentConfig, `{"control_url":"https://1.2.3.4:443/api/v1"}`)
			c.probeSources.lookup = func(context.Context, string, string) ([]netip.Addr, error) {
				t.Fatal("literal address used DNS")
				return nil, nil
			}
		}},
		{"control_all_answers", "control_destination", func(t *testing.T, c *Config) {
			c.probeSources.lookup = func(ctx context.Context, network, host string) ([]netip.Addr, error) {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 2*time.Second || network != "ip" || host != "control.example.invalid" {
					t.Fatal("unbounded or incorrect control lookup")
				}
				return []netip.Addr{netip.MustParseAddr("2001:db8::1"), netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("::ffff:1.2.3.4")}, nil
			}
		}},
		{"result_host", "control_destination", func(_ *testing.T, c *Config) {
			c.ResultURL = "https://1.2.3.4/api/v1/result"
		}},
		{"resolver_listener", "resolver_destination", func(t *testing.T, c *Config) {
			writeProbeAgentConfig(t, c.AgentConfig, `{"control_url":"https://192.0.2.1","dns":{"resolvers":[{"admin_url":"http://127.0.0.1:3000","dns_addr":"1.2.3.4:53"}]}}`)
		}},
		{"resolver_admin", "resolver_destination", func(t *testing.T, c *Config) {
			writeProbeAgentConfig(t, c.AgentConfig, `{"control_url":"https://192.0.2.1","dns":{"resolvers":[{"admin_url":"http://1.2.3.4:3000","dns_addr":"127.0.0.1:53"}]}}`)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := probeConfigFixture(t)
			cfg.ProbePeer = probeFixture()
			// Resolver exclusions also apply when no DNS exception is requested.
			cfg.ProbePeer.ResolverDNS = false
			tc.change(t, &cfg)
			if _, err := loadProbeConfig(t, cfg); err == nil || err.Error() != errProbePeer.Error() {
				t.Fatalf("LoadConfig = %v", err)
			}
			if _, err := cfg.Engine(); err == nil || err.Error() != errProbePeer.Error() {
				t.Fatalf("Engine = %v", err)
			}
			err := cfg.probeEnvironment().validate(context.Background(), cfg.ProbePeer)
			var refusal probeRefusal
			if !errors.As(err, &refusal) || string(refusal) != tc.reason || !errors.Is(err, errProbePeer) {
				t.Fatalf("wrong refusal class: %v", err)
			}
		})
	}
}

func TestUPD_D063_ProbeAddressSourcesFailClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*testing.T, *Config)
	}{
		{"interfaces", func(_ *testing.T, c *Config) {
			c.probeSources.interfaces = func() ([]net.Addr, error) { return nil, errors.New("secret interface") }
		}},
		{"lookup", func(_ *testing.T, c *Config) {
			c.probeSources.lookup = func(context.Context, string, string) ([]netip.Addr, error) {
				return nil, errors.New("secret DNS address")
			}
		}},
		{"no_answers", func(_ *testing.T, c *Config) {
			c.probeSources.lookup = func(context.Context, string, string) ([]netip.Addr, error) { return nil, nil }
		}},
		{"missing_agent_config", func(t *testing.T, c *Config) {
			if err := os.Remove(c.AgentConfig); err != nil {
				t.Fatal(err)
			}
		}},
		{"bad_agent_config", func(t *testing.T, c *Config) { writeProbeAgentConfig(t, c.AgentConfig, `{"secret`) }},
		{"missing_control", func(t *testing.T, c *Config) { writeProbeAgentConfig(t, c.AgentConfig, `{}`) }},
		{"bad_resolver", func(t *testing.T, c *Config) {
			writeProbeAgentConfig(t, c.AgentConfig, `{"control_url":"https://192.0.2.1","dns":{"resolvers":[{"admin_url":"secret","dns_addr":"secret"}]}}`)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := probeConfigFixture(t)
			cfg.ProbePeer = probeFixture()
			tc.change(t, &cfg)
			if _, err := loadProbeConfig(t, cfg); err == nil || err.Error() != errProbePeer.Error() {
				t.Fatalf("unavailable input accepted or leaked: %v", err)
			}
			cfg.ProbePeer = nil
			if _, err := loadProbeConfig(t, cfg); err != nil {
				t.Fatalf("probe-less config depends on address sources: %v", err)
			}
		})
	}
}

func TestUPD_D063_ProbeNodeAddressesIncludeEveryGlobalUnicastV4(t *testing.T) {
	cfg := probeConfigFixture(t)
	cfg.probeSources.interfaces = probeInterfaces("192.0.2.1", "10.0.0.1", "100.64.0.1", "127.0.0.1", "169.254.1.1", "2001:db8::1", "224.0.0.1")
	known, err := cfg.probeEnvironment().addresses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []netip.Addr{netip.MustParseAddr("192.0.2.1"), netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("100.64.0.1")}
	if !reflect.DeepEqual(known.node, want) {
		t.Fatal("node addresses masked, lost or not filtered")
	}
}

func TestUPD_D063_ProbeRevalidatesEachIssuance(t *testing.T) {
	cfg := probeConfigFixture(t)
	cfg.ProbePeer = probeFixture()
	nodeAddresses := []string{"192.0.2.10"}
	control := netip.MustParseAddr("192.0.2.1")
	cfg.probeSources.interfaces = func() ([]net.Addr, error) { return probeInterfaces(nodeAddresses...)() }
	cfg.probeSources.lookup = func(context.Context, string, string) ([]netip.Addr, error) {
		return []netip.Addr{control}, nil
	}
	loaded, err := loadProbeConfig(t, cfg)
	if err != nil {
		t.Fatal(err)
	}
	configured, err := loaded.Engine()
	if err != nil {
		t.Fatal(err)
	}
	defer configured.Close()
	w := newWorld(t)
	rel := w.makeRelease("probe", 1, []int{1})
	w.seed(rel)
	w.eng.ProbePeer, w.eng.probeEnv = configured.ProbePeer, configured.probeEnv
	var logs bytes.Buffer
	w.eng.Log = slog.New(slog.NewJSONHandler(&logs, nil))
	var nonce string
	issue := func(wantProbe bool) {
		t.Helper()
		if err := w.eng.issueChallenge(context.Background(), updatejournal.Txn{TxnID: uuid.Must(uuid.NewV7()).String()}, rel.id.String(), "rollback"); err != nil {
			t.Fatal("refusal aborted challenge", err)
		}
		ch, _, err := healthattest.ReadChallenge(w.layout.Journal)
		if err != nil {
			t.Fatal(err)
		}
		if (ch.Probe != nil) != wantProbe || ch.Nonce == nonce || !ch.ExpiresAt.Equal(ch.IssuedAt.Add(HealthChallengeTTL)) {
			t.Fatal("wrong probe, nonce or window after revalidation")
		}
		nonce = ch.Nonce
	}
	issue(true)
	for _, reason := range []string{"node_destination", "node_source", "control_destination", "resolver_destination", "agent_config_unavailable"} {
		nodeAddresses, control = []string{"192.0.2.10"}, netip.MustParseAddr("192.0.2.1")
		writeProbeAgentConfig(t, cfg.AgentConfig, `{"control_url":"https://control.example.invalid"}`)
		switch reason {
		case "node_destination":
			nodeAddresses = []string{"1.2.3.4"}
		case "node_source":
			nodeAddresses = []string{"100.64.8.1"}
		case "control_destination":
			control = netip.MustParseAddr("1.2.3.4")
		case "resolver_destination":
			writeProbeAgentConfig(t, cfg.AgentConfig, `{"control_url":"https://control.example.invalid","dns":{"resolvers":[{"admin_url":"http://127.0.0.1:3000","dns_addr":"1.2.3.4:53"}]}}`)
		case "agent_config_unavailable":
			writeProbeAgentConfig(t, cfg.AgentConfig, `{"secret`)
		}
		issue(false)
		issue(false)
		if strings.Count(logs.String(), `"reason":"`+reason+`"`) != 1 {
			t.Fatal("refusal was not logged once per class")
		}
	}
	writeProbeAgentConfig(t, cfg.AgentConfig, `{"control_url":"https://control.example.invalid"}`)
	issue(true)
	if !reflect.DeepEqual(w.eng.ProbePeer, cfg.ProbePeer) || strings.Count(logs.String(), "\n") != 5 {
		t.Fatal("configuration mutated or unexpected diagnostics")
	}
	for _, value := range []string{cfg.AgentConfig, "secret", "example.invalid", "1.2.3.4", "100.64.8.1", cfg.ProbePeer.PeerID} {
		if strings.Contains(logs.String(), value) {
			t.Fatal("probe diagnostic leaked private values")
		}
	}
}
