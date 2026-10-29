package updater

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// continueTxn runs the journaled steps from the durable stage (§13.6 steps
// 5-14, §13.9). The hook fires after the stage is durable and before its
// side effect, so a crash resumes that side effect.
func (e *Engine) continueTxn(ctx context.Context, txn updatejournal.Txn) error {
	if txn.Stage != updatejournal.StagePrepared && e.Gate != nil {
		if err := e.Gate.Block(ctx); err != nil {
			return transient(apierr.UpdateBusy, "re-block")
		}
	}
	var err error
	for !txn.Stage.Terminal() {
		if err = e.hookStage(string(txn.Stage)); err != nil {
			return err
		}
		txn, err = e.perform(ctx, txn)
		if err != nil {
			return err
		}
	}
	return e.seal(ctx, txn)
}

func (e *Engine) perform(ctx context.Context, txn updatejournal.Txn) (updatejournal.Txn, error) {
	c, err := e.claimedFor(ctx, txn)
	if err != nil {
		return txn, err
	}
	if txn.Stage != updatejournal.StageRollingBack && e.drifted(c) {
		if txn.Destructive {
			return e.toRecovery(ctx, txn, apierr.IncarnationMismatch, api.UpdateStageRollback, nil)
		}
		return e.beginRollback(ctx, txn, apierr.IncarnationMismatch, api.UpdateStageClaim, nil)
	}
	switch txn.Stage {
	case updatejournal.StagePrepared:
		return e.stepBlock(ctx, txn)
	case updatejournal.StageBlocked:
		return e.stepSample(ctx, txn)
	case updatejournal.StageFinalSampled:
		return e.stepStop(ctx, txn)
	case updatejournal.StageStopped:
		return e.stepMigrate(ctx, txn)
	case updatejournal.StageMigrated:
		return e.stepSwitch(ctx, txn)
	case updatejournal.StageSwitched:
		return e.stepStart(ctx, txn)
	case updatejournal.StageChecking:
		return e.stepCheck(ctx, txn)
	case updatejournal.StageRollingBack:
		return e.stepRollback(ctx, txn, c)
	default:
		return txn, permanent(apierr.RollbackFailed, "unknown stage "+string(txn.Stage))
	}
}

func (e *Engine) stepBlock(ctx context.Context, txn updatejournal.Txn) (updatejournal.Txn, error) {
	if e.Gate == nil {
		return txn, errors.New("updater: no forwarding gate")
	}
	if err := e.Gate.Block(ctx); err != nil {
		// Do not end the transaction. The next start retries the block (§13.6 step 7).
		return txn, transient(apierr.UpdateBusy, "block")
	}
	return e.advance(ctx, txn, updatejournal.StageBlocked, updatejournal.StageUpdate{})
}

func (e *Engine) stepSample(ctx context.Context, txn updatejournal.Txn) (updatejournal.Txn, error) {
	if e.Meter == nil {
		return e.beginRollback(ctx, txn, apierr.CheckpointFailed, api.UpdateStageFinalSample, nil)
	}
	cp, err := e.Meter.FinalSample(ctx)
	if err != nil {
		return e.beginRollback(ctx, txn, apierr.CheckpointFailed, api.UpdateStageFinalSample, nil)
	}
	apiCP, err := checkpointFromUsage(cp, "")
	if err != nil {
		return e.beginRollback(ctx, txn, apierr.CheckpointFailed, api.UpdateStageFinalSample, nil)
	}
	return e.advance(ctx, txn, updatejournal.StageFinalSampled, updatejournal.StageUpdate{Checkpoint: apiCP})
}

func schemaChange(txn updatejournal.Txn) bool {
	return txn.PreviousReleaseID != "" && txn.StateSchemaFrom > 0 && txn.StateSchemaFrom != txn.StateSchemaTo
}

