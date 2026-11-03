package agent

// TEST ONLY disk SQLite and injected data planes; no host networking.
import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func f40Live(t *testing.T) (*rig, *r20DynamicSource, string) {
	t.Helper()
	r := barrierRig(t)
	b := r.eng.appliedModel.Bindings[0]
	up := meter.CounterName(b.DeviceID, meter.DirUpload)
	src := &r20DynamicSource{boot: r.eng.bootID.String(), bindings: r.eng.meterBindings, counters: map[string]enforce.CounterValue{up: {Packets: 1, Bytes: 4096}}}
	r.eng.meter.Close()
	var err error
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	src.setCounter(up, enforce.CounterValue{Packets: 2, Bytes: 8192})
	return r, src, up
}

type f40MarkerError struct{ *memPlane }

func (p f40MarkerError) RestartGeneration(context.Context) (uint64, error) {
	return 0, errors.New("TEST ONLY transient marker read failure")
}

func TestR23_H1c_MarkerReadFailureKeepsIntentAcrossRestart(t *testing.T) {
	r, src, _ := f40Live(t)
	// No legacy bound-counter evidence: the new device has counted only since
	// the real pre-Apply baseline. The intent must supply the sole candidate.
	b := oneBinding(r.clock.Wall())
	b.DeviceId, b.MemberId = uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	b.BindingRevision, b.AllowedIpv4 = 2, []string{"100.64.0.12/32"}
	r.srv.SetFixture(generationFixture(3, b))
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	src.setCounter(meter.CounterName(b.DeviceId, meter.DirUpload), enforce.CounterValue{Packets: 1, Bytes: 2048})
	r20Crash(r.eng)
	next := r20Successor(t, r, r20ChangedSets(r), src, f40MarkerError{r.plane})
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || u.byRevision[2] != 2048 || len(u.gaps) != 0 {
		t.Fatalf("intent lost: %+v", u)
	}
	r.eng = next
	r20Crash(next)
	next = r20Successor(t, r, r.eng.cfg, src, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u = r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || u.byRevision[2] != 2048 || len(u.gaps) != 0 {
		t.Fatalf("restart recounted: %+v", u)
	}
}

func TestR23_H1_IntentCannotCrossSettledBoundaryOrFailedIntentWrite(t *testing.T) {
	r, src, _ := f40Live(t)
	old := r.eng.restartIntent
	r.eng.handoff(t.Context())
	if !r.eng.finalDone {
		t.Fatal("handoff failed")
	}
	if err := r.eng.meter.PrepareSampling(t.Context()); err != nil {
		t.Fatal(err)
	}
	h, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ok, err := r.eng.usage.SameCountingInterval(t.Context(), old.Epoch, h.OpenEpoch, src.boot)
	if err != nil || ok {
		t.Fatalf("settled intent crosses baseline: %v %v", ok, err)
	}
	// Actual failed metadata write before any later Apply. The old intent is
	// durable, but cannot label this unbound epoch, even on the same OS boot.
	db := f40AgentDB(t, r.eng.cfg.DBPath())
	if _, err := db.Exec(`CREATE TRIGGER f40_intent_fail BEFORE UPDATE ON meta WHEN NEW.key='usage_apply_intent' BEGIN SELECT RAISE(FAIL,'TEST ONLY intent failure'); END`); err != nil {
		t.Fatal(err)
	}
	model := *r.eng.appliedModel
	model.Generation++
	model.Bindings = append([]enforce.Binding(nil), model.Bindings...)
	model.Bindings[0].MemberID = uuid.Must(uuid.NewV7())
	if err := r.eng.recordUsageIntent(t.Context(), model); err == nil {
		t.Fatal("intent write did not fail")
	}
	r20Crash(r.eng)
	next := r20Successor(t, r, r20ChangedSets(r), src, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || len(u.gaps) != 0 {
		t.Fatalf("settled bytes changed: %+v", u)
	}
}

// A production reset/nudge retains the renderer's idle zero counters. This
// variant exercises the same switch/restart as the original missing-counter
// guard, without claiming that absent counters are a healthy verified state.
func TestR23_H1b_IdleZeroCounterSurvivesSwitchAndRestart(t *testing.T) {
	r23H1b(t, true)
}

func f40AgentDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// Reopen the test's owned usage handle to install its existing fault hook.
// Loops have been joined by manualUsage before this helper is called.
func f40UsageHook(t *testing.T, r *rig, src meter.Source, hook usagestore.Hook) {
	t.Helper()
	r.eng.meter.Close()
	if err := r.eng.usage.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := usagestore.Open(r.eng.cfg.UsageDBPath(), r.eng.nodeID, r.eng.incID, usagestore.Options{Hook: hook})
	if err != nil {
		t.Fatal(err)
	}
	r.eng.usage, r.eng.usageStatus = st, st
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: st, Source: src})
	if err != nil {
		t.Fatal(err)
	}
}

