package updater

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// claimed is the job snapshot stored in the journal. Recovery finishes the
// transaction from this snapshot when the control plane is down (§13.6).
type claimed struct {
	Work
	BootID      uuid.UUID `json:"boot_id"`
	StateSchema int       `json:"state_schema"`
}

// Engine runs one node's update transactions (§13.4 to §13.9).
type Engine struct {
	Layout        Layout
	Node          Node
	Platform      Platform
	ReleaseTrust  *envelope.TrustStore
	DesiredTrust  *envelope.TrustStore
	Source        Source
	HTTP          *http.Client
	Meter         Meter
	Services      ServiceManager
	Health        HealthChecks
	Gate          Gate
	Migrator      Migrator
	Sender        ResultSink
	Journal       *updatejournal.Journal
	Download      DownloadConfig
	Hooks         Hooks
	Now           func() time.Time
	Sleep         func(time.Duration)
	Jitter        func() float64
	FreeBytes     func(path string) (uint64, error)
	AllowLoopback bool
	Faults        Faults
	Log           *slog.Logger
	// lockHeld is true while this process already owns the update lock.
	// Flush and Retain run from inside Run and must not take it again.
	lockHeld bool
}

// Close closes the journal when this engine opened it.
func (e *Engine) Close() error {
	if e.Journal == nil {
		return nil
	}
	return e.Journal.Close()
}

// Run applies the current authorized job, resuming a journaled transaction
// first (§13.5). Nil means the local transaction finished or there was no
// work. It does not mean the center has accepted the result.
func (e *Engine) Run(ctx context.Context) error {
	if err := e.open(ctx); err != nil {
		return err
	}
	return e.withLock(ctx, func() error {
		blocked := false
		if err := e.Recover(ctx); err != nil {
			if !errors.Is(err, ErrMembersBlocked) {
				return err
			}
			blocked = true
		}
		for i := 0; i < 8; i++ {
			again, err := e.runOnce(ctx)
			if err != nil {
				return err
			}
			if !again {
				if blocked {
					if on, _ := e.Journal.BlocksForwarding(ctx); on {
						return ErrMembersBlocked
					}
				}
				return nil
			}
			blocked = false
		}
		return permanent(apierr.StaleGeneration, "too many superseded generations")
	})
}

// withLock acquires the node update lock before any journaled update work
// other than opening the database. A caller that already holds it, such as
// Flush during seal, runs fn directly (D-034d).
func (e *Engine) withLock(ctx context.Context, fn func() error) error {
	if e.lockHeld {
		return fn()
	}
	lk, err := nodelock.Acquire(ctx, e.Layout.Lock, 5, 20*time.Millisecond)
	if err != nil {
		if errors.Is(err, nodelock.ErrBusy) {
			return transient(apierr.UpdateBusy, "lock")
		}
		return err
	}
	defer lk.Release()
	e.lockHeld = true
	defer func() { e.lockHeld = false }()
	return fn()
}

// Recover finishes an applying or checking transaction before member
// services start (§13.5). ErrMembersBlocked means forwarding stays off.
func (e *Engine) Recover(ctx context.Context) error {
	if err := e.open(ctx); err != nil {
		return err
	}
	return e.withLock(ctx, func() error { return e.recoverLocked(ctx) })
}

func (e *Engine) recoverLocked(ctx context.Context) error {
	txn, err := e.Journal.ActiveTxn(ctx)
	if err == nil {
		if err := e.continueTxn(ctx, txn); err != nil {
			return err
		}
	} else if !errors.Is(err, updatejournal.ErrNotFound) {
		return err
	}
	if err := e.ensureReported(ctx); err != nil {
		return err
	}
	if err := e.holdIfRevoked(ctx); err != nil {
		return err
	}
	on, _ := e.Journal.BlocksForwarding(ctx)
	if on {
		if e.Gate != nil {
			_ = e.Gate.Block(ctx)
		}
		return ErrMembersBlocked
	}
	return e.reapplyPublication(ctx)
}

func (e *Engine) open(context.Context) error {
	if err := e.Layout.Ensure(); err != nil {
		return err
	}
	if e.Journal != nil {
		return nil
	}
	j, err := updatejournal.Open(e.Layout.Journal, updatejournal.Options{Now: e.now})
	if err != nil {
		return err
	}
	e.Journal = j
	return nil
}

