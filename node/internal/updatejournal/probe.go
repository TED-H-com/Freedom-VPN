package updatejournal

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/google/uuid"
)

const probeOmissionPrefix = "probe_omission/"

// ProbeOmission is durable local evidence for an issuance. It contains no
// network addresses. Standalone revocation recovery uses a dedicated TxnID
// without an apply transaction row, so the evidence also lives in meta.
type ProbeOmission struct {
	Kind         string                     `json:"kind"`
	TxnID        string                     `json:"txn_id"`
	Nonce        string                     `json:"nonce"`
	ReleaseID    string                     `json:"release_id"`
	Phase        string                     `json:"phase"`
	IssuedAt     time.Time                  `json:"issued_at"`
	ProbeOmitted healthattest.ProbeOmission `json:"probe_omitted"`
}

// RecordProbeOmission commits evidence before the caller publishes a challenge.
// The meta record and apply transaction event (when present) commit together.
// Repeating the same issuance is idempotent; conflicting evidence is refused.
// New meta keys do not change the SQLite schema version.
func (j *Journal) RecordProbeOmission(ctx context.Context, ch healthattest.Challenge, phase string) error {
	nonce, err := hex.DecodeString(ch.Nonce)
	if err != nil || len(nonce) != 32 || uuid.Validate(ch.TxnID) != nil || uuid.Validate(ch.ReleaseID) != nil || ch.IssuedAt.IsZero() ||
		(phase != "candidate" && phase != "rollback") || ch.Probe != nil || ch.ProbeOmitted == nil || !ch.ProbeOmitted.Valid() {
		return errors.New("updatejournal: invalid probe omission")
	}
	r := ProbeOmission{Kind: "probe_omitted", TxnID: ch.TxnID, Nonce: ch.Nonce, ReleaseID: ch.ReleaseID,
		Phase: phase, IssuedAt: ch.IssuedAt, ProbeOmitted: *ch.ProbeOmitted}
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	key := probeOmissionPrefix + ch.TxnID + "/" + ch.Nonce
	var prior string
	err = tx.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&prior)
	if err == nil {
		if prior != string(b) {
			return errors.New("updatejournal: conflicting probe omission")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err := setMeta(ctx, tx, key, string(b)); err != nil {
		return err
	}
	if err := j.recordEventTx(ctx, tx, ch.TxnID, string(b)); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	return tx.Commit()
}

// ProbeOmissions returns all recorded omissions for this transaction, including
// standalone recovery challenges. Records survive sidecar consumption/restart.
func (j *Journal) ProbeOmissions(ctx context.Context, txnID string) ([]ProbeOmission, error) {
	prefix := probeOmissionPrefix + txnID + "/"
	rows, err := j.db.QueryContext(ctx, `SELECT value FROM meta WHERE substr(key, 1, length(?)) = ? ORDER BY key`, prefix, prefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProbeOmission
	for rows.Next() {
		var raw string
		var r ProbeOmission
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &r); err != nil {
			return nil, errors.New("updatejournal: invalid stored probe omission")
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