func TestR23_H2_InterruptedCheckpointPreparesNextEpoch(t *testing.T) {
	for _, fault := range []string{"removed", "replaced", "checkpoint_read", "checkpoint_write"} {
		for _, restart := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_restart_%v", fault, restart), func(t *testing.T) {
				r, src, up := f40Live(t)
				writeBarrier(t, r, uuid.NewString())
				handCtx, cancel := context.WithCancel(t.Context())
				defer cancel()
				switch fault {
				case "removed", "replaced":
					r.eng.plane = barrierPlane{Plane: r.plane, baseline: func() error {
						if fault == "removed" {
							return os.Remove(filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked"))
						}
						writeBarrier(t, r, uuid.NewString())
						return nil
					}}
				case "checkpoint_read":
					f40UsageHook(t, r, src, func(step string) error {
						if step == usagestore.StepAfterCommit {
							cancel()
						}
						return nil
					})
				case "checkpoint_write":
					if err := os.MkdirAll(r.eng.cfg.MeterCheckpoint, 0700); err != nil {
						t.Fatal(err)
					}
				}
				r.eng.handoff(handCtx)
				r.eng.plane = r.plane
				h, err := r.eng.usage.Head(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				if h.OpenEpoch != uuid.Nil || !r.eng.epochClosed || r.eng.finalDone {
					t.Fatalf("closed epoch/barrier conflated: %+v closed=%v done=%v", h, r.eng.epochClosed, r.eng.finalDone)
				}
				if restart {
					r20Crash(r.eng)
					next := r20Successor(t, r, r.eng.cfg, src, r.plane)
					r.eng = next
					if err := next.Boot(t.Context()); err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Remove(filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked")); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				if fault == "checkpoint_write" {
					if err := os.Remove(r.eng.cfg.MeterCheckpoint); err != nil {
						t.Fatal(err)
					}
				}
				r.clock.Advance(time.Second)
				alignControlClock(r)
				stepOK(t, r)
				if !r.eng.serving || r.eng.applyError != nil || r.eng.usagePersist {
					t.Fatalf("resume: serving=%v reasons=%v error=%v", r.eng.serving, r.eng.decision.Reasons, r.eng.applyError)
				}
				if restart {
					r.manualUsage(t)
				}
				src.setCounter(up, enforce.CounterValue{Packets: 3, Bytes: 9216})
				// A subsequent restart also must not baseline away the first new bytes.
				r20Crash(r.eng)
				cfg := r20ChangedSets(r)
				cfg.ExtraDeniedV4 = append(cfg.ExtraDeniedV4, "198.51.100.0/24")
				next := r20Successor(t, r, cfg, src, r.plane)
				if err := next.Boot(t.Context()); err != nil {
					t.Fatal(err)
				}
				u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
				if u.byRevision[1] != 9216 || len(u.gaps) != 0 {
					t.Fatalf("lost/recounted bytes: %+v", u)
				}
			})
		}
	}
}

type f40CheckpointRace struct {
	*memPlane
	once func()
}

func (p *f40CheckpointRace) MemberPath(ctx context.Context, m enforce.Model, previous *MemberPath) (MemberPath, error) {
	if p.once != nil {
		f := p.once
		p.once = nil
		f()
	}
	return p.memPlane.MemberPath(ctx, m, previous)
}

func f40AssertBlockedReport(t *testing.T, r *rig) {
	t.Helper()
	hb := mustBeat(t, r)
	rep, ok := r.srv.LastObservation()
	if !ok || r.eng.serving || hb.Serving || rep.MemberForwardingEnabled || hb.Reason == "-" || len(rep.ForwardingBlockReasons) == 0 || hb.Reason != string(rep.ForwardingBlockReasons[0]) || rep.SafetyStatus != api.SafetyStatusFailed {
		t.Fatalf("inconsistent report: hb=%+v rep=%+v", hb, rep)
	}
}

