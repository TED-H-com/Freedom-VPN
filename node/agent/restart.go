package agent

import (
	"context"
	"maps"
	"net/netip"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Adoption does not reset any kernel timeout. Both signed documents, the
// original same-boot lease anchor, the current release, live rules and meter
// ownership must still qualify. A reboot or uncertain state takes the baseline.
func (e *Engine) adoptStoredServing(ctx context.Context, beat watchdog.Beat) bool {
	if e.lease == nil || e.leaseBoot != e.bootID || e.lease.NodeId != e.nodeID || e.lease.NodeIncarnationId != e.incID ||
		e.desired == nil || e.appliedModel == nil || e.appliedGen == 0 ||
		e.appliedGen != uint64(e.desired.ConfigGeneration) || e.desired.ConfigGeneration != e.lease.ConfigGeneration {
		return false
	}
	now := e.clock.Mono()
	if beat.BootID != e.bootID || !beat.Serving || !beat.BaselineOK || beat.Generation != e.appliedGen ||
		beat.Mono > now || now-beat.Mono > time.Duration(e.cfg.WatchdogStaleSeconds)*time.Second || beat.PolicyExpires <= now {
		return false
	}
	installed := e.appliedModel
	// retained_applied is written before checkMemberPath and the ACK. A crash
	// in that window leaves it ahead of applied_snapshot; the stored desired
	// generation is then also ahead and the check above already refuses. Keep
	// the invariant local: only the acknowledged installation is adoptable
	// (r20 L-1).
	if installed.Generation != e.appliedGen {
		return false
	}
	// nft retains both the initial timeout and the decreasing expires value.
	// Verify the former against the actual installation sample, while the
	// latter is still checked at the live read against the signed deadlines.
	// Rebuild all authorization fields below; only reuse the sample, and only
	// when it belongs to this same-boot lease anchor. These checks need only
	// the retained model and the stored lease, so they run before any lock
	// wait or network I/O (r20 M-2).
	if installed.NowMono < e.leaseMono || installed.NowMono > now ||
		!installed.Now.Equal(e.lease.IssuedAt.Add(installed.NowMono-e.leaseMono)) {
		return false
	}
	e.refreshDesired()
	e.leaseFresh, e.baselineOK = true, true
	// Preliminary authorization only; the probe and DNS are run below before
	// adopting, and every candidate is verified against the live kernel.
	in := e.input()
	in.ProbeRan, in.ProbeCheck = true, api.HealthCheckStatusPassed
	in.DNSFailed = false // applyResolver below supplies this process's fresh result
	prelim := Decide(in)
	if !prelim.Serve {
		return false
	}
	// Adoption never installs the candidate. Any changed deny/host set,
	// including a shrink, requires Boot's baseline followed by a full Apply.
	// Compare with retained_applied before replacing that installed model.
	// The sets come from local configuration, so the preliminary candidate
	// already carries them: refuse before the node lock, the live probe and
	// the resolver refresh, while the old rules are still forwarding (r20 M-2).
	if !sameRestartSets(*installed, prelim.Model) {
		return false
	}
	// A healthy watchdog briefly holds this same lock for its presence read.
	// Contention alone must not turn a process replacement into withdrawal.
	lock, err := nodelock.Acquire(localWorkContext(ctx), e.cfg.LockPath, 201, 10*time.Millisecond)
	if err != nil {
		return false
	}
	defer lock.Release()
	// This process now holds the node lock; updateLockHeld must not report
	// that descriptor as the updater's (D-064).
	e.planeLocked = true
	defer func() { e.planeLocked = false }()
	e.latch()
	if e.latchBlocked {
		return false
	}
	e.evaluate(ctx)
	// A stop request during the probe or resolver refresh leaves no fresh
	// evidence to adopt on; Boot takes the baseline under its local budget.
	if localWorkContext(ctx).Err() != nil || ctx.Err() != nil {
		return false
	}
	d := e.decision
	// Re-check against the evaluated candidate before verifying it.
	if !sameRestartSets(*installed, d.Model) {
		return false
	}
	d.Model.Now, d.Model.NowMono = installed.Now, installed.NowMono
	if !d.Serve || !e.verifyServingModel(ctx, d.Model, e.appliedAssignments) {
		return false
	}
	if !e.checkMemberPath(ctx, d.Model) {
		return false
	}
	// Prepare the meter read-only. The last verification can still refuse
	// adoption; it must not leave an epoch, new bindings or an uploader behind
	// (r22 M-2). Reconciliation that needs a write takes Boot's baseline path.
	prepared := !e.usageOnce
	e.usageAdopting = true
	e.ensureUsage(ctx)
	defer func() { e.usageAdopting = false }()
	e.readMeter(ctx)
	e.decision = Decide(e.input())
	usageKnown := false
	if e.usage != nil {
		if h, err := e.usage.Head(ctx); err == nil {
			bindings, err := e.restartHeadBindings(ctx, e.usage, h)
			usageKnown = err == nil && sameUsageBindings(bindings, d.Model.Bindings)
		}
	}
	if localWorkContext(ctx).Err() != nil || e.meter == nil || !usageKnown || !e.decision.Serve || !e.verifyServingModel(ctx, d.Model, e.appliedAssignments) {
		if prepared {
			e.discardUsagePreparation()
		}
		return false
	}
	e.usageAdopting = false
	e.setMeterBindings(d.Model.Bindings)
	e.activateUsage(ctx)
	e.appliedModel = &d.Model
	e.policyExpires = d.PolicyExpires
	e.serving = true
	e.expectServing.Store(true)
	e.resumeSampling()
	e.leaseRefreshPending = true // try central promptly, retain authorization on an outage
	e.log.Info("serving rules adopted", "generation", e.appliedGen, "policy_expires_mono_ns", int64(e.servingDeadline()), "next_binding_expiry_mono_ns", int64(e.policyExpires))
	return true
}

func sameRestartSets(installed, candidate enforce.Model) bool {
	// A non-nil empty DeniedV4 is not the spec default: enforce rejects it,
	// so it can never describe installed rules. Never call it equal (r20 L-1).
	if (installed.DeniedV4 != nil && len(installed.DeniedV4) == 0) ||
		(candidate.DeniedV4 != nil && len(candidate.DeniedV4) == 0) {
		return false
	}
	// nil DeniedV4 means the spec defaults, not an empty deny set.
	if installed.DeniedV4 == nil {
		installed.DeniedV4 = enforce.SpecDeniedV4()
	}
	if candidate.DeniedV4 == nil {
		candidate.DeniedV4 = enforce.SpecDeniedV4()
	}
	return sameRestartPrefixes(installed.DeniedV4, candidate.DeniedV4) &&
		sameRestartPrefixes(installed.ExtraDeniedV4, candidate.ExtraDeniedV4) &&
		sameRestartPrefixes(installed.HostV4, candidate.HostV4)
}

func sameRestartPrefixes(a, b []netip.Prefix) bool {
	sets := [2]map[netip.Prefix]bool{}
	for i, prefixes := range [][]netip.Prefix{a, b} {
		sets[i] = make(map[netip.Prefix]bool, len(prefixes))
		for _, p := range prefixes {
			if !p.IsValid() || !p.Addr().Is4() {
				return false
			}
			sets[i][p.Masked()] = true
		}
	}
	return maps.Equal(sets[0], sets[1])
}

// An ordinary process replacement leaves the live, expiring rules and open
// meter epoch intact. Updater barriers still require the full final hand-off.
// Kernel counters account for bytes until the successor resumes sampling;
// watchdog staleness and the original kernel timeout continue to bound service.
func (e *Engine) preserveServingOnStop(ctx context.Context) bool {
	if !e.serving || e.meter == nil || e.leaseBoot != e.bootID {
		return false
	}
	lock, err := nodelock.Acquire(ctx, e.cfg.LockPath, 201, 10*time.Millisecond)
	if err != nil {
		return false
	}
	defer lock.Release()
	e.planeLocked = true
	defer func() { e.planeLocked = false }()
	e.latch()
	e.refreshDesired()
	if !Decide(e.input()).Serve || !e.verifyCurrentServing(ctx) {
		return false
	}
	if err := e.meter.CheckpointForRestart(ctx); err != nil {
		return false
	}
	e.log.Info("serving rules retained for restart", "generation", e.appliedGen, "policy_expires_mono_ns", int64(e.servingDeadline()), "next_binding_expiry_mono_ns", int64(e.policyExpires))
	return true
}
