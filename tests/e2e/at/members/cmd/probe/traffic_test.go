//go:build linux

package main

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestAT02PacketObserverScope(t *testing.T) {
	p := make([]byte, 40)
	p[0] = 0x45
	p[9] = 6
	copy(p[12:16], net.ParseIP("100.64.0.1").To4())
	copy(p[16:20], net.ParseIP(controlledOrigin).To4())
	binary.BigEndian.PutUint16(p[20:22], probePort)
	binary.BigEndian.PutUint16(p[22:24], 443)
	if packetDirection(p, probePort) != "to_origin" {
		t.Fatal("controlled request missing")
	}
	if packetDirection(p, 20001) != "" {
		t.Fatal("previous transfer admitted into current port observation")
	}
	binary.BigEndian.PutUint16(p[20:22], 20001)
	if packetDirection(p, 20001) != "to_origin" {
		t.Fatal("selected per-transfer port missing")
	}
	binary.BigEndian.PutUint16(p[20:22], probePort)
	for n := 0; n < len(p); n++ {
		if packetDirection(p[:n], probePort) != "" {
			t.Fatal("truncated packet admitted")
		}
	}
	p[16] = 203
	if packetDirection(p, probePort) != "" {
		t.Fatal("unrelated destination admitted")
	}
	p[16] = 192
	p[20] = 0
	if packetDirection(p, probePort) != "" {
		t.Fatal("other connection admitted")
	}
}
