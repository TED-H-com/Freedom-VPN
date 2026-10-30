package contracts

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/contracts/internal/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestCanonicalJSONVectors(t *testing.T) {
	raw := readFile(t, "vectors/canonical-json.json")
	var tagged struct {
		Cases []struct {
			Name      string `json:"name"`
			InputB64  string `json:"input_b64"`
			Input     string `json:"input"`
			Canonical string `json:"canonical"`
			SHA256    string `json:"sha256"`
		} `json:"cases"`
		Rejects []struct {
			Name     string `json:"name"`
			InputB64 string `json:"input_b64"`
			Error    string `json:"error"`
		} `json:"rejects"`
	}
	if err := json.Unmarshal(raw, &tagged); err != nil {
		t.Fatal(err)
	}
	if len(tagged.Cases) < 10 || len(tagged.Rejects) < 10 {
		t.Fatalf("too few vectors: %d cases, %d rejects", len(tagged.Cases), len(tagged.Rejects))
	}
	for _, c := range tagged.Cases {
		in, err := base64.StdEncoding.DecodeString(c.InputB64)
		if err != nil || string(in) != c.Input {
			t.Fatalf("%s: input_b64 does not match input", c.Name)
		}
		got, err := canonjson.Canonicalize(in)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		if string(got) != c.Canonical {
			t.Errorf("%s: canonical\n got  %q\n want %q", c.Name, got, c.Canonical)
		}
		sum := sha256.Sum256(got)
		if hex.EncodeToString(sum[:]) != c.SHA256 {
			t.Errorf("%s: sha256 mismatch", c.Name)
		}
		hh, err := canonjson.HashHex(in)
		if err != nil || hh != c.SHA256 {
			t.Errorf("%s: HashHex mismatch", c.Name)
		}
	}
	for _, r := range tagged.Rejects {
		in, _ := base64.StdEncoding.DecodeString(r.InputB64)
		_, err := canonjson.Canonicalize(in)
		if err == nil {
			t.Errorf("%s: accepted", r.Name)
			continue
		}
		if got := gen.ErrorClass(err); got != r.Error {
			t.Errorf("%s: error class %s, want %s", r.Name, got, r.Error)
		}
	}
}

func TestMTR03_DecimalStringVectors(t *testing.T) {
	var v struct {
		Pattern string   `json:"pattern"`
		Max     string   `json:"max"`
		Accept  []string `json:"accept"`
		Reject  []struct {
			JSON   string `json:"json"`
			Reason string `json:"reason"`
		} `json:"reject"`
	}
	readJSON(t, "vectors/decimal-strings.json", &v)
	if v.Pattern != decstr.Pattern || v.Max != decstr.Max.String() {
		t.Fatalf("pattern/max differ from pkg/decstr")
	}
	for _, s := range v.Accept {
		u, err := decstr.Parse(s)
		if err != nil || u.String() != s {
			t.Errorf("accept %q: %v", s, err)
		}
		var w struct{ V decstr.U64 }
		if err := json.Unmarshal([]byte(`{"V":"`+s+`"}`), &w); err != nil || w.V.String() != s {
			t.Errorf("accept JSON %q: %v", s, err)
		}
	}
	for _, r := range v.Reject {
		var u decstr.U64
		err := json.Unmarshal([]byte(r.JSON), &u)
		if err == nil {
			t.Errorf("reject %s (%s): accepted", r.JSON, r.Reason)
			continue
		}
		want := decstr.ErrSyntax
		switch r.Reason {
		case "out_of_range":
			want = decstr.ErrRange
		case "not_a_string":
			want = decstr.ErrJSONType
		}
		if !errors.Is(err, want) {
			t.Errorf("reject %s: error %v, want %v", r.JSON, err, want)
		}
	}
}

