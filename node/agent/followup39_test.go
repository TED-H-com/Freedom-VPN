package agent

// TEST ONLY in-memory data planes and disk SQLite. No host networking.
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

type r22CrashFault struct{}
type r22CrashPlane struct {
	*memPlane
	afterApply    func()
	afterBaseline func()
}

func (p *r22CrashPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.memPlane.Apply(ctx, m, prev)
	if err == nil && p.afterApply != nil {
		p.afterApply()
	}
	return res, err
}
func (p *r22CrashPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	err := p.memPlane.Baseline(ctx, dp)
	if err == nil && p.afterBaseline != nil {
		p.afterBaseline()
	}
	return err
}
func r22CrashDuringStep(t *testing.T, r *rig) {
	t.Helper()
	killed := false
	func() {
		defer func() {
			if v := recover(); v != nil {
				if _, ok := v.(r22CrashFault); !ok {
					panic(v)
				}
				killed = true
			}
		}()
		_ = r.eng.Step(t.Context())
	}()
	if !killed {
		t.Fatalf("fault was not reached: serving=%v reasons=%v", r.eng.serving, r.eng.decision.Reasons)
	}
	r20Crash(r.eng)
}

// r22WaitPastCheckpoint sleeps until real time is 10ms past the newest
// usage checkpoint. A production meter reads time.Now, so callers run
// this before one samples. The sleep is a lower bound: a slower host
// arrives later and waits less.
func r22WaitPastCheckpoint(t *testing.T, e *Engine) {
	t.Helper()
	st, err := usagestore.Open(e.cfg.UsageDBPath(), e.nodeID, e.incID, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	head, err := st.Head(t.Context())
	_ = st.Close()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(max(0, time.Until(head.LastSample)+10*time.Millisecond))
}

func TestR22_H1B_FailedHandoffSettlementAndSampling(t *testing.T) {
	for _, kind := range []string{"hold", "read_error"} {
		for _, restart := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s_restart_%v", kind, restart), func(t *testing.T) {
				r := newRig(t)
				alignControlClock(r)
				r.srv.SetUsageUnavailable(1000)
				serveBoundGeneration(t, r)
				r.manualUsage(t)
				var fail atomic.Bool
				var offset atomic.Int64
				if kind == "hold" {
					offset.Store(int64(1500 * time.Millisecond))
				}
				// This meter reads the rig clock. The production meter has
				// already checkpointed time.Now, so catch the rig clock up
				// before sampling: a stamp behind that checkpoint is held,
				// and one a full sample interval behind is a clock gap.
				// The first reading is the rig clock plus offset. The rig
				// clock starts at real time, so a hold steps the wall clock
				// back by that offset and the reading is real time, not
				// ahead of it. The handoff then reads with offset 0, 1.5s
				// behind the checkpoint, and stays held whatever real time does.
				delta := time.Now().Add(-time.Duration(offset.Load())).Sub(r.clock.Wall())
				if delta > 0 {
					r.clock.Advance(delta)
				} else if delta < 0 {
					r.clock.JumpWall(delta)
				}
				a := oneBinding(r.clock.Wall())
				b := a
				b.DeviceId, b.MemberId = uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
				b.PeerId, b.BindingRevision, b.AllowedIpv4 = "peer-b", 2, []string{"100.64.1.2/32"}
				upA, upB := meter.CounterName(a.DeviceId, meter.DirUpload), meter.CounterName(b.DeviceId, meter.DirUpload)
				src := &r20DynamicSource{boot: r.eng.bootID.String(), bindings: r.eng.meterBindings, counters: map[string]enforce.CounterValue{upA: {Packets: 1, Bytes: 1000}, upB: {}}}
				source := r21ReadFunc(func(ctx context.Context) (meter.Observation, error) {
					if fail.Load() {
						return meter.Observation{}, errors.New("TEST ONLY counter read failure")
					}
					return src.Read(ctx)
				})
				r.eng.meter.Close()
				var err error
				r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: source, Now: func() time.Time { return r.clock.Wall().Add(time.Duration(offset.Load())) }})
				if err != nil {
					t.Fatal(err)
				}
				if err := r.eng.meter.Sample(t.Context()); err != nil {
					t.Fatal(err)
				}
				f := generationFixture(3, a)
				f.Bindings = []api.DeviceBinding{a, b}
				r.srv.SetFixture(f)
				r.eng.leaseRefreshPending = true
				stepOK(t, r)
				if !r.eng.serving || r.eng.appliedGen != 3 {
					t.Fatal("generation 3 not serving")
				}
				src.setCounter(upA, enforce.CounterValue{Packets: 2, Bytes: 3000})
				src.setCounter(upB, enforce.CounterValue{Packets: 1, Bytes: 4096})
				offset.Store(0)
				fail.Store(kind == "read_error")
				r.eng.handoff(t.Context())
				if r.eng.serving || !r.eng.retainedPending || r.eng.finalDone {
					t.Fatal("failed final sample was not left pending")
				}
				if restart {
					// The restarted meter reads real time, so only a failing
					// read keeps the final sample pending without depending
					// on elapsed time.
					fail.Store(true)
					r20Crash(r.eng)
					// A production meter reads time.Now. Wait until real
					// time is past the newest checkpoint before it samples.
					// A slower host reaches here later and waits less.
					r22WaitPastCheckpoint(t, r.eng)
					next, err := Start(Options{Config: r20ChangedSets(r), Clock: r.clock, Plane: r.plane, MeterSource: source, Prober: passProbe{}})
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = next.Close() })
					src.setBindings(next.meterBindings)
					r.eng = next
					if err := next.Boot(t.Context()); err != nil {
						t.Fatal(err)
					}
				}
				rebound := b
				rebound.MemberId, rebound.BindingRevision = uuid.Must(uuid.NewV7()), 3
				f = generationFixture(4, a)
				f.Bindings = []api.DeviceBinding{a, rebound}
				r.srv.SetFixture(f)
				before := r.plane.applies
				stepOK(t, r)
				if r.eng.serving || !r.eng.retainedPending || r.plane.applies != before {
					t.Fatal("pending final sample allowed Apply or serving")
				}
				if r.eng.desired != nil && r.eng.desired.ConfigGeneration == 4 {
					t.Fatal("pending interval allowed desired refresh")
				}
				if kind == "hold" && !restart {
					st, err := usagestore.Open(r.eng.cfg.UsageDBPath(), r.eng.nodeID, r.eng.incID, usagestore.Options{})
					if err != nil {
						t.Fatal(err)
					}
					head, err := st.Head(t.Context())
					_ = st.Close()
					if err != nil {
						t.Fatal(err)
					}
					// This meter still reads the rig clock. The restarted
					// meter does not reach here: its reads fail until below,
					// and real time is already past the checkpoint. Sleeping
					// would not release this sample.
					if wait := head.LastSample.Sub(r.clock.Wall()) + 10*time.Millisecond; wait > 0 {
						r.clock.Advance(wait)
					}
				} else if !restart {
					// A changed counter read at the checkpoint time is held.
					// This meter reads the rig clock, so move it forward.
					r.clock.Advance(time.Millisecond)
				}
				fail.Store(false)
				r.eng.leaseRefreshPending = true
				stepOK(t, r)
				if !r.eng.serving || r.eng.retainedPending || r.eng.appliedGen != 4 {
					t.Fatalf("recovery did not serve: %v", r.eng.decision.Reasons)
				}
				u := r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
				if u.byRevision[1] != 3000 || u.byRevision[2] != 4096 || u.byRevision[3] != 0 || len(u.gaps) != 0 {
					t.Fatalf("outgoing attribution: %v gaps=%d", u.byRevision, len(u.gaps))
				}

				// Exercise Run itself, not just Sample: a paused sampler would silently
				// skip this delta. Stop the fixture's old loop before starting this one.
				r.manualUsage(t)
				if !restart {
					// Run reads this same clock. Settlement just checkpointed
					// the open epoch at the current reading, and a sample at
					// that reading is held.
					r.clock.Advance(time.Millisecond)
				}
				src.setCounter(upA, enforce.CounterValue{Packets: 3, Bytes: 4000})
				src.setCounter(upB, enforce.CounterValue{Packets: 2, Bytes: 5120})
				ctx, cancel := context.WithCancel(t.Context())
				done := make(chan struct{})
				go func() { defer close(done); _ = r.eng.meter.Run(ctx) }()
				r.eng.meter.ResumeSampling()
				// A loaded host can take longer than a second to publish the
				// sample. Leave as soon as the ledger shows the delta.
				deadline := time.Now().Add(30 * time.Second)
				for time.Now().Before(deadline) {
					u = r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
					if u.byRevision[3] == 1024 {
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
				cancel()
				<-done
				if u.byRevision[3] != 1024 || u.byRevision[1] != 4000 || len(u.gaps) != 0 {
					t.Fatalf("periodic sampling did not resume: %v gaps=%d", u.byRevision, len(u.gaps))
				}
				r20Crash(r.eng)
				cfg := r.eng.cfg
				cfg.ExtraDeniedV4 = append(append([]string(nil), cfg.ExtraDeniedV4...), "198.51.100.0/24")
				next := r20Successor(t, r, cfg, src, r.plane)
				// The successor's meter reads time.Now. Wait until real
				// time is past the newest checkpoint before it samples.
				// A slower host reaches here later and waits less.
				r22WaitPastCheckpoint(t, next)
				if err := next.Boot(t.Context()); err != nil {
					t.Fatal(err)
				}
				u = r21ReadLedger(t, cfg, next.nodeID, next.incID)
				if u.byRevision[1] != 4000 || u.byRevision[2] != 4096 || u.byRevision[3] != 1024 || len(u.gaps) != 0 {
					t.Fatalf("post-recovery restart changed attribution: %v gaps=%d", u.byRevision, len(u.gaps))
				}
			})
		}
	}
}

