package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestF32_AT13_CloseDeadlineRetainsLedgerUntilLoopsFinish(t *testing.T) {
	r := barrierRig(t)
	finished := make(chan struct{})
	r.eng.loops.Add(1)
	go func() { defer r.eng.loops.Done(); <-finished }()
	start := time.Now()
	err := r.eng.Close()
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 4*time.Second {
		close(finished)
		<-r.eng.closeDone
		t.Fatalf("Close did not return on its three-second budget: %v", err)
	}
	// A second Close must reuse the original deadline, not renew it.
	start = time.Now()
	if err := r.eng.Close(); !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Errorf("second Close started another budget: %v", err)
	}
	lock, err := nodelock.Acquire(t.Context(), r.eng.cfg.UsageDBPath()+".lock", 1, 0)
	if err == nil {
		lock.Release()
		t.Error("Close released ledger ownership while a loop was still running")
	}
	close(finished)
	<-r.eng.closeDone // no leaked goroutine or test database
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
}

type f32BudgetPlane struct {
	*memPlane
	deadline time.Time
	reads    int
}

type f32QuarantineBudgetPlane struct {
	*memPlane
	deadlines []time.Time
}

func (p *f32QuarantineBudgetPlane) Quarantine(ctx context.Context, m enforce.Model) (enforce.QuarantineEvidence, error) {
	d, _ := ctx.Deadline()
	p.deadlines = append(p.deadlines, d)
	return p.memPlane.Quarantine(ctx, m)
}

func TestF32_AT13_QuarantineRetriesShareAllowance(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	p := &f32QuarantineBudgetPlane{memPlane: r.plane}
	r.eng.plane = p
	ctx, cancel := newLocalBudget(t.Context(), 10*time.Second)
	defer cancel()
	r.eng.quarantine(ctx)
	r.eng.quarantine(ctx) // a final audit can re-contain while time remains
	if len(p.deadlines) != 2 || !p.deadlines[0].Equal(p.deadlines[1]) {
		t.Fatal("quarantine retry refreshed its deadline")
	}
	ctx.Value(localBudgetKey{}).(*localBudget).cancelQuarantine()
	r.eng.quarantine(ctx)
	if len(p.deadlines) != 2 {
		t.Fatal("spent allowance allowed another quarantine")
	}
}

func (p *f32BudgetPlane) VerifyServing(ctx context.Context, _ enforce.Model, _ map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	p.reads++
	p.deadline, _ = ctx.Deadline()
	<-ctx.Done()
	return enforce.ServingEvidence{}, ctx.Err()
}

func TestF32_AT13_FinalReadCannotRestartLocalBudget(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	p := &f32BudgetPlane{memPlane: r.plane}
	r.eng.plane = p
	ctx, cancel := newLocalBudget(t.Context(), 40*time.Millisecond)
	defer cancel()
	deadline, _ := ctx.Deadline()
	start := time.Now()
	r.eng.finishServingStep(ctx)
	if p.deadline.After(deadline) || time.Since(start) > 500*time.Millisecond || r.eng.serving {
		t.Fatalf("final read or blocked audit extended deadline: reads=%d deadline=%s want<=%s", p.reads, p.deadline, deadline)
	}
}

func TestF32_AT13_StaleHoldMonotonicAlarmAndRecovery(t *testing.T) {
	r := newRig(t)
	keep, _ := servePair(t, r)
	cacheStickyLease(t, r)
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
	stepOK(t, r)
	if r.eng.staleHold.steps != 1 || r.eng.staleHold.alarm {
		t.Fatalf("first hold: %+v", r.eng.staleHold)
	}
	since := r.eng.staleHold.since
	r.clock.JumpWall(-time.Hour)
	r.clock.Advance(staleHoldDuration)
	// Exercise the monotonic alarm without needing three steps or trusted
	// wall time to recover. Observation calls cannot increment a step twice.
	r.eng.updateStaleHold()
	if !r.eng.staleHold.alarm || r.eng.staleHold.steps != 1 || r.eng.staleHold.elapsed != staleHoldDuration {
		t.Fatalf("elapsed alarm depends on wall time or observations: %+v", r.eng.staleHold)
	}
	r.eng.desired.ConfigGeneration++
	r.eng.appliedGen++
	r.eng.staleHold.counted = false
	r.eng.updateStaleHold()
	if r.eng.staleHold.since != since || r.eng.staleHold.steps != 2 {
		t.Fatal("new desired generation reset the persistent hold")
	}
	rep := mustReport(t, r, t.Context())
	requireObservationSchema(t, rep)
	if !strings.Contains(r.log.String(), "serving lease generation hold") {
		t.Fatal("missing local alarm")
	}
	r.eng.lease.ConfigGeneration = r.eng.desired.ConfigGeneration
	if !r.eng.updateStaleHold() || r.eng.staleHold.steps != 0 {
		t.Fatal("matching lease did not end hold")
	}
}

