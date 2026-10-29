package usagestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"

	_ "modernc.org/sqlite" // pure-Go SQLite (AGENTS.md §3)
)

const schemaVersion = 1

// Options configure Open. Zero durations and byte caps take the §10.5 and
// §10.8 defaults. MinFreeBytes stays zero until an operator sets it.
type Options struct {
	Now             func() time.Time
	Hook            Hook
	Trust           *envelope.TrustStore
	UnackedMaxAge   time.Duration
	UnackedMaxBytes uint64
	ArchiveRetain   time.Duration
	ArchiveMaxBytes uint64
	MinFreeBytes    uint64
	DiskFree        func(path string) (uint64, error)
	DiskFootprint   func(path string) (uint64, error)
}

// Store is an open usage database. It is safe for one process. The node
// lock serializes processes that would otherwise share the file.
type Store struct {
	db   *sql.DB
	path string
	mu   sync.Mutex

	nodeID uuid.UUID
	incID  uuid.UUID

	now   func() time.Time
	hook  Hook
	trust *envelope.TrustStore

	unackedMaxAge   time.Duration
	unackedMaxBytes uint64
	archiveRetain   time.Duration
	archiveMaxBytes uint64
	minFreeBytes    uint64
	diskFree        func(string) (uint64, error)
	diskFoot        func(string) (uint64, error)

	persistFailed bool
	closed        bool
}

// Open opens or creates the database at path. The file is mode 0600 and the
// directory is mode 0700. WAL and synchronous=FULL are set so a commit is
// durable before the caller is told it succeeded (§10.5).
func Open(path string, nodeID, incID uuid.UUID, opt Options) (*Store, error) {
	if nodeID == uuid.Nil || incID == uuid.Nil {
		return nil, fmt.Errorf("usagestore: node and incarnation are required")
	}
	if path == "" {
		path = DefaultPath
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	_, statErr := os.Stat(path)
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{
		db: db, path: path, nodeID: nodeID, incID: incID,
		now: opt.Now, hook: opt.Hook, trust: opt.Trust,
		unackedMaxAge: opt.UnackedMaxAge, unackedMaxBytes: opt.UnackedMaxBytes,
		archiveRetain: opt.ArchiveRetain, archiveMaxBytes: opt.ArchiveMaxBytes,
		minFreeBytes: opt.MinFreeBytes,
		diskFree:     opt.DiskFree, diskFoot: opt.DiskFootprint,
	}
	if s.now == nil {
		s.now = time.Now
	}
	if s.unackedMaxAge <= 0 {
		s.unackedMaxAge = DefaultUnackedMaxAge
	}
	if s.unackedMaxBytes == 0 {
		s.unackedMaxBytes = DefaultUnackedMaxBytes
	}
	if s.archiveRetain <= 0 {
		s.archiveRetain = DefaultArchiveRetain
	}
	if s.archiveMaxBytes == 0 {
		s.archiveMaxBytes = DefaultArchiveMaxBytes
	}
	if s.diskFree == nil {
		s.diskFree = freeSpace
	}
	if s.diskFoot == nil {
		s.diskFoot = footprint
	}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if errors.Is(statErr, os.ErrNotExist) {
		_ = os.Chmod(path, 0o600)
		if d, err := os.Open(dir); err == nil {
			_ = d.Sync()
			_ = d.Close()
		}
	}
	return s, nil
}

// Close closes the database.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return s.db.Close()
}

// Path returns the database path.
func (s *Store) Path() string { return s.path }

// Identity returns the node and incarnation this database serves.
func (s *Store) Identity() (uuid.UUID, uuid.UUID) { return s.nodeID, s.incID }

// Fault runs the test hook. A nil hook is a no-op.
func (s *Store) Fault(step string) error {
	if s.hook == nil {
		return nil
	}
	if err := s.hook(step); err != nil {
		if errors.Is(err, ErrInjectedCrash) {
			return err
		}
		return fmt.Errorf("%w: %s", ErrInjectedCrash, err.Error())
	}
	return nil
}

// NotePersistFailure records that a sample could not be written (§10.7).
func (s *Store) NotePersistFailure() {
	s.mu.Lock()
	s.persistFailed = true
	s.mu.Unlock()
}

