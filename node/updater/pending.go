package updater

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// PendingReleaseFile is the installer hand-off name under the staging directory.
const PendingReleaseFile = "pending-release.json"

// SourceInstaller is the journal intent source for an installer-staged apply.
// The wire update result has no source member (additionalProperties: false).
const SourceInstaller = "installer"

// PendingRelease is the record packaging/installer writes. The writer emits
// release_id, manifest_sha256, artifact_path, tree_path and updater_command.
// manifest and manifest_path are optional. The current writer sets neither.
// tree_path is not installed. updater_command is not executed.
type PendingRelease struct {
	ReleaseID      string          `json:"release_id,omitempty"`
	ManifestSHA256 string          `json:"manifest_sha256,omitempty"`
	ArtifactPath   string          `json:"artifact_path,omitempty"`
	TreePath       string          `json:"tree_path,omitempty"`
	UpdaterCommand string          `json:"updater_command,omitempty"`
	ManifestPath   string          `json:"manifest_path,omitempty"`
	Manifest       json.RawMessage `json:"manifest,omitempty"`
	digest         string
}

// ApplyPending verifies a staged release and runs it through the update
// state machine. path is required. A missing file is an error. The oneshot
// uses applyPendingLocked with required false.
func (e *Engine) ApplyPending(ctx context.Context, path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("updater: pending path is required")
	}
	if err := e.open(ctx); err != nil {
		return err
	}
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	return e.withLock(ctx, func() error {
		// Recovery may seal this very pending job and consume its record.
		_, before := os.Lstat(path)
		recovered := e.recoverLocked(ctx)
		if recovered != nil && !errors.Is(recovered, ErrMembersBlocked) {
			return recovered
		}
		if err := e.applyPendingLocked(ctx, path, before != nil); err != nil {
			return err
		}
		if errors.Is(recovered, ErrMembersBlocked) {
			blocked, _ := e.Journal.BlocksForwarding(ctx)
			if blocked {
				return ErrMembersBlocked
			}
		}
		return nil
	})
}

func (e *Engine) pendingRecordPath() string {
	if e == nil || e.Layout.Staging == "" {
		return ""
	}
	return filepath.Join(e.Layout.Staging, PendingReleaseFile)
}

// applyPendingLocked applies path when it exists. required is false for the
// oneshot, where a missing file is no work.
func (e *Engine) applyPendingLocked(ctx context.Context, path string, required bool) error {
	if path == "" {
		return nil
	}
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, err := os.Lstat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) && !required {
			return nil
		}
		return err
	}
	_, err = e.applyPendingBody(ctx, path)
	return err
}