func (e *Engine) runOnce(ctx context.Context) (bool, error) {
	if e.Source == nil {
		return false, errors.New("updater: no desired-state source")
	}
	work, err := e.Source.Current(ctx)
	noWork := errors.Is(err, ErrNoWork)
	if err != nil && !noWork {
		return false, err
	}
	// A legacy envelope with no job carries no policy and no signed set.
	// Leave the latch alone, as before.
	if noWork && work.PolicyGeneration == 0 && !work.PolicyKnown && work.SignedRevoked == nil {
		return false, nil
	}
	if err := e.acceptDesired(ctx, &work); err != nil {
		return false, err
	}
	if err := e.rollbackIfRevoked(ctx, e.claimFrom(work)); err != nil {
		return false, err
	}
	if noWork {
		return false, nil
	}
	if err := e.preflight(ctx, work); err != nil {
		return false, err
	}
	if stop, err := e.rejectStale(ctx, work); stop || err != nil {
		return false, err
	}
	if err := e.supersedeOlder(ctx, work); err != nil {
		return false, err
	}
	if !authorized(work, e.now()) {
		return false, transient(apierr.AuthorizationExpired, "update authorization")
	}
	c := e.claimFrom(work)
	raw, err := json.Marshal(c)
	if err != nil {
		return false, err
	}
	stored, _, err := e.Journal.ClaimJob(ctx, updatejournal.Job{
		JobID:          c.Intent.JobId.String(),
		Attempt:        c.Intent.Attempt,
		Generation:     c.Intent.Generation,
		ReleaseID:      c.Intent.ReleaseId.String(),
		ManifestSHA256: strings.ToLower(c.Intent.ManifestSha256),
		IntentJSON:     raw,
	})
	if errors.Is(err, updatejournal.ErrStaleGeneration) {
		return false, permanent(apierr.StaleGeneration, "claim")
	}
	if err != nil {
		return false, err
	}
	if stored.Phase == updatejournal.JobDone {
		return false, nil
	}
	if decoded, err := decodeClaimed(stored.IntentJSON); err == nil {
		c = decoded
	}
	if err := e.hookPhase(ctx, "claimed", c); err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStageClaim)
	}
	if done, err := e.maybeAlreadyCurrent(ctx, c, stored); done || err != nil {
		return false, err
	}
	if err := e.setPhase(ctx, stored, updatejournal.JobDownloading); err != nil {
		return false, err
	}
	if err := e.hookPhase(ctx, "downloading", c); err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStageDownload)
	}
	env, manifest, err := e.fetchManifest(ctx, c)
	if err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStageDownload)
	}
	if err := e.hookPhase(ctx, "verifying", c); err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStageVerify)
	}
	if err := e.setPhase(ctx, stored, updatejournal.JobVerifying); err != nil {
		return false, err
	}
	art, err := e.verifyRelease(ctx, c, manifest)
	if err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStageVerify)
	}
	path, err := e.fetchArtifact(ctx, c, art)
	if err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStageDownload)
	}
	if err := e.hookPhase(ctx, "verified", c); err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStageVerify)
	}
	if err := e.installTree(ctx, c, manifest, env, art, path); err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStageVerify)
	}
	if err := e.setPhase(ctx, stored, updatejournal.JobVerified); err != nil {
		return false, err
	}
	if !authorized(c.Work, e.now()) {
		return false, transient(apierr.AuthorizationExpired, "authorization ended before apply")
	}
	same, err := fsutil.SameFilesystem(e.Layout.Releases, filepath.Dir(e.Layout.Current))
	if err != nil || !same {
		return e.afterPrepareErr(ctx, stored, c, permanent(apierr.StartFailed, "releases and current are not on the same filesystem"), api.UpdateStagePrepare)
	}
	if err := e.catchNewer(ctx, c); err != nil {
		return e.afterPrepareErr(ctx, stored, c, err, api.UpdateStagePrepare)
	}
	if err := e.setPhase(ctx, stored, updatejournal.JobApplying); err != nil {
		return false, err
	}
	txn, err := e.beginOrResume(ctx, c, manifest)
	if err != nil {
		return false, err
	}
	if err := e.continueTxn(ctx, txn); err != nil {
		return false, err
	}
	// A newer generation that arrived while this transaction was applying
	// waits until the transaction is terminal (§13.8), then runs.
	if err := e.catchNewer(ctx, c); errors.Is(err, ErrSuperseded) {
		return true, nil
	}
	return false, nil
}

