package usagestore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// TestOutboxStatusIsPure copies the checkpointed database, including the WAL
// and shared-memory files, and shows repeated status reads leave those bytes
// alone in every gate state. The returned status matches the stream fields
// EvaluateGate would report. The read does not persist a stop or a resume.
func TestOutboxStatusIsPure(t *testing.T) {
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	t.Run("open", func(t *testing.T) {
		opt := Options{UnackedMaxBytes: 1_000_000}
		s := openTest(t, opt)
		epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
		if _, err := s.Commit(context.Background(), delta(epoch, at, 0, 5)); err != nil {
			t.Fatal(err)
		}
		st := proveStatusPure(t, s, opt)
		if st.UnackedEventCount != 1 || st.CentralAckedThrough.String() != "0" {
			t.Fatalf("%s", mustStatusJSON(t, st))
		}
		if gateStopped(t, s) != 0 {
			t.Fatal("open gate was persisted shut")
		}
		h := evalClone(t, s, opt)
		if !h.ForwardingAllowed || h.Reason != "" {
			t.Fatalf("evaluation would not leave the gate open: %+v", h)
		}
	})
	t.Run("would-stop", func(t *testing.T) {
		opt := Options{UnackedMaxBytes: 1}
		s := openTest(t, opt)
		epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
		if _, err := s.Commit(context.Background(), delta(epoch, at, 0, 5)); err != nil {
			t.Fatal(err)
		}
		if gateStopped(t, s) != 0 {
			t.Fatal("gate stopped before any evaluation")
		}
		st := proveStatusPure(t, s, opt)
		if st.UnackedEventCount != 1 || st.UnackedPayloadBytes.String() == "0" {
			t.Fatalf("%s", mustStatusJSON(t, st))
		}
		if gateStopped(t, s) != 0 {
			t.Fatal("status read persisted gate_stopped")
		}
		h := evalClone(t, s, opt)
		if h.ForwardingAllowed || h.Reason != ReasonOutboxBytes {
			t.Fatalf("evaluation would not stop: %+v", h)
		}
		if mustStatusJSON(t, statusFromHealth(h)) != mustStatusJSON(t, st) {
			t.Fatalf("status %s\neval %s", mustStatusJSON(t, st), mustStatusJSON(t, statusFromHealth(h)))
		}
	})
	t.Run("stopped", func(t *testing.T) {
		opt := Options{UnackedMaxBytes: 1}
		s := openTest(t, opt)
		epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
		if _, err := s.Commit(context.Background(), delta(epoch, at, 0, 5)); err != nil {
			t.Fatal(err)
		}
		h, err := s.EvaluateGate(context.Background())
		if err != nil || h.ForwardingAllowed || h.Reason != ReasonOutboxBytes {
			t.Fatalf("setup %+v %v", h, err)
		}
		if gateStopped(t, s) != 1 {
			t.Fatal("gate was not stopped")
		}
		// The backlog is now empty. EvaluateGate would resume. The status
		// read must leave the persisted stop in place.
		if err := s.ApplyACK(context.Background(), BatchACK{
			Highest: 1, Results: []EventResult{{Seq: 1, Status: StatusAccepted}},
		}); err != nil {
			t.Fatal(err)
		}
		st := proveStatusPure(t, s, opt)
		if st.UnackedEventCount != 0 || st.CentralAckedThrough.String() != "1" {
			t.Fatalf("%s", mustStatusJSON(t, st))
		}
		if gateStopped(t, s) != 1 {
			t.Fatal("status read cleared gate_stopped")
		}
		h = evalClone(t, s, opt)
		if !h.ForwardingAllowed || h.Reason != "" {
			t.Fatalf("evaluation would not resume: %+v", h)
		}
		if mustStatusJSON(t, statusFromHealth(h)) != mustStatusJSON(t, st) {
			t.Fatalf("status %s\neval %s", mustStatusJSON(t, st), mustStatusJSON(t, statusFromHealth(h)))
		}
	})
}

