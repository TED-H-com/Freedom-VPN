package signing

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestSEC01_SnapshotMutationsAndCopies(t *testing.T) {
	// TEST ONLY: every key is generated in the test's disk directory.
	p := vault(t)
	ctx := context.Background()
	assertSnapshot := func() FileSnapshot {
		t.Helper()
		got, err := p.Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := os.ReadFile(filepath.Join(p.dir, "state.json"))
		if err != nil {
			t.Fatal(err)
		}
		var want ledger
		if err := json.Unmarshal(raw, &want); err != nil {
			t.Fatal(err)
		}
		if got.Revision != want.Revision || got.ReleaseRevision != want.ReleaseRevision || got.DeniedIdentityCount != len(want.DeniedIdentities) || !reflect.DeepEqual(got.Keys, want.Keys) {
			t.Fatal("snapshot differs from ledger")
		}
		trust, err := p.TrustBundle(ctx)
		if err != nil {
			t.Fatal(err)
		}
		a, _ := json.Marshal(trust)
		b, _ := json.Marshal(got.NodeTrust)
		if string(a) != string(b) {
			t.Fatal("snapshot bundle differs")
		}
		return got
	}
	assertSnapshot()
	id, err := p.Rotate(ctx, DesiredState, time.Hour, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	old := assertSnapshot()
	if _, err = p.Rotate(ctx, DesiredState, time.Hour, time.Minute); err != nil {
		t.Fatal(err)
	}
	assertSnapshot()
	if old.Keys[0].State != "active" || old.Revision != 1 {
		t.Fatal("later rotation changed earlier snapshot")
	}
	if err = p.Revoke(ctx, id); err != nil {
		t.Fatal(err)
	}
	assertSnapshot()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	root := envelope.TrustedKey{KeyID: "TEST-ONLY-release", PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
	if err = p.PinReleaseRoot(ctx, root); err != nil {
		t.Fatal(err)
	}
	assertSnapshot()
	if err = p.DenyIdentity(ctx, uuid.NewString()); err != nil {
		t.Fatal(err)
	}
	got := assertSnapshot()
	// Exercise release revision as well as ordinary ledger revision.
	release := got.Keys[len(got.Keys)-1]
	release.NotAfter = release.NotAfter.Add(-time.Minute)
	update, err := envelope.Sign(ReleaseUpdate{Kind: "fvpn-release-trust-v1", Revision: 1, Keys: []KeyRecord{release}}, priv, root.KeyID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := update.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	if err = p.UpdateReleaseTrust(ctx, raw); err != nil {
		t.Fatal(err)
	}
	got = assertSnapshot()
	if got.DeniedIdentityCount != 1 || got.ReleaseRevision != 1 {
		t.Fatal("metadata missing")
	}
	trustBefore, _ := json.Marshal(got.NodeTrust)
	got.Keys[0].Public[0] ^= 255
	*got.Keys[0].RevokedAt = time.Time{}
	got.Keys[0].State = "active"
	trustAfter, _ := json.Marshal(got.NodeTrust)
	if string(trustBefore) != string(trustAfter) {
		t.Fatal("keys alias snapshot bundle")
	}
	trusted, _ := got.NodeTrust.Lookup(id)
	trusted.PublicKey[0] ^= 255
	trusted.Purposes[0] = envelope.PurposeRelease
	again := assertSnapshot()
	if again.Keys[0].State != "revoked" || again.Keys[0].RevokedAt.IsZero() || reflect.DeepEqual(again.Keys[0].Public, got.Keys[0].Public) {
		t.Fatal("snapshot aliases later state")
	}
}
func TestSEC01_SnapshotCanceledContext(t *testing.T) {
	p := vault(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Snapshot(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Snapshot: %v", err)
	}
	if _, err := OpenFileProviderContext(ctx, p.dir); !errors.Is(err, context.Canceled) {
		t.Fatalf("Open: %v", err)
	}
}
