package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"regexp"
	"strings"
)

// CurrencyDigits is an explicit supported-currency table (§14.1). Unknown codes
// fail closed; expanding this table requires reviewing provider currency rules.
func CurrencyDigits(currency string) (int, error) {
	switch currency {
	case "JPY", "KRW", "VND", "CLP", "XOF", "XAF":
		return 0, nil
	case "BHD", "JOD", "KWD", "OMR", "TND":
		return 3, nil
	case "USD", "TWD", "EUR", "GBP", "CAD", "AUD", "NZD", "CHF", "HKD", "SGD", "CNY", "SEK", "NOK", "DKK", "INR", "BRL", "MXN", "THB", "MYR", "IDR", "PHP", "PLN", "CZK", "ILS", "ZAR":
		return 2, nil
	default:
		return 0, ErrPayload
	}
}
func FormatAmount(amount int64, currency string) (string, error) {
	digits, err := CurrencyDigits(currency)
	if err != nil || amount < 0 {
		return "", ErrPayload
	}
	scale := int64(1)
	for range digits {
		scale *= 10
	}
	if digits == 0 {
		return fmt.Sprintf("%s %d", currency, amount), nil
	}
	return fmt.Sprintf("%s %d.%0*d", currency, amount/scale, digits, amount%scale), nil
}

type PlanRevision struct {
	ID, PlanID uuid.UUID
	Revision   decstr.U64
	Payload    api.PlanPayload
}

// CreatePlanRevision is called by the policy store in its transaction, with the
// same definition/revision IDs as the generic plan policy. It never activates
// or edits existing subscribers. Default device limit 3 is available via
// DefaultPlanPayload; HTTP schema still requires an explicit value.
func DefaultPlanPayload() api.PlanPayload {
	return api.PlanPayload{DeviceLimit: entitlement.DefaultDeviceLimit}
}
func CreatePlanRevision(ctx context.Context, tx pgx.Tx, planID, revisionID uuid.UUID, code string, p api.PlanPayload) (PlanRevision, error) {
	out := PlanRevision{ID: revisionID, PlanID: planID, Payload: p}
	if planID == uuid.Nil || revisionID == uuid.Nil || !regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`).MatchString(code) || !entitlement.Reason(p.DisplayName.ZhTW, 200) || p.DeviceLimit < 1 || p.DeviceLimit > 100 || len(p.ExitGroupIds) < 1 || len(p.ExitGroupIds) > 100 || p.FairUseRevisionId == uuid.Nil || p.DnsProfileRevisionId == uuid.Nil || p.Price.AmountMinor < 0 || p.Price.AmountMinor > canonjson.MaxSafeInteger || !p.Price.BillingInterval.Valid() || !regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`).MatchString(p.Price.Provider) || !entitlement.Reason(p.Price.ProviderPriceRef, 255) {
		return out, ErrPayload
	}
	if _, err := CurrencyDigits(p.Price.Currency); err != nil {
		return out, err
	}
	if p.RateLimit != nil && (p.RateLimit.DownBps < 0 || p.RateLimit.UpBps < 0) {
		return out, ErrPayload
	}
	groups := map[uuid.UUID]bool{}
	for _, id := range p.ExitGroupIds {
		if id == uuid.Nil || groups[id] {
			return out, ErrPayload
		}
		groups[id] = true
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM member_groups WHERE id=$1)`, id).Scan(&exists); err != nil {
			return out, err
		}
		if !exists {
			return out, ErrPayload
		}
	}
	for id, typ := range map[uuid.UUID]string{p.FairUseRevisionId: "fair_use", p.DnsProfileRevisionId: "dns_profile"} {
		var actual string
		if err := tx.QueryRow(ctx, `SELECT type FROM policy_revisions WHERE id=$1`, id).Scan(&actual); err != nil {
			return out, db.MapError(err)
		}
		if actual != typ {
			return out, ErrPayload
		}
	}
	_, err := tx.Exec(ctx, `INSERT INTO plans(id,code) VALUES($1,$2) ON CONFLICT(id) DO NOTHING`, planID, code)
	if err != nil {
		return out, err
	}
	var storedCode string
	if err = tx.QueryRow(ctx, `SELECT code FROM plans WHERE id=$1 FOR UPDATE`, planID).Scan(&storedCode); err != nil {
		return out, err
	}
	if storedCode != code {
		return out, entitlement.ErrConflict
	}
	if err = tx.QueryRow(ctx, `SELECT (coalesce(max(revision),0)+1)::text FROM plan_revisions WHERE plan_id=$1`, planID).Scan(&out.Revision); err != nil {
		return out, err
	}
	raw, err := canonjson.Marshal(p)
	if err != nil {
		return out, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,fup_policy_revision_id,provider,provider_price_ref,schema_version,payload) VALUES($1,$2,$3::numeric,$4,$5,$6,$7,$8,1,$9)`, revisionID, planID, out.Revision.String(), p.DeviceLimit, p.ExitGroupIds, p.FairUseRevisionId, p.Price.Provider, p.Price.ProviderPriceRef, raw)
	return out, db.MapError(err)
}
func ActivatePlanRevision(ctx context.Context, tx pgx.Tx, planID, revisionID uuid.UUID, expected decstr.U64) error {
	var owner uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT plan_id FROM plan_revisions WHERE id=$1`, revisionID).Scan(&owner); err != nil {
		return db.MapError(err)
	}
	if owner != planID {
		return ErrPayload
	}
	_, err := db.UpdateWithVersion(ctx, tx, "plans", planID, expected.String(), map[string]any{"active_revision_id": revisionID})
	return err
}
func validExternal(s string) bool {
	return s != "" && len(s) <= 255 && !strings.ContainsAny(s, "\x00\r\n")
}

// ReadPlan returns the immutable revision referenced by subscriptions or grants.
func ReadPlan(ctx context.Context, tx pgx.Tx, revisionID uuid.UUID) (PlanRevision, error) {
	var out PlanRevision
	var raw []byte
	err := tx.QueryRow(ctx, `SELECT id,plan_id,revision::text,payload FROM plan_revisions WHERE id=$1`, revisionID).Scan(&out.ID, &out.PlanID, &out.Revision, &raw)
	if err != nil {
		return out, db.MapError(err)
	}
	if err = json.Unmarshal(raw, &out.Payload); err != nil {
		return out, ErrPayload
	}
	return out, nil
}

// ListPlans returns offered active revisions in stable definition order. The
// caller may expose only these known plan IDs to checkout.
func ListPlans(ctx context.Context, tx pgx.Tx, after uuid.UUID, limit int) ([]PlanRevision, error) {
	if limit < 1 || limit > 100 {
		return nil, ErrPayload
	}
	rows, err := tx.Query(ctx, `SELECT r.id,r.plan_id,r.revision::text,r.payload FROM plans p JOIN plan_revisions r ON r.id=p.active_revision_id WHERE p.id>$1 ORDER BY p.id LIMIT $2`, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PlanRevision{}
	for rows.Next() {
		var v PlanRevision
		var raw []byte
		if err = rows.Scan(&v.ID, &v.PlanID, &v.Revision, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &v.Payload); err != nil {
			return nil, ErrPayload
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
