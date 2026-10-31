package usagestore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// Unacked returns outbox events with seq > after, in order, until adding
// another event would pass maxEvents or maxBytes. maxEvents and maxBytes
// are the batch threshold (§10.5). A single event larger than maxBytes is
// still returned alone so it can be uploaded.
func (s *Store) Unacked(ctx context.Context, after uint64, maxEvents int, maxBytes int) ([]StoredEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrClosed
	}
	if maxEvents <= 0 {
		maxEvents = 100
	}
	if maxBytes <= 0 {
		maxBytes = 512_000
	}
	rows, err := s.db.QueryContext(ctx, `SELECT seq, epoch_id, interval_start, interval_end, quality,
		payload, payload_sha256, created_at FROM events
		WHERE state = 'outbox' AND seq > ? ORDER BY seq`, int64(after))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StoredEvent
	var used int
	for rows.Next() {
		ev, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		if len(out) > 0 && (len(out) >= maxEvents || used+len(ev.Payload) > maxBytes) {
			break
		}
		used += len(ev.Payload)
		out = append(out, ev)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(sc rowScanner) (StoredEvent, error) {
	var ev StoredEvent
	var seq int64
	var epoch, start, end, created string
	if err := sc.Scan(&seq, &epoch, &start, &end, &ev.Quality, &ev.Payload, &ev.Hash, &created); err != nil {
		return StoredEvent{}, err
	}
	var err error
	ev.Seq = uint64(seq)
	ev.EpochID, err = uuid.Parse(epoch)
	if err != nil {
		return StoredEvent{}, err
	}
	ev.Start, err = parseTime(start)
	if err != nil {
		return StoredEvent{}, err
	}
	ev.End, err = parseTime(end)
	if err != nil {
		return StoredEvent{}, err
	}
	ev.CreatedAt, err = parseTime(created)
	if err != nil {
		return StoredEvent{}, err
	}
	ev.State = "outbox"
	return ev, nil
}

// Event returns one stored event, outbox or archive. The payload is the
// original canonical bytes (§10.5).
func (s *Store) Event(ctx context.Context, seq uint64) (StoredEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.db.QueryRowContext(ctx, `SELECT seq, epoch_id, interval_start, interval_end, quality,
		payload, payload_sha256, state, created_at, acked_at FROM events WHERE seq = ?`, int64(seq))
	var ev StoredEvent
	var n int64
	var epoch, start, end, created string
	var acked sql.NullString
	if err := row.Scan(&n, &epoch, &start, &end, &ev.Quality, &ev.Payload, &ev.Hash, &ev.State, &created, &acked); err != nil {
		return StoredEvent{}, err
	}
	var err error
	ev.Seq = uint64(n)
	ev.EpochID, err = uuid.Parse(epoch)
	if err != nil {
		return StoredEvent{}, err
	}
	ev.Start, err = parseTime(start)
	if err != nil {
		return StoredEvent{}, err
	}
	ev.End, err = parseTime(end)
	if err != nil {
		return StoredEvent{}, err
	}
	ev.CreatedAt, err = parseTime(created)
	if err != nil {
		return StoredEvent{}, err
	}
	if acked.Valid {
		ev.AckedAt, err = parseTime(acked.String)
		if err != nil {
			return StoredEvent{}, err
		}
	}
	return ev, nil
}

// ApplyACK moves the contiguous prefix into the replay archive (§10.6).
// It never lowers the ACK, never skips a number, and never archives past
// blocked_seq or past a conflict or rejection.
func (s *Store) ApplyACK(ctx context.Context, ack BatchACK) error {
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
	if err := s.applyACKTx(ctx, tx, ack); err != nil {
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

func (s *Store) applyACKTx(ctx context.Context, tx *sql.Tx, ack BatchACK) error {
	h, err := s.headTx(ctx, tx)
	if err != nil {
		return err
	}
	if h.LedgerEpoch != uuid.Nil && ack.LedgerEpoch != uuid.Nil && ack.LedgerEpoch != h.LedgerEpoch {
		return ErrLedgerEpoch
	}
	limit := ack.Highest
	if ack.Blocked != nil && *ack.Blocked > 0 && *ack.Blocked-1 < limit {
		limit = *ack.Blocked - 1
	}
	accepted := make(map[uint64]bool, len(ack.Results))
	for _, r := range ack.Results {
		switch r.Status {
		case StatusAccepted, StatusDuplicate:
			accepted[r.Seq] = true
		case StatusConflict, StatusRejected:
			if r.Seq > 0 && r.Seq-1 < limit {
				limit = r.Seq - 1
			}
		default:
			return fmt.Errorf("usagestore: unknown event status %q", r.Status)
		}
	}
	if limit < h.AckedThrough {
		return ErrAckRegress
	}
	// Do not skip a number. A sequence counts only when it is already
	// archived, declared replay-unavailable, or accepted in this result
	// and still stored here (§10.6).
	accounted, err := s.accountedThrough(ctx, tx, h.AckedThrough, limit, accepted)
	if err != nil {
		return err
	}
	if accounted < limit {
		limit = accounted
	}
	if limit < h.AckedThrough {
		return ErrAckRegress
	}
	now := s.ts(s.now())
	epoch := ""
	if h.LedgerEpoch != uuid.Nil {
		epoch = h.LedgerEpoch.String()
	}
	if limit > h.AckedThrough {
		if _, err := tx.ExecContext(ctx, `UPDATE events SET state = 'archived', acked_at = ?, acked_epoch = ?
			WHERE state = 'outbox' AND seq > ? AND seq <= ?`,
			now, epoch, int64(h.AckedThrough), int64(limit)); err != nil {
			return err
		}
	}
	var blocked any
	if ack.Blocked != nil && *ack.Blocked > 0 {
		blocked = int64(*ack.Blocked)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE meta SET acked_through = ?, blocked_seq = ? WHERE id = 1`, int64(limit), blocked); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM replay_unavailable WHERE end_seq <= ?`, int64(limit)); err != nil {
		return err
	}
	return nil
}

// accountedThrough is the furthest seq <= want that is a contiguous prefix
// of accepted results, archived rows, or declared replay_unavailable ranges.
func (s *Store) accountedThrough(ctx context.Context, tx *sql.Tx, from, want uint64, accepted map[uint64]bool) (uint64, error) {
	if want <= from {
		return from, nil
	}
	have := map[uint64]bool{}
	rows, err := tx.QueryContext(ctx, `SELECT seq, state FROM events WHERE seq > ? AND seq <= ?`, int64(from), int64(want))
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	archived := map[uint64]bool{}
	for rows.Next() {
		var seq int64
		var state string
		if err := rows.Scan(&seq, &state); err != nil {
			return 0, err
		}
		have[uint64(seq)] = true
		if state == "archived" {
			archived[uint64(seq)] = true
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	unavail, err := s.replayRangesTx(ctx, tx)
	if err != nil {
		return 0, err
	}
	cur := from
	for cur < want {
		next := cur + 1
		if archived[next] || inRanges(unavail, next) || (accepted[next] && have[next]) {
			cur = next
			continue
		}
		break
	}
	return cur, nil
}

func inRanges(rs []seqRange, seq uint64) bool {
	for _, r := range rs {
		if seq >= r.Start && seq <= r.End {
			return true
		}
	}
	return false
}

type seqRange struct {
	Start  uint64
	End    uint64
	Reason string
}

func (s *Store) replayRangesTx(ctx context.Context, tx *sql.Tx) ([]seqRange, error) {
	rows, err := tx.QueryContext(ctx, `SELECT start_seq, end_seq, reason FROM replay_unavailable ORDER BY start_seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []seqRange
	for rows.Next() {
		var r seqRange
		var a, b int64
		if err := rows.Scan(&a, &b, &r.Reason); err != nil {
			return nil, err
		}
		r.Start, r.End = uint64(a), uint64(b)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReplayUnavailable returns inclusive ranges the node can no longer replay
// (§10.8 step 4).
func (s *Store) ReplayUnavailable(ctx context.Context) ([]api.ReplayUnavailableRange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT start_seq, end_seq, reason FROM replay_unavailable ORDER BY start_seq`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []api.ReplayUnavailableRange
	for rows.Next() {
		var a, b int64
		var reason string
		if err := rows.Scan(&a, &b, &reason); err != nil {
			return nil, err
		}
		out = append(out, api.ReplayUnavailableRange{
			StartSeq: decstr.U64(a),
			EndSeq:   decstr.U64(b),
			Reason:   api.ReplayUnavailableRangeReason(reason),
		})
	}
	return out, rows.Err()
}

// RememberBatch stores the exact request body and its Idempotency-Key.
// A retry of the same bytes reuses the key (§14.5, submitUsageBatch).
func (s *Store) RememberBatch(ctx context.Context, sha, key string, body []byte, first, last uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `INSERT INTO upload_batches (body_sha256, idempotency_key, body, first_seq, last_seq, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(body_sha256) DO NOTHING`,
		sha, key, body, int64(first), int64(last), s.ts(s.now()))
	return err
}

// LookupBatch returns the idempotency key and body saved for sha.
func (s *Store) LookupBatch(ctx context.Context, sha string) (key string, body []byte, ok bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.db.QueryRowContext(ctx, `SELECT idempotency_key, body FROM upload_batches WHERE body_sha256 = ?`, sha).Scan(&key, &body)
	if err == sql.ErrNoRows {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}
	return key, body, true, nil
}

// PurgeArchive deletes acknowledged payloads whose local ACK is older than
// the retention window (§10.8, §16.4). Rows inside the window stay.
// Unacknowledged events are never purged. Deletion is a prefix so a hole is
// not left inside the retained archive.
func (s *Store) PurgeArchive(ctx context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.IsZero() {
		now = s.now()
	}
	cutoff := now.Add(-s.archiveRetain)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `SELECT seq, acked_at FROM events WHERE state = 'archived' ORDER BY seq`)
	if err != nil {
		return 0, err
	}
	var drop []int64
	var purged int64
	for rows.Next() {
		var seq int64
		var acked string
		if err := rows.Scan(&seq, &acked); err != nil {
			rows.Close()
			return 0, err
		}
		at, err := parseTime(acked)
		if err != nil {
			rows.Close()
			return 0, err
		}
		if at.After(cutoff) {
			break
		}
		drop = append(drop, seq)
		purged = seq
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	for _, seq := range drop {
		if _, err := tx.ExecContext(ctx, `DELETE FROM events WHERE seq = ? AND state = 'archived'`, seq); err != nil {
			return 0, err
		}
	}
	if purged > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE meta SET purged_through = MAX(purged_through, ?) WHERE id = 1`, purged); err != nil {
			return 0, err
		}
	}
	retired, err := s.purgeRetiredTx(ctx, tx, cutoff)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(drop) + retired, nil
}

// EvaluateGate applies the outbox, archive and disk caps (§10.8).
// Stopping forwarding is sticky until every tripped meter is under 80%
// (hysteresis). A persist failure stops forwarding until a later commit
// succeeds (§10.7).
func (s *Store) EvaluateGate(ctx context.Context) (Health, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.evaluateGate(ctx)
}

// streamView reads the counters OutboxStatus and EvaluateGate share.
// It does not stat the disk, does not change persistFailed, and does not
// write the gate.
func (s *Store) streamView(ctx context.Context) (Health, error) {
	h := Health{ForwardingAllowed: true, PersistFailed: s.persistFailed}
	head, err := s.head(ctx)
	if err != nil {
		return Health{}, err
	}
	h.AckedThrough = head.AckedThrough
	h.BlockedSeq = head.BlockedSeq
	h.LedgerEpoch = head.LedgerEpoch
	h.MeterEpoch = head.OpenEpoch
	var last int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) FROM events`).Scan(&last); err != nil {
		return Health{}, err
	}
	h.LastSeq = uint64(last)

	var count int
	var bytes sql.NullInt64
	var oldest sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(payload_len), 0), MIN(created_at)
		FROM events WHERE state = 'outbox'`).Scan(&count, &bytes, &oldest); err != nil {
		return Health{}, err
	}
	h.UnackedCount = count
	if bytes.Valid && bytes.Int64 > 0 {
		h.UnackedBytes = uint64(bytes.Int64)
	}
	if oldest.Valid {
		h.OldestUnacked, err = parseTime(oldest.String)
		if err != nil {
			return Health{}, err
		}
	}
	var arch sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(payload_len), 0) FROM events WHERE state = 'archived'`).Scan(&arch); err != nil {
		return Health{}, err
	}
	if arch.Valid && arch.Int64 > 0 {
		h.ArchiveBytes = uint64(arch.Int64)
	}
	return h, nil
}

func (s *Store) evaluateGate(ctx context.Context) (Health, error) {
	h, err := s.streamView(ctx)
	if err != nil {
		return Health{}, err
	}
	if err := s.addRetiredHealth(ctx, &h); err != nil {
		return Health{}, err
	}
	if s.diskFoot != nil {
		n, err := s.diskFoot(s.path)
		if err != nil {
			s.persistFailed = true
			h.PersistFailed = true
		} else {
			h.DBBytes = n
		}
	}
	if s.minFreeBytes > 0 && s.diskFree != nil {
		free, err := s.diskFree(s.path)
		if err != nil {
			s.persistFailed = true
			h.PersistFailed = true
		} else {
			h.FreeBytes = free
		}
	}

	type breach struct {
		reason string
		hot    bool // at or over the cap
		warm   bool // not yet under the 80% resume line
	}
	var breaches []breach
	// Persist is first so a failed write is the reported reason even when
	// a disk stat also failed (§10.7).
	if h.PersistFailed {
		breaches = append(breaches, breach{ReasonPersistFail, true, true})
	}
	if h.UnackedBytes >= s.unackedMaxBytes {
		breaches = append(breaches, breach{ReasonOutboxBytes, true, true})
	} else if !under80(h.UnackedBytes, s.unackedMaxBytes) {
		breaches = append(breaches, breach{ReasonOutboxBytes, false, true})
	}
	if !h.OldestUnacked.IsZero() {
		age := s.now().Sub(h.OldestUnacked)
		if age < 0 {
			age = 0
		}
		if age >= s.unackedMaxAge {
			breaches = append(breaches, breach{ReasonOutboxAge, true, true})
		} else if age >= s.unackedMaxAge*resumeNum/resumeDen {
			breaches = append(breaches, breach{ReasonOutboxAge, false, true})
		}
	}
	if h.ArchiveBytes >= s.archiveMaxBytes {
		breaches = append(breaches, breach{ReasonArchiveBytes, true, true})
	} else if !under80(h.ArchiveBytes, s.archiveMaxBytes) {
		breaches = append(breaches, breach{ReasonArchiveBytes, false, true})
	}
	if s.minFreeBytes > 0 {
		resumeAt := s.minFreeBytes + s.minFreeBytes/4 // free must exceed the floor by 25% (the inverse of an 80% fill)
		if h.FreeBytes < s.minFreeBytes {
			breaches = append(breaches, breach{ReasonDiskSpace, true, true})
		} else if h.FreeBytes < resumeAt {
			breaches = append(breaches, breach{ReasonDiskSpace, false, true})
		}
	}

	var stopped int
	var reason string
	if err := s.db.QueryRowContext(ctx, `SELECT gate_stopped, gate_reason FROM meta WHERE id = 1`).Scan(&stopped, &reason); err != nil {
		return Health{}, err
	}
	hot := false
	warmReason := ""
	for _, b := range breaches {
		if b.hot {
			hot = true
			if reason == "" {
				reason = b.reason
			}
		}
		if b.warm && warmReason == "" {
			warmReason = b.reason
		}
	}
	// A new hot breach stops the gate. An already stopped gate stays stopped
	// while any metric is still warm, so it does not flap at the cap.
	if hot {
		stopped = 1
		if reason == "" {
			reason = breaches[0].reason
		}
		// Prefer a currently hot reason so the report matches the breach.
		for _, b := range breaches {
			if b.hot {
				reason = b.reason
				break
			}
		}
	} else if stopped == 1 {
		still := false
		for _, b := range breaches {
			if b.warm {
				still = true
				break
			}
		}
		if !still {
			stopped = 0
			reason = ""
		} else if warmReason != "" {
			reason = warmReason
		}
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE meta SET gate_stopped = ?, gate_reason = ? WHERE id = 1`, stopped, reason); err != nil {
		return Health{}, err
	}
	if stopped == 1 {
		h.ForwardingAllowed = false
		h.Reason = reason
		h.Detail = gateDetail(reason)
	}
	return h, nil
}

func gateDetail(reason string) string {
	switch reason {
	case ReasonOutboxAge:
		return "oldest unacknowledged event reached its age cap; member forwarding is stopped (§10.8)"
	case ReasonOutboxBytes:
		return "unacknowledged canonical payload reached its byte cap; member forwarding is stopped (§10.8)"
	case ReasonDiskSpace:
		return "filesystem free space is under the configured floor; member forwarding is stopped (§10.8)"
	case ReasonPersistFail:
		return "the usage ledger cannot be persisted; member forwarding is stopped (§10.7)"
	case ReasonArchiveBytes:
		return "the replay archive reached its own budget; member forwarding is stopped (§10.8)"
	default:
		return reason
	}
}

func under80(used, limit uint64) bool {
	if limit == 0 {
		return true
	}
	if used > (1<<64-1)/resumeDen {
		return false
	}
	return used*resumeDen < limit*resumeNum
}

// OutboxStatus is the contract view of the local stream (§10.5, §10.6, §10.8).
// It only reads. It does not evaluate the forwarding gate and it does not
// write gate_stopped. The agent run loop persists that flag through
// meter.Health, which calls EvaluateGate.
//
// LedgerEpoch is the central generation adopted by ApplyCheckpoint. It stays
// null until that verified checkpoint, including after batch ACKs have
// advanced CentralAckedThrough. A batch result names an epoch but is not a
// signed checkpoint, so it does not become the epoch this node is
// synchronized with (§17.3). Null means no verified generation has been
// accepted. The contiguous watermark can still be non-zero.
func (s *Store) OutboxStatus(ctx context.Context) (api.UsageOutboxStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, err := s.streamView(ctx)
	if err != nil {
		return api.UsageOutboxStatus{}, err
	}
	return statusFromHealth(h), nil
}

func statusFromHealth(h Health) api.UsageOutboxStatus {
	st := api.UsageOutboxStatus{
		AckedArchiveBytes:   decstr.U64(h.ArchiveBytes),
		BlockedSeq:          presence.NewNullNullable[decstr.U64](),
		CentralAckedThrough: decstr.U64(h.AckedThrough),
		LastDurableEventSeq: decstr.U64(h.LastSeq),
		// Null until ApplyCheckpoint. The epoch on a batch ACK is not stored.
		LedgerEpoch:         presence.NewNullNullable[uuid.UUID](),
		MeterEpochId:        presence.NewNullNullable[uuid.UUID](),
		OldestUnackedAt:     presence.NewNullNullable[time.Time](),
		UnackedEventCount:   h.UnackedCount,
		UnackedPayloadBytes: decstr.U64(h.UnackedBytes),
	}
	if h.BlockedSeq != nil {
		st.BlockedSeq = presence.NewNullableWithValue(decstr.U64(*h.BlockedSeq))
	}
	if h.LedgerEpoch != uuid.Nil {
		st.LedgerEpoch = presence.NewNullableWithValue(h.LedgerEpoch)
	}
	if h.MeterEpoch != uuid.Nil {
		st.MeterEpochId = presence.NewNullableWithValue(h.MeterEpoch)
	}
	if !h.OldestUnacked.IsZero() {
		st.OldestUnackedAt = presence.NewNullableWithValue(h.OldestUnacked.UTC())
	}
	return st
}
