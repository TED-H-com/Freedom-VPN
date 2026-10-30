package main

import (
	"encoding/json"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

// TEST ONLY isolated Docker origin. LoadConfig validates against the actual
// interface inventory and the agent's control/resolver addresses (#154).
// Literal endpoints exercise that contract without DNS or network traffic.
func TestD063_LabProbeKeepsTheProductDestinationGuard(t *testing.T) {
	interfaces, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	local := make(map[netip.Addr]bool)
	for _, addr := range interfaces {
		if prefix, err := netip.ParsePrefix(addr.String()); err == nil {
			local[prefix.Addr().Unmap()] = true
		}
	}
	source := netip.MustParseAddr("100.64.8.1")
	for local[source] {
		source = source.Next()
	}
	for _, tc := range []struct {
		name, destination, control, resolver string
		missingAgent, accepted               bool
	}{
		{name: "lab_origin", destination: "192.0.3.10/32", accepted: true},
		{name: "documentation_destination", destination: "192.0.2.10/32"},
		{name: "overlay_destination", destination: "100.64.1.10/32"},
		{name: "missing_agent_config", destination: "192.0.3.10/32", missingAgent: true},
		{name: "control_collision", destination: "192.0.3.10/32", control: "https://192.0.3.10:9443"},
		{name: "resolver_collision", destination: "192.0.3.10/32", resolver: "192.0.3.10:53"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			agentPath := filepath.Join(dir, "agent-run.json")
			control, resolver := tc.control, tc.resolver
			if control == "" {
				control = "https://192.0.2.1:9443"
			}
			if resolver == "" {
				resolver = "127.0.0.1:53"
			}
			write := func(path string, value any) {
				t.Helper()
				raw, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if !tc.missingAgent {
				write(agentPath, map[string]any{
					"control_url": control,
					"dns": map[string]any{"resolvers": []any{map[string]string{
						"id": "lab", "admin_url": "http://127.0.0.1:8080", "dns_addr": resolver,
					}}},
				})
			}
			path := filepath.Join(dir, "updater.json")
			write(path, map[string]any{
				"node_id": uuid.NewString(), "incarnation_id": uuid.NewString(), "boot_id": uuid.NewString(),
				"platform":      map[string]string{"os_id": "ubuntu", "os_version": "24.04", "architecture": "amd64", "updater_version": "1.0.0"},
				"release_trust": "test", "desired_trust": "test", "desired_state": "test", "meter_checkpoint": "test",
				"agent_config": agentPath, "result_url": "https://192.0.2.2:9443/api/v1/agent/update-results",
				"probe_peer": map[string]any{
					"peer_id": "TEST ONLY synthetic peer", "source_v4": source.String() + "/32",
					"destinations": []any{map[string]any{"addr_v4": tc.destination, "proto": "udp", "port": 4443}},
				},
			})
			loaded, err := updater.LoadConfig(path)
			if (err == nil) != tc.accepted {
				t.Fatalf("accepted=%t error=%v", tc.accepted, err)
			}
			if err != nil && err.Error() != "updater: invalid probe_peer" {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if tc.accepted && (loaded.ProbePeer == nil || loaded.AgentConfig != agentPath) {
				t.Fatal("accepted configuration lost its probe or agent configuration")
			}
		})
	}
}
