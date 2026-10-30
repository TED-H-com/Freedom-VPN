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
	if packetDirection(p) != "to_origin" {
		t.Fatal("controlled request missing")
	}
	for n := 0; n < len(p); n++ {
		if packetDirection(p[:n]) != "" {
			t.Fatal("truncated packet admitted")
		}
	}
	p[16] = 203
	if packetDirection(p) != "" {
		t.Fatal("unrelated destination admitted")
	}
	p[16] = 192
	p[20] = 0
	if packetDirection(p) != "" {
		t.Fatal("other connection admitted")
	}
}