func TestR23_H3_M1_FailedHandoffReportsRecoversAndRestarts(t *testing.T) {
	for _, fault := range []string{"H3a_hold", "H3a_read", "H3a_persist", "H3b_baseline", "H3b_prime_checkpoint", "M1_first_baseline"} {
		for _, restart := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_restart_%v", fault, restart), func(t *testing.T) {
				r, src, up := f40Live(t)
				b := oneBinding(r.clock.Wall())
				b.MemberId = uuid.Must(uuid.NewV7())
				b.BindingRevision = 2
				r.srv.SetFixture(generationFixture(3, b))
				r.eng.leaseRefreshPending = true
				var fail atomic.Bool
				fail.Store(true)
				source := r21ReadFunc(func(ctx context.Context) (meter.Observation, error) {
					if fault == "H3a_read" && fail.Load() {
						return meter.Observation{}, errors.New("TEST ONLY sample failure")
					}
					return src.Read(ctx)
				})
				h, err := r.eng.usage.Head(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				r.eng.meter.Close()
				// Initialization is continuous; inject a read error only after construction.
				fail.Store(false)
				r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: source, Now: func() time.Time {
					if fault == "H3a_hold" && fail.Load() {
						return h.LastSample.Add(-time.Microsecond)
					}
					return time.Now()
				}})
				if err != nil {
					t.Fatal(err)
				}
				if fault == "H3a_persist" {
					f40UsageHook(t, r, src, func(step string) error {
						if fail.Load() && step == usagestore.StepBeforeCommit {
							return errors.New("TEST ONLY usage persistence failure")
						}
						return nil
					})
				}
				fail.Store(true)
				calls := 0
				if fault == "H3b_baseline" || fault == "M1_first_baseline" {
					r.eng.plane = barrierPlane{Plane: r.plane, baseline: func() error {
						calls++
						if fail.Load() && (fault == "H3b_baseline" || calls == 1) {
							return errors.New("TEST ONLY baseline failure")
						}
						return nil
					}}
				}
				if fault == "H3b_prime_checkpoint" {
					r.eng.plane = &f40CheckpointRace{memPlane: r.plane, once: func() {
						if err := os.MkdirAll(r.eng.cfg.MeterCheckpoint, 0700); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(filepath.Join(r.eng.cfg.MeterCheckpoint, "held"), []byte("TEST ONLY"), 0600); err != nil {
							t.Fatal(err)
						}
					}}
				}
				if fault == "M1_first_baseline" {
					r.eng.withdraw(t.Context())
					r.eng.decision = Decide(r.eng.input())
					r.eng.beat()
					_ = r.eng.observe(t.Context())
					_ = r.eng.flush(t.Context())
				} else {
					stepOK(t, r)
				}
				if !r.eng.retainedPending {
					t.Fatal("failed handoff lost pending interval")
				}
				f40AssertBlockedReport(t, r)
				if fault != "H3b_baseline" && r.plane.last.Generation != 0 {
					t.Fatal("usage failure left live forwarding")
				}
				before := r.plane.applies
				if fault != "M1_first_baseline" && fault != "H3b_prime_checkpoint" {
					stepOK(t, r)
					if r.plane.applies != before {
						t.Fatal("failed handoff bypassed")
					}
					f40AssertBlockedReport(t, r)
				}
				// Restart while withdrawal is pending, then repair the failure. Boot uses
				// the same durable candidate, never the fetched but unapplied rebinding.
				if restart {
					r20Crash(r.eng)
					next := r20Successor(t, r, r20ChangedSets(r), src, r.plane)
					r.eng = next
					if err := next.Boot(t.Context()); err != nil {
						t.Fatal(err)
					}
				}
				fail.Store(false)
				r.eng.plane = r.plane
				if fault == "H3b_prime_checkpoint" {
					if err := os.RemoveAll(r.eng.cfg.MeterCheckpoint); err != nil {
						t.Fatal(err)
					}
				}
				r.clock.Advance(time.Second)
				alignControlClock(r)
				stepOK(t, r)
				if !r.eng.serving || r.eng.retainedPending {
					t.Fatalf("no recovery: %v", r.eng.decision.Reasons)
				}
				if restart {
					r.manualUsage(t)
				}
				src.setCounter(up, enforce.CounterValue{Packets: 3, Bytes: 9216})
				r20Crash(r.eng)
				cfg := r20ChangedSets(r)
				cfg.ExtraDeniedV4 = append(cfg.ExtraDeniedV4, "198.51.100.0/24")
				next := r20Successor(t, r, cfg, src, r.plane)
				if err := next.Boot(t.Context()); err != nil {
					t.Fatal(err)
				}
				u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
				if u.byRevision[1] != 8192 || u.byRevision[2] != 1024 || len(u.gaps) != 0 {
					t.Fatalf("bad attribution after recovery/restart: %+v", u)
				}
			})
		}
	}
}

type f40RenderPlane struct {
	*memPlane
	src *r20DynamicSource
	t   *testing.T
}

