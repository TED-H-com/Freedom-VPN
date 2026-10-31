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
	if e.appliedModel == nil {
		e.rejectServing(ctx, enforce.Model{}, enforce.ServingEvidence{}, errors.New("missing applied model"))
		return false
	}
	return e.verifyServingModel(ctx, *e.appliedModel, e.appliedAssignments)
}

// rejectServing is the common withdrawal point for Apply and live reads.
// The exact model and evidence remain available here for the D-090 quarantine
// consumer in follow-up 22. Today withdrawal installs the existing baseline;
// it does not remove or neutralize a foreign ingress redirect.
func (e *Engine) rejectServing(ctx context.Context, _ enforce.Model, _ enforce.ServingEvidence, err error) {
	e.noteApply(err)
	// A failed/canceled read must still be able to install the baseline.
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	e.withdraw(cleanupCtx)
	e.decision = Decide(e.input())
	e.lastObs = time.Time{}
}

// finishServingStep runs on every return, including control I/O cancellation
// and a busy apply lock. Write the heartbeat immediately after the live read;
// a failure also queues and delivers a corrected observation on this step.
func (e *Engine) finishServingStep(ctx context.Context) {
	ok := e.verifyCurrentServing(ctx)
	e.beat()
	if !ok {
		reportCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = e.observe(reportCtx)
		_ = e.flush(reportCtx)
	}
}
