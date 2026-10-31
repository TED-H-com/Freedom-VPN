package meter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

var (
	nodeID = uuid.MustParse("93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb")
	incID  = uuid.MustParse("d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0")
	devA   = uuid.MustParse("11111111-1111-4111-8111-111111111111")
	devB   = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	memA   = uuid.MustParse("dc01a74b-9e6f-41ac-81cd-03291d0a88c0")
	memB   = uuid.MustParse("4e8ca24f-b523-45ef-a782-068cc1c9f809")
)

type clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *clock) Set(t time.Time) {
	c.mu.Lock()
	c.t = t
	c.mu.Unlock()
}

type fakeSrc struct {
	mu    sync.Mutex
	obs   Observation
	err   error
	block <-chan struct{}
	hit   chan struct{}
}

func (f *fakeSrc) Read(ctx context.Context) (Observation, error) {
	if f.hit != nil {
		select {
		case f.hit <- struct{}{}:
		default:
		}
	}
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return Observation{}, ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.obs, f.err
}

func (f *fakeSrc) set(obs Observation) {
	f.mu.Lock()
	f.obs = obs
	f.err = nil
	f.mu.Unlock()
}

type reading struct {
	dev, mem uuid.UUID
	rev      uint64
	upB, upP uint64
	dnB, dnP uint64
}

func observe(boot string, rs ...reading) Observation {
	obs := Observation{BootID: boot, Counters: map[string]enforce.CounterValue{}}
	for _, r := range rs {
		obs.Bindings = append(obs.Bindings, enforce.Binding{
			MemberID: r.mem, DeviceID: r.dev, BindingRevision: r.rev,
		})
		obs.Counters[CounterName(r.dev, DirUpload)] = enforce.CounterValue{Packets: r.upP, Bytes: r.upB}
		obs.Counters[CounterName(r.dev, DirDownload)] = enforce.CounterValue{Packets: r.dnP, Bytes: r.dnB}
	}
	return obs
}

