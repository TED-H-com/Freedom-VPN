package usagestore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

var (
	testNode = uuid.MustParse("93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb")
	testInc  = uuid.MustParse("d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0")
	testDev  = uuid.MustParse("2df6c7df-e7e8-4bd4-9f2c-a5836dd0d8ea")
	testMem  = uuid.MustParse("dc01a74b-9e6f-41ac-81cd-03291d0a88c0")
	testDev2 = uuid.MustParse("de484d49-0949-45b1-b38a-40709eb79a7c")
	testMem2 = uuid.MustParse("4e8ca24f-b523-45ef-a782-068cc1c9f809")
)

func openTest(t *testing.T, opt Options) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "usage.db"), testNode, testInc, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func counterAt(dev, mem uuid.UUID, dir string, packets, bytes, rev uint64) Counter {
	hexID := strings.ReplaceAll(dev.String(), "-", "")
	name := "d_" + hexID
	if dir == "upload" {
		name = "u_" + hexID
	}
	return Counter{
		Name: name, DeviceID: dev, Direction: dir,
		Packets: packets, Bytes: bytes, MemberID: mem, BindingRevision: rev,
	}
}

func openEpoch(t *testing.T, s *Store, cs []Counter, at time.Time) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	_, err := s.Commit(context.Background(), Commit{
		Counters:  cs,
		Open:      &NewEpoch{ID: id, Reason: "baseline", BootID: "boot-1"},
		BootID:    "boot-1",
		SampledAt: at,
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestPayloadSHA256_Vector(t *testing.T) {
	epoch := uuid.MustParse("b91a511c-0af5-48af-85cb-eb69c4a0070c")
	start := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	payload, sum, err := canonicalEvent(testNode, testInc, 1234, Draft{
		EpochID: epoch,
		Start:   start,
		End:     start.Add(10 * time.Second),
		Quality: "complete",
		Records: []DraftRecord{
			{MemberID: testMem, DeviceID: testDev, BindingRevision: 42, PeriodID: "2026-10", Upload: 123456, Download: 987654},
			{MemberID: testMem2, DeviceID: testDev2, BindingRevision: 43, PeriodID: "2026-10", Upload: 9007199254740993, Download: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	const want = "27f5775980082d8355caccc9b08aeca47bf53779f7fa3be0b7942f2d0866d1f9"
	if sum != want {
		t.Fatalf("payload_sha256 = %s, want %s\n%s", sum, want, payload)
	}
	if !strings.Contains(string(payload), `"upload_bytes":"9007199254740993"`) {
		t.Fatalf("decimal above 2^53 was not preserved: %s", payload)
	}
	without, err := canonjson.CanonicalizeWithout(payload, "payload_sha256")
	if err != nil {
		t.Fatal(err)
	}
	again, err := canonjson.HashHex(without)
	if err != nil {
		t.Fatal(err)
	}
	if again != want {
		t.Fatalf("rehash = %s", again)
	}
}

func TestDecimalBytesAndACK(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	zero := []Counter{
		counterAt(testDev, testMem, "upload", 0, 0, 7),
		counterAt(testDev2, testMem2, "upload", 0, 0, 9),
	}
	epoch := openEpoch(t, s, zero, at)
	const big = uint64(9007199254740993)
	next := []Counter{
		counterAt(testDev, testMem, "upload", 2, math.MaxUint64, 7),
		counterAt(testDev2, testMem2, "upload", 3, big, 9),
	}
	seq, err := s.Commit(ctx, Commit{
		Epoch: epoch, Counters: next, BootID: "boot-1", SampledAt: at.Add(10 * time.Second),
		Event: &Draft{
			EpochID: epoch, Start: at, End: at.Add(10 * time.Second), Quality: "complete",
			Records: []DraftRecord{
				{MemberID: testMem, DeviceID: testDev, BindingRevision: 7, PeriodID: "2026-10", Upload: math.MaxUint64},
				{MemberID: testMem2, DeviceID: testDev2, BindingRevision: 9, PeriodID: "2026-10", Upload: big},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := s.Event(ctx, seq)
	if err != nil {
		t.Fatal(err)
	}
	body := string(ev.Payload)
	if !strings.Contains(body, `"upload_bytes":"18446744073709551615"`) || !strings.Contains(body, `"upload_bytes":"9007199254740993"`) {
		t.Fatalf("payload lost uint64 precision: %s", body)
	}
	// Move the row to a sequence above 2^53. INTEGER holds it; JSON must too.
	if _, err := s.db.ExecContext(ctx, `UPDATE events SET seq = ? WHERE seq = ?`, int64(big), int64(seq)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE meta SET next_seq = ?, acked_through = ? WHERE id = 1`, int64(big)+1, int64(big-1)); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{
		"blocked_seq": null,
		"highest_contiguous_seq": "9007199254740993",
		"ledger_epoch": "7612b7d9-1e70-4390-b86b-7c3456e4df27",
		"node_id": "93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb",
		"node_incarnation_id": "d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0",
		"received_at": "2026-10-28T15:00:00Z",
		"results": [{"event_seq": "9007199254740993", "status": "accepted", "error": null}]
	}`)
	var res api.UsageBatchResult
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatal(err)
	}
	if uint64(res.HighestContiguousSeq) != big {
		t.Fatalf("highest = %d", uint64(res.HighestContiguousSeq))
	}
	if err := s.ApplyACK(ctx, BatchACK{
		Highest: uint64(res.HighestContiguousSeq),
		Results: []EventResult{{Seq: uint64(res.Results[0].EventSeq), Status: string(res.Results[0].Status)}},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.AckedThrough(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != big || decstr.U64(got).String() != "9007199254740993" {
		t.Fatalf("acked = %d %s", got, decstr.U64(got))
	}
	ev, err = s.Event(ctx, big)
	if err != nil {
		t.Fatal(err)
	}
	if ev.State != "archived" {
		t.Fatalf("state = %s", ev.State)
	}
}

func TestCheckpointRequiresEvent(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 10, 7)}, at)
	before, err := s.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Commit(ctx, Commit{
		Epoch: epoch, BootID: "boot-1", SampledAt: at.Add(time.Second),
		Counters: []Counter{counterAt(testDev, testMem, "upload", 2, 11, 7)},
	})
	if !errors.Is(err, ErrCheckpointWithoutEvent) {
		t.Fatalf("err = %v", err)
	}
	after, err := s.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if after.NextSeq != before.NextSeq || after.Counters[0].Bytes != 10 {
		t.Fatalf("checkpoint moved: %+v", after)
	}
	_, err = s.Commit(ctx, Commit{
		Open: &NewEpoch{ID: uuid.Must(uuid.NewV7()), Reason: "baseline"},
	})
	if err == nil || !strings.Contains(err.Error(), "counter checkpoint") {
		t.Fatalf("err = %v", err)
	}
}

func TestAT16_CrashConsistency(t *testing.T) {
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	ctx := context.Background()
	for _, step := range []string{StepInsert, StepCheckpoint, StepBeforeCommit} {
		t.Run(step, func(t *testing.T) {
			armed := ""
			s := openTest(t, Options{Hook: func(got string) error {
				if armed != "" && got == armed {
					armed = ""
					return ErrInjectedCrash
				}
				return nil
			}})
			epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
			c := delta(epoch, at, 0, 40)
			armed = step
			seq, err := s.Commit(ctx, c)
			if !errors.Is(err, ErrInjectedCrash) || seq != 0 {
				t.Fatalf("seq %d err %v", seq, err)
			}
			if _, err := s.Event(ctx, 1); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("event survived crash: %v", err)
			}
			head, err := s.Head(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if head.NextSeq != 1 || head.Counters[0].Bytes != 0 {
				t.Fatalf("cursor moved: seq %d bytes %d", head.NextSeq, head.Counters[0].Bytes)
			}
			seq, err = s.Commit(ctx, c)
			if err != nil || seq != 1 {
				t.Fatalf("retry seq %d err %v", seq, err)
			}
			seq, err = s.Commit(ctx, delta(epoch, at.Add(10*time.Second), 40, 55))
			if err != nil || seq != 2 {
				t.Fatalf("next seq %d err %v", seq, err)
			}
		})
	}
	t.Run(StepAfterCommit, func(t *testing.T) {
		armed := ""
		s := openTest(t, Options{Hook: func(got string) error {
			if armed != "" && got == armed {
				armed = ""
				return ErrInjectedCrash
			}
			return nil
		}})
		epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
		c := delta(epoch, at, 0, 40)
		armed = StepAfterCommit
		seq, err := s.Commit(ctx, c)
		if !errors.Is(err, ErrInjectedCrash) || seq != 1 {
			t.Fatalf("seq %d err %v", seq, err)
		}
		_, err = s.Commit(ctx, c)
		if !errors.Is(err, ErrDeltaMismatch) {
			t.Fatalf("replay err = %v", err)
		}
		out, arch, err := s.CountEvents(ctx)
		if err != nil || out != 1 || arch != 0 {
			t.Fatalf("events out=%d arch=%d err=%v", out, arch, err)
		}
		seq, err = s.Commit(ctx, delta(epoch, at.Add(10*time.Second), 40, 70))
		if err != nil || seq != 2 {
			t.Fatalf("next seq %d err %v", seq, err)
		}
		ev, err := s.Event(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		again, err := s.Event(ctx, 1)
		if err != nil || string(again.Payload) != string(ev.Payload) {
			t.Fatal("stored payload changed")
		}
	})
}

func delta(epoch uuid.UUID, start time.Time, from, to uint64) Commit {
	end := start.Add(10 * time.Second)
	return Commit{
		Epoch: epoch, BootID: "boot-1", SampledAt: end,
		Counters: []Counter{counterAt(testDev, testMem, "upload", 2, to, 7)},
		Event: &Draft{
			EpochID: epoch, Start: start, End: end, Quality: "complete",
			Records: []DraftRecord{{
				MemberID: testMem, DeviceID: testDev, BindingRevision: 7,
				PeriodID: "2026-10", Upload: to - from,
			}},
		},
	}
}

func TestCapsAndArchiveBudget(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	const limit = uint64(5000)
	s := openTest(t, Options{UnackedMaxBytes: limit, ArchiveMaxBytes: 1_000_000_000})
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
	var bytes uint64
	var n int
	for n = 1; n <= 30; n++ {
		to := bytes + 10
		if _, err := s.Commit(ctx, delta(epoch, at.Add(time.Duration(n)*10*time.Second), bytes, to)); err != nil {
			t.Fatal(err)
		}
		bytes = to
		h, err := s.EvaluateGate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !h.ForwardingAllowed {
			if h.Reason != ReasonOutboxBytes || h.UnackedBytes < limit {
				t.Fatalf("stop %+v", h)
			}
			break
		}
	}
	if n > 30 {
		t.Fatal("outbox cap did not trip")
	}
	for seq := uint64(1); ; seq++ {
		if err := s.ApplyACK(ctx, BatchACK{
			Highest: seq,
			Results: []EventResult{{Seq: seq, Status: StatusAccepted}},
		}); err != nil {
			t.Fatal(err)
		}
		h, err := s.EvaluateGate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if h.UnackedBytes*5 < limit*4 {
			if !h.ForwardingAllowed || h.Reason != "" {
				t.Fatalf("resume %+v", h)
			}
			break
		}
		if h.ForwardingAllowed {
			t.Fatalf("resumed above 80%%: %+v", h)
		}
	}

	t.Run("archive", func(t *testing.T) {
		s := openTest(t, Options{UnackedMaxBytes: 1, ArchiveMaxBytes: 1_000_000_000})
		epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
		if _, err := s.Commit(ctx, delta(epoch, at, 0, 5)); err != nil {
			t.Fatal(err)
		}
		if err := s.ApplyACK(ctx, BatchACK{Highest: 1, Results: []EventResult{{Seq: 1, Status: StatusAccepted}}}); err != nil {
			t.Fatal(err)
		}
		h, err := s.EvaluateGate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !h.ForwardingAllowed || h.ArchiveBytes == 0 || h.UnackedBytes != 0 {
			t.Fatalf("archive tripped the unacked cap: %+v", h)
		}
	})
}

func TestArchiveCapSeparate(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	s := openTest(t, Options{ArchiveMaxBytes: 1, UnackedMaxBytes: 256_000_000})
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
	if _, err := s.Commit(ctx, delta(epoch, at, 0, 5)); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyACK(ctx, BatchACK{Highest: 1, Results: []EventResult{{Seq: 1, Status: StatusAccepted}}}); err != nil {
		t.Fatal(err)
	}
	h, err := s.EvaluateGate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if h.ForwardingAllowed || h.Reason != ReasonArchiveBytes || h.UnackedBytes != 0 {
		t.Fatalf("%+v", h)
	}
}

func TestOutboxAgeAndDiskFloor(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	s := openTest(t, Options{
		Now:           func() time.Time { return now },
		UnackedMaxAge: time.Hour,
	})
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, now)
	if _, err := s.Commit(ctx, delta(epoch, now, 0, 5)); err != nil {
		t.Fatal(err)
	}
	h, err := s.EvaluateGate(ctx)
	if err != nil || !h.ForwardingAllowed {
		t.Fatalf("%+v %v", h, err)
	}
	now = now.Add(time.Hour)
	h, err = s.EvaluateGate(ctx)
	if err != nil || h.ForwardingAllowed || h.Reason != ReasonOutboxAge {
		t.Fatalf("%+v %v", h, err)
	}

	free := uint64(100)
	disk := openTest(t, Options{
		MinFreeBytes: 1000,
		DiskFree:     func(string) (uint64, error) { return free, nil },
	})
	h, err = disk.EvaluateGate(ctx)
	if err != nil || h.Reason != ReasonDiskSpace || h.ForwardingAllowed {
		t.Fatalf("%+v %v", h, err)
	}
	free = 1000 + 1000/4
	h, err = disk.EvaluateGate(ctx)
	if err != nil || !h.ForwardingAllowed {
		t.Fatalf("resume %+v %v", h, err)
	}
	if h.DBBytes == 0 {
		t.Fatal("disk footprint was not measured")
	}
}

func TestPersistFailureStopsForwarding(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	s.NotePersistFailure()
	h, err := s.EvaluateGate(ctx)
	if err != nil || h.ForwardingAllowed || h.Reason != ReasonPersistFail {
		t.Fatalf("%+v %v", h, err)
	}
	s.NotePersistSuccess()
	h, err = s.EvaluateGate(ctx)
	if err != nil || !h.ForwardingAllowed {
		t.Fatalf("%+v %v", h, err)
	}
}

func TestPurgeArchiveKeepsUnacked(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	s := openTest(t, Options{Now: func() time.Time { return now }, ArchiveRetain: time.Hour})
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, now)
	if _, err := s.Commit(ctx, delta(epoch, now, 0, 5)); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyACK(ctx, BatchACK{Highest: 1, Results: []EventResult{{Seq: 1, Status: StatusAccepted}}}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Minute)
	if _, err := s.Commit(ctx, delta(epoch, now, 5, 9)); err != nil {
		t.Fatal(err)
	}
	if err := s.ApplyACK(ctx, BatchACK{Highest: 2, Results: []EventResult{{Seq: 2, Status: StatusDuplicate}}}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(10 * time.Minute)
	if _, err := s.Commit(ctx, delta(epoch, now, 9, 12)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Minute)
	n, err := s.PurgeArchive(ctx, time.Time{})
	if err != nil || n != 1 {
		t.Fatalf("purged %d err %v", n, err)
	}
	if _, err := s.Event(ctx, 1); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("seq 1 still present: %v", err)
	}
	ev, err := s.Event(ctx, 2)
	if err != nil || ev.State != "archived" {
		t.Fatalf("seq 2 %+v %v", ev, err)
	}
	ev, err = s.Event(ctx, 3)
	if err != nil || ev.State != "outbox" {
		t.Fatalf("seq 3 %+v %v", ev, err)
	}
}

func TestACKDoesNotSkip(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	s := openTest(t, Options{})
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
	var from uint64
	for i := 1; i <= 3; i++ {
		to := from + 4
		if _, err := s.Commit(ctx, delta(epoch, at.Add(time.Duration(i)*10*time.Second), from, to)); err != nil {
			t.Fatal(err)
		}
		from = to
	}
	blocked := uint64(2)
	if err := s.ApplyACK(ctx, BatchACK{
		Highest: 3,
		Blocked: &blocked,
		Results: []EventResult{
			{Seq: 1, Status: StatusAccepted},
			{Seq: 3, Status: StatusAccepted},
		},
	}); err != nil {
		t.Fatal(err)
	}
	h, err := s.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if h.AckedThrough != 1 || h.BlockedSeq == nil || *h.BlockedSeq != 2 {
		t.Fatalf("head %+v", h)
	}
	for _, seq := range []uint64{2, 3} {
		ev, err := s.Event(ctx, seq)
		if err != nil || ev.State != "outbox" {
			t.Fatalf("seq %d %+v %v", seq, ev, err)
		}
	}
	if err := s.ApplyACK(ctx, BatchACK{
		Highest: 3,
		Results: []EventResult{
			{Seq: 2, Status: StatusDuplicate},
			{Seq: 3, Status: StatusAccepted},
		},
	}); err != nil {
		t.Fatal(err)
	}
	h, err = s.Head(ctx)
	if err != nil || h.AckedThrough != 3 || h.BlockedSeq != nil {
		t.Fatalf("%+v %v", h, err)
	}
}

func TestIdentityAndLowerACK(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "usage.db")
	s, err := Open(path, testNode, testInc, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = Open(path, uuid.Must(uuid.NewV7()), testInc, Options{})
	if !errors.Is(err, ErrIdentity) {
		t.Fatalf("err = %v", err)
	}
	s, err = Open(path, testNode, testInc, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.ApplyACK(ctx, BatchACK{Highest: 0}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE meta SET acked_through = 5 WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	err = s.ApplyACK(ctx, BatchACK{Highest: 4, Results: []EventResult{{Seq: 4, Status: StatusAccepted}}})
	if !errors.Is(err, ErrAckRegress) {
		t.Fatalf("err = %v", err)
	}
}
