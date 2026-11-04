package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

// r27BootMissingInterfacePlane models the post-restart namespace state in
// AT09's RestartNodeNamespace90 (docs/e2e/at/members/F18-AT09-restart.json):
//   - the namespace has been recreated so RestartGeneration returns an error
//     (no table inet fvpn yet);
//   - while missingInterface is true (before `netbird up` creates fvmb0),
//     VerifyServing's tc qdisc read fails with "read_failed"
//     (ReasonTCForeignFilter) and Quarantine's link-down fails with
//     "down_verify_failed".
type r27BootMissingInterfacePlane struct {
	*memPlane
	missingInterface bool
	foreignTC        bool
	quarantined      bool
	linkDown         bool
	quarantines      int
	verifyCalls      int
	baselineCalls    int
}

// The fixture supplies the same independent link inventory read as SystemPlane.
func (p *r27BootMissingInterfacePlane) InterfaceExists(context.Context, string) (bool, error) {
	return !p.missingInterface, nil
}

func (p *r27BootMissingInterfacePlane) RestartGeneration(context.Context) (uint64, error) {
	return 0, fmt.Errorf("no table inet fvpn")
}

func (p *r27BootMissingInterfacePlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	p.baselineCalls++
	return p.memPlane.Baseline(ctx, dp)
}

func (p *r27BootMissingInterfacePlane) VerifyServing(_ context.Context, m enforce.Model, _ map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	p.verifyCalls++
	if p.missingInterface {
		r := enforce.TCReadEvidence{
			Interface:  m.Datapath.MemberIngress,
			Attachment: "qdisc",
			Error:      "read_failed",
		}
		ev := enforce.ServingEvidence{
			Quarantined: p.quarantined,
			TC:          []enforce.TCReadEvidence{r},
			TCSummary: enforce.TCSummary{
				Generation: decstr.U64(m.Generation),
				Phase:      "verify",
				Reads:      1,
				Objects:    0,
				Rejected:   &r,
				SHA256:     strings.Repeat("0", 64),
			},
		}
		return ev, &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter, Detail: m.Datapath.MemberIngress + " qdisc: kind unreadable: read_failed"}
	}
	if p.foreignTC {
		obj := enforce.TCObject{Kind: "flower", Handle: "0x1", Actions: []string{"drop"}}
		r := enforce.TCReadEvidence{
			Interface:  m.Datapath.MemberIngress,
			Attachment: "ingress",
			Objects:    []enforce.TCObject{obj},
		}
		ev := enforce.ServingEvidence{
			Quarantined: p.quarantined,
			TC:          []enforce.TCReadEvidence{r},
			TCSummary: enforce.TCSummary{
				Generation: decstr.U64(m.Generation),
				Phase:      "verify",
				Reads:      4,
				Objects:    1,
				Rejected:   &r,
				SHA256:     strings.Repeat("b", 64),
			},
		}
		return ev, &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter, Detail: m.Datapath.MemberIngress + " ingress: foreign filter flower"}
	}
	ev := enforce.ServingEvidence{
		Quarantined: p.quarantined,
		TCSummary: enforce.TCSummary{
			Generation: decstr.U64(m.Generation),
			Phase:      "verify",
			Reads:      8,
			Objects:    4,
			SHA256:     strings.Repeat("a", 64),
		},
	}
	if p.quarantined {
		return ev, &enforce.CheckError{Reason: enforce.ReasonGenerationMismatch, Detail: "quarantined"}
	}
	return ev, nil
}

func (p *r27BootMissingInterfacePlane) Quarantine(_ context.Context, model enforce.Model) (enforce.QuarantineEvidence, error) {
	p.quarantines++
	p.baselineCalls++
	p.quarantined = true
	if p.missingInterface {
		return enforce.QuarantineEvidence{
			Quarantined: false,
			Recorded:    true,
			Links:       []enforce.QuarantineLink{{Interface: model.Datapath.MemberIngress, Error: "down_verify_failed"}},
			Baseline:    enforce.BaselineEvidence{Installed: true, Verified: true},
		}, errors.New("quarantine incomplete")
	}
	p.linkDown = true
	return enforce.QuarantineEvidence{
		Quarantined: true,
		Recorded:    true,
		Links: []enforce.QuarantineLink{{
			Interface: model.Datapath.MemberIngress,
			Before:    &enforce.LinkState{AdminUp: true, OperState: "UNKNOWN"},
			After:     &enforce.LinkState{AdminUp: false, OperState: "DOWN"},
		}},
		Baseline: enforce.BaselineEvidence{Installed: true, Verified: true},
	}, nil
}

