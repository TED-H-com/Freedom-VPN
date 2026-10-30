// Package fleetupdate owns fleet snapshots, authorized jobs and result history
// (Project Spec §13.1, §13.8, §13.12, §15.5.6).
package fleetupdate

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// These timings are deployment settings requiring measurement, not invented SLOs.
type Config struct{ AuthorizationTTL, OnlineWindow, StuckWindow time.Duration }
type Service struct {
	pool     *pgxpool.Pool
	releases *releases.Service
	config   Config
}

func New(pool *pgxpool.Pool, r *releases.Service, c Config) (*Service, error) {
	if pool == nil || r == nil || r.Validator == nil || c.AuthorizationTTL <= 0 || c.OnlineWindow <= 0 || c.StuckWindow <= 0 {
		return nil, errors.New("database, releases and measured fleet timing settings are required")
	}
	return &Service{pool: pool, releases: r, config: c}, nil
}

// Identity comes exclusively from the verified mTLS binding (§14.4).
type Identity struct{ NodeID, IncarnationID uuid.UUID }

type node struct {
	id                               uuid.UUID
	incarnation                      *uuid.UUID
	lifecycle, publication, os, arch string
	generation                       decstr.U64
	current, lastGood                *uuid.UUID
	online                           bool
	job, target                      *uuid.UUID
	observedHighWater                decstr.U64
	observedAt                       *time.Time
}

func (s *Service) lock(ctx context.Context, tx pgx.Tx) error {
	_, e := tx.Exec(ctx, `SELECT id FROM fleet_config WHERE id=$1 FOR UPDATE`, releases.FleetID)
	return e
}
func (s *Service) readNode(ctx context.Context, tx pgx.Tx, id uuid.UUID) (node, error) {
	n := node{id: id}
	var generation, observed string
	e := tx.QueryRow(ctx, `SELECT n.current_incarnation_id,n.lifecycle_state,coalesce(x.publication_intent,'unpublished'),coalesce(i.os,''),coalesce(i.arch,''),n.update_generation::text,n.current_release_id,n.last_good_release_id,
 coalesce((SELECT o.control_status='online' AND o.received_at>clock_timestamp()-$2::bigint*interval '1 millisecond' FROM node_observations o WHERE o.node_id=n.id AND o.incarnation_id=n.current_incarnation_id ORDER BY o.received_at DESC LIMIT 1),false)
 OR coalesce(s.observation_incarnation_id=n.current_incarnation_id AND s.received_at>clock_timestamp()-$2::bigint*interval '1 millisecond',false),s.current_job_id,s.current_target_id,coalesce(s.observed_update_generation,0)::text,s.last_observed_at
 FROM nodes n LEFT JOIN exits x ON x.node_id=n.id LEFT JOIN node_incarnations i ON i.id=n.current_incarnation_id LEFT JOIN fleet_node_state s ON s.node_id=n.id WHERE n.id=$1 FOR UPDATE OF n`, id, s.config.OnlineWindow.Milliseconds()).Scan(&n.incarnation, &n.lifecycle, &n.publication, &n.os, &n.arch, &generation, &n.current, &n.lastGood, &n.online, &n.job, &n.target, &observed, &n.observedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		return n, apierr.New(apierr.ResourceNotFound, "Node does not exist")
	}
	if e != nil {
		return n, e
	}
	n.generation, e = decstr.Parse(generation)
	if e != nil {
		return n, e
	}
	n.observedHighWater, e = decstr.Parse(observed)
	return n, e
}
func authenticate(ctx context.Context, tx pgx.Tx, n node, i Identity) error {
	if i.NodeID != n.id || i.IncarnationID == uuid.Nil || n.incarnation == nil || *n.incarnation != i.IncarnationID {
		return apierr.New(apierr.IncarnationMismatch, "Node incarnation does not match authenticated identity")
	}
	if n.lifecycle == "revoked" || n.lifecycle == "retired" {
		return apierr.New(apierr.IdentityRevoked, "Node identity is revoked or retired")
	}
	var revoked bool
	if e := tx.QueryRow(ctx, `SELECT revoked_at IS NOT NULL FROM node_incarnations WHERE id=$1 AND node_id=$2`, i.IncarnationID, i.NodeID).Scan(&revoked); e != nil {
		return e
	}
	if revoked {
		return apierr.New(apierr.IdentityRevoked, "Node incarnation is revoked")
	}
	return nil
}