func (e *Engine) stepStop(ctx context.Context, txn updatejournal.Txn) (updatejournal.Txn, error) {
	same, err := fsutil.SameFilesystem(e.Layout.Releases, filepath.Dir(e.Layout.Current))
	if err != nil || !same {
		return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageStopServices, nil)
	}
	// A release that needs a schema change and has no real migrator fails
	// before services stop, so the rollback is not destructive (D-034c).
	if schemaChange(txn) && e.migrationUnsupported() {
		return e.beginRollback(ctx, txn, apierr.MigrationFailed, api.UpdateStageMigrate, nil)
	}
	if e.Services == nil {
		return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageStopServices, nil)
	}
	if err := e.Services.Stop(ctx); err != nil {
		// The stop may have been partial. Treat it as destructive so rollback
		// restarts last_good instead of assuming the old process is intact.
		// There is no STOP_FAILED code (§13.11); this is reported as START_FAILED.
		return e.rollbackDestructive(ctx, txn, apierr.StartFailed, api.UpdateStageStopServices)
	}
	return e.advance(ctx, txn, updatejournal.StageStopped, updatejournal.StageUpdate{Destructive: true})
}

func (e *Engine) stepMigrate(ctx context.Context, txn updatejournal.Txn) (updatejournal.Txn, error) {
	if e.Services != nil {
		if err := e.Services.Stop(ctx); err != nil {
			return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageStopServices, nil)
		}
	}
	if txn.PreviousReleaseID != "" && txn.StateSchemaFrom > 0 && txn.StateSchemaFrom != txn.StateSchemaTo {
		if e.Migrator == nil {
			return e.beginRollback(ctx, txn, apierr.MigrationFailed, api.UpdateStageMigrate, nil)
		}
		err := e.Migrator.Migrate(ctx, txn.StateSchemaFrom, txn.StateSchemaTo, txn.MigrationID, e.Layout.State)
		if errors.Is(err, ErrStateUnsafe) {
			return e.toRecovery(ctx, txn, apierr.SchemaIncompatible, api.UpdateStageMigrate, nil)
		}
		if err != nil {
			return e.beginRollback(ctx, txn, apierr.MigrationFailed, api.UpdateStageMigrate, nil)
		}
	}
	return e.advance(ctx, txn, updatejournal.StageMigrated, updatejournal.StageUpdate{})
}

func (e *Engine) stepSwitch(ctx context.Context, txn updatejournal.Txn) (updatejournal.Txn, error) {
	dir := e.Layout.releaseDir(txn.ReleaseID)
	if err := fsutil.SyncTree(dir); err != nil {
		return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageSwitch, nil)
	}
	rel, err := e.Layout.relativeRelease(txn.ReleaseID)
	if err != nil {
		return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageSwitch, nil)
	}
	if err := fsutil.SwitchSymlink(e.Layout.Current, rel); err != nil {
		return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageSwitch, nil)
	}
	if err := e.Journal.SetPointers(ctx, updatejournal.Pointers{CurrentRelease: txn.ReleaseID}); err != nil {
		return txn, err
	}
	_ = fsutil.SyncDir(e.Layout.Journal)
	return e.advance(ctx, txn, updatejournal.StageSwitched, updatejournal.StageUpdate{})
}

