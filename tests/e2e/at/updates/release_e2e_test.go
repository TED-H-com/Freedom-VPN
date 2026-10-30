//go:build e2e

package updates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// builtRelease is one signed package registered with the lab origin (§13.3).
type builtRelease struct {
	ID        uuid.UUID
	AMDSHA    string
	ARMSHA    string
	Canonical []byte
}

type releaseSpec struct {
	name       string
	product    string
	sequence   string
	commit     string
	build      string
	marker     string
	blobBytes  int
	schemaFrom []int
	schemaTo   int
	installed  string
}

func (l *lab) buildRelease(ctx context.Context, spec releaseSpec) (builtRelease, error) {
	dir, err := os.MkdirTemp(filepath.Join(l.root, ".agent", "tmp"), "at-rel-")
	if err != nil {
		return builtRelease{}, err
	}
	defer os.RemoveAll(dir)
	when := time.Unix(1_700_000_000, 0).UTC()
	var blob []byte
	if spec.blobBytes > 0 {
		blob = make([]byte, spec.blobBytes)
		for i := range blob {
			blob[i] = byte(i*17 + 3)
		}
	}
	amd := filepath.Join(dir, "amd64.tar.zst")
	arm := filepath.Join(dir, "arm64.tar.zst")
	// Distinct trees so the journal sha identifies which architecture was fetched (§13.3, AT23).
	if err := packageArch(dir, "amd64", spec.marker, blob, when, amd); err != nil {
		return builtRelease{}, err
	}
	if err := packageArch(dir, "arm64", spec.marker, blob, when, arm); err != nil {
		return builtRelease{}, err
	}
	amdSum, err := fileSHA(amd)
	if err != nil {
		return builtRelease{}, err
	}
	armSum, err := fileSHA(arm)
	if err != nil {
		return builtRelease{}, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return builtRelease{}, err
	}
	if spec.schemaFrom == nil {
		spec.schemaFrom = []int{1}
	}
	if spec.schemaTo == 0 {
		spec.schemaTo = 1
	}
	if spec.installed == "" {
		spec.installed = "1000"
	}
	sbom := filepath.Join(dir, "sbom.json")
	if err := os.WriteFile(sbom, []byte("{}\n"), 0o644); err != nil {
		return builtRelease{}, err
	}
	key, err := release.LoadKey(filepath.Join(l.state, "release", "key.json"))
	if err != nil {
		return builtRelease{}, err
	}
	base := "https://releases.example.invalid/r/" + id.String()
	signed, _, err := release.BuildManifest(release.BuildInput{
		ReleaseID: id.String(), ProductVersion: spec.product, ReleaseSequence: spec.sequence,
		SourceCommit: spec.commit, BuildID: spec.build,
		Components: api.ManifestComponents{NodeAgent: "1.0.0", NodeUpdater: "1.0.0", Netbird: "0.82.0", Enforcement: "1.0.0"},
		Compatibility: api.ManifestCompatibility{
			ControlApiMajor: 1, PolicySchemaVersions: []int{1}, UsageEventSchemaVersions: []int{1},
			MinimumUpdaterVersion: "1.0.0", StateSchemaFrom: spec.schemaFrom, StateSchemaTo: spec.schemaTo,
			RequiresServiceRestart: true, MigrationId: presence.NewNullNullable[string](),
		},
		Rollback: api.ManifestRollback{
			AllowedReleaseIds: []openapi_types.UUID{}, StateReadableByPrevious: true,
			RequiredFreeBytes: decstr.MustParse("4096"), HealthProfile: "vpn-node-standard-v1",
		},
		Evidence: release.EvidenceInput{
			SBOMURL: base + "/sbom.json", SBOMPath: sbom,
			NoticesURL: base + "/NOTICE.txt", SourceURL: base + "/source.txt",
		},
		Artifacts: []release.ArtifactInput{
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "amd64", Path: amd, URL: base + "/amd64.tar.zst", InstalledSizeBytes: spec.installed},
			{OSID: "ubuntu", OSVersions: []string{"24.04"}, Architecture: "arm64", Path: arm, URL: base + "/arm64.tar.zst", InstalledSizeBytes: spec.installed},
		},
	}, time.Now().UTC())
	if err != nil {
		return builtRelease{}, err
	}
	_, canonical, err := release.Sign(signed, key)
	if err != nil {
		return builtRelease{}, err
	}
	for _, item := range []struct {
		name string
		body []byte
	}{
		{"amd64.tar.zst", mustRead(amd)},
		{"arm64.tar.zst", mustRead(arm)},
		{"sbom.json", []byte("{}\n")},
		{"NOTICE.txt", []byte("e2e\n")},
		{"source.txt", []byte("e2e\n")},
		{"manifest.json", canonical},
	} {
		if err := l.putObject(ctx, "r/"+id.String()+"/"+item.name, item.body); err != nil {
			return builtRelease{}, err
		}
	}
	return builtRelease{ID: id, AMDSHA: amdSum, ARMSHA: armSum, Canonical: canonical}, nil
}

func packageArch(dir, arch, marker string, blob []byte, when time.Time, dest string) error {
	tree := filepath.Join(dir, "tree-"+arch, "bin")
	if err := os.MkdirAll(tree, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tree, "marker"), []byte(marker+arch+"\n"), 0o644); err != nil {
		return err
	}
	if len(blob) > 0 {
		if err := os.WriteFile(filepath.Join(tree, "blob"), blob, 0o644); err != nil {
			return err
		}
	}
	return release.PackageDir(filepath.Join(dir, "tree-"+arch), dest, when)
}

func fileSHA(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func mustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

func (l *lab) registerRelease(ctx context.Context, admin *session, built builtRelease) (reply, error) {
	return admin.call(ctx, "POST", l.public+"/api/v1/admin/releases", map[string]any{
		"manifest":     json.RawMessage(built.Canonical),
		"manifest_url": "https://releases.example.invalid/r/" + built.ID.String() + "/manifest.json",
	}, map[string]string{"Idempotency-Key": uuid.NewString()})
}

func (l *lab) awaitVerified(ctx context.Context, admin *session, ids []uuid.UUID) (map[string]string, error) {
	states := map[string]string{}
	ok := waitUntil(ctx, 5*time.Second, func() bool {
		list, err := admin.call(ctx, "GET", l.public+"/api/v1/admin/releases", nil, nil)
		if err != nil || list.Status != 200 {
			return false
		}
		var page api.ReleaseList
		if json.Unmarshal(list.Body, &page) != nil {
			return false
		}
		seen := 0
		for _, item := range page.Items {
			id := uuid.UUID(item.ReleaseId).String()
			for _, want := range ids {
				if id == want.String() {
					states[id] = string(item.VerificationState)
					if item.VerificationState == "verified" || item.VerificationState == "failed" {
						seen++
					}
				}
			}
		}
		return seen == len(ids)
	})
	if !ok {
		return states, fmt.Errorf("verification timed out")
	}
	return states, nil
}
