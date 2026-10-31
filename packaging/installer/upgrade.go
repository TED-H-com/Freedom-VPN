package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// pendingRelease is the hand-off the running updater does not read today.
// fvpn-updater has no local-apply command (run, recover, flush, status,
// version only). upgrade stages a verified tree and starts `fvpn-updater run`.
// It does not move current and does not write the update journal (§13.6).
type pendingRelease struct {
	ReleaseID      string `json:"release_id"`
	ManifestSHA256 string `json:"manifest_sha256"`
	ArtifactPath   string `json:"artifact_path"`
	TreePath       string `json:"tree_path"`
	UpdaterCommand string `json:"updater_command"`
}

// Upgrade verifies an archive, stages it under releases/<release_id>, and
// asks the updater to run. A busy lock exits without staging (§13.5).
func Upgrade(ctx context.Context, env *Env, manifestPath, artifactPath, trustPath string) (*Result, error) {
	if env == nil {
		env = &Env{}
	}
	p, err := Identify(env)
	if err != nil {
		return nil, err
	}
	if err := requireSystemd(env); err != nil {
		return nil, err
	}
	verified, err := VerifyArchive(manifestPath, artifactPath, trustPath, p.Arch, env.now())
	if err != nil {
		return nil, err
	}
	l := env.layout()
	lk, err := tryLock(l.LockPath)
	if err != nil {
		if errors.Is(err, errBusy) {
			return nil, errf(ExitBusy, "busy", "update lock is held")
		}
		return nil, err
	}
	defer lk.Release()

	id := verified.Manifest.ReleaseId.String()
	dest := l.releaseDir(id)
	if err := os.MkdirAll(l.StagingDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(l.StagingDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(l.ReleasesDir, 0o755); err != nil {
		return nil, err
	}
	if err := extractRelease(artifactPath, dest, p.Arch, verified, l); err != nil {
		return nil, err
	}
	sum, err := fileSHA(artifactPath)
	if err != nil {
		return nil, err
	}
	staged := filepath.Join(l.StagingDir, sum+".tar.zst")
	if err := copyFile(artifactPath, staged, 0o644); err != nil {
		return nil, err
	}
	cmd := l.updaterBin() + " run -config " + l.updaterConfig()
	pending := pendingRelease{
		ReleaseID: id, ManifestSHA256: verified.SHA,
		ArtifactPath: staged, TreePath: dest, UpdaterCommand: cmd,
	}
	raw, err := marshalIndent(pending)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(l.pendingUpgrade(), raw, 0o600); err != nil {
		return nil, err
	}
	if err := lk.Release(); err != nil {
		return nil, err
	}
	// The updater takes the same lock. Do not hold it across run.
	if _, err := os.Stat(l.updaterConfig()); err != nil {
		return &Result{
			Status: statusStaged, ReleaseID: id, ManifestSHA256: verified.SHA,
			UpdaterCommand: cmd,
			Message:        "staged; updater config is missing so the updater was not started",
		}, errf(ExitFailed, "updater_failed", "updater config is missing")
	}
	if err := env.services().Start(ctx, UnitUpdater); err != nil {
		return &Result{
			Status: statusStaged, ReleaseID: id, ManifestSHA256: verified.SHA,
			UpdaterCommand: cmd,
			Message:        "staged; updater run failed and current was not moved",
		}, errf(ExitFailed, "updater_failed", "updater run failed")
	}
	return &Result{
		Status: statusStaged, ReleaseID: id, ManifestSHA256: verified.SHA,
		UpdaterCommand: cmd,
		Message:        "staged; current was not moved",
	}, nil
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
