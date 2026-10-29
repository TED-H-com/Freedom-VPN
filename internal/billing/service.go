package billing

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Pool         *pgxpool.Pool
	Provider     Provider
	Vault        *Vault
	Entitlements *entitlement.Store
	Config       Config
}

func NewService(pool *pgxpool.Pool, provider Provider, vault *Vault, engine *entitlement.Store, config Config) (*Service, error) {
	if pool == nil || provider == nil || !validExternal(provider.Name()) || vault == nil || engine == nil || (config.FullRefundPolicy != RevokeRefundedPeriod && config.FullRefundPolicy != KeepRefundedPeriod) || config.LeaseDuration < time.Second || config.LeaseDuration > 10*time.Minute || config.MaxWebhookBytes < 1024 || config.MaxWebhookBytes > 4<<20 || len(config.ReturnURLs) == 0 {
		return nil, ErrPayload
	}
	for _, raw := range config.ReturnURLs {
		u, e := url.Parse(raw)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || len(raw) > 2048 {
			return nil, ErrReturnURL
		}
	}
	return &Service{Pool: pool, Provider: provider, Vault: vault, Entitlements: engine, Config: config}, nil
}

type IngestResult struct {
	ID        uuid.UUID
	Duplicate bool
}

// Ingest returns success only after the authenticated, minimized payload is
// encrypted and durably committed. Map ErrSignature to contract 401 WEBHOOK_SIGNATURE_INVALID in pkg/apierr at the HTTP boundary.
func (s *Service) Ingest(ctx context.Context, raw []byte, signature string) (IngestResult, error) {
	var out IngestResult
	if len(raw) == 0 || len(raw) > s.Config.MaxWebhookBytes {
		return out, ErrPayload
	}
	if err := s.Provider.VerifyWebhook(raw, signature, s.Entitlements.Time()); err != nil {
		return out, ErrSignature
	}

	// Signature verification precedes minimization. Standard Stripe charge
	// notifications include card metadata without expansion; remove it before
	// encrypted persistence while retaining the digest of authenticated bytes.
	minimized, err := minimizePayload(raw)
	if err != nil {
		return out, err
	}
	event, err := s.Provider.ParseEvent(raw)
	if err != nil {
		return out, err
	}
	if !validExternal(event.ID) || !validExternal(event.Type) {
		return out, ErrPayload
	}
	return s.insertInbox(ctx, event, minimized, raw)
}

// minimizePayload retains the exact bytes if no sensitive fields occur. Card
// metadata is removed; unexpected credentials or full card inputs are rejected.
func minimizePayload(raw []byte) ([]byte, error) {
	var value any
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if dec.Decode(&value) != nil {
		return nil, ErrPayload
	}
	changed := false
	var scrub func(any) (any, error)
	scrub = func(v any) (any, error) {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				switch strings.ToLower(k) {
				case "card_number", "cvc", "cvv", "pan", "webhook_secret", "api_key", "authorization":
					if v != nil {
						return nil, ErrPayload
					}
				}
				child, err := scrub(v)
				if err != nil {
					return nil, err
				}
				x[k] = child
				switch strings.ToLower(k) {
				case "card", "payment_method_details", "last4", "fingerprint":
					if card, ok := v.(map[string]any); ok && card["number"] != nil {
						return nil, ErrPayload
					}
					delete(x, k)
					changed = true
				}
			}
			if x["object"] == "card" {
				if x["number"] != nil {
					return nil, ErrPayload
				}
				changed = true
				return nil, nil
			}
		case []any:
			for i, v := range x {
				child, err := scrub(v)
				if err != nil {
					return nil, err
				}
				x[i] = child
			}
		}
		return v, nil
	}
	if _, err := scrub(value); err != nil {
		return nil, err
	}
	if !changed {
		return raw, nil
	}
	minimized, err := json.Marshal(value)
	if err != nil {
		return nil, ErrPayload
	}
	return minimized, nil
}
func (s *Service) insertInbox(ctx context.Context, event Event, raw, digestSource []byte) (IngestResult, error) {
	out := IngestResult{}
	id, err := uuid.NewV7()
	if err != nil {
		return out, err
	}
	hash, err := payloadHash(digestSource)
	if err != nil {
		return out, ErrPayload
	}
	encrypted, err := s.Vault.Seal(raw, aad(s.Provider.Name(), event.ID))
	if err != nil {
		return out, err
	}
	err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		err = tx.QueryRow(ctx, `INSERT INTO inbox_events(id,source,external_event_id,payload_hash,encrypted_payload,subscription_ref,event_type) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(source,external_event_id) DO NOTHING RETURNING id`, id, s.Provider.Name(), event.ID, hash, encrypted, event.SubscriptionID, event.Type).Scan(&out.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			out.Duplicate = true
			err = tx.QueryRow(ctx, `SELECT id FROM inbox_events WHERE source=$1 AND external_event_id=$2 AND payload_hash=$3`, s.Provider.Name(), event.ID, hash).Scan(&out.ID)
			if errors.Is(err, pgx.ErrNoRows) {
				return db.ErrIdempotencyConflict
			}
		}
		if err != nil {
			return err
		}
		if out.Duplicate {
			return nil
		}
		payload, err := canonjson.Marshal(map[string]any{"inbox_id": out.ID})
		if err != nil {
			return err
		}
		return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: out.ID, AggregateType: "billing_inbox", AggregateVersion: "1", EventType: "billing.inbox_received", SchemaVersion: 1, Payload: payload})
	})
	return out, err
}

