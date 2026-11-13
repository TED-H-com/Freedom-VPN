package agent

// Review r20 proofs (r20-adoption-signal-review, commit 639eda6b), judged and
// converted by agent-core follow-up 37 into passing regressions. Every name
// carries the r20 finding (H-1, M-1, M-2, M-3, L-1) or question (Q2, Q3, Q6)
// it proves. TEST ONLY fixtures; no test reaches a host network or kernel.

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type r20DynamicSource struct {
	mu       sync.Mutex
	boot     string
	bindings func() []enforce.Binding
	counters map[string]enforce.CounterValue
}

func (s *r20DynamicSource) setBindings(fn func() []enforce.Binding) {
	s.mu.Lock()
	s.bindings = fn
	s.mu.Unlock()
}

func (s *r20DynamicSource) setCounter(name string, val enforce.CounterValue) {
	s.mu.Lock()
	s.counters[name] = val
	s.mu.Unlock()
}

func (s *r20DynamicSource) Read(context.Context) (meter.Observation, error) {
	s.mu.Lock()
	out := make(map[string]enforce.CounterValue, len(s.counters))
	for k, v := range s.counters {
		out[k] = v
	}
	fn := s.bindings
	boot := s.boot
	s.mu.Unlock()
	var b []enforce.Binding
	if fn != nil {
		b = fn()
	}
	return meter.Observation{
		BootID:   boot,
		Bindings: b,
		Counters: out,
	}, nil
}

// r20Crash abandons an engine the way a killed process does: loops stop, the
// SQLite handles and the ledger lock are released, and nothing is sampled,
// handed off or written to the heartbeat.
func r20Crash(e *Engine) {
	if e.loopCancel != nil {
		e.loopCancel()
		e.loops.Wait()
	}
	if e.meter != nil {
		e.meter.Close()
		e.meter = nil
	}
	if e.usage != nil {
		_ = e.usage.Close()
		e.usage = nil
	}
	if e.usageLock != nil {
		e.usageLock.Release()
		e.usageLock = nil
	}
	e.usageStatus = nil
	e.shutUsage = true
	if e.store != nil {
		_ = e.store.Close()
		e.store = nil
	}
}

type r20Usage struct {
	byRevision map[uint64]uint64 // attributed upload bytes per binding revision
	gaps       []api.UsageEvent
}

func (u r20Usage) total() uint64 {
	var n uint64
	for _, b := range u.byRevision {
		n += b
	}
	return n
}

// r20ReadUsage parses every immutable outbox payload in the ledger.
func r20ReadUsage(t *testing.T, cfg Config, node, inc uuid.UUID) r20Usage {
	t.Helper()
	st, err := usagestore.Open(cfg.UsageDBPath(), node, inc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	events, err := st.Unacked(t.Context(), 0, 1000, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	u := r20Usage{byRevision: map[uint64]uint64{}}
	for _, ev := range events {
		var p api.UsageEvent
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			t.Fatal(err)
		}
		if p.Quality == api.UsageQualityGap {
			u.gaps = append(u.gaps, p)
		}
		for _, rec := range p.Records {
			u.byRevision[uint64(rec.BindingRevision)] += uint64(rec.UploadBytes)
		}
	}
	return u
}

// r20ServedPendingDelta builds process 1: it serves generation 2 and samples
// the member's upload counter at 4096 bytes, then stops normally, retaining
// the serving rules and the open same-boot epoch. The returned source is the
// kernel the successor reads: the retained rules counted 4096 more bytes in
// the process gap.
// beforeSample lets a bubble caller stop network I/O and advance the sample
// clock; ordinary callers keep their existing wall-clock setup.
func r20ServedPendingDelta(t *testing.T, beforeSample ...func(*rig)) (*rig, *r20DynamicSource, string) {
	t.Helper()
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000) // keep every event in the outbox
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	for _, prepare := range beforeSample {
		prepare(r)
	}
	binding := r.eng.appliedModel.Bindings[0]
	upName := meter.CounterName(binding.DeviceID, meter.DirUpload)
	r.eng.meter.Close()
	src1 := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		bindings: r.eng.meterBindings, // production kernelSource reads e.meterBindings
		counters: map[string]enforce.CounterValue{upName: {Packets: 1, Bytes: 4096}},
	}
	var err error
	r.eng.meter, err = meter.New(meter.Config{
		NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	if b := mustBeat(t, r); !b.Serving {
		t.Fatal("precondition: process 1 did not retain serving rules for restart")
	}
	return r, &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		counters: map[string]enforce.CounterValue{upName: {Packets: 2, Bytes: 8192}},
	}, upName
}

