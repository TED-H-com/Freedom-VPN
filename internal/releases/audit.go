package releases

import (
	"context"
	"errors"
	"regexp"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Audit validates contract identities before using control-core's redacting
// writer (§14.5, §16.5). The allowlist deliberately excludes free-form reasons.
func Audit(ctx context.Context, tx pgx.Tx, actor uuid.UUID, actorType api.AuditEventActorType, action string, resource api.ResourceRef, op *uuid.UUID, result api.AuditEventResult, after map[string]any) error {
	if !actorType.Valid() || actor == uuid.Nil && actorType != api.AuditEventActorTypeSystem || !resource.Type.Valid() || resource.Id == uuid.Nil || !result.Valid() || len(action) > 128 || !regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`).MatchString(action) {
		return errors.New("invalid fleet audit event")
	}
	requestID := commandtx.RequestID(ctx)
	if op != nil {
		var original *uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT request_id FROM operations WHERE id=$1`, *op).Scan(&original); err != nil {
			return err
		}
		if original != nil {
			requestID = *original
		}
	}
	identity := "system"
	if actor != uuid.Nil {
		identity = actor.String()
	}
	return audit.Append(ctx, tx, db.AuditEvent{Actor: identity, ActorType: string(actorType), Action: action, Resource: string(resource.Type) + ":" + resource.Id.String(), OperationID: op, RequestID: &requestID, Result: string(result)}, nil, after,
		"verification_state", "error_code", "node_id", "release_id", "target_count", "state", "accepted_as_history", "result_id", "job_id", "conflicting_hash", "recovery_required", "approved", "serving", "revoked")
}
