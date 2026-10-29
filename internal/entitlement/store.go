package entitlement

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Config Config
	Now    func() time.Time
}

func NewStore(config Config) (*Store, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Store{Config: config, Now: time.Now}, nil
}
func (s *Store) Time() time.Time {
	if s.Now != nil {
		return s.Now().UTC().Truncate(time.Microsecond)
	}
	return time.Now().UTC().Truncate(time.Microsecond)
}
func LoadPlan(ctx context.Context, tx pgx.Tx, id uuid.UUID) (Plan, error) {
	var p Plan
	err := tx.QueryRow(ctx, `SELECT id,device_limit,exit_group_ids FROM plan_revisions WHERE id=$1`, id).Scan(&p.ID, &p.DeviceLimit, &p.ExitGroups)
	return p, db.MapError(err)
}

// Recompute locks the member before reading any entitlement source. Every writer
// of a source must use the same member lock (§15.5.2–3). The caller commits.
func (s *Store) Recompute(ctx context.Context, tx pgx.Tx, memberID uuid.UUID) (Projection, error) {
	var out Projection
	in := Input{Now: s.Time(), Config: s.Config}
	err := tx.QueryRow(ctx, `SELECT status,admin_suspended,member_group_ids FROM members WHERE id=$1 FOR UPDATE`, memberID).Scan(&in.MemberStatus, &in.AdminSuspended, &in.MemberGroups)
	if err != nil {
		return out, db.MapError(err)
	}
	var sub Subscription
	var planID uuid.UUID
	var snapshot []byte
	err = tx.QueryRow(ctx, `SELECT id,plan_revision_id,provider_status,paid_from,paid_through,cancel_at_period_end,grace_from,grace_until,coalesce(grace_reason,''),provider_snapshot FROM subscriptions WHERE member_id=$1 AND is_primary ORDER BY is_active DESC,created_at DESC,id DESC LIMIT 1`, memberID).Scan(&sub.ID, &planID, &sub.ProviderStatus, &sub.PaidFrom, &sub.PaidThrough, &sub.CancelAtPeriodEnd, &sub.GraceFrom, &sub.GraceUntil, &sub.GraceReason, &snapshot)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if err == nil {
		p, e := LoadPlan(ctx, tx, planID)
		if e != nil {
			return out, e
		}
		in.Plan = &p
		var data struct {
			ServicePeriods []Period `json:"service_periods"`
		}
		if e = json.Unmarshal(snapshot, &data); e != nil {
			return out, ErrInvalid
		}
		// Only a worker-validated snapshot supplies service intervals. A provider's
		// status string alone is never payment evidence (§9.3).
		sub.ServicePeriods = data.ServicePeriods
		in.Subscription = &sub
	}
	rows, err := tx.Query(ctx, `SELECT id,source,reason,valid_from,valid_until,plan_revision_id FROM entitlement_grants WHERE member_id=$1 AND revoked_at IS NULL ORDER BY valid_from,id`, memberID)
	if err != nil {
		return out, err
	}
	var grants []Grant
	var grantPlans []uuid.UUID
	for rows.Next() {
		var g Grant
		var pid *uuid.UUID
		if err = rows.Scan(&g.ID, &g.Source, &g.Reason, &g.Start, &g.End, &pid); err != nil {
			rows.Close()
			return out, err
		}
		grants = append(grants, g)
		if pid == nil {
			pid = &planID
		}
		grantPlans = append(grantPlans, *pid)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	for i := range grants {
		p, e := LoadPlan(ctx, tx, grantPlans[i])
		if e != nil {
			return out, e
		}
		grants[i].Plan = p
	}
	in.Grants = grants
	result, err := Compute(in)
	if err != nil {
		return out, err
	}
	raw, err := canonjson.Marshal(result)
	if err != nil {
		return out, err
	}
	var old []byte
	var version decstr.U64
	var computed time.Time
	err = tx.QueryRow(ctx, `SELECT projection,revision::text,computed_at FROM entitlements WHERE member_id=$1`, memberID).Scan(&old, &version, &computed)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if err == nil {
		canonical, e := canonjson.Canonicalize(old)
		if e != nil {
			return out, e
		}
		if bytes.Equal(canonical, raw) {
			return Projection{Result: result, MemberID: memberID, Revision: version, ComputedAt: computed}, s.schedule(ctx, tx, memberID, in)
		}
	}
	// Keep the coarse account status and ETag consistent with the separate,
	// more detailed entitlement state. Manual suspension remains an input.
	status := result.State
	switch status {
	case "grace", "cancel_at_period_end":
		status = "active"
	case "past_due":
		status = "expired"
	}
	if _, err = tx.Exec(ctx, `UPDATE members SET status=$2,resource_version=resource_version+1 WHERE id=$1`, memberID, status); err != nil {
		return out, err
	}

	version, err = version.Add(1)
	if err != nil {
		return out, err
	}
	var pid *uuid.UUID
	if result.Plan != nil {
		pid = &result.Plan.ID
	}
	_, err = tx.Exec(ctx, `INSERT INTO entitlements(member_id,revision,plan_revision_id,device_limit,valid_from,effective_service_end,admin_suspended,source_refs,computed_at,state,projection)
 VALUES($1,$2::numeric,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT(member_id) DO UPDATE SET revision=excluded.revision,plan_revision_id=excluded.plan_revision_id,device_limit=excluded.device_limit,valid_from=excluded.valid_from,effective_service_end=excluded.effective_service_end,admin_suspended=excluded.admin_suspended,source_refs=excluded.source_refs,computed_at=excluded.computed_at,state=excluded.state,projection=excluded.projection`, memberID, version.String(), pid, result.DeviceLimit, result.ValidFrom, result.EffectiveServiceEnd, result.AdminSuspended, result.SourceRefs, in.Now, result.State, raw)
	if err != nil {
		return out, err
	}
	historyID, err := uuid.NewV7()
	if err != nil {
		return out, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO entitlement_history(id,member_id,revision,projection) VALUES($1,$2,$3::numeric,$4)`, historyID, memberID, version.String(), raw)
	if err != nil {
		return out, err
	}
	out = Projection{Result: result, MemberID: memberID, Revision: version, ComputedAt: in.Now, Changed: true}
	// A future grant's operation follows the revision that first includes its
	// source, rather than becoming permanently stuck on its acceptance revision.
	if _, err = tx.Exec(ctx, `UPDATE operations o SET desired_generation=$2::numeric WHERE o.id IN (SELECT operation_id FROM entitlement_grants WHERE member_id=$1 AND id=ANY($3) AND operation_id IS NOT NULL) AND o.status IN ('pending','running')`, memberID, version.String(), result.SourceRefs); err != nil {
		return out, err
	}

	payload, err := canonjson.Marshal(out)
	if err != nil {
		return out, err
	}
	err = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: memberID, AggregateType: "member", AggregateVersion: version.String(), EventType: "member.entitlement_changed", SchemaVersion: 1, Payload: payload})
	if err != nil {
		return out, err
	}
	if err = Audit(ctx, tx, Actor{ID: memberID, Role: "system"}, "member.entitlement_changed", memberID, nil, map[string]any{"revision": version, "state": result.State}); err != nil {
		return out, err
	}
	return out, s.schedule(ctx, tx, memberID, in)
}
func (s *Store) schedule(ctx context.Context, tx pgx.Tx, id uuid.UUID, in Input) error {
	var next *time.Time
	add := func(t time.Time) {
		if t.After(in.Now) && (next == nil || t.Before(*next)) {
			next = utc(t)
		}
	}
	if in.Subscription != nil {
		for _, p := range in.Subscription.ServicePeriods {
			add(p.Start)
			add(p.End)
		}
		if in.Subscription.GraceUntil != nil {
			add(*in.Subscription.GraceFrom)
			add(*in.Subscription.GraceUntil)
		}
	}
	for _, g := range in.Grants {
		add(g.Start)
		add(g.End)
	}
	_, err := tx.Exec(ctx, `UPDATE entitlements SET next_transition_at=$2 WHERE member_id=$1`, id, next)
	return err
}

// Sweep advances time-driven transitions and is safe to run concurrently. Nodes
// must still enforce the finite deadline while this worker is unavailable.
func (s *Store) Sweep(ctx context.Context, pool *pgxpool.Pool, limit int) (int, error) {
	if limit < 1 || limit > 1000 {
		return 0, ErrInvalid
	}
	rows, err := pool.Query(ctx, `SELECT member_id FROM entitlements WHERE next_transition_at<=$1 ORDER BY next_transition_at,member_id LIMIT $2`, s.Time(), limit)
	if err != nil {
		return 0, err
	}
	var members []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		members = append(members, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, id := range members {
		err = db.InTx(ctx, pool, db.TxOptions{}, func(tx pgx.Tx) error { _, e := s.Recompute(ctx, tx, id); return e })
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
