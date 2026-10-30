package netlab

import (
	"encoding/binary"
	"net/netip"
	"testing"
)

func TestMechanismForLinkType(t *testing.T) {
	got, err := mechanismFor(LinkEther)
	if err != nil || got != MechanismVethMirred {
		t.Fatalf("ether: %q %v", got, err)
	}
	got, err = mechanismFor(LinkNone)
	if err != nil || got != MechanismTunMirred {
		t.Fatalf("none: %q %v", got, err)
	}
	if _, err := mechanismFor(768); err == nil {
		t.Fatal("ARPHRD_TUNNEL selected a transport")
	}
}

func TestUDPPacketChecksum(t *testing.T) {
	src := netip.MustParseAddr("100.64.9.9")
	dst := netip.MustParseAddr("203.0.113.10")
	pkt := udpPacket(src, dst, IngressSourcePort, 9, []byte("x"))
	if len(pkt) != 29 {
		t.Fatalf("len %d", len(pkt))
	}
	if pkt[0]>>4 != 4 || pkt[9] != 17 {
		t.Fatalf("header %x", pkt[:12])
	}
	if binary.BigEndian.Uint16(pkt[2:4]) != 29 {
		t.Fatalf("total length %d", binary.BigEndian.Uint16(pkt[2:4]))
	}
	if pkt[26] != 0 || pkt[27] != 0 {
		t.Fatalf("udp checksum %x", pkt[26:28])
	}
	sum := ipChecksum(pkt[:20])
	if sum != 0 && sum != 0xffff {
		t.Fatalf("ipv4 checksum residue %x", sum)
	}
	frame := ethernetFrame([]byte{2, 0, 0, 0, 0, 1}, []byte{2, 0, 0, 0, 0, 2}, pkt)
	if len(frame) != 60 || frame[12] != 0x08 || frame[13] != 0x00 {
		t.Fatalf("frame len %d type %x", len(frame), frame[12:14])
	}
}
