package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestJudgeProbe(t *testing.T) {
	up := nbStatus{}
	up.Management.Connected = true
	up.Relays.Total = 1
	up.Relays.Available = 1
	partial := up
	partial.Relays.Available = 0
	direct := up
	direct.Relays.Total = 0
	direct.Relays.Available = 0
	down := up
	down.Management.Connected = false

	cases := []struct {
		name                    string
		published, egress, egOK bool
		member, statusOK        bool
		st                      nbStatus
		health                  api.DataHealth
		check                   api.HealthCheckStatus
	}{
		{"overlay up", false, false, false, false, true, up, api.DataHealthHealthy, api.HealthCheckStatusPassed},
		{"direct only", false, false, false, false, true, direct, api.DataHealthHealthy, api.HealthCheckStatusPassed},
		{"relay partial", false, false, false, false, true, partial, api.DataHealthDegraded, api.HealthCheckStatusPassed},
		{"management down", false, false, false, false, true, down, api.DataHealthFailed, api.HealthCheckStatusFailed},
		{"status unreadable", false, false, false, false, false, up, api.DataHealthFailed, api.HealthCheckStatusFailed},
		{"unpublished ignores members", false, false, false, false, true, up, api.DataHealthHealthy, api.HealthCheckStatusPassed},
		{"published members up", true, false, false, true, true, up, api.DataHealthHealthy, api.HealthCheckStatusPassed},
		{"published members missing", true, false, false, false, true, up, api.DataHealthDegraded, api.HealthCheckStatusFailed},
		{"egress up", false, true, true, false, true, up, api.DataHealthHealthy, api.HealthCheckStatusPassed},
		{"egress down", false, true, false, false, true, up, api.DataHealthDegraded, api.HealthCheckStatusFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			health, check := judgeProbe(tc.published, tc.egress, tc.egOK, tc.member, tc.st, tc.statusOK)
			if health != tc.health || check != tc.check {
				t.Fatalf("health %s check %s", health, check)
			}
		})
	}
}

