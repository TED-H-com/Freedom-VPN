package usagestore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"modernc.org/sqlite"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// SnapshotBusyTimeout is how long a snapshot waits when SQLite is busy.
// The wait is bounded. It is not the node update lock.
const SnapshotBusyTimeout = time.Second

// sqlite result codes. Extended codes keep the primary code in the low byte.
const (
	sqliteBusy   = 5 // SQLITE_BUSY
	sqliteLocked = 6 // SQLITE_LOCKED
)

// Snapshot is one consistent read of the local usage ledger (§10.5, §10.6).
// Decimal fields are canonical decimal strings (§10.4). Sequences and byte
// totals never pass through float64.
//
// JSON object order is the field order below. A nil LedgerEpoch is JSON null.
// Empty slices are empty arrays. The lab name UsageLedgerSnapshot is this
// object, plus one trailing newline, from `fvpn-agent usage snapshot -json`.
type Snapshot struct {
	SchemaVersion       int            `json:"schema_version"`
	NodeID              string         `json:"node_id"`
	NodeIncarnationID   string         `json:"node_incarnation_id"`
	LedgerEpoch         *string        `json:"ledger_epoch"`
	MeterEpochs         []MeterEpoch   `json:"meter_epochs"`
	LastEventSeq        string         `json:"last_event_seq"`
	AckedThrough        string         `json:"acked_through"`
	UnackedCount        int            `json:"unacked_count"`
	UnackedPayloadBytes string         `json:"unacked_payload_bytes"`
	UnackedEvents       []UnackedEvent `json:"unacked_events"`
}

// MeterEpoch is one counter generation (§10.7). State is current or closed.
type MeterEpoch struct {
	MeterEpochID string `json:"meter_epoch_id"`
	State        string `json:"state"`
}

// UnackedEvent is one outbox row. PayloadSHA256 is the immutable hash of
// the stored canonical payload, the bytes the uploader resends (§10.5).
type UnackedEvent struct {
	EventSeq      string          `json:"event_seq"`
	PeriodIDs     []string        `json:"period_ids"`
	Records       []UnackedRecord `json:"records"`
	PayloadSHA256 string          `json:"payload_sha256"`
}

// UnackedRecord is one device row inside an unacknowledged event, copied
// from the stored payload.
type UnackedRecord struct {
	MemberID        string `json:"member_id"`
	DeviceID        string `json:"device_id"`
	BindingRevision string `json:"binding_revision"`
	PeriodID        string `json:"period_id"`
	UploadBytes     string `json:"upload_bytes"`
	DownloadBytes   string `json:"download_bytes"`
}

type storedPayload struct {
	NodeID            string         `json:"node_id"`
	NodeIncarnationID string         `json:"node_incarnation_id"`
	EventSeq          string         `json:"event_seq"`
	PayloadSHA256     string         `json:"payload_sha256"`
	Records           []storedRecord `json:"records"`
}

type storedRecord struct {
	MemberID        string `json:"member_id"`
	DeviceID        string `json:"device_id"`
	BindingRevision string `json:"binding_revision"`
	PeriodID        string `json:"period_id"`
	UploadBytes     string `json:"upload_bytes"`
	DownloadBytes   string `json:"download_bytes"`
}

// JSON encodes the snapshot as one compact object and a trailing newline.
func (s Snapshot) JSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Text is the operator table. It contains the same facts as JSON.
func (s Snapshot) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "schema_version: %d\n", s.SchemaVersion)
	fmt.Fprintf(&b, "node_id: %s\n", s.NodeID)
	fmt.Fprintf(&b, "node_incarnation_id: %s\n", s.NodeIncarnationID)
	epoch := "none"
	if s.LedgerEpoch != nil {
		epoch = *s.LedgerEpoch
	}
	fmt.Fprintf(&b, "ledger_epoch: %s\n", epoch)
	b.WriteString("meter_epochs:\n")
	if len(s.MeterEpochs) == 0 {
		b.WriteString("none\n")
	} else {
		for _, ep := range s.MeterEpochs {
			fmt.Fprintf(&b, "%s %s\n", ep.State, ep.MeterEpochID)
		}
	}
	fmt.Fprintf(&b, "last_event_seq: %s\n", s.LastEventSeq)
	fmt.Fprintf(&b, "acked_through: %s\n", s.AckedThrough)
	fmt.Fprintf(&b, "unacked_count: %d\n", s.UnackedCount)
	fmt.Fprintf(&b, "unacked_payload_bytes: %s\n", s.UnackedPayloadBytes)
	b.WriteString("unacked_events:\n")
	b.WriteString("event_seq period_id member_id device_id binding_revision upload_bytes download_bytes payload_sha256\n")
	for _, ev := range s.UnackedEvents {
		if len(ev.Records) == 0 {
			fmt.Fprintf(&b, "%s - - - - - - %s\n", ev.EventSeq, ev.PayloadSHA256)
			continue
		}
		for _, r := range ev.Records {
			fmt.Fprintf(&b, "%s %s %s %s %s %s %s %s\n",
				ev.EventSeq, r.PeriodID, r.MemberID, r.DeviceID, r.BindingRevision,
				r.UploadBytes, r.DownloadBytes, ev.PayloadSHA256)
		}
	}
	return b.String()
}

