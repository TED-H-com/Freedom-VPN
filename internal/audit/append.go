package audit

import (
	"context"
	"errors"
	"regexp"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Actor is an authenticated principal or a UUID identifying a system process.
// An agent is represented by the contract actor type node, never agent/service.
type Actor struct {
	Type api.AuditEventActorType
	ID   uuid.UUID
}

// Entry describes a contract-conformant audit write (§16.4, D-029). Fields is the
// allowlist for Diff; callers must not include unbounded user/provider text.
type Entry struct {
	Actor                  Actor
	Action                 string
	Resource               api.ResourceRef
	Result                 api.AuditEventResult
	OperationID, RequestID *uuid.UUID
	Before, After          map[string]any
	Fields                 []string
}

// Event can only be populated by NewEvent, before starting any database write.
type Event struct{ row db.AuditEvent }

func NewEvent(e Entry) (Event, error) {
	if !e.Actor.Type.Valid() || e.Actor.ID == uuid.Nil || !e.Resource.Type.Valid() || e.Resource.Id == uuid.Nil || !e.Result.Valid() || len(e.Action) > 128 || !regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`).MatchString(e.Action) {
		return Event{}, errors.New("invalid audit entry")
	}
	for _, id := range []*uuid.UUID{e.OperationID, e.RequestID} {
		if id != nil && *id == uuid.Nil {
			return Event{}, errors.New("invalid audit correlation ID")
		}
	}
	before, after, err := Diff(e.Before, e.After, e.Fields...)
	if err != nil {
		return Event{}, err
	}
	// Copy optional IDs so mutation of the caller's input cannot invalidate Event.
	clone := func(id *uuid.UUID) *uuid.UUID {
		if id == nil {
			return nil
		}
		v := *id
		return &v
	}
	return Event{row: db.AuditEvent{Actor: e.Actor.ID.String(), ActorType: string(e.Actor.Type), Action: e.Action, Resource: string(e.Resource.Type) + ":" + e.Resource.Id.String(), Result: string(e.Result), OperationID: clone(e.OperationID), RequestID: clone(e.RequestID), Before: before, After: after, SchemaVersion: 1}}, nil
}

// AppendEvent shares the transaction of the action; a zero Event is rejected.
func AppendEvent(ctx context.Context, tx pgx.Tx, e Event) error {
	if e.row.SchemaVersion != 1 {
		return errors.New("unconstructed audit event")
	}
	return db.AppendAudit(ctx, tx, e.row)
}
