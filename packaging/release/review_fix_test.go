package release

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestNodeTrustStoreAndDuplicateArtifact(t *testing.T) {
	key, err := GenerateKey("node-trust-test")
	if err != nil {
		t.Fatal(err)
	}
	ts, err := key.TrustStore(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := ts.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := envelope.ParseNodeTrustStoreJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed == nil {
		t.Fatal("node trust store parsed empty")
	}
	art := api.ManifestArtifact{
		Architecture: api.ArchitectureAmd64,
		OsId:         api.ManifestArtifactOsIdUbuntu,
		OsVersions:   []string{BaselineOSVersion},
		Sha256:       strings.Repeat("ab", 32),
		SizeBytes:    decstr.MustParse("1"),
	}
	err = CheckArtifact(api.ReleaseManifest{Artifacts: []api.ManifestArtifact{art, art}}, "amd64", filepath.Join(t.TempDir(), "missing"))
	if err == nil || !strings.Contains(err.Error(), "VALIDATION_FAILED") || !strings.Contains(err.Error(), "duplicate") {
		t.Fatal(err)
	}
}
