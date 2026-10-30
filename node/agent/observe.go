package agent

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/controlclient"
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
	rep, err := e.report(wall.UTC())
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

func (e *Engine) report(at time.Time) (api.ObservationReport, error) {
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
		Usage: api.UsageOutboxStatus{
			BlockedSeq:      presence.NewNullNullable[decstr.U64](),
			LedgerEpoch:     presence.NewNullNullable[uuid.UUID](),
			MeterEpochId:    presence.NewNullNullable[uuid.UUID](),
			OldestUnackedAt: presence.NewNullNullable[time.Time](),
		},
	}
	rep.Lease.ExpiresAt = exp
	rep.Lease.LeaseId = lid
	rep.Netbird.Backend = backend
	rep.Netbird.PeerId = presence.NewNullNullable[string]()
	rep.Netbird.Version = presence.NewNullNullable[string]()
	rep.Update.Generation = presence.NewNullNullable[decstr.U64]()
	rep.Update.JobId = presence.NewNullNullable[uuid.UUID]()
	rep.Update.Stage = presence.NewNullNullable[api.UpdateStage]()
	rep.Update.State = presence.NewNullNullable[api.UpdateJobState]()
	return rep, nil
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