func (p f40RenderPlane) check(plan enforce.RenderPlan, err error) {
	p.t.Helper()
	if err != nil {
		p.t.Fatal(err)
	}
	if strings.Contains(plan.NFT, "delete counter") || strings.Contains(plan.NFT, "delete table") {
		p.t.Fatal("renderer destroyed counter history")
	}
	obs, _ := p.src.Read(context.Background())
	for name := range obs.Counters {
		if strings.Contains(plan.NFT, "add counter inet fvpn "+name+"\n") {
			p.t.Fatal("renderer recreated existing counter", name)
		}
	}
}
func (p f40RenderPlane) state() enforce.State {
	obs, _ := p.src.Read(context.Background())
	return enforce.State{Table: true, Chains: map[string]bool{"forward": true}, Counters: obs.Counters, Generation: p.last.Generation}
}
func (p f40RenderPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	p.check(enforce.Render(m, p.state(), enforce.TCState{}))
	return p.memPlane.Apply(ctx, m, prev)
}
func (p f40RenderPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	raw, err := enforce.RenderBaseline(enforce.Model{Datapath: dp}, p.state())
	p.check(enforce.RenderPlan{NFT: raw}, err)
	return p.memPlane.Baseline(ctx, dp)
}

func TestR23_Held_RemovedReaddedUsesRealRendererPreservedCounters(t *testing.T) {
	r, src, up := f40Live(t)
	p := f40RenderPlane{memPlane: r.plane, src: src, t: t}
	r.eng.plane = p
	b := oneBinding(r.clock.Wall())
	f := generationFixture(3, b)
	f.Bindings = nil
	r.srv.SetFixture(f)
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	if len(r.plane.last.Bindings) != 0 {
		t.Fatal("device not removed")
	}
	// The removed device's named counter still contains all old bytes. Both
	// Baseline and serving renders above used that actual preserved State.
	obs, _ := src.Read(t.Context())
	if obs.Counters[up].Bytes != 8192 {
		t.Fatal("counter did not survive removal")
	}
	r20Crash(r.eng)
	next := r20Successor(t, r, r.eng.cfg, src, p)
	r.eng = next
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	b.MemberId = uuid.Must(uuid.NewV7())
	b.BindingRevision = 2
	r.srv.SetFixture(generationFixture(4, b))
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	if !r.eng.serving {
		t.Fatal(r.eng.decision.Reasons)
	}
	r.manualUsage(t)
	src.setCounter(up, enforce.CounterValue{Packets: 3, Bytes: 9216})
	r20Crash(r.eng)
	next = r20Successor(t, r, r20ChangedSets(r), src, p)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || u.byRevision[2] != 1024 || len(u.gaps) != 0 {
		t.Fatalf("counter reuse: %+v", u)
	}
}

type f40RepairPlane struct {
	*memPlane
	fail   bool
	kernel *kernelSource
	src    *r20DynamicSource
}

func (p *f40RepairPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if p.fail {
		return enforce.ApplyResult{}, errors.New("TEST ONLY transient apply failure")
	}
	res, err := p.memPlane.Apply(ctx, m, prev)
	obs, _ := p.src.Read(ctx)
	res.Counters = obs.Counters
	// Inject the pending ruleset-switch notification at the last possible
	// repair window, after admission intent but before enforce's switchEpoch.
	p.kernel.rearm()
	return res, err
}
func TestR23_H1b_RepairApplySwitchEpochKeepsIntent(t *testing.T) {
	r, src, up := f40Live(t)
	k := &kernelSource{boot: src.boot, bindings: r.eng.meterBindings, expect: func() bool { return r.eng.expectServing.Load() }, read: func(ctx context.Context) (enforce.State, error) {
		obs, err := src.Read(ctx)
		return enforce.State{Table: true, Chains: map[string]bool{"forward": true}, Generation: r.plane.last.Generation, Counters: obs.Counters}, err
	}}
	r.eng.kernel = k
	r.eng.meter.Close()
	var err error
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: k})
	if err != nil {
		t.Fatal(err)
	}
	p := &f40RepairPlane{memPlane: r.plane, fail: true, kernel: k, src: src}
	r.eng.plane = p
	stepOK(t, r)
	if r.eng.serving || r.eng.applyError == nil {
		t.Fatal("Apply did not fail closed")
	}
	p.fail = false
	r.clock.Advance(time.Second)
	alignControlClock(r)
	stepOK(t, r)
	if !r.eng.serving {
		t.Fatal(r.eng.decision.Reasons)
	}
	h, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if r.eng.restartIntent.Epoch == h.OpenEpoch || k.nudgePending() {
		t.Fatal("enforce did not switch epoch after intent")
	}
	src.setCounter(up, enforce.CounterValue{Packets: 3, Bytes: 9216})
	r20Crash(r.eng)
	next := r20Successor(t, r, r20ChangedSets(r), src, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 9216 || len(u.gaps) != 1 || u.gaps[0].Gap.Reason != api.UsageGapReasonCounterDiscontinuity {
		t.Fatalf("repair intent lost: %+v", u)
	}
}
