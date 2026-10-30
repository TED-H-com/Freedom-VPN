package main

import (
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/probeauth"
	"github.com/google/uuid"
	"testing"
)

func TestProbeOriginRequiresAuthenticationAndCurrentSlot(t *testing.T) {
	key := []byte("TEST ONLY - NOT A SECRET")
	id := uuid.New()
	p := &trafficProbe{id: id.String()}
	raw := probeauth.Seal(key, id, false)
	reply := p.datagram(key, raw, "192.0.2.10")
	if _, ok := probeauth.Open(key, reply, true); !ok || p.count != 1 {
		t.Fatal("authenticated echo")
	}
	raw[len(raw)-1] ^= 1
	if p.datagram(key, raw, "192.0.2.10") != nil || p.count != 1 {
		t.Fatal("unauthenticated receipt")
	}
	if p.datagram(key, probeauth.Seal(key, uuid.New(), false), "192.0.2.10") != nil || p.count != 1 {
		t.Fatal("wrong slot")
	}
}