func r20Successor(t *testing.T, r *rig, cfg Config, src *r20DynamicSource, plane Plane) *Engine {
	t.Helper()
	next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: plane, MeterSource: src, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	src.setBindings(next.meterBindings)
	return next
}

func r20ChangedSets(r *rig) Config {
	cfg := r.eng.cfg
	cfg.ExtraDeniedV4 = []string{"203.0.113.1/32", "192.0.2.0/24"}
	return cfg
}

// H-1: a refused same-boot adoption must not turn the delta the retained
// rules counted into an unattributed gap. Attributed exactly once.
func TestR20_H1_RefusedAdoptionAttributesPendingDeltaOnce(t *testing.T) {
	t.Run("unpublished_first_step", func(t *testing.T) {
		// r20's original scenario: the first Step does not serve and hands off.
		r, src, _ := r20ServedPendingDelta(t)
		cfg := r20ChangedSets(r)
		next := r20Successor(t, r, cfg, src, r.plane)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if next.serving {
			t.Fatal("expected Boot to refuse same-boot adoption on changed ExtraDeniedV4")
		}
		r.srv.SetPublication(api.PublicationIntentUnpublished)
		if err := next.Step(t.Context()); err != nil {
			t.Fatal(err)
		}
		u := r20ReadUsage(t, cfg, next.nodeID, next.incID)
		if len(u.gaps) != 0 || u.total() != 8192 || u.byRevision[1] != 8192 {
			t.Fatalf("H-1: refused adoption attributed %v with %d gap event(s) %+v, want 8192 bytes at revision 1 and no gap",
				u.byRevision, len(u.gaps), u.gaps)
		}
	})

	t.Run("rebinding_first_apply", func(t *testing.T) {
		// The first Apply rotates the device's binding revision. The retained
		// delta belongs to revision 1; nothing is attributed to revision 2.
		r, src, _ := r20ServedPendingDelta(t)
		cfg := r20ChangedSets(r)
		next := r20Successor(t, r, cfg, src, r.plane)
		r.eng = next
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if next.serving {
			t.Fatal("expected Boot to refuse same-boot adoption on changed ExtraDeniedV4")
		}
		b := oneBinding(time.Now().UTC())
		b.BindingRevision = 2
		r.srv.SetFixture(generationFixture(3, b))
		stepOK(t, r)
		if !next.serving || next.appliedGen != 3 || next.appliedModel.Bindings[0].BindingRevision != 2 {
			t.Fatalf("setup: first Apply did not serve revision 2: serving=%v gen=%d reasons=%v", next.serving, next.appliedGen, next.decision.Reasons)
		}
		u := r20ReadUsage(t, cfg, next.nodeID, next.incID)
		if len(u.gaps) != 0 || u.byRevision[1] != 8192 || u.byRevision[2] != 0 {
			t.Fatalf("H-1: rebinding Apply attributed %v with %d gap event(s), want 8192 bytes at revision 1 and no gap",
				u.byRevision, len(u.gaps))
		}
	})

	t.Run("two_crashes_same_boot", func(t *testing.T) {
		// Process 2 refuses, settles and is killed; process 3 refuses again
		// on the same kernel counters. No double count, no gap.
		r, src, _ := r20ServedPendingDelta(t)
		cfg := r20ChangedSets(r)
		second := r20Successor(t, r, cfg, src, r.plane)
		if err := second.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if second.serving {
			t.Fatal("expected process 2 to refuse adoption")
		}
		r20Crash(second)
		third := r20Successor(t, r, cfg, src, r.plane)
		if err := third.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if third.serving {
			t.Fatal("expected process 3 to refuse adoption")
		}
		r.srv.SetPublication(api.PublicationIntentUnpublished)
		if err := third.Step(t.Context()); err != nil {
			t.Fatal(err)
		}
		u := r20ReadUsage(t, cfg, third.nodeID, third.incID)
		if len(u.gaps) != 0 || u.total() != 8192 || u.byRevision[1] != 8192 {
			t.Fatalf("H-1: two crashes attributed %v with %d gap event(s), want exactly 8192 bytes and no gap", u.byRevision, len(u.gaps))
		}
	})

	t.Run("boot_changed", func(t *testing.T) {
		// A reboot rebuilt the counters. Even numbers that look continuous
		// must not carry over: the epoch closes as a power-loss gap with no
		// records, exactly as before this change.
		r, src, upName := r20ServedPendingDelta(t)
		cfg := r20ChangedSets(r)
		newBoot := uuid.Must(uuid.NewV7()).String()
		if err := os.WriteFile(cfg.BootIDFile, []byte(newBoot+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		src.boot = newBoot
		src.setCounter(upName, enforce.CounterValue{Packets: 2, Bytes: 8192})
		next := r20Successor(t, r, cfg, src, r.plane)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if next.serving {
			t.Fatal("a changed boot adopted serving rules")
		}
		r.srv.SetPublication(api.PublicationIntentUnpublished)
		if err := next.Step(t.Context()); err != nil {
			t.Fatal(err)
		}
		u := r20ReadUsage(t, cfg, next.nodeID, next.incID)
		if u.total() != 4096 || len(u.gaps) != 1 || u.gaps[0].Gap == nil ||
			u.gaps[0].Gap.Reason != api.UsageGapReasonPowerLoss || len(u.gaps[0].Records) != 0 {
			t.Fatalf("boot change carried bytes over: attributed=%v gaps=%+v, want only process 1's 4096 bytes and one empty power-loss gap",
				u.byRevision, u.gaps)
		}
	})
}

type r20TermPlane struct {
	*memPlane
	onBaseline func()
}

// Baseline honors cancellation like the real nft executor.
func (p *r20TermPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	if p.onBaseline != nil {
		p.onBaseline()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.memPlane.Baseline(ctx, dp)
}

type r20CancelProber struct {
	cancel context.CancelFunc
}

func (p *r20CancelProber) Probe(ctx context.Context, req ProbeRequest) ProbeResult {
	if p.cancel != nil {
		p.cancel()
	}
	if ctx.Err() != nil {
		return ProbeResult{
			Ran:    true,
			Check:  api.HealthCheckStatusFailed,
			Health: api.DataHealthFailed,
			Detail: "probe_canceled",
		}
	}
	return passProbe{}.Probe(ctx, req)
}

// M-1: TERM (the run command's signal context) at each Boot stage. The exit
// status is computed with fvpn-agent run's rule (main.go: 1 only when Run
// fails without a stop request); TestR20_Q5_Held_* covers that rule in the
// command itself.
func TestR20_M1_TermDuringBootInstallsBaselineAndExitsZero(t *testing.T) {
	for _, stage := range []string{"lock_wait", "evaluation", "baseline"} {
		t.Run(stage, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			if err := r.eng.Close(); err != nil {
				t.Fatal(err)
			}
			if b := mustBeat(t, r); !b.Serving || !b.BaselineOK {
				t.Fatal("precondition: expected clean serving heartbeat after Close")
			}
			cfg := r.eng.cfg
			ctx, term := context.WithCancel(t.Context())
			defer term()
			plane := &r20TermPlane{memPlane: r.plane}
			var prober Prober = passProbe{}
			switch stage {
			case "lock_wait":
				// Unchanged sets: adoption proceeds to the node lock, which
				// another holder keeps busy until TERM arrives.
				lock, err := nodelock.TryAcquire(cfg.LockPath)
				if err != nil {
					t.Fatal(err)
				}
				defer lock.Release()
				timer := time.AfterFunc(100*time.Millisecond, term)
				defer timer.Stop()
			case "evaluation":
				prober = &r20CancelProber{cancel: term}
			case "baseline":
				cfg.ExtraDeniedV4 = []string{"203.0.113.1/32", "192.0.2.0/24"} // refused
				plane.onBaseline = term
			}
			next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: plane,
				MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: prober})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = next.Close() })
			r.eng = next
			baselines := r.plane.baselines
			started := r.clock.Mono()
			runErr := next.Run(ctx)
			exit := 0
			if runErr != nil && ctx.Err() == nil {
				exit = 1
			}
			hb := mustBeat(t, r)
			if !errors.Is(runErr, context.Canceled) || exit != 0 {
				t.Fatalf("M-1 %s: Run returned %v (exit %d), want context.Canceled and exit 0", stage, runErr, exit)
			}
			// r22 M-3: Boot reports the installed baseline generation, while
			// the durable applied ACK remains separately available on Engine.
			if hb.Serving || !hb.BaselineOK || hb.Generation != 0 || hb.Mono < started {
				t.Fatalf("M-1 %s: heartbeat %+v, want this process's Serving=false BaselineOK=true generation 0", stage, hb)
			}
			if r.plane.baselines <= baselines || r.plane.last.Generation != 0 || len(r.plane.last.Bindings) != 0 || next.serving {
				t.Fatalf("M-1 %s: kernel not on baseline: baselines %d->%d generation=%d bindings=%d serving=%v",
					stage, baselines, r.plane.baselines, r.plane.last.Generation, len(r.plane.last.Bindings), next.serving)
			}
		})
	}
}

