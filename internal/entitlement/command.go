package entitlement

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
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

// Audit admits only reviewed domain fields. Free-text member metadata, external
// identifiers and provider bodies are excluded (D-029). Revision 14 explicitly
// requires the bounded admin grant-withdrawal reason; other reasons are codes.
func Audit(ctx context.Context, tx pgx.Tx, actor Actor, action string, resource api.ResourceType, id uuid.UUID, before, after map[string]any, operation ...*uuid.UUID) error {
	return AuditResult(ctx, tx, actor, action, resource, id, api.AuditEventResultSucceeded, before, after, operation...)
}
func AuditResult(ctx context.Context, tx pgx.Tx, actor Actor, action string, resource api.ResourceType, id uuid.UUID, result api.AuditEventResult, before, after map[string]any, operation ...*uuid.UUID) error {
	var fields []string
	switch action {
	case "member.patch":
		fields = []string{"admin_suspended", "resource_version", "member_group_ids", "display_name_changed", "notes_changed"}
	case "member.close":
		fields = []string{"state"}
	case "member.grant":
		fields = []string{"grant_id", "source", "valid_until"}
	case "grant.revoke":
		fields = []string{"grant_id"}
	case "member.grant_revoke":
		fields = []string{"revoked_at", "member_id", "grant_id", "reason"}
	case "member.entitlement_changed":
		fields = []string{"revision", "state"}
	case "device.authorize", "device.revoke":
		fields = []string{"member_id", "status", "reason"}
	case "device.authorization.external_withdrawal":
		// peer_id and observed_groups name the withdrawn peer. "authorization"
		// in a field name would be redacted, so it stays in the action only.
		fields = []string{"member_id", "peer_id", "observed_groups", "reason"}
	case "identity_binding.created", "identity_binding.proof_updated", "identity_binding.rejected":
		fields = []string{"identity_binding_id", "revision", "reason"}
	case "billing.checkout", "billing.portal", "billing.checkout_completed", "billing.portal_completed":
		fields = []string{"session_id", "state", "expires_at"}
	case "billing.reconciled":
		fields = []string{"subscription_id", "inbox_event_id", "entitlement_revision", "anomalies"}
	default:
		return ErrInvalid
	}
	var req, op *uuid.UUID
	if actor.RequestID != uuid.Nil {
		req = &actor.RequestID
	}
	if len(operation) > 1 {
		return ErrInvalid
	}
	if len(operation) == 1 {
		op = operation[0]
	}
	event, err := audit.NewEvent(audit.Entry{Actor: audit.Actor{Type: api.AuditEventActorType(actor.Role), ID: actor.ID}, Action: action, Resource: api.ResourceRef{Type: resource, Id: id}, Result: result, OperationID: op, RequestID: req, Before: before, After: after, Fields: fields})
	if err != nil {
		return err
	}
	return audit.AppendEvent(ctx, tx, event)
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
func ResourceType(aggregate string) (api.ResourceType, error) {
	resource := api.ResourceType(aggregate)
	switch aggregate {
	case "device":
		resource = api.ResourceTypeMemberDevice
	case "grant":
		resource = api.ResourceTypeEntitlementGrant
	case "billing_session":
		resource = api.ResourceTypeCheckoutSession
	}
	if !resource.Valid() {
		return "", ErrInvalid
	}
	return resource, nil
}
