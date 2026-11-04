package agent

// Diagnosis review r28 proof tests for F20-01 (early tunnel_probe /
// HEALTH_CHECK_FAILED baseline during AT13 short whole-node lease expiry
// under control-plane stop in isolation follow-up 20).
// TEST ONLY in-memory data planes and local fakecontrol; no host networking or root.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

const r28HealthyNetbirdJSON = `{"daemonVersion":"0.82.0","management":{"url":"https://netbird.example.invalid:443","connected":true},"signal":{"url":"https://netbird.example.invalid:443","connected":true},"relays":{"total":1,"available":1}}`

// ctxAwareNetbirdProber constructs a production netbirdProber whose CLI runner
// and TCP dialer honor ctx.Err() the same way exec.CommandContext (probe.go:190)
// and net.Dialer.DialContext (probe.go:204) do in production, while returning a
// healthy NetBird status payload whenever ctx is still live.
func ctxAwareNetbirdProber(seenCtxErr *error) netbirdProber {
	return netbirdProber{
		run: func(ctx context.Context) ([]byte, error) {
			if seenCtxErr != nil {
				*seenCtxErr = ctx.Err()
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return []byte(r28HealthyNetbirdJSON), nil
		},
		dial: func(ctx context.Context, _ string) error {
			return ctx.Err()
		},
		iface: func(string) error {
			return nil
		},
	}
}

// TODO(review r28): H-1
// TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry reproduces F20-01
// in a unit test against Engine.Step:
//  1. The node is enrolled, published, and serving generation 2 under a signed
//     60-second serving lease with a healthy NetBird data path.
//  2. At D - 46s, the control plane stops answering (modeled by fakecontrol
//     holding the HTTP request until the per-step controlContext(ctx) deadline
//     expires, matching GET /api/v1/agent/desired-state timing out after 5s
//     when fvpn-control is stopped on the lab bridge).
//  3. Because refreshProbe (node/agent/engine.go:948) passes controlContext(ctx)
//     into e.prober.Probe(...), netbirdProber.Probe inherits the already-expired
//     control HTTP context from syncDesired(ctx). Its exec.CommandContext read
//     fails immediately with context.DeadlineExceeded without running netbird,
//     judgeProbe reports (DataHealthFailed, HealthCheckStatusFailed) with detail
//     "overlay_down", Decide emits safety_check_failed / HEALTH_CHECK_FAILED on
//     tunnel_probe, and Step installs the fail-closed baseline 46s before D.
func TestR28_ControlOutageDoesNotFailTunnelProbeBeforeLeaseExpiry(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.StickyLease(false)

	var probeCtxErr error
	r.eng.prober = ctxAwareNetbirdProber(&probeCtxErr)

	enrollBoot(t, r)
	binding := oneBinding(r.clock.Wall())
	binding.ValidUntil = r.clock.Wall().Add(time.Hour)
	fix := generationFixture(2, binding)
	fix.TTL = time.Hour
	r.srv.SetFixture(fix)
	r.srv.SetMaxOfflineSeconds(60)
	stepOK(t, r)
	r.manualUsage(t)

	if !r.eng.serving || r.eng.appliedGen != 2 {
		t.Fatalf("precondition: expected serving gen 2, got serving=%v gen=%d reasons=%v", r.eng.serving, r.eng.appliedGen, r.eng.decision.Reasons)
	}
	if probeCtxErr != nil {
		t.Fatalf("precondition: expected nil probeCtxErr while control is up, got %v", probeCtxErr)
	}
	deadline := r.eng.policyExpires
	remaining := deadline - r.clock.Mono()
	if remaining != 60*time.Second {
		t.Fatalf("precondition: expected 60s lease deadline, got remaining=%s", remaining)
	}

	// Advance to D - 46s (14s into the 60s lease, matching wholeNodeExpiry
	// stopping control at 00:25:57Z for lease issued_at 00:25:43.406Z).
	r.clock.Advance(14 * time.Second)

	// Shorten WatchdogStaleSeconds to 1s so controlPassTimeout() = min(5s, stale/3)
	// is 333ms in the unit test while exercising the exact ioCtx timeout path in Engine.Step.
	r.eng.cfg.WatchdogStaleSeconds = 1
	r.srv.SetDelay(5 * time.Second)
	defer r.srv.SetDelay(0)

	baselinesBefore := r.plane.baselines
	stepOK(t, r)

	// Under §16.3, §17.1, §13.9, and node/agent/serving.go:238-239, a control-plane
	// HTTP timeout must not cancel the local NetBird status probe or tear down
	// member forwarding while 46s remain on the signed serving lease.
	if probeCtxErr != nil || !r.eng.serving {
		probeCheck, _ := checkBy(r.eng.decision.Checks, api.NodeCheckNameTunnelProbe)
		beat := mustBeat(t, r)
		t.Fatalf("H-1: refreshProbe passed expired controlContext(ctx) (%v) to Prober.Probe after syncDesired timed out: serving=%v safety=%s dataHealth=%s reasons=%v probeDetail=%q probeCheckStatus=%s errors=%+v baselines=%d->%d heartbeat={serving:%v reason:%s}",
			probeCtxErr, r.eng.serving, r.eng.decision.Safety, r.eng.decision.DataHealth, r.eng.decision.Reasons, r.eng.lastProbe.Detail, probeCheck.Status, r.eng.decision.Errors, baselinesBefore, r.plane.baselines, beat.Serving, beat.Reason)
	}
}

// TestR28_WhatHeld_RefreshProbeAndNetbirdProberContextPropagation proves the
// exact boundary between controlContext(ctx) and localContext(ctx) in Engine.Step
// and netbirdProber.Probe:
//  1. When ioCtx (attached under controlBudgetKey{}) has timed out, controlContext(ctx)
//     has Err() == context.DeadlineExceeded while localContext(ctx) and the parent
//     step ctx remain live (Err() == nil).
//  2. netbirdProber.Probe(localContext(ctx), ...) succeeds for both CLI status reads
//     and egress TCP dial probes, returning Check=passed, Health=healthy, Detail="overlay_up".
//  3. Even when NetbirdStatusFile is used (where os.Open ignores ctx), passing an
//     expired controlContext(ctx) still fails netbirdProber.Probe whenever an egress
//     probe target is present because dialAddr (node/agent/probe.go:136, 198-204)
//     also inherits ctx.
func TestR28_WhatHeld_RefreshProbeAndNetbirdProberContextPropagation(t *testing.T) {
	stepCtx := t.Context()
	ioCtx, cancelIO := context.WithTimeout(stepCtx, time.Nanosecond)
	defer cancelIO()
	time.Sleep(time.Millisecond)

	ctx := context.WithValue(stepCtx, controlBudgetKey{}, ioCtx)
	localCtx, cancelLocal := newLocalBudget(ctx, 10*time.Second)
	defer cancelLocal()
	ctx = localCtx

	if controlContext(ctx).Err() != context.DeadlineExceeded {
		t.Fatalf("expected expired controlContext(ctx), got %v", controlContext(ctx).Err())
	}
	if localContext(ctx).Err() != nil || stepCtx.Err() != nil {
		t.Fatalf("expected live localContext and stepCtx, got local=%v step=%v", localContext(ctx).Err(), stepCtx.Err())
	}

	var seenErr error
	p := ctxAwareNetbirdProber(&seenErr)
	req := ProbeRequest{Published: true, Egress: "192.0.2.10:443", Ingress: "fv-fh", Uplink: "fv-ih"}

	// With controlContext(ctx), the CLI read fails immediately with DeadlineExceeded.
	resControl := p.Probe(controlContext(ctx), req)
	if seenErr != context.DeadlineExceeded || resControl.Check != api.HealthCheckStatusFailed || resControl.Health != api.DataHealthFailed || resControl.Detail != "overlay_down" {
		t.Fatalf("unexpected controlContext probe result: seenErr=%v res=%+v", seenErr, resControl)
	}

	// With localContext(ctx), both the CLI read and egress dial succeed.
	resLocal := p.Probe(localContext(ctx), req)
	if seenErr != nil || resLocal.Check != api.HealthCheckStatusPassed || resLocal.Health != api.DataHealthHealthy || resLocal.Detail != "overlay_up" {
		t.Fatalf("unexpected localContext probe result: seenErr=%v res=%+v", seenErr, resLocal)
	}
}

// TestR28_WhatHeld_FastControlRefusalPreservesServingUntilDeadline proves that
// when control-plane requests fail fast without exhausting the 5s controlContext(ctx)
// budget (for example, an immediate HTTP 503), controlContext(ctx) is still live
// when refreshProbe runs, so netbirdProber.Probe succeeds, the node keeps serving
// throughout the remaining lease lifetime, and transitions cleanly to lease_expired
// only at the monotonic lease deadline D.
func TestR28_WhatHeld_FastControlRefusalPreservesServingUntilDeadline(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.StickyLease(false)

	var probeCtxErr error
	r.eng.prober = ctxAwareNetbirdProber(&probeCtxErr)

	enrollBoot(t, r)
	binding := oneBinding(r.clock.Wall())
	binding.ValidUntil = r.clock.Wall().Add(time.Hour)
	fix := generationFixture(2, binding)
	fix.TTL = time.Hour
	r.srv.SetFixture(fix)
	r.srv.SetMaxOfflineSeconds(60)
	stepOK(t, r)
	r.manualUsage(t)

	deadline := r.eng.policyExpires
	baselinesAtServing := r.plane.baselines

	// Fail all subsequent control requests immediately (HTTP 503) so ioCtx
	// still has remaining budget when refreshProbe runs.
	r.srv.FailNext(100)
	r.srv.FailLeases(100)

	// Step at D - 46s and D - 4s: serving stays up and no baseline is written.
	for _, remaining := range []time.Duration{46 * time.Second, 4 * time.Second} {
		r.clock.Advance(deadline - r.clock.Mono() - remaining)
		stepOK(t, r)
		if probeCtxErr != nil || !r.eng.serving || r.plane.baselines != baselinesAtServing {
			t.Fatalf("at remaining=%s: probeCtxErr=%v serving=%v baselines=%d->%d reasons=%v",
				remaining, probeCtxErr, r.eng.serving, baselinesAtServing, r.plane.baselines, r.eng.decision.Reasons)
		}
		beat := mustBeat(t, r)
		if !beat.Serving || beat.Reason != "serving" || beat.PolicyExpires != deadline {
			t.Fatalf("at remaining=%s: unexpected heartbeat %+v", remaining, beat)
		}
	}

	// Advance to exact deadline D: Step withdraws serving with lease_expired while
	// tunnel_probe remains passed, Safety remains passed, and DataHealth remains healthy.
	r.clock.Advance(deadline - r.clock.Mono())
	stepOK(t, r)
	if r.eng.serving {
		t.Fatal("expected serving=false at lease deadline D")
	}
	if r.eng.decision.Safety != api.SafetyStatusPassed || r.eng.decision.DataHealth != api.DataHealthHealthy {
		t.Fatalf("at D: safety=%s dataHealth=%s, want passed/healthy", r.eng.decision.Safety, r.eng.decision.DataHealth)
	}
	if len(r.eng.decision.Reasons) != 1 || r.eng.decision.Reasons[0] != api.ForwardingBlockReasonLeaseExpired {
		t.Fatalf("at D: reasons=%v, want [lease_expired]", r.eng.decision.Reasons)
	}
	probeCheck, ok := checkBy(r.eng.decision.Checks, api.NodeCheckNameTunnelProbe)
	if !ok || probeCheck.Status != api.HealthCheckStatusPassed {
		t.Fatalf("at D: tunnel_probe check=%+v, want passed", probeCheck)
	}
	leaseChk, ok := checkBy(r.eng.decision.Checks, api.NodeCheckNameLease)
	if !ok || leaseChk.Status != api.HealthCheckStatusFailed || codeOf(t, leaseChk) != api.ErrorCodeLEASEEXPIRED {
		t.Fatalf("at D: lease check=%+v, want failed/LEASE_EXPIRED", leaseChk)
	}
	beat := mustBeat(t, r)
	if beat.Serving || beat.Reason != string(api.ForwardingBlockReasonLeaseExpired) {
		t.Fatalf("at D: heartbeat=%+v, want serving=false reason=lease_expired", beat)
	}
}

// TestR28_WhatHeld_PostDeadlineReasonOrderingInDecideAndBaseline proves why
// F20-01's agent log recorded cause=safety_check_failed on every step from
// 00:26:02.373Z to 00:26:43.299Z (before D=00:26:43.406Z) and then switched to
// cause=lease_expired at 00:26:48.301Z (after D) even though DesiredState still
// timed out controlContext(ctx) on that post-deadline step:
//  1. Before D, leaseOK is true and ProbeCheck is failed, so Decide produces
//     Reasons=[safety_check_failed] and clears d.PolicyExpires=0 (decide.go:267),
//     and baseline/enforce clear e.policyExpires=0 (engine.go:1074, 1145).
//     baselineReason() (measurement.go:88-89) therefore returns "safety_check_failed".
//  2. After D, leaseOK is false and ProbeCheck is still failed. Decide appends
//     lease_expired first (decide.go:201) and safety_check_failed second
//     (decide.go:248), so Reasons=[lease_expired, safety_check_failed]. Because
//     e.policyExpires is already 0, baselineReason() skips "policy_deadline"
//     (measurement.go:86-87) and returns Reasons[0] == "lease_expired".
func TestR28_WhatHeld_PostDeadlineReasonOrderingInDecideAndBaseline(t *testing.T) {
	in := withAdmittedMember(servingInput())
	in.ProbeRan = true
	in.ProbeCheck = api.HealthCheckStatusFailed
	in.ProbeHealth = api.DataHealthFailed

	// 1. Before D: lease is in force, probe failed -> Reasons = [safety_check_failed].
	dBefore := Decide(in)
	if dBefore.Serve || dBefore.PolicyExpires != 0 {
		t.Fatalf("before D: Serve=%v PolicyExpires=%v", dBefore.Serve, dBefore.PolicyExpires)
	}
	if len(dBefore.Reasons) != 1 || dBefore.Reasons[0] != api.ForwardingBlockReasonSafetyCheckFailed {
		t.Fatalf("before D: Reasons=%v, want [safety_check_failed]", dBefore.Reasons)
	}
	eng := &Engine{
		clock:         timeanchor.NewManual(in.Now, in.MonoNow),
		decision:      dBefore,
		policyExpires: dBefore.PolicyExpires,
	}
	if got := eng.baselineReason(); got != "safety_check_failed" {
		t.Fatalf("before D: baselineReason()=%q, want %q", got, "safety_check_failed")
	}

	// 2. After D: lease has expired and probe still fails -> Reasons = [lease_expired, safety_check_failed].
	inAfter := in
	inAfter.MonoNow = in.LeaseMonoSent + in.Lease.ExpiresAt.Sub(in.Lease.IssuedAt) + time.Second
	dAfter := Decide(inAfter)
	if len(dAfter.Reasons) != 2 || dAfter.Reasons[0] != api.ForwardingBlockReasonLeaseExpired || dAfter.Reasons[1] != api.ForwardingBlockReasonSafetyCheckFailed {
		t.Fatalf("after D: Reasons=%v, want [lease_expired, safety_check_failed]", dAfter.Reasons)
	}
	eng.clock = timeanchor.NewManual(inAfter.Now, inAfter.MonoNow)
	eng.decision = dAfter
	if got := eng.baselineReason(); got != "lease_expired" {
		t.Fatalf("after D: baselineReason()=%q, want %q", got, "lease_expired")
	}
	if !strings.Contains(dAfter.Errors[0].Message, "tunnel_probe") || dAfter.Errors[0].Code != api.ErrorCodeHEALTHCHECKFAILED {
		t.Fatalf("after D: expected HEALTH_CHECK_FAILED error retained alongside lease_expired, got %+v", dAfter.Errors)
	}
}
