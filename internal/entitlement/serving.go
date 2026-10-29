package entitlement

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ServingQuerier allows callers to use their existing transaction/snapshot.
type ServingQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type ServingMember struct {
	MemberID       uuid.UUID
	Revision       decstr.U64
	ExitGroupIDs   []uuid.UUID
	DeviceLimit    int
	PlanRevisionID *uuid.UUID
	// Nil means the plan adds no rate cap. No production rate is inferred.
	PlanRateLimit       *api.RatePair
	ValidFrom           time.Time
	EffectiveServiceEnd time.Time
	State               api.EntitlementState
}

// D-030 / §9.2: this is the single serving predicate for policy, lease and exit
// readers. Grace is already included in effective_service_end; assignments is
// reserved and must never be consulted. The entitlement's pinned plan wins over
// a newer active plan definition or an unrelated subscription.
const servingQuery = `SELECT m.id,e.revision::text,e.projection->'exit_group_ids',e.device_limit,e.plan_revision_id,
 pr.payload->'rate_limit',e.valid_from,e.effective_service_end,e.state
 FROM members m JOIN entitlements e ON e.member_id=m.id
 LEFT JOIN plan_revisions pr ON pr.id=e.plan_revision_id
 WHERE m.status='active' AND NOT m.admin_suspended AND NOT e.admin_suspended
 AND e.state NOT IN ('pending_payment','suspended','expired','closed')
 AND e.valid_from<=$3 AND $3<e.effective_service_end
 AND ($1::uuid IS NULL OR m.id=$1)
 AND ($2::text[] IS NULL OR (e.projection->'exit_group_ids') ?| $2::text[])
 ORDER BY m.id`

func ListServingMembersForGroups(ctx context.Context, q ServingQuerier, groupIDs []uuid.UUID, asOf time.Time) ([]ServingMember, error) {
	return listServingMembersForGroups(ctx, q, groupIDs, asOf, false)
}

// ListServingMembersForGroupsForShare returns the serving members for groupIDs
// using the identical D-030 servingQuery predicate while acquiring FOR SHARE
// row locks on members and entitlements in the caller's transaction (§16.3, D-030).
func ListServingMembersForGroupsForShare(ctx context.Context, q ServingQuerier, groupIDs []uuid.UUID, asOf time.Time) ([]ServingMember, error) {
	return listServingMembersForGroups(ctx, q, groupIDs, asOf, true)
}

func listServingMembersForGroups(ctx context.Context, q ServingQuerier, groupIDs []uuid.UUID, asOf time.Time, lockForShare bool) ([]ServingMember, error) {
	if len(groupIDs) == 0 {
		return []ServingMember{}, nil
	}
	groups := make([]string, len(groupIDs))
	for i, id := range groupIDs {
		if id == uuid.Nil {
			return nil, ErrInvalid
		}
		groups[i] = id.String()
	}
	return servingMembers(ctx, q, nil, groups, asOf, lockForShare)
}

// ServingExitGroups returns an empty set for unknown/non-serving members.
func ServingExitGroups(ctx context.Context, q ServingQuerier, memberID uuid.UUID, asOf time.Time) ([]uuid.UUID, error) {
	if memberID == uuid.Nil {
		return nil, ErrInvalid
	}
	members, err := servingMembers(ctx, q, &memberID, nil, asOf, false)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return []uuid.UUID{}, nil
	}
	return members[0].ExitGroupIDs, nil
}
func servingMembers(ctx context.Context, q ServingQuerier, member *uuid.UUID, groups []string, asOf time.Time, lockForShare bool) ([]ServingMember, error) {
	if !instant(asOf) {
		return nil, ErrInvalid
	}
	query := servingQuery
	if lockForShare {
		query += " FOR SHARE OF m, e"
	}
	rows, err := q.Query(ctx, query, member, groups, asOf.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ServingMember{}
	for rows.Next() {
		var m ServingMember
		var groupJSON, rateJSON []byte
		if err = rows.Scan(&m.MemberID, &m.Revision, &groupJSON, &m.DeviceLimit, &m.PlanRevisionID, &rateJSON, &m.ValidFrom, &m.EffectiveServiceEnd, &m.State); err != nil {
			return nil, err
		}
		if len(groupJSON) == 0 || json.Unmarshal(groupJSON, &m.ExitGroupIDs) != nil {
			return nil, ErrInvalid
		}
		if len(rateJSON) > 0 && json.Unmarshal(rateJSON, &m.PlanRateLimit) != nil {
			return nil, ErrInvalid
		}
		if m.PlanRateLimit != nil && (m.PlanRateLimit.DownBps < 0 || m.PlanRateLimit.UpBps < 0) {
			return nil, ErrInvalid
		}
		m.ExitGroupIDs = ids(m.ExitGroupIDs)
		m.ValidFrom = m.ValidFrom.UTC()
		m.EffectiveServiceEnd = m.EffectiveServiceEnd.UTC()
		out = append(out, m)
	}
	return out, rows.Err()
}