// ReadSnapshot reads path without creating it and without writing the ledger.
// The connection is mode=ro and query_only. One deferred read transaction
// covers the schema version, the cursor, the meter epochs and every
// unacknowledged payload. A commit of ApplyACK is therefore either fully
// before this view or fully after it (§10.6).
func ReadSnapshot(ctx context.Context, path string) (Snapshot, error) {
	return readSnapshot(ctx, path, SnapshotBusyTimeout)
}

func readSnapshot(ctx context.Context, path string, busy time.Duration) (Snapshot, error) {
	if path == "" {
		return Snapshot{}, ErrAbsent
	}
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Snapshot{}, ErrAbsent
		}
		return Snapshot{}, ErrCorrupt
	}
	if !st.Mode().IsRegular() {
		return Snapshot{}, ErrCorrupt
	}
	if busy <= 0 {
		busy = SnapshotBusyTimeout
	}
	ms := busy.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	dsn := "file:" + path + "?mode=ro&_pragma=busy_timeout(" + strconv.FormatInt(ms, 10) + ")&_pragma=query_only(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return Snapshot{}, ErrCorrupt
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	snap, err := readSnapshotDB(ctx, db)
	if err != nil {
		if errors.Is(err, ErrCorrupt) || errors.Is(err, ErrLocked) || errors.Is(err, ErrAbsent) {
			return Snapshot{}, err
		}
		if isSQLiteBusy(err) {
			return Snapshot{}, ErrLocked
		}
		return Snapshot{}, ErrCorrupt
	}
	return snap, nil
}

func isSQLiteBusy(err error) bool {
	var se *sqlite.Error
	if !errors.As(err, &se) {
		return false
	}
	code := se.Code() & 0xff
	return code == sqliteBusy || code == sqliteLocked
}

func readSnapshotDB(ctx context.Context, db *sql.DB) (Snapshot, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Snapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var userVersion int
	if err := tx.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&userVersion); err != nil {
		return Snapshot{}, err
	}
	if userVersion != schemaVersion {
		return Snapshot{}, ErrCorrupt
	}

	var nodeID, incID string
	var openEpoch, ledger sql.NullString
	var nextSeq, acked int64
	err = tx.QueryRowContext(ctx, `SELECT node_id, incarnation_id, next_seq, acked_through, open_epoch, ledger_epoch
		FROM meta WHERE id = 1`).Scan(&nodeID, &incID, &nextSeq, &acked, &openEpoch, &ledger)
	if err != nil {
		return Snapshot{}, err
	}
	if nextSeq < 1 || acked < 0 || acked > nextSeq-1 {
		return Snapshot{}, ErrCorrupt
	}
	if _, err := canonicalUUID(nodeID); err != nil {
		return Snapshot{}, ErrCorrupt
	}
	if _, err := canonicalUUID(incID); err != nil {
		return Snapshot{}, ErrCorrupt
	}
	var maxSeq int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) FROM events`).Scan(&maxSeq); err != nil {
		return Snapshot{}, err
	}
	if maxSeq < 0 || maxSeq >= nextSeq {
		return Snapshot{}, ErrCorrupt
	}

	epochs, err := readMeterEpochs(ctx, tx, openEpoch)
	if err != nil {
		return Snapshot{}, err
	}
	events, nbytes, err := readUnacked(ctx, tx, nodeID, incID, acked, nextSeq-1)
	if err != nil {
		return Snapshot{}, err
	}

	snap := Snapshot{
		SchemaVersion:       userVersion,
		NodeID:              nodeID,
		NodeIncarnationID:   incID,
		MeterEpochs:         epochs,
		LastEventSeq:        decstr.U64(uint64(nextSeq - 1)).String(),
		AckedThrough:        decstr.U64(uint64(acked)).String(),
		UnackedCount:        len(events),
		UnackedPayloadBytes: decstr.U64(nbytes).String(),
		UnackedEvents:       events,
	}
	if ledger.Valid && ledger.String != "" {
		if _, err := canonicalUUID(ledger.String); err != nil {
			return Snapshot{}, ErrCorrupt
		}
		v := ledger.String
		snap.LedgerEpoch = &v
	}
	if err := tx.Commit(); err != nil {
		return Snapshot{}, err
	}
	return snap, nil
}

func readMeterEpochs(ctx context.Context, tx *sql.Tx, open sql.NullString) ([]MeterEpoch, error) {
	rows, err := tx.QueryContext(ctx, `SELECT epoch_id, closed_at FROM epochs ORDER BY opened_at, epoch_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var openID string
	if open.Valid {
		openID = open.String
	}
	var out []MeterEpoch
	var sawCurrent bool
	for rows.Next() {
		var id string
		var closed sql.NullString
		if err := rows.Scan(&id, &closed); err != nil {
			return nil, err
		}
		if _, err := canonicalUUID(id); err != nil {
			return nil, ErrCorrupt
		}
		state := "closed"
		if !closed.Valid || closed.String == "" {
			if openID == "" || id != openID || sawCurrent {
				return nil, ErrCorrupt
			}
			state = "current"
			sawCurrent = true
		} else if id == openID {
			return nil, ErrCorrupt
		}
		out = append(out, MeterEpoch{MeterEpochID: id, State: state})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if openID != "" && !sawCurrent {
		return nil, ErrCorrupt
	}
	if out == nil {
		out = []MeterEpoch{}
	}
	return out, nil
}

