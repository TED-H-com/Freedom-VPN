package db

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AuditEvent contains already-redacted diffs. This layer validates shape only;
// the domain caller must omit tokens, keys, certificates and browsing data.
type AuditEvent struct {
	ID                                         uuid.UUID
	Actor, ActorType, Action, Resource, Result string
	RequestID, OperationID                     *uuid.UUID
	SchemaVersion                              int
	Before, After                              json.RawMessage
}

func jsonObject(b []byte) bool {
	var obj map[string]json.RawMessage
	return json.Unmarshal(b, &obj) == nil && obj != nil
}

// AppendAudit atomically records an action alongside its local state change.
func AppendAudit(ctx context.Context, tx pgx.Tx, evt AuditEvent) error {
	if !jsonObject(evt.Before) || !jsonObject(evt.After) || evt.SchemaVersion < 1 {
		return errors.New("audit diffs must be versioned JSON objects")
	}
	if evt.ID == uuid.Nil {
		var err error
		evt.ID, err = uuid.NewV7()
		if err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `INSERT INTO audit_events(id,actor,actor_type,action,resource,result,request_id,operation_id,schema_version,before_diff,after_diff)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, evt.ID, evt.Actor, evt.ActorType, evt.Action, evt.Resource, evt.Result, evt.RequestID, evt.OperationID, evt.SchemaVersion, evt.Before, evt.After)
	return MapError(err)
}
