package meter

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
)

func TestReviewR17_MeterIntervalIsNotRuntimeCapped(t *testing.T) {
	clk := &clock{t: time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)}
	src := &fakeSrc{obs: observe("boot")}
	s, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), nodeID, incID, usagestore.Options{Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	m, err := New(Config{NodeID: nodeID, IncarnationID: incID, Store: s, Source: src, Interval: time.Hour, Now: clk.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	if m.interval != time.Hour {
		t.Fatalf("positive interval was rewritten to %s", m.interval)
	}
	if DefaultSampleInterval != 10*time.Second {
		t.Fatalf("default %s", DefaultSampleInterval)
	}
}

func TestReviewR17_FinalSampleDoesNotInventAnEpoch(t *testing.T) {
	clk := &clock{t: time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)}
	m, s := start(t, &fakeSrc{obs: observe("boot")}, clk, usagestore.Options{})
	if _, err := s.ClosedCheckpoint(t.Context()); err == nil {
		t.Fatal("checkpoint existed before a final sample")
	}
	head, err := s.Head(t.Context())
	if err != nil || head.OpenEpoch == uuid.Nil {
		t.Fatal(head, err)
	}
	opened := head.OpenEpoch
	if err = m.FinalSample(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = m.FinalSample(t.Context()); !errors.Is(err, ErrNoEpoch) {
		t.Fatalf("second final sample: %v", err)
	}
	cp, err := s.ClosedCheckpoint(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if cp.ClosedMeterEpochID != opened.String() || cp.NewMeterEpochID != "" {
		t.Fatalf("checkpoint invented an epoch: %+v opened %s", cp, opened)
	}
}
