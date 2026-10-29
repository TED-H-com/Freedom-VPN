package envelope

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

type testKey struct {
	Label   string  `json:"label"`
	KeyID   string  `json:"key_id"`
	Purpose Purpose `json:"purpose"`
	SeedHex string  `json:"seed_hex"`
	priv    ed25519.PrivateKey
}

func loadTestKeys(t *testing.T) map[string]*testKey {
	t.Helper()
	raw, err := os.ReadFile("testdata/test_keys.json")
	if err != nil {
		t.Fatal(err)
	}
	var f struct {
		Comment string     `json:"_comment"`
		Keys    []*testKey `json:"keys"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.Comment, "TEST ONLY - NOT A SECRET") {
		t.Fatal("test key file must be labeled TEST ONLY - NOT A SECRET")
	}
	out := map[string]*testKey{}
	for _, k := range f.Keys {
		seed, err := hex.DecodeString(k.SeedHex)
		if err != nil || len(seed) != ed25519.SeedSize {
			t.Fatalf("bad seed for %s", k.Label)
		}
		want := sha256.Sum256([]byte("TEST ONLY - NOT A SECRET: fvpn " + k.Label + " test key"))
		if hex.EncodeToString(want[:]) != k.SeedHex {
			t.Fatalf("seed for %s does not match its documented derivation", k.Label)
		}
		k.priv = ed25519.NewKeyFromSeed(seed)
		out[k.Label] = k
	}
	return out
}

var (
	t0      = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t1      = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	nowTest = time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
)

func trusted(k *testKey, mods ...func(*TrustedKey)) TrustedKey {
	tk := TrustedKey{
		KeyID:     k.KeyID,
		PublicKey: k.priv.Public().(ed25519.PublicKey),
		Purposes:  []Purpose{k.Purpose},
		NotBefore: t0,
		NotAfter:  t1,
	}
	for _, m := range mods {
		m(&tk)
	}
	return tk
}

func mustStore(t *testing.T, keys ...TrustedKey) *TrustStore {
	t.Helper()
	ts, err := NewTrustStore(keys...)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

var doc = map[string]any{"release_id": "6d2797d0-b9a5-4724-96af-9b3039d351b1", "release_sequence": "42", "n": 1}

func TestSignVerifyRoundTrip(t *testing.T) {
	keys := loadTestKeys(t)
	rel := keys["release"]
	env, err := Sign(doc, rel.priv, rel.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	if string(env.Signed) != `{"n":1,"release_id":"6d2797d0-b9a5-4724-96af-9b3039d351b1","release_sequence":"42"}` {
		t.Fatalf("signed not canonical: %s", env.Signed)
	}
	if len(env.Signatures) != 1 || env.Signatures[0].Algorithm != "ed25519" || env.Signatures[0].KeyID != rel.KeyID {
		t.Fatalf("signatures: %+v", env.Signatures)
	}
	// Ed25519 is deterministic: signing twice yields the same bytes.
	env2, _ := Sign(json.RawMessage(`{"release_sequence":"42", "n":1, "release_id":"6d2797d0-b9a5-4724-96af-9b3039d351b1"}`), rel.priv, rel.KeyID)
	if env2.Signatures[0].SignatureB64 != env.Signatures[0].SignatureB64 {
		t.Fatal("signature not deterministic over equivalent JSON")
	}
	ts := mustStore(t, trusted(rel))
	raw := mustJSON(t, env)
	signed, kid, err := Verify(raw, ts, PurposeRelease, nowTest)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if kid != rel.KeyID || string(signed) != string(env.Signed) {
		t.Fatalf("got %s %s", kid, signed)
	}
	// Re-formatting the envelope (whitespace, member order) keeps it valid,
	// because the signature covers the canonical form of "signed".
	pretty, _ := json.MarshalIndent(json.RawMessage(raw), "", "  ")
	if _, _, err := Verify(pretty, ts, PurposeRelease, nowTest); err != nil {
		t.Fatalf("pretty envelope: %v", err)
	}
	canon, err := env.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Verify(canon, ts, PurposeRelease, nowTest); err != nil {
		t.Fatalf("canonical envelope: %v", err)
	}
}

func TestSignErrors(t *testing.T) {
	keys := loadTestKeys(t)
	rel := keys["release"]
	cases := []struct {
		name   string
		signed any
		priv   ed25519.PrivateKey
		keyID  string
	}{
		{"array", []any{1}, rel.priv, rel.KeyID},
		{"string", "x", rel.priv, rel.KeyID},
		{"float", map[string]any{"x": 1.5}, rel.priv, rel.KeyID},
		{"bad json bytes", []byte(`{`), rel.priv, rel.KeyID},
		{"bad raw", json.RawMessage(`{"a":1,"a":2}`), rel.priv, rel.KeyID},
		{"short key", doc, ed25519.PrivateKey{1, 2}, rel.KeyID},
		{"empty key id", doc, rel.priv, ""},
	}
	for _, tc := range cases {
		if _, err := Sign(tc.signed, tc.priv, tc.keyID); err == nil {
			t.Errorf("%s: Sign succeeded", tc.name)
		}
	}
}

func TestVerifyRejections(t *testing.T) {
	keys := loadTestKeys(t)
	rel, rel2, ds, un := keys["release"], keys["release-2"], keys["desired-state"], keys["untrusted"]
	good, _ := Sign(doc, rel.priv, rel.KeyID)
	goodRaw := mustJSON(t, good)
	ts := mustStore(t, trusted(rel), trusted(ds), trusted(rel2))

	mutate := func(f func(*Envelope)) []byte {
		e := Envelope{Signed: append([]byte(nil), good.Signed...), Signatures: append([]Signature(nil), good.Signatures...)}
		f(&e)
		return mustJSON(t, e)
	}
	byUntrusted, _ := Sign(doc, un.priv, un.KeyID)
	wrongKeySameID := mutate(func(e *Envelope) {
		e.Signatures[0].SignatureB64 = byUntrusted.Signatures[0].SignatureB64
	})
	bothGoodAndBad := mutate(func(e *Envelope) {
		e.Signatures = append(e.Signatures, Signature{KeyID: rel2.KeyID, Algorithm: "ed25519", SignatureB64: byUntrusted.Signatures[0].SignatureB64})
	})

	cases := []struct {
		name    string
		raw     []byte
		purpose Purpose
		now     time.Time
		ts      *TrustStore
		want    error
	}{
		{"tampered signed", mutate(func(e *Envelope) {
			e.Signed = []byte(`{"n":2,"release_id":"6d2797d0-b9a5-4724-96af-9b3039d351b1","release_sequence":"42"}`)
		}), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"unknown algorithm", mutate(func(e *Envelope) { e.Signatures[0].Algorithm = "rsa-pss" }), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"algorithm case", mutate(func(e *Envelope) { e.Signatures[0].Algorithm = "Ed25519" }), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"empty signatures", mutate(func(e *Envelope) { e.Signatures = []Signature{} }), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"null signatures", []byte(`{"signed":{"a":1},"signatures":null}`), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"bad base64", mutate(func(e *Envelope) { e.Signatures[0].SignatureB64 = "!!!" }), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"unpadded base64", mutate(func(e *Envelope) { e.Signatures[0].SignatureB64 = strings.TrimRight(e.Signatures[0].SignatureB64, "=") }), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"url base64", mutate(func(e *Envelope) {
			b, _ := base64.StdEncoding.DecodeString(e.Signatures[0].SignatureB64)
			e.Signatures[0].SignatureB64 = base64.URLEncoding.EncodeToString(append(b[:0:0], 0xfb, 0xff)) + e.Signatures[0].SignatureB64[4:]
		}), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"short signature", mutate(func(e *Envelope) { e.Signatures[0].SignatureB64 = base64.StdEncoding.EncodeToString([]byte("short")) }), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"empty key id", mutate(func(e *Envelope) { e.Signatures[0].KeyID = "" }), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"wrong key behind known id", wrongKeySameID, PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"one good one bad known", bothGoodAndBad, PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"only unknown key", mustJSON(t, byUntrusted), PurposeRelease, nowTest, ts, ErrKeyUntrusted},
		{"wrong purpose", goodRaw, PurposeDesiredState, nowTest, ts, ErrKeyUntrusted},
		{"unknown purpose", goodRaw, Purpose("admin"), nowTest, ts, ErrKeyUntrusted},
		{"nil trust store", goodRaw, PurposeRelease, nowTest, nil, ErrKeyUntrusted},
		{"not yet valid", goodRaw, PurposeRelease, t0.Add(-time.Second), ts, ErrKeyUntrusted},
		{"expired at not_after", goodRaw, PurposeRelease, t1, ts, ErrKeyUntrusted},
		{"revoked", goodRaw, PurposeRelease, nowTest, mustStore(t, trusted(rel, func(k *TrustedKey) { k.Revoked = true })), ErrKeyUntrusted},
		{"empty trust store", goodRaw, PurposeRelease, nowTest, mustStore(t), ErrKeyUntrusted},
		{"not json", []byte(`not json`), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"duplicate member", []byte(`{"signed":{"a":1},"signed":{"a":2},"signatures":[]}`), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"duplicate inside signed", []byte(strings.Replace(string(goodRaw), `"n":1`, `"n":1,"n":2`, 1)), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"unknown top-level member", []byte(strings.Replace(string(goodRaw), `{"signed"`, `{"extra":1,"signed"`, 1)), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"unknown signature member", []byte(strings.Replace(string(goodRaw), `"algorithm"`, `"x":1,"algorithm"`, 1)), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"missing signed", []byte(`{"signatures":[{"key_id":"a","algorithm":"ed25519","signature_b64":"AA=="}]}`), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"null signed", []byte(`{"signed":null,"signatures":[]}`), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"array signed", []byte(`{"signed":[1],"signatures":[]}`), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"float in signed", []byte(`{"signed":{"a":1.5},"signatures":[]}`), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
		{"envelope is array", []byte(`[1]`), PurposeRelease, nowTest, ts, ErrSignatureInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Verify(tc.raw, tc.ts, tc.purpose, tc.now)
			if err == nil {
				t.Fatal("Verify succeeded")
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("error %v, want %v", err, tc.want)
			}
			var e *Error
			if !errors.As(err, &e) || (e.Code != CodeKeyUntrusted && e.Code != CodeSignatureInvalid) {
				t.Fatalf("not a typed error: %#v", err)
			}
			if strings.Contains(err.Error(), "seed") {
				t.Fatal("error mentions key material")
			}
		})
	}
}

func TestVerifyIgnoresUnknownKeysAndRotation(t *testing.T) {
	keys := loadTestKeys(t)
	rel, rel2, un := keys["release"], keys["release-2"], keys["untrusted"]
	env, _ := Sign(doc, un.priv, un.KeyID)
	if err := env.AddSignature(rel.priv, rel.KeyID); err != nil {
		t.Fatal(err)
	}
	if err := env.AddSignature(rel2.priv, rel2.KeyID); err != nil {
		t.Fatal(err)
	}
	raw := mustJSON(t, env)
	// Unknown first signature is ignored; the first trusted key wins.
	_, kid, err := Verify(raw, mustStore(t, trusted(rel), trusted(rel2)), PurposeRelease, nowTest)
	if err != nil || kid != rel.KeyID {
		t.Fatalf("kid=%s err=%v", kid, err)
	}
	// After rotation only the new key is known: still valid.
	_, kid, err = Verify(raw, mustStore(t, trusted(rel2)), PurposeRelease, nowTest)
	if err != nil || kid != rel2.KeyID {
		t.Fatalf("kid=%s err=%v", kid, err)
	}
	// A revoked old key that still signed the document taints it.
	_, _, err = Verify(raw, mustStore(t, trusted(rel, func(k *TrustedKey) { k.Revoked = true }), trusted(rel2)), PurposeRelease, nowTest)
	if !errors.Is(err, ErrKeyUntrusted) {
		t.Fatalf("revoked co-signer: %v", err)
	}
	// Bad base64 in an unknown key's signature is still rejected.
	env.Signatures[0].SignatureB64 = "@@"
	if _, _, err := Verify(mustJSON(t, env), mustStore(t, trusted(rel)), PurposeRelease, nowTest); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("malformed unknown signature: %v", err)
	}
}

func TestNewTrustStoreValidation(t *testing.T) {
	keys := loadTestKeys(t)
	rel := keys["release"]
	cases := map[string]TrustedKey{
		"empty id":       trusted(rel, func(k *TrustedKey) { k.KeyID = "" }),
		"short key":      trusted(rel, func(k *TrustedKey) { k.PublicKey = k.PublicKey[:10] }),
		"no purpose":     trusted(rel, func(k *TrustedKey) { k.Purposes = nil }),
		"bad purpose":    trusted(rel, func(k *TrustedKey) { k.Purposes = []Purpose{"root"} }),
		"zero window":    trusted(rel, func(k *TrustedKey) { k.NotAfter = time.Time{} }),
		"zero start":     trusted(rel, func(k *TrustedKey) { k.NotBefore = time.Time{} }),
		"inverted":       trusted(rel, func(k *TrustedKey) { k.NotBefore, k.NotAfter = t1, t0 }),
		"empty interval": trusted(rel, func(k *TrustedKey) { k.NotAfter = t0 }),
	}
	for name, k := range cases {
		if _, err := NewTrustStore(k); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := NewTrustStore(trusted(rel), trusted(rel)); err == nil {
		t.Error("duplicate key id accepted")
	}
	ts := mustStore(t, trusted(rel))
	got, ok := ts.Lookup(rel.KeyID)
	if !ok || got.KeyID != rel.KeyID || ts.Len() != 1 {
		t.Fatal("Lookup")
	}
	got.PublicKey[0] ^= 0xff // must not affect the store
	again, _ := ts.Lookup(rel.KeyID)
	if again.PublicKey[0] == got.PublicKey[0] {
		t.Fatal("Lookup returned shared memory")
	}
	if _, ok := ts.Lookup("missing"); ok {
		t.Fatal("Lookup(missing)")
	}
}

func TestTrustStoreJSON(t *testing.T) {
	keys := loadTestKeys(t)
	rel, ds := keys["release"], keys["desired-state"]
	ts := mustStore(t, trusted(ds), trusted(rel, func(k *TrustedKey) { k.Revoked = true }))
	b, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(b), ds.KeyID) > strings.Index(string(b), rel.KeyID) {
		t.Fatalf("keys not sorted: %s", b)
	}
	back, err := ParseTrustStoreJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	k, ok := back.Lookup(rel.KeyID)
	if !ok || !k.Revoked || !k.NotBefore.Equal(t0) || !k.NotAfter.Equal(t1) || k.Purposes[0] != PurposeRelease {
		t.Fatalf("round trip: %+v", k)
	}
	bad := []string{
		`{"keys":[{"key_id":"a","algorithm":"rsa","public_key_b64":"","purposes":["release"],"not_before":"2026-01-01T00:00:00Z","not_after":"2027-01-01T00:00:00Z","revoked":false}]}`,
		`{"keys":[{"key_id":"a","algorithm":"ed25519","public_key_b64":"%%","purposes":["release"],"not_before":"2026-01-01T00:00:00Z","not_after":"2027-01-01T00:00:00Z","revoked":false}]}`,
		`{"keys":[{"key_id":"a","algorithm":"ed25519","public_key_b64":"AAAA","purposes":["release"],"not_before":"2026-01-01T00:00:00Z","not_after":"2027-01-01T00:00:00Z","revoked":false}]}`,
		`{"keys":[],"extra":true}`,
		`{`,
	}
	for _, in := range bad {
		if _, err := ParseTrustStoreJSON([]byte(in)); err == nil {
			t.Errorf("accepted %s", in)
		}
	}
}

func TestErrorText(t *testing.T) {
	e := &Error{Code: CodeKeyUntrusted, Reason: "r"}
	if e.Error() != "KEY_UNTRUSTED: r" || errors.Is(e, ErrSignatureInvalid) {
		t.Fatal(e.Error())
	}
	e2 := &Error{Code: CodeSignatureInvalid, KeyID: "k", Reason: "r"}
	if e2.Error() != `SIGNATURE_INVALID: key "k": r` || !errors.Is(e2, ErrSignatureInvalid) || errors.Is(e2, fmt.Errorf("x")) {
		t.Fatal(e2.Error())
	}
	long := strings.Repeat("a", 100)
	if len(short(long)) > 40 {
		t.Fatal("short")
	}
}

func TestAddSignatureRejectsInvalidSigned(t *testing.T) {
	keys := loadTestKeys(t)
	e := Envelope{Signed: json.RawMessage(`[1]`)}
	if err := e.AddSignature(keys["release"].priv, "k"); err == nil {
		t.Fatal("accepted array")
	}
}