func (e *Engine) applyPendingBody(ctx context.Context, path string) (string, error) {
	rec, err := readPending(path)
	if err != nil {
		return "", err
	}
	releaseID, err := uuid.Parse(rec.ReleaseID)
	if err != nil || releaseID == uuid.Nil || !validSHA256(rec.ManifestSHA256) {
		return "", permanent(apierr.ValidationFailed, "pending release record")
	}
	if e.Node.BootID == uuid.Nil {
		return "", errors.New("updater: boot_id is required when the kernel boot id is unreadable")
	}
	c, err := e.installerClaim(ctx, path, releaseID, rec.ManifestSHA256)
	if err != nil {
		return "", err
	}
	c.PendingSHA256 = rec.digest
	// Content changes (for example, supplying a previously missing manifest)
	// are a new local request. Replays of the same record remain idempotent.
	c.Intent.JobId = uuid.NewSHA1(c.Intent.JobId, []byte(rec.digest+"|"+e.Node.Incarnation.String()))
	jobID := c.Intent.JobId.String()
	// Reuse the original claim rather than a new generation or boot result.
	// A retry must use the same intent and result bytes after a restart.
	stored, err := e.Journal.GetJob(ctx, jobID, 1)
	if err == nil {
		c, err = decodeClaimed(stored.IntentJSON)
		if err != nil {
			return jobID, err
		}
		if c.PendingSHA256 != rec.digest || c.PendingPath != path {
			return jobID, permanent(apierr.ValidationFailed, "pending record changed for an existing installer job")
		}
		if stored.Phase == updatejournal.JobDone {
			return jobID, e.consumeInstallerPending(c, stored.Outcome)
		}
	} else if errors.Is(err, updatejournal.ErrNotFound) {
		raw, err := json.Marshal(c)
		if err != nil {
			return jobID, err
		}
		stored, err = e.Journal.RecordJob(ctx, updatejournal.Job{
			JobID: jobID, Attempt: c.Intent.Attempt, Generation: c.Intent.Generation,
			ReleaseID: releaseID.String(), ManifestSHA256: strings.ToLower(rec.ManifestSHA256), IntentJSON: raw,
		})
		if err != nil {
			return jobID, err
		}
	} else {
		return jobID, err
	}
	fail := func(cause error) (string, error) {
		if retryable(cause) {
			return jobID, cause
		}
		code := apierr.ValidationFailed
		if op := asOp(cause); op != nil {
			code = op.Code
		}
		if err := e.failJob(ctx, stored, c, code, api.UpdateStageVerify); err != nil {
			return jobID, err
		}
		return jobID, cause
	}
	if e.Node.Revoked || e.drifted(c) {
		if err := e.noteBlocked(ctx, c, updatejournal.OutcomeFailed); err != nil {
			return jobID, err
		}
		return fail(permanent(apierr.IdentityRevoked, "installer node identity"))
	}
	if containsID(e.RevokedReleaseIDs, releaseID) || containsID(c.Revoked, releaseID) {
		return fail(permanent(apierr.ReleaseNotApproved, "revoked release"))
	}
	env, err := pendingEnvelope(rec)
	if err != nil {
		return fail(err)
	}
	manifest, art, err := e.verifyStaged(ctx, rec, env)
	if err != nil {
		return fail(err)
	}
	artifact, err := e.snapshotStagedArtifact(rec.ArtifactPath, art)
	if err != nil {
		return fail(err)
	}
	defer os.Remove(artifact)
	if err := release.CheckArtifact(manifest, e.Platform.Architecture, artifact); err != nil {
		return fail(artifactOp(err))
	}
	// An installer-created current pointer without updater metadata must not
	// be mistaken for an empty node (which would skip schema/rollback checks).
	pointers, err := e.Journal.Pointers(ctx)
	if err != nil {
		return jobID, err
	}
	current, err := e.Layout.currentID()
	if err != nil {
		return jobID, err
	}
	if current != pointers.CurrentRelease {
		return fail(permanent(apierr.CheckpointFailed, "current release has no matching updater journal"))
	}
	e.log().Info("applying installer pending release", "release_id", releaseID.String())
	if done, err := e.maybeAlreadyCurrent(ctx, c, stored); done || err != nil {
		return jobID, err
	}
	same, err := fsutil.SameFilesystem(e.Layout.Releases, filepath.Dir(e.Layout.Current))
	if err != nil || !same {
		return jobID, e.failInstaller(ctx, stored, c, permanent(apierr.StartFailed, "releases and current are not on the same filesystem"))
	}
	if err := e.installTree(ctx, c, manifest, env, art, artifact); err != nil {
		if retryable(err) {
			return jobID, err
		}
		return jobID, e.failInstaller(ctx, stored, c, err)
	}
	if err := e.setPhase(ctx, stored, updatejournal.JobApplying); err != nil {
		return jobID, err
	}
	txn, err := e.beginOrResume(ctx, c, manifest)
	if err != nil {
		return jobID, err
	}
	if err := e.continueTxn(ctx, txn); err != nil {
		return jobID, err
	}
	return jobID, nil
}

