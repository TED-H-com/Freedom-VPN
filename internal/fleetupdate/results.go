package fleetupdate

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/jackc/pgx/v5"
)

func (s *Service) SubmitResult(ctx context.Context, i Identity, raw []byte) (api.UpdateResultAck, error) {
	var ack api.UpdateResultAck
	if e := s.releases.Validator.ValidateBody("UpdateResult", raw); e != nil {
		return ack, e
	}
	var r api.UpdateResult
	if e := json.Unmarshal(raw, &r); e != nil {
		return ack, apierr.New(apierr.ValidationFailed, "Invalid result integers")
	}
	if r.NodeId != i.NodeID || r.IncarnationId != i.IncarnationID {
		return ack, apierr.New(apierr.IncarnationMismatch, "Result identity differs from mTLS identity")
	}
	hash, e := canonjson.HashHex(raw, "payload_hash")
	if e != nil || hash != r.PayloadHash {
		return ack, apierr.New(apierr.HashMismatch, "Result payload hash does not match")
	}
	digest, _ := hex.DecodeString(hash)
	var rejection error
	e = db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		rejection = nil
		ack = api.UpdateResultAck{}
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
		var oldHash string
		var oldAck []byte
		var sameIdentity bool
		e = tx.QueryRow(ctx, `SELECT encode(payload_hash,'hex'),ack,node_id=$2 AND incarnation_id=$3 FROM update_results WHERE result_id=$1`, r.ResultId, i.NodeID, i.IncarnationID).Scan(&oldHash, &oldAck, &sameIdentity)
		if e == nil {
			if !sameIdentity {
				return apierr.New(apierr.IncarnationMismatch, "Result ID belongs to another identity")
			}
			if oldHash == hash {
				return json.Unmarshal(oldAck, &ack)
			}
			_, e = tx.Exec(ctx, `INSERT INTO update_result_conflicts(result_id,conflicting_hash) VALUES($1,$2) ON CONFLICT DO NOTHING`, r.ResultId, digest)
			if e != nil {
				return e
			}
			rejection = apierr.New(apierr.AckConflict, "Result ID already has different content")
			return releases.Audit(ctx, tx, i.NodeID.String(), "agent", "update_result.conflict", r.ResultId.String(), nil, map[string]any{"result_id": r.ResultId, "conflicting_hash": hash})
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		j, e := readJob(ctx, tx, r.JobId)
		if e != nil {
			return e
		}
		if j.nodeID != i.NodeID || j.incarnation == nil || *j.incarnation != i.IncarnationID {
			return apierr.New(apierr.IncarnationMismatch, "Job is not bound to this incarnation")
		}
		if j.generation != r.Generation || j.attempt != r.Attempt || !equalID(j.batch, r.BatchId) || !equalID(j.target, r.TargetId) {
			return apierr.New(apierr.StaleGeneration, "Result does not match the authorized job attempt")
		}
		if r.RequestedReleaseId != j.releaseID || r.ManifestSha256 != j.manifestHash {
			return apierr.New(apierr.HashMismatch, "Result does not match authorized release content")
		}
		if j.issued == nil || j.validUntil == nil {
			return apierr.New(apierr.AuthorizationExpired, "Job was never authorized")
		}
		isTerminal := terminal(string(r.State))
		if time.Now().After(*j.validUntil) && !isTerminal && j.state != "applying" && j.state != "checking" {
			return apierr.New(apierr.AuthorizationExpired, "Refresh the expired update authorization")
		}
		if !r.StartedAt.Before(*j.validUntil) || r.StartedAt.Before(*j.issued) {
			return apierr.New(apierr.AuthorizationExpired, "Transaction started after authorization expired")
		}
		if isTerminal && (r.CurrentReleaseId == nil || r.Health == nil || r.FinishedAt == nil || r.FinishedAt.Before(r.StartedAt)) {
			return apierr.New(apierr.ValidationFailed, "Terminal result requires actual version, checks and completion time")
		}
		if r.UsageCheckpoint != nil && r.UsageCheckpoint.CentralAckedThrough > r.UsageCheckpoint.LastDurableEventSeq {
			return apierr.New(apierr.CheckpointFailed, "Usage acknowledgement exceeds durable checkpoint")
		}
		success := r.State == "succeeded" || r.State == "already_current"
		if success && (r.CurrentReleaseId == nil || *r.CurrentReleaseId != j.releaseID || !healthy(r.Health) || r.RecoveryRequired || r.ErrorCode != nil || r.UsageCheckpoint == nil) {
			return apierr.New(apierr.HealthCheckFailed, "Target completion requires version, health and durable checkpoint evidence")
		}
		manifest, e := s.releases.Manifest(ctx, tx, j.releaseID)
		if e != nil {
			return e
		}
		if success && r.StateSchemaVersion != manifest.Signed.Compatibility.StateSchemaTo {
			return apierr.New(apierr.SchemaIncompatible, "Completed state schema differs from manifest")
		}
		var serving, revoked []apiID
		if e = tx.QueryRow(ctx, `SELECT supported_release_ids,revoked_release_ids FROM fleet_config`).Scan(&serving, &revoked); e != nil {
			return e
		}
		currentJob := n.job != nil && *n.job == j.id && j.generation == n.generation
		recovery := r.RecoveryRequired || n.lastGood != nil && slices.Contains(revoked, *n.lastGood) || r.CurrentReleaseId != nil && slices.Contains(revoked, *r.CurrentReleaseId)
		if r.State == "rolled_back" {
			if r.CurrentReleaseId == nil || !slices.Contains(serving, *r.CurrentReleaseId) || slices.Contains(revoked, *r.CurrentReleaseId) {
				_, e = tx.Exec(ctx, `UPDATE fleet_node_state SET recovery_required=true WHERE node_id=$1 AND $2`, i.NodeID, currentJob)
				if e != nil {
					return e
				}
				rejection = apierr.New(apierr.MinimumVersionViolation, "Rollback release may not serve")
				return releases.Audit(ctx, tx, i.NodeID.String(), "agent", "update_result.rollback_refused", j.id.String(), nil, map[string]any{"recovery_required": true})
			}
			rollbackManifest, _, _, e := s.releases.Eligible(ctx, tx, *r.CurrentReleaseId)
			if e != nil {
				return e
			}
			if r.StateSchemaVersion != rollbackManifest.Signed.Compatibility.StateSchemaTo {
				return apierr.New(apierr.SchemaIncompatible, "Rollback state schema differs from the active release")
			}
			if !manifest.Signed.Rollback.StateReadableByPrevious || !slices.Contains(manifest.Signed.Rollback.AllowedReleaseIds, *r.CurrentReleaseId) || *r.CurrentReleaseId == j.releaseID || !healthy(r.Health) {
				return apierr.New(apierr.SchemaIncompatible, "Rollback lacks allowed release and health evidence")
			}
		}
		if r.MemberForwardingEnabled && (n.publication != "published" || r.PublicationIntent != "published" || r.State == "applying" || r.State == "checking" || recovery || r.CurrentReleaseId == nil || !slices.Contains(serving, *r.CurrentReleaseId) || slices.Contains(revoked, *r.CurrentReleaseId)) {
			return apierr.New(apierr.NodeNotServiceable, "Forwarding contradicts current serving restrictions")
		}
		history := n.job == nil || *n.job != j.id || j.generation != n.generation || terminal(j.state) || !canAdvance(j.state, string(r.State))
		// A superseded terminal ledger is never rewritten by a late successful ACK.
		// Applying/checking older jobs may finish their own ledger, but not the pointer.
		updateJob := !terminal(j.state) && canAdvance(j.state, string(r.State))
		if updateJob {
			_, e = tx.Exec(ctx, `UPDATE update_jobs SET state=$2,phase_since=CASE WHEN state<>$2 THEN clock_timestamp() ELSE phase_since END,needs_attention=false,started_at=coalesce(started_at,clock_timestamp()),completed_at=CASE WHEN $3 THEN clock_timestamp() ELSE completed_at END,
 actual_release_id=$4,last_good_release_id=CASE WHEN $5 THEN $4 ELSE last_good_release_id END,last_result_at=clock_timestamp(),result_id=CASE WHEN $3 THEN $6 ELSE result_id END,result_hash=CASE WHEN $3 THEN $7 ELSE result_hash END,error_code=$8,observed_evidence=$9,recovery_required=$10,stage=$11 WHERE id=$1`, j.id, string(r.State), isTerminal, r.CurrentReleaseId, success || r.State == "rolled_back", r.ResultId, digest, r.ErrorCode, raw, recovery, r.Stage)
			if e != nil {
				return e
			}
		} else {
			if _, e = tx.Exec(ctx, `UPDATE update_jobs SET last_result_at=clock_timestamp() WHERE id=$1`, j.id); e != nil {
				return e
			}
		}
		applied := false
		if !history && isTerminal && r.CurrentReleaseId != nil {
			// Result time is not an observation sequence. Once a heartbeat arrived
			// after authorization, wait for another heartbeat rather than replacing it.
			tag, e := tx.Exec(ctx, `UPDATE nodes n SET current_release_id=$2,last_good_release_id=CASE WHEN $3 THEN $2 ELSE last_good_release_id END,resource_version=resource_version+1 FROM fleet_node_state s WHERE n.id=$1 AND s.node_id=n.id AND (s.received_at IS NULL OR s.received_at<=$4) AND s.observed_update_generation<=$5::numeric`, i.NodeID, r.CurrentReleaseId, success || r.State == "rolled_back", j.issued, r.Generation.String())
			if e != nil {
				return e
			}
			applied = tag.RowsAffected() == 1
			if applied {
				_, e = tx.Exec(ctx, `UPDATE fleet_node_state SET observed_update_generation=$2::numeric WHERE node_id=$1`, i.NodeID, r.Generation.String())
				if e != nil {
					return e
				}
			}
		}
		if recovery && !history {
			if _, e = tx.Exec(ctx, `UPDATE fleet_node_state SET recovery_required=true WHERE node_id=$1`, i.NodeID); e != nil {
				return e
			}
		}
		ack = api.UpdateResultAck{ResultId: r.ResultId, Accepted: true, AcceptedAsHistory: history, AppliedToCurrentObservation: applied, ObservedUpdateGeneration: n.observedGeneration, ReceivedAt: time.Now().UTC()}
		if applied {
			ack.ObservedUpdateGeneration = r.Generation
		}
		ackRaw, e := json.Marshal(ack)
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `INSERT INTO update_results(result_id,job_id,node_id,incarnation_id,payload_hash,payload,ack) VALUES($1,$2,$3,$4,$5,$6,$7)`, r.ResultId, j.id, i.NodeID, i.IncarnationID, digest, raw, ackRaw)
		if e != nil {
			return e
		}
		if e = releases.Audit(ctx, tx, i.NodeID.String(), "agent", "update_result.accept", j.id.String(), nil, map[string]any{"state": r.State, "accepted_as_history": history, "result_id": r.ResultId}); e != nil {
			return e
		}
		return recomputeAll(ctx, tx)
	})
	if e != nil {
		return ack, e
	}
	return ack, rejection
}

func healthy(h *api.UpdateHealth) bool {
	return h != nil && h.Processes == "passed" && h.IdentityAndState == "passed" && h.InternetOnly == "passed" && h.DnsPolicy == "passed" && h.TunnelProbe == "passed"
}
func canAdvance(from, to string) bool {
	if terminal(from) {
		return false
	}
	switch to {
	case "already_current":
		return from == "queued" || from == "waiting_online"
	case "rolled_back":
		return from == "applying" || from == "checking"
	case "superseded":
		return from == "queued" || from == "waiting_online" || from == "downloading" || from == "verifying"
	case "succeeded", "failed", "superseded_incarnation", "cancelled_revoked", "cancelled_retired":
		return true
	}
	rank := map[string]int{"queued": 0, "waiting_online": 0, "downloading": 1, "verifying": 2, "applying": 3, "checking": 4}
	a, ok := rank[from]
	b, ok2 := rank[to]
	return ok && ok2 && b >= a
}
