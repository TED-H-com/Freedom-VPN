package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// Version is the updater binary version reported in the user agent.
// Security decisions use release_id and minimum_updater_version, not this
// string (§13.3).
const Version = "1.0.0"

// resultID is stable across a replay of the same job, attempt and state.
// UUIDv7 is not replay-stable, so this is UUIDv5 over those fields.
func resultID(jobID string, attempt int, state api.UpdateResultState) uuid.UUID {
	name := fmt.Sprintf("fvpn-update-result|%s|%d|%s", jobID, attempt, state)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(name))
}

type resultInput struct {
	Claimed    claimed
	State      api.UpdateResultState
	Stage      api.UpdateStage
	Code       apierr.Code
	Current    string
	Previous   string
	Recovery   bool
	Forwarding bool
	Started    time.Time
	Finished   time.Time
	Schema     int
	Health     *api.UpdateHealth
	Checkpoint *api.UpdateUsageCheckpoint
}

func buildResult(in resultInput) (payload []byte, hash string, id uuid.UUID, err error) {
	c := in.Claimed
	schema := in.Schema
	if schema < 1 {
		schema = 1
	}
	started := utcSec(in.Started)
	if started.IsZero() {
		started = utcSec(c.Intent.IssuedAt)
	}
	finished := utcSec(in.Finished)
	if finished.IsZero() {
		finished = started
	}
	res := api.UpdateResult{
		Attempt:                 c.Intent.Attempt,
		BatchId:                 c.Intent.BatchId,
		BootId:                  c.BootID,
		CurrentReleaseId:        presence.FromPointer(parseID(in.Current)),
		ErrorCode:               presence.FromPointer(codePtr(in.Code)),
		FinishedAt:              presence.FromPointer(timePtr(finished)),
		Generation:              c.Intent.Generation,
		Health:                  presence.FromPointer(in.Health),
		IncarnationId:           c.Intent.IncarnationId,
		JobId:                   c.Intent.JobId,
		ManifestSha256:          c.Intent.ManifestSha256,
		MemberForwardingEnabled: in.Forwarding,
		NodeId:                  c.Intent.NodeId,
		PolicyGeneration:        c.PolicyGeneration,
		PreviousReleaseId:       presence.FromPointer(parseID(in.Previous)),
		PublicationIntent:       c.Intent.PublicationIntent,
		RecoveryRequired:        presence.NewRequired(in.Recovery),
		RequestedReleaseId:      c.Intent.ReleaseId,
		ResultId:                resultID(c.Intent.JobId.String(), c.Intent.Attempt, in.State),
		Stage:                   presence.FromPointer(stagePtr(in.Stage)),
		StartedAt:               started,
		State:                   in.State,
		StateSchemaVersion:      schema,
		TargetId:                c.Intent.TargetId,
		UsageCheckpoint:         presence.FromPointer(in.Checkpoint),
	}
	raw, err := json.Marshal(res)
	if err != nil {
		return nil, "", uuid.Nil, err
	}
	hash, err = canonjson.HashHex(raw, "payload_hash")
	if err != nil {
		return nil, "", uuid.Nil, err
	}
	res.PayloadHash = hash
	raw, err = json.Marshal(res)
	if err != nil {
		return nil, "", uuid.Nil, err
	}
	canon, err := canonjson.Canonicalize(raw)
	if err != nil {
		return nil, "", uuid.Nil, err
	}
	return canon, hash, res.ResultId, nil
}

func (e *Engine) enqueueResult(ctx context.Context, in resultInput) error {
	payload, hash, id, err := buildResult(in)
	if err != nil {
		return err
	}
	entry := updatejournal.OutboxEntry{
		ResultID:    id.String(),
		JobID:       in.Claimed.Intent.JobId.String(),
		Attempt:     in.Claimed.Intent.Attempt,
		Generation:  in.Claimed.Intent.Generation,
		State:       string(in.State),
		Payload:     payload,
		PayloadHash: hash,
	}
	if err := e.Journal.Enqueue(ctx, entry); err != nil {
		if errors.Is(err, updatejournal.ErrResultConflict) {
			_ = e.Journal.MarkConflict(ctx, id.String(), "payload hash mismatch")
			return permanent(apierr.AckConflict, "result payload changed")
		}
		return err
	}
	return nil
}