func (e *Engine) installerClaim(ctx context.Context, path string, releaseID uuid.UUID, manifestSHA string) (claimed, error) {
	gen, err := e.installerGeneration(ctx)
	if err != nil {
		return claimed{}, err
	}
	policy, err := e.installerPolicy(ctx)
	if err != nil {
		return claimed{}, err
	}
	g, err := e.Journal.GateRecord(ctx)
	if err != nil {
		return claimed{}, err
	}
	intent := publicationOrDisabled(g.PublicationIntent)
	revoked, err := e.signedRevokedIDs(ctx)
	if err != nil {
		return claimed{}, err
	}
	ptr, err := e.Journal.Pointers(ctx)
	if err != nil {
		return claimed{}, err
	}
	var rollback []uuid.UUID
	for _, raw := range []string{ptr.LastGood, ptr.CurrentRelease} {
		if id, err := uuid.Parse(raw); err == nil {
			rollback = unionUUIDs(rollback, []uuid.UUID{id})
		}
	}
	now := e.now()
	return claimed{
		Work: Work{
			Intent: api.UpdateIntent{
				Attempt:    1,
				Generation: gen, IncarnationId: e.Node.Incarnation,
				IssuedAt: now, JobId: installerJobID(releaseID.String(), manifestSHA),
				ManifestSha256: strings.ToLower(manifestSHA),
				NodeId:         e.Node.ID, PublicationIntent: intent, ReleaseId: releaseID,
				BatchId:  presence.NewNullNullable[uuid.UUID](),
				TargetId: presence.NewNullNullable[uuid.UUID](),
			},
			// A local request is not a serving authorization. The agent must
			// receive a valid desired state before member forwarding resumes.
			Revoked:          unionUUIDs(revoked, e.RevokedReleaseIDs),
			Serving:          rollback, // permits verified rollback; enforce keeps installer forwarding off
			PolicyGeneration: policy,
			PolicyKnown:      true,
		},
		BootID: e.Node.BootID, StateSchema: e.Platform.StateSchema,
		Source: SourceInstaller, PendingPath: path,
	}, nil
}

func installerJobID(releaseID, manifestSHA string) uuid.UUID {
	name := "fvpn-installer-apply|" + strings.ToLower(releaseID) + "|" + strings.ToLower(manifestSHA)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(name))
}

func (e *Engine) installerGeneration(ctx context.Context) (decstr.U64, error) {
	gen, _, ok, err := e.Journal.AcceptedGeneration(ctx)
	if err != nil {
		return 0, err
	}
	if !ok || gen == 0 {
		return decstr.MustParse("1"), nil
	}
	return gen, nil
}

func (e *Engine) installerPolicy(ctx context.Context) (decstr.U64, error) {
	gen, ok, err := e.Journal.AppliedConfig(ctx)
	if err != nil || !ok {
		return 0, err
	}
	return gen, nil
}

func publicationOrDisabled(s string) api.PublicationIntent {
	switch api.PublicationIntent(s) {
	case api.PublicationIntentPublished, api.PublicationIntentUnpublished, api.PublicationIntentDisabled:
		return api.PublicationIntent(s)
	default:
		return api.PublicationIntentDisabled
	}
}