func TestMTR03_UsageEventPayloadSHA256Vector(t *testing.T) {
	var v struct {
		EventWithoutHash json.RawMessage `json:"event_without_hash"`
		Canonical        string          `json:"canonical_without_hash"`
		PayloadSHA256    string          `json:"payload_sha256"`
		Event            json.RawMessage `json:"event"`
	}
	readJSON(t, "vectors/usage-event-payload-sha256.json", &v)
	c, err := canonjson.Canonicalize(v.EventWithoutHash)
	if err != nil || string(c) != v.Canonical {
		t.Fatalf("canonical form mismatch: %v", err)
	}
	sum := sha256.Sum256(c)
	if hex.EncodeToString(sum[:]) != v.PayloadSHA256 {
		t.Fatal("payload_sha256 of the canonical form mismatch")
	}
	h, err := canonjson.HashHex(v.Event, "payload_sha256")
	if err != nil || h != v.PayloadSHA256 {
		t.Fatalf("HashHex(event) = %s, %v", h, err)
	}
	evCanon, _ := canonjson.Canonicalize(v.Event)
	if !strings.Contains(string(evCanon), `"payload_sha256":"`+v.PayloadSHA256+`"`) {
		t.Fatal("event does not carry the hash")
	}
	if !strings.Contains(v.Canonical, `"9007199254740993"`) {
		t.Fatal("vector must carry 2^53+1 bytes as a string")
	}
	if err := validateComponent(t, "UsageEvent", v.Event); err != nil {
		t.Fatal(err)
	}
	if err := validateJSONSchema(t, "usage-event.schema.json", v.Event); err != nil {
		t.Fatal(err)
	}
}

func TestUPD05_UpdateResultPayloadHashVector(t *testing.T) {
	var v struct {
		Canonical   string          `json:"canonical_without_hash"`
		PayloadHash string          `json:"payload_hash"`
		Result      json.RawMessage `json:"result"`
	}
	readJSON(t, "vectors/update-result-payload-hash.json", &v)
	c, err := canonjson.CanonicalizeWithout(v.Result, "payload_hash")
	if err != nil || string(c) != v.Canonical {
		t.Fatalf("canonical mismatch: %v", err)
	}
	if h, _ := canonjson.HashHex(v.Result, "payload_hash"); h != v.PayloadHash {
		t.Fatal("payload_hash mismatch")
	}
	if err := validateComponent(t, "UpdateResult", v.Result); err != nil {
		t.Fatal(err)
	}
}

