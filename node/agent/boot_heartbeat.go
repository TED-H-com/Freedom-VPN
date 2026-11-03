package agent

import (
	"context"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
)

// bootHeartbeatBudget is a cap within Boot's existing deadline, never an
// additional phase. A blocked worker has already published conservative state.
const bootHeartbeatBudget = 100 * time.Millisecond

func (e *Engine) bootHeartbeat(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, bootHeartbeatBudget)
	defer cancel()
	b := watchdog.Beat{
		BootID: e.bootID, Mono: e.clock.Mono(), Wall: e.clock.Wall().UTC(),
		BaselineOK: e.baselineOK, Reason: "boot_unverified",
	}
	if e.baselineOK {
		b.Reason = "boot_baseline"
	}
	if e.serving && ctx.Err() == nil {
		b.Serving, b.Generation, b.PolicyExpires = true, e.appliedGen, e.servingDeadline()
		b.Reason = "serving"
	}
	return watchdog.WriteHeartbeatContext(ctx, e.cfg.HeartbeatPath, b)
}
