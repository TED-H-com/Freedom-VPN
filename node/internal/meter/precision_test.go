package meter

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func assertMicrosecondInterval(t *testing.T, start, end time.Time) {
	t.Helper()
	for _, endpoint := range []time.Time{start, end} {
		if endpoint.Location() != time.UTC || endpoint.Nanosecond()%1000 != 0 {
			t.Fatalf("endpoint is not a UTC whole microsecond: %s", endpoint)
		}
	}
	if end.Sub(start) < time.Microsecond {
		t.Fatalf("interval collapses at central precision: [%s, %s)", start, end)
	}
}

func TestD130_AllIntervalPaths(t *testing.T) {
	// Supply an offset clock with sub-microsecond precision to all four paths.
	now := time.Date(2026, 10, 31, 14, 1, 15, 198144331, time.UTC).In(time.FixedZone("TEST ONLY", 3600))
	last := now.UTC().Truncate(time.Microsecond)
	for _, tc := range []struct {
		name    string
		path    string
		last    time.Time
		now     time.Time
		opening bool
	}{
		{"baseline/fresh", "baseline", time.Time{}, now, false},
		{"baseline/history", "baseline", last.Add(-time.Second), now, false},
		{"baseline/rollback", "baseline", last, now.Add(-time.Hour), false},
		{"sample/no_checkpoint", "sample", time.Time{}, now, false},
		{"sample/normal", "sample", last.Add(-time.Second), now, false},
		{"sample/same_microsecond", "sample", last, now, false},
		{"clock_gap", "clock_gap", last, now.Add(-time.Hour), false},
		{"switch/normal", "switch", last.Add(-time.Second), now, false},
		{"switch/same_microsecond", "switch", last, now, false},
		{"switch/equal", "switch", last, last, false},
		{"switch/rollback", "switch", last, now.Add(-time.Hour), false},
		{"switch/restart_rollback", "switch", last, now.Add(-time.Hour), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			head := usagestore.Head{
				OpenEpoch: uuid.New(), BootID: "boot", LastSample: tc.last,
				HasHistory: !tc.last.IsZero(),
				Counters:   usageCounters(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 10, upP: 1})),
			}
			obs := observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 20, upP: 2})
			var p plan
			var err error
			switch tc.path {
			case "baseline":
				head.OpenEpoch = uuid.Nil
				p = planBaseline(head, usageCounters(obs), tc.now, obs.BootID)
			case "sample", "clock_gap":
				p, err = planSample(head, obs, tc.now, DefaultSampleInterval, false)
			case "switch":
				obs.BootID = "new-boot"
				if tc.opening {
					p, err = planContinuity(head, obs, tc.now)
				} else {
					p, err = planSample(head, obs, tc.now, DefaultSampleInterval, false)
				}
			}
			if err != nil || p.hold || p.noop || p.commit.Event == nil {
				t.Fatalf("missing interval: %+v, %v", p, err)
			}
			d := p.commit.Event
			assertMicrosecondInterval(t, d.Start, d.End)
			if !tc.last.IsZero() && !d.Start.Equal(tc.last) {
				t.Fatalf("previous end %s != start %s", tc.last, d.Start)
			}
			if !p.commit.SampledAt.Equal(d.End) {
				t.Fatalf("checkpoint %s != event end %s", p.commit.SampledAt, d.End)
			}
			// Follow the commit's cursor into a counted sample on the same stream,
			// including when the preceding gap opened a different meter epoch.
			head.LastSample = p.commit.SampledAt
			head.Counters = p.commit.Counters
			head.BootID = p.commit.BootID
			if p.commit.Open != nil {
				head.OpenEpoch = p.commit.Open.ID
			}
			next, err := planSample(head,
				observe(head.BootID, reading{dev: devA, mem: memA, rev: 1, upB: 21, upP: 3}),
				d.End.Add(DefaultSampleInterval+331*time.Nanosecond), DefaultSampleInterval, false)
			if err != nil || next.commit.Event == nil {
				t.Fatalf("next sample: %+v, %v", next, err)
			}
			n := next.commit.Event
			assertMicrosecondInterval(t, n.Start, n.End)
			if !n.Start.Equal(d.End) || !next.commit.SampledAt.Equal(n.End) {
				t.Fatalf("noncontiguous interval/checkpoint: previous %+v, next %+v", d, next.commit)
			}
			if len(n.Records) != 1 || n.Records[0].Upload != 1 {
				t.Fatalf("next delta: %+v", n.Records)
			}
		})
	}
}