// NotePersistSuccess clears a persist failure after a durable commit.
func (s *Store) NotePersistSuccess() {
	s.mu.Lock()
	s.persistFailed = false
	s.mu.Unlock()
}

func (s *Store) migrate(ctx context.Context) error {
	var v int
	if err := s.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil {
		return fmt.Errorf("usagestore: read schema version: %w", err)
	}
	if v > schemaVersion {
		return fmt.Errorf("usagestore: schema %d is newer than supported %d", v, schemaVersion)
	}
	if v == schemaVersion {
		return s.checkIdentity(ctx)
	}
	stmts := []string{
		`CREATE TABLE meta (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			node_id TEXT NOT NULL,
			incarnation_id TEXT NOT NULL,
			next_seq INTEGER NOT NULL,
			open_epoch TEXT,
			open_reason TEXT NOT NULL DEFAULT '',
			boot_id TEXT NOT NULL DEFAULT '',
			last_close_reason TEXT NOT NULL DEFAULT '',
			ledger_epoch TEXT,
			acked_through INTEGER NOT NULL DEFAULT 0,
			blocked_seq INTEGER,
			last_sample TEXT,
			gate_stopped INTEGER NOT NULL DEFAULT 0,
			gate_reason TEXT NOT NULL DEFAULT '',
			purged_through INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE epochs (
			epoch_id TEXT PRIMARY KEY,
			opened_at TEXT NOT NULL,
			closed_at TEXT,
			open_reason TEXT NOT NULL,
			close_reason TEXT,
			boot_id TEXT NOT NULL)`,
		`CREATE TABLE counters (
			epoch_id TEXT NOT NULL,
			name TEXT NOT NULL,
			device_id TEXT NOT NULL,
			direction TEXT NOT NULL,
			packets TEXT NOT NULL,
			bytes TEXT NOT NULL,
			member_id TEXT NOT NULL,
			binding_revision TEXT NOT NULL,
			PRIMARY KEY (epoch_id, name))`,
		`CREATE TABLE events (
			seq INTEGER PRIMARY KEY,
			epoch_id TEXT NOT NULL,
			interval_start TEXT NOT NULL,
			interval_end TEXT NOT NULL,
			quality TEXT NOT NULL,
			payload BLOB NOT NULL,
			payload_sha256 TEXT NOT NULL,
			payload_len INTEGER NOT NULL,
			state TEXT NOT NULL CHECK (state IN ('outbox', 'archived')),
			created_at TEXT NOT NULL,
			acked_at TEXT,
			acked_epoch TEXT)`,
		`CREATE INDEX events_state ON events(state, seq)`,
		`CREATE TABLE gaps (
			id INTEGER PRIMARY KEY,
			seq INTEGER,
			reason TEXT NOT NULL,
			last_checkpoint_at TEXT,
			detail TEXT NOT NULL,
			created_at TEXT NOT NULL)`,
		`CREATE TABLE ledger_epochs (
			ledger_epoch TEXT PRIMARY KEY,
			accepted_at TEXT NOT NULL,
			acked_through INTEGER NOT NULL,
			recovery_record_id TEXT,
			checkpoint_json BLOB NOT NULL,
			superseded_at TEXT)`,
		`CREATE TABLE replay_unavailable (
			start_seq INTEGER NOT NULL,
			end_seq INTEGER NOT NULL,
			reason TEXT NOT NULL,
			PRIMARY KEY (start_seq, end_seq))`,
		`CREATE TABLE upload_batches (
			body_sha256 TEXT PRIMARY KEY,
			idempotency_key TEXT NOT NULL,
			body BLOB NOT NULL,
			first_seq INTEGER NOT NULL,
			last_seq INTEGER NOT NULL,
			created_at TEXT NOT NULL)`,
		fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion),
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("usagestore: migrate: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO meta (id, node_id, incarnation_id, next_seq) VALUES (1, ?, ?, 1)`,
		s.nodeID.String(), s.incID.String()); err != nil {
		return fmt.Errorf("usagestore: migrate: %w", err)
	}
	return tx.Commit()
}

func (s *Store) checkIdentity(ctx context.Context) error {
	var node, inc string
	err := s.db.QueryRowContext(ctx, `SELECT node_id, incarnation_id FROM meta WHERE id = 1`).Scan(&node, &inc)
	if err != nil {
		return fmt.Errorf("usagestore: read identity: %w", err)
	}
	if node != s.nodeID.String() || inc != s.incID.String() {
		return ErrIdentity
	}
	return nil
}