func start(t *testing.T, src *fakeSrc, clk *clock, opt usagestore.Options) (*Meter, *usagestore.Store) {
	t.Helper()
	opt.Now = clk.Now
	s, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), nodeID, incID, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	m, err := New(Config{
		NodeID: nodeID, IncarnationID: incID, Store: s, Source: src,
		Interval: 10 * time.Second, Now: clk.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	return m, s
}

func loadEvent(t *testing.T, s *usagestore.Store, seq uint64) api.UsageEvent {
	t.Helper()
	ev, err := s.Event(context.Background(), seq)
	if err != nil {
		t.Fatal(err)
	}
	var out api.UsageEvent
	if err := json.Unmarshal(ev.Payload, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDeltaResetAndContinuity(t *testing.T) {
	ctx := context.Background()
	clk := &clock{t: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)}
	src := &fakeSrc{}
	base := observe("boot-1",
		reading{dev: devA, mem: memA, rev: 7},
		reading{dev: devB, mem: memB, rev: 8},
	)
	src.set(base)
	m, s := start(t, src, clk, usagestore.Options{})
	head, err := s.Head(ctx)
	if err != nil || head.OpenEpoch == uuid.Nil || head.NextSeq != 1 || head.BootID != "boot-1" {
		t.Fatalf("%+v %v", head, err)
	}
	epoch := head.OpenEpoch

	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot-1",
		reading{dev: devA, mem: memA, rev: 7, upB: 1000, upP: 4, dnB: 50, dnP: 2},
		reading{dev: devB, mem: memB, rev: 8, upB: 9007199254740993, upP: 9},
	))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	ev := loadEvent(t, s, 1)
	if ev.Quality != api.UsageQualityComplete || ev.MeterEpochId != epoch || len(ev.Records) != 2 {
		t.Fatalf("%+v", ev)
	}
	if ev.Records[0].DeviceId != devA || uint64(ev.Records[0].UploadBytes) != 1000 || uint64(ev.Records[0].DownloadBytes) != 50 {
		t.Fatalf("record A %+v", ev.Records[0])
	}
	if ev.Records[1].DeviceId != devB || uint64(ev.Records[1].UploadBytes) != 9007199254740993 || ev.Records[1].BindingRevision.String() != "8" {
		t.Fatalf("record B %+v", ev.Records[1])
	}
	if ev.Records[0].PeriodId != "2026-10" || ev.Records[0].BindingRevision.String() != "7" {
		t.Fatalf("period %+v", ev.Records[0])
	}
	if !strings.Contains(string(mustPayload(t, s, 1)), `"upload_bytes":"9007199254740993"`) {
		t.Fatal("decimal rendered as a float")
	}

	// Quiet interval: no second event.
	clk.Set(clk.Now().Add(10 * time.Second))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Event(ctx, 2); err == nil {
		t.Fatal("quiet interval invented an event")
	}

	// DNS and diagnostic counters are not member usage (D-019).
	clk.Set(clk.Now().Add(10 * time.Second))
	obs := observe("boot-1",
		reading{dev: devA, mem: memA, rev: 7, upB: 1000, upP: 4, dnB: 50, dnP: 2},
		reading{dev: devB, mem: memB, rev: 8, upB: 9007199254740993, upP: 9},
	)
	dns := "nu_" + strings.ReplaceAll(devA.String(), "-", "")
	obs.Counters[dns] = enforce.CounterValue{Packets: 10, Bytes: 99999}
	obs.Counters["ct_diag"] = enforce.CounterValue{Packets: 1, Bytes: 5}
	src.set(obs)
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Event(ctx, 2); err == nil {
		t.Fatal("DNS bytes became member usage")
	}

	// Backwards, including a wrap to a small value, opens a meter epoch.
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot-1",
		reading{dev: devA, mem: memA, rev: 7, upB: 1000, upP: 4, dnB: 50, dnP: 2},
		reading{dev: devB, mem: memB, rev: 8, upB: 1, upP: 1},
	))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	gap := loadEvent(t, s, 2)
	if gap.Quality != api.UsageQualityGap || gap.Gap == nil || gap.Gap.Reason != api.UsageGapReasonCounterDiscontinuity {
		t.Fatalf("%+v", gap)
	}
	for _, r := range gap.Records {
		if r.DeviceId == devB {
			t.Fatalf("reset device was billed: %+v", r)
		}
	}
	head, err = s.Head(ctx)
	if err != nil || head.OpenEpoch == epoch || head.OpenEpoch == uuid.Nil {
		t.Fatalf("epoch did not change: %+v", head)
	}
	epoch = head.OpenEpoch

	// Missing counter is the same kind of epoch change.
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot-1", reading{dev: devA, mem: memA, rev: 7, upB: 1100, upP: 5, dnB: 50, dnP: 2}))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	miss := loadEvent(t, s, 3)
	if miss.Quality != api.UsageQualityGap {
		t.Fatalf("%+v", miss)
	}
	head, _ = s.Head(ctx)
	if head.OpenEpoch == epoch {
		t.Fatal("missing counter kept the epoch")
	}

	// Short restart with the same boot and intact counters continues.
	cont := head.OpenEpoch
	m.Close()
	_ = s.Close()
	path := s.Path()
	s2, err := usagestore.Open(path, nodeID, incID, usagestore.Options{Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	cur, _ := s2.Head(ctx)
	src.set(countersFrom(cur, "boot-1"))
	m2, err := New(Config{NodeID: nodeID, IncarnationID: incID, Store: s2, Source: src, Interval: 10 * time.Second, Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer m2.Close()
	again, _ := s2.Head(ctx)
	if again.OpenEpoch != cont || again.NextSeq != cur.NextSeq {
		t.Fatalf("restart broke continuity: %+v", again)
	}

	// A new boot id does not continue, and event_seq is not reset.
	m2.Close()
	clk.Set(clk.Now().Add(time.Minute))
	src.set(observe("boot-2", reading{dev: devA, mem: memA, rev: 7, upB: 1, upP: 1, dnB: 1, dnP: 1}))
	m3, err := New(Config{NodeID: nodeID, IncarnationID: incID, Store: s2, Source: src, Interval: 10 * time.Second, Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer m3.Close()
	booted, _ := s2.Head(ctx)
	if booted.OpenEpoch == cont || booted.BootID != "boot-2" {
		t.Fatalf("%+v", booted)
	}
	power := loadEvent(t, s2, again.NextSeq)
	if power.Quality != api.UsageQualityGap || power.Gap.Reason != api.UsageGapReasonPowerLoss {
		t.Fatalf("%+v", power)
	}
	if power.EventSeq.String() == "1" && again.NextSeq != 1 {
		t.Fatal("event_seq restarted")
	}
}

func countersFrom(h usagestore.Head, boot string) Observation {
	obs := Observation{BootID: boot, Counters: map[string]enforce.CounterValue{}}
	seen := map[uuid.UUID]bool{}
	for _, c := range h.Counters {
		obs.Counters[c.Name] = enforce.CounterValue{Packets: c.Packets, Bytes: c.Bytes}
		if seen[c.DeviceID] {
			continue
		}
		seen[c.DeviceID] = true
		obs.Bindings = append(obs.Bindings, enforce.Binding{
			MemberID: c.MemberID, DeviceID: c.DeviceID, BindingRevision: c.BindingRevision,
		})
	}
	return obs
}

func mustPayload(t *testing.T, s *usagestore.Store, seq uint64) []byte {
	t.Helper()
	ev, err := s.Event(context.Background(), seq)
	if err != nil {
		t.Fatal(err)
	}
	return ev.Payload
}

func TestMappingConflictAndWrap(t *testing.T) {
	ctx := context.Background()
	clk := &clock{t: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 3, upB: 0, upP: 0}))
	m, s := start(t, src, clk, usagestore.Options{})
	head, _ := s.Head(ctx)
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memB, rev: 4, upB: 80, upP: 2}))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	ev := loadEvent(t, s, 1)
	if ev.Quality != api.UsageQualityGap || len(ev.Records) != 0 || ev.MeterEpochId != head.OpenEpoch {
		t.Fatalf("mapping %+v", ev)
	}
	// Wrap of a maxed counter is an epoch change, not a negative or a huge delta.
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memB, rev: 4, upB: math.MaxUint64, upP: math.MaxUint64}))
	// The mapping sample checkpointed 80. Jumping to max is a normal increase.
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memB, rev: 4, upB: 2, upP: 1}))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	wrap := loadEvent(t, s, 3)
	if wrap.Quality != api.UsageQualityGap {
		t.Fatalf("wrap quality %s", wrap.Quality)
	}
	for _, r := range wrap.Records {
		if uint64(r.UploadBytes) > 1000 {
			t.Fatalf("wrap billed %d", uint64(r.UploadBytes))
		}
	}
}

