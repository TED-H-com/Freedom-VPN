package agent

// TEST ONLY: manual clocks, in-memory planes and disk SQLite; no timing run.
import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func f42Siblings(t *testing.T, member bool) *rig {
	t.Helper()
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	now := r.clock.Wall()
	survivor := oneBinding(now)
	survivor.ValidUntil = now.Add(80 * time.Second)
	expired := survivor
	expired.DeviceId = uuid.Must(uuid.NewV7())
	expired.AllowedIpv4 = []string{"100.64.0.12/32"}
	if member {
		expired.MemberId = uuid.Must(uuid.NewV7())
	} else {
		expired.ValidUntil = now.Add(20 * time.Second)
	}
	f := fixtureWithBindings(3, []api.DeviceBinding{survivor, expired})
	f.TTL = 2 * time.Hour
	r.srv.SetFixture(f)
	stepOK(t, r)
	if member {
		for i := range r.eng.lease.Members {
			if r.eng.lease.Members[i].MemberId == expired.MemberId {
				r.eng.lease.Members[i].LeaseEnd = now.Add(20 * time.Second)
			}
		}
		stepOK(t, r)
	}
	signedAt := r.eng.desired.IssuedAt.Add(time.Minute)
	r.srv.SetClock(func() time.Time { return signedAt })
	return r
}

type f42WatchPlane struct{ *memPlane }

func (p f42WatchPlane) BaselinePresent(context.Context) (bool, error) { return true, nil }
func f42Watch(r *rig) *watchdog.Watch {
	return &watchdog.Watch{BootID: r.eng.bootID, Clock: r.clock, Enforce: f42WatchPlane{r.plane}, Config: watchdog.Config{
		LockPath: r.eng.cfg.LockPath, HeartbeatPath: r.eng.cfg.HeartbeatPath, WatchdogStaleSeconds: 30,
		Datapath: watchdog.DatapathConfig{MemberIngress: "fvtest", Uplink: "fvup", SNAT: "192.0.2.1"}}}
}

func TestAT13_F42_SiblingDeadlineAppliesWithoutBaseline(t *testing.T) {
	for _, member := range []bool{false, true} {
		for _, path := range []string{"entry", "verify", "retain"} {
			t.Run(map[bool]string{false: "device", true: "member"}[member]+"/"+path, func(t *testing.T) {
				r := f42Siblings(t, member)
				deadline := r.eng.policyExpires
				last := mustBeat(t, r).PolicyExpires
				if last <= deadline {
					t.Fatal("watchdog still uses first binding")
				}
				before, applies := r.plane.baselines, r.plane.applies
				r.clock.Advance(deadline - r.clock.Mono() - 200*time.Millisecond)
				if got := r.eng.nextStepDelay(time.Second); got != 200*time.Millisecond {
					t.Fatal("lost early wake", got)
				}
				calls := 0
				if path == "entry" {
					r.clock.Advance(200 * time.Millisecond)
				} else {
					r.eng.plane = &livePlane{memPlane: r.plane, read: func(context.Context) error {
						calls++
						if (path == "verify" && calls == 1) || (path == "retain" && calls == 2) {
							r.clock.Advance(200 * time.Millisecond)
						}
						return nil
					}}
				}
				stepOK(t, r)
				if !r.eng.serving || len(r.plane.last.Bindings) != 1 || r.plane.baselines != before || r.plane.applies != applies+1 {
					t.Fatalf("survivor transition: serving=%v bindings=%d baseline=%d->%d apply=%d->%d", r.eng.serving, len(r.plane.last.Bindings), before, r.plane.baselines, applies, r.plane.applies)
				}
				hb := mustBeat(t, r)
				rep, ok := r.srv.LastObservation()
				if !hb.Serving || hb.Reason != "serving" || hb.PolicyExpires != last || !ok || !rep.MemberForwardingEnabled || len(rep.ForwardingBlockReasons) != 0 {
					t.Fatalf("untruthful state: %+v %+v", hb, rep)
				}
				r.clock.Advance(last - r.clock.Mono())
				stepOK(t, r)
				hb = mustBeat(t, r)
				rep, _ = r.srv.LastObservation()
				if r.eng.serving || r.plane.baselines == before || hb.Serving || hb.PolicyExpires != 0 || hb.Reason == "-" || rep.MemberForwardingEnabled || len(rep.ForwardingBlockReasons) == 0 {
					t.Fatal("last expiry did not withdraw/report", hb, rep.ForwardingBlockReasons)
				}
			})
		}
	}
}