func TestR22_M3_DatapathValidationGuard(t *testing.T) {
	r := newRig(t)
	cfg := r.eng.cfg
	cfg.Datapath.MemberIngress = "invalid-interface"
	if err := cfg.validate(); err == nil {
		t.Fatal("production configuration accepted invalid datapath")
	}
	// TEST ONLY mutation after LoadConfig: keep the unreachable Boot guard.
	r.eng.cfg = cfg
	err := r.eng.Boot(t.Context())
	if !errors.Is(err, ErrBoot) {
		t.Fatalf("guard returned %v", err)
	}
	hb, err := watchdog.ReadHeartbeat(cfg.HeartbeatPath)
	if err != nil || hb.Serving || hb.BaselineOK {
		t.Fatalf("guard heartbeat: %+v %v", hb, err)
	}
}

func TestR22_M3_ProbeClearRetriesWithoutBurningNonceOrExtendingLifetime(t *testing.T) {
	r := newRig(t)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	enrollBoot(t, r)
	stepOK(t, r)
	holdUpdateLock(t, r)
	ch := probeChallenge(t, r, labProbe(), 0)
	calls := 0
	r.eng.plane = &r22FailClearProbePlane{memPlane: r.plane, calls: &calls}
	if err := r.eng.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r.eng.probeCleanup == nil || r.plane.last.Probe == nil {
		t.Fatal("failed clear did not stay pending")
	}
	installed := *r.plane.last.Probe
	pending := r.eng.probeCleanup.issuance
	if pending.Cleared || pending.Nonce != ch.Nonce || installed.TimeoutCeilSeconds > 300 {
		t.Fatal("nonce or timeout changed")
	}
	before := len(r.plane.calls)
	r.clock.Advance(time.Second)
	r.eng.plane = r.plane
	ran := false
	r.eng.withProbeException(t.Context(), func() { ran = true })
	if !ran || r.eng.probeCleanup != nil || r.plane.last.Probe != nil {
		t.Fatal("clear retry failed")
	}
	for _, call := range r.plane.calls[before:] {
		if call.model.Probe != nil {
			t.Fatal("clear retry reinstalled the probe exception")
		}
	}
	raw, ok, err := r.eng.store.Meta(t.Context(), probeIssuanceKey)
	if err != nil || !ok {
		t.Fatal("missing clear receipt")
	}
	var cleared probeIssuance
	if err := json.Unmarshal([]byte(raw), &cleared); err != nil {
		t.Fatal(err)
	}
	if !cleared.Cleared || cleared.Nonce != pending.Nonce || cleared.Deadline != pending.Deadline {
		t.Fatal("clear receipt changed issuance or extended lifetime")
	}
	r.eng.beat()
	hb := mustBeat(t, r)
	if !hb.BaselineOK || hb.Serving {
		t.Fatal("clear recovery heartbeat is not baseline")
	}
}

func TestR22_H2_RemovedThenReaddedCounterKeepsOlderBytesOutOfNewBinding(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	b := oneBinding(r.clock.Wall())
	up := meter.CounterName(b.DeviceId, meter.DirUpload)
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
	f := generationFixture(3, b)
	f.Bindings = nil
	r.srv.SetFixture(f)
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	if len(r.plane.last.Bindings) != 0 {
		t.Fatal("device not removed")
	}
	b.MemberId, b.BindingRevision = uuid.Must(uuid.NewV7()), 2
	r.srv.SetFixture(generationFixture(4, b))
	r.eng.leaseRefreshPending = true
	r.eng.plane = &r22CrashPlane{memPlane: r.plane, afterApply: func() {
		src.setCounter(up, enforce.CounterValue{Packets: 3, Bytes: 9216})
		panic(r22CrashFault{})
	}}
	r22CrashDuringStep(t, r)
	next := r20Successor(t, r, r20ChangedSets(r), src, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || u.byRevision[2] != 1024 || len(u.gaps) != 0 {
		t.Fatalf("counter reuse: %v gaps=%d", u.byRevision, len(u.gaps))
	}
}
