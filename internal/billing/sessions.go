package billing

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SessionIntent struct {
	ID, OperationID uuid.UUID
	Replayed        bool
	CheckoutURL     string
	ExpiresAt       time.Time
}

func (s *Service) Checkout(ctx context.Context, c entitlement.Command, plan uuid.UUID, returnURL string) (SessionIntent, error) {
	if c.Tx != nil {
		return SessionIntent{}, entitlement.ErrInvalid
	}
	intent, err := s.ReserveCheckout(ctx, c, plan, returnURL)
	if err != nil {
		return intent, err
	}
	hosted, err := s.CompleteSession(ctx, c.Actor, intent.ID)
	if err == nil {
		intent.CheckoutURL = hosted.URL
		intent.ExpiresAt = hosted.ExpiresAt
	}
	return intent, err
}

// ReserveCheckout persists intent in the HTTP transaction; its visitor must
// call CompleteSession after commit before emitting the synchronous 201 (D-031).
func (s *Service) ReserveCheckout(ctx context.Context, c entitlement.Command, plan uuid.UUID, returnURL string) (SessionIntent, error) {
	return s.prepareSession(ctx, c, "checkout", &plan, returnURL)
}
func (s *Service) Portal(ctx context.Context, c entitlement.Command, returnURL string) (SessionIntent, error) {
	return s.prepareSession(ctx, c, "portal", nil, returnURL)
}
func (s *Service) prepareSession(ctx context.Context, c entitlement.Command, kind string, plan *uuid.UUID, returnURL string) (SessionIntent, error) {
	var out SessionIntent
	if c.Actor.Role != "member" || c.Actor.ID == uuid.Nil {
		return out, entitlement.ErrForbidden
	}
	// Exact URLs, not suffix/prefix matching: queries and ports cannot redirect
	// a trusted origin to an attacker-controlled destination (§16.2).
	if !slices.Contains(s.Config.ReturnURLs, returnURL) {
		return out, ErrReturnURL
	}
	err := c.InTx(ctx, s.Pool, func(tx pgx.Tx) error {
		record, err := c.Begin(ctx, tx, "POST", "/member/"+kind+"-sessions", map[string]any{"plan_id": plan, "return_url": returnURL})
		if err != nil {
			return err
		}
		if !record.New {
			out = SessionIntent{ID: *record.ResourceID, Replayed: true}
			if record.OperationID != nil {
				out.OperationID = *record.OperationID
			}
			return nil
		}
		var status string
		if err = tx.QueryRow(ctx, `SELECT status FROM members WHERE id=$1 FOR UPDATE`, c.Actor.ID).Scan(&status); err != nil {
			return db.MapError(err)
		}
		if status == "closed" {
			return entitlement.ErrConflict
		}
		var revision *uuid.UUID
		if kind == "checkout" {
			var id uuid.UUID
			var price string
			err = tx.QueryRow(ctx, `SELECT r.id,r.provider_price_ref FROM plans p JOIN plan_revisions r ON r.id=p.active_revision_id WHERE p.id=$1 AND r.provider=$2 AND r.provider_price_ref IS NOT NULL`, plan, s.Provider.Name()).Scan(&id, &price)
			if err != nil {
				return db.MapError(err)
			}
			if s.Config.AllowedPriceIDs != nil && !slices.Contains(s.Config.AllowedPriceIDs, price) {
				return ErrPayload
			}
			revision = &id
		} else {
			var customer string
			if err = tx.QueryRow(ctx, `SELECT external_customer_id FROM billing_customers WHERE member_id=$1 AND provider=$2`, c.Actor.ID, s.Provider.Name()).Scan(&customer); err != nil {
				return db.MapError(err)
			}
		}
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}

		var op *uuid.UUID
		if kind == "checkout" {
			operationKind, resourceType := api.OperationKindCheckoutCreate, api.ResourceTypeCheckoutSession
			if !operationKind.Valid() || !resourceType.Valid() {
				return entitlement.ErrInvalid
			}
			operationID, e := uuid.NewV7()
			if e != nil {
				return e
			}
			op = &operationID
			_, err = tx.Exec(ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id,request_id,steps) VALUES($1,$5,$2,$6,$3,$4,'{"provider":{"required":true,"status":"pending"}}')`, op, c.Actor.Role+":"+c.Actor.ID.String(), id, c.Actor.RequestID, operationKind, resourceType)
			if err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO billing_sessions(id,member_id,provider,kind,plan_revision_id,operation_id,return_url) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, c.Actor.ID, s.Provider.Name(), kind, revision, op, returnURL)
		if err != nil {
			return err
		}

		if err = entitlement.Audit(ctx, tx, c.Actor, "billing."+kind, api.ResourceTypeCheckoutSession, id, nil, map[string]any{"session_id": id}); err != nil {
			return err
		}
		out = SessionIntent{ID: id}
		if op != nil {
			out.OperationID = *op
		}

		return entitlement.Complete(ctx, tx, record, api.ResourceTypeCheckoutSession, id, op, 201)
	})
	return out, err
}

// FulfillSession is safe to retry with the same provider idempotency key and
// never modifies entitlement. It performs no external calls inside a transaction.
func (s *Service) FulfillSession(ctx context.Context, id uuid.UUID) (retErr error) {
	token, err := uuid.NewV7()
	if err != nil {
		return err
	}
	var member uuid.UUID
	var operation *uuid.UUID
	var plan *uuid.UUID
	var kind, returnURL string
	var created time.Time
	var cached []byte
	err = s.Pool.QueryRow(ctx, `UPDATE billing_sessions SET fulfillment_state=CASE WHEN encrypted_response IS NULL THEN 'pending' ELSE 'succeeded' END,lease_token=$2,lease_until=clock_timestamp()+$3::bigint*interval '1 millisecond' WHERE id=$1 AND provider=$4 AND (lease_until IS NULL OR lease_until<=clock_timestamp()) RETURNING member_id,operation_id,plan_revision_id,kind,return_url,created_at,encrypted_response`, id, token, s.Config.LeaseDuration.Milliseconds(), s.Provider.Name()).Scan(&member, &operation, &plan, &kind, &returnURL, &created, &cached)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBusy
	}
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = db.InTx(cleanup, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			tag, e := tx.Exec(cleanup, `UPDATE billing_sessions SET lease_token=NULL,lease_until=NULL,fulfillment_state=CASE WHEN $3 AND encrypted_response IS NULL THEN 'failed' ELSE fulfillment_state END WHERE id=$1 AND lease_token=$2`, id, token, retErr != nil)
			if e != nil || tag.RowsAffected() == 0 || retErr == nil || operation == nil {
				return e
			}
			var eligible bool
			if e = tx.QueryRow(cleanup, `SELECT status IN ('pending','running') AND steps->'provider'->>'status' IN ('pending','running') FROM operations WHERE id=$1 FOR UPDATE`, operation).Scan(&eligible); e != nil {
				return e
			}
			if eligible {
				return entitlement.TransitionOperation(cleanup, tx, *operation, "provider", api.OperationStepStatusFailed, nil)
			}
			return nil
		})
	}()
	if len(cached) > 0 {
		return nil
	}
	// Stripe may prune keys after 24h. Never blindly recreate an ambiguous old
	// request after provider deduplication can expire (§14.5).
	if s.Entitlements.Time().After(created.Add(23 * time.Hour)) {
		return ErrAnomaly
	}
	if operation != nil {
		err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
			var status api.OperationStepStatus
			if e := tx.QueryRow(ctx, `SELECT steps->'provider'->>'status' FROM operations WHERE id=$1 FOR UPDATE`, operation).Scan(&status); e != nil {
				return e
			}
			if status == api.OperationStepStatusPending || status == api.OperationStepStatusFailed {
				return entitlement.TransitionOperation(ctx, tx, *operation, "provider", api.OperationStepStatusRunning, nil)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	var customer string
	err = s.Pool.QueryRow(ctx, `SELECT external_customer_id FROM billing_customers WHERE member_id=$1 AND provider=$2`, member, s.Provider.Name()).Scan(&customer)
	if err != nil && (!errors.Is(err, pgx.ErrNoRows) || kind == "portal") {
		return db.MapError(err)
	}
	var hosted HostedSession
	if kind == "checkout" {
		var price string
		if plan == nil {
			return ErrPayload
		}
		if err = s.Pool.QueryRow(ctx, `SELECT provider_price_ref FROM plan_revisions WHERE id=$1 AND provider=$2`, plan, s.Provider.Name()).Scan(&price); err != nil {
			return err
		}
		hosted, err = s.Provider.CreateCheckout(ctx, CheckoutRequest{ID: id, MemberID: member, PlanRevisionID: *plan, CustomerID: customer, PriceID: price, ReturnURL: returnURL, ExpiresAt: created.Add(23 * time.Hour)})
	} else {
		hosted, err = s.Provider.CreatePortal(ctx, PortalRequest{ID: id, CustomerID: customer, ReturnURL: returnURL})
	}
	if err != nil {
		return ErrProvider
	}
	u, err := url.Parse(hosted.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || !validExternal(hosted.ID) || !hosted.ExpiresAt.After(s.Entitlements.Time()) {
		return ErrProvider
	}
	hosted.ExpiresAt = hosted.ExpiresAt.UTC()
	raw, err := json.Marshal(hosted)
	if err != nil {
		return err
	}
	encrypted, err := s.Vault.Seal(raw, aad("session", id.String()))
	if err != nil {
		return err
	}
	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE billing_sessions SET fulfillment_state='succeeded',external_session_id=$3,encrypted_response=$4,expires_at=$5 WHERE id=$1 AND lease_token=$2 AND lease_until>clock_timestamp()`, id, token, hosted.ID, encrypted, hosted.ExpiresAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return db.ErrLeaseLost
		}
		// Record completion only after the provider result is durably stored.
		// Cached replays return above, so each session has one completion audit.
		actor := entitlement.Actor{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Role: string(api.AuditEventActorTypeSystem)}
		if err = entitlement.Audit(ctx, tx, actor, "billing."+kind+"_completed", api.ResourceTypeCheckoutSession, id, nil, map[string]any{"session_id": id, "state": "succeeded", "expires_at": hosted.ExpiresAt}); err != nil {
			return err
		}
		if operation == nil {
			return nil
		}
		// The encrypted persisted provider-owned session is the evidence; URLs
		// never appear in operations or audit. Provider idempotency recovers timeouts.
		return entitlement.TransitionOperation(ctx, tx, *operation, "provider", api.OperationStepStatusSucceeded, &operations.Evidence{Reference: string(api.ResourceTypeCheckoutSession) + ":" + id.String(), ObservedAt: s.Entitlements.Time()})
	})
}

