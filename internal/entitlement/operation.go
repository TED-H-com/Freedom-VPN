package entitlement

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TransitionOperation keeps both reader-visible step errors and top-level error
// fields consistent (D-029). Messages are fixed safe text, never provider errors.
func TransitionOperation(ctx context.Context, tx pgx.Tx, id uuid.UUID, step string, status api.OperationStepStatus, evidence *operations.Evidence) error {
	var code *api.ErrorCode
	if status == api.OperationStepStatusFailed {
		c := api.ErrorCodeDEPENDENCYUNAVAILABLE
		code = &c
	}
	if err := operations.Transition(ctx, tx, id, step, status, evidence, code, nil); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE operations SET status=CASE WHEN $2 THEN CASE WHEN EXISTS(SELECT 1 FROM jsonb_each(steps) WHERE value->>'status'='succeeded' AND value->>'required'='true') THEN 'partially_succeeded' ELSE 'failed' END ELSE status END,completed_at=CASE WHEN $2 THEN clock_timestamp() ELSE completed_at END,error_code=(SELECT value->>'error_code' FROM jsonb_each(steps) WHERE value->>'status'='failed' ORDER BY key LIMIT 1),error_message=CASE WHEN EXISTS(SELECT 1 FROM jsonb_each(steps) WHERE value->>'status'='failed') THEN 'External dependency is unavailable.' ELSE NULL END WHERE id=$1 AND status NOT IN ('succeeded','superseded')`, id, status == api.OperationStepStatusFailed)
	return err
}