func TestSigningVectors(t *testing.T) {
	files := map[string]envelope.Purpose{
		"vectors/release-trust-update-signing.json":         envelope.PurposeRelease,
		"vectors/release-manifest-signing.json":             envelope.PurposeRelease,
		"vectors/desired-state-revocations-signing.json":    envelope.PurposeDesiredState,
		"vectors/desired-state-resolver-probe-signing.json": envelope.PurposeDesiredState,
		"vectors/desired-state-signing.json":                envelope.PurposeDesiredState,
		"vectors/serving-lease-signing.json":                envelope.PurposeLease,
		"vectors/usage-checkpoint-signing.json":             envelope.PurposeRecovery,
	}
	components := map[envelope.Purpose]string{
		envelope.PurposeRelease: "ReleaseManifestEnvelope", envelope.PurposeDesiredState: "DesiredStateEnvelope",
		envelope.PurposeLease: "ServingLeaseEnvelope", envelope.PurposeRecovery: "UsageCheckpointEnvelope",
	}
	for file, purpose := range files {
		t.Run(file, func(t *testing.T) {
			var v struct {
				Comment         string          `json:"_comment"`
				Purpose         string          `json:"purpose"`
				KeyID           string          `json:"key_id"`
				SeedHex         string          `json:"seed_hex"`
				PublicKeyB64    string          `json:"public_key_b64"`
				VerifyAt        time.Time       `json:"verify_at"`
				Signed          json.RawMessage `json:"signed"`
				CanonicalSigned string          `json:"canonical_signed"`
				CanonicalSHA    string          `json:"canonical_signed_sha256"`
				SignatureB64    string          `json:"signature_b64"`
				Envelope        json.RawMessage `json:"envelope"`
				EnvelopeSHA     string          `json:"envelope_sha256"`
			}
			readJSON(t, file, &v)
			if !strings.HasPrefix(v.Comment, "TEST ONLY - NOT A SECRET") {
				t.Fatal("vector must be labeled TEST ONLY - NOT A SECRET")
			}
			if v.Purpose != string(purpose) {
				t.Fatalf("purpose %s", v.Purpose)
			}
			seed, err := hex.DecodeString(v.SeedHex)
			if err != nil || len(seed) != ed25519.SeedSize {
				t.Fatal("bad seed")
			}
			priv := ed25519.NewKeyFromSeed(seed)
			pub := priv.Public().(ed25519.PublicKey)
			if base64.StdEncoding.EncodeToString(pub) != v.PublicKeyB64 {
				t.Fatal("public key does not derive from seed")
			}
			c, err := canonjson.Canonicalize(v.Signed)
			if err != nil || string(c) != v.CanonicalSigned {
				t.Fatalf("canonical signed mismatch: %v", err)
			}
			if s := sha256.Sum256(c); hex.EncodeToString(s[:]) != v.CanonicalSHA {
				t.Fatal("canonical_signed_sha256 mismatch")
			}
			// Ed25519 is deterministic: re-signing yields the same bytes.
			if got := base64.StdEncoding.EncodeToString(ed25519.Sign(priv, c)); got != v.SignatureB64 {
				t.Fatalf("signature not reproducible")
			}
			env, err := envelope.Sign(v.Signed, priv, v.KeyID)
			if err != nil {
				t.Fatal(err)
			}
			envCanon, _ := env.MarshalCanonical()
			wantEnv, _ := canonjson.Canonicalize(v.Envelope)
			if !bytes.Equal(envCanon, wantEnv) {
				t.Fatal("envelope.Sign does not reproduce the vector envelope")
			}
			if s := sha256.Sum256(wantEnv); hex.EncodeToString(s[:]) != v.EnvelopeSHA {
				t.Fatal("envelope_sha256 mismatch")
			}
			ts, err := envelope.NewTrustStore(envelope.TrustedKey{KeyID: v.KeyID, PublicKey: pub,
				Purposes: []envelope.Purpose{purpose}, NotBefore: v.VerifyAt.Add(-time.Hour), NotAfter: v.VerifyAt.Add(time.Hour)})
			if err != nil {
				t.Fatal(err)
			}
			signed, kid, err := envelope.Verify(v.Envelope, ts, purpose, v.VerifyAt)
			if err != nil || kid != v.KeyID || string(signed) != v.CanonicalSigned {
				t.Fatalf("Verify: %v", err)
			}
			if file == "vectors/release-trust-update-signing.json" {
				if err := validateJSONSchema(t, "trust/release-update-envelope.schema.json", v.Envelope); err != nil {
					t.Fatal(err)
				}
			} else if err := validateComponent(t, components[purpose], v.Envelope); err != nil {
				t.Fatalf("envelope does not match %s: %v", components[purpose], err)
			}
		})
	}
}

// TestUPD02_ManifestHashBindsDesiredState checks that the update intent in
// the desired-state vector binds the exact manifest envelope (§13.10.2).
func TestUPD02_ManifestHashBindsDesiredState(t *testing.T) {
	var man, ds struct {
		EnvelopeSHA string          `json:"envelope_sha256"`
		Signed      json.RawMessage `json:"signed"`
	}
	readJSON(t, "vectors/release-manifest-signing.json", &man)
	readJSON(t, "vectors/desired-state-signing.json", &ds)
	var d struct {
		Update struct {
			ManifestSHA256 string `json:"manifest_sha256"`
		} `json:"update"`
	}
	if err := json.Unmarshal(ds.Signed, &d); err != nil {
		t.Fatal(err)
	}
	if d.Update.ManifestSHA256 != man.EnvelopeSHA {
		t.Fatalf("desired state manifest_sha256 %s != manifest envelope sha256 %s", d.Update.ManifestSHA256, man.EnvelopeSHA)
	}
}
