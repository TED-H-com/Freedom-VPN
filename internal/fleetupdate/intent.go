package fleetupdate

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type job struct {
	id, nodeID, releaseID      uuid.UUID
	target, batch, incarnation *uuid.UUID
	generation                 decstr.U64
	attempt                    int
	state                      string
	issued, validUntil         *time.Time
	manifestHash               string
	resultID                   *uuid.UUID
}

func readJob(ctx context.Context, tx pgx.Tx, id uuid.UUID) (job, error) {
	var j job
	var gen string
	e := tx.QueryRow(ctx, `SELECT j.id,j.node_id,j.release_id,j.target_id,t.batch_id,j.execution_incarnation_id,j.generation::text,j.attempt,j.state,j.issued_at,j.authorization_valid_until,encode(r.manifest_hash,'hex'),j.result_id FROM update_jobs j LEFT JOIN update_targets t ON t.id=j.target_id JOIN releases r ON r.id=j.release_id WHERE j.id=$1 FOR UPDATE OF j`, id).Scan(&j.id, &j.nodeID, &j.releaseID, &j.target, &j.batch, &j.incarnation, &gen, &j.attempt, &j.state, &j.issued, &j.validUntil, &j.manifestHash, &j.resultID)
	if errors.Is(e, pgx.ErrNoRows) {
		return j, apierr.New(apierr.ResourceNotFound, "Update job does not exist")
	}
	if e != nil {
		return j, e
	}
	j.generation, e = decstr.Parse(gen)
	return j, e
}

// Work has the exact contract intent plus the selected immutable artifact for
// compiler/preflight use. The artifact is not added to the signed intent format.
type Work struct {
	Intent   api.UpdateIntent
	Artifact api.ManifestArtifact
}

func (s *Service) NodeUpdateIntent(ctx context.Context, i Identity) (*Work, error) {
	var out *Work
	e := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		out = nil
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
		// A reinstall cannot inherit another incarnation's work, including a job
		// that was applying before the identity changed (§13.8).
		_, e = tx.Exec(ctx, `UPDATE update_jobs SET state='superseded_incarnation',phase_since=clock_timestamp() WHERE node_id=$1 AND execution_incarnation_id IS NOT NULL AND execution_incarnation_id<>$2 AND state IN ('queued','waiting_online','downloading','verifying','applying','checking')`, i.NodeID, i.IncarnationID)
		if e != nil {
			return e
		}
		// Bind the latest unexecuted enrolling target once, then create attempt 1.
		if n.job == nil && n.target != nil {
			var release uuid.UUID
			e = tx.QueryRow(ctx, `UPDATE update_targets t SET execution_incarnation_id=$2 FROM update_batches b
 WHERE t.id=$1 AND t.batch_id=b.id AND t.target_state='active' AND t.target_generation=$3::numeric
 AND t.execution_incarnation_id IS NULL AND (t.incarnation_at_snapshot IS NULL OR t.incarnation_at_snapshot=$2)
 RETURNING b.target_release_id`, n.target, i.IncarnationID, n.generation.String()).Scan(&release)
			if e == nil {
				n.online = true
				if e = s.newJob(ctx, tx, n, n.target, release, n.generation, 1); e != nil {
					return e
				}
				n, e = s.readNode(ctx, tx, i.NodeID)
			}
			if e != nil && !errors.Is(e, pgx.ErrNoRows) {
				return e
			}
		}
		var current job
		if n.job != nil {
			current, e = readJob(ctx, tx, *n.job)
			if e != nil {
				return e
			}
		}
		if n.job == nil || current.incarnation != nil && *current.incarnation != i.IncarnationID {
			var desired *uuid.UUID
			if e = tx.QueryRow(ctx, `SELECT desired_release_id FROM fleet_config`).Scan(&desired); e != nil {
				return e
			}
			if desired == nil {
				return nil
			}
			if _, _, _, e = s.releases.Eligible(ctx, tx, *desired); e != nil {
				return e
			}
			// §17.4: an agent ahead of the restored central generation requires an
			// explicit new batch, never an automatic fleet-default downgrade.
			if n.observedGeneration > n.generation {
				return apierr.New(apierr.StaleGeneration, "Agent generation requires restore reconciliation")
			}
			gen, e := n.generation.Add(1)
			if e != nil {
				return e
			}
			n.online = true
			if e = s.firstSync(ctx, tx, n, *desired, gen); e != nil {
				return e
			}
			n, e = s.readNode(ctx, tx, i.NodeID)
			if e != nil {
				return e
			}
			current, e = readJob(ctx, tx, *n.job)
			if e != nil {
				return e
			}
			if e = notify(ctx, tx, i.NodeID, "fleet.first_sync_created"); e != nil {
				return e
			}
		}
		if n.observedGeneration > n.generation {
			return apierr.New(apierr.StaleGeneration, "Agent generation requires restore reconciliation")
		}
		// A disruptive older transaction finishes before the newest desired work.
		var applying uuid.UUID
		e = tx.QueryRow(ctx, `SELECT id FROM update_jobs WHERE node_id=$1 AND execution_incarnation_id=$2 AND state IN ('applying','checking') ORDER BY generation,attempt LIMIT 1`, i.NodeID, i.IncarnationID).Scan(&applying)
		if e == nil {
			current, e = readJob(ctx, tx, applying)
			if e != nil {
				return e
			}
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if terminal(current.state) {
			return recomputeAll(ctx, tx)
		}
		manifest, hash, url, e := s.releases.Eligible(ctx, tx, current.releaseID)
		if e != nil {
			return e
		}
		var artifact *api.ManifestArtifact
		for _, a := range manifest.Signed.Artifacts {
			if string(a.Architecture) == n.arch && (n.os == "ubuntu" || n.os == "ubuntu/24.04" || n.os == "ubuntu:24.04") && a.OsId == "ubuntu" && slices.Contains(a.OsVersions, "24.04") {
				a := a
				artifact = &a
				break
			}
		}
		if artifact == nil {
			return apierr.New(apierr.UnsupportedPlatform, "Release has no artifact for the authenticated node platform")
		}
		now := time.Now().UTC()
		until := now.Add(s.config.AuthorizationTTL)
		_, e = tx.Exec(ctx, `UPDATE update_jobs SET issued_at=coalesce(issued_at,$2),authorization_valid_until=$3,state=CASE WHEN state='waiting_online' THEN 'queued' ELSE state END WHERE id=$1`, current.id, now, until)
		if e != nil {
			return e
		}
		out = &Work{Intent: api.UpdateIntent{NodeId: i.NodeID, IncarnationId: i.IncarnationID, BatchId: presence.FromPointer(current.batch), TargetId: presence.FromPointer(current.target), JobId: current.id, Attempt: current.attempt, Generation: current.generation, ReleaseId: current.releaseID, ManifestSha256: hash, ManifestUrl: url, IssuedAt: now, AuthorizationValidUntil: until, PublicationIntent: api.PublicationIntent(n.publication)}, Artifact: *artifact}
		return recomputeAll(ctx, tx)
	})
	return out, e
}

func terminal(state string) bool {
	switch state {
	case "succeeded", "already_current", "failed", "rolled_back", "superseded", "superseded_incarnation", "cancelled_revoked", "cancelled_retired":
		return true
	}
	return false
}