func (e *Engine) stepStart(ctx context.Context, txn updatejournal.Txn) (updatejournal.Txn, error) {
	if e.Services == nil {
		return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageStart, nil)
	}
	// Snapshot the immutable set after migration and before the new
	// services or the meter epoch run (D-034c).
	if err := e.snapshotIdentity(); err != nil {
		return e.beginRollback(ctx, txn, apierr.CheckpointFailed, api.UpdateStageStart, nil)
	}
	if err := e.snapshotPointers(ctx); err != nil {
		return e.beginRollback(ctx, txn, apierr.CheckpointFailed, api.UpdateStageStart, nil)
	}
	if err := e.Services.Start(ctx, e.Layout.releaseDir(txn.ReleaseID)); err != nil {
		return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageStart, nil)
	}
	epoch, err := e.openEpoch(ctx, txn.TxnID)
	if err != nil {
		return e.beginRollback(ctx, txn, apierr.CheckpointFailed, api.UpdateStageStart, nil)
	}
	cp, _ := decodeCheckpoint(txn.Checkpoint)
	if cp == nil {
		cp = &api.UpdateUsageCheckpoint{
			ClosedMeterEpochId: presence.NewNullNullable[uuid.UUID](),
			NewMeterEpochId:    presence.NewNullNullable[uuid.UUID](),
		}
	}
	if id, err := uuid.Parse(epoch); err == nil {
		cp.NewMeterEpochId.Set(id)
	}
	_ = e.Journal.SetJobPhase(ctx, txn.JobID, txn.Attempt, updatejournal.JobChecking)
	return e.advance(ctx, txn, updatejournal.StageChecking, updatejournal.StageUpdate{Checkpoint: cp})
}

func (e *Engine) stepCheck(ctx context.Context, txn updatejournal.Txn) (updatejournal.Txn, error) {
	n, err := bumpCounter(filepath.Join(e.Layout.Journal, "checks-"+txn.TxnID))
	if err != nil {
		return txn, err
	}
	if n > maxCheckAttempts {
		return e.beginRollback(ctx, txn, apierr.HealthCheckFailed, api.UpdateStageCheck, nil)
	}
	if e.Services != nil {
		if err := e.Services.Start(ctx, e.Layout.releaseDir(txn.ReleaseID)); err != nil {
			return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageStart, nil)
		}
	}
	manifest := e.manifestOf(ctx, txn.ReleaseID)
	view := e.viewFor(ctx, txn.ReleaseID, manifest)
	h, ok := e.runHealth(ctx, view)
	_ = e.Journal.RecordHealth(ctx, txn.TxnID, h)
	if !ok {
		return e.beginRollback(ctx, txn, apierr.HealthCheckFailed, api.UpdateStageCheck, &h)
	}
	if err := e.Journal.SetPointers(ctx, updatejournal.Pointers{LastGood: txn.ReleaseID}); err != nil {
		return txn, err
	}
	schema := txn.StateSchemaTo
	if schema < 1 {
		schema = schemaAtLeast1(e.Platform.StateSchema)
	}
	if err := e.Journal.SetStateSchema(ctx, schema); err != nil {
		return txn, err
	}
	return e.advance(ctx, txn, updatejournal.StageLocallySucceeded, updatejournal.StageUpdate{Health: h})
}

