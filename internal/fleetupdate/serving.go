package fleetupdate

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
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
	ObservedGeneration                                    *decstr.U64
}

func (s *Service) NodeVersionView(ctx context.Context, nodeID uuid.UUID) (VersionView, error) {
	var v VersionView
	var generation *string
	e := s.pool.QueryRow(ctx, `SELECT n.current_release_id,n.desired_release_id,n.last_good_release_id,
 coalesce(f.supported_release_policy='allowlist' AND n.current_release_id=ANY(f.supported_release_ids) AND NOT n.current_release_id=ANY(f.revoked_release_ids),false),
 coalesce(s.recovery_required,false) OR coalesce(n.last_good_release_id=ANY(f.revoked_release_ids),false),
 s.last_observed_at,j.last_result_at,(`+observedGenerationSQL+`)::text,
 `+unconfirmedSQL+`
 FROM nodes n CROSS JOIN fleet_config f LEFT JOIN fleet_node_state s ON s.node_id=n.id LEFT JOIN update_jobs j ON j.id=s.current_job_id WHERE n.id=$1`, nodeID).Scan(&v.CurrentReleaseID, &v.DesiredReleaseID, &v.LastGoodReleaseID, &v.ReleaseServiceable, &v.RecoveryRequired, &v.LastObservedAt, &v.LastResultAt, &generation, &v.Unconfirmed)
	if e != nil {
		return v, e
	}
	utcTime(v.LastResultAt)
	utcTime(v.LastObservedAt)
	if generation != nil {
		value, err := decstr.Parse(*generation)
		if err != nil {
			return v, err
		}
		v.ObservedGeneration = &value
	}
	return v, nil
}

// Observation is the trusted sequence adapter, including restore reconciliation.
// Sequence is monotonic within the incarnation, independent of device wall-clock
// time. HTTP callers must use ObserveReportTx, which requires a bound job to
// advance a generation and identifies the exact attempt. Trusted sequence
// callers can only confirm the current job at its matching generation.
type Observation struct {
	Sequence, AppliedUpdateGeneration   decstr.U64
	CurrentReleaseID, LastGoodReleaseID *uuid.UUID
	ObservedAt                          time.Time
	ReceivedAt                          time.Time
}

func (s *Service) Observe(ctx context.Context, i Identity, o Observation) error {
	return commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error { return s.ObserveTx(ctx, tx, i, o) })
}
func (s *Service) ObserveTx(ctx context.Context, tx pgx.Tx, i Identity, o Observation) error {
	return s.observeTx(ctx, tx, i, o, nil)
}

