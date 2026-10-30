package updatejournal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// PopulateCandidateHealth adds the contract 1.2.0 member to a pending rollback
// sealed before that member existed. All previous fields, including result ID,
// generation and timestamps, remain unchanged except the covering payload hash.
// The added evidence and hash commit together before delivery. Subsequent calls
// reuse those exact bytes, even if the transaction's diagnostic data changes.
// Rejected/conflicted results and acknowledged history are never rewritten.
func (j *Journal) PopulateCandidateHealth(ctx context.Context, resultID string) (OutboxEntry, error) {
	tx, err := j.db.BeginTx(ctx, nil)
	if err != nil {
		return OutboxEntry{}, err
	}
	defer func() { _ = tx.Rollback() }()
	e, err := scanOutbox(tx.QueryRowContext(ctx, outboxSelect+` WHERE result_id = ?`, resultID))
	if err != nil || e.Status != OutboxPending || e.State != "rolled_back" {
		return e, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(e.Payload, &fields); err != nil {
		return e, err
	}
	if _, present := fields["candidate_health"]; present {
		return e, nil
	}
	hash, err := canonjson.HashHex(e.Payload, "payload_hash")
	if err != nil {
		return e, err
	}
	if hash != e.PayloadHash {
		return e, ErrResultConflict
	}
	var releaseID string
	if err := json.Unmarshal(fields["requested_release_id"], &releaseID); err != nil {
		return e, err
	}
	var candidate sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT candidate_health_json FROM txns
		WHERE job_id = ? AND attempt = ? AND generation = ? AND release_id = ? AND stage = 'rolled_back'`,
		e.JobID, e.Attempt, e.Generation.String(), releaseID).Scan(&candidate)
	if errors.Is(err, sql.ErrNoRows) {
		return e, ErrNotFound
	}
	if err != nil {
		return e, err
	}
	fields["candidate_health"] = json.RawMessage("null")
	if candidate.Valid && candidate.String != "" {
		fields["candidate_health"] = json.RawMessage(candidate.String)
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		return e, err
	}
	hash, err = canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		return e, err
	}
	fields["payload_hash"], err = json.Marshal(hash)
	if err != nil {
		return e, err
	}
	raw, err = json.Marshal(fields)
	if err != nil {
		return e, err
	}
	payload, err := canonjson.Canonicalize(raw)
	if err != nil {
		return e, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE result_outbox SET payload = ?, payload_hash = ? WHERE result_id = ?`, payload, hash, resultID); err != nil {
		return e, err
	}
	if err := tx.Commit(); err != nil {
		return e, err
	}
	e.Payload, e.PayloadHash = payload, hash
	return e, nil
}
