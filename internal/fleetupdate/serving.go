package fleetupdate

import (
	"context"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ServingConditions intentionally has no desired release field (§5.1, §13.2).
// PolicyValid includes expiry, isolation and the independently signed serving set.
type ServingConditions struct {
	IdentityValid, ProvisioningReady, PolicyValid, MemberEntitlementValid bool
	PublicationIntent, UpdateState                                        string
	CurrentReleaseID                                                      uuid.UUID
	ServingReleaseIDs, RevokedReleaseIDs                                  []uuid.UUID
	RecoveryRequired                                                      bool
}

func ServingAllowed(c ServingConditions) bool {
	return c.IdentityValid && c.ProvisioningReady && c.PolicyValid && c.MemberEntitlementValid && c.PublicationIntent == "published" && c.UpdateState != "applying" && c.UpdateState != "checking" && !c.RecoveryRequired && slices.Contains(c.ServingReleaseIDs, c.CurrentReleaseID) && !slices.Contains(c.RevokedReleaseIDs, c.CurrentReleaseID)
}

// VersionView is consumed by internal/nodes and the desired-state compiler.
// ReleaseServiceable is only the version predicate, not full serving permission.
type VersionView struct {
	CurrentReleaseID, DesiredReleaseID, LastGoodReleaseID *uuid.UUID
	ReleaseServiceable, RecoveryRequired, Unconfirmed     bool
	LastResultAt, LastObservedAt                          *time.Time
	ObservedGeneration                                    decstr.U64
}

func (s *Service) NodeVersionView(ctx context.Context, nodeID uuid.UUID) (VersionView, error) {
	var v VersionView
	var generation string
	e := s.pool.QueryRow(ctx, `SELECT n.current_release_id,n.desired_release_id,n.last_good_release_id,
 coalesce(n.current_release_id=ANY(f.supported_release_ids) AND NOT n.current_release_id=ANY(f.revoked_release_ids),false),
 coalesce(s.recovery_required,false) OR coalesce(n.last_good_release_id=ANY(f.revoked_release_ids),false),
 s.last_observed_at,j.last_result_at,coalesce(s.observed_update_generation,0)::text,
 s.last_observed_at IS NULL OR j.last_result_at>s.received_at
 FROM nodes n CROSS JOIN fleet_config f LEFT JOIN fleet_node_state s ON s.node_id=n.id LEFT JOIN update_jobs j ON j.id=s.current_job_id WHERE n.id=$1`, nodeID).Scan(&v.CurrentReleaseID, &v.DesiredReleaseID, &v.LastGoodReleaseID, &v.ReleaseServiceable, &v.RecoveryRequired, &v.LastObservedAt, &v.LastResultAt, &generation, &v.Unconfirmed)
	if e != nil {
		return v, e
	}
	v.ObservedGeneration, e = decstr.Parse(generation)
	return v, e
}

// Observation is called by the heartbeat owner after identity validation. Seq is
// monotonic within the incarnation, independent of device wall-clock time.
type Observation struct {
	Sequence, AppliedUpdateGeneration   decstr.U64
	CurrentReleaseID, LastGoodReleaseID *uuid.UUID
	ObservedAt                          time.Time
}

func (s *Service) Observe(ctx context.Context, i Identity, o Observation) error {
	return db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error { return s.ObserveTx(ctx, tx, i, o) })
}
func (s *Service) ObserveTx(ctx context.Context, tx pgx.Tx, i Identity, o Observation) error {
	if e := s.lock(ctx, tx); e != nil {
		return e
	}
	n, e := s.readNode(ctx, tx, i.NodeID)
	if e != nil {
		return e
	}
	if e = authenticate(ctx, tx, n, i); e != nil {
		return e
	}
	if o.Sequence == 0 || o.ObservedAt.IsZero() {
		return apierr.New(apierr.ValidationFailed, "Observation sequence and time are required")
	}
	var recovery bool
	if e = tx.QueryRow(ctx, `SELECT coalesce($1::uuid=ANY(revoked_release_ids),false) FROM fleet_config`, o.LastGoodReleaseID).Scan(&recovery); e != nil {
		return e
	}
	tag, e := tx.Exec(ctx, `INSERT INTO fleet_node_state(node_id,observation_incarnation_id,observation_seq,observed_update_generation,last_observed_at,received_at,recovery_required) VALUES($1,$2,$3::numeric,$4::numeric,$5,clock_timestamp(),$6)
 ON CONFLICT(node_id) DO UPDATE SET observation_incarnation_id=excluded.observation_incarnation_id,observation_seq=excluded.observation_seq,observed_update_generation=excluded.observed_update_generation,last_observed_at=excluded.last_observed_at,received_at=excluded.received_at,recovery_required=excluded.recovery_required
 WHERE fleet_node_state.observation_incarnation_id IS DISTINCT FROM excluded.observation_incarnation_id OR (fleet_node_state.observation_seq<excluded.observation_seq AND fleet_node_state.observed_update_generation<=excluded.observed_update_generation)`, i.NodeID, i.IncarnationID, o.Sequence.String(), o.AppliedUpdateGeneration.String(), o.ObservedAt, recovery)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	_, e = tx.Exec(ctx, `UPDATE nodes SET current_release_id=$2,last_good_release_id=$3,resource_version=resource_version+1 WHERE id=$1`, i.NodeID, o.CurrentReleaseID, o.LastGoodReleaseID)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `UPDATE update_jobs SET last_observed_at=$2 WHERE node_id=$1 AND execution_incarnation_id=$3`, i.NodeID, o.ObservedAt, i.IncarnationID)
	return e
}
