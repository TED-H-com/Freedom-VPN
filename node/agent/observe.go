package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func (e *Engine) observe(ctx context.Context) error {
	if !e.enrolled || e.nodeID == uuid.Nil {
		return nil
	}
	wall := e.clock.Wall()
	interval := time.Duration(e.cfg.ObservationSeconds) * time.Second
	due := e.lastObs.IsZero() || wall.Sub(e.lastObs) >= interval || e.serving != e.lastServing
	if !due || e.outboxFull {
		return nil
	}
	rep, err := e.report(ctx, wall.UTC())
	if err != nil {
		return err
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		return err
	}
	if err := e.store.Enqueue(ctx, rep.ReportId, raw, wall.UTC()); err != nil {
		e.storageOK = false
		return err
	}
	e.lastObs = wall
	e.lastServing = e.serving
	return nil
}

func (e *Engine) report(ctx context.Context, at time.Time) (api.ObservationReport, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return api.ObservationReport{}, err
	}
	appliedGen := presence.NewNullNullable[decstr.U64]()
	appliedBind := presence.NewNullNullable[decstr.U64]()
	appliedPol := presence.NewNullNullable[api.PolicyRefs]()
	if e.appliedGen > 0 && e.desired != nil {
		appliedGen = presence.NewNullableWithValue(decstr.U64(e.appliedGen))
		appliedBind = presence.NewNullableWithValue(e.desired.BindingRevision)
		appliedPol = presence.NewNullableWithValue(e.desired.PolicyRefs)
	}
	release := presence.NewNullNullable[uuid.UUID]()
	if rid := e.cfg.ReleaseUUID(); rid != uuid.Nil {
		release = presence.NewNullableWithValue(rid)
	}
	lastGood := presence.NewNullNullable[uuid.UUID]()
	if e.appliedGen > 0 {
		lastGood = release
	}
	exp := presence.NewNullNullable[time.Time]()
	lid := presence.NewNullNullable[uuid.UUID]()
	if e.lease != nil {
		exp = presence.NewNullableWithValue(e.lease.ExpiresAt.UTC())
		lid = presence.NewNullableWithValue(e.lease.LeaseId)
	}
	reasons := e.decision.Reasons
	if reasons == nil {
		reasons = []api.ForwardingBlockReason{}
	}
	errs := e.decision.Errors
	if errs == nil {
		errs = []api.NodeError{}
	}
	checks := e.decision.Checks
	if checks == nil {
		checks = []api.NodeCheck{}
	}
	backend := api.ObservationReportNetbirdBackendUnknown
	switch e.cfg.NetbirdBackend {
	case "kernel":
		backend = api.ObservationReportNetbirdBackendKernel
	case "userspace":
		backend = api.ObservationReportNetbirdBackendUserspace
	}
	rep := api.ObservationReport{
		AgentVersion:            e.cfg.AgentRelease,
		AppliedBindingRevision:  appliedBind,
		AppliedConfigGeneration: appliedGen,
		AppliedPolicyRefs:       appliedPol,
		BootId:                  e.bootID,
		Checks:                  checks,
		CurrentReleaseId:        release,
		DataHealth:              reportedHealth(e.decision.DataHealth),
		Errors:                  errs,
		ForwardingBlockReasons:  reasons,
		LastGoodReleaseId:       lastGood,
		MemberForwardingEnabled: e.serving,
		NodeId:                  e.nodeID,
		NodeIncarnationId:       e.incID,
		ObservedAt:              at,
		ProvisioningState:       e.decision.Provisioning,
		ReportId:                id,
		SafetyStatus:            e.decision.Safety,
	}
	rep.Lease.ExpiresAt = exp
	rep.Lease.LeaseId = lid
	rep.Netbird.Backend = backend
	rep.Netbird.PeerId = presence.NewNullNullable[string]()
	rep.Netbird.Version = presence.NewNullNullable[string]()
	e.attachUsage(ctx, &rep)
	e.attachUpdate(ctx, &rep)
	if e.dnsEv != nil {
		rep.DnsPolicyEvidence = presence.NewNullableWithValue(*e.dnsEv)
	}
	return rep, nil
}

