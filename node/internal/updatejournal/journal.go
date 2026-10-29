// Package updatejournal is the node update journal of §13.5 and §13.6: a
// durable SQLite store (WAL, synchronous=FULL) under
// /var/lib/vpn-node/update-journal that records
//
//   - the highest accepted update generation and claimed jobs (§13.4, §13.8);
//   - the local apply transaction and its stage history (§13.6 steps 5-14);
//   - verified releases with their signed manifest and installed tree hash,
//     plus the current and last_good pointers (§13.2, §13.9);
//   - the durable update result outbox and the acknowledged history (§13.6
//     step 13, §13.8);
//   - the verified artifact cache reused across superseded jobs (§13.8).
//
// Every write is one SQLite transaction committed with synchronous=FULL, so a
// stage is on disk before the updater performs the next irreversible step.
// The journal never stores identity material, usage events or member data.
package updatejournal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (AGENTS.md §3)
)

// DefaultDir is the default journal directory (AGENTS.md §4).
const DefaultDir = "/var/lib/vpn-node/update-journal"

// FileName is the database file inside the journal directory.
const FileName = "journal.db"

// Stage is a step of the local apply transaction (§13.6).
type Stage string

// Transaction stages. The first seven follow §13.6 steps 6-11 in order;
// rolling_back is the journaled rollback of §13.9; the last three are
// terminal.
const (
	StagePrepared         Stage = "prepared"          // step 6: journal written, last_good and schema recorded
	StageBlocked          Stage = "blocked"           // step 7: outer member forwarding block active
	StageFinalSampled     Stage = "final_sampled"     // step 8: final usage sample and outbox durable
	StageStopped          Stage = "stopped"           // step 9: replaceable services stopped
	StageMigrated         Stage = "migrated"          // step 9: journaled state migration finished (or none needed)
	StageSwitched         Stage = "switched"          // step 10: current pointer switched and synced
	StageChecking         Stage = "checking"          // step 11: new services started, checks running
	StageRollingBack      Stage = "rolling_back"      // §13.9 steps 2-3 in progress
	StageLocallySucceeded Stage = "locally_succeeded" // step 12: all checks passed, last_good updated
	StageRolledBack       Stage = "rolled_back"       // §13.9 step 3: a verified earlier release serves again
	StageRecoveryRequired Stage = "recovery_required" // §13.9 step 4: members stay blocked
)

// AllStages lists every stage in transaction order.
var AllStages = []Stage{
	StagePrepared, StageBlocked, StageFinalSampled, StageStopped, StageMigrated,
	StageSwitched, StageChecking, StageRollingBack, StageLocallySucceeded,
	StageRolledBack, StageRecoveryRequired,
}

// Terminal reports whether s ends a transaction.
func (s Stage) Terminal() bool {
	switch s {
	case StageLocallySucceeded, StageRolledBack, StageRecoveryRequired:
		return true
	}
	return false
}

// Valid reports whether s is a defined stage.
func (s Stage) Valid() bool {
	for _, v := range AllStages {
		if v == s {
			return true
		}
	}
	return false
}

// Phase is the coarse update state of §13.2 that a stage belongs to.
type Phase string

// Phases of a transaction stage.
const (
	PhaseApplying Phase = "applying" // member forwarding must stay blocked
	PhaseChecking Phase = "checking" // member forwarding must stay blocked
	PhaseDone     Phase = "done"
)

// Phase maps a stage to applying, checking or done.
func (s Stage) Phase() Phase {
	switch s {
	case StageChecking:
		return PhaseChecking
	case StageLocallySucceeded, StageRolledBack, StageRecoveryRequired:
		return PhaseDone
	}
	return PhaseApplying
}

// transitions is the allowed stage graph. Every non-terminal stage may move
// to rolling_back or recovery_required; checking and rolling_back may repeat
// (a new check or rollback attempt after a crash).
var transitions = map[Stage][]Stage{
	StagePrepared:     {StageBlocked},
	StageBlocked:      {StageFinalSampled},
	StageFinalSampled: {StageStopped},
	StageStopped:      {StageMigrated},
	StageMigrated:     {StageSwitched},
	StageSwitched:     {StageChecking},
	StageChecking:     {StageChecking, StageLocallySucceeded},
	StageRollingBack:  {StageRollingBack, StageRolledBack},
}