func TestF32_AT13_StaleCodeSurvivesSafetyPrecedence(t *testing.T) {
	for _, bindings := range []bool{false, true} {
		in := servingInput()
		in.Desired.ConfigGeneration++
		if bindings {
			in.Desired.DeviceBindings = []api.DeviceBinding{oneBinding(in.Now)}
		}
		in.ApplyError = &api.NodeError{Code: api.ErrorCodeTCFOREIGNFILTER, Stage: "apply", ObservedAt: in.Now}
		d := Decide(in)
		stale := false
		for _, e := range d.Errors {
			stale = stale || e.Stage == "lease" && e.Code == api.ErrorCodeSTALEGENERATION
		}
		if !stale || d.Serve || !hasReason(d.Reasons, api.ForwardingBlockReasonPolicyMissing) {
			t.Fatalf("safety failure hid mismatch: %+v", d)
		}
	}
}

func TestF32_AT13_HoldStartsAtACKBeforeObservation(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.serving = false
	r.srv.SetFixture(generationFixture(r.eng.appliedGen+1, oneBinding(r.clock.Wall())))
	if err := r.eng.syncDesired(t.Context()); err != nil {
		t.Fatal(err)
	}
	r.eng.refreshDesired()
	r.eng.staleHold.counted = false
	ackedAt := r.clock.Mono()
	r.eng.ackApplied(t.Context(), uint64(r.eng.desired.ConfigGeneration))
	r.clock.Advance(4 * time.Second) // a delayed matching-lease request
	r.eng.updateStaleHold()
	if r.eng.staleHold.since != ackedAt || r.eng.staleHold.elapsed != 4*time.Second || r.eng.staleHold.steps != 1 {
		t.Fatalf("hold omitted time between ACK and observation: %+v", r.eng.staleHold)
	}
}

type f32PreinstallFailure struct{ *memPlane }

func (p *f32PreinstallFailure) VerifyServing(context.Context, enforce.Model, map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	return enforce.ServingEvidence{}, &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter}
}

func (p *f32PreinstallFailure) Apply(context.Context, enforce.Model, map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	// A real pre-install rejection need not reach the serving verifier.
	return enforce.ApplyResult{}, &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter}
}

func TestF32_ISOF601_RejectedCandidateSurvivesObservationBackoff(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.cfg.ObservationSeconds = 1
	r.eng.plane = &f32PreinstallFailure{r.plane}
	// Keep the old rejection pending across the newer generation. HTTP 429
	// uses the same retention path as 502 and also proves Retry-After wins.
	r.srv.SetObservationRetryAfter(1, 60)
	stepOK(t, r)
	old, err := r.eng.store.Pending(t.Context())
	if err != nil || len(old) == 0 {
		t.Fatalf("missing original rejection: %v", err)
	}
	original := append([]byte(nil), old[0].Payload...)
	posts := r.srv.ObservationPosts()
	priorGen := r.eng.appliedGen
	r.srv.SetFixture(generationFixture(priorGen+1, oneBinding(r.clock.Wall())))
	for range 7 {
		r.clock.Advance(time.Second)
		stepOK(t, r)
	}
	if r.srv.ObservationPosts() != posts || r.eng.serving || r.eng.appliedGen != priorGen {
		t.Fatal("retry budget bypassed or failed candidate acknowledged")
	}
	rows, err := r.eng.store.Pending(t.Context())
	if err != nil || len(rows) != len(old)+1 || !bytes.Equal(rows[0].Payload, original) {
		t.Fatalf("candidate transition lost, duplicated, or old body rewritten: before=%d after=%d err=%v", len(old), len(rows), err)
	}
	var candidate api.ObservationReport
	if err := json.Unmarshal(rows[len(rows)-1].Payload, &candidate); err != nil {
		t.Fatal(err)
	}
	requireObservationSchema(t, candidate)
	found := false
	for _, e := range candidate.Errors {
		if e.Stage == "apply_candidate" && e.Code == api.ErrorCodeTCFOREIGNFILTER && e.Message == "candidate_generation=3 verified=false" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing failed candidate without verifier evidence: %+v", candidate.Errors)
	}
	r.clock.Advance(time.Minute)
	if err := r.eng.flush(t.Context()); err != nil {
		t.Fatal(err)
	}
	accepted, ok := r.srv.LastObservation()
	if !ok || accepted.ReportId != candidate.ReportId || !accepted.ObservedAt.Equal(candidate.ObservedAt) {
		t.Fatal("restored control failed to deliver retained candidate identity/time")
	}
	rows, err = r.eng.store.Pending(t.Context())
	if err != nil || len(rows) != 0 {
		t.Fatalf("outbox did not drain: %d %v", len(rows), err)
	}
}