type r20CountingProber struct {
	calls int
}

func (p *r20CountingProber) Probe(ctx context.Context, req ProbeRequest) ProbeResult {
	p.calls++
	return passProbe{}.Probe(ctx, req)
}

// M-2: a refusal decided by the retained model and the preliminary decision
// never waits for the node lock or runs the live probe / resolver refresh.
func TestR20_M2_RefusalPrecedesLockAndProbe(t *testing.T) {
	for _, tc := range []string{"changed_sets", "installation_sample"} {
		t.Run(tc, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			if err := r.eng.Close(); err != nil {
				t.Fatal(err)
			}
			cfg := r.eng.cfg
			switch tc {
			case "changed_sets":
				cfg.ExtraDeniedV4 = []string{"203.0.113.1/32", "192.0.2.0/24"}
			case "installation_sample":
				// The retained installation time no longer matches the
				// lease anchor; adoption must refuse on the clock check.
				disk, err := agentstore.Open(cfg.DBPath())
				if err != nil {
					t.Fatal(err)
				}
				raw, _, err := disk.Meta(t.Context(), "retained_applied")
				var ret retainedApplied
				if err == nil {
					err = json.Unmarshal([]byte(raw), &ret)
				}
				if err != nil {
					t.Fatal(err)
				}
				ret.Model.Now = ret.Model.Now.Add(time.Second)
				out, _ := json.Marshal(ret)
				if err := disk.SetMeta(t.Context(), "retained_applied", string(out)); err != nil {
					t.Fatal(err)
				}
				_ = disk.Close()
			}
			// A busy lock would cost the full 2.01 s acquisition schedule.
			lock, err := nodelock.TryAcquire(cfg.LockPath)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Release()
			prober := &r20CountingProber{}
			next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane,
				MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: prober})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = next.Close() })
			started := time.Now()
			if err := next.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			elapsed := time.Since(started)
			if next.serving || prober.calls != 0 || elapsed >= time.Second {
				t.Fatalf("M-2 %s: serving=%v probes=%d boot=%v; refusal must precede the lock wait and live probe",
					tc, next.serving, prober.calls, elapsed)
			}
		})
	}
}

