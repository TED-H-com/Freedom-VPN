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
	readCtx, cancel := context.WithTimeout(ctx, e.servingReadTimeout())
	defer cancel()
	evidence, err := e.plane.VerifyServing(readCtx, model, assignments)
	// Even an executor that returns success at the deadline cannot supply
	// fresh evidence. Keep a specific CheckError when one accompanied it.
	err = errors.Join(err, readCtx.Err())
	if err == nil {
		return true
	}
	e.rejectServing(ctx, model, evidence, err)
	return false
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
	return !e.expireServing(ctx)
}

func (e *Engine) expireServing(ctx context.Context) bool {
	if !e.serving || (e.policyExpires > 0 && e.clock.Mono() < e.policyExpires) {
		return false
	}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	e.withdraw(cleanup)
	e.decision = Decide(e.input())
	e.lastObs = time.Time{}
	return true
}

// rejectServing is the common withdrawal point for Apply and live reads.
// Quarantine precedes withdrawal and always uses the retained applied model.
func (e *Engine) rejectServing(ctx context.Context, _ enforce.Model, evidence enforce.ServingEvidence, err error) {
	e.noteApply(err)
	// A failed/canceled read must still be able to install the baseline.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	var check *enforce.CheckError
	if errors.As(err, &check) && check.Reason == enforce.ReasonTCForeignFilter {
		e.quarantinePending = true
		e.memberPathPending = true
		e.recordForeign(evidence)
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
	ok := e.verifyCurrentServing(ctx)
	if !e.serving {
		e.auditBlocked(ctx)
	}
	if !e.serving && e.quarantinePending {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		e.quarantine(cleanup)
		cancel()
		e.decision = Decide(e.input())
		ok = false
	}
	e.beat()
	if !ok || (e.identityFrozen && e.lastObs.IsZero()) {
		reportCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = e.observe(reportCtx)
		_ = e.flush(reportCtx)
	}
}

// A baseline is not proof that an ingress redirect disappeared. Audit the
// retained member path even on revoked, canceled and already-blocked returns.
func (e *Engine) auditBlocked(ctx context.Context) {
	if e.appliedModel == nil {
		return
	}
	readCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), e.servingReadTimeout())
	defer cancel()
	ev, err := e.plane.VerifyServing(readCtx, *e.appliedModel, e.appliedAssignments)
	var check *enforce.CheckError
	if errors.As(err, &check) && check.Reason == enforce.ReasonTCForeignFilter {
		e.noteApply(err)
		e.recordForeign(ev)
		e.quarantinePending, e.memberPathPending = true, true
	}
	if ev.Quarantined {
		e.quarantinePending, e.memberPathPending = true, true
	}
}

// Retained solely for containment auditing; never used as desired state.
type retainedApplied struct {
	Model       enforce.Model        `json:"model"`
	Assignments map[uuid.UUID]uint16 `json:"assignments"`
}
