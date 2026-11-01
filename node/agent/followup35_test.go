package agent

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/google/uuid"
)

func TestAT16_F35_CtdenydResetAndReportConsumers(t *testing.T) {
	diagnostic := enforce.CounterValue{Packets: 7, Bytes: 700}
	only := map[string]enforce.CounterValue{"ctdenyd": diagnostic}
	if usageName("ctdenyd") || len(usageOnly(only)) != 0 || liveUsage(only) || !reflect.DeepEqual(nudgeOne(only), only) {
		t.Fatal("diagnostic matched a reset/epoch selector")
	}
	dev := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	up := meter.CounterName(dev, meter.DirUpload)
	for _, member := range []bool{false, true} {
		for _, change := range []string{"increase", "decrease", "disappear"} {
			name := "diagnostic_only/" + change
			if member {
				name = "with_member/" + change
			}
			t.Run(name, func(t *testing.T) {
				prev := cloneCounters(only)
				if member {
					prev[up] = enforce.CounterValue{Packets: 10, Bytes: 1000}
				}
				cur := enforce.State{Table: true, Chains: map[string]bool{"forward": true}, Generation: 1, Counters: cloneCounters(prev)}
				ks := &kernelSource{boot: "same-boot", expect: func() bool { return true }, read: func(context.Context) (enforce.State, error) { return cur, nil }}
				if _, err := ks.Read(t.Context()); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "increase":
					cur.Counters["ctdenyd"] = enforce.CounterValue{Packets: 99, Bytes: 9007199254740993}
				case "decrease":
					cur.Counters["ctdenyd"] = enforce.CounterValue{Packets: 1, Bytes: 1}
				case "disappear":
					delete(cur.Counters, "ctdenyd")
				}
				if restore, force, report := classifyExternal(prev, cur, true); restore || force || report {
					t.Fatalf("diagnostic caused reset: restore=%v force=%v report=%v", restore, force, report)
				}
				obs, err := ks.Read(t.Context())
				if err != nil || !reflect.DeepEqual(obs.Counters, cur.Counters) {
					t.Fatalf("diagnostic nudged the kernel observation: %+v %v", obs, err)
				}
				// Apply's raw reset evidence includes diagnostic drops. Its two
				// agent consumers must suppress a member epoch/error report.
				resets := enforce.DetectMeterResets(prev, cur)
				if change != "increase" && (len(resets) != 1 || resets[0].Name != "ctdenyd") {
					t.Fatalf("missing positive diagnostic reset evidence: %+v", resets)
				}
				ks.noteApply(prev, resets, cur.Counters, true)
				e := &Engine{kernel: ks, clock: systemClock{}, log: slog.New(slog.DiscardHandler)}
				e.publishMeterEpoch()
				e.noteResetsWithoutKernel(prev, resets, cur.Counters, true)
				if ks.nudgePending() || len(e.decision.Errors) != 0 {
					t.Fatalf("diagnostic became a meter epoch report: %+v", e.decision.Errors)
				}
				if member {
					cur.Counters[up] = enforce.CounterValue{Packets: 1, Bytes: 1}
					if _, _, report := classifyExternal(prev, cur, true); !report {
						t.Fatal("real member reset was suppressed")
					}
				}
			})
		}
	}
}

func TestAT14_F35_CtdenydCounterSamples(t *testing.T) {
	dev, member := uuid.New(), uuid.New()
	m := enforce.Model{Bindings: []enforce.Binding{{DeviceID: dev, MemberID: member, BindingRevision: 1}}}
	counters := map[string]enforce.CounterValue{
		"ctdenyd":                                 {Packets: 99, Bytes: 9007199254740993},
		meter.CounterName(dev, meter.DirUpload):   {Packets: 1, Bytes: 123},
		meter.CounterName(dev, meter.DirDownload): {Packets: 2, Bytes: 456},
	}
	// This is also the join used by the netlab counter-report command.
	samples, err := enforce.Samples(m, counters)
	if err != nil || len(samples) != 2 {
		t.Fatalf("counter report samples: %+v %v", samples, err)
	}
	for _, sample := range samples {
		want := counters[meter.CounterName(dev, sample.Direction)]
		if sample.Namespace != enforce.MeteringVersion || sample.DeviceID != dev || sample.MemberID != member || sample.Bytes != want.Bytes || sample.Packets != want.Packets {
			t.Fatalf("diagnostic contaminated sample: %+v", sample)
		}
	}
}

type f35CounterPlane struct {
	*memPlane
	counters map[string]enforce.CounterValue
}

func (p *f35CounterPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.memPlane.Apply(ctx, m, prev)
	res.Counters = cloneCounters(p.counters)
	res.Resets = enforce.DetectResets(prev, res.Counters)
	return res, err
}

func TestAT16_F35_CtdenydApplyPersistenceDoesNotBecomeUsage(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	p := &f35CounterPlane{memPlane: r.plane, counters: map[string]enforce.CounterValue{"ctdenyd": {Packets: 7, Bytes: 700}}}
	r.eng.plane = p
	// Exercise the real kernel-source classification and meter together,
	// with an in-memory nft read. No host networking command is executed.
	ks := &kernelSource{
		boot: r.eng.bootID.String(), bindings: r.eng.meterBindings,
		expect: func() bool { return r.eng.expectServing.Load() },
		read: func(context.Context) (enforce.State, error) {
			return enforce.State{Table: true, Chains: map[string]bool{"forward": true}, Generation: 1, Counters: cloneCounters(p.counters)}, nil
		},
	}
	r.eng.meterSource, r.eng.kernel = ks, ks
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	initial, err := r.eng.usage.Head(t.Context())
	if err != nil || initial.NextSeq != 1 || len(initial.Counters) != 0 {
		t.Fatalf("diagnostic created initial usage: %+v %v", initial, err)
	}
	initialSnapshot, err := usagestore.ReadSnapshot(t.Context(), r.eng.cfg.UsageDBPath())
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []uint64{700, 1, 0} {
		p.counters = map[string]enforce.CounterValue{"ctdenyd": {Packets: value, Bytes: value}}
		r.clock.Advance(time.Second)
		stepOK(t, r)
		// Raw enforcement readings are deliberately persisted in agentstore;
		// this separate connection proves that exclusion is in the consumers.
		st, err := agentstore.Open(r.eng.cfg.DBPath())
		if err != nil {
			t.Fatal(err)
		}
		raw, readErr := st.Counters(t.Context())
		closeErr := st.Close()
		if readErr != nil || closeErr != nil || raw["ctdenyd"].Bytes != value {
			t.Fatalf("raw Apply snapshot: %+v %v %v", raw, readErr, closeErr)
		}
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}
		head, err := r.eng.usage.Head(t.Context())
		if err != nil || head.OpenEpoch != initial.OpenEpoch || head.NextSeq != 1 || len(head.Counters) != 0 {
			t.Fatalf("raw diagnostic reached usage ledger: %+v %v", head, err)
		}
		snapshot, err := usagestore.ReadSnapshot(t.Context(), r.eng.cfg.UsageDBPath())
		if err != nil || !reflect.DeepEqual(snapshot.MeterEpochs, initialSnapshot.MeterEpochs) || snapshot.LastEventSeq != "0" {
			t.Fatalf("diagnostic reached usage report: %+v %v", snapshot, err)
		}
		if rep := mustReport(t, r, t.Context()); len(rep.Errors) != 0 || !rep.MemberForwardingEnabled {
			t.Fatalf("diagnostic changed observation: %+v", rep)
		}
	}
}
