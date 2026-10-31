package agent

import (
	"context"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Adoption does not reset any kernel timeout. Both signed documents, the
// original same-boot lease anchor, the current release, live rules and meter
// ownership must still qualify. A reboot or uncertain state takes the baseline.
func (e *Engine) adoptStoredServing(ctx context.Context) bool {
	if e.lease == nil || e.leaseBoot != e.bootID || e.lease.NodeId != e.nodeID || e.lease.NodeIncarnationId != e.incID ||
		e.desired == nil || e.appliedModel == nil || e.appliedGen == 0 ||
		e.appliedGen != uint64(e.desired.ConfigGeneration) || e.desired.ConfigGeneration != e.lease.ConfigGeneration {
		return false
	}
	beat, err := watchdog.ReadHeartbeat(e.cfg.HeartbeatPath)
	now := e.clock.Mono()
	if err != nil || beat.BootID != e.bootID || !beat.Serving || !beat.BaselineOK || beat.Generation != e.appliedGen ||
		beat.Mono > now || now-beat.Mono > time.Duration(e.cfg.WatchdogStaleSeconds)*time.Second || beat.PolicyExpires <= now {
		return false
	}
	e.refreshDesired()
	e.leaseFresh, e.baselineOK = true, true
	// Preliminary authorization only; the probe and DNS are run below before
	// adopting, and every candidate is verified against the live kernel.
	in := e.input()
	in.ProbeRan, in.ProbeCheck = true, api.HealthCheckStatusPassed
	if !Decide(in).Serve {
		return false
	}
	lock, err := nodelock.TryAcquire(e.cfg.LockPath)
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
	e.evaluate(ctx)
	d := e.decision
	if !d.Serve || !e.verifyServingModel(ctx, d.Model, e.appliedAssignments) {
		return false
	}
	if !e.checkMemberPath(ctx, d.Model) {
		return false
	}
	// Attribute bytes accumulated while the process was absent BEFORE opening
	// the meter. Its same-boot continuity check retains the open epoch.
	e.setMeterBindings(d.Model.Bindings)
	e.ensureUsage(ctx)
	e.readMeter(ctx)
	e.decision = Decide(e.input())
	if e.meter == nil || !e.decision.Serve || !e.verifyServingModel(ctx, d.Model, e.appliedAssignments) {
		return false
	}
	e.appliedModel = &d.Model
	e.policyExpires = d.PolicyExpires
	e.serving = true
	e.expectServing.Store(true)
	e.resumeSampling()
	e.leaseRefreshPending = true // try central promptly, retain authorization on an outage
	e.log.Info("serving rules adopted", "generation", e.appliedGen, "policy_expires_mono_ns", int64(e.policyExpires))
	return true
}

// An ordinary process replacement leaves the live, expiring rules and open
// meter epoch intact. Updater barriers still require the full final hand-off.
// Kernel counters account for bytes until the successor resumes sampling;
// watchdog staleness and the original kernel timeout continue to bound service.
func (e *Engine) preserveServingOnStop(ctx context.Context) bool {
	if !e.serving || e.meter == nil || e.leaseBoot != e.bootID {
		return false
	}
	lock, err := nodelock.TryAcquire(e.cfg.LockPath)
	if err != nil {
		return false
	}
	defer lock.Release()
	e.latch()
	e.refreshDesired()
	if !Decide(e.input()).Serve || !e.verifyCurrentServing(ctx) {
		return false
	}
	if err := e.meter.CheckpointForRestart(ctx); err != nil {
		return false
	}
	e.log.Info("serving rules retained for restart", "generation", e.appliedGen, "policy_expires_mono_ns", int64(e.policyExpires))
	return true
}
