package agent

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/google/uuid"
)

func f51Restart(t *testing.T, r *rig, plane Plane) *Engine {
	t.Helper()
	cfg, boot, log := r.eng.cfg, r.eng.bootID, r.eng.log
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	e, err := Start(Options{Config: cfg, Clock: r.clock, Plane: plane, Log: log, MeterSource: staticSource{boot: boot.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	r.eng = e
	return e
}

func f51Done(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("child or request did not finish")
	}
}

func TestD155_F51_PerCallCancellationDoesNotKillRecovery(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	marker := filepath.Join(t.TempDir(), "finish")
	calls := f49RecoverCommand(t, r, fmt.Sprintf("while [ ! -f %q ]; do sleep 0.01; done", marker))
	e := f51Restart(t, r, r.plane)
	boot, cancelBoot := context.WithCancel(t.Context())
	if err := e.Boot(boot); err != nil {
		t.Fatal(err)
	}
	cancelBoot()
	job := e.recoverJob
	if job == nil {
		t.Fatal("Boot did not retain a running child")
	}
	step, cancelStep := context.WithCancel(t.Context())
	if err := e.Step(step); err != nil {
		t.Fatal(err)
	}
	cancelStep()
	select {
	case <-job.done:
		t.Fatal("per-call cancellation killed child")
	default:
	}
	if e.serving || !e.recoverBlocked || e.recovered || f49Calls(t, calls) != 1 {
		t.Fatal("pending recovery served or duplicated child")
	}
	if err := os.WriteFile(marker, []byte("done"), 0600); err != nil {
		t.Fatal(err)
	}
	f51Done(t, job.done)
	stepOK(t, r)
	if !e.recovered || e.recoverBlocked || !e.serving || job.interrupted || f49Calls(t, calls) != 1 {
		t.Fatal("live Step did not finish original recovery")
	}
}

type f51LifecyclePlane struct {
	*memPlane
	baseline func()
	applied  func()
}

func (p *f51LifecyclePlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	err := p.memPlane.Baseline(ctx, dp)
	if p.baseline != nil {
		p.baseline()
	}
	return err
}
func (p *f51LifecyclePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.memPlane.Apply(ctx, m, prev)
	if p.applied != nil {
		p.applied()
	}
	return res, err
}

func TestD155_F51_RunDefersRetriesAndStops(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	marker := filepath.Join(t.TempDir(), "retry")
	calls := f49RecoverCommand(t, r, fmt.Sprintf("if [ ! -f %q ]; then touch %q; exit 3; fi\nexit 0", marker, marker))
	p := &f51LifecyclePlane{memPlane: r.plane}
	e := f51Restart(t, r, p)
	e.cfg.PollSeconds, e.cfg.HeartbeatSeconds = 1, 1
	lock, err := nodelock.Acquire(t.Context(), e.cfg.LockPath, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	baselines := 0
	p.baseline = func() {
		baselines++
		if baselines == 1 {
			if f49Calls(t, calls) != 0 || !e.recoverBlocked || e.recovered {
				t.Error("Boot did not defer under live lock")
			}
			if err := lock.Release(); err != nil {
				t.Error(err)
			}
		} else if baselines == 2 {
			if f49Calls(t, calls) != 1 || !e.recoverBlocked || e.recovered {
				t.Error("exit 3 was not deferred")
			}
			r.clock.Advance(recoverRetryInterval)
		}
	}
	p.applied = cancel
	if err := e.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run stop: %v", err)
	}
	if f49Calls(t, calls) != 2 || !e.recovered || e.recoverBlocked || !e.recoverClosed {
		t.Fatal("Run did not retry then stop cleanly")
	}
}

func TestD155_F51_RunStopAndOnceLifetime(t *testing.T) {
	for _, mode := range []string{"run", "once"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t)
			serveBoundGeneration(t, r)
			calls := f49RecoverCommand(t, r, "exec sleep 60")
			p := &f51LifecyclePlane{memPlane: r.plane}
			e := f51Restart(t, r, p)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if mode == "run" {
				p.baseline = cancel
				if err := e.Run(ctx); !errors.Is(err, context.Canceled) {
					t.Fatalf("Run stop: %v", err)
				}
			} else {
				if err := e.Boot(ctx); err != nil {
					t.Fatal(err)
				}
				if err := e.Step(ctx); err != nil {
					t.Fatal(err)
				}
				cancel()
				select {
				case <-e.recoverJob.done:
					t.Fatal("once caller killed recovery before Close")
				default:
				}
			}
			job := e.recoverJob
			if job == nil || f49Calls(t, calls) != 1 || !e.recoverBlocked || e.recovered || e.serving {
				t.Fatal("stop changed pending recovery into terminal evidence or served")
			}
			if err := e.Close(); err != nil {
				t.Fatal(err)
			}
			f51Done(t, job.done)
			if !job.interrupted || e.recovered {
				t.Fatal("stop counted as recovery failure evidence")
			}
		})
	}
}