func TestAT17_MonthBoundary(t *testing.T) {
	ctx := context.Background()
	clk := &clock{t: time.Date(2026, 10, 31, 23, 59, 40, 0, time.UTC)}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
	m, s := start(t, src, clk, usagestore.Options{})

	clk.Set(time.Date(2026, 10, 31, 23, 59, 50, 0, time.UTC))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 100, upP: 1}))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	inside := loadEvent(t, s, 1)
	if inside.Quality != api.UsageQualityComplete || inside.Records[0].PeriodId != "2026-10" {
		t.Fatalf("inside month %+v", inside)
	}

	clk.Set(time.Date(2026, 11, 1, 0, 0, 5, 0, time.UTC))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 140, upP: 2}))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	boundary := loadEvent(t, s, 2)
	wantEnd := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if boundary.Quality != api.UsageQualityComplete || !boundary.IntervalEnd.Equal(wantEnd) {
		t.Fatalf("boundary sample %+v end %s", boundary.Quality, boundary.IntervalEnd)
	}
	if boundary.Records[0].PeriodId != "2026-10" || uint64(boundary.Records[0].UploadBytes) != 40 {
		t.Fatalf("boundary record %+v", boundary.Records[0])
	}
	if _, err := s.Event(ctx, 3); err == nil {
		t.Fatal("boundary sample was split")
	}

	t.Run("missed", func(t *testing.T) {
		clk := &clock{t: time.Date(2026, 10, 31, 23, 59, 50, 0, time.UTC)}
		src := &fakeSrc{}
		src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
		m, s := start(t, src, clk, usagestore.Options{})
		clk.Set(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
		src.mu.Lock()
		src.err = errors.New("nft read failed")
		src.mu.Unlock()
		if err := m.Sample(ctx); err == nil {
			t.Fatal("failed read succeeded")
		}
		h, err := m.Health(ctx)
		if err != nil || h.ForwardingAllowed || h.Reason != ReasonMeterPath {
			t.Fatalf("health %+v %v", h, err)
		}
		if _, err := s.Event(ctx, 1); err == nil {
			t.Fatal("failed boundary read wrote an event")
		}
		clk.Set(time.Date(2026, 11, 1, 1, 0, 0, 0, time.UTC))
		src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 80, upP: 4}))
		if err := m.Sample(ctx); err != nil {
			t.Fatal(err)
		}
		ev := loadEvent(t, s, 1)
		if ev.Quality != api.UsageQualityBoundaryEstimated || len(ev.Records) != 1 || uint64(ev.Records[0].UploadBytes) != 80 {
			t.Fatalf("%+v", ev)
		}
		if _, err := s.Event(ctx, 2); err == nil {
			t.Fatal("estimated interval was split on the node")
		}
		h, err = m.Health(ctx)
		if err != nil || !h.ForwardingAllowed {
			t.Fatalf("health after recovery %+v %v", h, err)
		}
	})
}

