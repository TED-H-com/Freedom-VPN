package agent

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
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
	e.updateStaleHold()
	state := e.observationState()
	due := e.lastObs.IsZero() || wall.Sub(e.lastObs) >= interval || e.serving != e.lastServing || state != e.lastObservationState
	if !due || e.outboxFull {
		return nil
	}
	// Retain the first deciding body through backoff. Repeated copies of the
	// same pending state add no evidence; transitions always get their own ID.
	if state == e.lastObservationState {
		items, err := e.store.Pending(ctx)
		if err != nil {
			return err
		}
		if len(items) > 0 {
			return nil
		}
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
	e.lastObservationState = state
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
	if !e.identityFrozen && !e.freezeUncertain && e.appliedGen > 0 && e.desired != nil {
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
	errs := slices.Clone(e.decision.Errors)
	if errs == nil {
		errs = []api.NodeError{}
	}
	checks := slices.Clone(e.decision.Checks)
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
	e.attachServingRead(&rep)
	e.attachContainment(&rep, at)
	e.attachStaleHold(&rep)
	if f := e.failedCandidate; f != nil {
		err := f.Error
		err.Stage = "apply_candidate"
		err.Message = fmt.Sprintf("candidate_generation=%d verified=false", f.Generation)
		rep.Errors = append(rep.Errors, err)
	}
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
	if e.clock.Mono() < e.observationRetryAt {
		return nil
	}
	items, err := e.store.Pending(ctx)
	if err != nil {
		e.storageOK = false
		return err
	}
	e.outboxFull = len(items) >= 32
	for _, item := range items {
		if err := controlContext(ctx).Err(); err != nil {
			return err
		}
		var rep api.ObservationReport
		if err := json.Unmarshal(item.Payload, &rep); err != nil {
			_ = e.store.MarkConflict(ctx, item.ReportID)
			continue
		}
		if e.identityFrozen && e.freezeDurable && (rep.MemberForwardingEnabled || !rep.AppliedConfigGeneration.IsNull()) {
			_ = e.store.MarkConflict(ctx, item.ReportID)
			continue
		}
		// Persist uncertainty before sending. A process can die after central
		// accepts a freeze but before the response reaches local storage.
		if err := e.store.BumpAttempt(ctx, item.ReportID, "in_flight"); err != nil {
			e.storageOK = false
			return err
		}
		accepted, err := e.client.Observe(controlContext(ctx), item.ReportID, rep)
		if err == nil {
			e.observationFailures, e.observationRetryAt = 0, 0
			if accepted.IdentityFrozen != nil && *accepted.IdentityFrozen {
				e.freezeIdentity(ctx)
				if !e.freezeDurable {
					return nil
				}
			}
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
		e.observationFailures++
		wait := time.Second
		for i := 1; i < e.observationFailures && wait < 30*time.Second; i++ {
			wait *= 2
		}
		wait = min(time.Minute, wait+observationJitter(wait))
		if se != nil {
			wait = max(wait, se.RetryAfter)
		}
		e.observationRetryAt = e.clock.Mono() + wait
		return nil // One failed request per slot, even with a full outbox.
	}
	remaining, err := e.store.Pending(ctx)
	if err != nil {
		return err
	}
	e.freezeUncertain = false
	for _, item := range remaining {
		if item.Attempts > 0 {
			e.freezeUncertain = true
		}
	}
	return nil
}

func observationJitter(wait time.Duration) time.Duration {
	span := wait / 10
	if span <= 0 {
		return 0
	}
	// Nanosecond ranges need more than 16 bits of entropy; otherwise every
	// node retries within a few microseconds of the same lower bound.
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0
	}
	return time.Duration(binary.LittleEndian.Uint64(b[:])%uint64(2*span+1)) - span
}

// Timestamps and read-at detail change every tick. The semantic transition
// includes every check status/code and block reason, including provisional.
func (e *Engine) observationState() string {
	d := e.decision
	type check struct {
		Name   api.NodeCheckName
		Status api.HealthCheckStatus
		Code   string
	}
	cs := make([]check, 0, len(d.Checks))
	for _, c := range d.Checks {
		code, _ := c.ErrorCode.Get()
		cs = append(cs, check{c.Name, c.Status, string(code)})
	}
	// A rejected candidate is not an applied ACK. Keep its transition even
	// while older bodies await retry. Exclude read times/diagnostic durations
	// so identical pending evidence is still coalesced.
	var candidate uint64
	var code api.ErrorCode
	if f := e.failedCandidate; f != nil {
		candidate, code = f.Generation, f.Error.Code
	}
	var readGeneration uint64
	var verified bool
	if r := e.servingRead; r != nil {
		readGeneration, verified = r.Generation, r.Verified
	}
	b, _ := json.Marshal([]any{e.serving, e.appliedGen, d.Provisioning, d.Safety, d.Reasons, cs,
		candidate, code, readGeneration, verified, e.staleHold.steps > 0, e.staleHold.alarm})
	return string(b)
}

// An Apply failure can precede the verifier. Report the attempted generation
// without pretending that a missing verifier read supplied fresh tc evidence.
type failedCandidate struct {
	Generation uint64
	Error      api.NodeError
}