func TestD155_F51_CanceledRunDoesNotSpawn(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	calls := f49RecoverCommand(t, r, "exit 1")
	e := f51Restart(t, r, r.plane)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := e.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run stop: %v", err)
	}
	if f49Calls(t, calls) != 0 || e.recovered || !e.recoverBlocked || !e.input().RecoverBlocked || e.serving {
		t.Fatal("canceled Run spawned, latched failure or opened recovery gate")
	}
}

func TestD155_F51_UploadFollowsCallerInRunAndOnce(t *testing.T) {
	for _, mode := range []string{"run", "once"} {
		t.Run(mode, func(t *testing.T) {
			r, src, _ := r20ServedPendingDelta(t)
			r.srv.SetUsageUnavailable(0)
			e := r20Successor(t, r, r.eng.cfg, src, r.plane)
			request, release := r.srv.HoldNextUsageUntilCanceled()
			defer release()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			if mode == "run" {
				go func() { done <- e.Run(ctx) }()
			} else {
				if err := e.Boot(ctx); err != nil {
					t.Fatal(err)
				}
				if err := e.Step(ctx); err != nil {
					t.Fatal(err)
				}
			}
			var reqCtx context.Context
			select {
			case reqCtx = <-request:
			case <-time.After(5 * time.Second):
				t.Fatal("upload never started")
			}
			if reqCtx.Err() != nil {
				t.Fatal("Boot/Step return canceled upload")
			}
			cancel()
			// Observe the real HTTP request ending before Close, independently of local
			// Boot/Step allowances. No test edits the engine's upload parent or loop.
			f51Done(t, reqCtx.Done())
			if mode == "run" {
				select {
				case err := <-done:
					if !errors.Is(err, context.Canceled) {
						t.Fatal(err)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("Run did not stop")
				}
			}
			if err := e.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type f51RecoveryPlane struct {
	*r27BootMissingInterfacePlane
	afterApply func()
}

func (p *f51RecoveryPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.r27BootMissingInterfacePlane.Apply(ctx, m, prev)
	if err == nil && p.afterApply != nil {
		p.afterApply()
	}
	return res, err
}

func TestD157_F51_PreAttemptCatchupAndOneEarlyRenewalPerRecovery(t *testing.T) {
	r, p := f50Rig(t)
	f50Recovery(t, r, p, 14)
	stepOK(t, r) // consume recovery 14's early opportunity with an empty answer
	for _, gen := range []uint64{15, 16} {
		p.foreignTC = true
		stepOK(t, r)
		p.foreignTC = false
		r.clock.Advance(r.eng.provisionalRetryAt - r.clock.Mono())
		r.srv.SetFixture(generationFixture(gen, oneBinding(r.clock.Wall())))
		if gen == 15 {
			r.srv.FailLeases(1)
			// The injected 503 has Retry-After: 1; let it elapse inside Apply.
			r.eng.plane = &f51RecoveryPlane{r27BootMissingInterfacePlane: p, afterApply: func() { r.clock.Advance(time.Second) }}
		} else {
			r.srv.SetLeaseGeneration(gen - 1)
			r.eng.plane = &f51RecoveryPlane{r27BootMissingInterfacePlane: p, afterApply: func() { r.srv.SetLeaseGeneration(0) }}
		}
		before := r.srv.LeaseRenewals()
		previous := r.srv.LeaseRequests()
		stepOK(t, r)
		if r.srv.LeaseRenewals() != before+2 || r.eng.appliedGen != gen || !r.eng.earlyRecoveryRenewal() || r.eng.serving {
			t.Fatalf("pre-enforce catchup gen=%d calls=%d want=%d applied=%d lease=%d closed=%v renewal=%+v serving=%v retry=%s now=%s log=%s", gen, r.srv.LeaseRenewals(), before+2, r.eng.appliedGen, r.eng.lease.ConfigGeneration, r.eng.closedRecoveryHeld, r.eng.closedRecoveryRenewal, r.eng.serving, r.eng.leaseRetryAt, r.clock.Mono(), r.log.String())
		}
		reqs := r.srv.LeaseRequests()
		last := reqs[len(reqs)-1]
		ack, err := last.AppliedConfigGeneration.Get()
		if err != nil || uint64(ack) != gen || previous[len(previous)-1].RenewalRequestId == last.RenewalRequestId {
			t.Fatal("catchup did not send newly verified ACK with fresh ID")
		}
		// The 503 is injected before decoding; only the older-generation answer
		// records the pre-Apply request body. Counts above include both paths.
		if gen == 16 {
			first := reqs[len(reqs)-2]
			if len(reqs) != len(previous)+2 || !first.AppliedConfigGeneration.IsNull() || first.RenewalRequestId == last.RenewalRequestId {
				t.Fatal("pre-Apply request ACKed a failed generation or reused its ID")
			}
		} else if len(reqs) != len(previous)+1 {
			t.Fatal("catchup request missing after undecoded failure")
		}
		sent := r.clock.Mono()
		stepOK(t, r)
		if r.srv.LeaseRenewals() != before+3 || r.eng.earlyRecoveryRenewal() || r.eng.serving {
			t.Fatal("missing single early request")
		}
		slot := r.eng.provisionalRetryAt
		if slot != sent+r.eng.provisionalWait(r.eng.provisionalStreak) {
			t.Fatal("empty early answer changed D-067 schedule")
		}
		for range 3 {
			stepOK(t, r)
		}
		r.clock.Advance(slot - r.clock.Mono() - time.Nanosecond)
		stepOK(t, r)
		if r.srv.LeaseRenewals() != before+3 {
			t.Fatal("same recovery renewed twice before slot")
		}
		r.clock.Advance(time.Nanosecond)
		stepOK(t, r)
		if r.srv.LeaseRenewals() != before+4 {
			t.Fatal("ordinary D-067 slot missing")
		}
	}
}

func TestD157_F51_PendingDedupStillCoalescesWithoutUnboundRecovery(t *testing.T) {
	r, p := f50Rig(t)
	r.srv.SetObservationRetryAfter(100, 60)
	f50Recovery(t, r, p, 14)
	first := r.eng.closedRecoveryRenewal.reportID
	if first == uuid.Nil {
		t.Fatal("missing report binding")
	}
	for _, armed := range []bool{true, false} {
		if !armed {
			r.eng.closedRecoveryRenewal = recoveryRenewal{}
		}
		pending, err := r.eng.store.Pending(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for range 3 {
			r.eng.lastObs = time.Time{} // exercise due-but-identical pending dedup
			if err := r.eng.observe(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
		after, err := r.eng.store.Pending(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(after) != len(pending) {
			t.Fatal("ordinary pending state duplicated")
		}
	}
}

func TestD157_F51_FullOutboxStepsBindSecondRecovery(t *testing.T) {
	r, p := f50Rig(t)
	f50Recovery(t, r, p, 14)
	stepOK(t, r)
	// Real observations only: every two Steps adds failure then recovery. No
	// baseline/noteApply calls or assignments to dedup/recovery/outbox fields.
	r.srv.SetObservationRetryAfter(100, 60)
	for range 16 {
		p.foreignTC = true
		stepOK(t, r)
		p.foreignTC = false
		stepOK(t, r)
	}
	pending, err := r.eng.store.Pending(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	first, state := r.eng.closedRecoveryRenewal.reportID, r.eng.lastObservationState
	if len(pending) != 32 || first == uuid.Nil {
		t.Fatalf("did not fill outbox at clean recovery: pending=%d report=%v", len(pending), first)
	}
	p.foreignTC = true
	stepOK(t, r)
	if !r.eng.outboxFull || r.eng.applyError == nil || r.eng.lastObservationState != state {
		t.Fatal("full outbox recorded intermediate failure or failed to withdraw")
	}
	p.foreignTC = false
	// One old report is accepted; the next request receives Retry-After, so
	// the old clean report remains pending while space becomes available.
	r.srv.AllowObservationsBeforeRetry(1)
	r.clock.Advance(r.eng.observationRetryAt - r.clock.Mono())
	stepOK(t, r)
	if r.eng.serving || !r.eng.closedRecoveryHeld || r.eng.applyError != nil {
		t.Fatal("partial drain did not permit closed recovery")
	}
	if r.eng.lastObservationState != state {
		t.Fatal("sequence did not reproduce identical clean state")
	}
	second := r.eng.closedRecoveryRenewal.reportID
	if second == uuid.Nil || second == first {
		t.Fatal("new recovery was deduplicated against old pending report")
	}
	pending, err = r.eng.store.Pending(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 32 {
		t.Fatalf("new report not queued within cap: %d", len(pending))
	}
}