func (e *Engine) stepRollback(ctx context.Context, txn updatejournal.Txn, c claimed) (updatejournal.Txn, error) {
	n, err := bumpCounter(filepath.Join(e.Layout.Journal, "rollback-"+txn.TxnID))
	if err != nil {
		return txn, err
	}
	if n > maxRollbackAttempts {
		return e.toRecovery(ctx, txn, apierr.RollbackFailed, api.UpdateStageRollback, nil)
	}
	in := e.evalLastGood(ctx, txn.LastGoodReleaseID, c, e.manifestOf(ctx, txn.ReleaseID))
	plan := DecideRollback(in)
	if !plan.Restore {
		return e.toRecovery(ctx, txn, plan.Code, api.UpdateStageRollback, nil)
	}
	if txn.Destructive {
		rel, err := e.Layout.relativeRelease(txn.LastGoodReleaseID)
		if err != nil {
			return e.toRecovery(ctx, txn, apierr.RollbackFailed, api.UpdateStageRollback, nil)
		}
		if err := fsutil.SyncTree(e.Layout.releaseDir(txn.LastGoodReleaseID)); err != nil {
			return e.toRecovery(ctx, txn, apierr.RollbackFailed, api.UpdateStageRollback, nil)
		}
		if err := fsutil.SwitchSymlink(e.Layout.Current, rel); err != nil {
			return e.toRecovery(ctx, txn, apierr.RollbackFailed, api.UpdateStageRollback, nil)
		}
		if err := e.Journal.SetPointers(ctx, updatejournal.Pointers{CurrentRelease: txn.LastGoodReleaseID}); err != nil {
			return txn, err
		}
		if e.Services != nil {
			_ = e.Services.Stop(ctx)
			if err := e.Services.Start(ctx, e.Layout.releaseDir(txn.LastGoodReleaseID)); err != nil {
				return e.toRecovery(ctx, txn, apierr.StartFailed, api.UpdateStageRollback, nil)
			}
		}
		// Pointers now name last_good. Refresh that half of the snapshot
		// and keep the identity snapshot taken before the failed start.
		if err := e.snapshotPointers(ctx); err != nil {
			return e.toRecovery(ctx, txn, apierr.CheckpointFailed, api.UpdateStageRollback, nil)
		}
		view := e.viewFor(ctx, txn.LastGoodReleaseID, e.manifestOf(ctx, txn.LastGoodReleaseID))
		h, ok := e.runHealth(ctx, view)
		if !ok {
			return e.toRecovery(ctx, txn, apierr.RollbackFailed, api.UpdateStageRollback, &h)
		}
		return e.advance(ctx, txn, updatejournal.StageRolledBack, updatejournal.StageUpdate{
			Health: h, RollbackReleaseID: txn.LastGoodReleaseID,
		})
	}
	return e.advance(ctx, txn, updatejournal.StageRolledBack, updatejournal.StageUpdate{
		RollbackReleaseID: txn.LastGoodReleaseID,
	})
}

func (e *Engine) beginRollback(ctx context.Context, txn updatejournal.Txn, code apierr.Code, stage api.UpdateStage, health *api.UpdateHealth) (updatejournal.Txn, error) {
	u := updatejournal.StageUpdate{ErrorCode: string(code), ErrorStage: string(stage), ErrorDetail: string(code)}
	if health != nil {
		u.Health = health
	}
	return e.advance(ctx, txn, updatejournal.StageRollingBack, u)
}

func (e *Engine) rollbackDestructive(ctx context.Context, txn updatejournal.Txn, code apierr.Code, stage api.UpdateStage) (updatejournal.Txn, error) {
	u := updatejournal.StageUpdate{
		Destructive: true, ErrorCode: string(code), ErrorStage: string(stage), ErrorDetail: string(code),
	}
	return e.advance(ctx, txn, updatejournal.StageRollingBack, u)
}

func (e *Engine) toRecovery(ctx context.Context, txn updatejournal.Txn, code apierr.Code, stage api.UpdateStage, health *api.UpdateHealth) (updatejournal.Txn, error) {
	u := updatejournal.StageUpdate{ErrorCode: string(code), ErrorStage: string(stage), ErrorDetail: string(code)}
	if health != nil {
		u.Health = health
	}
	return e.advance(ctx, txn, updatejournal.StageRecoveryRequired, u)
}

func (e *Engine) advance(ctx context.Context, txn updatejournal.Txn, to updatejournal.Stage, u updatejournal.StageUpdate) (updatejournal.Txn, error) {
	next, err := e.Journal.Advance(ctx, txn.TxnID, to, u)
	if err != nil {
		return txn, err
	}
	_ = fsutil.SyncDir(e.Layout.Journal)
	return next, nil
}

