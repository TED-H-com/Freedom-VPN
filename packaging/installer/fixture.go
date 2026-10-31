package installer

import (
	"os"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// ReleaseFixture is a signed archive pair tests can install.
// Keys are generated at build time and are not product keys.
type ReleaseFixture struct {
	Dir       string
	Manifest  string
	Artifact  string
	Trust     string
	ReleaseID string
	Sequence  string
	SHA       string
}

// BuildReleaseFixture packs binaries and the deploy/node/systemd units into
// a signed release. unitDir is the directory of unit files. bins maps
// file names (fvpn-agent, fvpn-updater, fvpn-watchdog, fvpn-node-install)
// to contents. The other architecture is a second tiny archive so the
// manifest satisfies §13.4.
func BuildReleaseFixture(dir, unitDir, releaseID, sequence string, bins map[string][]byte) (ReleaseFixture, error) {
	tree := filepath.Join(dir, "tree-"+sequence)
	if err := os.MkdirAll(filepath.Join(tree, "lib", "systemd"), 0o755); err != nil {
		return ReleaseFixture{}, err
	}
	for name, body := range bins {
		if err := os.WriteFile(filepath.Join(tree, name), body, 0o755); err != nil {
			return ReleaseFixture{}, err
		}
	}
	entries, err := os.ReadDir(unitDir)
	if err != nil {
		return ReleaseFixture{}, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(unitDir, e.Name()))
		if err != nil {
			return ReleaseFixture{}, err
		}
		if err := os.WriteFile(filepath.Join(tree, "lib", "systemd", e.Name()), raw, 0o644); err != nil {
			return ReleaseFixture{}, err
		}
	}
	other := filepath.Join(dir, "tree-other-"+sequence)
	if err := os.MkdirAll(other, 0o755); err != nil {
		return ReleaseFixture{}, err
	}
	if err := os.WriteFile(filepath.Join(other, "fvpn-agent"), []byte("other\n"), 0o755); err != nil {
		return ReleaseFixture{}, err
	}
	amd := filepath.Join(dir, "amd64-"+sequence+".tar.zst")
	arm := filepath.Join(dir, "arm64-"+sequence+".tar.zst")
	when := time.Unix(1_700_000_000, 0).UTC()
	if err := release.PackageDir(tree, amd, when); err != nil {
		return ReleaseFixture{}, err
	}
	if err := release.PackageDir(other, arm, when); err != nil {
		return ReleaseFixture{}, err
	}
	sbom := filepath.Join(dir, "sbom-"+sequence+".json")
	if err := os.WriteFile(sbom, []byte("{}\n"), 0o644); err != nil {
		return ReleaseFixture{}, err
	}
	amdBytes, err := dirFileBytes(tree)
	if err != nil {
		return ReleaseFixture{}, err
	}
	armBytes, err := dirFileBytes(other)
	if err != nil {
		return ReleaseFixture{}, err
	}
	in := release.BuildInput{
		ReleaseID:       releaseID,
		ProductVersion:  "1.2." + sequence,
		ReleaseSequence: sequence,
		CreatedAt:       "2026-10-28T14:00:00Z",
		SourceCommit:    "1111111111111111111111111111111111111111",
		BuildID:         "build-test-" + sequence,
		Components: api.ManifestComponents{
			NodeAgent: "1.2.0", NodeUpdater: "1.2.0", Netbird: NetBirdPin, Enforcement: "1.2.0",
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
		Evidence: release.EvidenceInput{
			SBOMURL: "https://releases.example.invalid/r/sbom.json", SBOMPath: sbom,
			NoticesURL: "https://releases.example.invalid/r/NOTICE.txt",
			SourceURL:  "https://releases.example.invalid/r/source.tar.zst",
		},
		Artifacts: []release.ArtifactInput{
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "amd64", Path: amd,
				URL: "https://releases.example.invalid/r/amd64.tar.zst", InstalledSizeBytes: decstr.U64(amdBytes).String()},
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "arm64", Path: arm,
				URL: "https://releases.example.invalid/r/arm64.tar.zst", InstalledSizeBytes: decstr.U64(armBytes).String()},
		},
	}
	signed, _, err := release.BuildManifest(in, time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC))
	if err != nil {
		return ReleaseFixture{}, err
	}
	key, err := release.GenerateKey("release-signing-test-" + sequence)
	if err != nil {
		return ReleaseFixture{}, err
	}
	_, envRaw, err := release.Sign(signed, key)
	if err != nil {
		return ReleaseFixture{}, err
	}
	nb := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	na := time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)
	ts, err := key.TrustStore(nb, na)
	if err != nil {
		return ReleaseFixture{}, err
	}
	trustRaw, err := ts.MarshalJSON()
	if err != nil {
		return ReleaseFixture{}, err
	}
	manifest := filepath.Join(dir, "manifest-"+sequence+".json")
	trust := filepath.Join(dir, "trust-"+sequence+".json")
	if err := os.WriteFile(manifest, envRaw, 0o644); err != nil {
		return ReleaseFixture{}, err
	}
	if err := os.WriteFile(trust, append(trustRaw, '\n'), 0o644); err != nil {
		return ReleaseFixture{}, err
	}
	sum, err := release.EnvelopeSHA256(envRaw)
	if err != nil {
		return ReleaseFixture{}, err
	}
	return ReleaseFixture{
		Dir: dir, Manifest: manifest, Artifact: amd, Trust: trust,
		ReleaseID: releaseID, Sequence: sequence, SHA: sum,
	}, nil
}

func dirFileBytes(root string) (uint64, error) {
	var n uint64
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		st, err := d.Info()
		if err != nil {
			return err
		}
		n += uint64(st.Size())
		return nil
	})
	return n, err
}
