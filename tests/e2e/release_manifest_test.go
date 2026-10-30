//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestD034e_SignedManifestPreservesRollbackAllowlist(t *testing.T) {
	state := t.TempDir()
	if err := os.MkdirAll(filepath.Join(state, "release"), 0700); err != nil {
		t.Fatal(err)
	}
	key, err := release.GenerateKey("TEST-ONLY")
	if err != nil {
		t.Fatal(err)
	}
	if err = key.Save(filepath.Join(state, "release", "key.json")); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, arch := range []string{"amd64", "arm64"} {
		// Hash-only manifest unit input; live acceptance packages real binaries.
		if err = os.WriteFile(filepath.Join(dir, arch+".tar.zst"), []byte("TEST ONLY manifest hash input "+arch), 0600); err != nil {
			t.Fatal(err)
		}
	}
	last := uuid.New()
	for _, allowed := range [][]uuid.UUID{{}, RollbackAllowlist(last)} {
		_, raw, err := (&lab{state: state}).signedRelease(dir, "1.0.3", "4", strings.Repeat("a", 40), "TEST-ONLY", map[string]string{"amd64": "100", "arm64": "100"}, ReleaseOptions{AllowedReleaseIDs: allowed})
		if err != nil {
			t.Fatal(err)
		}
		var env struct{ Signed api.ReleaseManifest }
		if err = json.Unmarshal(raw, &env); err != nil {
			t.Fatal(err)
		}
		got := env.Signed.Rollback.AllowedReleaseIds
		if got == nil || len(got) != len(allowed) || (len(got) > 0 && got[0] != last) {
			t.Fatalf("manifest allowlist %v", got)
		}
		if err = release.ValidateEnvelope(raw); err != nil {
			t.Fatal(err)
		}
	}
}