// attachUpdate reads the updater journal once for this report. A null
// generation (no file, no recorded job, or a snapshot error) leaves the
// update members null. The read is not cached and does not fail the report.
// current_release_id defaults to the running binary. Only a completed local
// switch or rollback with a valid job ID can override it from the journal.
func (e *Engine) attachUpdate(ctx context.Context, rep *api.ObservationReport) {
	rep.Update.Generation = presence.NewNullNullable[decstr.U64]()
	rep.Update.JobId = presence.NewNullNullable[uuid.UUID]()
	rep.Update.Stage = presence.NewNullNullable[api.UpdateStage]()
	rep.Update.State = presence.NewNullNullable[api.UpdateJobState]()
	if e.cfg.UpdaterJournal == "" {
		return
	}
	snap, err := updatejournal.Snapshot(ctx, e.cfg.UpdaterJournal)
	e.noteUpdateSnapshot(err)
	if err != nil || snap.Generation == nil {
		return
	}
	rep.Update.Generation = presence.NewNullableWithValue(*snap.Generation)
	if id, perr := uuid.Parse(snap.JobID); perr == nil && id != uuid.Nil {
		rep.Update.JobId = presence.NewNullableWithValue(id)
	}
	if stage := api.UpdateStage(snap.Stage); stage.Valid() {
		rep.Update.Stage = presence.NewNullableWithValue(stage)
	}
	if state := api.UpdateJobState(snap.State); state.Valid() {
		rep.Update.State = presence.NewNullableWithValue(state)
	}
	current := presence.NewNullableWithValue(e.cfg.ReleaseUUID())
	_, jobErr := rep.Update.JobId.Get()
	if jobErr == nil && (snap.JournalStage == updatejournal.StageLocallySucceeded || snap.JournalStage == updatejournal.StageRolledBack) {
		if id, perr := uuid.Parse(snap.CurrentRelease); perr == nil && id != uuid.Nil {
			current = presence.NewNullableWithValue(id)
		}
	}
	last := presence.NewNullNullable[uuid.UUID]()
	if id, perr := uuid.Parse(snap.LastGoodRelease); perr == nil && id != uuid.Nil {
		last = presence.NewNullableWithValue(id)
	}
	rep.CurrentReleaseId = current
	rep.LastGoodReleaseId = last
}

func (e *Engine) noteUpdateSnapshot(err error) {
	reason := updateSnapshotReason(err)
	if reason == e.updateSnapReason {
		return
	}
	e.updateSnapReason = reason
	if reason == "" || e.log == nil {
		return
	}
	e.log.Warn("updater journal snapshot", "reason", reason)
}

// updateSnapshotReason is a short class. It is safe to log. Snapshot errors
// can carry a path or a schema number, and those stay out of the log.
func updateSnapshotReason(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "missing"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unsupported snapshot schema"):
		return "unsupported_schema"
	case strings.Contains(msg, "regular database file"):
		return "not_a_file"
	default:
		return "unreadable"
	}
}

func reportedHealth(h api.DataHealth) api.DataHealth {
	if !h.Valid() {
		return api.DataHealthUnknown
	}
	return h
}

func (e *Engine) flush(ctx context.Context) error {
	items, err := e.store.Pending(ctx)
	if err != nil {
		e.storageOK = false
		return err
	}
	e.outboxFull = len(items) >= 32
	for _, item := range items {
		var rep api.ObservationReport
		if err := json.Unmarshal(item.Payload, &rep); err != nil {
			_ = e.store.MarkConflict(ctx, item.ReportID)
			continue
		}
		_, err := e.client.Observe(ctx, item.ReportID, rep)
		if err == nil {
			if serr := e.store.MarkAccepted(ctx, item.ReportID, e.clock.Wall().UTC()); serr != nil {
				e.storageOK = false
			}
			continue
		}
		if isRevoked(err) {
			e.markRevoked(ctx)
			return nil
		}
		var se *controlclient.StatusError
		if errors.As(err, &se) && (se.Status == 409 || se.Status == 422) {
			_ = e.store.MarkConflict(ctx, item.ReportID)
			continue
		}
		_ = e.store.BumpAttempt(ctx, item.ReportID, safeErr(err))
	}
	return nil
}
