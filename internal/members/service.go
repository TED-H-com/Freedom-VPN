// Package members owns account metadata and explicit administrative decisions.
package members

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	Pool         *pgxpool.Pool
	Entitlements *entitlement.Store
}
type Member struct {
	ID                uuid.UUID               `json:"member_id"`
	DisplayName       string                  `json:"display_name"`
	Email             *string                 `json:"email"`
	Status            string                  `json:"status"`
	ResourceVersion   decstr.U64              `json:"resource_version"`
	AdminSuspended    bool                    `json:"admin_suspended"`
	SuspensionReason  *string                 `json:"suspension_reason"`
	SuspendedAt       *time.Time              `json:"suspended_at"`
	SuspendedBy       *uuid.UUID              `json:"suspended_by"`
	Notes             string                  `json:"notes"`
	Groups            []uuid.UUID             `json:"member_group_ids"`
	CreatedAt         time.Time               `json:"created_at"`
	ClosedAt          *time.Time              `json:"closed_at"`
	AuthorizedDevices int                     `json:"authorized_devices"`
	Entitlement       *entitlement.Projection `json:"entitlement"`
	Subscription      *Subscription           `json:"subscription,omitempty"`
	Grants            []Grant                 `json:"grants,omitempty"`
}

type Subscription struct {
	ID                                        uuid.UUID
	Provider, ProviderStatus                  string
	PlanRevisionID                            uuid.UUID
	PaidThrough, GraceUntil, LastReconciledAt *time.Time
	CancelAtPeriodEnd                         bool
	Anomalies                                 []string
}
type Grant struct {
	ID                    uuid.UUID
	Source, Reason        string
	PlanRevisionID        *uuid.UUID
	ValidFrom, ValidUntil time.Time
	RevokedAt             *time.Time
}

const columns = `m.id,m.display_name,m.email,m.status,m.resource_version::text,m.admin_suspended,m.suspension_reason,m.suspended_at,m.suspended_by,m.notes,m.member_group_ids,m.created_at,m.closed_at,(SELECT count(*) FROM member_devices d WHERE d.member_id=m.id AND d.status='authorized')`