func TestENR07_FirstGapSurvivesMicrosecondPersistence(t *testing.T) {
	// The proof's start is ...331; the old now-minus-1ns code used ...332
	// as its clock/end. Both exact values must produce a persistable gap.
	for _, ns := range []int{198144331, 198144332} {
		now := time.Date(2026, 10, 31, 14, 1, 15, ns, time.UTC)
		t.Run(now.Format(time.RFC3339Nano), func(t *testing.T) {
			clk := &clock{t: now}
			src := &fakeSrc{}
			src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 100, upP: 1}))
			m, s := start(t, src, clk, usagestore.Options{})
			gap := loadEvent(t, s, 1)
			assertMicrosecondInterval(t, gap.IntervalStart, gap.IntervalEnd)
			wantEnd := time.Date(2026, 10, 31, 14, 1, 15, 198144000, time.UTC)
			if !gap.IntervalStart.Equal(wantEnd.Add(-time.Microsecond)) || !gap.IntervalEnd.Equal(wantEnd) {
				t.Fatalf("gap endpoints: %s -> %s", gap.IntervalStart, gap.IntervalEnd)
			}
			if !gap.IntervalStart.Truncate(time.Microsecond).Before(gap.IntervalEnd.Truncate(time.Microsecond)) {
				t.Fatal("central microsecond projection collapses the interval")
			}
			if gap.Quality != api.UsageQualityGap || gap.Gap == nil || gap.Gap.Reason != api.UsageGapReasonUnknown || len(gap.Records) != 0 {
				t.Fatalf("gap semantics changed: %+v", gap)
			}
			head, err := s.Head(t.Context())
			if err != nil || !head.LastSample.Equal(gap.IntervalEnd) {
				t.Fatalf("durable checkpoint: %+v, %v", head, err)
			}
			clk.Set(now.Add(DefaultSampleInterval))
			src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 650, upP: 2, dnB: 358, dnP: 1}))
			if err := m.Sample(t.Context()); err != nil {
				t.Fatal(err)
			}
			next := loadEvent(t, s, 2)
			assertMicrosecondInterval(t, next.IntervalStart, next.IntervalEnd)
			if !next.IntervalStart.Equal(gap.IntervalEnd) || next.EventSeq != 2 || next.Quality != api.UsageQualityComplete || len(next.Records) != 1 {
				t.Fatalf("next event: %+v", next)
			}
			r := next.Records[0]
			if r.UploadBytes != 550 || r.DownloadBytes != 358 || r.MemberId != memA || r.DeviceId != devA || r.PeriodId != "2026-10" {
				t.Fatalf("908 bytes attributed incorrectly: %+v", r)
			}
			head, err = s.Head(t.Context())
			if err != nil || !head.LastSample.Equal(next.IntervalEnd) {
				t.Fatalf("next durable checkpoint: %+v, %v", head, err)
			}
		})
	}
}

func TestD130_QuietCheckpointsUseMicroseconds(t *testing.T) {
	clk := &clock{t: time.Date(2026, 10, 31, 14, 1, 15, 198144331, time.UTC)}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
	m, s := start(t, src, clk, usagestore.Options{})
	for i := 0; i < 2; i++ {
		head, err := s.Head(t.Context())
		if err != nil || head.NextSeq != 1 || !head.LastSample.Equal(clk.Now().Truncate(time.Microsecond)) {
			t.Fatalf("quiet checkpoint: %+v, %v", head, err)
		}
		clk.Set(clk.Now().Add(DefaultSampleInterval))
		if err := m.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	head, err := s.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	clk.Set(clk.Now().Add(DefaultSampleInterval))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 20, upP: 1}))
	if err := m.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	ev := loadEvent(t, s, 1)
	assertMicrosecondInterval(t, ev.IntervalStart, ev.IntervalEnd)
	if !ev.IntervalStart.Equal(head.LastSample) {
		t.Fatalf("quiet checkpoint %s != next start %s", head.LastSample, ev.IntervalStart)
	}
}

func TestD130_QuietReopenPreservesCheckpoint(t *testing.T) {
	last := time.Date(2026, 10, 31, 14, 1, 15, 198144000, time.UTC)
	for _, now := range []time.Time{last, last.Add(331 * time.Nanosecond), last.Add(-time.Hour)} {
		t.Run(now.Format(time.RFC3339Nano), func(t *testing.T) {
			head := usagestore.Head{HasHistory: true, LastCloseReason: "final_sample", LastSample: last}
			p := planBaseline(head, nil, now, "boot")
			if p.commit.Event != nil || !p.commit.SampledAt.Equal(last) {
				t.Fatalf("quiet reopen moved checkpoint: %+v", p.commit)
			}
		})
	}
}

func TestD130_LegacySealedGapUnchanged(t *testing.T) {
	clk := &clock{t: time.Date(2026, 10, 31, 14, 1, 15, 198144331, time.UTC)}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
	m, s := start(t, src, clk, usagestore.Options{})
	m.Close()
	head, err := s.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// Seal the measured legacy 1ns interval through the unchanged store API.
	end := clk.Now().Add(time.Nanosecond)
	if _, err := s.Commit(t.Context(), usagestore.Commit{
		Epoch: head.OpenEpoch, Counters: head.Counters, BootID: head.BootID, SampledAt: end,
		Event: &usagestore.Draft{
			EpochID: head.OpenEpoch, Start: clk.Now(), End: end,
			Quality: string(api.UsageQualityGap), GapReason: string(api.UsageGapReasonUnknown),
		},
	}); err != nil {
		t.Fatal(err)
	}
	before, err := s.Event(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := usagestore.Open(s.Path(), nodeID, incID, usagestore.Options{Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted, err := New(Config{NodeID: nodeID, IncarnationID: incID, Store: reopened, Source: src, Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	clk.Set(clk.Now().Add(DefaultSampleInterval))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 20, upP: 1}))
	if err := restarted.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	after, err := reopened.Event(t.Context(), 1)
	if err != nil || !bytes.Equal(before.Payload, after.Payload) || before.Hash != after.Hash {
		t.Fatalf("legacy payload/hash changed: %v", err)
	}
	next := loadEvent(t, reopened, 2)
	assertMicrosecondInterval(t, next.IntervalStart, next.IntervalEnd)
	if !next.IntervalStart.Equal(end.Truncate(time.Microsecond)) {
		t.Fatalf("legacy checkpoint projection: %s", next.IntervalStart)
	}
	head, err = reopened.Head(t.Context())
	if err != nil || head.NextSeq != 3 || head.AckedThrough != 0 || !head.LastSample.Equal(next.IntervalEnd) {
		t.Fatalf("legacy sequence was skipped or cursor differs: %+v, %v", head, err)
	}
}
