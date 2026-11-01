package agent

import (
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Diagnostic thresholds, not a serving grace or an availability SLA. Alarm on
// either repeated failed convergence or elapsed time; never relax the baseline.
const staleHoldSteps = 3
const staleHoldDuration = 30 * time.Second

type staleLeaseHold struct {
	steps   uint64
	since   time.Duration
	elapsed time.Duration
	alarm   bool
	counted bool // at most once per Step, even with multiple observations
}

// Returns true on start, alarm and recovery. A newer desired generation or
// another old lease cannot restart the streak. Wall-clock changes do not enter
// the calculation. Restart begins a new process streak; central retains history.
func (e *Engine) updateStaleHold() bool {
	h := &e.staleHold
	if e.lease == nil || e.desired == nil || !e.desiredOK || e.revoked || e.identityFrozen ||
		e.appliedGen <= uint64(e.lease.ConfigGeneration) || e.serving {
		changed := h.steps > 0
		*h = staleLeaseHold{counted: h.counted}
		return changed
	}
	changed := h.steps == 0
	if changed {
		h.since = e.clock.Mono()
	}
	if !h.counted {
		h.steps++
		h.counted = true
	}
	h.elapsed = max(h.elapsed, e.clock.Mono()-h.since)
	if !h.alarm && (h.steps >= staleHoldSteps || h.elapsed >= staleHoldDuration) {
		h.alarm, changed = true, true
		e.log.Warn("serving lease generation hold", "code", api.ErrorCodeSTALEGENERATION,
			"steps", h.steps, "elapsed_ms", h.elapsed.Milliseconds())
	}
	return changed
}

func (e *Engine) attachStaleHold(rep *api.ObservationReport) {
	h := e.staleHold
	if h.steps == 0 {
		return
	}
	rep.Errors = append(rep.Errors, api.NodeError{Code: api.ErrorCodeSTALEGENERATION, Stage: "lease",
		ObservedAt: rep.ObservedAt,
		Message: fmt.Sprintf("Serving lease generation hold: applied=%d lease=%d steps=%d elapsed_ms=%d alarm=%t.",
			e.appliedGen, uint64(e.lease.ConfigGeneration), h.steps, h.elapsed.Milliseconds(), h.alarm)})
}
