package meter

import (
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestAT14_F35_CtdenydUsageSelection(t *testing.T) {
	if _, _, ok := parseUsageCounter("ctdenyd"); ok {
		t.Fatal("diagnostic parsed as a device counter")
	}
	obs := observe("boot", reading{dev: devA, mem: memA, rev: 7, upB: 100, dnB: 200})
	want := usageCounters(obs)
	if len(want) != 2 {
		t.Fatal("missing positive usage controls")
	}
	obs.Counters["ctdenyd"] = enforce.CounterValue{Packets: 99, Bytes: 9007199254740993}
	if got := usageCounters(obs); !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostic altered selected usage: %+v", got)
	}
	if got := usageCounters(Observation{Counters: map[string]enforce.CounterValue{"ctdenyd": obs.Counters["ctdenyd"]}}); len(got) != 0 {
		t.Fatalf("diagnostic-only usage: %+v", got)
	}
}

func TestAT16_F35_CtdenydDoesNotChangePersistedUsageOrEpoch(t *testing.T) {
	for _, withMember := range []bool{false, true} {
		name := "diagnostic_only"
		if withMember {
			name = "with_member"
		}
		t.Run(name, func(t *testing.T) {
			ctx := t.Context()
			clk := &clock{t: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
			obs := observe("same-boot")
			if withMember {
				obs = observe("same-boot", reading{dev: devA, mem: memA, rev: 7})
			}
			obs.Counters["ctdenyd"] = enforce.CounterValue{Packets: 9, Bytes: 900}
			src := &fakeSrc{obs: obs}
			m, st := start(t, src, clk, usagestore.Options{})
			initial, err := st.Head(ctx)
			if err != nil || initial.NextSeq != 1 {
				t.Fatalf("nonzero diagnostic created an initial gap: %+v %v", initial, err)
			}
			// Meter.New always opens the ordinary baseline epoch. None of these
			// diagnostic changes may add an epoch, gap, event, or stored reading.
			for _, change := range []string{"increase", "decrease", "disappear", "reappear", "restart"} {
				clk.Set(clk.Now().Add(10 * time.Second))
				switch change {
				case "increase":
					obs.Counters["ctdenyd"] = enforce.CounterValue{Packets: 99, Bytes: 9007199254740993}
				case "decrease":
					obs.Counters["ctdenyd"] = enforce.CounterValue{Packets: 1, Bytes: 1}
				case "disappear":
					delete(obs.Counters, "ctdenyd")
				case "reappear":
					obs.Counters["ctdenyd"] = enforce.CounterValue{Packets: 7, Bytes: 700}
				case "restart":
					m.Close() // no final sample: same-boot process continuity
					path := st.Path()
					if err := st.Close(); err != nil {
						t.Fatal(err)
					}
					st, err = usagestore.Open(path, nodeID, incID, usagestore.Options{Now: clk.Now})
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = st.Close() })
					delete(obs.Counters, "ctdenyd")
					src.set(obs)
					m, err = New(Config{NodeID: nodeID, IncarnationID: incID, Store: st, Source: src, Now: clk.Now})
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(m.Close)
				}
				src.set(obs)
				if err := m.Sample(ctx); err != nil {
					t.Fatalf("%s: %v", change, err)
				}
				head, err := st.Head(ctx)
				if err != nil || head.OpenEpoch != initial.OpenEpoch || head.NextSeq != 1 || !reflect.DeepEqual(head.Counters, initial.Counters) {
					t.Fatalf("%s changed durable usage: %+v %v", change, head, err)
				}
				for _, c := range head.Counters {
					if c.Name == "ctdenyd" {
						t.Fatal("diagnostic persisted as a usage reading")
					}
				}
				// This independent read is the production `usage snapshot` consumer.
				snap, err := usagestore.ReadSnapshot(ctx, st.Path())
				if err != nil || len(snap.MeterEpochs) != 1 || snap.LastEventSeq != "0" || len(snap.UnackedEvents) != 0 {
					t.Fatalf("%s invented snapshot usage/epoch: %+v %v", change, snap, err)
				}
			}
			if withMember {
				clk.Set(clk.Now().Add(10 * time.Second))
				obs.Counters[CounterName(devA, DirUpload)] = enforce.CounterValue{Packets: 1, Bytes: 123}
				obs.Counters[CounterName(devA, DirDownload)] = enforce.CounterValue{Packets: 2, Bytes: 456}
				obs.Counters["ctdenyd"] = enforce.CounterValue{Packets: 99, Bytes: 99999}
				src.set(obs)
				if err := m.Sample(ctx); err != nil {
					t.Fatal(err)
				}
				ev := loadEvent(t, st, 1)
				if ev.Quality != api.UsageQualityComplete || ev.MeterEpochId != initial.OpenEpoch || len(ev.Records) != 1 || ev.Records[0].UploadBytes != 123 || ev.Records[0].DownloadBytes != 456 {
					t.Fatalf("member control lost or mixed with diagnostic: %+v", ev)
				}
			}
		})
	}
}