func (e *Engine) preflight(ctx context.Context, work Work) error {
	if work.Intent.Attempt < 1 {
		return permanent(apierr.SignatureInvalid, "attempt")
	}
	c := e.claimFrom(work)
	if e.Node.Revoked {
		return e.reportUnclaimed(ctx, c, api.UpdateResultStateFailed, api.UpdateStageClaim, apierr.IdentityRevoked)
	}
	if work.Intent.NodeId != e.Node.ID || work.Intent.IncarnationId != e.Node.Incarnation {
		return e.reportUnclaimed(ctx, c, api.UpdateResultStateFailed, api.UpdateStageClaim, apierr.IncarnationMismatch)
	}
	if containsID(work.Revoked, work.Intent.ReleaseId) {
		return e.reportUnclaimed(ctx, c, api.UpdateResultStateFailed, api.UpdateStagePreflight, apierr.ReleaseNotApproved)
	}
	return nil
}

func (e *Engine) rejectStale(ctx context.Context, work Work) (bool, error) {
	gen, _, ok, err := e.Journal.AcceptedGeneration(ctx)
	if err != nil || !ok {
		return false, err
	}
	if work.Intent.Generation >= gen {
		return false, nil
	}
	job, err := e.Journal.GetJob(ctx, work.Intent.JobId.String(), work.Intent.Attempt)
	if err == nil && job.Phase != updatejournal.JobDone {
		c, decErr := decodeClaimed(job.IntentJSON)
		if decErr != nil {
			c = e.claimFrom(work)
		}
		_, ferr := e.finishSupersede(ctx, job, c)
		return true, ferr
	}
	c := e.claimFrom(work)
	return true, e.reportUnclaimed(ctx, c, api.UpdateResultStateSuperseded, api.UpdateStagePreflight, apierr.StaleGeneration)
}