func (s *Service) observeTx(ctx context.Context, tx pgx.Tx, i Identity, o Observation, jobID *uuid.UUID) error {
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
	// D-062 / r9: heartbeat seeding ends at the first accepted result for this
	// incarnation. Thereafter only results can change serving version/recovery.
	var accepted, recovery bool
	if e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM update_results WHERE node_id=$1 AND incarnation_id=$2 AND ack->>'accepted'='true'),
 coalesce($3::uuid=ANY(revoked_release_ids),false) FROM fleet_config`, i.NodeID, i.IncarnationID, n.lastGood).Scan(&accepted, &recovery); e != nil {
		return e
	}
	if o.ReceivedAt.IsZero() {
		o.ReceivedAt = time.Now().UTC()
	}
	tag, e := tx.Exec(ctx, `INSERT INTO fleet_node_state(node_id,observation_incarnation_id,observation_seq,observed_update_generation,last_observed_at,received_at,recovery_required) VALUES($1,$2,$3::numeric,$4::numeric,$5,$7,$6)
 ON CONFLICT(node_id) DO UPDATE SET observation_incarnation_id=excluded.observation_incarnation_id,observation_seq=excluded.observation_seq,observed_update_generation=excluded.observed_update_generation,last_observed_at=excluded.last_observed_at,received_at=excluded.received_at,recovery_required=fleet_node_state.recovery_required OR excluded.recovery_required
 WHERE (fleet_node_state.observation_incarnation_id IS DISTINCT FROM excluded.observation_incarnation_id OR fleet_node_state.observation_seq<excluded.observation_seq) AND fleet_node_state.observed_update_generation<=excluded.observed_update_generation`, i.NodeID, i.IncarnationID, o.Sequence.String(), o.AppliedUpdateGeneration.String(), o.ObservedAt, recovery && !accepted, o.ReceivedAt)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	effectiveJob := jobID
	if effectiveJob == nil {
		// Only trusted sequence adapters lack a reported job. The wire adapter
		// returns before observeTx when either job or generation is absent.
		effectiveJob = n.job
	}
	agrees := equalID(n.current, o.CurrentReleaseID) && equalID(n.lastGood, o.LastGoodReleaseID)
	if !accepted {
		var targetRelease *uuid.UUID
		if effectiveJob != nil {
			j, err := readJob(ctx, tx, *effectiveJob)
			if err != nil {
				return err
			}
			if n.job != nil && equalID(effectiveJob, n.job) && equalID(j.incarnation, &i.IncarnationID) && j.issued != nil && j.state != "superseded" && j.state != "superseded_incarnation" {
				targetRelease = &j.releaseID
			}
		}
		// §13.2, §13.8 / D-068: pre-result heartbeat seeding fills only a null
		// current_release_id (a fresh incarnation with no record). A non-null
		// current release advances only through an accepted UpdateResult.
		expected := func(id *uuid.UUID) bool {
			return id != nil && (equalID(id, targetRelease) || equalID(id, n.current) || equalID(id, n.lastGood))
		}
		if n.current == nil && expected(o.CurrentReleaseID) && equalID(n.lastGood, o.LastGoodReleaseID) {
			if _, e = tx.Exec(ctx, `UPDATE nodes SET current_release_id=$2,resource_version=resource_version+1 WHERE id=$1 AND current_release_id IS NULL`, i.NodeID, o.CurrentReleaseID); e != nil {
				return e
			}
			agrees = equalID(n.lastGood, o.LastGoodReleaseID)
		}
		if effectiveJob != nil {
			for _, id := range []*uuid.UUID{o.CurrentReleaseID, o.LastGoodReleaseID} {
				if id != nil && !expected(id) {
					if e = auditUnexpectedRelease(ctx, tx, i, *effectiveJob, *id); e != nil {
						return e
					}
				}
			}
		}
	}
	// A disagreement invalidates confirmation without changing accepted values.
	_, e = tx.Exec(ctx, `UPDATE update_jobs SET last_observed_at=CASE WHEN $5 THEN $2::timestamptz ELSE NULL END WHERE id=$6 AND node_id=$1 AND execution_incarnation_id=$3 AND generation=$4::numeric`, i.NodeID, o.ObservedAt, i.IncarnationID, o.AppliedUpdateGeneration.String(), agrees, effectiveJob)
	return e
}

// The fleet lock held by observeTx serializes this check and append, including
// concurrent reports. D-029: one immutable audit per unexpected release/job,
// regardless of heartbeat count or which version field contained the release.
func auditUnexpectedRelease(ctx context.Context, tx pgx.Tx, i Identity, jobID, releaseID uuid.UUID) error {
	var recorded bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events WHERE resource=$1 AND action='update_observation.unexpected_release' AND after_diff->>'job_id'=$2 AND after_diff->>'release_id'=$3)`, "node:"+i.NodeID.String(), jobID.String(), releaseID.String()).Scan(&recorded); err != nil || recorded {
		return err
	}
	requestID := commandtx.RequestID(ctx)
	event, err := audit.NewEvent(audit.Entry{
		Actor: audit.Actor{Type: api.AuditEventActorTypeNode, ID: i.NodeID}, Action: "update_observation.unexpected_release",
		Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: i.NodeID}, Result: api.AuditEventResultFailed, RequestID: &requestID,
		After: map[string]any{"job_id": jobID, "release_id": releaseID}, Fields: []string{"job_id", "release_id"},
	})
	if err != nil {
		return err
	}
	return audit.AppendEvent(ctx, tx, event)
}