type r20GenVerifyPlane struct {
	*memPlane
}

func (p *r20GenVerifyPlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	if err := ctx.Err(); err != nil {
		return enforce.ServingEvidence{}, err
	}
	if p.last.Generation != m.Generation {
		return enforce.ServingEvidence{}, &enforce.CheckError{Reason: enforce.ReasonGenerationMismatch}
	}
	return p.memPlane.VerifyServing(ctx, m, a)
}

// M-2: adoption's evaluation runs while this process holds the node lock. A
// live health challenge must not read that descriptor as the updater's lock
// (false D-064) and replace the retained rules with a probe baseline.
func TestR20_M2_AdoptionLockIsNotUpdaterLock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		probeChallenge(t, r, labProbe(), 0)
		closeAdoptionControl(t, r)
		if err := r.eng.Close(); err != nil {
			t.Fatal(err)
		}
		plane := &r20GenVerifyPlane{memPlane: r.plane}
		next, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: plane,
			MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = next.Close() })
		before := len(r.plane.calls)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		for _, c := range r.plane.calls[before:] {
			if c.kind == "baseline_model" && c.model.Probe != nil {
				t.Fatalf("M-2: adoption read its own node lock as the updater's and installed a probe baseline (D-064); serving=%v", next.serving)
			}
		}
		if !next.serving || r.plane.last.Generation != 2 {
			t.Fatalf("M-2: unchanged same-boot rules not adopted: serving=%v kernel generation=%d errors=%v",
				next.serving, r.plane.last.Generation, next.decision.Errors)
		}
	})
}

