package signing

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func vault(t *testing.T) *FileProvider {
	t.Helper()
	p, err := InitializeFileProvider(filepath.Join(t.TempDir(), "vault"))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSEC01_NodeTrustBundleExcludesLedgerEpoch(t *testing.T) {
	p := vault(t)
	ctx := context.Background()
	for _, purpose := range []Purpose{DesiredState, Lease, Recovery, LedgerEpoch} {
		if _, err := p.Rotate(ctx, purpose, time.Hour, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := p.TrustBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := envelope.ParseNodeTrustStoreJSON(raw)
	if err != nil {
		t.Fatal("node rejected central bundle", err)
	}
	encoded, err := json.Marshal(parsed)
	if err != nil {
		t.Fatal(err)
	}
	var public struct {
		Keys []envelope.TrustedKey `json:"keys"`
	}
	if err = json.Unmarshal(encoded, &public); err != nil || len(public.Keys) != 3 {
		t.Fatal("node bundle lost an eligible purpose", err)
	}
	for _, key := range public.Keys {
		if len(key.Purposes) != 1 || key.Purposes[0] == envelope.PurposeLedgerEpoch {
			t.Fatal("central-only or combined purpose exported")
		}
	}
}
func TestSEC01_RotationOverlapAndRevocation(t *testing.T) {
	ctx := context.Background()
	p := vault(t)
	now := time.Now().UTC()
	p.now = func() time.Time { return now }
	old, err := p.Rotate(ctx, DesiredState, time.Hour, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ForPurpose(p, DesiredState)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := signer.Sign(ctx, map[string]string{"test": "public"})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := signed.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	next, err := p.Rotate(ctx, DesiredState, time.Hour, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := p.TrustBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = envelope.Verify(raw, ts, envelope.PurposeDesiredState, now); err != nil {
		t.Fatal("overlap failed", err)
	}
	if _, _, err = envelope.Verify(raw, ts, envelope.PurposeDesiredState, now.Add(10*time.Minute)); err == nil {
		t.Fatal("overlap end accepted")
	}
	got, err := signer.Sign(ctx, map[string]bool{"ok": true})
	if err != nil || got.Signatures[0].KeyID != next || got.Signatures[0].KeyID == old {
		t.Fatal("retired key active")
	}
	if err = p.Revoke(ctx, next); err != nil {
		t.Fatal(err)
	}
	if _, err = signer.Sign(ctx, map[string]bool{"ok": true}); err == nil {
		t.Fatal("revoked key signed")
	}
	reopened, err := OpenFileProvider(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.SignEnvelope(ctx, DesiredState, map[string]bool{"ok": true}); err == nil {
		t.Fatal("reopen revived key")
	}
	if _, err = ForPurpose(p, Release); err == nil {
		t.Fatal("release signer exposed")
	}
	if _, err = p.Rotate(ctx, Release, time.Hour, 0); err == nil {
		t.Fatal("release private key created")
	}
	for _, purpose := range []Purpose{Lease, Recovery, LedgerEpoch} {
		if _, err = p.Rotate(ctx, purpose, time.Hour, 0); err != nil {
			t.Fatal(err)
		}
		if _, err = p.SignEnvelope(ctx, purpose, map[string]string{"purpose": string(purpose)}); err != nil {
			t.Fatal(err)
		}
	}
	ts, err = p.TrustBundle(ctx)
	if err != nil {
		t.Fatal(err)
	}
	public, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = envelope.ParseTrustStoreJSON(public); err != nil {
		t.Fatal(err)
	}
}
func TestSEC01_CAHandleAndSecretBoundary(t *testing.T) {
	p := vault(t)
	ctx := context.Background()
	id, err := p.Rotate(ctx, NodeCA, time.Hour, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	h, err := p.CertificateSigner(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Revoke(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err = h.Sign(rand.Reader, make([]byte, 32), crypto.SHA256); err == nil {
		t.Fatal("old CA handle signed after revoke")
	}
	id, err = p.Rotate(ctx, BootstrapSecret, time.Hour, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := p.Seal(ctx, []byte("TEST ONLY bootstrap"), []byte("node-a"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Open(ctx, sealed, []byte("node-b")); err == nil {
		t.Fatal("ciphertext moved to another identity")
	}
	if _, err = p.Rotate(ctx, BootstrapSecret, time.Hour, 0); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Open(ctx, sealed, []byte("node-a")); err != nil {
		t.Fatal("retired encryption key cannot decrypt pending secret", err)
	}
	if err = p.Revoke(ctx, id); err != nil {
		t.Fatal(err)
	}
	if _, err = p.Open(ctx, sealed, []byte("node-a")); err == nil {
		t.Fatal("revoked encryption key used")
	}
	node := uuid.NewString()
	if err = p.DenyIdentity(ctx, node); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenFileProvider(p.dir)
	if err != nil {
		t.Fatal(err)
	}
	denied, err := reopened.IdentityDenied(ctx, node)
	if err != nil || !denied {
		t.Fatal("revocation tombstone lost")
	}
	if err = os.Remove(filepath.Join(p.dir, "state.json")); err != nil {
		t.Fatal(err)
	}
	if _, err = OpenFileProvider(p.dir); err == nil {
		t.Fatal("missing ledger recreated")
	}
}
func TestSEC01_ReleaseTrustRequiresExistingAuthorization(t *testing.T) {
	p := vault(t)
	ctx := context.Background()
	now := time.Now().UTC()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := uuid.NewString()
	if err = p.PinReleaseRoot(ctx, envelope.TrustedKey{KeyID: root, PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	nextPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	update := ReleaseUpdate{Kind: "fvpn-release-trust-v1", Revision: 1, Keys: []KeyRecord{{ID: uuid.NewString(), Purpose: Release, Public: nextPub, State: "active", NotBefore: now, NotAfter: now.Add(time.Hour)}}}
	env, err := envelope.Sign(update, priv, root)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := env.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	if err = p.UpdateReleaseTrust(ctx, raw); err != nil {
		t.Fatal(err)
	}
	if err = p.UpdateReleaseTrust(ctx, raw); err == nil {
		t.Fatal("replayed trust revision accepted")
	}
	if err = p.Revoke(ctx, root); err != nil {
		t.Fatal(err)
	}
	update.Revision = 2
	env, err = envelope.Sign(update, priv, root)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = env.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	if err = p.UpdateReleaseTrust(ctx, raw); err == nil {
		t.Fatal("revoked root authorized rotation")
	}
}

func TestSEC01_NodeTrustBundleExcludesCentralLedgerKey(t *testing.T) {
	p := vault(t)
	ctx := context.Background()
	id, e := p.Rotate(ctx, LedgerEpoch, time.Hour, 0)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Rotate(ctx, Lease, time.Hour, 0); e != nil {
		t.Fatal(e)
	}
	trust, e := p.TrustBundle(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, ok := trust.Lookup(id); ok {
		t.Fatal("central ledger key exposed to nodes")
	}
	raw, e := json.Marshal(trust)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = envelope.ParseNodeTrustStoreJSON(raw); e != nil {
		t.Fatal(e)
	}
}
