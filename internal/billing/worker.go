package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ProcessOne claims an inbox row durably, fetches current provider state outside
// all transactions, and fences completion. Expired claims recover after crashes.
func (s *Service) ProcessOne(ctx context.Context) (bool, error) {
	token, err := uuid.NewV7()
	if err != nil {
		return false, err
	}
	var id uuid.UUID
	var external, kind string
	var encrypted []byte
	err = s.Pool.QueryRow(ctx, `WITH candidate AS (SELECT id FROM inbox_events WHERE source=$1 AND processing_status<>'succeeded' AND (lease_until IS NULL OR lease_until<=clock_timestamp()) ORDER BY received_at,id FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE inbox_events e SET processing_status='processing',attempt=attempt+1,lease_token=$2,lease_until=clock_timestamp()+$3::bigint*interval '1 millisecond' FROM candidate c WHERE e.id=c.id RETURNING e.id,e.external_event_id,e.event_type,e.encrypted_payload`, s.Provider.Name(), token, s.Config.LeaseDuration.Milliseconds()).Scan(&id, &external, &kind, &encrypted)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	raw, err := s.Vault.Open(encrypted, aad(s.Provider.Name(), external))
	var event Event
	if err == nil {
		if kind == "reconciliation" {
			err = json.Unmarshal(raw, &event)
		} else {
			event, err = s.Provider.ParseEvent(raw)
		}
	}
	if err == nil {
		err = s.processEvent(ctx, id, event)
	}
	if err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Keep a bounded backoff so a poison event cannot starve the inbox. Do not
		// persist provider error bodies, URLs, secrets or SQL in failure diagnostics.
		_, _ = s.Pool.Exec(cleanup, `UPDATE inbox_events SET processing_status='failed',last_error_code='BILLING_SYNC_FAILED',lease_token=NULL,lease_until=clock_timestamp()+interval '30 seconds' WHERE id=$1 AND lease_token=$2`, id, token)
		return true, err
	}
	tag, err := s.Pool.Exec(ctx, `UPDATE inbox_events SET processing_status='succeeded',processed_at=clock_timestamp(),last_error_code=NULL,lease_token=NULL,lease_until=NULL WHERE id=$1 AND lease_token=$2 AND lease_until>clock_timestamp()`, id, token)
	if err != nil {
		return true, err
	}
	if tag.RowsAffected() != 1 {
		return true, db.ErrLeaseLost
	}
	return true, nil
}
func (s *Service) processEvent(ctx context.Context, inbox uuid.UUID, event Event) error {
	if event.SubscriptionID != "" {
		var id uuid.UUID
		err := s.Pool.QueryRow(ctx, `SELECT id FROM subscriptions WHERE provider=$1 AND external_subscription_id=$2`, s.Provider.Name(), event.SubscriptionID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			sub, e := s.Provider.FetchSubscription(ctx, event.SubscriptionID)
			if e != nil {
				return e
			}
			if sub.ID != event.SubscriptionID {
				return ErrAnomaly
			}
			id, err = s.bindSubscription(ctx, sub)
		}
		if err != nil {
			return err
		}
		return s.syncSubscription(ctx, id, inbox)
	}
	if event.CustomerID != "" {
		rows, err := s.Pool.Query(ctx, `SELECT id FROM subscriptions WHERE provider=$1 AND external_customer_id=$2 ORDER BY id`, s.Provider.Name(), event.CustomerID)
		if err != nil {
			return err
		}
		var ids []uuid.UUID
		for rows.Next() {
			var id uuid.UUID
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return ErrAnomaly
		}
		for _, id := range ids {
			if err = s.syncSubscription(ctx, id, inbox); err != nil {
				return err
			}
		}
		return nil
	}
	// Authenticated irrelevant events are durable acknowledgments, never grants.
	return nil
}

type snapshot struct {
	ProviderStatus string               `json:"provider_status"`
	ServicePeriods []entitlement.Period `json:"service_periods"`
	InvoiceRefs    []string             `json:"invoice_refs"`
	RefundPolicy   RefundPolicy         `json:"refund_policy"`
}

