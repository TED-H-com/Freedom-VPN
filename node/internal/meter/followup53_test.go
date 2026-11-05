package meter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TEST ONLY: start with an accounted, sealed interval and two counter identities.
func f53Accounted(t *testing.T) (*Meter, *usagestore.Store, *fakeSrc, *clock, usagestore.Head, usagestore.StoredEvent) {
	t.Helper()
	clk := &clock{t: time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)}
	src := &fakeSrc{}
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1}, reading{dev: devB, mem: memB, rev: 2}))
	m, s := start(t, src, clk, usagestore.Options{})
	clk.Set(clk.Now().Add(10 * time.Second))
	src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 100, upP: 1}, reading{dev: devB, mem: memB, rev: 2, upB: 200, upP: 2}))
	if err := m.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	h, err := s.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ev, err := s.Event(t.Context(), 1)
	if err != nil {
		t.Fatal(err)
	}
	accounted := loadEvent(t, s, 1)
	if len(accounted.Records) != 2 || accounted.Quality != api.UsageQualityComplete {
		t.Fatalf("precondition event: %+v", accounted)
	}
	var sum uint64
	for _, r := range accounted.Records {
		sum += uint64(r.UploadBytes)
	}
	if sum != 300 || h.NextSeq != 2 {
		t.Fatalf("precondition bytes=%d next=%d", sum, h.NextSeq)
	}
	return m, s, src, clk, h, ev
}

func f53Unchanged(t *testing.T, s *usagestore.Store, h usagestore.Head, ev usagestore.StoredEvent) {
	t.Helper()
	got, err := s.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, h) {
		t.Fatalf("checkpoint changed ledger head: before=%+v after=%+v", h, got)
	}
	f53Sealed(t, s, ev)
	if _, err := s.Event(t.Context(), h.NextSeq); err == nil {
		t.Fatal("unchanged/held checkpoint minted an event")
	}
}

func f53Sealed(t *testing.T, s *usagestore.Store, ev usagestore.StoredEvent) {
	t.Helper()
	got, err := s.Event(t.Context(), ev.Seq)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, ev) {
		t.Fatalf("sealed event changed: before=%+v after=%+v", ev, got)
	}
}

func TestMTR03_RestartCheckpointClockBoundaries(t *testing.T) {
	for _, rollback := range []time.Duration{0, time.Microsecond, 5 * time.Second, DefaultSampleInterval} {
		t.Run(rollback.String(), func(t *testing.T) {
			m, s, src, clk, h, ev := f53Accounted(t)
			clk.Set(h.LastSample.Add(-rollback))
			if err := m.Sample(t.Context()); !errors.Is(err, ErrHold) {
				t.Fatalf("ordinary Sample: %v, want ErrHold", err)
			}
			f53Unchanged(t, s, h, ev)
			if err := m.CheckpointForRestart(t.Context()); err != nil {
				t.Fatalf("unchanged restart within interval: %v", err)
			}
			f53Unchanged(t, s, h, ev)
			src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 101, upP: 2}, reading{dev: devB, mem: memB, rev: 2, upB: 200, upP: 2}))
			if err := m.CheckpointForRestart(t.Context()); !errors.Is(err, ErrHold) {
				t.Fatalf("changed restart: %v, want ErrHold", err)
			}
			if err := m.FinalSample(t.Context()); !errors.Is(err, ErrHold) {
				t.Fatalf("changed FinalSample: %v, want ErrHold", err)
			}
			f53Unchanged(t, s, h, ev)
			// The held byte is attributed once when time advances, then an unchanged
			// FinalSample at the same instant closes without another event.
			clk.Set(h.LastSample.Add(time.Second))
			if err := m.CheckpointForRestart(t.Context()); err != nil {
				t.Fatal(err)
			}
			delta := loadEvent(t, s, h.NextSeq)
			if delta.Quality != api.UsageQualityComplete || len(delta.Records) != 1 || delta.Records[0].MemberId != memA || uint64(delta.Records[0].BindingRevision) != 1 || uint64(delta.Records[0].UploadBytes) != 1 {
				t.Fatalf("pending delta: %+v", delta)
			}
			before, err := s.Head(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if before.OpenEpoch != h.OpenEpoch || before.NextSeq != h.NextSeq+1 {
				t.Fatalf("restart cursor: %+v", before)
			}
			if err := m.FinalSample(t.Context()); err != nil {
				t.Fatal(err)
			}
			after, err := s.Head(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			if after.OpenEpoch != uuid.Nil || after.LastCloseReason != "final_sample" || after.NextSeq != before.NextSeq || !after.LastSample.Equal(before.LastSample) || len(after.Counters) != 0 || after.AckedThrough != h.AckedThrough || after.LedgerEpoch != h.LedgerEpoch {
				t.Fatalf("final cursor: before=%+v after=%+v", before, after)
			}
			f53Sealed(t, s, ev)
		})
	}
}

