package envelope

import (
	"crypto/ed25519"
	"errors"
	"testing"
)

func TestDATA01_LedgerEpochPurposeIsCentralOnly(t *testing.T) {
	// Generate an independent key at test time; do not add it to node vectors.
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	key := TrustedKey{KeyID: "test-ledger-epoch", PublicKey: pub, Purposes: []Purpose{PurposeLedgerEpoch}, NotBefore: t0, NotAfter: t1}
	central := mustStore(t, key)
	central, err = ParseTrustStoreJSON(mustJSON(t, central))
	if err != nil {
		t.Fatal(err)
	}
	env, err := Sign(doc, priv, key.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	raw := mustJSON(t, env)
	if _, _, err := Verify(raw, central, PurposeLedgerEpoch, nowTest); err != nil {
		t.Fatal(err)
	}
	for _, purpose := range []Purpose{PurposeRecovery, PurposeLease, PurposeDesiredState, PurposeRelease} {
		if _, _, err := Verify(raw, central, purpose, nowTest); !errors.Is(err, ErrKeyUntrusted) {
			t.Fatalf("purpose %s: %v", purpose, err)
		}
	}
	if _, err := NewNodeTrustStore(key); err == nil {
		t.Fatal("ledger epoch key entered node bundle")
	}
	if _, err := ParseNodeTrustStoreJSON(mustJSON(t, central)); err == nil {
		t.Fatal("ledger epoch JSON entered node bundle")
	}
	key.Revoked = true
	if _, _, err := Verify(raw, mustStore(t, key), PurposeLedgerEpoch, nowTest); !errors.Is(err, ErrKeyUntrusted) {
		t.Fatalf("revoked: %v", err)
	}
	recovery := trusted(loadTestKeys(t)["recovery"])
	node, err := NewNodeTrustStore(recovery)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseNodeTrustStoreJSON(mustJSON(t, node)); err != nil {
		t.Fatal(err)
	}
	recovery.Purposes = []Purpose{PurposeLedgerEpoch, PurposeRecovery}
	if _, err := NewNodeTrustStore(recovery); err == nil {
		t.Fatal("multipurpose node key accepted")
	}
	if _, err := ParseNodeTrustStoreJSON([]byte(`{`)); err == nil {
		t.Fatal("bad JSON accepted")
	}
}
