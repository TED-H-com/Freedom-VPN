package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/google/uuid"
)

// TEST ONLY isolated Docker origin. The product validator must accept it
// unchanged and must still reject a documentation-range destination.
func TestD063_LabProbeKeepsTheProductDestinationGuard(t *testing.T) {
	for _, tc := range []struct {
		address  string
		accepted bool
	}{{"192.0.3.10/32", true}, {"192.0.2.10/32", false}, {"100.64.1.10/32", false}} {
		t.Run(tc.address, func(t *testing.T) {
			cfg := map[string]any{"node_id": uuid.NewString(), "incarnation_id": uuid.NewString(), "boot_id": uuid.NewString(), "platform": map[string]string{"os_id": "ubuntu", "os_version": "24.04", "architecture": "amd64", "updater_version": "1.0.0"}, "release_trust": "test", "desired_trust": "test", "desired_state": "test", "meter_checkpoint": "test", "probe_peer": map[string]any{"peer_id": "TEST ONLY synthetic peer", "source_v4": "100.64.8.1/32", "destinations": []any{map[string]any{"addr_v4": tc.address, "proto": "udp", "port": 4443}}}}
			raw, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "updater.json")
			if err = os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			_, err = updater.LoadConfig(path)
			if (err == nil) != tc.accepted {
				t.Fatalf("accepted=%t error=%v", tc.accepted, err)
			}
		})
	}
}
