package agent

import (
	"context"
	"time"
)

// A freeze is an incarnation latch, never cleared by an absent/false response
// or by a later signed envelope. Resolution is revoke and re-enroll.
func (e *Engine) freezeIdentity(ctx context.Context) {
	if e.identityFrozen {
		return
	}
	e.identityFrozen = true
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	// Stop the member path before persistence or control I/O can block.
	e.withdraw(cleanup)
	if err := e.store.SetMeta(cleanup, "identity_frozen", "1"); err != nil {
		e.storageOK = false
	}
	e.decision = Decide(e.input())
	e.lastObs = time.Time{}
	e.beat()
	// D-088 overrides both a renewal already attempted this step and D-087's
	// scheduled slot. Later renewals use the ordinary grant-less schedule.
	e.leaseAttempted = false
	e.leaseRetryAt = 0
	e.leaseFresh = false
	_ = e.syncLease(cleanup)
	e.decision = Decide(e.input())
}
