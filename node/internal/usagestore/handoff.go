package usagestore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// RetiredStream is a durable original stream, independent of its certificate.
// RetiredAt is local evidence only; central decides eligibility using revoked_at.
// Pending excludes final receipts; those payloads remain stored behind the ACK.
type RetiredStream struct {
	IncarnationID uuid.UUID
	RetiredAt     time.Time
	AckedThrough  uint64
	LastSeq       uint64
	LedgerEpoch   uuid.UUID
	BlockedSeq    *uint64
	Pending       int
	FinalCode     string
}

// HandoffRejection is an allowlisted local audit entry. Seq == 0 means the
// whole origin was refused. No remote message or event payload is copied here.
type HandoffRejection struct {
	Origin     uuid.UUID
	Presenter  uuid.UUID
	Seq        uint64
	Code       string
	RecordedAt time.Time
}

func (s *Store) migrateHandoff(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range []string{
		`CREATE TABLE retired_streams (
   incarnation_id TEXT PRIMARY KEY, retired_at TEXT NOT NULL,
   acked_through INTEGER NOT NULL, last_seq INTEGER NOT NULL,
   ledger_epoch TEXT, blocked_seq INTEGER, purged_through INTEGER NOT NULL DEFAULT 0,
   final_code TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE retired_events (
   incarnation_id TEXT NOT NULL REFERENCES retired_streams(incarnation_id),
   seq INTEGER NOT NULL, epoch_id TEXT NOT NULL, interval_start TEXT NOT NULL,
   interval_end TEXT NOT NULL, quality TEXT NOT NULL, payload BLOB NOT NULL,
   payload_sha256 TEXT NOT NULL, payload_len INTEGER NOT NULL,
   state TEXT NOT NULL, created_at TEXT NOT NULL, acked_at TEXT, acked_epoch TEXT,
   receipt TEXT NOT NULL DEFAULT '', PRIMARY KEY (incarnation_id, seq))`,
		`CREATE TABLE retired_replay_unavailable (
 incarnation_id TEXT NOT NULL REFERENCES retired_streams(incarnation_id),
 start_seq INTEGER NOT NULL, end_seq INTEGER NOT NULL, reason TEXT NOT NULL,
 PRIMARY KEY (incarnation_id, start_seq, end_seq))`,
		`CREATE TABLE handoff_rejections (
   origin TEXT NOT NULL REFERENCES retired_streams(incarnation_id),
   presenter TEXT NOT NULL, seq INTEGER NOT NULL, code TEXT NOT NULL,
   recorded_at TEXT NOT NULL, PRIMARY KEY (origin, seq))`,
		// Preserve non-payload recovery and meter history with its original identity.
		`CREATE TABLE retired_history (incarnation_id TEXT NOT NULL, kind TEXT NOT NULL,
   body TEXT NOT NULL, PRIMARY KEY (incarnation_id, kind))`,
		`PRAGMA user_version = 2`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) retirePrevious(ctx context.Context, origin string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var reused int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM retired_streams WHERE incarnation_id = ?`, s.incID.String()).Scan(&reused); err != nil {
		return err
	}
	if reused != 0 {
		return ErrIdentity
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO retired_streams
  (incarnation_id, retired_at, acked_through, last_seq, ledger_epoch, blocked_seq, purged_through)
  SELECT incarnation_id, ?, acked_through, next_seq - 1, ledger_epoch, blocked_seq, purged_through FROM meta WHERE id = 1`, s.ts(s.retirePreviousAt)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO retired_events
  (incarnation_id, seq, epoch_id, interval_start, interval_end, quality, payload,
   payload_sha256, payload_len, state, created_at, acked_at, acked_epoch)
  SELECT ?, seq, epoch_id, interval_start, interval_end, quality, payload,
   payload_sha256, payload_len, state, created_at, acked_at, acked_epoch FROM events`, origin); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO retired_replay_unavailable
 (incarnation_id, start_seq, end_seq, reason) SELECT ?, start_seq, end_seq, reason FROM replay_unavailable`, origin); err != nil {
		return err
	}
	// The history is audit evidence, never a current meter baseline. SQLite's
	// JSON aggregates retain integer/text values without a float64 conversion.
	for _, h := range []struct{ kind, query string }{
		{"epochs", `SELECT json_group_array(json_object('epoch_id',epoch_id,'opened_at',opened_at,'closed_at',closed_at,'open_reason',open_reason,'close_reason',close_reason,'boot_id',boot_id)) FROM epochs`},
		{"gaps", `SELECT json_group_array(json_object('seq',CAST(seq AS TEXT),'reason',reason,'last_checkpoint_at',last_checkpoint_at,'detail',detail,'created_at',created_at)) FROM gaps`},
		{"ledger_epochs", `SELECT json_group_array(json_object('ledger_epoch',ledger_epoch,'checkpoint_json',CAST(checkpoint_json AS TEXT))) FROM ledger_epochs`},
		{"replay_unavailable", `SELECT json_group_array(json_object('start_seq',CAST(start_seq AS TEXT),'end_seq',CAST(end_seq AS TEXT),'reason',reason)) FROM replay_unavailable`},
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO retired_history (incarnation_id, kind, body) SELECT ?, ?, (`+h.query+`)`, origin, h.kind); err != nil {
			return err
		}
	}
	for _, table := range []string{"events", "counters", "epochs", "gaps", "ledger_epochs", "replay_unavailable"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE meta SET incarnation_id = ?, next_seq = 1,
  open_epoch = NULL, open_reason = '', boot_id = '', last_close_reason = '',
  ledger_epoch = NULL, acked_through = 0, blocked_seq = NULL, last_sample = NULL,
  gate_stopped = 0, gate_reason = '', purged_through = 0 WHERE id = 1`, s.incID.String()); err != nil {
		return err
	}
	if err := s.hookStep(StepRetireBeforeCommit); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.hookStep(StepRetireAfterCommit)
}