// SessionResult is scoped to the authenticated member. HTTP adapters must use
// Cache-Control: no-store for this secret-bearing hosted URL (§14.3, §16.2).
func (s *Service) SessionResult(ctx context.Context, actor entitlement.Actor, id uuid.UUID) (HostedSession, error) {
	var out HostedSession
	if actor.Role != "member" || actor.ID == uuid.Nil {
		return out, db.ErrNotFound
	}
	var encrypted []byte
	var kind string
	err := s.Pool.QueryRow(ctx, `SELECT encrypted_response,kind FROM billing_sessions WHERE id=$1 AND member_id=$2 AND provider=$3`, id, actor.ID, s.Provider.Name()).Scan(&encrypted, &kind)
	if err != nil {
		return out, db.MapError(err)
	}
	if len(encrypted) == 0 {
		return out, ErrBusy
	}
	raw, err := s.Vault.Open(encrypted, aad("session", id.String()))
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(raw, &out); err != nil {
		return out, ErrCipher
	}
	if kind != "checkout" && !out.ExpiresAt.After(s.Entitlements.Time()) {
		return HostedSession{}, entitlement.ErrConflict
	}
	out.ExpiresAt = out.ExpiresAt.UTC()
	return out, nil
}

// CreateBillingPortalSession completes the provider effect before returning the
// short-lived URL (the contract's 201). Retries recover the durable session.
func (s *Service) CreateBillingPortalSession(ctx context.Context, c entitlement.Command, returnURL string) (HostedSession, error) {
	intent, err := s.Portal(ctx, c, returnURL)
	if err != nil {
		return HostedSession{}, err
	}
	if err = s.FulfillSession(ctx, intent.ID); err != nil {
		return HostedSession{}, err
	}
	return s.SessionResult(ctx, c.Actor, intent.ID)
}

// CompleteSession rechecks ownership before external I/O. Concurrent replays
// wait outside transactions for the winning provider call, bounded by context
// and a short deadline. Failed reservations retry with the SAME session UUID.
func (s *Service) CompleteSession(ctx context.Context, actor entitlement.Actor, id uuid.UUID) (HostedSession, error) {
	deadline := time.Now().Add(10 * time.Second)
	for {
		hosted, err := s.SessionResult(ctx, actor, id)
		if !errors.Is(err, ErrBusy) {
			return hosted, err
		}
		err = s.FulfillSession(ctx, id)
		if err == nil {
			return s.SessionResult(ctx, actor, id)
		}
		if !errors.Is(err, ErrBusy) || time.Now().After(deadline) {
			return HostedSession{}, err
		}
		timer := time.NewTimer(20 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return HostedSession{}, ctx.Err()
		case <-timer.C:
		}
	}
}
