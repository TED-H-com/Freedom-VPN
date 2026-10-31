package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// Seed verifies the installer's current release without switching, recovering,
// restarting services or claiming health. Only verified evidence is committed.
func (e *Engine) Seed(ctx context.Context, manifestPath, tree string) error {
	if strings.TrimSpace(manifestPath) == "" || strings.TrimSpace(tree) == "" {
		return permanent(apierr.ValidationFailed, "seed requires manifest and tree")
	}
	if err := e.open(ctx); err != nil {
		return err
	}
	return e.withLock(ctx, func() error { return e.seedLocked(ctx, manifestPath, tree) })
}

func (e *Engine) seedLocked(ctx context.Context, manifestPath, tree string) error {
	if e.Node.Revoked || e.Node.BootID == uuid.Nil {
		return permanent(apierr.IdentityRevoked, "seed requires a valid node and boot identity")
	}
	raw, err := readLocalFile(manifestPath)
	if err != nil {
		return err
	}
	if e.ReleaseTrust == nil {
		return permanent(apierr.KeyUntrusted, "release trust")
	}
	signed, _, err := release.Verify(raw, e.ReleaseTrust, e.now())
	if err != nil {
		return permanent(apierr.SignatureInvalid, "seed release signature or schema")
	}
	m, err := release.ParsedManifest(signed)
	if err != nil {
		return err
	}
	revoked, err := e.signedRevokedIDs(ctx)
	if err != nil {
		return err
	}
	if containsID(revoked, m.ReleaseId) || containsID(e.RevokedReleaseIDs, m.ReleaseId) {
		return permanent(apierr.ReleaseNotApproved, "seed release is revoked")
	}
	if err := checkCompat(m, e.Platform, 0, false); err != nil {
		return err
	}
	if e.Platform.StateSchema != m.Compatibility.StateSchemaTo {
		return permanent(apierr.SchemaIncompatible, "seed state schema differs")
	}
	if err := e.seedTreePath(tree, m.ReleaseId.String()); err != nil {
		return err
	}
	pointers, err := e.Journal.Pointers(ctx)
	if err != nil {
		return err
	}
	if pointers.CurrentRelease != "" && pointers.CurrentRelease != m.ReleaseId.String() {
		return permanent(apierr.CheckpointFailed, "seed refused: current release differs from journal")
	}
	manifestSHA, err := release.EnvelopeSHA256(raw)
	if err != nil {
		return err
	}
	art, err := selectArtifact(m, e.Platform.OSID, e.Platform.OSVersion, e.Platform.Architecture)
	if err != nil {
		return err
	}
	treeHash, err := e.verifySeedTree(ctx, tree, m, art)
	if err != nil {
		return err
	}
	// Recheck the live pointer before the atomic commit; shared writers hold
	// the node lock for the whole verification interval.
	if err := e.seedTreePath(tree, m.ReleaseId.String()); err != nil {
		return err
	}
	if err := e.Journal.SeedBaseline(ctx, updatejournal.Release{
		ReleaseID: m.ReleaseId.String(), ManifestSHA256: manifestSHA, ManifestEnvelope: raw,
		ArtifactSHA256: art.Sha256, TreeHash: treeHash, StateSchemaTo: m.Compatibility.StateSchemaTo,
	}, e.Node.BootID.String()); err != nil {
		return permanent(apierr.CheckpointFailed, err.Error())
	}
	e.log().Info("verified installer baseline seeded", "release_id", m.ReleaseId.String())
	return nil
}

func (e *Engine) seedTreePath(tree, id string) error {
	abs, err := filepath.Abs(tree)
	if err != nil {
		return err
	}
	expected, err := filepath.Abs(e.Layout.releaseDir(id))
	if err != nil {
		return err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil || resolved != abs || abs != expected {
		return permanent(apierr.CheckpointFailed, "seed tree must be the resolved release directory")
	}
	st, err := os.Lstat(abs)
	if err != nil || !st.IsDir() {
		return permanent(apierr.CheckpointFailed, "seed tree is not a directory")
	}
	st, err = os.Lstat(e.Layout.Current)
	if err != nil || st.Mode()&os.ModeSymlink == 0 {
		return permanent(apierr.CheckpointFailed, "seed requires the current symlink")
	}
	current, err := filepath.EvalSymlinks(e.Layout.Current)
	if err != nil {
		return err
	}
	current, err = filepath.Abs(current)
	if err != nil || current != abs {
		return permanent(apierr.CheckpointFailed, "seed tree differs from current")
	}
	return nil
}

// The manifest authenticates the archive, not a per-file inventory. Use the
// apply path's artifact verifier, safe extractor and full tree hash to derive
// that inventory. Never promote a hash computed solely from an untrusted tree.
func (e *Engine) verifySeedTree(ctx context.Context, tree string, m api.ReleaseManifest, art api.ManifestArtifact) (string, error) {
	if art.SizeBytes.Uint64() >= 1<<63-1 || art.InstalledSizeBytes.Uint64() >= 1<<63-1 {
		return "", permanent(apierr.SizeMismatch, "seed artifact size")
	}
	need, ok := sumU64(art.SizeBytes.Uint64(), art.InstalledSizeBytes.Uint64())
	if !ok {
		return "", transient(apierr.DiskSpaceLow, "seed size overflow")
	}
	if err := e.ensureSpace(e.Layout.Staging, need); err != nil {
		return "", err
	}
	scratch, err := os.MkdirTemp(e.Layout.Staging, ".seed-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(scratch)
	archive := filepath.Join(scratch, "artifact.tar.zst")
	cached := filepath.Join(e.Layout.Staging, art.Sha256+".tar.zst")
	if _, err := os.Lstat(cached); err == nil {
		archive, err = e.snapshotStagedArtifact(cached, art)
		if err != nil {
			return "", err
		}
		defer os.Remove(archive)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	} else if err := e.download(ctx, art.Url, archive, art.Sha256, hashRaw, int64(art.SizeBytes), int64(art.SizeBytes), nil); err != nil {
		return "", err
	}
	if err := release.CheckArtifact(m, e.Platform.Architecture, archive); err != nil {
		return "", artifactOp(err)
	}
	f, err := os.Open(archive)
	if err != nil {
		return "", err
	}
	defer f.Close()
	reference := filepath.Join(scratch, "tree")
	if err := ExtractArchive(f, reference, ExtractOptions{MaxBytes: int64(art.InstalledSizeBytes), Deny: e.Layout.PersistentDirs()}); err != nil {
		return "", err
	}
	want, err := fsutil.TreeHash(reference)
	if err != nil {
		return "", err
	}
	got, err := fsutil.TreeHash(tree)
	if err != nil {
		return "", permanent(apierr.HashMismatch, "seed tree contains an unreadable or unsupported entry")
	}
	if got != want {
		return "", permanent(apierr.HashMismatch, "seed tree differs from the signed artifact")
	}
	return got, nil
}