// BindSubscription is used only with provider-fetched current state. A new
// subscription must name a central checkout intent in server-owned metadata.
func (s *Service) bindSubscription(ctx context.Context, sub Subscription) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, `SELECT id FROM subscriptions WHERE provider=$1 AND external_subscription_id=$2`, s.Provider.Name(), sub.ID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return id, err
	}
	if sub.CheckoutID == uuid.Nil || !validExternal(sub.ID) || !validExternal(sub.CustomerID) {
		return id, ErrAnomaly
	}
	err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var member, plan uuid.UUID
		var price string
		err := tx.QueryRow(ctx, `SELECT s.member_id,s.plan_revision_id,p.provider_price_ref FROM billing_sessions s JOIN plan_revisions p ON p.id=s.plan_revision_id WHERE s.id=$1 AND s.provider=$2 AND s.kind='checkout'`, sub.CheckoutID, s.Provider.Name()).Scan(&member, &plan, &price)
		if err != nil {
			return ErrAnomaly
		}
		if price != sub.PriceID {
			return ErrAnomaly
		}
		var status string
		if err = tx.QueryRow(ctx, `SELECT status FROM members WHERE id=$1 FOR UPDATE`, member).Scan(&status); err != nil {
			return err
		}
		if status == "closed" {
			return ErrAnomaly
		}
		customerID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO billing_customers(id,member_id,provider,external_customer_id) VALUES($1,$2,$3,$4) ON CONFLICT(member_id,provider) DO NOTHING`, customerID, member, s.Provider.Name(), sub.CustomerID)
		if err != nil {
			return err
		}
		var customer string
		if err = tx.QueryRow(ctx, `SELECT external_customer_id FROM billing_customers WHERE member_id=$1 AND provider=$2`, member, s.Provider.Name()).Scan(&customer); err != nil {
			return err
		}
		if customer != sub.CustomerID {
			return ErrAnomaly
		}
		id, err = uuid.NewV7()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO subscriptions(id,member_id,provider,external_subscription_id,external_customer_id,plan_revision_id,provider_status,is_active) VALUES($1,$2,$3,$4,$5,$6,$7,true) ON CONFLICT(provider,external_subscription_id) DO NOTHING`, id, member, s.Provider.Name(), sub.ID, sub.CustomerID, plan, sub.Status)
		if err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT id FROM subscriptions WHERE provider=$1 AND external_subscription_id=$2`, s.Provider.Name(), sub.ID).Scan(&id)
	})
	return id, err
}