func (e *Engine) verifyStaged(ctx context.Context, rec PendingRelease, env []byte) (api.ReleaseManifest, api.ManifestArtifact, error) {
	if e.ReleaseTrust == nil {
		return api.ReleaseManifest{}, api.ManifestArtifact{}, permanent(apierr.KeyUntrusted, "release trust")
	}
	manifest, err := VerifyManifest(env, rec.ManifestSHA256, e.ReleaseTrust, e.now())
	if err != nil {
		return api.ReleaseManifest{}, api.ManifestArtifact{}, err
	}
	if _, _, err := release.Verify(env, e.ReleaseTrust, e.now()); err != nil {
		if errors.Is(err, envelope.ErrKeyUntrusted) {
			return api.ReleaseManifest{}, api.ManifestArtifact{}, permanent(apierr.KeyUntrusted, "release signature")
		}
		return api.ReleaseManifest{}, api.ManifestArtifact{}, permanent(apierr.SignatureInvalid, "release signature")
	}
	id, parseErr := uuid.Parse(rec.ReleaseID)
	if parseErr != nil || manifest.ReleaseId != id {
		return api.ReleaseManifest{}, api.ManifestArtifact{}, permanent(apierr.SignatureInvalid, "release_id")
	}
	schema, _ := e.Journal.StateSchema(ctx)
	if schema == 0 {
		schema = e.Platform.StateSchema
	}
	p, _ := e.Journal.Pointers(ctx)
	if err := checkCompat(manifest, e.Platform, schema, p.CurrentRelease != ""); err != nil {
		return api.ReleaseManifest{}, api.ManifestArtifact{}, err
	}
	art, err := selectArtifact(manifest, e.Platform.OSID, e.Platform.OSVersion, e.Platform.Architecture)
	if err != nil {
		return api.ReleaseManifest{}, api.ManifestArtifact{}, err
	}
	need, ok := sumU64(art.SizeBytes.Uint64(), art.InstalledSizeBytes.Uint64(), manifest.Rollback.RequiredFreeBytes.Uint64())
	if !ok {
		return api.ReleaseManifest{}, api.ManifestArtifact{}, transient(apierr.DiskSpaceLow, "size overflow")
	}
	if err := e.ensureSpace(e.Layout.Releases, need); err != nil {
		return api.ReleaseManifest{}, api.ManifestArtifact{}, err
	}
	if err := e.ensureSpace(e.Layout.Staging, art.SizeBytes.Uint64()); err != nil {
		return api.ReleaseManifest{}, api.ManifestArtifact{}, err
	}
	return manifest, art, nil
}

func (e *Engine) failInstaller(ctx context.Context, job updatejournal.Job, c claimed, cause error) error {
	code := apierr.ValidationFailed
	if op := asOp(cause); op != nil {
		code = op.Code
	}
	if err := e.failJob(ctx, job, c, code, api.UpdateStageVerify); err != nil {
		return err
	}
	return cause
}

// readLocalFile opens without following the final symlink and bounds the read.
// O_NONBLOCK prevents a substituted FIFO from hanging the updater.
func openLocalFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() {
		_ = f.Close()
		return nil, permanent(apierr.ValidationFailed, "local input is not a regular file")
	}
	return f, nil
}

func readLocalFile(path string) ([]byte, error) {
	f, err := openLocalFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, canonjson.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > canonjson.MaxBytes {
		return nil, permanent(apierr.ValidationFailed, "local JSON input is too large")
	}
	return raw, nil
}

func readPending(path string) (PendingRelease, error) {
	raw, err := readLocalFile(path)
	if err != nil {
		return PendingRelease{}, err
	}
	canonical, err := canonjson.Canonicalize(raw)
	if err != nil {
		return PendingRelease{}, permanent(apierr.ValidationFailed, "pending release record")
	}
	var rec PendingRelease
	if err := json.Unmarshal(canonical, &rec); err != nil {
		return rec, permanent(apierr.ValidationFailed, "pending release record")
	}
	sum := sha256.Sum256(canonical)
	rec.digest = hex.EncodeToString(sum[:])
	return rec, nil
}

func pendingEnvelope(rec PendingRelease) ([]byte, error) {
	raw := bytes.TrimSpace(rec.Manifest)
	if len(raw) > 0 && string(raw) != "null" {
		return append([]byte(nil), raw...), nil
	}
	if rec.ManifestPath == "" {
		return nil, permanent(apierr.SignatureInvalid, "pending record has no signed manifest")
	}
	raw, err := readLocalFile(rec.ManifestPath)
	if err != nil {
		return nil, permanent(apierr.SignatureInvalid, "pending manifest is unreadable")
	}
	return raw, nil
}

