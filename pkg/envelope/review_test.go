package envelope

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestReviewC01_TrustJSONCannotUnrevoke(t *testing.T) {
	key := trusted(loadTestKeys(t)["release"])
	key.Revoked = true
	raw := mustJSON(t, mustStore(t, key))
	// Exact un-revoke member sequence from the review: revoked:true,revoked:false.
	attack := bytes.Replace(raw, []byte(`"revoked":true`), []byte(`"revoked":true,"revoked":false`), 1)
	if bytes.Equal(raw, attack) {
		t.Fatal("fixture did not change")
	}
	if _, err := ParseTrustStoreJSON(attack); !errors.Is(err, canonjson.ErrDuplicateKey) {
		t.Fatalf("un-revoke accepted: %v", err)
	}
	for _, suffix := range []string{" trailing", " {}"} {
		if _, err := ParseTrustStoreJSON(append(bytes.Clone(raw), suffix...)); !errors.Is(err, canonjson.ErrSyntax) {
			t.Fatalf("trailing data accepted: %v", err)
		}
	}
	ts, err := ParseTrustStoreJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := ts.Lookup(key.KeyID); !got.Revoked {
		t.Fatal("revocation lost")
	}
}

func TestReviewC01_PublicKeyCannotHaveLiveAlias(t *testing.T) {
	old := trusted(loadTestKeys(t)["release"])
	old.Revoked = true
	live := old
	live.KeyID = "live-alias"
	live.Revoked = false
	if _, err := NewTrustStore(old, live); err == nil {
		t.Fatal("revoked public key accepted under a live alias")
	}
}

func TestReviewC01_DuplicateSignatureKeyIDs(t *testing.T) {
	key := loadTestKeys(t)["release"]
	ts := mustStore(t, trusted(key))
	env, err := Sign(doc, key.priv, key.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	if err := env.AddSignature(key.priv, key.KeyID); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("duplicate signing: %v", err)
	}
	for _, id := range []string{key.KeyID, "unknown"} {
		e := env
		sig := env.Signatures[0]
		sig.KeyID = id
		e.Signatures = append(append([]Signature(nil), env.Signatures...), sig, sig)
		if _, _, err := Verify(mustJSON(t, e), ts, PurposeRelease, nowTest); !errors.Is(err, ErrSignatureInvalid) {
			t.Fatalf("duplicate %s accepted: %v", id, err)
		}
	}
}

func TestReviewC10_ExactlyOnePurpose(t *testing.T) {
	key := trusted(loadTestKeys(t)["release"])
	for _, purposes := range [][]Purpose{nil, {PurposeRelease, PurposeLease}, {PurposeRelease, PurposeRelease}} {
		key.Purposes = purposes
		if _, err := NewTrustStore(key); err == nil {
			t.Fatalf("accepted purposes %v", purposes)
		}
	}
	key.Purposes = []Purpose{PurposeRelease}
	if _, err := NewTrustStore(key); err != nil {
		t.Fatal(err)
	}
}

func TestReviewC14_VerifyDoesNotExposeDuplicateMember(t *testing.T) {
	ts := mustStore(t, trusted(loadTestKeys(t)["release"]))
	_, _, err := Verify([]byte(`{"signed":{"secret_field":1,"secret_field":2},"signatures":[]}`), ts, PurposeRelease, nowTest)
	if !errors.Is(err, ErrSignatureInvalid) || strings.Contains(err.Error(), "secret_field") {
		t.Fatalf("unsafe error: %v", err)
	}
}