// M-3: a stop that withdraws serving records the withdrawal in the heartbeat.
func TestR20_M3_ShutdownHandoffRecordsWithdrawal(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	// The lease deadline passes before shutdown, so Close takes the hand-off.
	r.clock.Advance(2 * time.Hour)
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	hb := mustBeat(t, r)
	if hb.Serving || !hb.BaselineOK || r.plane.last.Generation != 0 {
		t.Fatalf("M-3: shutdown hand-off installed generation %d but left heartbeat Serving=%v BaselineOK=%v",
			r.plane.last.Generation, hb.Serving, hb.BaselineOK)
	}
}

type r20ForeignTCPlane struct {
	*memPlane
}

func (p *r20ForeignTCPlane) VerifyServing(context.Context, enforce.Model, map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	return enforce.ServingEvidence{}, &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter}
}

// M-3: rewritten from r20's proof, which rewound the monotonic clock two
// hours (production never does). Here the retained rules are replaced while
// no agent runs (another tool or an operator), inside the heartbeat's
// staleness window: adoption's live read fails and Boot installs a verified
// baseline. The stale apply error must not block a non-serving ACK.
func TestR20_M3_RefusedAdoptionErrorDoesNotBlockNonServingAck(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	dp, err := r.eng.cfg.DatapathValue()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.plane.Baseline(t.Context(), dp); err != nil { // out-of-band replacement
		t.Fatal(err)
	}
	next, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: &r20GenVerifyPlane{memPlane: r.plane},
		MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	r.eng = next
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.serving || !next.baselineOK {
		t.Fatalf("expected Boot to fall back to Baseline: serving=%v baselineOK=%v", next.serving, next.baselineOK)
	}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	if err := next.Step(t.Context()); err != nil {
		t.Fatal(err)
	}
	wantGen := uint64(next.desired.ConfigGeneration)
	if next.appliedGen != wantGen || next.decision.Safety != api.SafetyStatusPassed {
		t.Fatalf("M-3: stale adoption applyError=%v blocked unpublished generation ACK: appliedGen=%d want=%d safety=%s reasons=%v",
			next.applyError, next.appliedGen, wantGen, next.decision.Safety, next.decision.Reasons)
	}
}

// M-3 guard: a foreign tc filter found by adoption keeps its error and
// quarantine; only plain rejected reads are cleared by Boot's baseline.
func TestR20_M3_AdoptionQuarantineKeepsError(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: &r20ForeignTCPlane{memPlane: r.plane},
		MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.serving || !next.quarantinePending || next.applyError == nil || next.applyError.Code != api.ErrorCodeTCFOREIGNFILTER {
		t.Fatalf("foreign tc adoption lost containment: serving=%v quarantine=%v applyError=%v", next.serving, next.quarantinePending, next.applyError)
	}
}

