package installerjob_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

func TestD110_StableManifestIdentity(t *testing.T) {
	// TEST ONLY identifiers and digest; no keys or infrastructure.
	release := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	incarnation := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	digest := strings.Repeat("ab", 32)
	want := uuid.NewSHA1(uuid.NewSHA1(uuid.NameSpaceOID, []byte("fvpn-installer-apply|"+release.String()+"|"+digest)), []byte(digest+"|"+incarnation.String()))
	if got := installerjob.ID(release, digest, incarnation); got != want || got.Version() != 5 || installerjob.ID(release, strings.ToUpper(digest), incarnation) != got {
		t.Fatal("manifest identity or normalization changed", got, want)
	}
	for _, got := range []uuid.UUID{installerjob.ID(uuid.New(), digest, incarnation), installerjob.ID(release, strings.Repeat("cd", 32), incarnation), installerjob.ID(release, digest, uuid.New())} {
		if got == want {
			t.Fatal("release, digest and incarnation must each bind the identity")
		}
	}
}

func TestD110_MainUpdaterClaimNamespaceCompatibility(t *testing.T) {
	// Exercise main's actual ApplyPending: the missing manifest produces a
	// durable failure without networking, services or privileged operations.
	root := t.TempDir()
	e := &updater.Engine{Layout: updater.Layout{
		Releases: filepath.Join(root, "releases"), Current: filepath.Join(root, "current"),
		Identity: filepath.Join(root, "identity"), State: filepath.Join(root, "state"),
		Journal: filepath.Join(root, "journal"), Lock: filepath.Join(root, "update.lock"),
		Staging: filepath.Join(root, "staging"), Trust: filepath.Join(root, "trust.json"),
	}, Node: updater.Node{ID: uuid.New(), Incarnation: uuid.New(), BootID: uuid.New()}, Platform: updater.Platform{StateSchema: 1}}
	t.Cleanup(func() {
		if err := e.Close(); err != nil {
			t.Error(err)
		}
	})
	release, digest := uuid.New(), strings.Repeat("ab", 32)
	raw, err := json.Marshal(map[string]string{"release_id": release.String(), "manifest_sha256": digest, "request_id": uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "pending-release.json")
	if err = os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = e.ApplyPending(t.Context(), path); err == nil {
		t.Fatal("missing manifest must fail")
	}
	rows, err := e.Journal.Outbox(t.Context())
	if err != nil || len(rows) != 1 {
		t.Fatal("updater did not mint a durable result", len(rows), err)
	}
	var minted api.UpdateResult
	if err = json.Unmarshal(rows[0].Payload, &minted); err != nil {
		t.Fatal(err)
	}
	canonical, err := canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	pendingDigest := hex.EncodeToString(sum[:])
	legacy := uuid.NewSHA1(uuid.NewSHA1(uuid.NameSpaceOID, []byte("fvpn-installer-apply|"+release.String()+"|"+digest)), []byte(pendingDigest+"|"+e.Node.Incarnation.String()))
	desired := installerjob.ID(release, digest, e.Node.Incarnation)
	if legacy == desired {
		t.Fatal("legacy pending-record identity unexpectedly matches D-110")
	}
	// Main currently mints legacy; the same interop test must also accept the
	// coordinated follow-up 15 switch to the shared D-110 helper.
	switch minted.JobId {
	case legacy:
		t.Log("reproduced main updater's legacy pending-record identity; follow-up 15 must switch to D-110")
	case desired:
		t.Log("updater uses the shared D-110 manifest identity")
	default:
		t.Fatalf("updater job = %s; want legacy %s or coordinated D-110 %s", minted.JobId, legacy, desired)
	}
}
