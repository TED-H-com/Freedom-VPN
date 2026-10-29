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
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type SessionIntent struct {
	ID, OperationID uuid.UUID
	Replayed        bool
}

func (s *Service) Checkout(ctx context.Context, c entitlement.Command, plan uuid.UUID, returnURL string) (SessionIntent, error) {
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
			operationID, e := uuid.NewV7()
			if e != nil {
				return e
			}
			op = &operationID
			_, err = tx.Exec(ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id,request_id,steps) VALUES($1,'checkout.create',$2,'billing_session',$3,$4,'{"provider":{"required":true,"status":"pending"}}')`, op, c.Actor.Role+":"+c.Actor.ID.String(), id, c.Actor.RequestID)
			if err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO billing_sessions(id,member_id,provider,kind,plan_revision_id,operation_id,return_url) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, c.Actor.ID, s.Provider.Name(), kind, revision, op, returnURL)
		if err != nil {
			return err
		}

		raw, err := canonjson.Marshal(map[string]any{"session_id": id, "member_id": c.Actor.ID, "operation_id": op})
		if err != nil {
			return err
		}
		if err = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: id, AggregateType: "billing_session", AggregateVersion: "1", EventType: "billing.session_requested", SchemaVersion: 1, Payload: raw}); err != nil {
			return err
		}
		if err = entitlement.Audit(ctx, tx, c.Actor, "billing."+kind, c.Actor.ID, nil, map[string]any{"session_id": id}); err != nil {
			return err
		}
		out = SessionIntent{ID: id}
		if op != nil {
			out.OperationID = *op
		}
		statusCode := 202
		if kind == "portal" {
			statusCode = 201
		}
		return entitlement.Complete(ctx, tx, record, "billing_session", id, op, statusCode)
	})
	return out, err
}

// FulfillSession is safe to retry with the same provider idempotency key and
// never modifies entitlement. It performs no external calls inside a transaction.
func (s *Service) FulfillSession(ctx context.Context, id uuid.UUID) error {
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
	err = s.Pool.QueryRow(ctx, `UPDATE billing_sessions SET lease_token=$2,lease_until=clock_timestamp()+$3::bigint*interval '1 millisecond' WHERE id=$1 AND provider=$4 AND (lease_until IS NULL OR lease_until<=clock_timestamp()) RETURNING member_id,operation_id,plan_revision_id,kind,return_url,created_at,encrypted_response`, id, token, s.Config.LeaseDuration.Milliseconds(), s.Provider.Name()).Scan(&member, &operation, &plan, &kind, &returnURL, &created, &cached)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBusy
	}
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = s.Pool.Exec(cleanup, `UPDATE billing_sessions SET lease_token=NULL,lease_until=NULL WHERE id=$1 AND lease_token=$2`, id, token)
	}()
	if len(cached) > 0 {
		return nil
	}
	// Stripe may prune keys after 24h. Never blindly recreate an ambiguous old
	// request after provider deduplication can expire (§14.5).
	if s.Entitlements.Time().After(created.Add(23 * time.Hour)) {
		return ErrAnomaly
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
		return err
	}
	u, err := url.Parse(hosted.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || !validExternal(hosted.ID) || !hosted.ExpiresAt.After(s.Entitlements.Time()) {
		return ErrPayload
	}
	raw, err := json.Marshal(hosted)
	if err != nil {
		return err
	}
	encrypted, err := s.Vault.Seal(raw, aad("session", id.String()))
	if err != nil {
		return err
	}
	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE billing_sessions SET external_session_id=$3,encrypted_response=$4,expires_at=$5 WHERE id=$1 AND lease_token=$2 AND lease_until>clock_timestamp()`, id, token, hosted.ID, encrypted, hosted.ExpiresAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return db.ErrLeaseLost
		}
		if operation == nil {
			return nil
		}
		// The encrypted persisted provider-owned session is the evidence; URLs
		// never appear in operations or audit. Provider idempotency recovers timeouts.
		return operations.Transition(ctx, tx, *operation, "provider", api.OperationStepStatusSucceeded, &operations.Evidence{Reference: "billing_session:" + id.String(), ObservedAt: s.Entitlements.Time()}, nil, nil)
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
	err := s.Pool.QueryRow(ctx, `SELECT encrypted_response FROM billing_sessions WHERE id=$1 AND member_id=$2 AND provider=$3`, id, actor.ID, s.Provider.Name()).Scan(&encrypted)
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
	if !out.ExpiresAt.After(s.Entitlements.Time()) {
		return HostedSession{}, entitlement.ErrConflict
	}
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