func TestAT13_F42_WatchdogAcrossSiblingAndLastDeadline(t *testing.T) {
	for _, member := range []bool{false, true} {
		for _, end := range []string{"stale", "last"} {
			t.Run(map[bool]string{false: "device", true: "member"}[member]+"/"+end, func(t *testing.T) {
				r := f42Siblings(t, member)
				first, last := r.eng.policyExpires, mustBeat(t, r).PolicyExpires
				r.clock.Advance(first - r.clock.Mono() - time.Second)
				stepOK(t, r)
				before := r.plane.baselines
				r.clock.Advance(time.Second)
				w := f42Watch(r)
				// No agent heartbeat since the earlier device expired, and lock is free.
				check := func() {
					lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
					if err != nil {
						t.Fatal(err)
					}
					lock.Release()
					if err := w.Once(t.Context()); err != nil {
						t.Fatal(err)
					}
					if r.plane.baselines != before {
						t.Fatal("watchdog closed surviving binding")
					}
				}
				check()
				r.eng.prober = r25ProbeRecorder{next: r.eng.prober, onProbe: check}
				stepOK(t, r) // Watch.Once also runs during the unlocked control/probe pass.
				r.eng.prober = passProbe{}
				if end == "stale" {
					r.clock.Advance(31 * time.Second)
				} else {
					r.clock.Advance(last - r.clock.Mono() - time.Second)
					stepOK(t, r)
					r.clock.Advance(time.Second)
				}
				if err := w.Once(t.Context()); err != nil {
					t.Fatal(err)
				}
				if r.plane.baselines != before+1 {
					t.Fatal("watchdog did not close", end)
				}
			})
		}
	}
}

func TestMTR03_F42_ExpiredRemovalSettlesBytesWithoutClosingInterval(t *testing.T) {
	r := f42Siblings(t, false)
	r.srv.SetUsageUnavailable(1000) // keep committed events visible across restart
	bindings := r.eng.meterBindings()
	src := &r20DynamicSource{boot: r.eng.bootID.String(), bindings: r.eng.meterBindings, counters: map[string]enforce.CounterValue{}}
	r.eng.meter.Close()
	var err error
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bindings {
		src.setCounter(meter.CounterName(b.DeviceID, meter.DirUpload), enforce.CounterValue{Packets: 1, Bytes: 1024})
	}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, b := range bindings {
		src.setCounter(meter.CounterName(b.DeviceID, meter.DirUpload), enforce.CounterValue{Packets: 2, Bytes: 2048})
	}
	h, _ := r.eng.usage.Head(t.Context())
	before := r.plane.baselines
	r.clock.Advance(r.eng.policyExpires - r.clock.Mono())
	stepOK(t, r)
	next, _ := r.eng.usage.Head(t.Context())
	if next.OpenEpoch != h.OpenEpoch || r.plane.baselines != before {
		t.Fatal("expired-only removal closed interval")
	}
	survivor := r.eng.meterBindings()[0]
	src.setCounter(meter.CounterName(survivor.DeviceID, meter.DirUpload), enforce.CounterValue{Packets: 3, Bytes: 3072})
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	ledger := r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
	if ledger.byRevision[1] != 5120 || len(ledger.gaps) != 0 {
		t.Fatalf("expiry lost or recounted usage: %+v", ledger)
	}
	r20Crash(r.eng)
	nextEngine := r20Successor(t, r, r.eng.cfg, src, r.plane)
	if err := nextEngine.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	ledger = r21ReadLedger(t, nextEngine.cfg, nextEngine.nodeID, nextEngine.incID)
	if ledger.byRevision[1] != 5120 || len(ledger.gaps) != 0 {
		t.Fatalf("restart after expiry: %+v", ledger)
	}
}