// SetFleetRelease is the command interface used by the control/desired-state
// integration. Its implementation is the complete atomic CreateBatch operation.
func (s *Service) SetFleetRelease(ctx context.Context, c releases.Command, r api.UpdateBatchCreateRequest) (api.UpdateBatchAccepted, error) {
	return s.CreateBatch(ctx, c, r)
}
func (s *Service) CreateBatch(ctx context.Context, c releases.Command, r api.UpdateBatchCreateRequest) (api.UpdateBatchAccepted, error) {
	var out api.UpdateBatchAccepted
	raw, e := json.Marshal(r)
	if e != nil {
		return out, e
	}
	if e = s.releases.Validator.ValidateBody("UpdateBatchCreateRequest", raw); e != nil {
		return out, e
	}
	e = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		rec, e := releases.Begin(ctx, tx, c, "/api/v1/admin/update-batches", raw)
		if e != nil {
			return e
		}
		if !rec.New {
			out, e = accepted(ctx, tx, *rec.ResourceID)
			return e
		}
		if e = s.lock(ctx, tx); e != nil {
			return e
		}
		var gen string
		if e = tx.QueryRow(ctx, `SELECT generation::text FROM fleet_config`).Scan(&gen); e != nil {
			return e
		}
		if gen != r.ExpectedGeneration.String() {
			return apierr.New(apierr.StateConflict, "Fleet generation changed")
		}
		if _, _, _, e = s.releases.Eligible(ctx, tx, r.ReleaseId); e != nil {
			return e
		}
		// Take the base snapshot lock before reading the population (§15.5.6).
		if _, e = tx.Exec(ctx, `LOCK TABLE nodes IN SHARE MODE`); e != nil {
			return e
		}
		rows, e := tx.Query(ctx, `SELECT id FROM nodes WHERE lifecycle_state NOT IN ('revoked','retired') ORDER BY id FOR UPDATE`)
		if e != nil {
			return e
		}
		ids, e := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
		if e != nil {
			return e
		}
		nodes := make([]node, 0, len(ids))
		online := 0
		for _, id := range ids {
			n, e := s.readNode(ctx, tx, id)
			if e != nil {
				return e
			}
			nodes = append(nodes, n)
			if n.online {
				online++
			}
		}
		batch := uuid.Must(uuid.NewV7())
		_, e = tx.Exec(ctx, `UPDATE fleet_config SET generation=generation+1,resource_version=resource_version+1,desired_release_id=$1`, r.ReleaseId)
		if e != nil {
			return e
		}
		desiredGeneration, e := r.ExpectedGeneration.Add(1)
		if e != nil {
			return e
		}
		requestID := commandtx.RequestID(ctx)
		op, e := operations.Create(ctx, tx, operations.CreateRequest{
			Kind: api.OperationKindUpdateBatchCreate, Principal: auth.Principal{Role: auth.Admin, ID: c.ActorID},
			Resource: api.ResourceRef{Type: api.ResourceTypeUpdateBatch, Id: batch}, RequestID: requestID,
			DesiredGeneration: &desiredGeneration,
			Steps:             map[string]operations.Step{"snapshot": {Required: true}, "targets": {Required: true}},
		})
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO update_batches(id,operation_id,target_release_id,scope,created_by,target_count,generation,reason,online_at_snapshot) SELECT $1,$2,$3,'all_nodes',$4,$5,generation,$6,$7 FROM fleet_config`, batch, op, r.ReleaseId, c.ActorID.String(), len(nodes), r.Reason, online)
		if e != nil {
			return e
		}
		for _, n := range nodes {
			gen, e := max(n.generation, n.observedHighWater).Add(1)
			if e != nil {
				return e
			}
			target := uuid.Must(uuid.NewV7())
			_, e = tx.Exec(ctx, `INSERT INTO update_targets(id,batch_id,node_id,incarnation_at_snapshot,execution_incarnation_id,target_generation,online_at_snapshot) VALUES($1,$2,$3,$4,$4,$5::numeric,$6)`, target, batch, n.id, n.incarnation, gen.String(), n.online)
			if e != nil {
				return e
			}
			if e = s.newJob(ctx, tx, n, &target, r.ReleaseId, gen, 1); e != nil {
				return e
			}
		}
		if e = notify(ctx, tx, batch, "fleet.update_created"); e != nil {
			return e
		}
		event, e := audit.NewEvent(audit.Entry{
			Actor: audit.Actor{Type: api.AuditEventActorTypeAdmin, ID: c.ActorID}, Action: "update_batch.create",
			Resource: api.ResourceRef{Type: api.ResourceTypeUpdateBatch, Id: batch}, Result: api.AuditEventResultSucceeded,
			OperationID: &op, RequestID: &requestID,
			After: map[string]any{"target_count": len(nodes), "release_id": r.ReleaseId}, Fields: []string{"target_count", "release_id"},
		})
		if e != nil {
			return e
		}
		if e = audit.AppendEvent(ctx, tx, event); e != nil {
			return e
		}
		if e = recomputeAll(ctx, tx); e != nil {
			return e
		}
		out, e = accepted(ctx, tx, batch)
		if e != nil {
			return e
		}
		return commandtx.CompleteIdempotent(ctx, tx, rec.ID, db.IdempotencyResult{ResourceType: "update_batch", ResourceID: &batch, OperationID: &op, ResponseStatus: 202})
	})
	return out, e
}

func (s *Service) newJob(ctx context.Context, tx pgx.Tx, n node, target *uuid.UUID, release uuid.UUID, generation decstr.U64, attempt int) error {
	// §13.8: applying/checking remain authorized to finish; current pointer always
	// names the newest desired job. No publication field is ever written here.
	_, e := tx.Exec(ctx, `UPDATE update_jobs SET state='superseded',phase_since=clock_timestamp() WHERE node_id=$1 AND state IN ('queued','waiting_online','downloading','verifying')`, n.id)
	if e != nil {
		return e
	}
	if target == nil {
		return errors.New("every job requires a sealed snapshot target")
	}
	state := "queued"
	if !n.online {
		state = "waiting_online"
	}
	// §13.1 / AT24: an enrolling target reserves its generation, but a job
	// is only created once authentication supplies its immutable incarnation.
	var job *uuid.UUID
	if n.incarnation != nil {
		id := uuid.Must(uuid.NewV7())
		job = &id
		_, e = tx.Exec(ctx, `INSERT INTO update_jobs(id,target_id,node_id,release_id,attempt,execution_incarnation_id,generation,state) VALUES($1,$2,$3,$4,$5,$6,$7::numeric,$8)`, job, target, n.id, release, attempt, n.incarnation, generation.String(), state)
		if e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `UPDATE nodes SET update_generation=$2::numeric,desired_release_id=$3,resource_version=resource_version+1 WHERE id=$1`, n.id, generation.String(), release)
	if e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO fleet_node_state(node_id,current_job_id,current_target_id) VALUES($1,$2,$3) ON CONFLICT(node_id) DO UPDATE SET current_job_id=excluded.current_job_id,current_target_id=excluded.current_target_id`, n.id, job, target)
	return e
}
func notify(ctx context.Context, tx pgx.Tx, id uuid.UUID, event string) error {
	return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: id, AggregateType: "fleet_update", AggregateVersion: "1", EventType: event, SchemaVersion: 1, Payload: json.RawMessage(`{"schema_version":1}`)})
}
func accepted(ctx context.Context, tx pgx.Tx, id uuid.UUID) (api.UpdateBatchAccepted, error) {
	var a api.UpdateBatchAccepted
	var gen string
	e := tx.QueryRow(ctx, `SELECT id,operation_id,target_release_id,target_count,online_at_snapshot,generation::text,created_at FROM update_batches WHERE id=$1`, id).Scan(&a.BatchId, &a.OperationId, &a.ReleaseId, &a.TargetCount, &a.OnlineAtSnapshot, &gen, &a.CreatedAt)
	if e != nil {
		return a, e
	}
	a.CreatedAt = a.CreatedAt.UTC()
	a.FleetGeneration, e = decstr.Parse(gen)
	a.FleetDefaultReleaseId = a.ReleaseId
	a.OfflineAtSnapshot = a.TargetCount - a.OnlineAtSnapshot
	a.State = "accepted"
	a.StatusUrl = "/api/v1/admin/update-batches/" + id.String()
	return a, e
}

// Accepted re-reads the immutable acceptance snapshot for HTTP command replay.
func (s *Service) Accepted(ctx context.Context, id uuid.UUID) (out api.UpdateBatchAccepted, err error) {
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{TxOptions: pgx.TxOptions{AccessMode: pgx.ReadOnly}}, func(tx pgx.Tx) error { var e error; out, e = accepted(ctx, tx, id); return e })
	return
}