func scan(row pgx.Row) (Member, error) {
	var m Member
	err := row.Scan(&m.ID, &m.DisplayName, &m.Email, &m.Status, &m.ResourceVersion, &m.AdminSuspended, &m.SuspensionReason, &m.SuspendedAt, &m.SuspendedBy, &m.Notes, &m.Groups, &m.CreatedAt, &m.ClosedAt, &m.AuthorizedDevices)
	return m, db.MapError(err)
}
func (s *Service) Get(ctx context.Context, actor entitlement.Actor, id uuid.UUID) (Member, error) {
	if err := actor.Own(id); err != nil {
		return Member{}, err
	}
	m, err := scan(s.Pool.QueryRow(ctx, `SELECT `+columns+` FROM members m WHERE m.id=$1`, id))
	if err != nil {
		return m, err
	}
	var p entitlement.Projection
	var raw []byte
	err = s.Pool.QueryRow(ctx, `SELECT projection,revision::text,computed_at FROM entitlements WHERE member_id=$1`, id).Scan(&raw, &p.Revision, &p.ComputedAt)
	if err == nil {
		if err = json.Unmarshal(raw, &p.Result); err != nil {
			return m, err
		}
		p.MemberID = id
		m.Entitlement = &p
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return m, err
	}
	var subscription Subscription
	err = s.Pool.QueryRow(ctx, `SELECT id,provider,provider_status,plan_revision_id,paid_through,grace_until,last_reconciled_at,cancel_at_period_end,reconciliation_anomalies FROM subscriptions WHERE member_id=$1 AND is_primary ORDER BY is_active DESC,created_at DESC,id DESC LIMIT 1`, id).Scan(&subscription.ID, &subscription.Provider, &subscription.ProviderStatus, &subscription.PlanRevisionID, &subscription.PaidThrough, &subscription.GraceUntil, &subscription.LastReconciledAt, &subscription.CancelAtPeriodEnd, &subscription.Anomalies)
	if err == nil {
		m.Subscription = &subscription
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return m, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT id,source,reason,plan_revision_id,valid_from,valid_until,revoked_at FROM entitlement_grants WHERE member_id=$1 ORDER BY created_at DESC,id DESC LIMIT 200`, id)
	if err != nil {
		return m, err
	}
	defer rows.Close()
	for rows.Next() {
		var g Grant
		if err = rows.Scan(&g.ID, &g.Source, &g.Reason, &g.PlanRevisionID, &g.ValidFrom, &g.ValidUntil, &g.RevokedAt); err != nil {
			return m, err
		}
		m.Grants = append(m.Grants, g)
	}
	if err = rows.Err(); err != nil {
		return m, err
	}
	if actor.Role == "member" {
		m.Notes = ""
		m.SuspendedBy = nil
	}
	return m, nil
}

type Page struct {
	Items      []Member
	NextCursor string
}

// List sorts on immutable UUIDs; the base64 cursor is versioned and opaque.
func (s *Service) List(ctx context.Context, actor entitlement.Actor, cursor string, limit int) (Page, error) {
	out := Page{Items: []Member{}}
	if err := actor.Admin(); err != nil {
		return out, err
	}
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return out, entitlement.ErrInvalid
	}
	after := uuid.Nil
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(raw) != 17 || raw[0] != 1 {
			return out, entitlement.ErrInvalid
		}
		copy(after[:], raw[1:])
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+columns+` FROM members m WHERE m.id>$1 ORDER BY m.id LIMIT $2`, after, limit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		m, e := scan(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, m)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		id := out.Items[limit-1].ID
		out.NextCursor = base64.RawURLEncoding.EncodeToString(append([]byte{1}, id[:]...))
	}

	rows.Close()
	ids := make([]uuid.UUID, len(out.Items))
	index := map[uuid.UUID]int{}
	for i, m := range out.Items {
		ids[i] = m.ID
		index[m.ID] = i
	}
	projections, err := s.Pool.Query(ctx, `SELECT member_id,projection,revision::text,computed_at FROM entitlements WHERE member_id=ANY($1)`, ids)
	if err != nil {
		return out, err
	}
	defer projections.Close()
	for projections.Next() {
		var p entitlement.Projection
		var raw []byte
		if err = projections.Scan(&p.MemberID, &raw, &p.Revision, &p.ComputedAt); err != nil {
			return out, err
		}
		if err = json.Unmarshal(raw, &p.Result); err != nil {
			return out, err
		}
		out.Items[index[p.MemberID]].Entitlement = &p
	}
	return out, projections.Err()
}

type Change struct {
	ResourceID  uuid.UUID
	OperationID *uuid.UUID
	Revision    decstr.U64
	Replayed    bool
}

func validatePatch(p api.MemberPatch) error {
	if p.DisplayName == nil && p.Notes == nil && p.MemberGroupIds == nil && p.AdminSuspended == nil {
		return entitlement.ErrInvalid
	}
	if p.DisplayName != nil && !entitlement.Reason(*p.DisplayName, 100) || p.Notes != nil && len([]rune(*p.Notes)) > 2000 {
		return entitlement.ErrInvalid
	}
	if p.SuspensionReason != nil && (!entitlement.Reason(*p.SuspensionReason, 500) || p.AdminSuspended == nil) {
		return entitlement.ErrInvalid
	}
	if p.AdminSuspended != nil && *p.AdminSuspended && (p.SuspensionReason == nil || !entitlement.Reason(*p.SuspensionReason, 500)) {
		return entitlement.ErrInvalid
	}
	if p.MemberGroupIds != nil {
		seen := map[uuid.UUID]bool{}
		if len(*p.MemberGroupIds) > 100 {
			return entitlement.ErrInvalid
		}
		for _, id := range *p.MemberGroupIds {
			if id == uuid.Nil || seen[id] {
				return entitlement.ErrInvalid
			}
			seen[id] = true
		}
	}
	return nil
}
func (s *Service) Patch(ctx context.Context, c entitlement.Command, id uuid.UUID, expected *decstr.U64, p api.MemberPatch) (Change, error) {
	var result Change
	if err := c.Actor.Admin(); err != nil {
		return result, err
	}
	if expected == nil {
		return result, entitlement.ErrPreconditionRequired
	}
	if err := validatePatch(p); err != nil {
		return result, err
	}
	err := db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		record, err := c.Begin(ctx, tx, "PATCH", "/admin/members/"+id.String(), p)
		if err != nil {
			return err
		}
		if !record.New {
			result = Change{ResourceID: id, OperationID: record.OperationID, Replayed: true}
			return nil
		}
		m, err := scan(tx.QueryRow(ctx, `SELECT `+columns+` FROM members m WHERE m.id=$1 FOR UPDATE`, id))
		if err != nil {
			return err
		}
		if m.ResourceVersion != *expected {
			return db.ErrVersionConflict
		}
		if m.Status == "closed" {
			return entitlement.ErrConflict
		}
		changes := map[string]any{}
		if p.DisplayName != nil {
			changes["display_name"] = *p.DisplayName
		}
		if p.Notes != nil {
			changes["notes"] = *p.Notes
		}
		if p.MemberGroupIds != nil {
			for _, group := range *p.MemberGroupIds {
				var exists bool
				if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM member_groups WHERE id=$1)`, group).Scan(&exists); err != nil {
					return err
				}
				if !exists {
					return entitlement.ErrInvalid
				}
			}
			changes["member_group_ids"] = *p.MemberGroupIds
		}
		if p.AdminSuspended != nil {
			changes["admin_suspended"] = *p.AdminSuspended
			if *p.AdminSuspended {
				changes["status"] = "suspended"
				changes["suspension_reason"] = *p.SuspensionReason
				changes["suspended_at"] = s.Entitlements.Time()
				changes["suspended_by"] = c.Actor.ID
			} else {
				changes["status"] = "pending_payment"
				changes["suspension_reason"] = nil
				changes["suspended_at"] = nil
				changes["suspended_by"] = nil
			}
		}
		_, err = db.UpdateWithVersion(ctx, tx, "members", id, expected.String(), changes)
		if err != nil {
			return err
		}
		result = Change{ResourceID: id}
		status := 200
		if p.AdminSuspended != nil || p.MemberGroupIds != nil {
			projection, e := s.Entitlements.Recompute(ctx, tx, id)
			if e != nil {
				return e
			}
			result.Revision = projection.Revision
			if projection.Changed {
				kind := "member.update"
				if p.AdminSuspended != nil {
					if *p.AdminSuspended {
						kind = "member.suspend"
					} else {
						kind = "member.unsuspend"
					}
				}
				op, e := entitlement.Operation(ctx, tx, c.Actor, kind, "member", id, projection.Revision)
				if e != nil {
					return e
				}
				result.OperationID = &op
				status = 202
			}
			if p.AdminSuspended != nil && *p.AdminSuspended && !m.AdminSuspended {
				payload, e := canonjson.Marshal(map[string]any{"member_id": id, "revision": projection.Revision, "admin_suspended": true, "operation_id": result.OperationID})
				if e != nil {
					return e
				}
				if e = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: id, AggregateType: "member", AggregateVersion: projection.Revision.String(), EventType: "member.suspended", SchemaVersion: 1, Payload: payload}); e != nil {
					return e
				}
			}
		}
		if err = entitlement.Audit(ctx, tx, c.Actor, "member.patch", id, map[string]any{"admin_suspended": m.AdminSuspended, "resource_version": m.ResourceVersion}, p); err != nil {
			return err
		}
		return entitlement.Complete(ctx, tx, record, "member", id, result.OperationID, status)
	})
	return result, err
}
func (s *Service) Close(ctx context.Context, c entitlement.Command, id uuid.UUID, expected *decstr.U64, reason string) (Change, error) {
	result := Change{ResourceID: id}
	if err := c.Actor.Admin(); err != nil {
		return result, err
	}
	if expected == nil {
		return result, entitlement.ErrPreconditionRequired
	}
	if !entitlement.Reason(reason, 500) {
		return result, entitlement.ErrInvalid
	}
	err := db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		record, err := c.Begin(ctx, tx, "POST", "member.close/"+id.String(), map[string]string{"reason": reason})
		if err != nil {
			return err
		}
		if !record.New {
			result.OperationID = record.OperationID
			result.Replayed = true
			return nil
		}
		var version decstr.U64
		err = tx.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id=$1 FOR UPDATE`, id).Scan(&version)
		if err != nil {
			return db.MapError(err)
		}
		if version != *expected {
			return db.ErrVersionConflict
		}
		now := s.Entitlements.Time()
		_, err = tx.Exec(ctx, `UPDATE members SET status='closed',closed_at=coalesce(closed_at,$2),resource_version=resource_version+1 WHERE id=$1`, id, now)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE device_bindings SET valid_until=greatest(valid_from,$2) WHERE member_id=$1 AND valid_until>$2`, id, now)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE member_devices SET status='closed',closed_at=$2,resource_version=resource_version+1 WHERE member_id=$1 AND status<>'closed'`, id, now)
		if err != nil {
			return err
		}
		p, err := s.Entitlements.Recompute(ctx, tx, id)
		if err != nil {
			return err
		}
		op, err := entitlement.Operation(ctx, tx, c.Actor, "member.update", "member", id, p.Revision)
		if err != nil {
			return err
		}
		result.OperationID = &op
		result.Revision = p.Revision
		if err = entitlement.Audit(ctx, tx, c.Actor, "member.close", id, nil, map[string]any{"reason": reason, "state": "closed"}); err != nil {
			return err
		}
		return entitlement.Complete(ctx, tx, record, "member", id, &op, 202)
	})
	return result, err
}
func (s *Service) Grant(ctx context.Context, c entitlement.Command, id uuid.UUID, g api.GrantCreateRequest) (Change, error) {
	var result Change
	if err := c.Actor.Admin(); err != nil {
		return result, err
	}
	if !g.Source.Valid() || !entitlement.Reason(g.Reason, 500) {
		return result, entitlement.ErrInvalid
	}
	err := db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		record, err := c.Begin(ctx, tx, "POST", "/admin/members/"+id.String()+"/grants", g)
		if err != nil {
			return err
		}
		if !record.New {
			result = Change{ResourceID: *record.ResourceID, OperationID: record.OperationID, Replayed: true}
			return nil
		}
		var status string
		if err = tx.QueryRow(ctx, `SELECT status FROM members WHERE id=$1 FOR UPDATE`, id).Scan(&status); err != nil {
			return db.MapError(err)
		}
		if status == "closed" {
			return entitlement.ErrConflict
		}
		start := s.Entitlements.Time()
		if g.ValidFrom != nil {
			start = g.ValidFrom.UTC()
		}
		maximum := s.Entitlements.Config.MaxGrantDuration
		if g.Source == api.GrantSourceTrial {
			maximum = s.Entitlements.Config.MaxTrialDuration
			if maximum == 0 {
				return entitlement.ErrInvalid
			}
		}
		if !entitlement.ValidPeriod(start, g.ValidUntil, maximum) || !g.ValidUntil.After(s.Entitlements.Time()) || start.After(s.Entitlements.Time().Add(maximum)) {
			return entitlement.ErrInvalid
		}
		var plan uuid.UUID
		if g.PlanRevisionId != nil {
			plan = *g.PlanRevisionId
		} else {
			if err = tx.QueryRow(ctx, `SELECT plan_revision_id FROM entitlements WHERE member_id=$1 AND plan_revision_id IS NOT NULL`, id).Scan(&plan); err != nil {
				return entitlement.ErrInvalid
			}
		}
		if _, err = entitlement.LoadPlan(ctx, tx, plan); err != nil {
			return err
		}
		var conflict bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM entitlement_grants WHERE member_id=$1 AND revoked_at IS NULL AND plan_revision_id<>$2 AND valid_from<$4 AND valid_until>$3) OR EXISTS(SELECT 1 FROM subscriptions WHERE member_id=$1 AND is_primary AND plan_revision_id<>$2 AND paid_from<$4 AND greatest(paid_through,grace_until)>$3)`, id, plan, start, g.ValidUntil.UTC()).Scan(&conflict); err != nil {
			return err
		}
		if conflict {
			return entitlement.ErrConflict
		}

		gid, err := uuid.NewV7()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO entitlement_grants(id,member_id,source,valid_from,valid_until,reason,created_by,plan_revision_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, gid, id, string(g.Source), start, g.ValidUntil.UTC(), g.Reason, c.Actor.ID.String(), plan)
		if err != nil {
			return err
		}
		p, err := s.Entitlements.Recompute(ctx, tx, id)
		if err != nil {
			return err
		}
		op, err := entitlement.Operation(ctx, tx, c.Actor, "member.grant", "member", id, p.Revision)
		if err != nil {
			return err
		}
		if err = entitlement.Audit(ctx, tx, c.Actor, "member.grant", id, nil, map[string]any{"grant_id": gid, "source": g.Source, "reason": g.Reason, "valid_until": g.ValidUntil.UTC()}); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE entitlement_grants SET operation_id=$2 WHERE id=$1`, gid, op); err != nil {
			return err
		}
		result = Change{ResourceID: gid, OperationID: &op, Revision: p.Revision}
		return entitlement.Complete(ctx, tx, record, "grant", gid, &op, 202)
	})
	return result, err
}

// RevokeGrant is an internal admin API; no extra HTTP endpoint is introduced.
func (s *Service) RevokeGrant(ctx context.Context, c entitlement.Command, member, grant uuid.UUID, reason string) error {
	if err := c.Actor.Admin(); err != nil {
		return err
	}
	if !entitlement.Reason(reason, 500) {
		return entitlement.ErrInvalid
	}
	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		record, err := c.Begin(ctx, tx, "POST", "grant.revoke/"+grant.String(), map[string]any{"member_id": member, "reason": reason})
		if err != nil || !record.New {
			return err
		}
		var status string
		if err = tx.QueryRow(ctx, `SELECT status FROM members WHERE id=$1 FOR UPDATE`, member).Scan(&status); err != nil {
			return db.MapError(err)
		}
		tag, err := tx.Exec(ctx, `UPDATE entitlement_grants SET revoked_at=coalesce(revoked_at,$3),revocation_reason=coalesce(revocation_reason,$4) WHERE id=$1 AND member_id=$2`, grant, member, s.Entitlements.Time(), reason)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return db.ErrNotFound
		}
		if _, err = tx.Exec(ctx, `UPDATE operations SET status='superseded' WHERE id=(SELECT operation_id FROM entitlement_grants WHERE id=$1) AND status IN ('pending','running')`, grant); err != nil {
			return err
		}
		p, err := s.Entitlements.Recompute(ctx, tx, member)
		if err != nil {
			return err
		}
		op, err := entitlement.Operation(ctx, tx, c.Actor, "member.grant", "member", member, p.Revision)
		if err != nil {
			return err
		}
		if err = entitlement.Audit(ctx, tx, c.Actor, "grant.revoke", member, nil, map[string]any{"grant_id": grant, "reason": reason}); err != nil {
			return err
		}
		return entitlement.Complete(ctx, tx, record, "grant", grant, &op, 202)
	})
}
