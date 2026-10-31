package installer

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// pendingRelease hands verification inputs to the updater's journaled apply
// path (D-080, D-089). TreePath and UpdaterCommand are informational only.
type pendingRelease struct {
	ReleaseID      string `json:"release_id"`
	ManifestPath   string `json:"manifest_path"`
	RequestID      string `json:"request_id"`
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
	lk, err := tryLock(ctx, l)
	if err != nil {
		if errors.Is(err, errBusy) {
			return nil, errf(ExitBusy, "busy", "update lock is held")
		}
		return nil, err
	}
	defer lk.Release()
	cur, err := upgradeCurrent(ctx, env)
	if err != nil {
		return nil, err
	}
	id := verified.Manifest.ReleaseId.String()
	if seqCmp(cur.ReleaseSequence, verified.Manifest.ReleaseSequence.String()) > 0 ||
		(cur.ReleaseID != id && seqCmp(cur.ReleaseSequence, verified.Manifest.ReleaseSequence.String()) == 0) {
		return nil, errf(ExitFailed, "DOWNGRADE", "refusing to replace release %s with %s", cur.ReleaseID, id)
	}
	if cur.ReleaseID == id && cur.ManifestSHA256 != verified.SHA {
		return nil, errf(ExitFailed, "IMMUTABLE", "release %s is already installed with a different manifest", id)
	}
	if err := reconcileMarker(ctx, env); err != nil {
		return nil, err
	}

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
	manifest := l.releaseManifest(id)
	if err := fsutil.WriteFileAtomic(manifest, verified.Raw, 0o600); err != nil {
		return nil, err
	}
	requestID := uuid.NewString()
	cmd := l.updaterBin() + " run -config " + l.updaterConfig()
	pending := pendingRelease{
		ReleaseID: id, ManifestSHA256: verified.SHA, ManifestPath: manifest, RequestID: requestID,
		ArtifactPath: staged, TreePath: dest, UpdaterCommand: cmd,
	}
	raw, err := marshalIndent(pending)
	if err != nil {
		return nil, err
	}
	if err := fsutil.WriteFileAtomic(l.pendingUpgrade(), raw, 0o600); err != nil {
		return nil, err
	}
	if err := lk.Release(); err != nil {
		return nil, err
	}
	// The updater takes the same lock. Do not hold it across run.
	if _, err := os.Stat(l.updaterConfig()); err != nil {
		return &Result{
			Status: statusStaged, ReleaseID: id, ManifestSHA256: verified.SHA,
			ManifestPath: manifest, RequestID: requestID,
			UpdaterCommand: cmd,
			Message:        "staged; updater config is missing so the updater was not started",
		}, errf(ExitFailed, "updater_failed", "updater config is missing")
	}
	if err := env.services().Start(ctx, UnitUpdater); err != nil {
		return &Result{
			Status: statusStaged, ReleaseID: id, ManifestSHA256: verified.SHA,
			ManifestPath: manifest, RequestID: requestID,
			UpdaterCommand: cmd,
			Message:        "staged; updater run failed; inspect fvpn-updater status",
		}, errf(ExitFailed, "updater_failed", "updater run failed")
	}
	return &Result{
		Status: statusStaged, ReleaseID: id, ManifestSHA256: verified.SHA,
		ManifestPath: manifest, RequestID: requestID,
		UpdaterCommand: cmd,
		Message:        "staged; updater run completed; inspect fvpn-updater status for the apply result",
	}, nil
}

// upgradeCurrent runs under the shared lock, before any staging or marker
// mutation. installed.json is install history: after a journaled switch the
// immutable, digest-checked journal envelope supplies the active sequence.
// Like the agent's release identity reader, this opens SQLite read-only and
// never initializes or migrates it. Unreadable evidence must not permit staging.
func upgradeCurrent(ctx context.Context, env *Env) (installedState, error) {
	l := env.layout()
	refused := func() (installedState, error) {
		return installedState{}, errf(ExitFailed, "installed_state", "cannot establish the installed release sequence")
	}
	cur, has := readInstalled(l)
	if !has {
		return refused()
	}
	if _, err := decstr.Parse(cur.ReleaseSequence); err != nil {
		return refused()
	}
	target, err := filepath.EvalSymlinks(l.CurrentLink)
	if err != nil {
		return refused()
	}
	if target != l.releaseDir(cur.ReleaseID) || readBaseline(l).Status == "seeded" {
		st, err := readUpdaterStatus(ctx, env, true)
		if err != nil || st == nil || st.CurrentReleaseID == "" || st.RecoveryRequired {
			return refused()
		}
		u := url.URL{Scheme: "file", Path: filepath.Join(l.UpdateJournal, "journal.db"), RawQuery: "mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(50)"}
		db, err := sql.Open("sqlite", u.String())
		if err != nil {
			return refused()
		}
		defer db.Close()
		var raw []byte
		var digest string
		if err := db.QueryRowContext(ctx, `SELECT manifest_envelope, manifest_sha256 FROM releases WHERE release_id = ?`, st.CurrentReleaseID).Scan(&raw, &digest); err != nil {
			return refused()
		}
		hash, err := canonjson.HashHex(raw)
		if err != nil || hash != digest {
			return refused()
		}
		var record struct {
			Signed api.ReleaseManifest `json:"signed"`
		}
		if json.Unmarshal(raw, &record) != nil || record.Signed.ReleaseId.String() != st.CurrentReleaseID {
			return refused()
		}
		cur.ReleaseID, cur.ManifestSHA256 = st.CurrentReleaseID, digest
		cur.ReleaseSequence = record.Signed.ReleaseSequence.String()
	}
	if _, err := decstr.Parse(cur.ReleaseSequence); err != nil {
		return refused()
	}
	id, err := uuid.Parse(cur.ReleaseID)
	if err != nil || id == uuid.Nil || id.String() != cur.ReleaseID {
		return refused()
	}
	if target != l.releaseDir(cur.ReleaseID) {
		return refused()
	}
	return cur, nil
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