func TestNetbirdProber(t *testing.T) {
	good := []byte(`{"management":{"url":"https://netbird.example.invalid","connected":true},"relays":{"total":1,"available":1,"details":[{"uri":"https://netbird.example.invalid","available":true}]},"publicKey":"not-a-log-line"}`)
	t.Run("status file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "status.json")
		if err := os.WriteFile(path, good, 0o644); err != nil {
			t.Fatal(err)
		}
		p := netbirdProber{
			statusFile: path,
			run: func(context.Context) ([]byte, error) {
				t.Fatal("status file still ran the CLI")
				return nil, errors.New("cli")
			},
		}
		res := p.Probe(context.Background(), ProbeRequest{})
		if !res.Ran || res.Check != api.HealthCheckStatusPassed || res.Health != api.DataHealthHealthy || res.Detail != "overlay_up" {
			t.Fatalf("%+v", res)
		}
		if strings.Contains(res.Detail, "not-a-log-line") || strings.Contains(res.Detail, "example.invalid") {
			t.Fatalf("detail leaked status: %s", res.Detail)
		}
	})

	t.Run("management down does not dial", func(t *testing.T) {
		p := netbirdProber{
			run: func(context.Context) ([]byte, error) {
				return []byte(`{"management":{"connected":false},"relays":{"total":1,"available":0}}`), nil
			},
			dial: func(context.Context, string) error {
				t.Fatal("dialed while management is down")
				return nil
			},
		}
		res := p.Probe(context.Background(), ProbeRequest{Egress: "192.0.2.10:443", Published: true})
		if !res.Ran || res.Health != api.DataHealthFailed || res.Check != api.HealthCheckStatusFailed {
			t.Fatalf("%+v", res)
		}
	})

	t.Run("egress miss", func(t *testing.T) {
		p := netbirdProber{
			run:  func(context.Context) ([]byte, error) { return good, nil },
			dial: func(context.Context, string) error { return errors.New("refused") },
		}
		res := p.Probe(context.Background(), ProbeRequest{Egress: "192.0.2.10:443"})
		if res.Health != api.DataHealthDegraded || res.Check != api.HealthCheckStatusFailed || res.Detail != "overlay_partial" {
			t.Fatalf("%+v", res)
		}
	})

	t.Run("unpublished skips member interfaces", func(t *testing.T) {
		p := netbirdProber{
			run: func(context.Context) ([]byte, error) { return good, nil },
			iface: func(string) error {
				t.Fatal("unpublished probe looked up a member interface")
				return errors.New("down")
			},
		}
		res := p.Probe(context.Background(), ProbeRequest{Published: false, Ingress: "fv-fh", Uplink: "fv-ih"})
		if res.Health != api.DataHealthHealthy || res.Check != api.HealthCheckStatusPassed {
			t.Fatalf("%+v", res)
		}
	})

	t.Run("published member path", func(t *testing.T) {
		p := netbirdProber{
			run:   func(context.Context) ([]byte, error) { return good, nil },
			iface: func(string) error { return errors.New("absent") },
		}
		res := p.Probe(context.Background(), ProbeRequest{Published: true, Ingress: "fv-fh", Uplink: "fv-ih"})
		if res.Health != api.DataHealthDegraded || res.Check != api.HealthCheckStatusFailed {
			t.Fatalf("missing interfaces %+v", res)
		}
		p.iface = func(string) error { return nil }
		res = p.Probe(context.Background(), ProbeRequest{Published: true, Ingress: "fv-fh", Uplink: "fv-ih"})
		if res.Health != api.DataHealthHealthy || res.Check != api.HealthCheckStatusPassed {
			t.Fatalf("interfaces up %+v", res)
		}
		res = p.Probe(context.Background(), ProbeRequest{Published: true, Ingress: "fv-fh", Uplink: "fv-fh"})
		if res.Check != api.HealthCheckStatusFailed {
			t.Fatalf("same interface was accepted %+v", res)
		}
	})

	t.Run("unreadable status is a completed failure", func(t *testing.T) {
		p := netbirdProber{run: func(context.Context) ([]byte, error) { return nil, errors.New("exit") }}
		res := p.Probe(context.Background(), ProbeRequest{})
		if !res.Ran || res.Health != api.DataHealthFailed || res.Check != api.HealthCheckStatusFailed || res.Detail != "overlay_down" {
			t.Fatalf("%+v", res)
		}
	})
}

func TestEgressProbeConfig(t *testing.T) {
	dir := t.TempDir()
	write := func(body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, "agent.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := `{
		"state_dir": "` + dir + `",
		"control_url": "https://control.example.invalid",
		"server_ca_file": "/etc/vpn-node/ca.pem",
		"trust_bundle": "/etc/vpn-node/trust.json",
		"datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release": "0.0.0-test",
		"netbird_backend": "kernel",
		"host_reserve_bps": 1,
		"netbird_status_file": "` + filepath.Join(dir, "status.json") + `",
		"egress_probe_target": "192.0.2.10:443"
	}`
	write(base)
	cfg, err := LoadConfig(filepath.Join(dir, "agent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EgressProbeTarget != "192.0.2.10:443" || cfg.NetbirdStatusFile == "" {
		t.Fatalf("%+v", cfg)
	}
	write(strings.Replace(base, "192.0.2.10:443", "192.0.2.10", 1))
	if _, err := LoadConfig(filepath.Join(dir, "agent.json")); err == nil || !strings.Contains(err.Error(), "egress_probe_target") {
		t.Fatalf("bare host: %v", err)
	}
	write(strings.Replace(base, `"egress_probe_target": "192.0.2.10:443"`, `"nope": true`, 1))
	if _, err := LoadConfig(filepath.Join(dir, "agent.json")); err == nil {
		t.Fatal("unknown field was accepted")
	}
}
