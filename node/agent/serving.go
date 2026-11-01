package agent

import (
	"context"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

func (e *Engine) stepInterval() time.Duration {
	interval := e.heartbeatInterval()
	if poll := time.Duration(e.cfg.PollSeconds) * time.Second; poll > 0 {
		interval = min(interval, poll)
	}
	return interval
}

func (e *Engine) remainingStepDelay(started, interval time.Duration) time.Duration {
	return max(0, interval-max(0, e.clock.Mono()-started))
}

// Each read gets at most half the shorter heartbeat/poll cadence, capped at
// one second. Slow or canceled reads cannot refresh a cached serving claim.
func (e *Engine) servingReadTimeout() time.Duration {
	return min(time.Second, e.stepInterval()/2)
}

func (e *Engine) verifyServingModel(ctx context.Context, model enforce.Model, assignments map[uuid.UUID]uint16) bool {
	// Liveness cancellation may stop control work, but cannot itself prove a
	// kernel defect. This independent bounded read still fails on real drift
	// or its own timeout, including when shutdown interrupts a Step read.
	readCtx, cancel := context.WithTimeout(localContext(ctx), e.servingReadTimeout())
	defer cancel()
	evidence, err := e.plane.VerifyServing(readCtx, model, assignments)
	// Even an executor that returns success at the deadline cannot supply
	// fresh evidence. Keep a specific CheckError when one accompanied it.
	err = errors.Join(err, readCtx.Err())
	e.measureTiming(evidence, "live")
	e.retainServingRead(evidence, model.Generation, e.clock.Wall(), "live", err == nil)
	if err == nil {
		return true
	}
	if !e.repairGeneration(ctx, model, evidence, err) {
		e.rejectServing(ctx, model, evidence, err)
	}
	return false
}

// A replaced generation is repairable with the already-authorized lease.
// Queue the truthful failed read BEFORE repair can replace its evidence. No
// control I/O, baseline, or new lease is required on the successful path.
// Other failures (especially foreign tc) still follow quarantine/withdrawal.
func (e *Engine) repairGeneration(ctx context.Context, model enforce.Model, evidence enforce.ServingEvidence, err error) bool {
	var check *enforce.CheckError
	if e.meter == nil || e.planeLocked || e.repairing || ctx.Err() != nil || !errors.As(err, &check) || check.Reason != enforce.ReasonGenerationMismatch {
		return false
	}
	e.refreshDesired()
	in := e.input()
	in.ApplyError = nil
	d := Decide(in)
	if !d.Serve || e.leaseBoot != e.bootID || e.appliedGen != model.Generation || d.Model.Generation != model.Generation {
		return false
	}
	e.repairing = true
	defer func() { e.repairing = false }()
	e.noteApply(err)
	if e.kernel != nil && e.expectServing.Load() {
		e.kernel.rejectedGeneration()
	}
	e.serving = false
	e.decision = Decide(e.input())
	e.lastObs = time.Time{}
	if err := e.observe(ctx); err != nil {
		return false
	}
	if err := e.enforce(ctx, d); err != nil {
		return false
	}
	if !e.serving {
		return false
	}
	e.decision = Decide(e.input())
	e.beat()
	return true
}

func (e *Engine) verifyCurrentServing(ctx context.Context) bool {
	if !e.serving {
		return true
	}
	if e.expireServing(ctx) {
		return false
	}
	if e.appliedModel == nil {
		e.rejectServing(ctx, enforce.Model{}, enforce.ServingEvidence{}, errors.New("missing applied model"))
		return false
	}
	if !e.verifyServingModel(ctx, *e.appliedModel, e.appliedAssignments) {
		return false
	}
	// The live read may straddle expiry. It proves rules, not a renewed lease.
	if e.expireServing(ctx) {
		return false
	}
	// Publish the verified state before synchronous control work can stall.
	e.beat()
	return true
}

func (e *Engine) expireServing(ctx context.Context) bool {
	if !e.serving || (e.policyExpires > 0 && e.clock.Mono() < e.policyExpires) {
		return false
	}
	cleanup, cancel := context.WithTimeout(localContext(ctx), 5*time.Second)
	defer cancel()
	e.withdraw(cleanup)
	e.decision = Decide(e.input())
	e.lastObs = time.Time{}
	return true
}

// rejectServing is the common withdrawal point for Apply and live reads.
// Quarantine precedes withdrawal and always uses the retained applied model.
func (e *Engine) rejectServing(ctx context.Context, model enforce.Model, evidence enforce.ServingEvidence, err error) {
	e.noteApply(err)
	var check *enforce.CheckError
	if e.kernel != nil && e.expectServing.Load() && errors.As(err, &check) && check.Reason == enforce.ReasonGenerationMismatch {
		// The real live read found a replaced/cleared ruleset. Retain that
		// incident before our planned baseline disables reset detection.
		e.kernel.rejectedGeneration()
	}
	// A failed/canceled read must still be able to install the baseline.
	cleanupCtx, cancel := cleanupBudget(ctx)
	defer cancel()
	if errors.As(err, &check) && check.Reason == enforce.ReasonTCForeignFilter {
		e.quarantinePending = true
		e.memberPathPending = true
		e.recordForeign(evidence)
		if e.containment.ReadAt.IsZero() {
			e.containment.Generation = model.Generation
			e.containment.ReadAt = e.applyError.ObservedAt
			if r := e.servingRead; r != nil && !r.Verified && r.Generation == model.Generation {
				e.containment.ReadAt = r.At
			}
		}
	}
	if e.quarantinePending {
		e.quarantine(cleanupCtx)
	}
	e.withdraw(cleanupCtx)
	e.decision = Decide(e.input())
	e.lastObs = time.Time{}
}

// finishServingStep runs on every return, including control I/O cancellation
// and a busy apply lock. Write the heartbeat immediately after the live read;
// a failure also queues and delivers a corrected observation on this step.
func (e *Engine) finishServingStep(ctx context.Context) {
	// SIGTERM can interrupt control I/O mid-Step. A canceled command is not
	// a kernel observation: perform the bounded final read independently so
	// normal process replacement cannot manufacture a foreign-tc quarantine.
	ctx = localContext(ctx)
	ok := e.verifyCurrentServing(ctx)
	if !e.serving {
		e.auditBlocked(ctx)
	}
	if !e.serving && e.quarantinePending {
		e.quarantine(ctx)
		e.decision = Decide(e.input())
		ok = false
	}
	changed := e.updateStaleHold()
	e.beat()
	if !ok || changed || (e.identityFrozen && e.lastObs.IsZero()) {
		_ = e.observe(ctx)
		_ = e.flush(ctx)
	}
}

// A baseline is not proof that an ingress redirect disappeared. Audit the
// retained member path even on revoked, canceled and already-blocked returns.
func (e *Engine) auditBlocked(ctx context.Context) {
	if e.appliedModel == nil {
		return
	}
	readCtx, cancel := context.WithTimeout(localContext(ctx), e.servingReadTimeout())
	defer cancel()
	ev, err := e.plane.VerifyServing(readCtx, *e.appliedModel, e.appliedAssignments)
	e.measureTiming(ev, "blocked_audit")
	e.retainTCReadback(ev)
	var check *enforce.CheckError
	if errors.As(err, &check) && check.Reason == enforce.ReasonTCForeignFilter {
		e.noteApply(err)
		if e.containment == nil || len(e.containment.TC) == 0 {
			e.recordForeign(ev)
			e.containment.Generation = e.appliedModel.Generation
			e.containment.ReadAt = e.clock.Wall().UTC()
		}
		e.quarantinePending, e.memberPathPending = true, true
	}
	if ev.Quarantined {
		e.quarantinePending, e.memberPathPending = true, true
	}
}

// Retained for containment auditing and the original installation time sample
// during verified same-boot adoption; never used as desired authorization.
type retainedApplied struct {
	Model       enforce.Model        `json:"model"`
	Assignments map[uuid.UUID]uint16 `json:"assignments"`
}

// One shared budget, independent of the number of endpoints and retries.
func (e *Engine) controlPassTimeout() time.Duration {
	stale := time.Duration(e.cfg.WatchdogStaleSeconds) * time.Second
	if stale <= 0 {
		stale = 30 * time.Second
	}
	return min(5*time.Second, stale/3)
}

// The budget constrains network requests only. Local persistence and cleanup
// use the caller context, so a control outage is not reported as disk failure.
type controlBudgetKey struct{}

func controlContext(ctx context.Context) context.Context {
	if budget, ok := ctx.Value(controlBudgetKey{}).(context.Context); ok {
		return budget
	}
	return ctx
}