func readUnacked(ctx context.Context, tx *sql.Tx, nodeID, incID string, acked, last int64) ([]UnackedEvent, uint64, error) {
	rows, err := tx.QueryContext(ctx, `SELECT seq, payload, payload_sha256, payload_len
		FROM events WHERE state = 'outbox' ORDER BY seq`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []UnackedEvent{}
	var total uint64
	for rows.Next() {
		var seq int64
		var payload []byte
		var columnHash string
		var payloadLen int64
		if err := rows.Scan(&seq, &payload, &columnHash, &payloadLen); err != nil {
			return nil, 0, err
		}
		if seq <= acked || seq > last || payloadLen != int64(len(payload)) || payloadLen < 0 {
			return nil, 0, ErrCorrupt
		}
		ev, err := unackedFromPayload(uint64(seq), payload, columnHash, nodeID, incID)
		if err != nil {
			return nil, 0, ErrCorrupt
		}
		n := uint64(len(payload))
		if total > ^uint64(0)-n {
			return nil, 0, ErrCorrupt
		}
		total += n
		out = append(out, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func unackedFromPayload(seq uint64, payload []byte, columnHash, nodeID, incID string) (UnackedEvent, error) {
	sum, err := canonjson.HashHex(payload, "payload_sha256")
	if err != nil || sum != columnHash || !hex64(sum) {
		return UnackedEvent{}, ErrCorrupt
	}
	var body storedPayload
	if err := json.Unmarshal(payload, &body); err != nil {
		return UnackedEvent{}, ErrCorrupt
	}
	if body.PayloadSHA256 != sum || body.NodeID != nodeID || body.NodeIncarnationID != incID {
		return UnackedEvent{}, ErrCorrupt
	}
	if body.EventSeq != decstr.U64(seq).String() {
		return UnackedEvent{}, ErrCorrupt
	}
	recs := make([]UnackedRecord, 0, len(body.Records))
	periods := []string{}
	seenPeriod := map[string]bool{}
	seenDevice := map[string]bool{}
	for _, r := range body.Records {
		if _, err := canonicalUUID(r.MemberID); err != nil {
			return UnackedEvent{}, ErrCorrupt
		}
		if _, err := canonicalUUID(r.DeviceID); err != nil {
			return UnackedEvent{}, ErrCorrupt
		}
		if seenDevice[r.DeviceID] {
			return UnackedEvent{}, ErrCorrupt
		}
		seenDevice[r.DeviceID] = true
		if err := canonicalDecimal(r.BindingRevision); err != nil || r.BindingRevision == "0" {
			return UnackedEvent{}, ErrCorrupt
		}
		if err := canonicalDecimal(r.UploadBytes); err != nil {
			return UnackedEvent{}, ErrCorrupt
		}
		if err := canonicalDecimal(r.DownloadBytes); err != nil {
			return UnackedEvent{}, ErrCorrupt
		}
		if r.PeriodID == "" || strings.ContainsAny(r.PeriodID, " \t\r\n") {
			return UnackedEvent{}, ErrCorrupt
		}
		recs = append(recs, UnackedRecord(r))
		if !seenPeriod[r.PeriodID] {
			seenPeriod[r.PeriodID] = true
			periods = append(periods, r.PeriodID)
		}
	}
	if periods == nil {
		periods = []string{}
	}
	return UnackedEvent{
		EventSeq: decstr.U64(seq).String(), PeriodIDs: periods, Records: recs, PayloadSHA256: sum,
	}, nil
}

func canonicalUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil || id.String() != s {
		return uuid.Nil, fmt.Errorf("usagestore: uuid")
	}
	return id, nil
}

func canonicalDecimal(s string) error {
	n, err := decstr.Parse(s)
	if err != nil || n.String() != s {
		return ErrCorrupt
	}
	return nil
}

func hex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