// RetiredStreams lists local retirement order, including completed streams.
func (s *Store) RetiredStreams(ctx context.Context) ([]RetiredStream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrClosed
	}
	rows, err := s.db.QueryContext(ctx, `SELECT incarnation_id, retired_at, acked_through,
  last_seq, ledger_epoch, blocked_seq, final_code,
  (SELECT COUNT(*) FROM retired_events e WHERE e.incarnation_id = r.incarnation_id
   AND e.state = 'outbox' AND (e.receipt = '' OR
 (e.receipt IN ('accepted', 'duplicate') AND NOT EXISTS
  (SELECT 1 FROM handoff_rejections h WHERE h.origin = r.incarnation_id AND h.seq <= e.seq))))
  FROM retired_streams r ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RetiredStream
	for rows.Next() {
		var r RetiredStream
		var id, at string
		var epoch sql.NullString
		var blocked sql.NullInt64
		if err := rows.Scan(&id, &at, &r.AckedThrough, &r.LastSeq, &epoch, &blocked, &r.FinalCode, &r.Pending); err != nil {
			return nil, err
		}
		r.IncarnationID, err = uuid.Parse(id)
		if err != nil {
			return nil, err
		}
		r.RetiredAt, err = parseTime(at)
		if err != nil {
			return nil, err
		}
		if epoch.Valid && epoch.String != "" {
			r.LedgerEpoch, err = uuid.Parse(epoch.String)
			if err != nil {
				return nil, err
			}
		}
		if blocked.Valid {
			n := uint64(blocked.Int64)
			r.BlockedSeq = &n
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RetiredUnacked returns only pending original events, in sequence order.
// Terminal receipts stay in the database; they are not re-minted or trimmed.
func (s *Store) RetiredUnacked(ctx context.Context, origin uuid.UUID, maxEvents, maxBytes int) ([]StoredEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrClosed
	}
	if maxEvents <= 0 || maxEvents > 1000 {
		maxEvents = 1000
	}
	if maxBytes <= 0 {
		maxBytes = 512_000
	}
	rows, err := s.db.QueryContext(ctx, `SELECT seq, epoch_id, interval_start, interval_end,
  quality, payload, payload_sha256, created_at FROM retired_events
  WHERE incarnation_id = ? AND state = 'outbox' AND receipt = ''
  AND EXISTS (SELECT 1 FROM retired_streams WHERE incarnation_id = ? AND final_code = '')
  ORDER BY seq LIMIT ?`, origin.String(), origin.String(), maxEvents)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoredEvent
	used := 0
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		if len(out) > 0 && used+len(ev.Payload) > maxBytes {
			break
		}
		out = append(out, ev)
		used += len(ev.Payload)
	}
	return out, rows.Err()
}

// ApplyRetiredACK commits original-stream receipts, audit and watermark
// together. An accepted event behind a rejection stays unacknowledged but
// need not be sent again. The current stream is never touched.
func (s *Store) ApplyRetiredACK(ctx context.Context, origin uuid.UUID, ack BatchACK) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var highest, last uint64
	var epoch sql.NullString
	var oldBlocked sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT acked_through, last_seq, ledger_epoch, blocked_seq
  FROM retired_streams WHERE incarnation_id = ?`, origin.String()).Scan(&highest, &last, &epoch, &oldBlocked); err != nil {
		return err
	}
	if epoch.Valid && epoch.String != "" && epoch.String != ack.LedgerEpoch.String() {
		return ErrLedgerEpoch
	}
	if ack.Highest < highest {
		return ErrAckRegress
	}
	if ack.Highest > last {
		return fmt.Errorf("usagestore: hand-off ACK beyond original stream")
	}
	if ack.Blocked != nil && (*ack.Blocked == 0 || *ack.Blocked > last || *ack.Blocked <= highest) {
		return fmt.Errorf("usagestore: invalid hand-off blocked sequence")
	}
	blocked := ack.Blocked
	if oldBlocked.Valid && (blocked == nil || uint64(oldBlocked.Int64) < *blocked) {
		v := uint64(oldBlocked.Int64)
		blocked = &v
	}
	for _, r := range ack.Results {
		if r.Seq == 0 || r.Seq > last {
			return fmt.Errorf("usagestore: invalid hand-off receipt sequence")
		}
		switch r.Status {
		case StatusAccepted, StatusDuplicate:
		case StatusRejected, StatusConflict:
			if blocked == nil || r.Seq < *blocked {
				v := r.Seq
				blocked = &v
			}
			code := r.Code
			if !apierr.Code(code).Valid() {
				code = string(apierr.ValidationFailed)
			}
			if err := s.rejectTx(ctx, tx, origin, r.Seq, code); err != nil {
				return err
			}
		default:
			return fmt.Errorf("usagestore: invalid hand-off receipt status")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE retired_events SET receipt = ? WHERE incarnation_id = ? AND seq = ? AND receipt = ''`, r.Status, origin.String(), int64(r.Seq)); err != nil {
			return err
		}
	}
	limit := ack.Highest
	if blocked != nil && *blocked > 0 && *blocked-1 < limit {
		limit = *blocked - 1
	}
	if limit < highest {
		return ErrAckRegress
	}
	ranges, err := retiredRanges(ctx, tx, origin)
	if err != nil {
		return err
	}
	advanceMissing := func() {
		for _, r := range ranges {
			if uint64(r.StartSeq) <= highest+1 && uint64(r.EndSeq) > highest {
				highest = min(uint64(r.EndSeq), limit)
			}
		}
	}
	advanceMissing()
	rows, err := tx.QueryContext(ctx, `SELECT seq, receipt FROM retired_events
  WHERE incarnation_id = ? AND seq > ? AND seq <= ? ORDER BY seq`, origin.String(), int64(highest), int64(limit))
	if err != nil {
		return err
	}
	for rows.Next() {
		var seq uint64
		var receipt string
		if err := rows.Scan(&seq, &receipt); err != nil {
			rows.Close()
			return err
		}
		if seq != highest+1 || (receipt != StatusAccepted && receipt != StatusDuplicate) {
			break
		}
		highest = seq
		advanceMissing()
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE retired_events SET state = 'archived', acked_at = ?, acked_epoch = ?
  WHERE incarnation_id = ? AND seq <= ? AND state = 'outbox'`, s.ts(s.now()), epoch.String, origin.String(), int64(highest)); err != nil {
		return err
	}
	var b any
	if blocked != nil {
		b = int64(*blocked)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE retired_streams SET acked_through = ?, blocked_seq = ? WHERE incarnation_id = ?`, int64(highest), b, origin.String()); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM retired_replay_unavailable WHERE incarnation_id = ? AND end_seq <= ?`, origin.String(), int64(highest)); err != nil {
		return err
	}
	if err := s.hookStep(StepACKBeforeCommit); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.hookStep(StepACKAfterCommit)
}

