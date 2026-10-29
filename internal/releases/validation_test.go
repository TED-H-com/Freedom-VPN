package releases_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestUPD02_ManifestValidation(t *testing.T) {
	f := releasetest.New(t)
	raw := f.Envelope(t, f.Manifest)
	if _, _, e := f.Validator.ValidateManifest(raw, time.Now()); e != nil {
		t.Fatal(e)
	}
	cases := map[string][]byte{"tampered": bytes.Replace(raw, []byte(`"build-test-42"`), []byte(`"build-test-43"`), 1)}
	m := f.Manifest
	m.Artifacts = m.Artifacts[:1]
	cases["missing_arm64"] = f.Envelope(t, m)
	b, _ := json.Marshal(f.Manifest)
	_ = json.Unmarshal(b, &m)
	m.Artifacts[1].Url = "https://untrusted.example.invalid/package"
	cases["storage_origin"] = f.Envelope(t, m)
	_ = json.Unmarshal(b, &m)
	m.Artifacts[1] = m.Artifacts[0]
	cases["duplicate_platform"] = f.Envelope(t, m)
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, e := f.Validator.ValidateManifest(raw, time.Now()); e == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
	for _, u := range []string{"http://releases.example.invalid/x", "https://releases.example.invalid.evil.example.com/x", "https://releases.example.invalid@evil.example.com/x", "https://releases.example.invalid:444/x", "https://releases.example.invalid/x#fragment"} {
		if f.Validator.AllowURL(u) {
			t.Fatalf("allowed %s", u)
		}
	}
}

func TestUPD02_SharedManifestSigningVector(t *testing.T) {
	raw, e := os.ReadFile("../../contracts/vectors/release-manifest-signing.json")
	if e != nil {
		t.Fatal(e)
	}
	var vector struct {
		Envelope json.RawMessage `json:"envelope"`
		Hash     string          `json:"envelope_sha256"`
		KeyID    string          `json:"key_id"`
		Public   string          `json:"public_key_b64"`
		At       time.Time       `json:"verify_at"`
	}
	if e = json.Unmarshal(raw, &vector); e != nil {
		t.Fatal(e)
	}
	pub, e := base64.StdEncoding.DecodeString(vector.Public)
	if e != nil {
		t.Fatal(e)
	}
	trust, e := envelope.NewTrustStore(envelope.TrustedKey{KeyID: vector.KeyID, PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: vector.At.Add(-time.Hour), NotAfter: vector.At.Add(time.Hour)})
	if e != nil {
		t.Fatal(e)
	}
	v, e := releases.NewValidator(trust, []string{"https://releases.example.invalid"})
	if e != nil {
		t.Fatal(e)
	}
	_, hash, e := v.ValidateManifest(vector.Envelope, vector.At)
	if e != nil || hash != vector.Hash {
		t.Fatal(hash, e)
	}
}