func TestClockRollback(t *testing.T) {
	ctx := context.Background()
	startAt := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	clk := &clock{t: startAt}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
	m, s := start(t, src, clk, usagestore.Options{})
	clk.Set(startAt.Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 30, upP: 1}))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	clk.Set(startAt.Add(9 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 30, upP: 1}))
	if err := m.Sample(ctx); !errors.Is(err, ErrHold) {
		t.Fatalf("err = %v", err)
	}
	if _, err := s.Event(ctx, 2); err == nil {
		t.Fatal("jitter wrote an event")
	}
	clk.Set(startAt.Add(-time.Hour))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	ev := loadEvent(t, s, 2)
	if ev.Quality != api.UsageQualityGap || ev.Gap.Reason != api.UsageGapReasonClockAnomaly {
		t.Fatalf("%+v", ev)
	}
	if !ev.IntervalEnd.After(ev.IntervalStart) || ev.IntervalEnd.Sub(ev.IntervalStart) != time.Microsecond {
		t.Fatalf("interval %s %s", ev.IntervalStart, ev.IntervalEnd)
	}
	if len(ev.Records) != 0 {
		t.Fatalf("clock gap claimed records %+v", ev.Records)
	}
}

func TestFinalSampleAndBarrier(t *testing.T) {
	ctx := context.Background()
	clk := &clock{t: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 2, upB: 5, upP: 1}))
	m, s := start(t, src, clk, usagestore.Options{})
	// Non-zero at first open is a gap, then the final sample closes.
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 2, upB: 25, upP: 3}))
	if err := m.FinalSample(ctx); err != nil {
		t.Fatal(err)
	}
	head, err := s.Head(ctx)
	if err != nil || head.OpenEpoch != uuid.Nil || head.LastCloseReason != "final_sample" {
		t.Fatalf("%+v %v", head, err)
	}
	finalEv := loadEvent(t, s, 2)
	if finalEv.Quality != api.UsageQualityComplete || uint64(finalEv.Records[0].UploadBytes) != 20 {
		t.Fatalf("final delta %+v", finalEv)
	}
	m.Close()
	_ = s.Close()
	s2, err := usagestore.Open(s.Path(), nodeID, incID, usagestore.Options{Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 2, upB: 25, upP: 3}))
	m2, err := New(Config{NodeID: nodeID, IncarnationID: incID, Store: s2, Source: src, Interval: 10 * time.Second, Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer m2.Close()
	if _, err := s2.Event(ctx, 3); err == nil {
		t.Fatal("clean handoff emitted a gap")
	}
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 2, upB: 40, upP: 4}))
	if err := m2.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	next := loadEvent(t, s2, 3)
	if uint64(next.Records[0].UploadBytes) != 15 {
		t.Fatalf("recounted from zero: %+v", next.Records[0])
	}

	block := make(chan struct{})
	src.block = block
	src.hit = make(chan struct{}, 1)
	clk.Set(clk.Now().Add(10 * time.Second))
	done := make(chan error, 1)
	go func() { done <- m2.Sample(ctx) }()
	select {
	case <-src.hit:
	case <-time.After(2 * time.Second):
		t.Fatal("sample did not start")
	}
	bdone := make(chan error, 1)
	go func() { bdone <- m2.Barrier(ctx) }()
	select {
	case err := <-bdone:
		t.Fatalf("barrier returned early: %v", err)
	case <-time.After(30 * time.Millisecond):
	}
	close(block)
	if err := <-bdone; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := m2.Sample(ctx); !errors.Is(err, ErrPaused) {
		t.Fatalf("after barrier: %v", err)
	}
	m2.ResumeSampling()
	src.block = nil
	clk.Set(clk.Now().Add(10 * time.Second))
	if err := m2.Sample(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAT16_MeterCrashSteps(t *testing.T) {
	ctx := context.Background()
	for _, step := range []string{StepRead, StepPlan, usagestore.StepInsert, usagestore.StepCheckpoint, usagestore.StepBeforeCommit} {
		t.Run(step, func(t *testing.T) {
			armed := ""
			clk := &clock{t: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)}
			src := &fakeSrc{}
			src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
			m, s := start(t, src, clk, usagestore.Options{Hook: func(got string) error {
				if armed != "" && got == armed {
					armed = ""
					return usagestore.ErrInjectedCrash
				}
				return nil
			}})
			clk.Set(clk.Now().Add(10 * time.Second))
			src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 12, upP: 1}))
			armed = step
			if err := m.Sample(ctx); !errors.Is(err, usagestore.ErrInjectedCrash) {
				t.Fatalf("%v", err)
			}
			if _, err := s.Event(ctx, 1); err == nil {
				t.Fatal("crash committed an event")
			}
			src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 12, upP: 1}))
			if err := m.Sample(ctx); err != nil {
				t.Fatal(err)
			}
			ev := loadEvent(t, s, 1)
			if uint64(ev.Records[0].UploadBytes) != 12 || ev.EventSeq.String() != "1" {
				t.Fatalf("%+v", ev)
			}
		})
	}
	t.Run(usagestore.StepAfterCommit, func(t *testing.T) {
		armed := ""
		clk := &clock{t: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)}
		src := &fakeSrc{}
		src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
		m, s := start(t, src, clk, usagestore.Options{Hook: func(got string) error {
			if armed != "" && got == armed {
				armed = ""
				return usagestore.ErrInjectedCrash
			}
			return nil
		}})
		clk.Set(clk.Now().Add(10 * time.Second))
		src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 12, upP: 1}))
		armed = usagestore.StepAfterCommit
		if err := m.Sample(ctx); !errors.Is(err, usagestore.ErrInjectedCrash) {
			t.Fatalf("%v", err)
		}
		ev := loadEvent(t, s, 1)
		saved := string(mustPayload(t, s, 1))
		clk.Set(clk.Now().Add(10 * time.Second))
		if err := m.Sample(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Event(ctx, 2); err == nil {
			t.Fatal("durable sample was duplicated")
		}
		if string(mustPayload(t, s, 1)) != saved {
			t.Fatal("payload rebuilt")
		}
		clk.Set(clk.Now().Add(10 * time.Second))
		src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 20, upP: 2}))
		if err := m.Sample(ctx); err != nil {
			t.Fatal(err)
		}
		next := loadEvent(t, s, 2)
		if next.EventSeq.String() != "2" || uint64(next.Records[0].UploadBytes) != 8 {
			t.Fatalf("%+v after %s", next, ev.EventSeq)
		}
	})
}