// Copy to a private updater file before verification. Extraction reads only
// this verified snapshot, never a staging pathname that could be replaced.
func (e *Engine) snapshotStagedArtifact(path string, art api.ManifestArtifact) (string, error) {
	src, err := openLocalFile(path)
	if err != nil {
		return "", permanent(apierr.ArtifactInvalid, "staged artifact is unreadable")
	}
	defer src.Close()
	dst, err := os.CreateTemp(e.Layout.Staging, ".installer-*.tar.zst")
	if err != nil {
		return "", transient(apierr.StoreUnavailable, "staged artifact snapshot")
	}
	ok := false
	defer func() {
		_ = dst.Close()
		if !ok {
			_ = os.Remove(dst.Name())
		}
	}()
	size := art.SizeBytes.Uint64()
	if size >= 1<<63-1 {
		return "", permanent(apierr.SizeMismatch, "staged artifact")
	}
	n, err := io.Copy(dst, io.LimitReader(src, int64(size)+1))
	if err != nil {
		return "", transient(apierr.StoreUnavailable, "staged artifact snapshot")
	}
	if n != int64(size) {
		return "", permanent(apierr.SizeMismatch, "staged artifact")
	}
	if err := dst.Close(); err != nil {
		return "", err
	}
	ok = true
	return dst.Name(), nil
}

// Snapshots are needed only during verification/extraction under the lock.
// An active transaction resumes from its installed, journaled release tree.
func (e *Engine) cleanInstallerSnapshots() error {
	entries, err := os.ReadDir(e.Layout.Staging)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), ".installer-") && strings.HasSuffix(entry.Name(), ".tar.zst") {
			if err := os.Remove(filepath.Join(e.Layout.Staging, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

func artifactOp(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "HASH_MISMATCH"):
		return permanent(apierr.HashMismatch, "staged artifact")
	case strings.Contains(msg, "SIZE_MISMATCH"):
		return permanent(apierr.SizeMismatch, "staged artifact")
	case strings.Contains(msg, "UNSUPPORTED_PLATFORM"):
		return permanent(apierr.UnsupportedPlatform, "staged artifact")
	case strings.Contains(msg, "VALIDATION_FAILED"):
		return permanent(apierr.ValidationFailed, "staged artifact")
	default:
		return permanent(apierr.ArtifactInvalid, "staged artifact")
	}
}

func validSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

// finishInstaller records cleanup in the same durable transaction that seals
// the job, then renames the matching record and clears the cleanup obligation.
func (e *Engine) finishInstaller(ctx context.Context, c claimed, job updatejournal.Job, outcome, code string) error {
	if c.Source != SourceInstaller {
		return e.finish(ctx, job, outcome, code)
	}
	if err := e.Journal.FinishInstallerJob(ctx, job.JobID, job.Attempt, outcome, code); err != nil {
		return err
	}
	return e.consumeFinishedInstallerJobs(ctx)
}

func retryable(err error) bool {
	op := asOp(err)
	// Unclassified I/O failures and interruption leave the job available.
	return op == nil || op.Retry
}

func (e *Engine) consumeFinishedInstallerJobs(ctx context.Context) error {
	jobs, err := e.Journal.InstallerCleanups(ctx)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		c, err := decodeClaimed(job.IntentJSON)
		if err != nil {
			return err
		}
		if c.Source != SourceInstaller {
			return permanent(apierr.ValidationFailed, "installer cleanup claim")
		}
		if err := e.consumeInstallerPending(c, job.Outcome); err != nil {
			return err
		}
		if err := e.Journal.ClearInstallerCleanup(ctx, job.JobID, job.Attempt); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) consumeInstallerPending(c claimed, state string) error {
	if c.PendingPath == "" || c.PendingSHA256 == "" {
		return permanent(apierr.ValidationFailed, "installer cleanup identity")
	}
	rec, err := readPending(c.PendingPath)
	if errors.Is(err, os.ErrNotExist) {
		return fsutil.SyncDir(filepath.Dir(c.PendingPath))
	}
	if err != nil {
		return err
	}
	// A later installer may have staged a different record after a crash.
	// Completing the old job must never consume that new request.
	if rec.digest != c.PendingSHA256 {
		return nil
	}
	if !api.UpdateResultState(state).Valid() {
		return permanent(apierr.ValidationFailed, "installer result state")
	}
	dest := strings.TrimSuffix(c.PendingPath, ".json") + "." + state + ".json"
	if err := os.Rename(c.PendingPath, dest); err != nil {
		return err
	}
	return fsutil.SyncDir(filepath.Dir(c.PendingPath))
}
