package updatejournal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// Txn is one local apply transaction (§13.6 steps 5-14).
type Txn struct {
	TxnID             string
	JobID             string
	Attempt           int
	Generation        decstr.U64
	ReleaseID         string // candidate release
	ManifestSHA256    string
	PreviousReleaseID string // release current pointed at when the transaction began
	LastGoodReleaseID string // last_good recorded at prepare time (§13.6 step 6)
	StateSchemaFrom   int
	StateSchemaTo     int
	MigrationID       string
	Stage             Stage
	// Destructive is set once services were stopped or the pointer moved:
	// from then on a failure is a rollback, not just an aborted attempt.
	Destructive       bool
	CheckAttempts     int
	RollbackAttempts  int
	ErrorCode         string
	ErrorStage        string
	ErrorDetail       string
	Checkpoint        json.RawMessage // usage checkpoint from the final sample (§13.6 step 8)
	Health            json.RawMessage // latest observed profile; restored health after rollback (D-062)
	CandidateHealth   json.RawMessage // latest candidate profile, retained through rollback for diagnosis
	RollbackReleaseID string          // release restored by a rollback
	StartedAt         time.Time
	UpdatedAt         time.Time
	FinishedAt        time.Time
}

// TxnEvent is one journal row for a transaction. A stage change appends one.
// RecordEvent appends a detail row at the current stage without moving it,
// which is how health reasons stay next to a later rollback (D-053).
type TxnEvent struct {
	Seq    int
	Stage  Stage
	At     time.Time
	Detail string
}