// CanTransition reports whether a transaction may move from one stage to
// another.
func CanTransition(from, to Stage) bool {
	if from.Terminal() || !to.Valid() {
		return false
	}
	if to == StageRollingBack || to == StageRecoveryRequired {
		return true
	}
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// Errors.
var (
	ErrNotFound          = errors.New("updatejournal: not found")
	ErrActiveTransaction = errors.New("updatejournal: a transaction is still active")
	ErrBadTransition     = errors.New("updatejournal: stage transition not allowed")
	// ErrStaleGeneration: the job's generation is older than the accepted
	// one, or reuses the accepted generation with different content
	// (STALE_GENERATION, §13.4, §13.11).
	ErrStaleGeneration = errors.New("STALE_GENERATION")
	// ErrResultConflict: the same result_id with a different payload hash.
	ErrResultConflict = errors.New("ACK_CONFLICT")
)

// Journal is an open update journal. It is safe for concurrent use by one
// process; the node-wide flock serializes processes.
type Journal struct {
	db  *sql.DB
	dir string
	now func() time.Time
}

// Options configure Open.
type Options struct {
	// Now overrides the clock (tests).
	Now func() time.Time
}

// Open opens or creates the journal in dir with mode 0700.
func Open(dir string, opt Options) (*Journal, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, FileName)
	_, statErr := os.Stat(path)
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	j := &Journal{db: db, dir: dir, now: opt.Now}
	if j.now == nil {
		j.now = time.Now
	}
	if err := j.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if errors.Is(statErr, os.ErrNotExist) {
		_ = os.Chmod(path, 0o600)
		if err := syncDir(dir); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return j, nil
}

// Close closes the database.
func (j *Journal) Close() error { return j.db.Close() }

// Dir returns the journal directory.
func (j *Journal) Dir() string { return j.dir }

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

const schemaVersion = 1

func (j *Journal) migrate(ctx context.Context) error {
	var v int
	if err := j.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil {
		return fmt.Errorf("updatejournal: read schema version: %w", err)
	}
	if v > schemaVersion {
		// Written by a newer updater; never guess (fail closed, §13.7).
		return fmt.Errorf("updatejournal: journal schema %d is newer than supported %d", v, schemaVersion)
	}
	if v == schemaVersion {
		return nil
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS jobs (
			job_id TEXT NOT NULL,
			attempt INTEGER NOT NULL,
			generation TEXT NOT NULL,
			release_id TEXT NOT NULL,
			manifest_sha256 TEXT NOT NULL,
			intent_json TEXT NOT NULL,
			phase TEXT NOT NULL,
			outcome TEXT,
			error_code TEXT,
			claimed_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (job_id, attempt))`,
		`CREATE TABLE IF NOT EXISTS releases (
			release_id TEXT PRIMARY KEY,
			manifest_sha256 TEXT NOT NULL,
			manifest_envelope BLOB NOT NULL,
			artifact_sha256 TEXT NOT NULL,
			tree_hash TEXT NOT NULL,
			state_schema_to INTEGER NOT NULL,
			installed_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS txns (
			txn_id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			attempt INTEGER NOT NULL,
			generation TEXT NOT NULL,
			release_id TEXT NOT NULL,
			manifest_sha256 TEXT NOT NULL,
			previous_release_id TEXT NOT NULL,
			last_good_release_id TEXT NOT NULL,
			state_schema_from INTEGER NOT NULL,
			state_schema_to INTEGER NOT NULL,
			migration_id TEXT,
			stage TEXT NOT NULL,
			destructive INTEGER NOT NULL DEFAULT 0,
			check_attempts INTEGER NOT NULL DEFAULT 0,
			rollback_attempts INTEGER NOT NULL DEFAULT 0,
			error_code TEXT,
			error_stage TEXT,
			error_detail TEXT,
			checkpoint_json TEXT,
			health_json TEXT,
			rollback_release_id TEXT,
			started_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			finished_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS txn_events (
			txn_id TEXT NOT NULL REFERENCES txns(txn_id),
			seq INTEGER NOT NULL,
			stage TEXT NOT NULL,
			at TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (txn_id, seq))`,
		`CREATE TABLE IF NOT EXISTS result_outbox (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			result_id TEXT NOT NULL UNIQUE,
			job_id TEXT NOT NULL,
			attempt INTEGER NOT NULL,
			generation TEXT NOT NULL,
			state TEXT NOT NULL,
			payload BLOB NOT NULL,
			payload_hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			send_attempts INTEGER NOT NULL DEFAULT 0,
			last_attempt_at TEXT,
			last_error TEXT,
			created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS result_history (
			result_id TEXT PRIMARY KEY,
			job_id TEXT NOT NULL,
			attempt INTEGER NOT NULL,
			generation TEXT NOT NULL,
			state TEXT NOT NULL,
			payload_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			acked_at TEXT NOT NULL,
			accepted_as_history INTEGER NOT NULL,
			ack_json TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS artifacts (
			sha256 TEXT PRIMARY KEY,
			release_id TEXT NOT NULL,
			path TEXT NOT NULL,
			size INTEGER NOT NULL,
			verified_at TEXT NOT NULL)`,
		fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion),
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("updatejournal: migrate: %w", err)
		}
	}
	return tx.Commit()
}

func (j *Journal) ts() string { return j.now().UTC().Format(time.RFC3339Nano) }

func parseTS(s sql.NullString) time.Time {
	if !s.Valid {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, s.String)
	return t
}

// ---------------------------------------------------------------- meta

// Meta keys.
const (
	metaAcceptedGeneration = "accepted_generation"
	metaAcceptedJob        = "accepted_job"
	metaCurrentRelease     = "current_release_id"
	metaLastGoodRelease    = "last_good_release_id"
	metaDesiredState       = "last_desired_state"
	metaStateSchema        = "state_schema_version"
	metaPublicationIntent  = "publication_intent"
	metaForwardingBlocked  = "forwarding_blocked"
	metaTerminalOutcome    = "terminal_outcome"
	// Applied desired-state config_generation (§15.3, D-043). An older
	// generation never overwrites a newer one. This is not the update job
	// generation.
	metaAppliedConfigGeneration = "applied_config_generation"
	// Signed revoked_release_ids already accepted. The set only grows.
	metaRevokedReleaseIDs = "revoked_release_ids"
	// config_generation at which the signed set was last allowed to grow.
	metaRevocationGeneration = "revocation_config_generation"
)

// Terminal outcomes stored with the forwarding gate. There is no journal
// stage named failed; a failed terminal job is this outcome (D-034b).
const (
	OutcomeSucceeded        = "succeeded"
	OutcomeFailed           = "failed"
	OutcomeRolledBack       = "rolled_back"
	OutcomeRecoveryRequired = "recovery_required"
	OutcomeAlreadyCurrent   = "already_current"
)

func getMeta(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, key string) (string, bool, error) {
	var v string
	err := q.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return v, err == nil, err
}

func setMeta(ctx context.Context, q interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, key, value string) error {
	_, err := q.ExecContext(ctx, `INSERT INTO meta(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// Pointers are the release pointers the journal believes in. CurrentRelease
// is the release the current symlink is meant to point at; LastGood is the
// last release that passed all local checks (§13.2).
type Pointers struct {
	CurrentRelease string
	LastGood       string
}

// Pointers returns the recorded release pointers ("" when unset).
func (j *Journal) Pointers(ctx context.Context) (Pointers, error) {
	var p Pointers
	var err error
	if p.CurrentRelease, _, err = getMeta(ctx, j.db, metaCurrentRelease); err != nil {
		return p, err
	}
	if p.LastGood, _, err = getMeta(ctx, j.db, metaLastGoodRelease); err != nil {
		return p, err
	}
	return p, nil
}

// SetPointers records both release pointers in one transaction. Empty values
// leave the pointer unchanged.
func (j *Journal) SetPointers(ctx context.Context, p Pointers) error {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if p.CurrentRelease != "" {
		if err := setMeta(ctx, tx, metaCurrentRelease, p.CurrentRelease); err != nil {
			return err
		}
	}
	if p.LastGood != "" {
		if err := setMeta(ctx, tx, metaLastGoodRelease, p.LastGood); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GateRecord is the latest applied publication intent and the forwarding
// decision, written in one step. Recover replays this record. It does not
// infer the decision from an older transaction (D-034b).
type GateRecord struct {
	PublicationIntent string
	ForwardingBlocked bool
	TerminalOutcome   string
	Present           bool
}

// GateRecord returns the persisted forwarding decision. Present is false
// when none of the gate keys have been written.
func (j *Journal) GateRecord(ctx context.Context) (GateRecord, error) {
	var g GateRecord
	pub, okPub, err := getMeta(ctx, j.db, metaPublicationIntent)
	if err != nil {
		return g, err
	}
	blk, okBlk, err := getMeta(ctx, j.db, metaForwardingBlocked)
	if err != nil {
		return g, err
	}
	out, okOut, err := getMeta(ctx, j.db, metaTerminalOutcome)
	if err != nil {
		return g, err
	}
	g.PublicationIntent = pub
	g.ForwardingBlocked = blk == "1"
	g.TerminalOutcome = out
	g.Present = okPub || okBlk || okOut
	return g, nil
}

// SetGate writes the publication intent, the blocked bit and the terminal
// outcome in one transaction.
func (j *Journal) SetGate(ctx context.Context, g GateRecord) error {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := writeGate(ctx, tx, g); err != nil {
		return err
	}
	return tx.Commit()
}

func writeGate(ctx context.Context, q interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, g GateRecord) error {
	blocked := "0"
	if g.ForwardingBlocked {
		blocked = "1"
	}
	if err := setMeta(ctx, q, metaPublicationIntent, g.PublicationIntent); err != nil {
		return err
	}
	if err := setMeta(ctx, q, metaForwardingBlocked, blocked); err != nil {
		return err
	}
	return setMeta(ctx, q, metaTerminalOutcome, g.TerminalOutcome)
}

// SaveDesiredState stores the raw bytes of the latest verified desired state
// envelope. It is re-verified whenever it is used.
func (j *Journal) SaveDesiredState(ctx context.Context, raw []byte) error {
	return setMeta(ctx, j.db, metaDesiredState, string(raw))
}

// DesiredState returns the stored desired state envelope, or nil.
func (j *Journal) DesiredState(ctx context.Context) ([]byte, error) {
	v, ok, err := getMeta(ctx, j.db, metaDesiredState)
	if err != nil || !ok {
		return nil, err
	}
	return []byte(v), nil
}

// AcceptedGeneration returns the highest accepted update generation and the
// job it belongs to (ok=false when none was accepted yet).
func (j *Journal) AcceptedGeneration(ctx context.Context) (gen decstr.U64, jobKey string, ok bool, err error) {
	v, ok, err := getMeta(ctx, j.db, metaAcceptedGeneration)
	if err != nil || !ok {
		return 0, "", false, err
	}
	gen, err = decstr.Parse(v)
	if err != nil {
		return 0, "", false, fmt.Errorf("updatejournal: corrupt accepted generation: %w", err)
	}
	jobKey, _, err = getMeta(ctx, j.db, metaAcceptedJob)
	return gen, jobKey, true, err
}

// ---------------------------------------------------------------- jobs

// JobPhase is the local progress of a claimed job.
type JobPhase string

// Job phases (§13.2): claimed, downloading, verifying, verified, applying,
// checking, done.
const (
	JobClaimed     JobPhase = "claimed"
	JobDownloading JobPhase = "downloading"
	JobVerifying   JobPhase = "verifying"
	JobVerified    JobPhase = "verified"
	JobApplying    JobPhase = "applying"
	JobChecking    JobPhase = "checking"
	JobDone        JobPhase = "done"
)

// Job is a claimed update job (§13.6 step 1).
type Job struct {
	JobID          string
	Attempt        int
	Generation     decstr.U64
	ReleaseID      string
	ManifestSHA256 string
	IntentJSON     []byte
	Phase          JobPhase
	Outcome        string // result state once done
	ErrorCode      string
	ClaimedAt      time.Time
	UpdatedAt      time.Time
}

// ClaimJob persists a job claim and raises the accepted generation (§13.4).
// It refuses a generation older than the accepted one, and the accepted
// generation with a different job, release or manifest (ErrStaleGeneration).
// Re-claiming the same job and attempt is idempotent and returns the stored
// job (created=false).
func (j *Journal) ClaimJob(ctx context.Context, job Job) (stored Job, created bool, err error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, false, err
	}
	defer func() { _ = tx.Rollback() }()

	accepted, ok, err := getMeta(ctx, tx, metaAcceptedGeneration)
	if err != nil {
		return Job{}, false, err
	}
	key := job.JobID + "|" + job.ReleaseID + "|" + job.ManifestSHA256
	if ok {
		agen, err := decstr.Parse(accepted)
		if err != nil {
			return Job{}, false, fmt.Errorf("updatejournal: corrupt accepted generation: %w", err)
		}
		switch {
		case job.Generation < agen:
			return Job{}, false, fmt.Errorf("%w: generation %s is older than accepted %s", ErrStaleGeneration, job.Generation, agen)
		case job.Generation == agen:
			akey, _, err := getMeta(ctx, tx, metaAcceptedJob)
			if err != nil {
				return Job{}, false, err
			}
			if akey != key {
				return Job{}, false, fmt.Errorf("%w: generation %s already accepted for a different job", ErrStaleGeneration, agen)
			}
		}
	}
	existing, err := scanJob(tx.QueryRowContext(ctx, jobSelect+` WHERE job_id = ? AND attempt = ?`, job.JobID, job.Attempt))
	if err == nil {
		// An attempt is immutable (§13.8). A higher generation reuses the
		// job id only with a new attempt; the same attempt with different
		// content is a conflict, not a silent overwrite.
		if existing.Generation != job.Generation || existing.ReleaseID != job.ReleaseID || existing.ManifestSHA256 != job.ManifestSHA256 {
			return Job{}, false, fmt.Errorf("%w: job %s attempt %d already claimed with different content", ErrStaleGeneration, job.JobID, job.Attempt)
		}
		return existing, false, tx.Commit()
	}
	if !errors.Is(err, ErrNotFound) {
		return Job{}, false, err
	}
	if !ok || job.Generation > mustParse(accepted) {
		if err := setMeta(ctx, tx, metaAcceptedGeneration, job.Generation.String()); err != nil {
			return Job{}, false, err
		}
		if err := setMeta(ctx, tx, metaAcceptedJob, key); err != nil {
			return Job{}, false, err
		}
	}
	now := j.ts()
	if job.Phase == "" {
		job.Phase = JobClaimed
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO jobs(job_id, attempt, generation, release_id, manifest_sha256, intent_json, phase, claimed_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.JobID, job.Attempt, job.Generation.String(), job.ReleaseID, job.ManifestSHA256, string(job.IntentJSON), string(job.Phase), now, now); err != nil {
		return Job{}, false, err
	}
	stored, err = scanJob(tx.QueryRowContext(ctx, jobSelect+` WHERE job_id = ? AND attempt = ?`, job.JobID, job.Attempt))
	if err != nil {
		return Job{}, false, err
	}
	return stored, true, tx.Commit()
}

func mustParse(s string) decstr.U64 {
	v, err := decstr.Parse(s)
	if err != nil {
		return 0
	}
	return v
}

const jobSelect = `SELECT job_id, attempt, generation, release_id, manifest_sha256, intent_json, phase, outcome, error_code, claimed_at, updated_at FROM jobs`

type rowScanner interface{ Scan(...any) error }

func scanJob(r rowScanner) (Job, error) {
	var jb Job
	var gen, phase, intent string
	var outcome, code, claimed, updated sql.NullString
	err := r.Scan(&jb.JobID, &jb.Attempt, &gen, &jb.ReleaseID, &jb.ManifestSHA256, &intent, &phase, &outcome, &code, &claimed, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, err
	}
	if jb.Generation, err = decstr.Parse(gen); err != nil {
		return Job{}, fmt.Errorf("updatejournal: corrupt job generation: %w", err)
	}
	jb.IntentJSON = []byte(intent)
	jb.Phase = JobPhase(phase)
	jb.Outcome = outcome.String
	jb.ErrorCode = code.String
	jb.ClaimedAt = parseTS(claimed)
	jb.UpdatedAt = parseTS(updated)
	return jb, nil
}

// GetJob returns a claimed job.
func (j *Journal) GetJob(ctx context.Context, jobID string, attempt int) (Job, error) {
	return scanJob(j.db.QueryRowContext(ctx, jobSelect+` WHERE job_id = ? AND attempt = ?`, jobID, attempt))
}

// OpenJobs returns jobs that are not done, oldest claim first.
func (j *Journal) OpenJobs(ctx context.Context) ([]Job, error) {
	rows, err := j.db.QueryContext(ctx, jobSelect+` WHERE phase != ? ORDER BY claimed_at, job_id`, string(JobDone))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Job
	for rows.Next() {
		jb, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, jb)
	}
	return out, rows.Err()
}

// SetJobPhase moves a job to a phase. Done jobs never reopen.
func (j *Journal) SetJobPhase(ctx context.Context, jobID string, attempt int, phase JobPhase) error {
	res, err := j.db.ExecContext(ctx, `UPDATE jobs SET phase = ?, updated_at = ? WHERE job_id = ? AND attempt = ? AND phase != ?`,
		string(phase), j.ts(), jobID, attempt, string(JobDone))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// FinishJob marks a job done with its result state and error code.
func (j *Journal) FinishJob(ctx context.Context, jobID string, attempt int, outcome, errorCode string) error {
	res, err := j.db.ExecContext(ctx, `UPDATE jobs SET phase = ?, outcome = ?, error_code = NULLIF(?, ''), updated_at = ? WHERE job_id = ? AND attempt = ? AND phase != ?`,
		string(JobDone), outcome, errorCode, j.ts(), jobID, attempt, string(JobDone))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------- releases

// Release is a verified, installed release (§13.3, §13.5).
type Release struct {
	ReleaseID        string
	ManifestSHA256   string
	ManifestEnvelope []byte // raw signed manifest envelope, re-verified before any rollback
	ArtifactSHA256   string
	TreeHash         string // digest of the extracted tree, rechecked before use
	StateSchemaTo    int
	InstalledAt      time.Time
}

// PutRelease records an installed release. A release_id is immutable: a
// second record with a different manifest or tree hash is refused (§13.3).
func (j *Journal) PutRelease(ctx context.Context, r Release) error {
	old, err := j.GetRelease(ctx, r.ReleaseID)
	if err == nil {
		if old.ManifestSHA256 != r.ManifestSHA256 || old.TreeHash != r.TreeHash || old.ArtifactSHA256 != r.ArtifactSHA256 {
			return fmt.Errorf("updatejournal: release %s is immutable and already recorded with different content", r.ReleaseID)
		}
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	_, err = j.db.ExecContext(ctx, `INSERT INTO releases(release_id, manifest_sha256, manifest_envelope, artifact_sha256, tree_hash, state_schema_to, installed_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`, r.ReleaseID, r.ManifestSHA256, r.ManifestEnvelope, r.ArtifactSHA256, r.TreeHash, r.StateSchemaTo, j.ts())
	return err
}

// GetRelease returns an installed release record.
func (j *Journal) GetRelease(ctx context.Context, id string) (Release, error) {
	var r Release
	var at sql.NullString
	err := j.db.QueryRowContext(ctx, `SELECT release_id, manifest_sha256, manifest_envelope, artifact_sha256, tree_hash, state_schema_to, installed_at FROM releases WHERE release_id = ?`, id).
		Scan(&r.ReleaseID, &r.ManifestSHA256, &r.ManifestEnvelope, &r.ArtifactSHA256, &r.TreeHash, &r.StateSchemaTo, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return Release{}, ErrNotFound
	}
	r.InstalledAt = parseTS(at)
	return r, err
}

// ListReleases returns all recorded release IDs.
func (j *Journal) ListReleases(ctx context.Context) ([]string, error) {
	rows, err := j.db.QueryContext(ctx, `SELECT release_id FROM releases ORDER BY installed_at, release_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// DeleteRelease forgets a release (after its directory was removed by
// retention). current and last_good are refused.
func (j *Journal) DeleteRelease(ctx context.Context, id string) error {
	p, err := j.Pointers(ctx)
	if err != nil {
		return err
	}
	if id == p.CurrentRelease || id == p.LastGood {
		return fmt.Errorf("updatejournal: release %s is current or last_good and must be retained", id)
	}
	_, err = j.db.ExecContext(ctx, `DELETE FROM releases WHERE release_id = ?`, id)
	return err
}

// ---------------------------------------------------------------- artifacts

// Artifact is a downloaded artifact whose size and SHA-256 were verified.
type Artifact struct {
	SHA256     string
	ReleaseID  string
	Path       string
	Size       int64
	VerifiedAt time.Time
}

// PutArtifact records a verified artifact for reuse (§13.8).
func (j *Journal) PutArtifact(ctx context.Context, a Artifact) error {
	_, err := j.db.ExecContext(ctx, `INSERT INTO artifacts(sha256, release_id, path, size, verified_at) VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(sha256) DO UPDATE SET release_id = excluded.release_id, path = excluded.path, size = excluded.size, verified_at = excluded.verified_at`,
		a.SHA256, a.ReleaseID, a.Path, a.Size, j.ts())
	return err
}

// GetArtifact returns a verified artifact by SHA-256.
func (j *Journal) GetArtifact(ctx context.Context, sha string) (Artifact, error) {
	var a Artifact
	var at sql.NullString
	err := j.db.QueryRowContext(ctx, `SELECT sha256, release_id, path, size, verified_at FROM artifacts WHERE sha256 = ?`, sha).
		Scan(&a.SHA256, &a.ReleaseID, &a.Path, &a.Size, &at)
	if errors.Is(err, sql.ErrNoRows) {
		return Artifact{}, ErrNotFound
	}
	a.VerifiedAt = parseTS(at)
	return a, err
}

// ListArtifacts returns every cached artifact.
func (j *Journal) ListArtifacts(ctx context.Context) ([]Artifact, error) {
	rows, err := j.db.QueryContext(ctx, `SELECT sha256, release_id, path, size, verified_at FROM artifacts ORDER BY verified_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Artifact
	for rows.Next() {
		var a Artifact
		var at sql.NullString
		if err := rows.Scan(&a.SHA256, &a.ReleaseID, &a.Path, &a.Size, &at); err != nil {
			return nil, err
		}
		a.VerifiedAt = parseTS(at)
		out = append(out, a)
	}
	return out, rows.Err()
}

// DeleteArtifact forgets a cached artifact.
func (j *Journal) DeleteArtifact(ctx context.Context, sha string) error {
	_, err := j.db.ExecContext(ctx, `DELETE FROM artifacts WHERE sha256 = ?`, sha)
	return err
}

// ---------------------------------------------------------------- state schema

// StateSchema returns the local state schema version recorded by the last
// successful transaction (0 when unknown).
func (j *Journal) StateSchema(ctx context.Context) (int, error) {
	v, ok, err := getMeta(ctx, j.db, metaStateSchema)
	if err != nil || !ok {
		return 0, err
	}
	var n int
	_, err = fmt.Sscanf(v, "%d", &n)
	return n, err
}

// SetStateSchema records the local state schema version.
func (j *Journal) SetStateSchema(ctx context.Context, v int) error {
	return setMeta(ctx, j.db, metaStateSchema, fmt.Sprint(v))
}

// AppliedConfig returns the desired-state config_generation the node has
// applied (ok is false when none has been applied).
func (j *Journal) AppliedConfig(ctx context.Context) (decstr.U64, bool, error) {
	raw, ok, err := getMeta(ctx, j.db, metaAppliedConfigGeneration)
	if err != nil || !ok {
		return 0, false, err
	}
	gen, err := decstr.Parse(raw)
	if err != nil {
		return 0, false, fmt.Errorf("updatejournal: corrupt applied config generation: %w", err)
	}
	return gen, true, nil
}

// AcceptDesired records a verified desired-state config_generation and, when
// includeRevocations is true, unions signed release ids into the persisted
// revocation set. An older generation does not overwrite the applied
// generation and does not change the set (§13.4, §15.3, §17.2). accepted is
// false when gen is older than the applied generation. An empty id list
// still counts as a present signed member: it adds nothing and does not
// clear the set. New meta keys do not change PRAGMA user_version.
func (j *Journal) AcceptDesired(ctx context.Context, gen decstr.U64, includeRevocations bool, ids []string) (applied decstr.U64, accepted bool, err error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer func() { _ = tx.Rollback() }()

	raw, ok, err := getMeta(ctx, tx, metaAppliedConfigGeneration)
	if err != nil {
		return 0, false, err
	}
	var stored decstr.U64
	if ok {
		stored, err = decstr.Parse(raw)
		if err != nil {
			return 0, false, fmt.Errorf("updatejournal: corrupt applied config generation: %w", err)
		}
		if gen < stored {
			return stored, false, nil
		}
	}
	if !ok || gen > stored {
		if err := setMeta(ctx, tx, metaAppliedConfigGeneration, gen.String()); err != nil {
			return 0, false, err
		}
	}
	if includeRevocations {
		if err := unionRevoked(ctx, tx, gen, ids); err != nil {
			return 0, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return gen, true, nil
}

// RevokedReleases returns the persisted signed revocation set. The slice is
// empty when no signed member has been accepted.
func (j *Journal) RevokedReleases(ctx context.Context) ([]string, error) {
	raw, ok, err := getMeta(ctx, j.db, metaRevokedReleaseIDs)
	if err != nil || !ok || raw == "" {
		return nil, err
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("updatejournal: corrupt revoked releases: %w", err)
	}
	return ids, nil
}

// unionRevoked adds ids to the signed set. A generation older than the one
// recorded for the set does not add or remove ids (§17.2, §17.4).
func unionRevoked(ctx context.Context, tx *sql.Tx, gen decstr.U64, ids []string) error {
	rawGen, ok, err := getMeta(ctx, tx, metaRevocationGeneration)
	if err != nil {
		return err
	}
	if ok {
		cur, err := decstr.Parse(rawGen)
		if err != nil {
			return fmt.Errorf("updatejournal: corrupt revocation generation: %w", err)
		}
		if gen < cur {
			return nil
		}
	}
	have := map[string]struct{}{}
	raw, ok, err := getMeta(ctx, tx, metaRevokedReleaseIDs)
	if err != nil {
		return err
	}
	if ok && raw != "" {
		var prev []string
		if err := json.Unmarshal([]byte(raw), &prev); err != nil {
			return fmt.Errorf("updatejournal: corrupt revoked releases: %w", err)
		}
		for _, id := range prev {
			if id != "" {
				have[id] = struct{}{}
			}
		}
	}
	for _, id := range ids {
		if id != "" {
			have[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(have))
	for id := range have {
		out = append(out, id)
	}
	sort.Strings(out)
	body, err := json.Marshal(out)
	if err != nil {
		return err
	}
	if err := setMeta(ctx, tx, metaRevokedReleaseIDs, string(body)); err != nil {
		return err
	}
	return setMeta(ctx, tx, metaRevocationGeneration, gen.String())
}

// ---------------------------------------------------------------- helpers

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func marshalOrNull(v any) (sql.NullString, error) {
	if v == nil {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}
