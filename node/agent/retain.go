package agent

import (
	"context"
	"reflect"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

// sameEnforcement compares every renderer input, including new fields added to
// Model. Clock samples advance, but their wall/monotonic mapping must not.
// Revoke is a consumed action checked separately; actual assigned classes
// replace the pre-Apply allocation hints in the retained model.
func sameEnforcement(applied, next enforce.Model, assignments map[uuid.UUID]uint16) bool {
	if applied.NowMono <= 0 || next.NowMono < applied.NowMono ||
		!applied.Now.Add(next.NowMono-applied.NowMono).Equal(next.Now) {
		return false
	}
	applied.Now, next.Now = time.Time{}, time.Time{}
	applied.NowMono, next.NowMono = 0, 0
	applied.QoS.Assignments = assignments
	applied.Revoke = nil
	return reflect.DeepEqual(applied, next)
}

// Called only under the node lock after rereading the updater latch. An audit
// model survives withdrawal/reboot, so its presence alone permits nothing.
func (e *Engine) canRetainServing(ctx context.Context, d Decision) bool {
	if !e.serving || !d.Serve || !e.baselineOK || !e.storageOK || e.applyError != nil ||
		e.failedCandidate != nil || e.appliedModel == nil || e.appliedSnapshot == nil ||
		e.leaseBoot != e.bootID || e.appliedGen != d.Model.Generation || e.policyExpires != d.PolicyExpires ||
		e.policyExpires <= e.clock.Mono() || e.latchBlocked || e.repairing ||
		len(e.pendingRevoke) != 0 || e.conntrackAttempts != 0 || e.conntrackBlocked ||
		len(planRevokes(nil, e.addresses, d.Model.Bindings)) != 0 ||
		e.memberPath == nil || e.memberPathPending || e.quarantinePending || e.probeCleanup != nil ||
		e.retainedPending || e.usagePersist || e.usageAdopting || e.epochClosed || e.finalDone ||
		e.finalBarrierID != "" || e.shutUsage || !e.sampleOn || !e.expectServing.Load() ||
		e.meter == nil || e.usage == nil || !sameUsageBindings(e.meterBindings(), d.Model.Bindings) ||
		(e.kernel != nil && e.kernel.nudgePending()) ||
		!sameEnforcement(*e.appliedModel, d.Model, e.appliedAssignments) {
		return false
	}
	// Same-boot counter resets do not change the counting interval. A final
	// sample, closed epoch or boot boundary still defeats retention. The caller
	// durably advances the intent to the current head before publishing a skip.
	h, err := e.usage.Head(ctx)
	if err != nil {
		return false
	}
	intent := e.restartIntent
	if intent == nil || h.OpenEpoch == uuid.Nil || h.BootID != e.bootID.String() ||
		intent.Boot != e.bootID || uint64(intent.Generation) != d.Model.Generation ||
		!sameUsageBindings(intent.bindings(), d.Model.Bindings) {
		return false
	}
	same, err := e.usage.SameCountingInterval(ctx, intent.Epoch, h.OpenEpoch, e.bootID.String())
	return err == nil && same
}
