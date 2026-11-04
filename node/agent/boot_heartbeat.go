package agent

import (
	"context"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
)

// startupHeartbeat records pending adoption without renewing the predecessor's
// liveness or asserting that this process verified its rules. A second startup
// cannot refresh a pending receipt: only an actual serving receipt qualifies.
func (e *Engine) startupHeartbeat(ctx context.Context, prior watchdog.Beat) error {
	return e.writeBootHeartbeat(ctx, func(ctx context.Context, path string, b watchdog.Beat) error {
		now := e.clock.Mono()
		if prior.BootID == e.bootID && prior.Serving && prior.BaselineOK && prior.Generation > 0 &&
			prior.Mono > 0 && prior.Mono <= now && now-prior.Mono <= time.Duration(e.cfg.WatchdogStaleSeconds)*time.Second && prior.PolicyExpires > now {
			b.Mono, b.Wall = prior.Mono, prior.Wall
			b.Generation, b.PolicyExpires = prior.Generation, prior.PolicyExpires
			b.Reason = watchdog.AdoptionPending
		}
		return watchdog.WriteHeartbeatContext(ctx, path, b)
	})
}

// Heartbeat publication consumes Boot's single local deadline. An independent
// 100ms cap made healthy disk fsync/scheduler delay look like startup failure.
// WriteHeartbeatContext still checks the deadline before atomic publication.
func (e *Engine) bootHeartbeat(parent context.Context) error {
	return e.writeBootHeartbeat(parent, watchdog.WriteHeartbeatContext)
}

func (e *Engine) writeBootHeartbeat(ctx context.Context, write func(context.Context, string, watchdog.Beat) error) error {
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
	return write(ctx, e.cfg.HeartbeatPath, b)
}
