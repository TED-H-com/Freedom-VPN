package release

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func TestGoldenReleaseManifestVector(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "contracts/vectors/release-manifest-signing.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Comment      string          `json:"_comment"`
		Canonical    string          `json:"canonical_signed"`
		Envelope     json.RawMessage `json:"envelope"`
		EnvelopeSHA  string          `json:"envelope_sha256"`
		KeyID        string          `json:"key_id"`
		PublicB64    string          `json:"public_key_b64"`
		Purpose      string          `json:"purpose"`
		SeedHex      string          `json:"seed_hex"`
		SignatureB64 string          `json:"signature_b64"`
		VerifyAt     time.Time       `json:"verify_at"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(v.Comment, "TEST ONLY - NOT A SECRET") {
		t.Fatal("vector must be labeled TEST ONLY - NOT A SECRET")
	}
	if v.Purpose != string(envelope.PurposeRelease) {
		t.Fatalf("purpose %s", v.Purpose)
	}
	seed, err := hex.DecodeString(v.SeedHex)
	if err != nil {
		t.Fatal(err)
	}
	key, err := KeyFromSeed(v.KeyID, seed)
	if err != nil {
		t.Fatal(err)
	}
	env, canonical, err := Sign([]byte(v.Canonical), key)
	if err != nil {
		t.Fatal(err)
	}
	if len(env.Signatures) != 1 || env.Signatures[0].SignatureB64 != v.SignatureB64 {
		t.Fatalf("signature = %s, want %s", env.Signatures[0].SignatureB64, v.SignatureB64)
	}
	sum, err := EnvelopeSHA256(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if sum != v.EnvelopeSHA {
		t.Fatalf("envelope sha256 = %s, want %s", sum, v.EnvelopeSHA)
	}
	pub, err := base64.StdEncoding.DecodeString(v.PublicB64)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := envelope.NewTrustStore(envelope.TrustedKey{
		KeyID: v.KeyID, PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease},
		NotBefore: v.VerifyAt.Add(-time.Hour), NotAfter: v.VerifyAt.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	signed, keyID, err := Verify(v.Envelope, ts, v.VerifyAt)
	if err != nil {
		t.Fatal(err)
	}
	if keyID != v.KeyID || string(signed) != v.Canonical {
		t.Fatalf("verify key %s", keyID)
	}
	tampered := strings.Replace(string(v.Envelope), `"product_version": "1.1.0"`, `"product_version": "1.1.1"`, 1)
	if tampered == string(v.Envelope) {
		t.Fatal("tamper did not change the envelope")
	}
	if _, _, err := Verify([]byte(tampered), ts, v.VerifyAt); err == nil {
		t.Fatal("tampered envelope verified")
	}
	badSig := strings.Replace(string(v.Envelope), v.SignatureB64[:8], "AAAAAAAA", 1)
	if _, _, err := Verify([]byte(badSig), ts, v.VerifyAt); err == nil {
		t.Fatal("bad signature verified")
	}
}

func TestBuildSignVerifyRoundTripAndTamper(t *testing.T) {
	dir := t.TempDir()
	amd := filepath.Join(dir, "amd64.tar.zst")
	arm := filepath.Join(dir, "arm64.tar.zst")
	if err := PackageDir(writeTree(t, dir, "amd"), amd, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := PackageDir(writeTree(t, dir, "arm"), arm, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	sbom := filepath.Join(dir, "sbom.json")
	if err := os.WriteFile(sbom, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := sampleInput(amd, arm, sbom)
	signed, manifest, err := BuildManifest(in, time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ProductVersion != "1.2.0" || manifest.ReleaseId == uuid.Nil {
		t.Fatalf("manifest release %s version %s", manifest.ReleaseId, manifest.ProductVersion)
	}
	key, err := GenerateKey("release-signing-test-round")
	if err != nil {
		t.Fatal(err)
	}
	_, env, err := Sign(signed, key)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := key.TrustStore(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := Verify(env, ts, time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(signed) {
		t.Fatal("round trip signed bytes differ")
	}
	if err := CheckArtifact(manifest, "amd64", amd); err != nil {
		t.Fatal(err)
	}
	if err := CheckArtifact(manifest, "arm64", amd); err == nil {
		t.Fatal("arm64 check accepted the amd64 file")
	}
	if err := os.WriteFile(amd, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckArtifact(manifest, "amd64", amd); err == nil {
		t.Fatal("tampered artifact verified")
	}
	in.Artifacts = in.Artifacts[:1]
	if _, _, err := BuildManifest(in, time.Now()); err == nil {
		t.Fatal("amd64-only release was accepted")
	}
}

func sampleInput(amd, arm, sbom string) BuildInput {
	return BuildInput{
		ProductVersion:  "1.2.0",
		ReleaseSequence: "7",
		CreatedAt:       "2026-10-28T14:00:00Z",
		SourceCommit:    "1111111111111111111111111111111111111111",
		BuildID:         "build-test-7",
		Components: api.ManifestComponents{
			NodeAgent: "1.2.0", NodeUpdater: "1.2.0", Netbird: "0.76.2", Enforcement: "1.2.0",
		},
		Compatibility: api.ManifestCompatibility{
			ControlApiMajor: 1, PolicySchemaVersions: []int{1}, UsageEventSchemaVersions: []int{1},
			MinimumUpdaterVersion: "1.0.0", StateSchemaFrom: []int{1}, StateSchemaTo: 1,
			RequiresServiceRestart: true,
			MigrationId:            presence.NewNullNullable[string](),
		},
		Rollback: api.ManifestRollback{
			AllowedReleaseIds: []uuid.UUID{}, StateReadableByPrevious: true,
			RequiredFreeBytes: decstr.MustParse("4096"), HealthProfile: "vpn-node-standard-v1",
		},
		Evidence: EvidenceInput{
			SBOMURL: "https://releases.example.invalid/r7/sbom.json", SBOMPath: sbom,
			NoticesURL: "https://releases.example.invalid/r7/NOTICE.txt",
			SourceURL:  "https://releases.example.invalid/r7/source.tar.zst",
		},
		Artifacts: []ArtifactInput{
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "amd64", Path: amd,
				URL: "https://releases.example.invalid/r7/amd64.tar.zst", InstalledSizeBytes: "1000"},
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "arm64", Path: arm,
				URL: "https://releases.example.invalid/r7/arm64.tar.zst", InstalledSizeBytes: "1000"},
		},
	}
}
