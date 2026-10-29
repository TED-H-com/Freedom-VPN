package main

import "testing"

func TestParseLayoutDefaultAndSide(t *testing.T) {
	def, err := parseLayout("e2e", 18000)
	if err != nil {
		t.Fatal(err)
	}
	if def.LabPrefix != "172.28.30" || def.OriginPrefix != "192.0.3" || def.OriginIP != "192.0.3.10" {
		t.Fatalf("default layout: %+v", def)
	}
	if def.Ports["control"] != 18443 || def.Ports["agent"] != 19443 || def.Ports["postgres"] != 23432 || def.Ports["gateway"] != 18480 {
		t.Fatalf("default ports: %+v", def.Ports)
	}
	side, err := parseLayout("side", 24000)
	if err != nil {
		t.Fatal(err)
	}
	if side.LabPrefix != "172.28.31" || side.OriginPrefix != "192.0.4" || side.Ports["control"] != 24443 {
		t.Fatalf("side layout: %+v", side)
	}
	if _, err := parseLayout("e2e", 19000); err == nil {
		t.Fatal("19000 collides with the default agent port and must be rejected")
	}
	if _, err := parseLayout("E2E", 18000); err == nil {
		t.Fatal("uppercase prefix")
	}
}