func (p *r27BootMissingInterfacePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if p.foreignTC {
		q, _ := p.Quarantine(ctx, m)
		ev, err := p.VerifyServing(ctx, m, nil)
		ev.TCSummary.Phase = "pre_apply"
		return enforce.ApplyResult{Evidence: ev, Quarantine: &q}, err
	}
	res, err := p.memPlane.Apply(ctx, m, prev)
	if err != nil {
		return res, err
	}
	p.quarantined = false
	p.linkDown = false
	res.Evidence.TCSummary = enforce.TCSummary{
		Generation: decstr.U64(m.Generation),
		Phase:      "verify",
		Reads:      8,
		Objects:    4,
		SHA256:     strings.Repeat("a", 64),
	}
	return res, nil
}

func (p *r27BootMissingInterfacePlane) MemberPath(_ context.Context, _ enforce.Model, _ *MemberPath) (MemberPath, error) {
	if p.missingInterface || p.linkDown {
		return MemberPath{}, errMemberPath
	}
	return MemberPath{}, nil
}

type r27ToggleProbe struct {
	up bool
}

func (p *r27ToggleProbe) Probe(context.Context, ProbeRequest) ProbeResult {
	if !p.up {
		return ProbeResult{
			Ran:            true,
			Check:          api.HealthCheckStatusFailed,
			Health:         api.DataHealthFailed,
			Detail:         "Egress probe used no extra target.",
			NetbirdVersion: "0.76.2",
		}
	}
	return ProbeResult{
		Ran:            true,
		Check:          api.HealthCheckStatusPassed,
		Health:         api.DataHealthHealthy,
		Detail:         "Egress probe used no extra target.",
		NetbirdVersion: "0.76.2",
	}
}