func (s *Service) syncSubscription(ctx context.Context, id, inbox uuid.UUID) error {
	token, err := uuid.NewV7()
	if err != nil {
		return err
	}
	var external, customer, price string
	var member uuid.UUID
	err = s.Pool.QueryRow(ctx, `UPDATE subscriptions s SET sync_token=$2,sync_until=clock_timestamp()+$3::bigint*interval '1 millisecond' FROM plan_revisions p WHERE s.id=$1 AND s.provider=$4 AND p.id=s.plan_revision_id AND (s.sync_until IS NULL OR s.sync_until<=clock_timestamp()) RETURNING s.external_subscription_id,s.external_customer_id,s.member_id,p.provider_price_ref`, id, token, s.Config.LeaseDuration.Milliseconds(), s.Provider.Name()).Scan(&external, &customer, &member, &price)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBusy
	}
	if err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = s.Pool.Exec(cleanup, `UPDATE subscriptions SET sync_token=NULL,sync_until=NULL WHERE id=$1 AND sync_token=$2`, id, token)
	}()
	sub, err := s.Provider.FetchSubscription(ctx, external)
	if err != nil {
		return err
	}
	if sub.ID != external || sub.CustomerID != customer || sub.PriceID != price {
		return s.anomaly(ctx, id, "PROVIDER_MAPPING_MISMATCH", ErrAnomaly)
	}
	invoices, err := s.Provider.FetchInvoices(ctx, sub)
	if err != nil {
		return err
	}
	if err = validateInvoices(invoices, sub); err != nil {
		return s.anomaly(ctx, id, "INVALID_INVOICE_STATE", err)
	}
	err = db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var memberStatus string
		if err := tx.QueryRow(ctx, `SELECT status FROM members WHERE id=$1 FOR UPDATE`, member).Scan(&memberStatus); err != nil {
			return err
		}
		var valid bool
		var previousEnd *time.Time
		if err := tx.QueryRow(ctx, `SELECT coalesce(sync_token=$2 AND sync_until>clock_timestamp(),false),paid_through FROM subscriptions WHERE id=$1 FOR UPDATE`, id, token).Scan(&valid, &previousEnd); err != nil {
			return err
		}
		if !valid {
			return db.ErrLeaseLost
		}
		// Append-only invoice/refund uniqueness is permanent, independent of inbox
		// delivery IDs or the generic seven-day command retention (§9.3, AT03).
		for _, invoice := range invoices {
			if invoice.Status == "paid" {
				if err := s.record(ctx, tx, member, id, inbox, "paid_invoice", invoice.ID, invoice.ID, invoice.AmountMinor, invoice.Currency, invoice.Period); err != nil {
					return err
				}
			}
			for _, refund := range invoice.Refunds {
				if refund.Status == "succeeded" {
					if err := s.record(ctx, tx, member, id, inbox, "refund", refund.ID, invoice.ID, refund.AmountMinor, refund.Currency, invoice.Period); err != nil {
						return err
					}
				}
			}
		}
		// Rebuild from recorded paid evidence, not event order. Provider failures or
		// incomplete recent invoice lists cannot erase a known newer paid period.
		rows, err := tx.Query(ctx, `SELECT external_reference,amount_minor,currency,service_start,service_end FROM billing_records WHERE subscription_id=$1 AND record_type='paid_invoice' ORDER BY service_start,external_reference`, id)
		if err != nil {
			return err
		}
		var recorded []Invoice
		for rows.Next() {
			var inv Invoice
			if err = rows.Scan(&inv.ID, &inv.AmountMinor, &inv.Currency, &inv.Period.Start, &inv.Period.End); err != nil {
				rows.Close()
				return err
			}
			recorded = append(recorded, inv)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		snap := snapshot{ProviderStatus: sub.Status, ServicePeriods: []entitlement.Period{}, InvoiceRefs: []string{}, RefundPolicy: s.Config.FullRefundPolicy}
		var paidFrom, paidThrough *time.Time
		var anomalies []string
		for _, inv := range recorded {
			if paidFrom == nil || inv.Period.Start.Before(*paidFrom) {
				v := inv.Period.Start
				paidFrom = &v
			}
			if paidThrough == nil || inv.Period.End.After(*paidThrough) {
				v := inv.Period.End
				paidThrough = &v
			}
			var refunded int64
			if err = tx.QueryRow(ctx, `SELECT coalesce(sum(amount_minor),0)::bigint FROM billing_records WHERE subscription_id=$1 AND record_type='refund' AND original_invoice_ref=$2`, id, inv.ID).Scan(&refunded); err != nil {
				return err
			}
			snap.InvoiceRefs = append(snap.InvoiceRefs, inv.ID)
			if refunded > inv.AmountMinor {
				anomalies = append(anomalies, "REFUND_EXCEEDS_INVOICE")
			}
			if s.Config.FullRefundPolicy == RevokeRefundedPeriod && inv.AmountMinor > 0 && refunded >= inv.AmountMinor {
				continue
			}
			snap.ServicePeriods = append(snap.ServicePeriods, inv.Period)
		}
		if previousEnd != nil && (paidThrough == nil || previousEnd.After(*paidThrough)) {
			anomalies = append(anomalies, "PAID_FACT_WITHOUT_LEDGER_EVIDENCE")
		}
		var graceFrom, graceUntil *time.Time
		var graceReason *string
		// Grace is anchored to the paid boundary, never to webhook arrival or a
		// reconciliation clock. Full-refunded periods cannot create new grace.
		if sub.Status == "past_due" && !sub.CancelAtPeriodEnd && paidThrough != nil && s.Entitlements.Config.GraceDuration > 0 {
			covered := false
			for _, p := range snap.ServicePeriods {
				if p.End.Equal(*paidThrough) {
					covered = true
				}
			}
			if covered {
				graceFrom = paidThrough
				end := paidThrough.Add(s.Entitlements.Config.GraceDuration)
				graceUntil = &end
				v := s.Entitlements.Config.GraceReason
				graceReason = &v
			}
		}
		raw, err := canonjson.Marshal(snap)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE subscriptions SET provider_status=$2,paid_from=$3,paid_through=$4,cancel_at_period_end=$5,grace_from=$6,grace_until=$7,grace_reason=$8,provider_snapshot=$9,last_reconciled_at=$10,reconciliation_anomalies=$11,resource_version=resource_version+1,is_active=$12 WHERE id=$1`, id, sub.Status, paidFrom, paidThrough, sub.CancelAtPeriodEnd, graceFrom, graceUntil, graceReason, raw, s.Entitlements.Time(), append([]string{}, anomalies...), sub.Status != "canceled" && sub.Status != "incomplete_expired" || hasFutureService(snap.ServicePeriods, graceUntil, s.Entitlements.Time()))
		if err != nil {
			return err
		}
		projection, err := s.Entitlements.Recompute(ctx, tx, member)
		if err != nil {
			return err
		}
		return entitlement.Audit(ctx, tx, entitlement.Actor{ID: member, Role: "system"}, "billing.reconciled", member, nil, map[string]any{"subscription_id": id, "inbox_event_id": inbox, "entitlement_revision": projection.Revision, "anomalies": append([]string{}, anomalies...)})
	})
	if errors.Is(err, ErrAnomaly) {
		return s.anomaly(ctx, id, "IMMUTABLE_BILLING_FACT_CONFLICT", err)
	}
	return err
}
func (s *Service) record(ctx context.Context, tx pgx.Tx, member, sub, event uuid.UUID, kind, ref, invoice string, amount int64, currency string, period entitlement.Period) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `INSERT INTO billing_records(id,member_id,subscription_id,provider,record_type,external_reference,amount_minor,currency,service_start,service_end,source_event_id,original_invoice_ref) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT(provider,record_type,external_reference) DO NOTHING`, id, member, sub, s.Provider.Name(), kind, ref, amount, currency, period.Start, period.End, event, invoice)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var same bool
		err = tx.QueryRow(ctx, `SELECT member_id=$4 AND subscription_id=$5 AND amount_minor=$6 AND currency=$7 AND service_start=$8 AND service_end=$9 AND original_invoice_ref=$10 FROM billing_records WHERE provider=$1 AND record_type=$2 AND external_reference=$3`, s.Provider.Name(), kind, ref, member, sub, amount, currency, period.Start, period.End, invoice).Scan(&same)
		if err != nil {
			return err
		}
		if !same {
			return ErrAnomaly
		}
	}
	return nil
}
func validateInvoices(invoices []Invoice, sub Subscription) error {
	if !slices.Contains([]string{"incomplete", "incomplete_expired", "trialing", "active", "past_due", "canceled", "unpaid", "paused"}, sub.Status) {
		return ErrPayload
	}
	seen := map[string]bool{}
	refunds := map[string]bool{}
	for _, i := range invoices {
		if !validExternal(i.ID) || seen[i.ID] || i.SubscriptionID != sub.ID || i.CustomerID != sub.CustomerID || i.AmountMinor < 0 || i.AmountMinor > canonjson.MaxSafeInteger || !entitlement.ValidPeriod(i.Period.Start, i.Period.End, 0) {
			return ErrPayload
		}
		seen[i.ID] = true
		if _, err := CurrencyDigits(i.Currency); err != nil {
			return err
		}
		for _, r := range i.Refunds {
			if !validExternal(r.ID) || refunds[r.ID] || r.AmountMinor < 0 || r.AmountMinor > canonjson.MaxSafeInteger || r.Currency != i.Currency {
				return ErrPayload
			}
			refunds[r.ID] = true
		}
	}
	return nil
}
func (s *Service) anomaly(ctx context.Context, id uuid.UUID, code string, cause error) error {
	_, err := s.Pool.Exec(ctx, `UPDATE subscriptions SET reconciliation_anomalies=ARRAY[$2::text] WHERE id=$1`, id, code)
	if err != nil {
		return err
	}
	return cause
}

// Reconcile schedules durable synthetic inbox events for every known subscription,
// including canceled subscriptions whose refunds may arrive later. A scheduler
// supplies a stable run ID so retrying the same run creates no duplicate work.
func (s *Service) Reconcile(ctx context.Context, run uuid.UUID, after uuid.UUID, limit int) (uuid.UUID, int, error) {
	if run == uuid.Nil || limit < 1 || limit > 1000 {
		return after, 0, ErrPayload
	}
	rows, err := s.Pool.Query(ctx, `SELECT id,external_subscription_id FROM subscriptions WHERE provider=$1 AND id>$2 ORDER BY id LIMIT $3`, s.Provider.Name(), after, limit)
	if err != nil {
		return after, 0, err
	}
	type item struct {
		id       uuid.UUID
		external string
	}
	var items []item
	for rows.Next() {
		var v item
		if err = rows.Scan(&v.id, &v.external); err != nil {
			rows.Close()
			return after, 0, err
		}
		items = append(items, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return after, 0, err
	}
	for n, v := range items {
		event := Event{ID: fmt.Sprintf("reconcile:%s:%s", run, v.id), Type: "reconciliation", SubscriptionID: v.external}
		raw, e := canonjson.Marshal(event)
		if e != nil {
			return after, n, e
		}
		if _, e = s.insertInbox(ctx, event, raw); e != nil {
			return after, n, e
		}
		after = v.id
	}
	return after, len(items), nil
}

func hasFutureService(periods []entitlement.Period, graceUntil *time.Time, now time.Time) bool {
	if graceUntil != nil && graceUntil.After(now) {
		return true
	}
	for _, period := range periods {
		if period.End.After(now) {
			return true
		}
	}
	return false
}
