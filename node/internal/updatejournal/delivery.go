package updatejournal

import (
	"context"
	"errors"
	"time"
)

// DeliveryFailure is bounded, non-secret evidence from one delivery attempt.
// APICode contains only a validated API error code, never a response message.
type DeliveryFailure struct {
	Status         OutboxStatus
	HTTPStatus     int
	APICode        string
	ErrorCode      string
	RetryNotBefore time.Time
}

// RecordDeliveryFailure retains a rejected result without blocking subsequent
// results, or records the next permitted retry for a pending result (D-062d).
func (j *Journal) RecordDeliveryFailure(ctx context.Context, resultID string, f DeliveryFailure) error {
	if f.Status != OutboxPending && f.Status != OutboxRejected && f.Status != OutboxConflict {
		return errors.New("updatejournal: invalid delivery status")
	}
	var retryAt any
	if !f.RetryNotBefore.IsZero() {
		retryAt = f.RetryNotBefore.UTC().Format(time.RFC3339Nano)
	}
	_, err := j.db.ExecContext(ctx, `UPDATE result_outbox SET status = ?, send_attempts = send_attempts + 1,
		last_attempt_at = ?, last_error = ?, last_http_status = ?, last_api_code = ?, retry_not_before = ? WHERE result_id = ?`,
		string(f.Status), j.ts(), f.ErrorCode, f.HTTPStatus, nullStr(f.APICode), retryAt, resultID)
	return err
}
