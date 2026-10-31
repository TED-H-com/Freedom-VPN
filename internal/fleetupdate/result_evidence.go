package fleetupdate

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// validateCompletionEvidence applies the same terminal evidence rules to
// central work and installer evidence. The baseline is centrally known or an
// eligible installer seed verified by the caller (D-111).
func validateCompletionEvidence(r api.UpdateResult, release uuid.UUID, baseline *uuid.UUID) error {
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
	isTerminal := terminal(string(r.State))
	if isTerminal && (currentReleaseID == nil || finishedAt == nil || finishedAt.Before(r.StartedAt)) {
		return apierr.New(apierr.ValidationFailed, "Terminal result requires actual version and completion time")
	}
	if r.State == api.UpdateResultStateFailed && errorCode == nil {
		return apierr.New(apierr.ValidationFailed, "Failed result requires an error code")
	}
	// D-062(a): a failed pre-health stage cannot invent checks or claim the
	// candidate as current. The baseline was captured at first authorization.
	preHealthFailure := r.State == api.UpdateResultStateFailed && beforeHealth(stage)
	if preHealthFailure {
		if health != nil || baseline == nil || !equalID(baseline, currentReleaseID) {
			return apierr.New(apierr.ValidationFailed, "Pre-health failure requires null health, an error and the unchanged attempt release")
		}
	} else if isTerminal && health == nil {
		return apierr.New(apierr.ValidationFailed, "Terminal result at or after health checks requires health evidence")
	}
	if usageCheckpoint != nil && usageCheckpoint.CentralAckedThrough > usageCheckpoint.LastDurableEventSeq {
		return apierr.New(apierr.CheckpointFailed, "Usage acknowledgement exceeds durable checkpoint")
	}
	success := r.State == "succeeded" || r.State == "already_current"
	if success && (currentReleaseID == nil || *currentReleaseID != release || !healthy(health) || recoveryRequired || errorCode != nil || usageCheckpoint == nil) {
		return apierr.New(apierr.HealthCheckFailed, "Target completion requires version, health and durable checkpoint evidence")
	}
	return nil
}

// validateRollbackEvidence verifies the restored release, independently of job
// projection. Callers separately enforce the current serving allowlist.
func (s *Service) validateRollbackEvidence(ctx context.Context, tx pgx.Tx, manifest api.ReleaseManifestEnvelope, r api.UpdateResult) error {
	current, _ := r.CurrentReleaseId.Get()
	currentReleaseID := &current
	h, _ := r.Health.Get()
	health := &h
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
	if !manifest.Signed.Rollback.StateReadableByPrevious || (len(manifest.Signed.Rollback.AllowedReleaseIds) > 0 && !slices.Contains(manifest.Signed.Rollback.AllowedReleaseIds, *currentReleaseID)) || *currentReleaseID == r.RequestedReleaseId || !healthy(health) {
		return apierr.New(apierr.SchemaIncompatible, "Rollback lacks allowed release and health evidence")
	}
	return nil
}
