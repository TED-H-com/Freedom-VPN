package agent

import (
	"context"
	"net/netip"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
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

// TODO(review r20): H-1 — When adoptStoredServing returns false on a changed
// deny/host set, loadRuntime never populates e.meterSnap from retained_applied
// and Boot does not sample/handoff the open same-boot epoch before Step. An
// unpublished/non-serving Step or a Step that removes/rotates the binding
// records a mapping-conflict UsageQualityGap and drops the member's unsampled bytes.
func TestR20_H1_RefusedAdoptionDropsPendingUsageDelta(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	binding := r.eng.appliedModel.Bindings[0]
	upName := meter.CounterName(binding.DeviceID, meter.DirUpload)

	// Wire Process 1's meter to a source that reads Bindings through e.meterBindings(),
	// matching production kernelSource in ensureUsage (usage.go:162-168).
	r.eng.meter.Close()
	src1 := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		bindings: r.eng.meterBindings,
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

	// Simulate an interrupted Close (or packets traversing the retained ruleset
	// between preserveServingOnStop and the successor's Boot Baseline): 4096 more
	// bytes accumulate in the live kernel counter before Process 1 closes.
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}

	// Process 2 starts on the same boot with a grown ExtraDeniedV4, so
	// adoptStoredServing refuses adoption at sameRestartSets.
	cfg := r.eng.cfg
	cfg.ExtraDeniedV4 = []string{"203.0.113.1/32", "192.0.2.0/24"}
	src2 := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		counters: map[string]enforce.CounterValue{upName: {Packets: 2, Bytes: 8192}},
	}
	next, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		MeterSource: src2,
		Prober:      passProbe{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	src2.bindings = next.meterBindings

	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.serving {
		t.Fatal("expected Boot to refuse same-boot adoption on changed ExtraDeniedV4")
	}

	// Central transitions the node to unpublished (or any non-serving state)
	// before the first serving Apply. Step runs handoff -> FinalSample.
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	if err := next.Step(t.Context()); err != nil {
		t.Fatal(err)
	}

	snap, err := usagestore.ReadSnapshot(t.Context(), cfg.UsageDBPath())
	if err != nil {
		t.Fatal(err)
	}
	var totalBytes uint64
	for _, ev := range snap.UnackedEvents {
		if len(ev.Records) == 0 {
			t.Fatalf("H-1: refused same-boot adoption turned 4096 unsampled member bytes into an unattributed gap event (seq=%s records=0)",
				ev.EventSeq)
		}
		for _, rec := range ev.Records {
			b, err := strconv.ParseUint(rec.UploadBytes, 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			totalBytes += b
		}
	}
	if totalBytes != 8192 {
		t.Fatalf("H-1: attributed %d upload bytes across restart, want 8192", totalBytes)
	}
}

type r20CancelBootPlane struct {
	*memPlane
}

func (p *r20CancelBootPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
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

// TODO(review r20): M-1 — When SIGTERM cancels ctx during Boot (now reachable
// because NotifyContext is installed before open/Boot, e.g. while
// adoptStoredServing's evaluate/Probe is running), adoptStoredServing returns
// false and e.plane.Baseline(ctx, dp) runs with the canceled ctx and fails
// without installing the baseline, while e.beat() overwrites the valid serving
// heartbeat on disk with Serving: false, BaselineOK: false and Boot returns nil.
func TestR20_M1_TermDuringBootPoisonsHeartbeatWithoutBaseline(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	if !mustBeat(t, r).Serving || !mustBeat(t, r).BaselineOK {
		t.Fatal("precondition: expected clean serving heartbeat after Close")
	}

	cfg := r.eng.cfg
	bootID := r.eng.bootID.String()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	plane2 := &r20CancelBootPlane{memPlane: r.plane}
	next, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       plane2,
		MeterSource: staticSource{boot: bootID},
		Prober:      &r20CancelProber{cancel: cancel},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	r.eng = next

	baselinesBefore := r.plane.baselines
	bootErr := next.Boot(ctx)
	hb := mustBeat(t, r)
	if bootErr == nil && !hb.BaselineOK && r.plane.baselines == baselinesBefore {
		t.Fatalf("M-1: canceled Boot returned nil without installing Baseline (baselines=%d) and overwrote heartbeat with Serving=%v BaselineOK=%v Reason=%q",
			r.plane.baselines, hb.Serving, hb.BaselineOK, hb.Reason)
	}
}

type r20CountingProber struct {
	calls int
}

func (p *r20CountingProber) Probe(ctx context.Context, req ProbeRequest) ProbeResult {
	p.calls++
	return passProbe{}.Probe(ctx, req)
}

// TODO(review r20): M-2 — adoptStoredServing runs e.evaluate(ctx) (live TCP/DNS
// probes and HTTP resolver fetches) while holding nodelock and BEFORE checking
// sameRestartSets(*installed, d.Model), extending the old ruleset's live
// forwarding window and causing e.updateLockHeld() to self-contend on LockPath.
func TestR20_M2_SameRestartSetsRunsAfterProbeAndSelfLocksD064(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}

	cfg := r.eng.cfg
	cfg.ExtraDeniedV4 = []string{"203.0.113.1/32", "192.0.2.0/24"}
	prober := &r20CountingProber{}
	next, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		MeterSource: staticSource{boot: r.eng.bootID.String()},
		Prober:      prober,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })

	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if prober.calls != 0 {
		t.Fatalf("M-2: adoptStoredServing ran live Prober %d time(s) while old deny sets were still active before checking sameRestartSets", prober.calls)
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

// TODO(review r20): M-3 — When shutdownUsage runs handoff (because
// preserveServingOnStop is false), it installs Baseline without updating the
// on-disk heartbeat (Serving remains true). On the next Boot, adoptStoredServing
// fails verifyServingModel and latches e.applyError; Boot's Baseline does not
// clear e.applyError, blocking ACK of a subsequent unpublished generation.
func TestR20_M3_FailedAdoptionLatchesApplyErrorAndBlocksUnpublishedAck(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	// Advance clock past lease expiry so preserveServingOnStop returns false during Close.
	r.clock.Advance(2 * time.Hour)
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	// shutdownUsage -> handoff installed Baseline on r.plane (Generation=0), but left
	// Serving: true in the on-disk heartbeat.
	if !mustBeat(t, r).Serving {
		t.Fatal("precondition: expected shutdownUsage handoff to leave Serving=true in heartbeat")
	}
	// Rewind clock into the lease window to exercise adoptStoredServing hitting VerifyServing
	// against that Baseline, then transitioning to unpublished on Step.
	r.clock.Advance(-2 * time.Hour)
	cfg := r.eng.cfg
	next, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       &r20GenVerifyPlane{memPlane: r.plane},
		MeterSource: staticSource{boot: r.eng.bootID.String()},
		Prober:      passProbe{},
	})
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

	// Central now publishes an unpublished generation. Step must install/verify
	// Baseline and ACK the unpublished generation cleanly.
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

// TODO(review r20): L-1 — sameRestartSets treats two non-nil empty DeniedV4
// slices ([]netip.Prefix{}) as equal instead of rejecting or defaulting them,
// and adoptStoredServing does not check installed.Generation == e.appliedGen.
func TestR20_L1_SameRestartSetsEmptyNonNilDeniedV4AndGenerationMismatch(t *testing.T) {
	empty := []netip.Prefix{}
	m1 := enforce.Model{DeniedV4: empty}
	m2 := enforce.Model{DeniedV4: empty}
	if sameRestartSets(m1, m2) {
		t.Fatal("L-1: sameRestartSets accepted non-nil empty DeniedV4=[]netip.Prefix{} (which enforce.Model.denyV4 rejects)")
	}
}

func TestR20_Held_SameRestartSetsPrefixSemantics(t *testing.T) {
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

func TestR20_Held_CorruptOrRebootedRetainedAppliedFailsClosed(t *testing.T) {
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

func TestR20_Held_TwoCrashesAndRebootCounterRecovery(t *testing.T) {
	t.Run("two_consecutive_crashes_same_boot_recover_once", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		r.srv.SetUsageUnavailable(1000)
		serveBoundGeneration(t, r)
		r.manualUsage(t)

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
}