func TestDeltaProperty(t *testing.T) {
	ctx := context.Background()
	clk := &clock{t: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)}
	src := &fakeSrc{}
	up, pk := uint64(0), uint64(0)
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: up, upP: pk}))
	m, s := start(t, src, clk, usagestore.Options{})
	rng := rand.New(rand.NewSource(1))
	var seqs []uint64
	for i := 0; i < 40; i++ {
		clk.Set(clk.Now().Add(10 * time.Second))
		prevUp, prevPk := up, pk
		if rng.Intn(10) == 0 {
			up, pk = uint64(rng.Intn(50)), uint64(rng.Intn(5))
		} else {
			up += uint64(rng.Intn(500))
			pk += uint64(rng.Intn(3))
		}
		reset := up < prevUp || pk < prevPk
		src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: up, upP: pk}))
		if err := m.Sample(ctx); err != nil {
			t.Fatal(err)
		}
		head, err := s.Head(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if head.NextSeq == 0 {
			t.Fatal("seq zero")
		}
		if head.NextSeq > 1 {
			ev, err := s.Event(ctx, head.NextSeq-1)
			if err != nil {
				t.Fatal(err)
			}
			if len(seqs) == 0 || ev.Seq != seqs[len(seqs)-1] {
				if len(seqs) > 0 && ev.Seq != seqs[len(seqs)-1]+1 {
					t.Fatalf("seq jumped to %d from %v", ev.Seq, seqs)
				}
				seqs = append(seqs, ev.Seq)
				var parsed api.UsageEvent
				if err := json.Unmarshal(ev.Payload, &parsed); err != nil {
					t.Fatal(err)
				}
				if reset && parsed.Quality == api.UsageQualityComplete {
					t.Fatalf("reset claimed complete at seq %d", ev.Seq)
				}
				for _, r := range parsed.Records {
					if strings.Contains(r.UploadBytes.String(), "-") || strings.Contains(r.DownloadBytes.String(), "-") {
						t.Fatalf("negative %s", ev.Payload)
					}
				}
			}
		}
	}
	if len(seqs) < 5 {
		t.Fatalf("too few events %v", seqs)
	}
}

