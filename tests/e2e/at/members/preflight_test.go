//go:build e2e

package members

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestATMembers_AssignedStackGate accepts the corrected slot and checks the
// actual prepared instance. This is infrastructure readiness, not an AT pass.
func TestATMembers_AssignedStackGate(t *testing.T) {
	if os.Getenv("FVPN_E2E") != "1" {
		t.Skip("set FVPN_E2E=1 through at-members")
	}
	root := repository(t)
	raw, err := os.ReadFile(filepath.Join(root, ".agent/e2e-state-w4-members/stack.json"))
	if err != nil {
		t.Fatal("start the assigned w4-members stack first")
	}
	var s stackConfig
	if json.Unmarshal(raw, &s) != nil || s.Prefix != "fvpn-w4-members" || s.PortBase != 24000 || s.HostPorts["control"] != 24443 || s.HostPorts["postgres"] != 29432 {
		t.Fatal("assigned instance must be w4-members, base 24000")
	}
	t.Log("corrected layout accepted: w4-members / 24000")
}