func TestAT13_F42_ReportingRefreshesExistingCause(t *testing.T) {
	r := f42Siblings(t, false)
	r.eng.serving = false
	r.eng.usagePersist = true // failure after the cached serving decision
	r.eng.beat()
	rep, err := r.eng.report(t.Context(), r.clock.Wall())
	if err != nil {
		t.Fatal(err)
	}
	if mustBeat(t, r).Reason != "meter_storage_failed" || !hasReason(rep.ForwardingBlockReasons, api.ForwardingBlockReasonMeterStorageFailed) {
		t.Fatal("missing known cause", mustBeat(t, r), rep.ForwardingBlockReasons)
	}
	r.eng.decision = Decision{} // before the first evaluation, no decision yet
	r.eng.usagePersist = false
	r.eng.beat()
	if mustBeat(t, r).Reason != "-" {
		t.Fatal("zero decision invented a cause")
	}
}

func TestAT13_F42_WholeNodeDeadlineKeepsEveryCap(t *testing.T) {
	for _, limit := range []string{"lease", "desired", "offline", "member", "service", "binding"} {
		t.Run(limit, func(t *testing.T) {
			in := withAdmittedMember(servingInput())
			start := in.Lease.IssuedAt
			switch limit {
			case "lease":
				in.Lease.ExpiresAt = start.Add(2 * time.Second)
			case "desired":
				in.Desired.ExpiresAt = start.Add(2 * time.Second)
			case "offline":
				in.Lease.MaxOfflineSeconds = 2
			case "member":
				in.Lease.Members[0].LeaseEnd = start.Add(2 * time.Second)
			case "service":
				in.Lease.Members[0].EffectiveServiceEnd = start.Add(2 * time.Second)
			case "binding":
				in.Desired.DeviceBindings[0].ValidUntil = start.Add(2 * time.Second)
			}
			d := Decide(in)
			if !d.Serve {
				t.Fatal(d.Reasons)
			}
			e := &Engine{serving: true, policyExpires: d.PolicyExpires, appliedModel: &d.Model}
			if e.servingDeadline() != in.LeaseMonoSent+2*time.Second {
				t.Fatal("node cap lost", e.servingDeadline())
			}
			in.MonoNow = in.LeaseMonoSent + 2*time.Second
			in.Now = start.Add(2 * time.Second)
			if d := Decide(in); d.Serve {
				t.Fatal("admitted at cap", limit)
			}
		})
	}
}

func TestMTR03_F42_ExpiredSettlementFailureKeepsIntentAndCloses(t *testing.T) {
	r := f42Siblings(t, false)
	intent := r.eng.restartIntent
	before := r.plane.applies
	// A closed sampler cannot checkpoint outgoing bytes. Keep the provenance
	// pending; no surviving Apply is allowed to replace it on this step.
	r.eng.meter.Close()
	r.clock.Advance(r.eng.policyExpires - r.clock.Mono())
	d := Decide(r.eng.input())
	if !d.Serve || !r.eng.onlyExpiredBindingsRemoved(d.Model) {
		t.Fatal("not expired-only removal")
	}
	if err := r.eng.enforce(t.Context(), d); err != nil {
		t.Fatal(err)
	}
	r.eng.beat()
	rep, err := r.eng.report(t.Context(), r.clock.Wall())
	if err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || r.plane.applies != before || r.eng.restartIntent != intent || !r.eng.retainedPending || mustBeat(t, r).Reason == "-" || len(rep.ForwardingBlockReasons) == 0 {
		t.Fatal("failed settlement admitted or lost its cause")
	}
}
