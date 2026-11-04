package agent

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
)

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
