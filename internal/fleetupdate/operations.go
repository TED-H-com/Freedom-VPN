package fleetupdate

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// §14.5: operation steps describe the persisted snapshot and actual target
// outcomes. A completed-with-exceptions batch never fabricates a successful step.
func projectOperation(ctx context.Context, tx pgx.Tx, id uuid.UUID, name string, state api.UpdateBatchState) error {
	var raw []byte
	var status api.OperationStatus
	var resource api.ResourceRef
	if err := tx.QueryRow(ctx, `SELECT status,steps,resource_type,resource_id FROM operations WHERE id=$1 FOR UPDATE`, id).Scan(&status, &raw, &resource.Type, &resource.Id); err != nil {
		return err
	}
	if status == "succeeded" || status == "superseded" {
		return nil
	}
	var steps map[string]operations.Step
	if err := json.Unmarshal(raw, &steps); err != nil {
		return err
	}
	evidence := &operations.Evidence{Reference: string(resource.Type) + ":" + resource.Id.String(), ObservedAt: time.Now().UTC()}
	if snapshot, ok := steps["snapshot"]; ok && snapshot.Status == "pending" {
		if err := operations.Transition(ctx, tx, id, "snapshot", "succeeded", evidence, nil, nil); err != nil {
			return err
		}
	}
	var next api.OperationStepStatus = "running"
	var code *api.ErrorCode
	switch state {
	case "completed":
		next = "succeeded"
	case "completed_with_exceptions":
		next = "failed"
		c := api.ErrorCode(apierr.StateConflict)
		code = &c
	}
	if steps[name].Status == next {
		return nil
	}
	return operations.Transition(ctx, tx, id, name, next, evidence, code, nil)
}