func (e *Engine) supersedeOlder(ctx context.Context, work Work) error {
	open, err := e.Journal.OpenJobs(ctx)
	if err != nil {
		return err
	}
	active, aerr := e.Journal.ActiveTxn(ctx)
	for _, job := range open {
		if aerr == nil && active.JobID == job.JobID && active.Attempt == job.Attempt {
			continue
		}
		if job.JobID == work.Intent.JobId.String() && job.Attempt == work.Intent.Attempt && job.Generation == work.Intent.Generation && strings.EqualFold(job.ManifestSHA256, work.Intent.ManifestSha256) {
			continue
		}
		if job.Generation > work.Intent.Generation {
			continue
		}
		c, err := decodeClaimed(job.IntentJSON)
		if err != nil {
			return err
		}
		if _, err := e.finishSupersede(ctx, job, c); err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) maybeAlreadyCurrent(ctx context.Context, c claimed, job updatejournal.Job) (bool, error) {
	p, err := e.Journal.Pointers(ctx)
	if err != nil {
		return false, err
	}
	if p.CurrentRelease != c.Intent.ReleaseId.String() {
		return false, nil
	}
	rec, err := e.Journal.GateRecord(ctx)
	if err != nil {
		return false, err
	}
	blocked, _ := e.Journal.BlocksForwarding(ctx)
	// Same release id is not "already current" when the pointers disagree,
	// forwarding is held, or the last terminal outcome failed (D-034b).
	if p.CurrentRelease == "" || p.CurrentRelease != p.LastGood || blocked || terminalBlocksAlready(rec.TerminalOutcome) {
		return false, nil
	}
	if link, _ := e.Layout.currentID(); link != "" && link != p.CurrentRelease {
		return false, nil
	}
	rel, err := e.Journal.GetRelease(ctx, p.CurrentRelease)
	if err != nil {
		return false, nil
	}
	if !strings.EqualFold(rel.ManifestSHA256, c.Intent.ManifestSha256) {
		return false, nil
	}
	manifest, err := VerifyManifest(rel.ManifestEnvelope, rel.ManifestSHA256, e.ReleaseTrust, e.now())
	if err != nil {
		code := apierr.SignatureInvalid
		if op := asOp(err); op != nil {
			code = op.Code
		}
		if berr := e.noteBlocked(ctx, c, updatejournal.OutcomeFailed); berr != nil {
			return true, berr
		}
		return e.failRecorded(ctx, job, c, code, api.UpdateStagePreflight, err)
	}
	schema, _ := e.Journal.StateSchema(ctx)
	if err := checkCompat(manifest, e.Platform, schema, true); err != nil {
		code := apierr.SchemaIncompatible
		if op := asOp(err); op != nil {
			code = op.Code
		}
		if berr := e.noteBlocked(ctx, c, updatejournal.OutcomeFailed); berr != nil {
			return true, berr
		}
		return e.failRecorded(ctx, job, c, code, api.UpdateStagePreflight, err)
	}
	if !containsID(c.Serving, c.Intent.ReleaseId) {
		// The release is already current, but it is not allowed to serve.
		// Record the failure and return it. Do not restart services (§13.9).
		if berr := e.noteBlocked(ctx, c, updatejournal.OutcomeFailed); berr != nil {
			return true, berr
		}
		return e.failRecorded(ctx, job, c, apierr.MinimumVersionViolation, api.UpdateStagePreflight, nil)
	}
	enabled, err := e.enforce(ctx, p.CurrentRelease, c, updatejournal.OutcomeAlreadyCurrent)
	if err != nil {
		return true, err
	}
	in := resultInput{
		Claimed: c, State: api.UpdateResultStateAlreadyCurrent, Stage: api.UpdateStagePreflight,
		Current: p.CurrentRelease, Previous: p.CurrentRelease, Forwarding: enabled,
		Started: job.ClaimedAt, Finished: job.ClaimedAt, Schema: schemaAtLeast1(schema),
	}
	if err := e.enqueueResult(ctx, in); err != nil {
		return true, err
	}
	if err := e.finish(ctx, job, string(api.UpdateResultStateAlreadyCurrent), ""); err != nil {
		return true, err
	}
	return true, nil
}

func terminalBlocksAlready(outcome string) bool {
	switch outcome {
	case updatejournal.OutcomeFailed, updatejournal.OutcomeRolledBack, updatejournal.OutcomeRecoveryRequired:
		return true
	default:
		return false
	}
}

func (e *Engine) noteBlocked(ctx context.Context, c claimed, outcome string) error {
	if e.Journal != nil {
		if err := e.Journal.SetGate(ctx, updatejournal.GateRecord{
			PublicationIntent: string(c.Intent.PublicationIntent),
			ForwardingBlocked: true,
			TerminalOutcome:   outcome,
		}); err != nil {
			return err
		}
	}
	if e.Gate == nil {
		return nil
	}
	if err := e.Gate.Block(ctx); err != nil {
		return wrapCode(apierr.UpdateBusy, err, true)
	}
	return nil
}

func (e *Engine) fetchManifest(ctx context.Context, c claimed) ([]byte, api.ReleaseManifest, error) {
	dest := filepath.Join(e.Layout.Staging, "manifest-"+strings.ToLower(c.Intent.ManifestSha256)+".json")
	poll := func() error { return e.catchNewer(ctx, c) }
	if err := e.download(ctx, c.Intent.ManifestUrl, dest, strings.ToLower(c.Intent.ManifestSha256), hashCanonical, 0, canonjson.MaxBytes, poll); err != nil {
		return nil, api.ReleaseManifest{}, err
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		return nil, api.ReleaseManifest{}, err
	}
	m, err := VerifyManifest(b, c.Intent.ManifestSha256, e.ReleaseTrust, e.now())
	if err != nil {
		_ = os.Remove(dest)
		return nil, api.ReleaseManifest{}, err
	}
	if m.ReleaseId != c.Intent.ReleaseId {
		_ = os.Remove(dest)
		return nil, m, permanent(apierr.SignatureInvalid, "release_id")
	}
	return b, m, nil
}

func (e *Engine) verifyRelease(ctx context.Context, c claimed, m api.ReleaseManifest) (api.ManifestArtifact, error) {
	schema, _ := e.Journal.StateSchema(ctx)
	if schema == 0 {
		schema = e.Platform.StateSchema
	}
	p, _ := e.Journal.Pointers(ctx)
	if err := checkCompat(m, e.Platform, schema, p.CurrentRelease != ""); err != nil {
		return api.ManifestArtifact{}, err
	}
	art, err := selectArtifact(m, e.Platform.OSID, e.Platform.OSVersion, e.Platform.Architecture)
	if err != nil {
		return api.ManifestArtifact{}, err
	}
	if err := checkURL(art.Url, e.AllowLoopback); err != nil {
		return api.ManifestArtifact{}, err
	}
	need, ok := sumU64(art.SizeBytes.Uint64(), art.InstalledSizeBytes.Uint64(), m.Rollback.RequiredFreeBytes.Uint64())
	if !ok {
		return art, transient(apierr.DiskSpaceLow, "size overflow")
	}
	if err := e.ensureSpace(e.Layout.Releases, need); err != nil {
		return art, err
	}
	if err := e.ensureSpace(e.Layout.Staging, art.SizeBytes.Uint64()); err != nil {
		return art, err
	}
	return art, nil
}

func (e *Engine) fetchArtifact(ctx context.Context, c claimed, art api.ManifestArtifact) (string, error) {
	dest := filepath.Join(e.Layout.Staging, art.Sha256+".tar.zst")
	poll := func() error { return e.catchNewer(ctx, c) }
	if err := e.download(ctx, art.Url, dest, art.Sha256, hashRaw, int64(art.SizeBytes.Uint64()), int64(art.SizeBytes.Uint64()), poll); err != nil {
		return "", err
	}
	st, err := os.Stat(dest)
	if err != nil {
		return "", err
	}
	if err := e.Journal.PutArtifact(ctx, updatejournal.Artifact{
		SHA256: art.Sha256, ReleaseID: c.Intent.ReleaseId.String(), Path: dest, Size: st.Size(),
	}); err != nil {
		return "", err
	}
	return dest, nil
}

func (e *Engine) installTree(ctx context.Context, c claimed, m api.ReleaseManifest, env []byte, art api.ManifestArtifact, artifactPath string) error {
	id := c.Intent.ReleaseId.String()
	dir := e.Layout.releaseDir(id)
	signed, err := e.signedRevokedIDs(ctx)
	if err != nil {
		return err
	}
	if containsID(c.Revoked, c.Intent.ReleaseId) || containsID(signed, c.Intent.ReleaseId) {
		return permanent(apierr.ReleaseNotApproved, "revoked release")
	}
	if rel, err := e.Journal.GetRelease(ctx, id); err == nil {
		sum, herr := fsutil.TreeHash(dir)
		if herr == nil && sum == rel.TreeHash && strings.EqualFold(rel.ManifestSHA256, c.Intent.ManifestSha256) {
			return nil
		}
		return permanent(apierr.SignatureInvalid, "release is immutable")
	}
	p, err := e.Journal.Pointers(ctx)
	if err != nil {
		return err
	}
	if id == p.CurrentRelease || id == p.LastGood {
		return permanent(apierr.SignatureInvalid, "refusing to replace a retained release tree")
	}
	for _, persistent := range e.Layout.PersistentDirs() {
		if pathsOverlap(dir, persistent) {
			return permanent(apierr.ValidationFailed, "release directory overlaps persistent state")
		}
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	f, err := os.Open(artifactPath)
	if err != nil {
		return err
	}
	defer f.Close()
	max := int64(art.InstalledSizeBytes.Uint64()) * 2
	if max < int64(art.SizeBytes.Uint64())*2 {
		max = int64(art.SizeBytes.Uint64()) * 2
	}
	if max < 1 {
		max = 1 << 20
	}
	err = ExtractArchive(f, dir, ExtractOptions{
		MaxBytes: max, Deny: e.Layout.PersistentDirs(), DiskFull: e.Faults.DiskFullExtract,
	})
	if err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	if err := fsutil.SyncTree(dir); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	sum, err := fsutil.TreeHash(dir)
	if err != nil {
		return err
	}
	return e.Journal.PutRelease(ctx, updatejournal.Release{
		ReleaseID: id, ManifestSHA256: strings.ToLower(c.Intent.ManifestSha256),
		ManifestEnvelope: env, ArtifactSHA256: art.Sha256, TreeHash: sum,
		StateSchemaTo: m.Compatibility.StateSchemaTo,
	})
}

func (e *Engine) beginOrResume(ctx context.Context, c claimed, m api.ReleaseManifest) (updatejournal.Txn, error) {
	if txn, err := e.Journal.ActiveTxn(ctx); err == nil {
		return txn, nil
	} else if !errors.Is(err, updatejournal.ErrNotFound) {
		return updatejournal.Txn{}, err
	}
	p, err := e.Journal.Pointers(ctx)
	if err != nil {
		return updatejournal.Txn{}, err
	}
	schema, err := e.Journal.StateSchema(ctx)
	if err != nil {
		return updatejournal.Txn{}, err
	}
	mig := ""
	if id, err := m.Compatibility.MigrationId.Get(); err == nil {
		mig = id
	}
	id, err := uuid.NewV7()
	if err != nil {
		return updatejournal.Txn{}, err
	}
	from := schema
	if p.CurrentRelease == "" {
		from = 0
	}
	return e.Journal.BeginTxn(ctx, updatejournal.Txn{
		TxnID: id.String(), JobID: c.Intent.JobId.String(), Attempt: c.Intent.Attempt,
		Generation: c.Intent.Generation, ReleaseID: c.Intent.ReleaseId.String(),
		ManifestSHA256:    strings.ToLower(c.Intent.ManifestSha256),
		PreviousReleaseID: p.CurrentRelease, LastGoodReleaseID: p.LastGood,
		StateSchemaFrom: from, StateSchemaTo: m.Compatibility.StateSchemaTo, MigrationID: mig,
	})
}

func (e *Engine) afterPrepareErr(ctx context.Context, job updatejournal.Job, c claimed, err error, stage api.UpdateStage) (bool, error) {
	if errors.Is(err, ErrSuperseded) {
		return e.finishSupersede(ctx, job, c)
	}
	if errors.Is(err, ErrCrashed) || errors.Is(err, context.Canceled) {
		return false, err
	}
	op := asOp(err)
	if op != nil && !op.Retry {
		if ferr := e.failJob(ctx, job, c, op.Code, stage); ferr != nil {
			return false, ferr
		}
		return false, err
	}
	return false, err
}

// failRecorded stores a terminal failure and still returns that failure.
// failJob's nil means the outbox write succeeded, not that the update did.
func (e *Engine) failRecorded(ctx context.Context, job updatejournal.Job, c claimed, code apierr.Code, stage api.UpdateStage, cause error) (bool, error) {
	if ferr := e.failJob(ctx, job, c, code, stage); ferr != nil {
		return true, ferr
	}
	if cause != nil {
		return true, cause
	}
	return true, permanent(code, string(stage))
}

func (e *Engine) failJob(ctx context.Context, job updatejournal.Job, c claimed, code apierr.Code, stage api.UpdateStage) error {
	if c.Intent.JobId == uuid.Nil {
		decoded, err := decodeClaimed(job.IntentJSON)
		if err != nil {
			return err
		}
		c = decoded
	}
	p, _ := e.Journal.Pointers(ctx)
	schema, _ := e.Journal.StateSchema(ctx)
	started := job.ClaimedAt
	if started.IsZero() {
		started = c.Intent.IssuedAt
	}
	in := resultInput{
		Claimed: c, State: api.UpdateResultStateFailed, Stage: stage, Code: code,
		Current: p.CurrentRelease, Previous: p.CurrentRelease,
		Started: started, Finished: started, Schema: schemaAtLeast1(schemaOr(schema, e.Platform.StateSchema)),
	}
	if err := e.enqueueResult(ctx, in); err != nil {
		return err
	}
	return e.finish(ctx, job, string(api.UpdateResultStateFailed), string(code))
}

func (e *Engine) reportUnclaimed(ctx context.Context, c claimed, state api.UpdateResultState, stage api.UpdateStage, code apierr.Code) error {
	p, _ := e.Journal.Pointers(ctx)
	schema, _ := e.Journal.StateSchema(ctx)
	started := c.Intent.IssuedAt
	in := resultInput{
		Claimed: c, State: state, Stage: stage, Code: code,
		Current: p.CurrentRelease, Previous: p.CurrentRelease,
		Started: started, Finished: started, Schema: schemaAtLeast1(schemaOr(schema, e.Platform.StateSchema)),
	}
	if err := e.enqueueResult(ctx, in); err != nil {
		return err
	}
	return permanent(code, string(stage))
}

func (e *Engine) finishSupersede(ctx context.Context, job updatejournal.Job, c claimed) (bool, error) {
	if c.Intent.JobId == uuid.Nil {
		decoded, err := decodeClaimed(job.IntentJSON)
		if err != nil {
			return false, err
		}
		c = decoded
	}
	e.discardPartial()
	if _, err := e.Journal.GetRelease(ctx, job.ReleaseID); err != nil {
		p, _ := e.Journal.Pointers(ctx)
		if job.ReleaseID != p.CurrentRelease && job.ReleaseID != p.LastGood {
			_ = os.RemoveAll(e.Layout.releaseDir(job.ReleaseID))
		}
	}
	p, _ := e.Journal.Pointers(ctx)
	schema, _ := e.Journal.StateSchema(ctx)
	started := job.ClaimedAt
	if started.IsZero() {
		started = c.Intent.IssuedAt
	}
	in := resultInput{
		Claimed: c, State: api.UpdateResultStateSuperseded, Stage: api.UpdateStagePreflight,
		Code: apierr.StaleGeneration, Current: p.CurrentRelease, Previous: p.CurrentRelease,
		Started: started, Finished: started, Schema: schemaAtLeast1(schemaOr(schema, e.Platform.StateSchema)),
	}
	if err := e.enqueueResult(ctx, in); err != nil {
		return false, err
	}
	if err := e.finish(ctx, job, string(api.UpdateResultStateSuperseded), string(apierr.StaleGeneration)); err != nil {
		return false, err
	}
	return true, nil
}

func (e *Engine) discardPartial() {
	entries, err := os.ReadDir(e.Layout.Staging)
	if err != nil {
		return
	}
	for _, ent := range entries {
		name := ent.Name()
		if strings.HasSuffix(name, ".partial") || strings.HasSuffix(name, ".partial.meta") {
			_ = os.Remove(filepath.Join(e.Layout.Staging, name))
		}
	}
}

func (e *Engine) ensureReported(ctx context.Context) error {
	txn, err := e.Journal.LastTxn(ctx)
	if errors.Is(err, updatejournal.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !txn.Stage.Terminal() {
		return nil
	}
	job, err := e.Journal.GetJob(ctx, txn.JobID, txn.Attempt)
	if err != nil {
		return err
	}
	if job.Phase == updatejournal.JobDone {
		return nil
	}
	return e.seal(ctx, txn)
}

func (e *Engine) reapplyPublication(ctx context.Context) error {
	if e.Gate == nil {
		return nil
	}
	rec, err := e.Journal.GateRecord(ctx)
	if err != nil {
		_ = e.Gate.Block(ctx)
		return err
	}
	if !rec.Present {
		return nil
	}
	// failed is the blocking terminal state (there is no StageFailed).
	// recovery_required and an explicit blocked bit stay blocked. A stale
	// transaction is not consulted (D-034b).
	block := rec.ForwardingBlocked || rec.TerminalOutcome == updatejournal.OutcomeRecoveryRequired || rec.TerminalOutcome == updatejournal.OutcomeFailed
	if block || rec.PublicationIntent != string(api.PublicationIntentPublished) {
		if err := e.Gate.Block(ctx); err != nil {
			return wrapCode(apierr.UpdateBusy, err, true)
		}
		return nil
	}
	if err := e.Gate.Unblock(ctx); err != nil {
		return wrapCode(apierr.StartFailed, err, true)
	}
	return nil
}

func (e *Engine) enforce(ctx context.Context, releaseID string, c claimed, outcome string) (bool, error) {
	if e.Gate == nil {
		return false, errors.New("updater: no forwarding gate")
	}
	if outcome == updatejournal.OutcomeRecoveryRequired || outcome == updatejournal.OutcomeFailed {
		if err := e.noteBlocked(ctx, c, outcome); err != nil {
			return false, err
		}
		return false, nil
	}
	signed, err := e.signedRevokedIDs(ctx)
	if err != nil {
		return false, err
	}
	id, _ := uuid.Parse(releaseID)
	// A signed revocation is persisted in the journal. The claim's unsigned
	// list alone must not let that release pass the gate (§13.4).
	allow := c.Intent.PublicationIntent == api.PublicationIntentPublished &&
		releaseID != "" && containsID(c.Serving, id) && !containsID(c.Revoked, id) && !containsID(signed, id)
	if err := e.Journal.SetGate(ctx, updatejournal.GateRecord{
		PublicationIntent: string(c.Intent.PublicationIntent),
		ForwardingBlocked: !allow,
		TerminalOutcome:   outcome,
	}); err != nil {
		return false, err
	}
	if !allow {
		if err := e.Gate.Block(ctx); err != nil {
			return false, wrapCode(apierr.UpdateBusy, err, true)
		}
		return false, nil
	}
	if err := e.Gate.Unblock(ctx); err != nil {
		return false, wrapCode(apierr.StartFailed, err, true)
	}
	return true, nil
}

func (e *Engine) hookPhase(ctx context.Context, name string, c claimed) error {
	if e.Hooks.AfterPhase != nil {
		if err := e.Hooks.AfterPhase(name); err != nil {
			return err
		}
	}
	return e.catchNewer(ctx, c)
}

func (e *Engine) catchNewer(ctx context.Context, c claimed) error {
	if e.Source == nil {
		return nil
	}
	w, err := e.Source.Current(ctx)
	if err != nil {
		return nil
	}
	if w.Intent.Generation > c.Intent.Generation {
		return ErrSuperseded
	}
	if w.Intent.Generation == c.Intent.Generation &&
		(w.Intent.ReleaseId != c.Intent.ReleaseId || !strings.EqualFold(w.Intent.ManifestSha256, c.Intent.ManifestSha256)) {
		return permanent(apierr.StaleGeneration, "generation reused with different content")
	}
	return nil
}

func (e *Engine) hookStage(stage string) error {
	if e.Hooks.AfterStage == nil {
		return nil
	}
	return e.Hooks.AfterStage(stage)
}

func (e *Engine) setPhase(ctx context.Context, job updatejournal.Job, phase updatejournal.JobPhase) error {
	err := e.Journal.SetJobPhase(ctx, job.JobID, job.Attempt, phase)
	if errors.Is(err, updatejournal.ErrNotFound) {
		return nil
	}
	return err
}

func (e *Engine) finish(ctx context.Context, job updatejournal.Job, outcome, code string) error {
	err := e.Journal.FinishJob(ctx, job.JobID, job.Attempt, outcome, code)
	if errors.Is(err, updatejournal.ErrNotFound) {
		return nil
	}
	return err
}

func (e *Engine) claimFrom(work Work) claimed {
	return claimed{Work: work, BootID: e.Node.BootID, StateSchema: e.Platform.StateSchema}
}

func decodeClaimed(raw []byte) (claimed, error) {
	var c claimed
	err := json.Unmarshal(raw, &c)
	return c, err
}

func authorized(w Work, now time.Time) bool {
	until := w.Intent.AuthorizationValidUntil
	return !until.IsZero() && now.Before(until)
}

func (e *Engine) ensureSpace(path string, need uint64) error {
	free, err := e.freeBytes(path)
	if err != nil {
		return transient(apierr.StoreUnavailable, "free space")
	}
	if free < need {
		return transient(apierr.DiskSpaceLow, "reserve")
	}
	return nil
}

func (e *Engine) freeBytes(path string) (uint64, error) {
	if e.FreeBytes != nil {
		return e.FreeBytes(path)
	}
	return fsutil.FreeBytes(path)
}

func sumU64(parts ...uint64) (uint64, bool) {
	var s uint64
	for _, p := range parts {
		if s+p < s {
			return 0, false
		}
		s += p
	}
	return s, true
}

func schemaOr(a, b int) int {
	if a >= 1 {
		return a
	}
	return b
}

func (e *Engine) log() *slog.Logger {
	if e.Log != nil {
		return e.Log
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (e *Engine) jitter() float64 { return rand.Float64() }

func (e *Engine) httpClient() *http.Client {
	base := e.HTTP
	if base == nil {
		base = http.DefaultClient
	}
	clone := *base
	var tr *http.Transport
	switch t := base.Transport.(type) {
	case *http.Transport:
		tr = t.Clone()
	case nil:
		if def, ok := http.DefaultTransport.(*http.Transport); ok {
			tr = def.Clone()
		}
	}
	if tr != nil {
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialChecked(ctx, network, addr, e.AllowLoopback)
		}
		clone.Transport = tr
	}
	inner := base.CheckRedirect
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("updater: too many redirects")
		}
		if err := checkURL(req.URL.String(), e.AllowLoopback); err != nil {
			return err
		}
		if inner != nil {
			return inner(req, via)
		}
		return nil
	}
	return &clone
}

func safeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid-url"
	}
	u.RawQuery, u.User = "", nil
	return u.String()
}

func parseUUID(s string) (uuid.UUID, error) { return uuid.Parse(s) }