// NoteAck records central acceptance. A late duplicate ACK is a no-op.
// A different payload hash is ACK_CONFLICT: the entry is kept and not resent.
func (e *Engine) NoteAck(ctx context.Context, resultID, payloadHash string, acceptedAsHistory bool, raw []byte) (duplicate bool, err error) {
	dup, err := e.Journal.AckResult(ctx, updatejournal.Ack{
		ResultID:          resultID,
		PayloadHash:       payloadHash,
		AcceptedAsHistory: acceptedAsHistory,
		Raw:               raw,
	})
	if errors.Is(err, updatejournal.ErrResultConflict) {
		_ = e.Journal.MarkConflict(ctx, resultID, "ACK payload hash mismatch")
		return false, permanent(apierr.AckConflict, "result")
	}
	return dup, err
}

// Flush sends pending results. A send error leaves the entry pending.
// Local success is not changed into acceptance here (§13.6 step 13).
func (e *Engine) Flush(ctx context.Context) error {
	if e.Sender == nil {
		return nil
	}
	if err := e.open(ctx); err != nil {
		return err
	}
	return e.withLock(ctx, func() error { return e.flushLocked(ctx) })
}

func (e *Engine) flushLocked(ctx context.Context) error {
	pending, err := e.Journal.Pending(ctx, 50)
	if err != nil {
		return err
	}
	for _, ent := range pending {
		if ent.Status == updatejournal.OutboxConflict {
			continue
		}
		ack, conflict, err := e.Sender.Submit(ctx, ent.Payload)
		if conflict {
			if err := e.Journal.MarkConflict(ctx, ent.ResultID, "central ACK_CONFLICT"); err != nil {
				return err
			}
			continue
		}
		if err != nil {
			_ = e.Journal.RecordSendFailure(ctx, ent.ResultID, errCode(err))
			return transient(apierr.StoreUnavailable, "result submit")
		}
		raw := ack.Raw
		if len(raw) == 0 {
			raw = []byte("{}")
		}
		if _, err := e.NoteAck(ctx, ent.ResultID, ent.PayloadHash, ack.AcceptedAsHistory, raw); err != nil {
			return err
		}
	}
	return nil
}

func journalStageAPI(s updatejournal.Stage) api.UpdateStage {
	switch s {
	case updatejournal.StagePrepared:
		return api.UpdateStagePrepare
	case updatejournal.StageBlocked:
		return api.UpdateStageBlockForwarding
	case updatejournal.StageFinalSampled:
		return api.UpdateStageFinalSample
	case updatejournal.StageStopped:
		return api.UpdateStageStopServices
	case updatejournal.StageMigrated:
		return api.UpdateStageMigrate
	case updatejournal.StageSwitched:
		return api.UpdateStageSwitch
	case updatejournal.StageChecking:
		return api.UpdateStageCheck
	case updatejournal.StageRollingBack:
		return api.UpdateStageRollback
	default:
		return api.UpdateStageReport
	}
}

func utcSec(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC().Truncate(time.Second)
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	u := utcSec(t)
	return &u
}

func codePtr(c apierr.Code) *api.ErrorCode {
	if c == "" {
		return nil
	}
	v := api.ErrorCode(c)
	return &v
}

func stagePtr(s api.UpdateStage) *api.UpdateStage {
	if s == "" {
		return nil
	}
	v := s
	return &v
}

func parseID(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}

func checkpointFromUsage(c UsageCheckpoint, newEpoch string) (*api.UpdateUsageCheckpoint, error) {
	out := &api.UpdateUsageCheckpoint{
		LastDurableEventSeq: c.LastDurableEventSeq,
		CentralAckedThrough: c.CentralAckedThrough,
		ClosedMeterEpochId:  presence.NewNullNullable[uuid.UUID](),
		NewMeterEpochId:     presence.NewNullNullable[uuid.UUID](),
	}
	if c.ClosedMeterEpochID != "" {
		id, err := uuid.Parse(c.ClosedMeterEpochID)
		if err != nil {
			return nil, permanent(apierr.CheckpointFailed, "closed meter epoch")
		}
		out.ClosedMeterEpochId.Set(id)
	}
	if newEpoch != "" {
		id, err := uuid.Parse(newEpoch)
		if err != nil {
			return nil, permanent(apierr.CheckpointFailed, "new meter epoch")
		}
		out.NewMeterEpochId.Set(id)
	}
	return out, nil
}

func decodeCheckpoint(raw []byte) (*api.UpdateUsageCheckpoint, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var c api.UpdateUsageCheckpoint
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func decodeHealth(raw []byte) (*api.UpdateHealth, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var h api.UpdateHealth
	if err := json.Unmarshal(raw, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// schemaAtLeast1 satisfies the result schema (minimum 1) when the node has
// not recorded a schema yet.
func schemaAtLeast1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