func (s *Store) rejectTx(ctx context.Context, tx *sql.Tx, origin uuid.UUID, seq uint64, code string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO handoff_rejections (origin, presenter, seq, code, recorded_at)
  VALUES (?, ?, ?, ?, ?) ON CONFLICT(origin, seq) DO NOTHING`, origin.String(), s.incID.String(), int64(seq), code, s.ts(s.now()))
	return err
}

// RejectRetired records a final whole-batch identity/origin rejection once.
func (s *Store) RejectRetired(ctx context.Context, origin uuid.UUID, code string) error {
	if code != string(apierr.StateConflict) && code != string(apierr.IncarnationMismatch) {
		return fmt.Errorf("usagestore: not a final origin rejection")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.rejectTx(ctx, tx, origin, 0, code); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE retired_streams SET final_code = ? WHERE incarnation_id = ? AND final_code = ''`, code, origin.String()); err != nil {
		return err
	}
	if err := s.hookStep(StepACKBeforeCommit); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.hookStep(StepACKAfterCommit)
}

// HandoffRejections reads the durable, redacted audit evidence.
func (s *Store) HandoffRejections(ctx context.Context) ([]HandoffRejection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrClosed
	}
	rows, err := s.db.QueryContext(ctx, `SELECT origin, presenter, seq, code, recorded_at FROM handoff_rejections ORDER BY rowid`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HandoffRejection
	for rows.Next() {
		var r HandoffRejection
		var origin, presenter, at string
		if err := rows.Scan(&origin, &presenter, &r.Seq, &r.Code, &at); err != nil {
			return nil, err
		}
		r.Origin, err = uuid.Parse(origin)
		if err != nil {
			return nil, err
		}
		r.Presenter, err = uuid.Parse(presenter)
		if err != nil {
			return nil, err
		}
		r.RecordedAt, err = parseTime(at)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RetiredReplayUnavailable retains the original stream's recovery gap ranges.
func (s *Store) RetiredReplayUnavailable(ctx context.Context, origin uuid.UUID) ([]api.ReplayUnavailableRange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrClosed
	}
	return retiredRanges(ctx, s.db, origin)
}

type handoffQuery interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func retiredRanges(ctx context.Context, q handoffQuery, origin uuid.UUID) ([]api.ReplayUnavailableRange, error) {
	rows, err := q.QueryContext(ctx, `SELECT start_seq, end_seq, reason FROM retired_replay_unavailable WHERE incarnation_id = ? ORDER BY start_seq LIMIT 100`, origin.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []api.ReplayUnavailableRange
	for rows.Next() {
		var start, end uint64
		var reason string
		if err := rows.Scan(&start, &end, &reason); err != nil {
			return nil, err
		}
		out = append(out, api.ReplayUnavailableRange{StartSeq: decstr.U64(start), EndSeq: decstr.U64(end), Reason: api.ReplayUnavailableRangeReason(reason)})
	}
	return out, rows.Err()
}

// Retained outbox bytes (even final rejections) still consume the node's
// capacity budget. Status/snapshot remain views of the current stream.
func (s *Store) addRetiredHealth(ctx context.Context, h *Health) error {
	var count int
	var nbytes, archive uint64
	var oldest sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(payload_len),0), MIN(created_at) FROM retired_events WHERE state = 'outbox'`).Scan(&count, &nbytes, &oldest); err != nil {
		return err
	}
	h.UnackedCount += count
	h.UnackedBytes += nbytes
	if oldest.Valid {
		at, err := parseTime(oldest.String)
		if err != nil {
			return err
		}
		if h.OldestUnacked.IsZero() || at.Before(h.OldestUnacked) {
			h.OldestUnacked = at
		}
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(payload_len),0) FROM retired_events WHERE state = 'archived'`).Scan(&archive); err != nil {
		return err
	}
	h.ArchiveBytes += archive
	return nil
}

func (s *Store) purgeRetiredTx(ctx context.Context, tx *sql.Tx, cutoff time.Time) (int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT incarnation_id, seq, acked_at FROM retired_events WHERE state = 'archived' ORDER BY incarnation_id, seq`)
	if err != nil {
		return 0, err
	}
	type key struct {
		inc string
		seq int64
	}
	var drop []key
	protected := ""
	for rows.Next() {
		var k key
		var stamp string
		if err := rows.Scan(&k.inc, &k.seq, &stamp); err != nil {
			rows.Close()
			return 0, err
		}
		if k.inc == protected {
			continue
		}
		at, err := parseTime(stamp)
		if err != nil {
			rows.Close()
			return 0, err
		}
		if at.After(cutoff) {
			protected = k.inc
			continue
		}
		drop = append(drop, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, k := range drop {
		if _, err := tx.ExecContext(ctx, `DELETE FROM retired_events WHERE incarnation_id = ? AND seq = ? AND state = 'archived'`, k.inc, k.seq); err != nil {
			return 0, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE retired_streams SET purged_through = MAX(purged_through, ?) WHERE incarnation_id = ?`, k.seq, k.inc); err != nil {
			return 0, err
		}
	}
	return len(drop), nil
}
