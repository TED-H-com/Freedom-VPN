package release

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// Ubuntu baseline fixed for v1 (§3.2, §13.4). A release that lacks either
// architecture cannot target all_nodes.
const (
	BaselineOSID      = "ubuntu"
	BaselineOSVersion = "24.04"
)

// ArtifactInput is one on-disk package the manifest should describe.
// size_bytes and sha256 are computed from Path; they are not taken from the
// caller (§13.3).
type ArtifactInput struct {
	OSID               string   `json:"os_id"`
	OSVersions         []string `json:"os_versions"`
	Architecture       string   `json:"architecture"`
	Path               string   `json:"path"`
	URL                string   `json:"url"`
	InstalledSizeBytes string   `json:"installed_size_bytes"`
}

// EvidenceInput is the release evidence block. sbom_sha256 is used when set;
// otherwise SBOMPath is hashed.
type EvidenceInput struct {
	SBOMURL    string `json:"sbom_url"`
	SBOMPath   string `json:"sbom_path,omitempty"`
	SBOMSHA256 string `json:"sbom_sha256,omitempty"`
	NoticesURL string `json:"notices_url"`
	SourceURL  string `json:"source_url"`
}

// BuildInput is the build-manifest request. release_id is a UUID; when empty
// a new UUIDv7 is generated. product_version is display only (§13.3).
type BuildInput struct {
	ReleaseID       string                    `json:"release_id,omitempty"`
	ProductVersion  string                    `json:"product_version"`
	ReleaseSequence string                    `json:"release_sequence"`
	CreatedAt       string                    `json:"created_at,omitempty"`
	SourceCommit    string                    `json:"source_commit"`
	BuildID         string                    `json:"build_id"`
	Components      api.ManifestComponents    `json:"components"`
	Compatibility   api.ManifestCompatibility `json:"compatibility"`
	Rollback        api.ManifestRollback      `json:"rollback"`
	Evidence        EvidenceInput             `json:"evidence"`
	Artifacts       []ArtifactInput           `json:"artifacts"`
}

// BuildManifest produces the canonical JSON of the signed object (§13.3) and
// validates it against the release manifest schema. It does not sign.
func BuildManifest(in BuildInput, now time.Time) (signed []byte, manifest api.ReleaseManifest, err error) {
	id := in.ReleaseID
	if id == "" {
		u, gerr := uuid.NewV7()
		if gerr != nil {
			return nil, api.ReleaseManifest{}, gerr
		}
		id = u.String()
	} else if _, err := uuid.Parse(id); err != nil {
		return nil, api.ReleaseManifest{}, fmt.Errorf("release: release_id: %w", err)
	}
	created := in.CreatedAt
	if created == "" {
		created = now.UTC().Truncate(time.Second).Format(time.RFC3339)
	}
	ts, err := time.Parse(time.RFC3339, created)
	if err != nil {
		return nil, api.ReleaseManifest{}, fmt.Errorf("release: created_at: %w", err)
	}
	seq, err := decstr.Parse(in.ReleaseSequence)
	if err != nil {
		return nil, api.ReleaseManifest{}, fmt.Errorf("release: release_sequence: %w", err)
	}
	arts, err := buildArtifacts(in.Artifacts)
	if err != nil {
		return nil, api.ReleaseManifest{}, err
	}
	if err := requireBaseline(arts); err != nil {
		return nil, api.ReleaseManifest{}, err
	}
	sbom := in.Evidence.SBOMSHA256
	if sbom == "" {
		if in.Evidence.SBOMPath == "" {
			return nil, api.ReleaseManifest{}, fmt.Errorf("release: evidence needs sbom_sha256 or sbom_path")
		}
		sum, _, err := fileSHA256(in.Evidence.SBOMPath)
		if err != nil {
			return nil, api.ReleaseManifest{}, fmt.Errorf("release: sbom: %w", err)
		}
		sbom = sum
	}
	manifest = api.ReleaseManifest{
		ManifestSchemaVersion: api.ReleaseManifestManifestSchemaVersionN1,
		ReleaseId:             uuid.MustParse(id),
		ProductVersion:        in.ProductVersion,
		ReleaseSequence:       seq,
		CreatedAt:             ts.UTC(),
		SourceCommit:          in.SourceCommit,
		BuildId:               in.BuildID,
		Components:            in.Components,
		Compatibility:         in.Compatibility,
		Artifacts:             arts,
		Rollback:              in.Rollback,
		Evidence: api.ManifestEvidence{
			SbomUrl:    in.Evidence.SBOMURL,
			SbomSha256: sbom,
			NoticesUrl: in.Evidence.NoticesURL,
			SourceUrl:  in.Evidence.SourceURL,
		},
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return nil, api.ReleaseManifest{}, err
	}
	signed, err = canonjson.Canonicalize(raw)
	if err != nil {
		return nil, api.ReleaseManifest{}, fmt.Errorf("release: canonicalize manifest: %w", err)
	}
	if err := ValidateSigned(signed); err != nil {
		return nil, api.ReleaseManifest{}, err
	}
	if err := requireBaseline(manifest.Artifacts); err != nil {
		return nil, api.ReleaseManifest{}, err
	}
	return signed, manifest, nil
}

func buildArtifacts(in []ArtifactInput) ([]api.ManifestArtifact, error) {
	if len(in) < 2 {
		return nil, fmt.Errorf("release: need amd64 and arm64 artifacts (§13.4)")
	}
	out := make([]api.ManifestArtifact, 0, len(in))
	seen := map[string]bool{}
	for _, a := range in {
		sum, size, err := fileSHA256(a.Path)
		if err != nil {
			return nil, fmt.Errorf("release: artifact %s: %w", a.Architecture, err)
		}
		installed, err := decstr.Parse(a.InstalledSizeBytes)
		if err != nil {
			return nil, fmt.Errorf("release: artifact %s installed_size_bytes: %w", a.Architecture, err)
		}
		key := a.OSID + "/" + a.Architecture
		if seen[key] {
			return nil, fmt.Errorf("release: duplicate artifact %s", key)
		}
		seen[key] = true
		out = append(out, api.ManifestArtifact{
			OsId:               api.ManifestArtifactOsId(a.OSID),
			OsVersions:         a.OSVersions,
			Architecture:       api.Architecture(a.Architecture),
			Format:             api.ManifestArtifactFormatTarZst,
			Url:                a.URL,
			SizeBytes:          decstr.U64(size),
			InstalledSizeBytes: installed,
			Sha256:             sum,
		})
	}
	return out, nil
}

// requireBaseline enforces the rule the schema cannot express: both amd64
// and arm64 Ubuntu 24.04 packages are present (§13.4).
func requireBaseline(arts []api.ManifestArtifact) error {
	need := map[string]bool{"amd64": false, "arm64": false}
	for _, a := range arts {
		if string(a.OsId) != BaselineOSID || string(a.Format) != string(api.ManifestArtifactFormatTarZst) {
			continue
		}
		if _, ok := need[string(a.Architecture)]; !ok {
			continue
		}
		for _, v := range a.OsVersions {
			if v == BaselineOSVersion {
				need[string(a.Architecture)] = true
			}
		}
	}
	for arch, ok := range need {
		if !ok {
			return fmt.Errorf("release: missing ubuntu %s %s artifact (§13.4)", BaselineOSVersion, arch)
		}
	}
	return nil
}
