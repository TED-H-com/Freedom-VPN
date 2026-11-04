package updater

import (
	"context"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
)

// D-158 is a contention policy, not a maximum holder lifetime. It applies to
// run, seed, apply and standalone flush, which can overlap a serving agent.
const updateLockWaitBudget = 10 * time.Second

const lockRetryInterval = 20 * time.Millisecond
const promptLockAttempts = 5

// acquireCommandLock leaves the common lock APIs and their consumers unchanged.
// Stale metadata switches to the existing five-attempt window, counted from
// the first stale answer, even if live contention preceded it. Later attempts
// may acquire a released lock; otherwise the final diagnostic is preserved.
func (e *Engine) acquireCommandLock(ctx context.Context, command string) (lk *nodelock.Lock, err error) {
	started := time.Now()
	waited := false
	defer func() {
		if waited {
			e.log().Info("update lock wait", "command", command,
				"waited_ms", time.Since(started).Milliseconds(), "acquired", lk != nil,
				"budget_ms", updateLockWaitBudget.Milliseconds())
		}
	}()
	staleAttempts := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lk, err = nodelock.AcquireWithBootID(ctx, e.Layout.Lock, 1, 0, e.Node.BootID.String())
		if err == nil {
			return lk, nil
		}
		waited = true
		if !errors.Is(err, nodelock.ErrBusy) {
			return nil, err
		}
		if staleAttempts > 0 {
			staleAttempts++
		} else if lockBusyDetail(err) != "lock" {
			staleAttempts = 1
		}
		if staleAttempts == promptLockAttempts {
			return nil, err
		}
		delay := lockRetryInterval
		if staleAttempts == 0 {
			remaining := updateLockWaitBudget - time.Since(started)
			if remaining <= 0 {
				return nil, err
			}
			delay = min(delay, remaining)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
