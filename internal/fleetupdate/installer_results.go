package fleetupdate

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/installerjob"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// acceptInstallerResult runs after schema/hash/mTLS/current-incarnation checks,
// under the same fleet and node locks as central results. D-089 deliberately
// gives this path no job, target, batch, confirmation or generation writes.
func (s *Service) acceptInstallerResult(ctx context.Context, tx pgx.Tx, n node, i Identity, r api.UpdateResult, raw, digest []byte) (api.UpdateResultAck, error) {
	var ack api.UpdateResultAck
	var issued bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM update_jobs WHERE id=$1)`, r.JobId).Scan(&issued); err != nil {
		return ack, err
	}
	if issued {
		return ack, apierr.New(apierr.StateConflict, "Installer evidence cannot refer to a central job")
	}
	var oldHash string
	var oldAck []byte
	err := tx.QueryRow(ctx, `SELECT encode(payload_hash,'hex'),ack FROM installer_update_results WHERE node_id=$1 AND job_id=$2`, i.NodeID, r.JobId).Scan(&oldHash, &oldAck)
	if err == nil {
		if oldHash != r.PayloadHash {
			return ack, apierr.New(apierr.AckConflict, "Installer job already has different evidence")
		}
		err = json.Unmarshal(oldAck, &ack)
		return ack, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ack, err
	}
	if !r.BatchId.IsNull() || !r.TargetId.IsNull() || r.Attempt != 1 {
		return ack, apierr.New(apierr.StateConflict, "Installer evidence requires attempt 1 without a batch or target")
	}
	var exists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM releases WHERE id=$1)`, r.RequestedReleaseId).Scan(&exists); err != nil {
		return ack, err
	}
	if !exists {
		return ack, apierr.New(apierr.ResourceNotFound, "Installer release does not exist")
	}
	// The release catalog has no separate publication flag: verified, approved,
	// supported and nonrevoked is its existing publication/targeting boundary.
	manifest, hash, _, err := s.releases.Eligible(ctx, tx, r.RequestedReleaseId)
	if err != nil {
		var ae *apierr.Error
		if errors.As(err, &ae) && ae.Code == apierr.ReleaseNotApproved {
			return ack, apierr.New(apierr.StateConflict, "Installer release is not published or is revoked")
		}
		return ack, err
	}
	if hash != r.ManifestSha256 {
		return ack, apierr.New(apierr.HashMismatch, "Installer result does not match published release content")
	}
	if r.JobId != installerjob.ID(r.RequestedReleaseId, hash, i.IncarnationID) {
		return ack, apierr.New(apierr.StateConflict, "Installer job ID does not match release and incarnation").WithDetail("reason", "job_id_mismatch")
	}
	current, currentErr := r.CurrentReleaseId.Get()
	baseline := n.current
	success := r.State == api.UpdateResultStateSucceeded || r.State == api.UpdateResultStateAlreadyCurrent
	stage, _ := r.Stage.Get()
	// D-111: a verified installer seed may precede any central observation.
	// Only a distinct, eligible baseline can explain a failed attempt; a
	// failure never establishes the candidate as current or last-good.
	seed := n.current == nil && currentErr == nil && (success || r.State == api.UpdateResultStateRolledBack ||
		r.State == api.UpdateResultStateFailed && beforeHealth(&stage) && current != r.RequestedReleaseId)
	if seed {
		if _, _, _, err = s.releases.Eligible(ctx, tx, current); err != nil {
			return ack, err
		}
		baseline = &current
	}
	if err = validateCompletionEvidence(r, r.RequestedReleaseId, baseline); err != nil {
		return ack, err
	}
	if success && r.StateSchemaVersion != manifest.Signed.Compatibility.StateSchemaTo {
		return ack, apierr.New(apierr.SchemaIncompatible, "Completed state schema differs from manifest")
	}
	var serving, revoked []uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT supported_release_ids,revoked_release_ids FROM fleet_config`).Scan(&serving, &revoked); err != nil {
		return ack, err
	}
	if r.State == api.UpdateResultStateRolledBack {
		previous, e := r.PreviousReleaseId.Get()
		if e != nil || previous != r.RequestedReleaseId || r.ErrorCode.IsNull() {
			return ack, apierr.New(apierr.ValidationFailed, "Rollback requires the candidate release and its failure code")
		}
		if !slices.Contains(serving, current) || slices.Contains(revoked, current) {
			return ack, apierr.New(apierr.MinimumVersionViolation, "Rollback release may not serve")
		}
		if err = s.validateRollbackEvidence(ctx, tx, manifest, r); err != nil {
			return ack, err
		}
	}
	recovery, _ := r.RecoveryRequired.Get()
	lastGood := n.lastGood
	if success || r.State == api.UpdateResultStateRolledBack || seed {
		lastGood = &current
	}
	recovery = recovery || lastGood != nil && slices.Contains(revoked, *lastGood) || slices.Contains(revoked, current)
	if r.MemberForwardingEnabled && (n.publication != "published" || r.PublicationIntent != "published" || r.State == "applying" || r.State == "checking" || recovery || !slices.Contains(serving, current) || slices.Contains(revoked, current)) {
		return ack, apierr.New(apierr.NodeNotServiceable, "Forwarding contradicts current serving restrictions")
	}
	// Installer evidence never advances the central generation high-water,
	// and an older receipt cannot replace a newer central observation (§13.8).
	applied := (success || r.State == api.UpdateResultStateRolledBack || seed) &&
		n.observedHighWater <= r.Generation && (!equalID(n.current, &current) || !equalID(n.lastGood, &current))
	before, err := nodes.ReadUpdateSummary(ctx, tx, i.NodeID)
	if err != nil {
		return ack, err
	}
	history := r.Generation < n.observedHighWater
	ack = api.UpdateResultAck{ResultId: r.ResultId, Accepted: true, AcceptedAsHistory: history, AppliedToCurrentObservation: applied, ObservedUpdateGeneration: n.observedHighWater, ReceivedAt: time.Now().UTC().Truncate(time.Microsecond)}
	ackRaw, err := json.Marshal(ack)
	if err != nil {
		return api.UpdateResultAck{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO installer_update_results(node_id,job_id,incarnation_id,release_id,state,payload,payload_hash,ack,received_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, i.NodeID, r.JobId, i.IncarnationID, r.RequestedReleaseId, r.State, raw, digest, ackRaw, ack.ReceivedAt)
	if err != nil {
		return api.UpdateResultAck{}, err
	}
	after, err := nodes.ReadUpdateSummary(ctx, tx, i.NodeID)
	if err != nil {
		return api.UpdateResultAck{}, err
	}
	if applied || !reflect.DeepEqual(before, after) {
		_, err = tx.Exec(ctx, `UPDATE nodes SET current_release_id=CASE WHEN $3 THEN $2 ELSE current_release_id END,last_good_release_id=CASE WHEN $3 THEN $2 ELSE last_good_release_id END,resource_version=resource_version+1 WHERE id=$1`, i.NodeID, current, applied)
		if err != nil {
			return api.UpdateResultAck{}, err
		}
	}
	requestID := commandtx.RequestID(ctx)
	event, err := audit.NewEvent(audit.Entry{
		Actor: audit.Actor{Type: api.AuditEventActorTypeNode, ID: i.NodeID}, Action: "update_result.accept",
		Resource: api.ResourceRef{Type: api.ResourceTypeNode, Id: i.NodeID}, Result: api.AuditEventResultSucceeded, RequestID: &requestID,
		After: map[string]any{"state": r.State, "accepted_as_history": history, "result_id": r.ResultId, "job_id": r.JobId, "source": api.UpdateResultSourceInstaller}, Fields: []string{"state", "accepted_as_history", "result_id", "job_id", "source"},
	})
	if err == nil {
		err = audit.AppendEvent(ctx, tx, event)
	}
	return ack, err
}