// BeginTxn writes a new transaction in stage prepared (§13.6 step 6). Only
// one transaction may be active at a time.
func (j *Journal) BeginTxn(ctx context.Context, t Txn) (Txn, error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return Txn{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM txns WHERE stage NOT IN (?, ?, ?)`,
		string(StageLocallySucceeded), string(StageRolledBack), string(StageRecoveryRequired)).Scan(&n); err != nil {
		return Txn{}, err
	}
	if n > 0 {
		return Txn{}, ErrActiveTransaction
	}
	now := j.ts()
	t.Stage = StagePrepared
	if _, err := tx.ExecContext(ctx, `INSERT INTO txns(txn_id, job_id, attempt, generation, release_id, manifest_sha256, previous_release_id, last_good_release_id,
		state_schema_from, state_schema_to, migration_id, stage, started_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.TxnID, t.JobID, t.Attempt, t.Generation.String(), t.ReleaseID, t.ManifestSHA256, t.PreviousReleaseID, t.LastGoodReleaseID,
		t.StateSchemaFrom, t.StateSchemaTo, nullStr(t.MigrationID), string(StagePrepared), now, now); err != nil {
		return Txn{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO txn_events(txn_id, seq, stage, at, detail) VALUES(?, 1, ?, ?, '')`, t.TxnID, string(StagePrepared), now); err != nil {
		return Txn{}, err
	}
	if err := tx.Commit(); err != nil {
		return Txn{}, err
	}
	return j.GetTxn(ctx, t.TxnID)
}

// StageUpdate carries optional fields written together with a stage change.
type StageUpdate struct {
	Detail            string
	Destructive       bool // set (never cleared) the destructive flag
	ErrorCode         string
	ErrorStage        string
	ErrorDetail       string
	Checkpoint        any // JSON-encoded when non-nil
	Health            any // JSON-encoded when non-nil
	RollbackReleaseID string
	Pointers          Pointers // nonempty fields change atomically with the stage
	StateSchema       int      // positive values change atomically with the stage
}

// Advance moves a transaction to the next stage in one durable write. The
// transition must be allowed by CanTransition. Repeating checking or
// rolling_back increments the matching attempt counter, which bounds boot
// loops after crashes during checks.
func (j *Journal) Advance(ctx context.Context, txnID string, to Stage, u StageUpdate) (Txn, error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return Txn{}, err
	}
	defer func() { _ = tx.Rollback() }()
	cur, err := scanTxn(tx.QueryRowContext(ctx, txnSelect+` WHERE txn_id = ?`, txnID))
	if err != nil {
		return Txn{}, err
	}
	if !CanTransition(cur.Stage, to) {
		return Txn{}, fmt.Errorf("%w: %s -> %s", ErrBadTransition, cur.Stage, to)
	}
	cp, err := marshalOrNull(u.Checkpoint)
	if err != nil {
		return Txn{}, err
	}
	hl, err := marshalOrNull(u.Health)
	if err != nil {
		return Txn{}, err
	}
	now := j.ts()
	var finished sql.NullString
	if to.Terminal() {
		finished = sql.NullString{String: now, Valid: true}
	}
	checkInc, rbInc := 0, 0
	if to == StageChecking {
		checkInc = 1
	}
	if to == StageRollingBack {
		rbInc = 1
	}
	destructive := 0
	if u.Destructive {
		destructive = 1
	}
	if _, err := tx.ExecContext(ctx, `UPDATE txns SET stage = ?, updated_at = ?, finished_at = COALESCE(?, finished_at),
		destructive = MAX(destructive, ?),
		check_attempts = check_attempts + ?, rollback_attempts = rollback_attempts + ?,
		error_code = COALESCE(?, error_code), error_stage = COALESCE(?, error_stage), error_detail = COALESCE(?, error_detail),
		checkpoint_json = COALESCE(?, checkpoint_json),
		health_json = COALESCE(?, health_json),
		candidate_health_json = CASE WHEN stage = 'checking' THEN COALESCE(?, candidate_health_json) ELSE candidate_health_json END,
		rollback_release_id = COALESCE(?, rollback_release_id)
		WHERE txn_id = ?`,
		string(to), now, finished, destructive, checkInc, rbInc,
		nullStr(u.ErrorCode), nullStr(u.ErrorStage), nullStr(u.ErrorDetail), cp, hl, hl, nullStr(u.RollbackReleaseID), txnID); err != nil {
		return Txn{}, err
	}
	var seq int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) + 1 FROM txn_events WHERE txn_id = ?`, txnID).Scan(&seq); err != nil {
		return Txn{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO txn_events(txn_id, seq, stage, at, detail) VALUES(?, ?, ?, ?, ?)`, txnID, seq, string(to), now, u.Detail); err != nil {
		return Txn{}, err
	}
	if err := persistAdvanceGate(ctx, tx, to); err != nil {
		return Txn{}, err
	}
	if err := setPointers(ctx, tx, u.Pointers); err != nil {
		return Txn{}, err
	}
	if u.StateSchema > 0 {
		if err := setMeta(ctx, tx, metaStateSchema, fmt.Sprint(u.StateSchema)); err != nil {
			return Txn{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Txn{}, err
	}
	return j.GetTxn(ctx, txnID)
}

// persistAdvanceGate records the forwarding bit in the same transaction as
// the stage change. Publication intent is left for the engine, which knows
// the desired state. A non-terminal stage only raises the blocked bit.
func persistAdvanceGate(ctx context.Context, tx *sql.Tx, to Stage) error {
	switch to {
	case StageRecoveryRequired:
		if err := setMeta(ctx, tx, metaForwardingBlocked, "1"); err != nil {
			return err
		}
		return setMeta(ctx, tx, metaTerminalOutcome, OutcomeRecoveryRequired)
	case StageLocallySucceeded:
		if err := setMeta(ctx, tx, metaForwardingBlocked, "0"); err != nil {
			return err
		}
		return setMeta(ctx, tx, metaTerminalOutcome, OutcomeSucceeded)
	case StageRolledBack:
		if err := setMeta(ctx, tx, metaForwardingBlocked, "0"); err != nil {
			return err
		}
		return setMeta(ctx, tx, metaTerminalOutcome, OutcomeRolledBack)
	default:
		if !to.Terminal() {
			return setMeta(ctx, tx, metaForwardingBlocked, "1")
		}
	}
	return nil
}

// RecordHealth stores the latest observed profile. The latest candidate
// profile stays separately in candidate_health_json through rollback (D-062).
func (j *Journal) RecordHealth(ctx context.Context, txnID string, health any) error {
	hl, err := marshalOrNull(health)
	if err != nil {
		return err
	}
	if !hl.Valid {
		return errors.New("updatejournal: empty health")
	}
	res, err := j.db.ExecContext(ctx, `UPDATE txns SET health_json = ?,
		candidate_health_json = CASE WHEN stage = 'checking' THEN ? ELSE candidate_health_json END,
		updated_at = ? WHERE txn_id = ?`, hl.String, hl.String, j.ts(), txnID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err = j.GetTxn(ctx, txnID)
	return err
}

// RecordEvent appends detail without changing the stage or health profiles.
func (j *Journal) RecordEvent(ctx context.Context, txnID, detail string) error {
	if detail == "" {
		return errors.New("updatejournal: empty event detail")
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	cur, err := scanTxn(tx.QueryRowContext(ctx, txnSelect+` WHERE txn_id = ?`, txnID))
	if err != nil {
		return err
	}
	var seq int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) + 1 FROM txn_events WHERE txn_id = ?`, txnID).Scan(&seq); err != nil {
		return err
	}
	now := j.ts()
	if _, err := tx.ExecContext(ctx, `INSERT INTO txn_events(txn_id, seq, stage, at, detail) VALUES(?, ?, ?, ?, ?)`, txnID, seq, string(cur.Stage), now, detail); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE txns SET updated_at = ? WHERE txn_id = ?`, now, txnID); err != nil {
		return err
	}
	return tx.Commit()
}

const txnSelect = `SELECT txn_id, job_id, attempt, generation, release_id, manifest_sha256, previous_release_id, last_good_release_id,
	state_schema_from, state_schema_to, migration_id, stage, destructive, check_attempts, rollback_attempts,
	error_code, error_stage, error_detail, checkpoint_json, health_json, candidate_health_json, rollback_release_id, started_at, updated_at, finished_at FROM txns`

func scanTxn(r rowScanner) (Txn, error) {
	var t Txn
	var gen, stage string
	var mig, ecode, estage, edetail, cp, hl, candidate, rb, started, updated, finished sql.NullString
	var destructive int
	err := r.Scan(&t.TxnID, &t.JobID, &t.Attempt, &gen, &t.ReleaseID, &t.ManifestSHA256, &t.PreviousReleaseID, &t.LastGoodReleaseID,
		&t.StateSchemaFrom, &t.StateSchemaTo, &mig, &stage, &destructive, &t.CheckAttempts, &t.RollbackAttempts,
		&ecode, &estage, &edetail, &cp, &hl, &candidate, &rb, &started, &updated, &finished)
	if errors.Is(err, sql.ErrNoRows) {
		return Txn{}, ErrNotFound
	}
	if err != nil {
		return Txn{}, err
	}
	if t.Generation, err = decstr.Parse(gen); err != nil {
		return Txn{}, fmt.Errorf("updatejournal: corrupt txn generation: %w", err)
	}
	t.Stage = Stage(stage)
	if !t.Stage.Valid() {
		return Txn{}, fmt.Errorf("updatejournal: corrupt txn stage %q", stage)
	}
	t.MigrationID = mig.String
	t.Destructive = destructive != 0
	t.ErrorCode, t.ErrorStage, t.ErrorDetail = ecode.String, estage.String, edetail.String
	if cp.Valid {
		t.Checkpoint = json.RawMessage(cp.String)
	}
	if hl.Valid {
		t.Health = json.RawMessage(hl.String)
	}
	if candidate.Valid {
		t.CandidateHealth = json.RawMessage(candidate.String)
	}
	t.RollbackReleaseID = rb.String
	t.StartedAt, t.UpdatedAt, t.FinishedAt = parseTS(started), parseTS(updated), parseTS(finished)
	return t, nil
}

// GetTxn returns a transaction by ID.
func (j *Journal) GetTxn(ctx context.Context, id string) (Txn, error) {
	return scanTxn(j.db.QueryRowContext(ctx, txnSelect+` WHERE txn_id = ?`, id))
}

// ActiveTxn returns the non-terminal transaction, or ErrNotFound.
func (j *Journal) ActiveTxn(ctx context.Context) (Txn, error) {
	return scanTxn(j.db.QueryRowContext(ctx, txnSelect+` WHERE stage NOT IN (?, ?, ?) ORDER BY started_at DESC LIMIT 1`,
		string(StageLocallySucceeded), string(StageRolledBack), string(StageRecoveryRequired)))
}

// LastTxn returns the most recently started transaction, or ErrNotFound.
func (j *Journal) LastTxn(ctx context.Context) (Txn, error) {
	return scanTxn(j.db.QueryRowContext(ctx, txnSelect+` ORDER BY started_at DESC, rowid DESC LIMIT 1`))
}

// TxnEvents returns the stage history of a transaction.
func (j *Journal) TxnEvents(ctx context.Context, id string) ([]TxnEvent, error) {
	rows, err := j.db.QueryContext(ctx, `SELECT seq, stage, at, detail FROM txn_events WHERE txn_id = ? ORDER BY seq`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TxnEvent
	for rows.Next() {
		var e TxnEvent
		var stage string
		var at sql.NullString
		if err := rows.Scan(&e.Seq, &stage, &at, &e.Detail); err != nil {
			return nil, err
		}
		e.Stage, e.At = Stage(stage), parseTS(at)
		out = append(out, e)
	}
	return out, rows.Err()
}

// BlocksForwarding reports whether the journal requires member forwarding to
// stay off. An active transaction blocks. Otherwise the decision is the
// persisted gate record, not an older transaction (D-034b). recovery_required
// and a failed terminal outcome stay blocked. rolled_back does not, by
// itself, keep members off. Any read error is blocking.
func (j *Journal) BlocksForwarding(ctx context.Context) (bool, string) {
	t, err := j.ActiveTxn(ctx)
	if err == nil {
		return true, "update transaction " + string(t.Stage.Phase())
	}
	if !errors.Is(err, ErrNotFound) {
		return true, "update journal unreadable"
	}
	g, err := j.GateRecord(ctx)
	if err != nil {
		return true, "update journal unreadable"
	}
	if g.TerminalOutcome == OutcomeRecoveryRequired {
		return true, "update recovery_required"
	}
	if g.ForwardingBlocked || g.TerminalOutcome == OutcomeFailed {
		return true, "update forwarding blocked"
	}
	return false, ""
}

// ---------------------------------------------------------------- result outbox

// OutboxStatus is the delivery status of an outbox entry.
type OutboxStatus string

// Outbox statuses.
const (
	OutboxPending  OutboxStatus = "pending"
	OutboxRejected OutboxStatus = "rejected" // permanent HTTP rejection; evidence retained, later results proceed
	OutboxConflict OutboxStatus = "conflict" // central answered ACK_CONFLICT; evidence kept, never resent
)

// OutboxEntry is a durable, not yet acknowledged update result (§13.6 step
// 13). Payload is the exact canonical JSON sent on every retry after the
// one-time PopulateCandidateHealth compatibility update, when needed.
type OutboxEntry struct {
	Seq            int64
	ResultID       string
	JobID          string
	Attempt        int
	Generation     decstr.U64
	State          string
	Payload        []byte
	PayloadHash    string
	Status         OutboxStatus
	SendAttempts   int
	LastAttemptAt  time.Time
	LastError      string
	LastHTTPStatus int
	LastAPICode    string
	RetryNotBefore time.Time
	CreatedAt      time.Time
}

// Enqueue stores a result. Re-enqueueing the same result_id with the same
// payload hash is a no-op; with a different hash it is ErrResultConflict.
func (j *Journal) Enqueue(ctx context.Context, e OutboxEntry) error {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var hash string
	err = tx.QueryRowContext(ctx, `SELECT payload_hash FROM result_outbox WHERE result_id = ? UNION ALL SELECT payload_hash FROM result_history WHERE result_id = ?`, e.ResultID, e.ResultID).Scan(&hash)
	switch {
	case err == nil:
		if hash != e.PayloadHash {
			return fmt.Errorf("%w: result %s already stored with a different payload hash", ErrResultConflict, e.ResultID)
		}
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO result_outbox(result_id, job_id, attempt, generation, state, payload, payload_hash, status, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, e.ResultID, e.JobID, e.Attempt, e.Generation.String(), e.State, e.Payload, e.PayloadHash, string(OutboxPending), j.ts()); err != nil {
		return err
	}
	return tx.Commit()
}

const outboxSelect = `SELECT seq, result_id, job_id, attempt, generation, state, payload, payload_hash, status, send_attempts, last_attempt_at, last_error, last_http_status, last_api_code, retry_not_before, created_at FROM result_outbox`

func scanOutbox(r rowScanner) (OutboxEntry, error) {
	var e OutboxEntry
	var gen, status string
	var last, lerr, apiCode, retryAt, created sql.NullString
	err := r.Scan(&e.Seq, &e.ResultID, &e.JobID, &e.Attempt, &gen, &e.State, &e.Payload, &e.PayloadHash, &status, &e.SendAttempts, &last, &lerr, &e.LastHTTPStatus, &apiCode, &retryAt, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return OutboxEntry{}, ErrNotFound
	}
	if err != nil {
		return OutboxEntry{}, err
	}
	if e.Generation, err = decstr.Parse(gen); err != nil {
		return OutboxEntry{}, err
	}
	e.Status = OutboxStatus(status)
	e.LastAttemptAt, e.LastError, e.CreatedAt = parseTS(last), lerr.String, parseTS(created)
	e.LastAPICode, e.RetryNotBefore = apiCode.String, parseTS(retryAt)
	return e, nil
}

// Pending returns up to limit pending results in insertion order.
func (j *Journal) Pending(ctx context.Context, limit int) ([]OutboxEntry, error) {
	rows, err := j.db.QueryContext(ctx, outboxSelect+` WHERE status = ? ORDER BY seq LIMIT ?`, string(OutboxPending), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutboxEntry
	for rows.Next() {
		e, err := scanOutbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// HasResult reports whether a result was already sealed, regardless of whether
// it is pending, rejected, conflicted or acknowledged. Recovery must reuse that
// immutable evidence rather than rebuild it under a newer reporting policy.
func (j *Journal) HasResult(ctx context.Context, resultID string) (bool, error) {
	var found bool
	err := j.db.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM result_outbox WHERE result_id = ?
		UNION ALL SELECT 1 FROM result_history WHERE result_id = ?)`, resultID, resultID).Scan(&found)
	return found, err
}

// Outbox returns every entry still in the outbox (pending, rejected or conflict).
func (j *Journal) Outbox(ctx context.Context) ([]OutboxEntry, error) {
	rows, err := j.db.QueryContext(ctx, outboxSelect+` ORDER BY seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OutboxEntry
	for rows.Next() {
		e, err := scanOutbox(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// RecordSendFailure notes a failed delivery attempt; the entry stays pending.
func (j *Journal) RecordSendFailure(ctx context.Context, resultID, reason string) error {
	_, err := j.db.ExecContext(ctx, `UPDATE result_outbox SET send_attempts = send_attempts + 1, last_attempt_at = ?, last_error = ? WHERE result_id = ?`,
		j.ts(), reason, resultID)
	return err
}

// Ack is central acceptance of a result (UpdateResultAck, §13.10.3).
type Ack struct {
	ResultID          string
	PayloadHash       string // hash of the payload the ACK answers
	AcceptedAsHistory bool
	Raw               json.RawMessage
}

// AckResult moves an acknowledged result from the outbox to the history.
// A duplicate or late ACK for a result already in the history is a no-op
// (duplicate=true). An ACK whose payload hash does not match the stored one
// is ErrResultConflict and changes nothing.
func (j *Journal) AckResult(ctx context.Context, a Ack) (duplicate bool, err error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	e, err := scanOutbox(tx.QueryRowContext(ctx, outboxSelect+` WHERE result_id = ?`, a.ResultID))
	if errors.Is(err, ErrNotFound) {
		var hash string
		herr := tx.QueryRowContext(ctx, `SELECT payload_hash FROM result_history WHERE result_id = ?`, a.ResultID).Scan(&hash)
		if errors.Is(herr, sql.ErrNoRows) {
			return false, fmt.Errorf("updatejournal: ACK for unknown result %s: %w", a.ResultID, ErrNotFound)
		}
		if herr != nil {
			return false, herr
		}
		if a.PayloadHash != "" && hash != a.PayloadHash {
			return false, ErrResultConflict
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if a.PayloadHash != "" && a.PayloadHash != e.PayloadHash {
		return false, ErrResultConflict
	}
	raw := string(a.Raw)
	if raw == "" {
		raw = "{}"
	}
	hist := 0
	if a.AcceptedAsHistory {
		hist = 1
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO result_history(result_id, job_id, attempt, generation, state, payload_hash, created_at, acked_at, accepted_as_history, ack_json, stage)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CASE WHEN json_valid(?) THEN json_extract(?, '$.stage') END)`, e.ResultID, e.JobID, e.Attempt, e.Generation.String(), e.State, e.PayloadHash,
		e.CreatedAt.UTC().Format(time.RFC3339Nano), j.ts(), hist, raw, e.Payload, e.Payload); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM result_outbox WHERE result_id = ?`, a.ResultID); err != nil {
		return false, err
	}
	return false, tx.Commit()
}

// MarkConflict keeps a result whose ID central already holds with different
// content (ACK_CONFLICT, §13.8): the evidence stays, it is never resent and
// never overwritten.
func (j *Journal) MarkConflict(ctx context.Context, resultID, reason string) error {
	_, err := j.db.ExecContext(ctx, `UPDATE result_outbox SET status = ?, last_error = ?, last_attempt_at = ? WHERE result_id = ?`,
		string(OutboxConflict), reason, j.ts(), resultID)
	return err
}

// HistoryEntry is an acknowledged result summary kept for audit.
type HistoryEntry struct {
	ResultID          string
	JobID             string
	Attempt           int
	Generation        decstr.U64
	State             string
	PayloadHash       string
	AckedAt           time.Time
	AcceptedAsHistory bool
}

// History returns acknowledged results for a job ("" for all).
func (j *Journal) History(ctx context.Context, jobID string) ([]HistoryEntry, error) {
	q := `SELECT result_id, job_id, attempt, generation, state, payload_hash, acked_at, accepted_as_history FROM result_history`
	var args []any
	if jobID != "" {
		q += ` WHERE job_id = ?`
		args = append(args, jobID)
	}
	rows, err := j.db.QueryContext(ctx, q+` ORDER BY acked_at, result_id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HistoryEntry
	for rows.Next() {
		var h HistoryEntry
		var gen string
		var at sql.NullString
		var hist int
		if err := rows.Scan(&h.ResultID, &h.JobID, &h.Attempt, &gen, &h.State, &h.PayloadHash, &at, &hist); err != nil {
			return nil, err
		}
		h.Generation, _ = decstr.Parse(gen)
		h.AckedAt, h.AcceptedAsHistory = parseTS(at), hist != 0
		out = append(out, h)
	}
	return out, rows.Err()
}