// L-1 guard: a non-nil empty DeniedV4 never compares equal. It is also
// unreachable as installed state: enforce refuses to render it, so no
// verified Apply can retain it.
func TestR20_L1_EmptyDeniedV4NeverEqual(t *testing.T) {
	empty := []netip.Prefix{}
	for _, pair := range [][2]enforce.Model{
		{{DeniedV4: empty}, {DeniedV4: empty}},
		{{DeniedV4: empty}, {DeniedV4: nil}},
		{{DeniedV4: nil}, {DeniedV4: empty}},
	} {
		if sameRestartSets(pair[0], pair[1]) {
			t.Fatalf("L-1: sameRestartSets accepted non-nil empty DeniedV4: %v vs %v", pair[0].DeniedV4, pair[1].DeniedV4)
		}
	}
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	m := *r.eng.appliedModel
	m.DeniedV4 = empty
	if _, err := (&memPlane{}).Apply(t.Context(), m, nil); err == nil || !strings.Contains(err.Error(), "denied_v4 is empty") {
		t.Fatalf("an empty deny list rendered (err=%v); it could become retained state", err)
	}
}

// L-1: a crash between the retained_applied write and the ACK leaves the
// retained generation ahead of applied_snapshot. desired was accepted first,
// so it is ahead too: adoption refuses, Boot takes the baseline, and the
// meter keeps the bindings the kernel counters were installed with.
func TestR20_L1_RetainedAheadOfAckRefusesAdoption(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	ctx := t.Context()
	// enforce's durable order for generation 3: accept desired, Apply,
	// retained_applied; then the process dies before checkMemberPath/ACK.
	r.srv.SetFixture(generationFixture(3, oneBinding(time.Now().UTC())))
	if err := r.eng.syncDesired(ctx); err != nil || r.eng.desired.ConfigGeneration != 3 {
		t.Fatalf("setup: desired 3 not accepted: %v", err)
	}
	model := *r.eng.appliedModel
	model.Generation = 3
	if _, err := r.plane.Apply(ctx, model, nil); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(retainedApplied{Model: model, Assignments: r.eng.appliedAssignments})
	if err := r.eng.store.SetMeta(ctx, "retained_applied", string(raw)); err != nil {
		t.Fatal(err)
	}
	cfg, boot := r.eng.cfg, r.eng.bootID.String()
	r20Crash(r.eng)
	if b := mustBeat(t, r); !b.Serving || b.Generation != 2 {
		t.Fatalf("precondition: crash should leave the generation 2 serving heartbeat, got %+v", b)
	}
	next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, MeterSource: staticSource{boot: boot}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	if err := next.Boot(ctx); err != nil {
		t.Fatal(err)
	}
	if next.serving || next.appliedGen != 2 || next.appliedModel.Generation != 3 || r.plane.last.Generation != 0 {
		t.Fatalf("L-1: retained generation ahead of the ACK adopted or skipped baseline: serving=%v applied=%d retained=%d kernel=%d",
			next.serving, next.appliedGen, next.appliedModel.Generation, r.plane.last.Generation)
	}
	// F38: retired identities explain only the settled delta, never the new
	// epoch. F37's lasting-binding assertion encoded the M-1 regression.
	if got := next.meterBindings(); len(got) != 0 || !next.finalDone {
		t.Fatalf("retained epoch not closed and cleared: bindings=%v final=%v", got, next.finalDone)
	}
}

