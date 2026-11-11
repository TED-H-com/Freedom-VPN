package envelope

import (
	"crypto/ed25519"
	"testing"
	"time"
)

// §16.2: preflight eligibility agrees with signature verification's key rule.
func TestSEC01_HasUsableKey(t *testing.T) {
	now := time.Now().UTC()
	key := func(id string, purpose Purpose) TrustedKey {
		t.Helper()
		pub, _, err := ed25519.GenerateKey(nil)
		if err != nil {
			t.Fatal(err)
		}
		return TrustedKey{
			KeyID: id, PublicKey: pub, Purposes: []Purpose{purpose},
			NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		}
	}
	valid := key("desired", PurposeDesiredState)
	revoked, expired, future, atStart := valid, valid, valid, valid
	revoked.Revoked = true
	expired.NotAfter = now
	future.NotBefore = now.Add(time.Nanosecond)
	atStart.NotBefore = now
	release := key("release", PurposeRelease)
	for _, tt := range []struct {
		name string
		keys []TrustedKey
		want bool
	}{
		{name: "empty"},
		{name: "release_only", keys: []TrustedKey{release}},
		{name: "revoked", keys: []TrustedKey{revoked}},
		{name: "expired_at_not_after", keys: []TrustedKey{expired}},
		{name: "not_yet_valid", keys: []TrustedKey{future}},
		{name: "at_not_before", keys: []TrustedKey{atStart}, want: true},
		{name: "valid", keys: []TrustedKey{valid}, want: true},
		{name: "mixed_valid", keys: []TrustedKey{release, valid, key("lease", PurposeLease), key("recovery", PurposeRecovery)}, want: true},
		{name: "mixed_unusable", keys: []TrustedKey{release, revoked}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewNodeTrustStore(tt.keys...)
			if err != nil {
				t.Fatal(err)
			}
			if got := store.HasUsableKey(PurposeDesiredState, now); got != tt.want {
				t.Fatalf("HasUsableKey(desired-state) = %v, want %v", got, tt.want)
			}
			if store.HasUsableKey(Purpose("unknown"), now) {
				t.Fatal("unknown purpose was usable")
			}
		})
	}
}
