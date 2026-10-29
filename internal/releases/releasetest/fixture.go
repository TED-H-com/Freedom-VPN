// Package releasetest supplies TEST ONLY manifests signed by ephemeral keys.
package releasetest

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

type Fixture struct {
	Validator *releases.Validator
	Key       ed25519.PrivateKey
	Manifest  api.ReleaseManifest
}

func New(t testing.TB) *Fixture {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	raw, e := os.ReadFile(filepath.Join(filepath.Dir(file), "../../../contracts/vectors/release-manifest-signing.json"))
	if e != nil {
		t.Fatal(e)
	}
	var vector struct {
		Signed api.ReleaseManifest `json:"signed"`
	}
	if e = json.Unmarshal(raw, &vector); e != nil {
		t.Fatal(e)
	}
	pub, key, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	now := time.Now().UTC()
	trust, e := envelope.NewTrustStore(envelope.TrustedKey{KeyID: "TEST-ONLY-ephemeral-release", PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour)})
	if e != nil {
		t.Fatal(e)
	}
	v, e := releases.NewValidator(trust, []string{"https://releases.example.invalid"})
	if e != nil {
		t.Fatal(e)
	}
	vector.Signed.ReleaseId = uuid.Must(uuid.NewV7())
	return &Fixture{Validator: v, Key: key, Manifest: vector.Signed}
}
func (f *Fixture) Envelope(t testing.TB, m api.ReleaseManifest) []byte {
	t.Helper()
	env, e := envelope.Sign(m, f.Key, "TEST-ONLY-ephemeral-release")
	if e != nil {
		t.Fatal(e)
	}
	raw, e := env.MarshalCanonical()
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func (f *Fixture) Request(t testing.TB, m api.ReleaseManifest) []byte {
	t.Helper()
	raw, e := json.Marshal(map[string]any{"manifest": json.RawMessage(f.Envelope(t, m)), "manifest_url": "https://releases.example.invalid/" + m.ReleaseId.String() + "/manifest.json"})
	if e != nil {
		t.Fatal(e)
	}
	return raw
}
func Command() releases.Command {
	return releases.Command{ActorID: uuid.Must(uuid.NewV7()), IdempotencyKey: uuid.Must(uuid.NewV7())}
}
