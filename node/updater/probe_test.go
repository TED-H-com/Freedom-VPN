package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func probeFixture() *healthattest.ChallengeProbe {
	// TEST ONLY: parser/serialization fixture, never dialed. A documentation
	// destination cannot be the positive case because production refuses it.
	return &healthattest.ChallengeProbe{
		PeerID: "probe-private-marker", SourceV4: "100.64.8.1/32",
		Destinations: []healthattest.ChallengeProbeDest{{AddrV4: "1.2.3.4/32", Proto: "tcp", Port: 443}},
		ResolverDNS:  true,
	}
}

func probeConfigFixture(t *testing.T) Config {
	t.Helper()
	dir := t.TempDir()
	trust := filepath.Join(dir, "trust.json")
	writeTrust(t, trust)
	return Config{Layout: testLayout(dir), NodeID: uuid.Must(uuid.NewV7()), IncarnationID: uuid.Must(uuid.NewV7()),
		BootID: uuid.Must(uuid.NewV7()), Platform: testPlatform(), ReleaseTrust: trust, DesiredTrust: trust,
		DesiredState: filepath.Join(dir, "desired.json"), MeterCheckpoint: filepath.Join(dir, "meter.json")}
}

func loadProbeConfig(t *testing.T, cfg Config) (Config, error) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "updater.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return LoadConfig(path)
}