func TestR20_Q2_Held_SameRestartSetsPrefixSemantics(t *testing.T) {
	p := netip.MustParsePrefix
	spec := enforce.SpecDeniedV4()

	// 1. nil DeniedV4 vs explicit SpecDeniedV4() compare equal; nil vs empty ExtraDeniedV4/HostV4 compare equal.
	a := enforce.Model{DeniedV4: nil, ExtraDeniedV4: nil, HostV4: nil}
	b := enforce.Model{DeniedV4: append([]netip.Prefix(nil), spec...), ExtraDeniedV4: []netip.Prefix{}, HostV4: []netip.Prefix{}}
	if !sameRestartSets(a, b) {
		t.Fatal("nil vs explicit SpecDeniedV4 or empty ExtraDeniedV4/HostV4 compared unequal")
	}

	// 2. Non-canonical host bits, duplicates, and reordered prefixes compare equal after Masked().
	c := enforce.Model{
		ExtraDeniedV4: []netip.Prefix{p("203.0.113.7/24"), p("198.51.100.99/24"), p("203.0.113.0/24")},
		HostV4:        []netip.Prefix{p("192.0.2.2/32"), p("192.0.2.1/32")},
	}
	d := enforce.Model{
		ExtraDeniedV4: []netip.Prefix{p("198.51.100.0/24"), p("203.0.113.0/24")},
		HostV4:        []netip.Prefix{p("192.0.2.1/32"), p("192.0.2.2/32"), p("192.0.2.1/32")},
	}
	if !sameRestartSets(c, d) {
		t.Fatal("masked/reordered/duplicate prefixes compared unequal")
	}

	// 3. Invalid zero prefix, IPv6 prefix, and IPv4-mapped IPv6 prefix fail closed.
	for _, bad := range []netip.Prefix{
		{},
		p("2001:db8::/32"),
		p("::ffff:192.0.2.1/128"),
	} {
		m := enforce.Model{ExtraDeniedV4: []netip.Prefix{bad}}
		if sameRestartSets(m, m) {
			t.Fatalf("invalid or non-IPv4 prefix %v was accepted by sameRestartSets", bad)
		}
	}

	// 4. Split/merged prefixes (/24 vs two /25s, /0 vs two /1s) compare unequal (fail-closed availability-only baseline).
	whole := enforce.Model{ExtraDeniedV4: []netip.Prefix{p("203.0.113.0/24")}}
	halves := enforce.Model{ExtraDeniedV4: []netip.Prefix{p("203.0.113.0/25"), p("203.0.113.128/25")}}
	if sameRestartSets(whole, halves) {
		t.Fatal("expected exact prefix-set inequality between /24 and two /25s")
	}
}