// TestR27_H1_AuditBlockedMissingInterfaceAtBootLatchesFalseTCForeignFilter
// reproduces AT09 index 25's post-restart sequence on node2
// (docs/e2e/at/members/F18-AT09-restart.json, 21:49:53.318Z..21:49:58.590Z):
//  1. Before restart, node2 served generation 66 and persisted retained_applied.
//  2. On restart, fvpn-agent starts before netbird creates fvmb0:
//     tunnel_probe fails ("HEALTH_CHECK_FAILED") and fvmb0 does not exist.
//  3. Step 1's main body emits Observation 1 with safety_check_failed and
//     lease/qos/source_binding == passed (no TC_FOREIGN_FILTER).
//  4. Step 1's deferred finishServingStep -> auditBlocked (serving.go:180-220)
//     runs VerifyServing against retained_applied even though !e.serving and
//     tunnel_probe just failed. Missing fvmb0 causes tc qdisc show to fail
//     with "read_failed" -> ReasonTCForeignFilter, latching e.applyError =
//     TC_FOREIGN_FILTER, latching e.containment permanently, and running
//     e.quarantine (emitting Observation 2 with TC_FOREIGN_FILTER on
//     lease/qos/source_binding and stage=apply/tc_evidence/quarantine).
func TestR27_H1_AuditBlockedMissingInterfaceAtBootLatchesFalseTCForeignFilter(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.srv.SetFixture(generationFixture(66, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)
	if !r.eng.serving || r.eng.appliedGen != 66 {
		t.Fatalf("pre-restart setup: serving=%v applied=%d", r.eng.serving, r.eng.appliedGen)
	}

	// Clean stop (as in second_exit_test.go:90), then restart in a fresh
	// namespace where fvmb0 has not been created by netbird up yet.
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	plane := &r27BootMissingInterfacePlane{memPlane: r.plane, missingInterface: true}
	prober := &r27ToggleProbe{up: false}
	next, err := Start(Options{
		Config:      r.eng.cfg,
		Clock:       r.clock,
		Plane:       plane,
		MeterSource: staticSource{boot: r.eng.bootID.String()},
		Prober:      prober,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	r.eng = next
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}

	// Step 1 while fvmb0 is still missing (no foreign TC filter exists).
	stepOK(t, r)
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("missing post-restart observation")
	}

	// TODO(review r27): H-1 — When !e.serving and tunnel_probe failed because
	// fvmb0 does not exist yet at boot, finishServingStep -> auditBlocked
	// (node/agent/serving.go:180-185, 198-220) must not treat a missing-device
	// tc read_failed as ReasonTCForeignFilter, set e.applyError = TC_FOREIGN_FILTER,
	// latch e.containment for the life of the process, or quarantine fvmb0.
	if r.eng.applyError != nil && r.eng.applyError.Code == api.ErrorCodeTCFOREIGNFILTER {
		t.Fatalf("TODO(review r27): H-1: auditBlocked on missing interface at boot latched false applyError=%+v containment=%+v quarantines=%d report.Errors=%+v",
			r.eng.applyError, r.eng.containment, plane.quarantines, rep.Errors)
	}
	if next.serving || next.containment != nil || next.quarantinePending || plane.quarantines != 0 {
		t.Fatalf("missing interface was not simply blocked: serving=%v containment=%+v quarantines=%d", next.serving, next.containment, plane.quarantines)
	}
	// Once NetBird creates a clean interface, the next Apply can recover.
	plane.missingInterface = false
	prober.up = true
	stepOK(t, r)
	if !next.serving || next.applyError != nil || next.containment != nil || plane.quarantines != 0 {
		t.Fatalf("clean interface failed to recover: serving=%v error=%+v containment=%+v", next.serving, next.applyError, next.containment)
	}

}

// TestR27_M1_ProvisionalTimerRaceAndClosedRecoveryDelayPopulatedLease
// reproduces AT02's post-stimulus recovery sequence (seq=584..593 in
// docs/e2e/at/members/F18-AT02-recovery.json.gz):
//   - Step 1 after fault removal renews the lease at started + delta while
//     e.applyError != nil, receiving a 0-grant provisional lease and setting
//     provisionalRetryAt = monoSent + 5s;
//   - in the same Step 1, enforce runs D-136 closedRecovery (Apply 584,
//     Bindings = nil), clearing e.applyError and ACKing generation 13, while
//     renewAppliedLease is a no-op;
//   - Engine.Run's timer formula min(nextStepDelay(5s), remainingStepDelay(started, 5s))
//     (node/agent/engine.go:616) wakes at started + 5s (delta before
//     provisionalRetryAt), causing Step 2 to skip syncLease and execute a
//     redundant awaiting_member_grants baseline (seq=589) before Step 3 finally
//     renews the populated lease and runs the serving Apply (seq=593).
func TestR27_M1_ProvisionalTimerRaceAndClosedRecoveryDelayPopulatedLease(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.eng.cfg.PollSeconds = 5
	r.eng.cfg.HeartbeatSeconds = 5
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	plane := &r27BootMissingInterfacePlane{memPlane: r.plane}
	r.eng.plane = plane
	r.srv.SetFixture(generationFixture(12, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)
	if !r.eng.serving || r.eng.appliedGen != 12 {
		t.Fatalf("setup gen 12: serving=%v applied=%d", r.eng.serving, r.eng.appliedGen)
	}

	// Inject foreign TC filter during generation 12; central marks node not ready.
	plane.foreignTC = true
	stepOK(t, r)
	if r.eng.serving || r.eng.applyError == nil || r.eng.applyError.Code != api.ErrorCodeTCFOREIGNFILTER {
		t.Fatalf("expected TC_FOREIGN_FILTER withdrawal: serving=%v applyError=%+v", r.eng.serving, r.eng.applyError)
	}
	r.srv.SetMemberGrantsReady(false)

	// Remove foreign TC filter and advance desired state to generation 13.
	plane.foreignTC = false
	r.srv.SetFixture(generationFixture(13, oneBinding(r.clock.Wall())))

	// Simulate Engine.Run (node/agent/engine.go:609-616): started is captured at
	// Step entry, and syncDesired takes 11ms before syncLease captures monoSent.
	started := r.clock.Mono()
	r.clock.Advance(11 * time.Millisecond)
	stepOK(t, r)

	// Step 1 ran closedRecovery (Apply with 0 bindings), cleared applyError,
	// ACKed generation 13, and flushed the ready observation to central.
	r.srv.SetMemberGrantsReady(true)

	// Engine.Run sleeps min(nextStepDelay(5s), remainingStepDelay(started, 5s)).
	interval := 5 * time.Second
	// Call the production Run timer calculation, rather than copying its formula.
	sleep := r.eng.runStepDelay(started, interval)
	r.clock.Advance(sleep)
	stepOK(t, r)

	// TODO(review r27): M-1 — After closedRecovery clears e.applyError and the
	// next Run tick wakes, the agent should renew its provisional lease and
	// restore serving on that tick instead of waking 11ms before
	// provisionalRetryAt, skipping syncLease, and running a redundant
	// awaiting_member_grants baseline step.
	if !r.eng.serving || len(r.eng.lease.Members) == 0 {
		t.Fatalf("TODO(review r27): M-1: Run wakeup min(nextStepDelay, remainingStepDelay)=%s woke %s before provisionalRetryAt; node stayed non-serving (%v) on 0-grant lease instead of renewing populated lease",
			sleep, r.eng.provisionalRetryAt-r.clock.Mono(), r.eng.decision.Reasons)
	}
}

// TestR27_L1_ForeignTCQuarantineStepRedundantBaselinesAndReQuarantine proves
// that a single Step detecting ReasonTCForeignFilter executes Quarantine twice
// and installs the baseline 4 times (matching seq=553..557, 560..565, 568..573,
// 576..581 in docs/e2e/at/members/F18-AT02-recovery.json.gz) because
// finishServingStep -> auditBlocked (node/agent/serving.go:180-185, 217-219)
// re-arms e.quarantinePending = true when ev.Quarantined is true even though
// e.quarantine already ran earlier in the same Step.
func TestR27_L1_ForeignTCQuarantineStepRedundantBaselinesAndReQuarantine(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	plane := &r27BootMissingInterfacePlane{memPlane: r.plane}
	r.eng.plane = plane
	stepOK(t, r)

	plane.quarantines = 0
	plane.baselineCalls = 0
	plane.foreignTC = true
	stepOK(t, r)

	// TODO(review r27): L-1 — A single non-serving quarantine Step should run
	// Quarantine once (not twice) and avoid 4 back-to-back baseline installs
	// in the same Step.
	if plane.quarantines > 1 || plane.baselineCalls > 2 {
		t.Fatalf("TODO(review r27): L-1: single foreign-TC Step ran Quarantine %d times and Baseline %d times (want 1 Quarantine)",
			plane.quarantines, plane.baselineCalls)
	}
	if plane.quarantines != 1 || plane.baselineCalls != 1 {
		t.Fatalf("redundant containment: quarantines=%d baselines=%d", plane.quarantines, plane.baselineCalls)
	}
	// The next Step owns a new allowance and must recheck containment.
	checks := plane.verifyCalls
	stepOK(t, r)
	if plane.verifyCalls <= checks || plane.quarantines != 2 || plane.baselineCalls != 2 {
		t.Fatalf("next Step containment: checks=%d->%d quarantine=%d baseline=%d", checks, plane.verifyCalls, plane.quarantines, plane.baselineCalls)
	}

}

// TestR27_Held_SameGenerationRestartRenewsBeforeObserveAndSkipsProvisionalWhenPopulated
// proves why AT09 index 25 observed no stage=lease AWAITING_MEMBER_GRANTS error:
//   - Decide (node/agent/decide.go:154-261, 360-371) never skips lease
//     evaluation when a safety check fails;
//   - when the lease has 0 member grants (grantsProvisional == true), Decide
//     emits both awaiting_member_grants and safety_check_failed with
//     stage=lease AWAITING_MEMBER_GRANTS (D-126);
//   - however, on a clean same-generation restart after #376 (no observation
//     POST on SIGTERM), central remains ready, and Step 1 calls syncLease
//     (node/agent/engine.go:749) before observe/flush (engine.go:778-779),
//     receiving a populated lease (len(Members) > 0) so grantsProvisional is
//     false and Decide truthfully reports no stage=lease error.
func TestR27_Held_SameGenerationRestartRenewsBeforeObserveAndSkipsProvisionalWhenPopulated(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.srv.SetFixture(generationFixture(66, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)

	obsBeforeClose := r.srv.ObservationPosts()
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	if got := r.srv.ObservationPosts(); got != obsBeforeClose {
		t.Fatalf("Close posted %d observations after r18 M-1 fix, want 0", got-obsBeforeClose)
	}

	// Restart in a namespace where kernel rules are absent (Boot installs
	// boot_unverified baseline) and tunnel_probe fails on Step 1, while central
	// still issues a populated lease because no failure observation was sent.
	plane := &r27BootMissingInterfacePlane{memPlane: r.plane}
	prober := &r27ToggleProbe{up: false}
	next, err := Start(Options{
		Config:      r.eng.cfg,
		Clock:       r.clock,
		Plane:       plane,
		MeterSource: staticSource{boot: r.eng.bootID.String()},
		Prober:      prober,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	r.eng = next
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.leaseFresh || next.serving {
		t.Fatalf("post-Boot state: leaseFresh=%v serving=%v, want false/false", next.leaseFresh, next.serving)
	}

	renewalsBefore := r.srv.LeaseRenewals()
	stepOK(t, r)
	if r.srv.LeaseRenewals() != renewalsBefore+1 || len(next.lease.Members) == 0 {
		t.Fatalf("Step 1 lease renewal: renewals=%d->%d members=%d, want +1 populated lease",
			renewalsBefore, r.srv.LeaseRenewals(), len(next.lease.Members))
	}
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("missing Step 1 observation")
	}
	if !reflect.DeepEqual(rep.ForwardingBlockReasons, []api.ForwardingBlockReason{api.ForwardingBlockReasonSafetyCheckFailed}) {
		t.Fatalf("populated-lease failed-probe reasons = %v, want [safety_check_failed]", rep.ForwardingBlockReasons)
	}
	for _, e := range rep.Errors {
		if e.Stage == "lease" {
			t.Fatalf("unexpected stage=lease error on populated lease: %+v", e)
		}
	}

	// By contrast, on the exact same failed-probe input with an empty lease
	// (len(Members) == 0), Decide evaluates the lease check and emits
	// stage=lease AWAITING_MEMBER_GRANTS alongside safety_check_failed.
	in := next.input()
	emptyLease := *in.Lease
	emptyLease.Members = nil
	in.Lease = &emptyLease
	d := Decide(in)
	wantReasons := []api.ForwardingBlockReason{
		api.ForwardingBlockReasonAwaitingMemberGrants,
		api.ForwardingBlockReasonSafetyCheckFailed,
	}
	if !reflect.DeepEqual(d.Reasons, wantReasons) {
		t.Fatalf("empty-lease failed-probe reasons = %v, want %v", d.Reasons, wantReasons)
	}
	foundLeaseStage := false
	for _, e := range d.Errors {
		if e.Stage == "lease" && e.Code == api.ErrorCodeAWAITINGMEMBERGRANTS {
			foundLeaseStage = true
		}
	}
	if !foundLeaseStage {
		t.Fatalf("Decide omitted stage=lease AWAITING_MEMBER_GRANTS when Lease.Members is empty: %+v", d.Errors)
	}
}

// TestR27_Held_D106ContainmentRetainsFirstRejectionAcrossGenerationsUntilRestart
// proves why AT09 index 26's pre-restart observation (21:48:44.725565Z at
// generation 66) carried tc_evidence with candidate_generation="21" from
// 21:24:04Z (populated_hold_test.go:191-214) and STALE_GENERATION
// ("applied=66 lease=65"), and why candidate_generation="21" disappeared after
// RestartNodeNamespace90 replaced the process.
func TestR27_Held_D106ContainmentRetainsFirstRejectionAcrossGenerationsUntilRestart(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	plane := &r27BootMissingInterfacePlane{memPlane: r.plane}
	r.eng.plane = plane
	r.srv.SetFixture(generationFixture(21, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)

	// Inject foreign TC filter at generation 21 (as in populated_hold_test.go:191-214).
	plane.foreignTC = true
	stepOK(t, r)
	if r.eng.containment == nil || r.eng.containment.Generation != 21 {
		t.Fatalf("containment after gen 21 rejection = %+v, want Generation=21", r.eng.containment)
	}
	firstReadAt := r.eng.containment.ReadAt

	// Remove filter and recover to generation 65, then advance desired to 66
	// while the lease remains at generation 65 (producing STALE_GENERATION
	// "applied=66 lease=65" alongside D-106 candidate_generation="21").
	plane.foreignTC = false
	r.srv.SetFixture(generationFixture(65, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)
	if !r.eng.serving || r.eng.appliedGen != 65 {
		t.Fatalf("recovery at gen 65: serving=%v applied=%d", r.eng.serving, r.eng.appliedGen)
	}
	r.srv.SetLeaseGeneration(65)
	r.srv.SetFixture(generationFixture(66, oneBinding(r.clock.Wall())))
	stepOK(t, r)
	if r.eng.appliedGen != 66 {
		t.Fatalf("appliedGen = %d, want 66", r.eng.appliedGen)
	}
	if r.eng.containment == nil || r.eng.containment.Generation != 21 || !r.eng.containment.ReadAt.Equal(firstReadAt) {
		t.Fatalf("D-106 historical rejection altered at gen 66: %+v", r.eng.containment)
	}

	rep := mustReport(t, r, t.Context())
	raw, _ := json.Marshal(rep.Errors)
	if !strings.Contains(string(raw), `\"candidate_generation\":\"21\"`) || !strings.Contains(string(raw), `applied=66 lease=65`) {
		t.Fatalf("expected retained D-106 candidate_generation 21 and STALE_GENERATION applied=66 lease=65 in report errors: %s", string(raw))
	}

	// Process restart clears in-memory e.containment.
	restartEngine(t, r, plane)
	if r.eng.containment != nil {
		t.Fatalf("expected nil in-memory containment after process restart, got %+v", r.eng.containment)
	}
}