func (e *Engine) seal(ctx context.Context, txn updatejournal.Txn) error {
	if err := e.hookStage(string(txn.Stage)); err != nil {
		return err
	}
	job, err := e.Journal.GetJob(ctx, txn.JobID, txn.Attempt)
	if err != nil {
		return err
	}
	c, err := decodeClaimed(job.IntentJSON)
	if err != nil {
		return err
	}
	state := api.UpdateResultStateFailed
	switch txn.Stage {
	case updatejournal.StageLocallySucceeded:
		state = api.UpdateResultStateSucceeded
	case updatejournal.StageRolledBack:
		if txn.Destructive {
			state = api.UpdateResultStateRolledBack
		}
	}
	code := apierr.Code(txn.ErrorCode)
	if state == api.UpdateResultStateSucceeded {
		code = ""
	}
	p, err := e.Journal.Pointers(ctx)
	if err != nil {
		return err
	}
	current := p.CurrentRelease
	recovery := txn.Stage == updatejournal.StageRecoveryRequired
	forwarding := false
	if recovery {
		if e.Gate != nil {
			_ = e.Gate.Block(ctx)
		}
	} else if e.Gate != nil {
		forwarding, err = e.enforce(ctx, current, c, txnOutcome(txn))
		if err != nil {
			return err
		}
	}
	health, _ := decodeHealth(txn.Health)
	cp, _ := decodeCheckpoint(txn.Checkpoint)
	schema, _ := e.Journal.StateSchema(ctx)
	if schema < 1 {
		schema = schemaOr(txn.StateSchemaTo, e.Platform.StateSchema)
	}
	started := txn.StartedAt
	finished := txn.FinishedAt
	if finished.IsZero() {
		finished = txn.UpdatedAt
	}
	prev := txn.PreviousReleaseID
	in := resultInput{
		Claimed: c, State: state, Stage: journalStageAPI(txn.Stage), Code: code,
		Current: current, Previous: prev, Recovery: recovery, Forwarding: forwarding,
		Started: started, Finished: finished, Schema: schemaAtLeast1(schema),
		Health: health, Checkpoint: cp,
	}
	if txn.Stage == updatejournal.StageRolledBack && !txn.Destructive {
		in.Stage = api.UpdateStage(txn.ErrorStage)
		if in.Stage == "" {
			in.Stage = api.UpdateStageFinalSample
		}
	}
	if err := e.enqueueResult(ctx, in); err != nil {
		return err
	}
	if err := e.finish(ctx, job, string(state), string(code)); err != nil {
		return err
	}
	if e.Sender != nil {
		if err := e.Flush(ctx); err != nil {
			e.log().Info("result not accepted yet", "err", errCode(err))
		}
	}
	_ = e.Retain(ctx)
	e.cleanTxnNotes(txn.TxnID)
	if recovery {
		return ErrMembersBlocked
	}
	return nil
}

func txnOutcome(txn updatejournal.Txn) string {
	switch txn.Stage {
	case updatejournal.StageLocallySucceeded:
		return updatejournal.OutcomeSucceeded
	case updatejournal.StageRolledBack:
		return updatejournal.OutcomeRolledBack
	case updatejournal.StageRecoveryRequired:
		return updatejournal.OutcomeRecoveryRequired
	default:
		return updatejournal.OutcomeFailed
	}
}

func (e *Engine) cleanTxnNotes(txnID string) {
	for _, prefix := range []string{"epoch-", "checks-", "rollback-"} {
		_ = os.Remove(filepath.Join(e.Layout.Journal, prefix+txnID))
	}
}

func (e *Engine) claimedFor(ctx context.Context, txn updatejournal.Txn) (claimed, error) {
	job, err := e.Journal.GetJob(ctx, txn.JobID, txn.Attempt)
	if err != nil {
		return claimed{}, err
	}
	return decodeClaimed(job.IntentJSON)
}

func (e *Engine) drifted(c claimed) bool {
	return e.Node.Revoked || e.Node.ID != c.Intent.NodeId || e.Node.Incarnation != c.Intent.IncarnationId
}

func (e *Engine) manifestOf(ctx context.Context, id string) *api.ReleaseManifest {
	rel, err := e.Journal.GetRelease(ctx, id)
	if err != nil {
		return nil
	}
	m, err := VerifyManifest(rel.ManifestEnvelope, rel.ManifestSHA256, e.ReleaseTrust, e.now())
	if err != nil {
		return nil
	}
	return &m
}