func TestMTR03_RestartCheckpointOutsideClockWindow(t *testing.T) {
	for _, operation := range []string{"sample", "restart", "final"} {
		t.Run(operation, func(t *testing.T) {
			m, s, _, clk, h, ev := f53Accounted(t)
			clk.Set(h.LastSample.Add(-DefaultSampleInterval - time.Nanosecond))
			call := map[string]func(context.Context) error{"sample": m.Sample, "restart": m.CheckpointForRestart, "final": m.FinalSample}[operation]
			if err := call(t.Context()); err != nil {
				t.Fatal(err)
			}
			after, err := s.Head(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			gap := loadEvent(t, s, h.NextSeq)
			if gap.Quality != api.UsageQualityGap || gap.Gap == nil || gap.Gap.Reason != api.UsageGapReasonClockAnomaly || len(gap.Records) != 0 || gap.IntervalEnd.Sub(gap.IntervalStart) != time.Microsecond || !gap.IntervalStart.Equal(h.LastSample) {
				t.Fatalf("outside-window gap: %+v", gap)
			}
			wantEpoch := h.OpenEpoch
			if operation == "final" {
				wantEpoch = uuid.Nil
			}
			if after.OpenEpoch != wantEpoch || after.NextSeq != h.NextSeq+1 || !after.LastSample.Equal(gap.IntervalEnd) || (operation != "final" && !reflect.DeepEqual(after.Counters, h.Counters)) || (operation == "final" && len(after.Counters) != 0) || after.AckedThrough != h.AckedThrough || after.LedgerEpoch != h.LedgerEpoch {
				t.Fatalf("outside-window cursor: %+v", after)
			}
			if operation == "final" && after.LastCloseReason != "final_sample" {
				t.Fatalf("final reason: %+v", after)
			}
			f53Sealed(t, s, ev)
		})
	}
}

func TestMTR03_RestartCheckpointResetAndIdentityPrecedence(t *testing.T) {
	for _, kind := range []string{"boot", "counter", "binding"} {
		t.Run(kind, func(t *testing.T) {
			m, s, src, clk, h, ev := f53Accounted(t)
			clk.Set(h.LastSample.Add(-time.Second))
			switch kind {
			case "boot":
				src.set(observe("next-boot", reading{dev: devA, mem: memA, rev: 1, upB: 100, upP: 1}, reading{dev: devB, mem: memB, rev: 2, upB: 200, upP: 2}))
			case "counter":
				src.set(observe("boot", reading{dev: devA, mem: memA, rev: 1, upB: 1, upP: 0}, reading{dev: devB, mem: memB, rev: 2, upB: 220, upP: 3}))
			case "binding":
				src.set(observe("boot", reading{dev: devA, mem: memB, rev: 3, upB: 100, upP: 1}, reading{dev: devB, mem: memB, rev: 2, upB: 200, upP: 2}))
			}
			if err := m.CheckpointForRestart(t.Context()); err != nil {
				t.Fatal(err)
			}
			if kind == "binding" {
				// Equal numeric readings must not silently replace historical attribution.
				f53Unchanged(t, s, h, ev)
				clk.Set(h.LastSample.Add(time.Second))
				src.set(observe("boot", reading{dev: devA, mem: memB, rev: 3, upB: 101, upP: 2}, reading{dev: devB, mem: memB, rev: 2, upB: 200, upP: 2}))
				if err := m.CheckpointForRestart(t.Context()); err != nil {
					t.Fatal(err)
				}
				gap := loadEvent(t, s, h.NextSeq)
				if gap.Quality != api.UsageQualityGap || gap.Gap == nil || gap.Gap.Reason != api.UsageGapReasonUnknown || len(gap.Records) != 0 {
					t.Fatalf("identity change claimed bytes: %+v", gap)
				}
			} else {
				after, err := s.Head(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				wantReason := api.UsageGapReasonCounterDiscontinuity
				if kind == "boot" {
					wantReason = api.UsageGapReasonPowerLoss
				}
				gap := loadEvent(t, s, h.NextSeq)
				if after.OpenEpoch == h.OpenEpoch || after.OpenEpoch == uuid.Nil || after.NextSeq != h.NextSeq+1 || gap.Gap == nil || gap.Gap.Reason != wantReason || gap.MeterEpochId != h.OpenEpoch {
					t.Fatalf("reset must precede hold: head=%+v gap=%+v", after, gap)
				}
				if kind == "boot" && (len(gap.Records) != 0 || after.BootID != "next-boot") {
					t.Fatalf("boot discontinuity: %+v %+v", after, gap)
				}
				if kind == "counter" && (len(gap.Records) != 1 || gap.Records[0].DeviceId != devB || uint64(gap.Records[0].UploadBytes) != 20) {
					t.Fatalf("reset lost neighboring delta: %+v", gap)
				}
				if !reflect.DeepEqual(after.Counters, usageCounters(src.obs)) {
					t.Fatalf("reset checkpoint counters: %+v", after.Counters)
				}
			}
			f53Sealed(t, s, ev)
		})
	}
}

func TestMTR03_FinalSampleUnchangedRollback(t *testing.T) {
	m, s, _, clk, h, ev := f53Accounted(t)
	clk.Set(h.LastSample.Add(-5 * time.Second))
	if err := m.FinalSample(t.Context()); err != nil {
		t.Fatal(err)
	}
	after, err := s.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := s.ClosedCheckpoint(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if after.OpenEpoch != uuid.Nil || after.LastCloseReason != "final_sample" || after.NextSeq != h.NextSeq || !after.LastSample.Equal(h.LastSample) || len(after.Counters) != 0 || checkpoint.ClosedMeterEpochID != h.OpenEpoch.String() || uint64(checkpoint.LastDurableEventSeq) != h.NextSeq-1 || uint64(checkpoint.CentralAckedThrough) != h.AckedThrough {
		t.Fatalf("closed checkpoint: head=%+v checkpoint=%+v", after, checkpoint)
	}
	f53Sealed(t, s, ev)
}
