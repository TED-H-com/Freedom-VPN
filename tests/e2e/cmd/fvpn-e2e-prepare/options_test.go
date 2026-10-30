package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
)

func TestAT29_PrepareGraceAndExactGatewayTrust(t *testing.T) {
	lay, err := parseLayout("e2e", 18000)
	if err != nil {
		t.Fatal(err)
	}
	for _, grace := range []int{0, 90} {
		state := filepath.Join(t.TempDir(), "state")
		if err := run(state, true, lay, grace); err != nil {
			t.Fatal(err)
		}
		members, err := config.LoadMembers(filepath.Join(state, "config", "members.json"))
		if err != nil || members.GraceSeconds == nil || *members.GraceSeconds != grace || (grace > 0 && members.GraceReason == "") {
			t.Fatalf("grace %d not accepted by product config: %v", grace, err)
		}
		raw, err := os.ReadFile(filepath.Join(state, "config", "central.json"))
		if err != nil {
			t.Fatal(err)
		}
		central := config.Defaults()
		if json.Unmarshal(raw, &central) != nil || len(central.TrustedProxies) != 1 || central.TrustedProxies[0] != "172.28.30.21/32" {
			t.Fatal("gateway must be the only trusted hop")
		}
		if _, err := config.ParseTrustedProxies(central.TrustedProxies); err != nil {
			t.Fatal(err)
		}
	}
	for _, invalid := range []int{-1, 31622401} {
		if err := run(filepath.Join(t.TempDir(), "state"), true, lay, invalid); err == nil {
			t.Fatalf("accepted invalid grace %d", invalid)
		}
	}
}