func TestR20_Q3_Held_CorruptOrRebootedRetainedAppliedFailsClosed(t *testing.T) {
	for _, tc := range []string{"missing", "corrupt_json", "zero_gen", "new_boot_id"} {
		t.Run(tc, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			if err := r.eng.Close(); err != nil {
				t.Fatal(err)
			}
			cfg := r.eng.cfg
			bootID := r.eng.bootID.String()

			disk, err := agentstore.Open(cfg.DBPath())
			if err != nil {
				t.Fatal(err)
			}
			switch tc {
			case "missing":
				if err := disk.SetMeta(t.Context(), "retained_applied", ""); err != nil {
					t.Fatal(err)
				}
			case "corrupt_json":
				if err := disk.SetMeta(t.Context(), "retained_applied", "{not-json"); err != nil {
					t.Fatal(err)
				}
			case "zero_gen":
				if err := disk.SetMeta(t.Context(), "retained_applied", `{"model":{"Generation":0}}`); err != nil {
					t.Fatal(err)
				}
			case "new_boot_id":
				newBoot := uuid.Must(uuid.NewV7()).String()
				if err := os.WriteFile(cfg.BootIDFile, []byte(newBoot+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				bootID = newBoot
			}
			if err := disk.Close(); err != nil {
				t.Fatal(err)
			}

			next, err := Start(Options{
				Config:      cfg,
				Clock:       r.clock,
				Plane:       r.plane,
				MeterSource: staticSource{boot: bootID},
				Prober:      passProbe{},
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = next.Close() })
			baselines := r.plane.baselines
			bootErr := next.Boot(t.Context())
			if tc == "missing" || tc == "corrupt_json" {
				if bootErr == nil || next.serving || r.plane.baselines != baselines+1 {
					t.Fatalf("case %s did not fail closed on loadRuntime: err=%v serving=%v baselines=%d->%d",
						tc, bootErr, next.serving, baselines, r.plane.baselines)
				}
				return
			}
			if bootErr != nil {
				t.Fatal(bootErr)
			}
			if next.serving || !next.baselineOK || r.plane.baselines != baselines+1 {
				t.Fatalf("case %s did not fail closed to baseline: serving=%v baselineOK=%v baselines=%d->%d",
					tc, next.serving, next.baselineOK, baselines, r.plane.baselines)
			}
		})
	}
}

func TestR20_Q6_Held_TwoCrashesAndRebootCounterRecovery(t *testing.T) {
	t.Run("two_consecutive_crashes_same_boot_recover_once", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			r.srv.SetUsageUnavailable(1000)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			closeAdoptionControl(t, r)
			// A changed counter needs a nonempty interval on the bubble clock.
			time.Sleep(time.Microsecond)

			binding := r.eng.appliedModel.Bindings[0]
			upName := meter.CounterName(binding.DeviceID, meter.DirUpload)

			r.eng.meter.Close()
			src := &r20DynamicSource{
				boot:     r.eng.bootID.String(),
				bindings: r.eng.meterBindings,
				counters: map[string]enforce.CounterValue{upName: {Packets: 1, Bytes: 4096}},
			}
			var err error
			r.eng.meter, err = meter.New(meter.Config{
				NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src,
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := r.eng.meter.Sample(t.Context()); err != nil {
				t.Fatal(err)
			}
			// Crash 1: counter advances to 6144, process dies without final sample.
			src.setCounter(upName, enforce.CounterValue{Packets: 2, Bytes: 6144})
			r.eng.meter.Close()
			_ = r.eng.usage.Close()
			if r.eng.usageLock != nil {
				r.eng.usageLock.Release()
			}
			_ = r.eng.store.Close()

			// Process 2 adopts same-boot serving, then crashes before sampling while counter advances to 8192.
			eng2, err := Start(Options{
				Config:      r.eng.cfg,
				Clock:       r.clock,
				Plane:       r.plane,
				MeterSource: src,
				Prober:      passProbe{},
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				r20Crash(eng2)
				eng2.client.Close()
			})
			src.setBindings(eng2.meterBindings)
			if err := eng2.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			if !eng2.serving {
				t.Fatal("expected eng2 to adopt same-boot serving")
			}
			if eng2.loopCancel != nil {
				eng2.loopCancel()
				eng2.loops.Wait()
			}
			src.setCounter(upName, enforce.CounterValue{Packets: 3, Bytes: 8192})
			eng2.meter.Close()
			_ = eng2.usage.Close()
			if eng2.usageLock != nil {
				eng2.usageLock.Release()
			}
			_ = eng2.store.Close()

			// Process 3 adopts same-boot serving and samples once.
			eng3, err := Start(Options{
				Config:      r.eng.cfg,
				Clock:       r.clock,
				Plane:       r.plane,
				MeterSource: src,
				Prober:      passProbe{},
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = eng3.Close() })
			src.setBindings(eng3.meterBindings)
			if err := eng3.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			if !eng3.serving {
				t.Fatal("expected eng3 to adopt same-boot serving")
			}
			if eng3.loopCancel != nil {
				eng3.loopCancel()
				eng3.loops.Wait()
			}
			time.Sleep(time.Microsecond)
			if err := eng3.meter.Sample(t.Context()); err != nil {
				t.Fatal(err)
			}
			snap, err := usagestore.ReadSnapshot(t.Context(), r.eng.cfg.UsageDBPath())
			if err != nil {
				t.Fatal(err)
			}
			var total uint64
			for _, ev := range snap.UnackedEvents {
				for _, rec := range ev.Records {
					n, err := strconv.ParseUint(rec.UploadBytes, 10, 64)
					if err != nil {
						t.Fatal(err)
					}
					total += n
				}
			}
			if total != 8192 {
				t.Fatalf("recovered %d bytes across two consecutive crashes, want 8192", total)
			}
		})
	})
}
