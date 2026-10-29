package entitlement

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrForbidden            = errors.New("forbidden")
	ErrPreconditionRequired = errors.New("resource version required")
	ErrConflict             = errors.New("state conflict")
)

// Actor must be constructed from the authenticated central identity, never from
// body fields. The HTTP framework owns session/MFA/CSRF enforcement (§16.1).
type Actor struct {
	ID        uuid.UUID
	Role      string
	RequestID uuid.UUID
}
type Command struct {
	Actor Actor
	Key   uuid.UUID
	// Tx is supplied by the HTTP command boundary. Standalone callers leave it nil.
	Tx pgx.Tx
}

func (a Actor) Admin() error {
	if a.Role != "admin" || a.ID == uuid.Nil {
		return ErrForbidden
	}
	return nil
}
func (a Actor) Own(id uuid.UUID) error {
	if a.ID == uuid.Nil || a.Role != "admin" && (a.Role != "member" || a.ID != id) {
		return db.ErrNotFound
	}
	return nil
}

// InTx reuses the caller's transaction; it never commits a framework command.
func (c Command) InTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	if c.Tx != nil {
		return fn(c.Tx)
	}
	return db.InTx(ctx, pool, db.TxOptions{}, fn)
}
func (c Command) Begin(ctx context.Context, tx pgx.Tx, method, route string, payload any) (db.IdempotencyRecord, error) {
	if c.Key == uuid.Nil || c.Actor.ID == uuid.Nil || c.Actor.RequestID == uuid.Nil {
		return db.IdempotencyRecord{}, ErrInvalid
	}
	if c.Tx != nil {
		return db.IdempotencyRecord{New: true}, nil
	}
	raw, err := canonjson.Marshal(payload)
	if err != nil {
		return db.IdempotencyRecord{}, ErrInvalid
	}
	hash, err := canonjson.Hash(raw)
	if err != nil {
		return db.IdempotencyRecord{}, ErrInvalid
	}
	key := sha256.Sum256(c.Key[:])
	return db.BeginIdempotent(ctx, tx, db.IdempotencyScope{Principal: c.Actor.Role + ":" + c.Actor.ID.String(), Method: method, Route: route, KeyHash: key[:], RequestHash: hash[:]}, 0)
}
func Complete(ctx context.Context, tx pgx.Tx, record db.IdempotencyRecord, resourceType api.ResourceType, id uuid.UUID, operation *uuid.UUID, status int) error {
	if !resourceType.Valid() || (status == 202 && operation == nil) {
		return ErrInvalid
	}
	// HTTP owns reservation and completion through SetResult (§14.5).
	if record.ID == uuid.Nil {
		return nil
	}
	return db.CompleteIdempotent(ctx, tx, record.ID, db.IdempotencyResult{ResourceType: string(resourceType), ResourceID: &id, OperationID: operation, ResponseStatus: status})
}
func Audit(ctx context.Context, tx pgx.Tx, actor Actor, action string, resource api.ResourceType, id uuid.UUID, before, after any) error {
	if !resource.Valid() || !api.AuditEventActorType(actor.Role).Valid() || id == uuid.Nil || actor.ID == uuid.Nil {
		return ErrInvalid
	}
	if before == nil {
		before = map[string]any{}
	}
	if after == nil {
		after = map[string]any{}
	}
	b, err := canonjson.Marshal(before)
	if err != nil {
		return err
	}
	a, err := canonjson.Marshal(after)
	if err != nil {
		return err
	}
	var req *uuid.UUID
	if actor.RequestID != uuid.Nil {
		req = &actor.RequestID
	}
	return db.AppendAudit(ctx, tx, db.AuditEvent{Actor: actor.ID.String(), ActorType: actor.Role, Action: action, Resource: string(resource) + ":" + id.String(), RequestID: req, SchemaVersion: 1, Before: b, After: a, Result: string(api.AuditEventResultSucceeded)})
}
func Operation(ctx context.Context, tx pgx.Tx, actor Actor, kind api.OperationKind, resource api.ResourceType, id uuid.UUID, revision decstr.U64) (uuid.UUID, error) {
	if !kind.Valid() || !resource.Valid() || id == uuid.Nil || actor.ID == uuid.Nil || actor.RequestID == uuid.Nil || (actor.Role != string(auth.Member) && actor.Role != string(auth.Admin)) {
		return uuid.Nil, ErrInvalid
	}
	op, err := uuid.NewV7()
	if err != nil {
		return op, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id,desired_generation,request_id,steps) VALUES($1,$2,$3,$4,$5,$6::numeric,$7,'{"netbird":{"required":true,"status":"pending"},"nodes":{"required":true,"status":"pending"}}')`, op, kind, (auth.Principal{Role: auth.Role(actor.Role), ID: actor.ID}).Scope(), resource, id, revision.String(), actor.RequestID)
	return op, err
}
func Reason(s string, max int) bool { return strings.TrimSpace(s) != "" && len([]rune(s)) <= max }

// ResourceType translates internal aggregate names to the binding HTTP contract.
// Outbox aggregate names remain the documented domain event vocabulary.
func ResourceType(kind string) string {
	switch kind {
	case "device":
		return string(api.ResourceTypeMemberDevice)
	case "grant":
		return string(api.ResourceTypeEntitlementGrant)
	case "billing_session":
		return string(api.ResourceTypeCheckoutSession)
	default:
		return kind
	}
}
