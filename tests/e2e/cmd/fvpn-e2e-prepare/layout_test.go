package main

import (
	"fmt"
	"testing"
)

func TestENR01_LowerSlotsPreserveExistingBindings(t *testing.T) {
	usedPorts, labs, origins := map[int]bool{}, map[string]bool{}, map[string]bool{}
	for _, tc := range []struct{ base, subnet int }{
		{18000, 0}, {24000, 1}, {30000, 2}, {36000, 3},
		{42000, 4}, {48000, 5}, {54000, 6}, {60000, 7},
		{12000, 8}, {6000, 9},
	} {
		got, err := parseLayout("w4-enroll", tc.base)
		if err != nil {
			t.Fatal(err)
		}
		if got.LabPrefix != fmt.Sprintf("172.28.%d", 30+tc.subnet) || got.OriginPrefix != fmt.Sprintf("192.0.%d", 3+tc.subnet) || labs[got.LabPrefix] || origins[got.OriginPrefix] {
			t.Fatalf("changed or colliding subnet: %+v", got)
		}
		labs[got.LabPrefix], origins[got.OriginPrefix] = true, true
		for name, offset := range portOffsets {
			port := got.Ports[name]
			if port != tc.base+offset || port > 65535 || usedPorts[port] {
				t.Fatalf("invalid or colliding %s port %d", name, port)
			}
			usedPorts[port] = true
		}
	}
	for _, base := range []int{0, 5999, 7000, 66000} {
		if _, err := parseLayout("e2e", base); err == nil {
			t.Fatalf("accepted invalid base %d", base)
		}
	}
}

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
