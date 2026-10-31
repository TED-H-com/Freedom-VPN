package installer

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// releaseRE matches the agent_release pin. `latest` is rejected (§13.4).
var releaseRE = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z.+_-]{0,63}$`)

// Verified is a release archive the installer may stage (§13.4, PKG01).
type Verified struct {
	Manifest api.ReleaseManifest
	SHA      string
	Signed   []byte
	Raw      []byte
}

// VerifyArchive checks the signed manifest and the artifact hash by calling
// packaging/release. It does not reimplement the signature check.
func VerifyArchive(manifestPath, artifactPath, trustPath, arch string, now time.Time) (Verified, error) {
	if err := refuseLatestPath(manifestPath); err != nil {
		return Verified{}, err
	}
	if err := refuseLatestPath(artifactPath); err != nil {
		return Verified{}, err
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Verified{}, errf(ExitFailed, "failed", "cannot read manifest")
	}
	trustRaw, err := os.ReadFile(trustPath)
	if err != nil {
		return Verified{}, errf(ExitFailed, "failed", "cannot read trust root")
	}
	ts, err := envelope.ParseNodeTrustStoreJSON(trustRaw)
	if err != nil {
		return Verified{}, errf(ExitFailed, "failed", "trust root is not a node trust store")
	}
	signed, _, err := release.Verify(raw, ts, now)
	if err != nil {
		return Verified{}, errf(ExitFailed, "failed", "manifest signature check failed")
	}
	manifest, err := release.ParsedManifest(signed)
	if err != nil {
		return Verified{}, errf(ExitFailed, "failed", "manifest is not readable")
	}
	if err := refuseLatestManifest(manifest); err != nil {
		return Verified{}, err
	}
	if _, err := uuid.Parse(manifest.ReleaseId.String()); err != nil || manifest.ReleaseId == uuid.Nil {
		return Verified{}, errf(ExitFailed, "failed", "release_id is not a UUID")
	}
	if manifest.Components.Netbird != NetBirdPin {
		return Verified{}, errf(ExitDependency, "NETBIRD_VERSION", "manifest netbird component is %s, want %s", manifest.Components.Netbird, NetBirdPin)
	}
	for _, comp := range []string{manifest.Components.NodeAgent, manifest.Components.NodeUpdater, manifest.Components.Enforcement} {
		if comp == "" || comp == "latest" || !releaseRE.MatchString(comp) {
			return Verified{}, errf(ExitFailed, "failed", "component version is not a pin")
		}
	}
	if arch == "" {
		arch = "amd64"
	}
	if err := release.CheckArtifact(manifest, arch, artifactPath); err != nil {
		return Verified{}, errf(ExitFailed, "failed", "artifact check failed")
	}
	sum, err := release.EnvelopeSHA256(raw)
	if err != nil {
		return Verified{}, errf(ExitFailed, "failed", "manifest digest failed")
	}
	return Verified{Manifest: manifest, SHA: sum, Signed: signed, Raw: raw}, nil
}

func refuseLatestPath(path string) error {
	base := strings.ToLower(filepath.Base(path))
	if base == "latest" || base == "latest.tar.zst" || base == "latest.json" {
		return errf(ExitFailed, "failed", "refusing a latest tag")
	}
	low := strings.ToLower(filepath.ToSlash(path))
	if strings.Contains(low, "/latest/") || strings.HasSuffix(low, "/latest") {
		return errf(ExitFailed, "failed", "refusing a latest tag")
	}
	return nil
}

func refuseLatestManifest(m api.ReleaseManifest) error {
	if strings.EqualFold(m.ProductVersion, "latest") {
		return errf(ExitFailed, "failed", "refusing product_version latest")
	}
	if strings.EqualFold(m.Components.Netbird, "latest") || strings.EqualFold(m.Components.NodeAgent, "latest") {
		return errf(ExitFailed, "failed", "refusing component version latest")
	}
	return nil
}
