package agent

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"time"
)

// recoveryDecision builds a local deny-all candidate, never an authorization.
// An old-generation or grant-less lease cannot open member forwarding. Clearing
// bindings and grants allows a full nft/tc Apply and read-back before the new
// generation ACK. The ordinary signed documents remain untouched, and a fresh
// matching lease plus another verified Apply are required to serve.
func recoveryDecision(in Input) (Decision, bool) {
	if in.Desired == nil || in.Lease == nil {
		return Decide(in), false
	}
	desired, lease := *in.Desired, *in.Lease
	desired.DeviceBindings = nil
	lease.Members = nil
	lease.ConfigGeneration = desired.ConfigGeneration
	in.Desired, in.Lease = &desired, &lease
	d := Decide(in)
	return d, d.Serve
}

// Keep a closed-recovery Apply while its ordinary grant-less schedule waits.
// This never authorizes forwarding: require the unchanged empty model and a
// fresh kernel read under the node lock. All other failures take the baseline.
func (e *Engine) retainClosedRecovery(ctx context.Context) bool {
	if !e.closedRecoveryHeld || e.serving || e.applyError != nil || e.appliedModel == nil ||
		len(e.appliedModel.Bindings) != 0 || !e.baselineOK || e.retainedPending ||
		e.quarantinePending || e.memberPathPending || len(e.pendingRevoke) != 0 ||
		!grantsProvisional(e.input()) {
		return false
	}
	d, ok := recoveryDecision(e.input())
	if !ok || len(e.appliedModel.Leases) != 0 || e.appliedModel.Probe != nil {
		return false
	}
	// Empty models have no admission timeouts. A new lease anchor cannot
	// require reinstalling the same deny-all rules; compare every other input.
	d.Model.Now, d.Model.NowMono = e.appliedModel.Now, e.appliedModel.NowMono
	if !sameEnforcement(*e.appliedModel, d.Model, e.appliedAssignments) {
		return false
	}
	lock, err := nodelock.Acquire(ctx, e.cfg.LockPath, 3, 50*time.Millisecond)
	if err != nil {
		return false
	}
	defer lock.Release()
	e.planeLocked = true
	defer func() { e.planeLocked = false }()
	e.latch()
	if e.latchBlocked {
		return false
	}
	return e.verifyServingModel(ctx, *e.appliedModel, e.appliedAssignments)
}
