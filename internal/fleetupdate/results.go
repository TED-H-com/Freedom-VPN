package fleetupdate

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// SubmitResultTx joins update-result ingestion to the caller's command transaction (§14.5).
func (s *Service) SubmitResultTx(ctx context.Context, tx pgx.Tx, i Identity, raw []byte) (api.UpdateResultAck, error) {
	return s.SubmitResult(commandtx.WithTransaction(ctx, tx), i, raw)
}

func (s *Service) SubmitResult(ctx context.Context, i Identity, raw []byte) (api.UpdateResultAck, error) {
	ack, err := s.submitResult(ctx, i, raw)
	return ack, resultError(err)
}

func (s *Service) submitResult(ctx context.Context, i Identity, raw []byte) (api.UpdateResultAck, error) {
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
	// Raw schema validation above requires every member; null maps to SQL NULL.
	var batchID *uuid.UUID
	if value, err := r.BatchId.Get(); err == nil {
		batchID = &value
	}
	var targetID *uuid.UUID
	if value, err := r.TargetId.Get(); err == nil {
		targetID = &value
	}
	var currentReleaseID *uuid.UUID
	if value, err := r.CurrentReleaseId.Get(); err == nil {
		currentReleaseID = &value
	}
	var health *api.UpdateHealth
	if value, err := r.Health.Get(); err == nil {
		health = &value
	}
	var finishedAt *time.Time
	if value, err := r.FinishedAt.Get(); err == nil {
		finishedAt = &value
	}
	var usageCheckpoint *api.UpdateUsageCheckpoint
	if value, err := r.UsageCheckpoint.Get(); err == nil {
		usageCheckpoint = &value
	}
	var errorCode *api.ErrorCode
	if value, err := r.ErrorCode.Get(); err == nil {
		errorCode = &value
	}
	var stage *api.UpdateStage
	if value, err := r.Stage.Get(); err == nil {
		stage = &value
	}
	recoveryRequired, _ := r.RecoveryRequired.Get()
	hash, e := canonjson.HashHex(raw, "payload_hash")
	if e != nil || hash != r.PayloadHash {
		return ack, apierr.New(apierr.HashMismatch, "Result payload hash does not match")
	}
	digest, _ := hex.DecodeString(hash)
	requestID := commandtx.RequestID(ctx)
	var rejection error
	e = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
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
				var stored resultRejection
				if err := json.Unmarshal(oldAck, &stored); err != nil {
					return err
				}
				if stored.Error != nil {
					rejection = stored.apiError()
					return nil
				}
				return json.Unmarshal(oldAck, &ack)
			}
			_, e = tx.Exec(ctx, `INSERT INTO update_result_conflicts(result_id,conflicting_hash) VALUES($1,$2) ON CONFLICT DO NOTHING`, r.ResultId, digest)
			if e != nil {
				return e
			}
			rejection = apierr.New(apierr.AckConflict, "Result ID already has different content")
			event, e := audit.NewEvent(audit.Entry{
				Actor: audit.Actor{Type: api.AuditEventActorTypeNode, ID: i.NodeID}, Action: "update_result.conflict",
				Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: i.NodeID}, Result: api.AuditEventResultFailed, RequestID: &requestID,
				After: map[string]any{"result_id": r.ResultId, "conflicting_hash": hash}, Fields: []string{"result_id", "conflicting_hash"},
			})
			if e != nil {
				return e
			}
			return audit.AppendEvent(ctx, tx, event)
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

		// Rejected receipts commit without committing any attempted projection.
		// The savepoint also keeps SubmitResultTx inside the caller's transaction.
		effectTx, e := tx.Begin(ctx)
		if e != nil {
			return e
		}
		applyErr := func(tx pgx.Tx) error {
			if j.generation != r.Generation || j.attempt != r.Attempt || !equalID(j.batch, batchID) || !equalID(j.target, targetID) {
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
			if isTerminal && (currentReleaseID == nil || finishedAt == nil || finishedAt.Before(r.StartedAt)) {
				return apierr.New(apierr.ValidationFailed, "Terminal result requires actual version and completion time")
			}
			// D-062(a): a failed pre-health stage cannot invent checks or claim the
			// candidate as current. The baseline was captured at first authorization.
			preHealthFailure := r.State == api.UpdateResultStateFailed && beforeHealth(stage)
			if preHealthFailure {
				if health != nil || errorCode == nil || j.attemptRelease == nil || !equalID(j.attemptRelease, currentReleaseID) {
					return apierr.New(apierr.ValidationFailed, "Pre-health failure requires null health, an error and the unchanged attempt release")
				}
			} else if isTerminal && health == nil {
				return apierr.New(apierr.ValidationFailed, "Terminal result at or after health checks requires health evidence")
			}
			if usageCheckpoint != nil && usageCheckpoint.CentralAckedThrough > usageCheckpoint.LastDurableEventSeq {
				return apierr.New(apierr.CheckpointFailed, "Usage acknowledgement exceeds durable checkpoint")
			}
			success := r.State == "succeeded" || r.State == "already_current"
			if success && (currentReleaseID == nil || *currentReleaseID != j.releaseID || !healthy(health) || recoveryRequired || errorCode != nil || usageCheckpoint == nil) {
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
			recovery := recoveryRequired || n.lastGood != nil && slices.Contains(revoked, *n.lastGood) || currentReleaseID != nil && slices.Contains(revoked, *currentReleaseID)
			if r.State == "rolled_back" {
				previous, err := r.PreviousReleaseId.Get()
				if err != nil || previous != j.releaseID || errorCode == nil {
					return apierr.New(apierr.ValidationFailed, "Rollback requires the candidate release and its failure code")
				}
				if currentReleaseID == nil || !slices.Contains(serving, *currentReleaseID) || slices.Contains(revoked, *currentReleaseID) {
					_, e = tx.Exec(ctx, `UPDATE fleet_node_state SET recovery_required=true WHERE node_id=$1 AND $2`, i.NodeID, currentJob)
					if e != nil {
						return e
					}
					rejection = apierr.New(apierr.MinimumVersionViolation, "Rollback release may not serve")
					event, e := audit.NewEvent(audit.Entry{
						Actor: audit.Actor{Type: api.AuditEventActorTypeNode, ID: i.NodeID}, Action: "update_result.rollback_refused",
						Resource: api.ResourceRef{Type: api.ResourceTypeUpdateTarget, Id: *j.target}, Result: api.AuditEventResultFailed, RequestID: &requestID,
						After: map[string]any{"recovery_required": true, "result_id": r.ResultId, "job_id": j.id}, Fields: []string{"recovery_required", "result_id", "job_id"},
					})
					if e != nil {
						return e
					}
					return audit.AppendEvent(ctx, tx, event)
				}
				rollbackManifest, e := s.releases.Manifest(ctx, tx, *currentReleaseID)
				if e != nil {
					return e
				}
				// D-034e/D-062(c): serving authorization differs from approval to
				// target a new install. Still require a verified, trusted manifest.
				var verified bool
				if e = tx.QueryRow(ctx, `SELECT state='verified' FROM release_verifications WHERE release_id=$1`, *currentReleaseID).Scan(&verified); e != nil {
					return e
				}
				if !verified {
					return apierr.New(apierr.ReleaseNotApproved, "Rollback release is not verified")
				}
				rollbackRaw, e := json.Marshal(rollbackManifest)
				if e != nil {
					return e
				}
				if _, _, e = s.releases.Validator.ValidateManifest(rollbackRaw, time.Now().UTC()); e != nil {
					return e
				}
				if r.StateSchemaVersion != rollbackManifest.Signed.Compatibility.StateSchemaTo {
					return apierr.New(apierr.SchemaIncompatible, "Rollback state schema differs from the active release")
				}
				if !manifest.Signed.Rollback.StateReadableByPrevious || (len(manifest.Signed.Rollback.AllowedReleaseIds) > 0 && !slices.Contains(manifest.Signed.Rollback.AllowedReleaseIds, *currentReleaseID)) || *currentReleaseID == j.releaseID || !healthy(health) {
					return apierr.New(apierr.SchemaIncompatible, "Rollback lacks allowed release and health evidence")
				}
			}
			if r.MemberForwardingEnabled && (n.publication != "published" || r.PublicationIntent != "published" || r.State == "applying" || r.State == "checking" || recovery || currentReleaseID == nil || !slices.Contains(serving, *currentReleaseID) || slices.Contains(revoked, *currentReleaseID)) {
				return apierr.New(apierr.NodeNotServiceable, "Forwarding contradicts current serving restrictions")
			}
			history := n.job == nil || *n.job != j.id || j.generation != n.generation || terminal(j.state) || !canAdvance(j.state, string(r.State))
			// A superseded terminal ledger is never rewritten by a late successful ACK.
			// Applying/checking older jobs may finish their own ledger, but not the pointer.
			updateJob := !terminal(j.state) && canAdvance(j.state, string(r.State))
			if updateJob {
				if r.State == api.UpdateResultStateRolledBack && j.state != "applying" && j.state != "checking" {
					// §13.10.3 / AT26: a validated terminal rollback proves the
					// local apply happened even if no progress result was sent.
					// Traverse the base DB guard's required apply phase inside
					// this same savepoint, without a fabricated progress receipt
					// or a separately committed intermediate state.
					if _, e = tx.Exec(ctx, `UPDATE update_jobs SET state='applying' WHERE id=$1`, j.id); e != nil {
						return e
					}
				}
				_, e = tx.Exec(ctx, `UPDATE update_jobs SET state=$2,phase_since=CASE WHEN state<>$2 THEN clock_timestamp() ELSE phase_since END,needs_attention=($2='failed'),started_at=coalesce(started_at,clock_timestamp()),completed_at=CASE WHEN $3 THEN clock_timestamp() ELSE completed_at END,
 actual_release_id=$4,last_good_release_id=CASE WHEN $5 THEN $4 ELSE last_good_release_id END,last_result_at=clock_timestamp(),result_id=CASE WHEN $3 THEN $6 ELSE result_id END,result_hash=CASE WHEN $3 THEN $7 ELSE result_hash END,error_code=$8,observed_evidence=$9::jsonb||jsonb_build_object('attempt_release_id',observed_evidence->'attempt_release_id'),recovery_required=$10,stage=$11 WHERE id=$1`, j.id, string(r.State), isTerminal, currentReleaseID, success || r.State == "rolled_back", r.ResultId, digest, errorCode, raw, recovery, stage)
				if e != nil {
					return e
				}
			} else {
				if _, e = tx.Exec(ctx, `UPDATE update_jobs SET last_result_at=clock_timestamp() WHERE id=$1`, j.id); e != nil {
					return e
				}
			}
			applied := false
			if !history && isTerminal && currentReleaseID != nil {
				// §13.8: a pre-switch heartbeat may arrive after intent issuance.
				// Only a strictly newer update generation advances actual version;
				// same/newer-generation heartbeat evidence remains authoritative.
				tag, e := tx.Exec(ctx, `UPDATE nodes n SET current_release_id=$2,last_good_release_id=CASE WHEN $3 THEN $2 ELSE last_good_release_id END,resource_version=resource_version+1 FROM fleet_node_state s WHERE n.id=$1 AND s.node_id=n.id AND s.observed_update_generation<$4::numeric`, i.NodeID, currentReleaseID, success || r.State == "rolled_back", r.Generation.String())
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
			ack = api.UpdateResultAck{ResultId: r.ResultId, Accepted: true, AcceptedAsHistory: history, AppliedToCurrentObservation: applied, ObservedUpdateGeneration: n.observedHighWater, ReceivedAt: time.Now().UTC()}
			if applied {
				ack.ObservedUpdateGeneration = r.Generation
			}
			ackRaw, e := json.Marshal(ack)
			if e != nil {
				return e
			}
			_, e = tx.Exec(ctx, `INSERT INTO update_results(result_id,job_id,node_id,incarnation_id,payload_hash,payload,ack,policy_refs)
 VALUES($1,$2,$3,$4,$5,$6,$7,(SELECT policy_refs FROM desired_states WHERE node_id=$3 AND incarnation_id=$4 AND generation=$8::numeric))`, r.ResultId, j.id, i.NodeID, i.IncarnationID, digest, raw, ackRaw, r.PolicyGeneration.String())
			if e != nil {
				return e
			}
			event, e := audit.NewEvent(audit.Entry{
				Actor: audit.Actor{Type: api.AuditEventActorTypeNode, ID: i.NodeID}, Action: "update_result.accept",
				Resource: api.ResourceRef{Type: api.ResourceTypeUpdateTarget, Id: *j.target}, Result: api.AuditEventResultSucceeded, RequestID: &requestID,
				After: map[string]any{"state": r.State, "accepted_as_history": history, "result_id": r.ResultId, "job_id": j.id}, Fields: []string{"state", "accepted_as_history", "result_id", "job_id"},
			})
			if e != nil {
				return e
			}
			if e = audit.AppendEvent(ctx, tx, event); e != nil {
				return e
			}
			return recomputeAll(ctx, tx)
		}(effectTx)
		if applyErr != nil {
			if e := effectTx.Rollback(ctx); e != nil {
				return e
			}
			var apiError *apierr.Error
			if !errors.As(applyErr, &apiError) || apiError.HTTPStatus() < 400 || apiError.HTTPStatus() >= 500 {
				return applyErr
			}
			rejection = resultError(apiError)
		} else if e := effectTx.Commit(ctx); e != nil {
			return e
		}
		if rejection != nil {
			ack = api.UpdateResultAck{}
			return recordResultRejection(ctx, tx, i, r, raw, digest, rejection)
		}
		return nil
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
	case "superseded":
		return from == "queued" || from == "waiting_online" || from == "downloading" || from == "verifying"
	case "succeeded", "failed", "rolled_back", "superseded_incarnation", "cancelled_revoked", "cancelled_retired":
		// §13.8 / AT26: terminal evidence does not require prior progress
		// delivery. Identity/generation and health are validated before this.
		return true
	}
	rank := map[string]int{"queued": 0, "waiting_online": 0, "downloading": 1, "verifying": 2, "applying": 3, "checking": 4}
	a, ok := rank[from]
	b, ok2 := rank[to]
	return ok && ok2 && b >= a
}

// beforeHealth uses the shared contract stage enum in §13.6 execution order.
// Unknown/null stages are never evidence that health checks have not begun.
func beforeHealth(stage *api.UpdateStage) bool {
	if stage == nil {
		return false
	}
	switch *stage {
	case api.UpdateStageClaim, api.UpdateStageDownload, api.UpdateStageVerify,
		api.UpdateStagePreflight, api.UpdateStagePrepare, api.UpdateStageBlockForwarding,
		api.UpdateStageFinalSample, api.UpdateStageStopServices, api.UpdateStageMigrate,
		api.UpdateStageSwitch, api.UpdateStageStart:
		return true
	}
	return false
}