// ObserveReportTx adapts the existing HTTP ObservationReport without adding a
// wire sequence field. The observation owner must have inserted its immutable,
// deduplicated node_observations receipt in this same transaction first. Ordering
// uses authenticated job/generation plus observed time within that generation;
// the local sequence orders accepted central receipts, never client counters.
func (s *Service) ObserveReportTx(ctx context.Context, tx pgx.Tx, i Identity, r api.ObservationReport) error {
	if r.NodeId != i.NodeID || r.NodeIncarnationId != i.IncarnationID {
		return apierr.New(apierr.IncarnationMismatch, "Observation differs from mTLS identity")
	}
	if state, err := r.Update.State.Get(); err == nil && !state.Valid() {
		return apierr.New(apierr.ValidationFailed, "Unknown observation update state")
	}
	if stage, err := r.Update.Stage.Get(); err == nil && !stage.Valid() {
		return apierr.New(apierr.ValidationFailed, "Unknown observation update stage")
	}
	if e := s.lock(ctx, tx); e != nil {
		return e
	}
	var received time.Time
	if e := tx.QueryRow(ctx, `SELECT received_at FROM node_observations WHERE node_id=$1 AND incarnation_id=$2 AND report_id=$3`, i.NodeID, i.IncarnationID, r.ReportId).Scan(&received); e != nil {
		return e
	}
	n, err := s.readNode(ctx, tx, i.NodeID)
	if err != nil {
		return err
	}
	if err = authenticate(ctx, tx, n, i); err != nil {
		return err
	}
	generation, generationErr := r.Update.Generation.Get()
	jobID, jobErr := r.Update.JobId.Get()
	if generationErr != nil || jobErr != nil {
		// D-062(e) / r9: without both generation and job identity this receipt
		// cannot attest update state. Preserve the high-water
		// mark and actual version, but record its arrival so it cannot confirm
		// a target using a previous generation-bearing heartbeat.
		_, err = tx.Exec(ctx, `INSERT INTO fleet_node_state(node_id,observation_incarnation_id,last_observed_at,received_at)
 VALUES($1,NULL,$2,$3) ON CONFLICT(node_id) DO UPDATE SET observation_incarnation_id=NULL,
 last_observed_at=excluded.last_observed_at,received_at=excluded.received_at
 WHERE fleet_node_state.received_at IS NULL OR (excluded.received_at>fleet_node_state.received_at
 AND (fleet_node_state.last_observed_at IS NULL OR excluded.last_observed_at>fleet_node_state.last_observed_at))`, i.NodeID, r.ObservedAt, received)
		return err
	}
	{
		j, e := readJob(ctx, tx, jobID)
		if e != nil {
			return e
		}
		if j.nodeID != i.NodeID || j.incarnation == nil || *j.incarnation != i.IncarnationID || j.generation != generation {
			return apierr.New(apierr.StaleGeneration, "Observation job and generation do not match")
		}
		// D-068 / r13 M-1: a heartbeat is attributed only to the node's current,
		// non-superseded job. Observations carrying a superseded attempt's job id
		// (including same-generation retries) are ignored by the fleet state row.
		if n.job == nil || *n.job != jobID || j.state == "superseded" || j.state == "superseded_incarnation" {
			return nil
		}
	}
	var seqText, genText string
	var lastObserved, lastReceived *time.Time
	var incarnation *uuid.UUID
	e := tx.QueryRow(ctx, `SELECT observation_seq::text,observed_update_generation::text,last_observed_at,received_at,observation_incarnation_id FROM fleet_node_state WHERE node_id=$1`, i.NodeID).Scan(&seqText, &genText, &lastObserved, &lastReceived, &incarnation)
	sequence := decstr.U64(1)
	if e == nil && (incarnation == nil || *incarnation == i.IncarnationID) {
		previous, e := decstr.Parse(genText)
		if e != nil {
			return e
		}
		seq, e := decstr.Parse(seqText)
		if e != nil {
			return e
		}
		if generation < previous || lastReceived != nil && !received.After(*lastReceived) || generation == previous && lastObserved != nil && !r.ObservedAt.After(*lastObserved) {
			return nil
		}
		sequence, e = seq.Add(1)
		if e != nil {
			return e
		}
	} else if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	var currentReleaseID, lastGoodReleaseID *uuid.UUID
	if value, err := r.CurrentReleaseId.Get(); err == nil {
		currentReleaseID = &value
	}
	if value, err := r.LastGoodReleaseId.Get(); err == nil {
		lastGoodReleaseID = &value
	}
	return s.observeTx(ctx, tx, i, Observation{Sequence: sequence, AppliedUpdateGeneration: generation, CurrentReleaseID: currentReleaseID, LastGoodReleaseID: lastGoodReleaseID, ObservedAt: r.ObservedAt, ReceivedAt: received}, &jobID)
}
