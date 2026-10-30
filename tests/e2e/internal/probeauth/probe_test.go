package probeauth

import (
	"github.com/google/uuid"
	"testing"
)

func TestAuthenticatedProbeRejectsTamperAndReflection(t *testing.T) {
	key := []byte("TEST ONLY - NOT A SECRET")
	id := uuid.New()
	raw := Seal(key, id, false)
	got, ok := Open(key, raw, false)
	if !ok || got != id {
		t.Fatal("request")
	}
	if _, ok = Open(key, raw, true); ok {
		t.Fatal("request reflected as reply")
	}
	raw[len(raw)-1] ^= 1
	if _, ok = Open(key, raw, false); ok {
		t.Fatal("tampered MAC accepted")
	}
	if _, ok = Open(key, raw[:12], false); ok {
		t.Fatal("truncated packet accepted")
	}
	reply := Seal(key, id, true)
	if got, ok = Open(key, reply, true); !ok || got != id {
		t.Fatal("reply")
	}
}
