// Package operation records DNS domain read-back in policy activation operations.
package operation

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const DocumentStep = "dns_document"
const FeedStep = "dns_feed_refresh"

// Record serializes with other domain transitions, permits retry after failure,
// and never declares success without a persisted row reference read back by the
// caller. Step names follow OperationStep.name's open pattern, not a new enum.
func Record(ctx context.Context, tx pgx.Tx, id uuid.UUID, step, reference string, code *api.ErrorCode, at time.Time) error {
	if id == uuid.Nil {
		return nil
	}
	var status string
	var raw []byte
	if err := tx.QueryRow(ctx, `SELECT status,steps FROM operations WHERE id=$1 FOR UPDATE`, id).Scan(&status, &raw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if status == "succeeded" || status == "superseded" {
		return nil
	}
	var steps map[string]operations.Step
	if err := json.Unmarshal(raw, &steps); err != nil {
		return err
	}
	current, exists := steps[step]
	if !exists || current.Status == "succeeded" || current.Status == "cancelled" {
		return nil
	}
	if current.Status == "failed" {
		if code != nil {
			return nil
		}
		if err := operations.Transition(ctx, tx, id, step, api.OperationStepStatusRunning, nil, nil, nil); err != nil {
			return err
		}
	}
	if code != nil {
		return operations.Transition(ctx, tx, id, step, api.OperationStepStatusFailed, nil, code, nil)
	}
	return operations.Transition(ctx, tx, id, step, api.OperationStepStatusSucceeded, &operations.Evidence{Reference: reference, ObservedAt: at}, nil, nil)
}