func TestLedgerEpochNullUntilCheckpoint(t *testing.T) {
	ctx := context.Background()
	priv, trust := testTrust(t)
	now := time.Date(2026, 10, 28, 18, 0, 0, 0, time.UTC)
	s := openTest(t, Options{Trust: trust, Now: func() time.Time { return now }})
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
	if _, err := s.Commit(ctx, delta(epoch, at, 0, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Commit(ctx, delta(epoch, at.Add(10*time.Second), 5, 9)); err != nil {
		t.Fatal(err)
	}
	ackedIn := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
	if err := s.ApplyACK(ctx, BatchACK{
		LedgerEpoch: ackedIn, Highest: 1,
		Results: []EventResult{{Seq: 1, Status: StatusAccepted}},
	}); err != nil {
		t.Fatal(err)
	}
	st, err := s.OutboxStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.LedgerEpoch.Get(); err == nil {
		t.Fatalf("ACK adopted ledger epoch %s", mustStatusJSON(t, st))
	}
	if st.CentralAckedThrough.String() != "1" || st.LastDurableEventSeq.String() != "2" {
		t.Fatalf("%s", mustStatusJSON(t, st))
	}
	snap, err := ReadSnapshot(ctx, s.Path())
	if err != nil || snap.LedgerEpoch != nil || snap.AckedThrough != "1" {
		t.Fatalf("snapshot %+v %v", snap, err)
	}
	head, err := s.Head(ctx)
	if err != nil || head.LedgerEpoch != uuid.Nil || head.AckedThrough != 1 {
		t.Fatalf("head %+v %v", head, err)
	}

	adopted := uuid.MustParse("7612b7d9-1e70-4390-b86b-7c3456e4df27")
	raw := signCheckpoint(t, priv, api.UsageCheckpoint{
		LedgerEpoch: adopted, IssuedAt: now,
		HighestContiguousSeq: 1, MaxSeenSeq: 2,
	})
	if err := s.ApplyCheckpoint(ctx, raw, now); err != nil {
		t.Fatal(err)
	}
	st, err = s.OutboxStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.LedgerEpoch.Get()
	if err != nil || got != adopted {
		t.Fatalf("epoch %s err %v body %s", got, err, mustStatusJSON(t, st))
	}
	if st.CentralAckedThrough.String() != "1" {
		t.Fatalf("watermark moved with the checkpoint: %s", mustStatusJSON(t, st))
	}
	snap, err = ReadSnapshot(ctx, s.Path())
	if err != nil || snap.LedgerEpoch == nil || *snap.LedgerEpoch != adopted.String() {
		t.Fatalf("snapshot epoch %+v %v", snap.LedgerEpoch, err)
	}
}

func proveStatusPure(t *testing.T, s *Store, opt Options) api.UsageOutboxStatus {
	t.Helper()
	if _, err := s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	before := readLedgerFiles(t, s.Path())
	ctx := context.Background()
	first, err := s.OutboxStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.OutboxStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mustStatusJSON(t, first) != mustStatusJSON(t, second) {
		t.Fatalf("repeated status diverged\n%s\n%s", mustStatusJSON(t, first), mustStatusJSON(t, second))
	}
	after := readLedgerFiles(t, s.Path())
	if !ledgerFilesEqual(before, after) {
		t.Fatalf("OutboxStatus changed the ledger files\nbefore %#v\nafter %#v", fileSizes(before), fileSizes(after))
	}
	clone := cloneLedger(t, s, opt)
	h, err := clone.EvaluateGate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if mustStatusJSON(t, statusFromHealth(h)) != mustStatusJSON(t, first) {
		t.Fatalf("status %s\neval %s", mustStatusJSON(t, first), mustStatusJSON(t, statusFromHealth(h)))
	}
	return first
}

func evalClone(t *testing.T, s *Store, opt Options) Health {
	t.Helper()
	h, err := cloneLedger(t, s, opt).EvaluateGate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func cloneLedger(t *testing.T, src *Store, opt Options) *Store {
	t.Helper()
	if _, err := src.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "usage.db")
	raw, err := os.ReadFile(src.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dst, testNode, testInc, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func gateStopped(t *testing.T, s *Store) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow(`SELECT gate_stopped FROM meta WHERE id = 1`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

type ledgerFile struct {
	name string
	body []byte
	ok   bool
}

func readLedgerFiles(t *testing.T, path string) []ledgerFile {
	t.Helper()
	out := make([]ledgerFile, 0, 3)
	for _, name := range []string{path, path + "-wal", path + "-shm"} {
		body, err := os.ReadFile(name)
		if errors.Is(err, os.ErrNotExist) {
			out = append(out, ledgerFile{name: name})
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, ledgerFile{name: name, body: body, ok: true})
	}
	return out
}

func ledgerFilesEqual(a, b []ledgerFile) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].name != b[i].name || a[i].ok != b[i].ok || !bytes.Equal(a[i].body, b[i].body) {
			return false
		}
	}
	return true
}

func fileSizes(files []ledgerFile) map[string]int {
	out := map[string]int{}
	for _, f := range files {
		if !f.ok {
			out[filepath.Base(f.name)] = -1
			continue
		}
		out[filepath.Base(f.name)] = len(f.body)
	}
	return out
}

func mustStatusJSON(t *testing.T, st api.UsageOutboxStatus) string {
	t.Helper()
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