func TestMonotonicWaitIgnoresWallJump(t *testing.T) {
	ctx := context.Background()
	wall := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	monoAt := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	clk := &clock{t: wall}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
	s, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), nodeID, incID, usagestore.Options{Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	m, err := New(Config{
		NodeID: nodeID, IncarnationID: incID, Store: s, Source: src,
		Interval: 10 * time.Second, Now: clk.Now, Mono: func() time.Time { return monoAt },
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	// A one-hour wall jump must not make the sampler run early.
	clk.Set(wall.Add(time.Hour))
	wait := m.nextWait(ctx)
	if wait < 9*time.Second {
		t.Fatalf("wall jump shortened the interval to %s", wait)
	}
	monoAt = monoAt.Add(10 * time.Second)
	if wait = m.nextWait(ctx); wait != 0 {
		t.Fatalf("monotonic interval elapsed, wait %s", wait)
	}
	// A backward wall step must not stall the monotonic period.
	monoAt = monoAt.Add(-10 * time.Second)
	clk.Set(wall.Add(-time.Hour))
	wait = m.nextWait(ctx)
	if wait > 10*time.Second || wait < 9*time.Second {
		t.Fatalf("backward wall stalled the sampler: %s", wait)
	}
	clk.Set(time.Date(2026, 11, 1, 0, 0, 1, 0, time.UTC))
	if m.nextWait(ctx) != 0 {
		t.Fatal("month boundary did not wake the sampler")
	}
}

func TestMTR03_RunSkipsBusySampleWithoutLosingCounters(t *testing.T) {
	ctx := t.Context()
	clk := &clock{t: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
	m, s := start(t, src, clk, usagestore.Options{})
	before, err := s.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	m.log = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// Hold a real sample in Source.Read. Run's first tick must see ErrBusy.
	// Its nextWait calls Mono after the tick; cancel there so the test joins
	// Run after exactly that tick, without a timer or a scheduler assumption.
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	m.mono = func() time.Time {
		cancel()
		return clk.Now()
	}
	block := make(chan struct{})
	src.block = block
	src.hit = make(chan struct{}, 1)
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 1000, upP: 10, dnB: 50, dnP: 1}))
	done := make(chan struct{})
	var sampleErr error
	go func() {
		sampleErr = m.Sample(ctx)
		close(done)
	}()
	var release sync.Once
	defer func() {
		release.Do(func() { close(block) })
		<-done
	}()
	<-src.hit
	if err := m.Run(runCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("busy tick: %v", err)
	}
	if logs.Len() != 0 {
		t.Fatalf("busy tick logged an error: %s", logs.String())
	}
	select {
	case <-src.hit:
		t.Fatal("busy tick read the source again")
	default:
	}
	after, err := s.Head(ctx)
	if err != nil || after.NextSeq != before.NextSeq || after.OpenEpoch != before.OpenEpoch || !after.LastSample.Equal(before.LastSample) {
		t.Fatalf("busy tick changed the checkpoint: %+v, %v", after, err)
	}

	release.Do(func() { close(block) })
	<-done
	if sampleErr != nil {
		t.Fatal(sampleErr)
	}
	first := loadEvent(t, s, 1)
	if first.Quality != api.UsageQualityComplete || len(first.Records) != 1 || first.Records[0].UploadBytes != 1000 || first.Records[0].DownloadBytes != 50 {
		t.Fatalf("in-flight counters lost: %+v", first)
	}
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 1750, upP: 15, dnB: 150, dnP: 3}))
	if err := m.Sample(ctx); err != nil {
		t.Fatal(err)
	}
	second := loadEvent(t, s, 2)
	if second.Quality != api.UsageQualityComplete || second.MeterEpochId != before.OpenEpoch || len(second.Records) != 1 || second.Records[0].UploadBytes != 750 || second.Records[0].DownloadBytes != 100 {
		t.Fatalf("next sample lost or recounted counters: %+v", second)
	}
	health, err := m.Health(ctx)
	if err != nil || !health.ForwardingAllowed || health.Reason != "" {
		t.Fatalf("busy tick affected meter health: %+v, %v", health, err)
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	clk := &clock{t: time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}))
	m, _ := start(t, src, clk, usagestore.Options{})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- m.Run(ctx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not stop")
	}
}