func TestUPD_D063_ProbeConfigIssueAndNonce(t *testing.T) {
	for _, count := range []int{0, 1, 4} {
		t.Run(string(rune('0'+count)), func(t *testing.T) {
			cfg := probeConfigFixture(t)
			if count > 0 {
				cfg.ProbePeer = probeFixture()
				for i := 1; i < count; i++ {
					cfg.ProbePeer.Destinations = append(cfg.ProbePeer.Destinations, healthattest.ChallengeProbeDest{
						AddrV4: "1.2.3.4/32", Proto: "udp", Port: uint16(8000 + i)})
				}
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
			want, _ := json.Marshal(cfg.ProbePeer)
			if loaded.ProbePeer != nil {
				loaded.ProbePeer.Destinations[0].Port++
			}
			w := newWorld(t)
			rel := w.makeRelease("probe", 1, []int{1})
			w.seed(rel)
			w.eng.ProbePeer = configured.ProbePeer
			nc := &NodeChecks{Journal: w.layout.Journal, Node: w.node, Now: w.eng.now, rt: &checkRT{}}
			w.eng.Health = nc
			var priorNonce string
			for _, phase := range []string{"candidate", "rollback"} {
				if err := w.eng.issueChallenge(context.Background(), updatejournal.Txn{TxnID: uuid.Must(uuid.NewV7()).String()}, rel.id.String(), phase); err != nil {
					t.Fatal(err)
				}
				ch, _, err := healthattest.ReadChallenge(w.layout.Journal)
				if err != nil {
					t.Fatal(err)
				}
				got, _ := json.Marshal(ch.Probe)
				if !bytes.Equal(got, want) || ch.Nonce == priorNonce || !ch.IssuedAt.Equal(w.now) ||
					!ch.ExpiresAt.Equal(ch.IssuedAt.Add(HealthChallengeTTL)) || HealthChallengeTTL > 300*time.Second {
					t.Fatal("challenge did not preserve configured probe, fresh nonce and 45s window")
				}
				priorNonce = ch.Nonce
				raw, _ := os.ReadFile(filepath.Join(w.layout.Journal, healthattest.ChallengeFile))
				if bytes.Contains(raw, []byte(`"probe":`)) != (count != 0) {
					t.Fatal("omitted probe shape changed")
				}
				at := goodAttest(ch, api.PublicationIntentUnpublished)
				if err := healthattest.WriteAttestation(w.layout.Journal, at); err != nil {
					t.Fatal(err)
				}
				if reason, answer := nc.readOnce(ch, healthattest.Observed{}); reason != "" || answer == nil || answer.Checks.TunnelProbe.Status != api.HealthCheckStatusPassed {
					t.Fatal("bound probe answer refused", reason)
				}
				at.Nonce = strings.Repeat("f", 64)
				if err := healthattest.WriteAttestation(w.layout.Journal, at); err != nil {
					t.Fatal(err)
				}
				if reason, answer := nc.readOnce(ch, healthattest.Observed{}); reason != healthattest.ReasonNonceMismatch || answer != nil {
					t.Fatal("passed probe answer bypassed nonce", reason)
				}
				if ch.Probe != nil {
					changed := ch
					changed.Probe = cloneProbePeer(ch.Probe)
					changed.Probe.Destinations[0].Port++
					if err := healthattest.WriteChallenge(w.layout.Journal, changed); err != nil {
						t.Fatal(err)
					}
					if reason, answer := nc.readOnce(ch, healthattest.Observed{}); reason != healthattest.ReasonChallengeReplaced || answer != nil {
						t.Fatal("changed probe matched bound challenge", reason)
					}
				}
			}
		})
	}
}

func TestUPD_D063_ProbeConfigRefusalsAreBounded(t *testing.T) {
	cfg := probeConfigFixture(t)
	cases := []struct {
		name   string
		mutate func(*healthattest.ChallengeProbe)
	}{
		{"empty_peer", func(p *healthattest.ChallengeProbe) { p.PeerID = "" }},
		{"long_peer", func(p *healthattest.ChallengeProbe) { p.PeerID = strings.Repeat("secret", 30) }},
		{"control_peer", func(p *healthattest.ChallengeProbe) { p.PeerID += "\n" }},
		{"wide_source", func(p *healthattest.ChallengeProbe) { p.SourceV4 = "100.64.8.0/24" }},
		{"bare_source", func(p *healthattest.ChallengeProbe) { p.SourceV4 = "100.64.8.1" }},
		{"v6_source", func(p *healthattest.ChallengeProbe) { p.SourceV4 = "2001:db8::1/128" }},
		{"loopback_source", func(p *healthattest.ChallengeProbe) { p.SourceV4 = "127.0.0.1/32" }},
		{"no_destinations", func(p *healthattest.ChallengeProbe) { p.Destinations = nil }},
		{"five_destinations", func(p *healthattest.ChallengeProbe) {
			p.Destinations = append(p.Destinations, p.Destinations[0], p.Destinations[0], p.Destinations[0], p.Destinations[0])
		}},
		{"wide_destination", func(p *healthattest.ChallengeProbe) { p.Destinations[0].AddrV4 = "1.2.3.0/24" }},
		{"bare_destination", func(p *healthattest.ChallengeProbe) { p.Destinations[0].AddrV4 = "1.2.3.4" }},
		{"v6_destination", func(p *healthattest.ChallengeProbe) { p.Destinations[0].AddrV4 = "2001:db8::1/128" }},
		{"zero_port", func(p *healthattest.ChallengeProbe) { p.Destinations[0].Port = 0 }},
		{"unknown_proto", func(p *healthattest.ChallengeProbe) { p.Destinations[0].Proto = "icmp" }},
		{"upper_proto", func(p *healthattest.ChallengeProbe) { p.Destinations[0].Proto = "TCP" }},
	}
	for _, prefix := range enforce.SpecDeniedV4() {
		addr := prefix.Addr().String() + "/32"
		cases = append(cases, struct {
			name   string
			mutate func(*healthattest.ChallengeProbe)
		}{
			"denied_" + prefix.String(), func(p *healthattest.ChallengeProbe) { p.Destinations[0].AddrV4 = addr }})
	}
	cases = append(cases, struct {
		name   string
		mutate func(*healthattest.ChallengeProbe)
	}{
		"metadata", func(p *healthattest.ChallengeProbe) { p.Destinations[0].AddrV4 = "169.254.169.254/32" }})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg.ProbePeer = probeFixture()
			tc.mutate(cfg.ProbePeer)
			if _, err := loadProbeConfig(t, cfg); err == nil || err.Error() != errProbePeer.Error() || len(err.Error()) > 64 {
				t.Fatal("unsafe configuration accepted or diagnostic leaked", err)
			}
			if _, err := cfg.Engine(); err == nil || err.Error() != errProbePeer.Error() {
				t.Fatal("programmatic config bypassed validation", err)
			}
		})
	}
}

func TestUPD_D063_MalformedProbeJSONDoesNotLeak(t *testing.T) {
	cfg := probeConfigFixture(t)
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, block := range []string{
		`{"peer_id":"probe-private-marker","source_v4":"100.64.8.1/32","destinations":[{"addr_v4":"1.2.3.4/32","proto":"tcp","port":65536}]}`,
		`{"peer_id":"probe-private-marker","100.64.8.1/32-secret":true}`,
		`{"peer_id":"probe-private-marker","resolver_dns":"100.64.8.1/32"}`,
		`{"peer_id":`,
	} {
		body := append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"probe_peer":`+block+`}`)...)
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadConfig(path); err == nil || err.Error() != "updater: invalid config JSON" {
			t.Fatal("decoder leaked config values or accepted malformed probe", err)
		}
	}
}