func (s *Store) ts(t time.Time) string {
	if t.IsZero() {
		t = s.now()
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// Head reads the cursor. It does not take the write lock for long; callers
// that plan a commit must tolerate a concurrent writer only within this process,
// and Commit holds the lock across the read it needs.
func (s *Store) Head(ctx context.Context) (Head, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Head{}, ErrClosed
	}
	return s.head(ctx)
}

func (s *Store) head(ctx context.Context) (Head, error) {
	var h Head
	var openEpoch, ledger, lastSample sql.NullString
	var blocked sql.NullInt64
	var next, acked int64
	var openReason, boot, lastClose string
	err := s.db.QueryRowContext(ctx, `SELECT next_seq, open_epoch, open_reason, boot_id,
		last_close_reason, ledger_epoch, acked_through, blocked_seq, last_sample
		FROM meta WHERE id = 1`).Scan(
		&next, &openEpoch, &openReason, &boot, &lastClose, &ledger, &acked, &blocked, &lastSample)
	if err != nil {
		return Head{}, err
	}
	if next < 1 {
		return Head{}, fmt.Errorf("usagestore: next_seq corrupt")
	}
	h.NextSeq = uint64(next)
	h.AckedThrough = uint64(acked)
	h.OpenReason = openReason
	h.BootID = boot
	h.LastCloseReason = lastClose
	if openEpoch.Valid && openEpoch.String != "" {
		id, err := uuid.Parse(openEpoch.String)
		if err != nil {
			return Head{}, err
		}
		h.OpenEpoch = id
	}
	if ledger.Valid && ledger.String != "" {
		id, err := uuid.Parse(ledger.String)
		if err != nil {
			return Head{}, err
		}
		h.LedgerEpoch = id
	}
	if blocked.Valid && blocked.Int64 > 0 {
		v := uint64(blocked.Int64)
		h.BlockedSeq = &v
	}
	if lastSample.Valid {
		t, err := parseTime(lastSample.String)
		if err != nil {
			return Head{}, err
		}
		h.LastSample = t
	}
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM epochs`).Scan(&n); err != nil {
		return Head{}, err
	}
	h.HasHistory = n > 0
	if h.OpenEpoch != uuid.Nil {
		h.Counters, err = s.counters(ctx, h.OpenEpoch)
		if err != nil {
			return Head{}, err
		}
	}
	return h, nil
}

func (s *Store) counters(ctx context.Context, epoch uuid.UUID) ([]Counter, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name, device_id, direction, packets, bytes, member_id, binding_revision
		FROM counters WHERE epoch_id = ? ORDER BY name`, epoch.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Counter
	for rows.Next() {
		var c Counter
		var dev, mem, packets, bytes, rev string
		if err := rows.Scan(&c.Name, &dev, &c.Direction, &packets, &bytes, &mem, &rev); err != nil {
			return nil, err
		}
		c.DeviceID, err = uuid.Parse(dev)
		if err != nil {
			return nil, err
		}
		c.MemberID, err = uuid.Parse(mem)
		if err != nil {
			return nil, err
		}
		p, err := decstr.Parse(packets)
		if err != nil {
			return nil, err
		}
		b, err := decstr.Parse(bytes)
		if err != nil {
			return nil, err
		}
		r, err := decstr.Parse(rev)
		if err != nil {
			return nil, err
		}
		c.Packets = p.Uint64()
		c.Bytes = b.Uint64()
		c.BindingRevision = r.Uint64()
		out = append(out, c)
	}
	return out, rows.Err()
}

// Commit persists a sample. See the Commit type for the transaction rules.
func (s *Store) Commit(ctx context.Context, c Commit) (seq uint64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return 0, ErrClosed
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	seq, err = s.commitTx(ctx, tx, c)
	if err != nil {
		return 0, err
	}
	if err := s.hookStep(StepBeforeCommit); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	if err := s.hookStep(StepAfterCommit); err != nil {
		return seq, err
	}
	return seq, nil
}

func (s *Store) hookStep(step string) error {
	if s.hook == nil {
		return nil
	}
	if err := s.hook(step); err != nil {
		if errors.Is(err, ErrInjectedCrash) {
			return err
		}
		return fmt.Errorf("%w: %s", ErrInjectedCrash, err.Error())
	}
	return nil
}

func (s *Store) commitTx(ctx context.Context, tx *sql.Tx, c Commit) (uint64, error) {
	h, err := s.headTx(ctx, tx)
	if err != nil {
		return 0, err
	}
	target := c.Epoch
	if c.Open != nil {
		if c.Open.ID == uuid.Nil {
			return 0, fmt.Errorf("usagestore: new epoch id is required")
		}
		if c.Counters == nil {
			return 0, fmt.Errorf("usagestore: opening an epoch requires a counter checkpoint")
		}
		target = c.Open.ID
	}
	if target == uuid.Nil && !c.CloseEpoch {
		return 0, fmt.Errorf("usagestore: epoch is required")
	}
	if c.Open == nil && c.Epoch != uuid.Nil && h.OpenEpoch != uuid.Nil && c.Epoch != h.OpenEpoch {
		return 0, fmt.Errorf("usagestore: commit epoch is not the open epoch")
	}
	if c.Open == nil && h.OpenEpoch == uuid.Nil && !c.CloseEpoch && c.Event != nil {
		return 0, fmt.Errorf("usagestore: no open meter epoch")
	}

	sameEpoch := c.Open == nil && !c.CloseEpoch
	if c.Open == nil && c.CloseEpoch {
		sameEpoch = true // final sample still updates the epoch it closes
	}
	if sameEpoch && h.OpenEpoch != uuid.Nil {
		if err := checkAdvance(h.Counters, c.Counters, c.Event); err != nil {
			return 0, err
		}
	}
	if c.Open != nil && h.OpenEpoch != uuid.Nil && !c.CloseEpoch {
		return 0, fmt.Errorf("usagestore: meter epoch is already open")
	}
	if c.Open != nil && c.CloseEpoch && h.OpenEpoch != uuid.Nil {
		if c.Event == nil || c.Event.Quality != "gap" {
			return 0, fmt.Errorf("usagestore: an epoch switch requires a gap event")
		}
	}

	var seq uint64
	if c.Event != nil {
		if err := s.hookStep(StepInsert); err != nil {
			return 0, err
		}
		seq, err = s.insertEvent(ctx, tx, h.NextSeq, c)
		if err != nil {
			return 0, err
		}
	}
	if err := s.hookStep(StepCheckpoint); err != nil {
		return 0, err
	}
	now := c.SampledAt
	if now.IsZero() {
		now = s.now()
	}
	if c.Event != nil {
		now = c.Event.End
	}
	if err := s.writeCursor(ctx, tx, h, c, now); err != nil {
		return 0, err
	}
	return seq, nil
}

func checkAdvance(old, next []Counter, ev *Draft) error {
	prev := make(map[string]Counter, len(old))
	for _, c := range old {
		prev[c.Name] = c
	}
	changed := false
	decreased := false
	up := map[uuid.UUID]uint64{}
	down := map[uuid.UUID]uint64{}
	seen := map[string]bool{}
	for _, c := range next {
		seen[c.Name] = true
		p, ok := prev[c.Name]
		var delta uint64
		if !ok {
			delta = c.Bytes
		} else {
			if c.Bytes < p.Bytes || c.Packets < p.Packets {
				decreased = true
				continue
			}
			delta = c.Bytes - p.Bytes
		}
		if delta > 0 {
			changed = true
			if c.Direction == "upload" {
				up[c.DeviceID] += delta
			} else if c.Direction == "download" {
				down[c.DeviceID] += delta
			}
		}
	}
	for name := range prev {
		if !seen[name] {
			decreased = true
		}
	}
	if !changed && !decreased {
		if ev != nil && ev.Quality != "gap" && ev.Quality != "boundary_estimated" {
			// A quiet interval must not invent an event. A gap or a
			// boundary mark may still be recorded with no new bytes.
			if len(ev.Records) > 0 {
				return ErrDeltaMismatch
			}
		}
		return nil
	}
	if ev == nil {
		return ErrCheckpointWithoutEvent
	}
	if decreased && ev.Quality != "gap" {
		return ErrCounterRegress
	}
	if ev.Quality == "gap" {
		// A gap may report only the increases of counters that stayed
		// continuous. Reset counters contribute unknown bytes, never a
		// claimed delta (§10.7).
		for _, r := range ev.Records {
			if r.Upload > up[r.DeviceID] || r.Download > down[r.DeviceID] {
				return ErrDeltaMismatch
			}
		}
		return nil
	}
	gotUp := map[uuid.UUID]uint64{}
	gotDown := map[uuid.UUID]uint64{}
	for _, r := range ev.Records {
		gotUp[r.DeviceID] += r.Upload
		gotDown[r.DeviceID] += r.Download
	}
	if !sameTotals(up, gotUp) || !sameTotals(down, gotDown) {
		return ErrDeltaMismatch
	}
	return nil
}

func sameTotals(a, b map[uuid.UUID]uint64) bool {
	for id, v := range a {
		if b[id] != v {
			return false
		}
	}
	for id, v := range b {
		if v != 0 && a[id] != v {
			return false
		}
	}
	return true
}

func (s *Store) insertEvent(ctx context.Context, tx *sql.Tx, seq uint64, c Commit) (uint64, error) {
	if seq == 0 || seq >= 1<<63-1 {
		return 0, fmt.Errorf("usagestore: event_seq %d is outside the durable range", seq)
	}
	d := *c.Event
	if d.EpochID == uuid.Nil {
		if c.Open != nil {
			d.EpochID = c.Open.ID
		} else {
			d.EpochID = c.Epoch
		}
	}
	payload, sum, err := canonicalEvent(s.nodeID, s.incID, seq, d)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO events (
		seq, epoch_id, interval_start, interval_end, quality, payload, payload_sha256, payload_len, state, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'outbox', ?)`,
		int64(seq), d.EpochID.String(), s.ts(d.Start), s.ts(d.End), d.Quality,
		payload, sum, len(payload), s.ts(s.now()))
	if err != nil {
		return 0, fmt.Errorf("usagestore: insert event: %w", err)
	}
	if d.Quality == "gap" {
		var last any
		if d.LastCheckpoint != nil {
			last = s.ts(*d.LastCheckpoint)
		}
		detail := d.Detail
		if len(detail) > 200 {
			detail = detail[:200]
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO gaps (seq, reason, last_checkpoint_at, detail, created_at)
			VALUES (?, ?, ?, ?, ?)`, int64(seq), d.GapReason, last, detail, s.ts(s.now())); err != nil {
			return 0, err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE meta SET next_seq = ? WHERE id = 1`, int64(seq)+1); err != nil {
		return 0, err
	}
	return seq, nil
}

func (s *Store) writeCursor(ctx context.Context, tx *sql.Tx, h Head, c Commit, sampled time.Time) error {
	// Counters of a final sample stay on the epoch that is closing. An
	// epoch switch writes the new baseline only on the new epoch and leaves
	// the old checkpoint in place for audit (§10.7, §13.6 step 8).
	counterEpoch := h.OpenEpoch
	if c.Open != nil {
		counterEpoch = c.Open.ID
	}
	if c.CloseEpoch {
		if h.OpenEpoch == uuid.Nil {
			return fmt.Errorf("usagestore: no open epoch to close")
		}
		reason := c.CloseReason
		if reason == "" {
			reason = "closed"
		}
		if _, err := tx.ExecContext(ctx, `UPDATE epochs SET closed_at = ?, close_reason = ? WHERE epoch_id = ? AND closed_at IS NULL`,
			s.ts(sampled), reason, h.OpenEpoch.String()); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE meta SET open_epoch = NULL, open_reason = '', last_close_reason = ? WHERE id = 1`, reason); err != nil {
			return err
		}
	}
	if c.Open != nil {
		boot := c.Open.BootID
		if boot == "" {
			boot = c.BootID
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO epochs (epoch_id, opened_at, open_reason, boot_id)
			VALUES (?, ?, ?, ?)`, c.Open.ID.String(), s.ts(sampled), c.Open.Reason, boot); err != nil {
			return fmt.Errorf("usagestore: open epoch: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE meta SET open_epoch = ?, open_reason = ?, boot_id = ? WHERE id = 1`,
			c.Open.ID.String(), c.Open.Reason, boot); err != nil {
			return err
		}
	} else if c.BootID != "" && counterEpoch != uuid.Nil {
		if _, err := tx.ExecContext(ctx, `UPDATE meta SET boot_id = ? WHERE id = 1`, c.BootID); err != nil {
			return err
		}
	}
	if counterEpoch != uuid.Nil && c.Counters != nil {
		if _, err := tx.ExecContext(ctx, `DELETE FROM counters WHERE epoch_id = ?`, counterEpoch.String()); err != nil {
			return err
		}
		for _, ctr := range c.Counters {
			if ctr.Name == "" || ctr.DeviceID == uuid.Nil {
				return fmt.Errorf("usagestore: counter name and device are required")
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO counters (
				epoch_id, name, device_id, direction, packets, bytes, member_id, binding_revision)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				counterEpoch.String(), ctr.Name, ctr.DeviceID.String(), ctr.Direction,
				decstr.U64(ctr.Packets).String(), decstr.U64(ctr.Bytes).String(),
				ctr.MemberID.String(), decstr.U64(ctr.BindingRevision).String()); err != nil {
				return fmt.Errorf("usagestore: checkpoint counter: %w", err)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE meta SET last_sample = ? WHERE id = 1`, s.ts(sampled)); err != nil {
		return err
	}
	return nil
}

func (s *Store) headTx(ctx context.Context, tx *sql.Tx) (Head, error) {
	var h Head
	var openEpoch, ledger, lastSample sql.NullString
	var blocked sql.NullInt64
	var next, acked int64
	var openReason, boot, lastClose string
	err := tx.QueryRowContext(ctx, `SELECT next_seq, open_epoch, open_reason, boot_id,
		last_close_reason, ledger_epoch, acked_through, blocked_seq, last_sample
		FROM meta WHERE id = 1`).Scan(
		&next, &openEpoch, &openReason, &boot, &lastClose, &ledger, &acked, &blocked, &lastSample)
	if err != nil {
		return Head{}, err
	}
	h.NextSeq = uint64(next)
	h.AckedThrough = uint64(acked)
	h.OpenReason = openReason
	h.BootID = boot
	h.LastCloseReason = lastClose
	if openEpoch.Valid && openEpoch.String != "" {
		h.OpenEpoch, err = uuid.Parse(openEpoch.String)
		if err != nil {
			return Head{}, err
		}
	}
	if ledger.Valid && ledger.String != "" {
		h.LedgerEpoch, err = uuid.Parse(ledger.String)
		if err != nil {
			return Head{}, err
		}
	}
	if blocked.Valid && blocked.Int64 > 0 {
		v := uint64(blocked.Int64)
		h.BlockedSeq = &v
	}
	if lastSample.Valid {
		h.LastSample, err = parseTime(lastSample.String)
		if err != nil {
			return Head{}, err
		}
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM epochs`).Scan(&n); err != nil {
		return Head{}, err
	}
	h.HasHistory = n > 0
	if h.OpenEpoch != uuid.Nil {
		rows, err := tx.QueryContext(ctx, `SELECT name, device_id, direction, packets, bytes, member_id, binding_revision
			FROM counters WHERE epoch_id = ? ORDER BY name`, h.OpenEpoch.String())
		if err != nil {
			return Head{}, err
		}
		defer rows.Close()
		for rows.Next() {
			var c Counter
			var dev, mem, packets, bytes, rev string
			if err := rows.Scan(&c.Name, &dev, &c.Direction, &packets, &bytes, &mem, &rev); err != nil {
				rows.Close()
				return Head{}, err
			}
			c.DeviceID, err = uuid.Parse(dev)
			if err != nil {
				rows.Close()
				return Head{}, err
			}
			c.MemberID, err = uuid.Parse(mem)
			if err != nil {
				rows.Close()
				return Head{}, err
			}
			pv, err := decstr.Parse(packets)
			if err != nil {
				rows.Close()
				return Head{}, err
			}
			bv, err := decstr.Parse(bytes)
			if err != nil {
				rows.Close()
				return Head{}, err
			}
			rv, err := decstr.Parse(rev)
			if err != nil {
				rows.Close()
				return Head{}, err
			}
			c.Packets, c.Bytes, c.BindingRevision = pv.Uint64(), bv.Uint64(), rv.Uint64()
			h.Counters = append(h.Counters, c)
		}
		if err := rows.Err(); err != nil {
			return Head{}, err
		}
	}
	return h, nil
}
