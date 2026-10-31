package usagestore

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

var nextInc = uuid.MustParse("01000000-0000-4000-8000-000000000014")

func handoffStore(t *testing.T, opt Options) (*Store, StoredEvent) {
	t.Helper()
	s := openTest(t, opt)
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	ep := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 0, 0, 42)}, at)
	_, err := s.Commit(context.Background(), Commit{
		Epoch: ep, Counters: []Counter{counterAt(testDev, testMem, "upload", 1, 9007199254740993, 42)}, SampledAt: at.Add(10 * time.Second),
		Event: &Draft{EpochID: ep, Start: at, End: at.Add(10 * time.Second), Quality: "complete", Records: []DraftRecord{{MemberID: testMem, DeviceID: testDev, BindingRevision: 42, PeriodID: "2026-10", Upload: 9007199254740993}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := s.Event(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	return s, ev
}

func TestENR05_ReinstallRetainsOriginalStream(t *testing.T) {
	ctx := context.Background()
	s, before := handoffStore(t, Options{})
	path := s.Path()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, testNode, nextInc, Options{}); !errors.Is(err, ErrIdentity) {
		t.Fatalf("unapproved switch: %v", err)
	}
	at := before.End
	st, err := Open(path, testNode, nextInc, Options{RetirePreviousAt: at})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	for i := 0; i < 2; i++ {
		streams, err := st.RetiredStreams(ctx)
		if err != nil || len(streams) != 1 {
			t.Fatalf("streams %+v: %v", streams, err)
		}
		r := streams[0]
		if r.IncarnationID != testInc || !r.RetiredAt.Equal(at) || r.AckedThrough != 0 || r.LastSeq != 1 {
			t.Fatalf("retired identity %+v", r)
		}
		events, err := st.RetiredUnacked(ctx, testInc, 100, 512000)
		if err != nil || len(events) != 1 || !bytes.Equal(events[0].Payload, before.Payload) || events[0].Hash != before.Hash {
			t.Fatalf("changed original event: %+v %v", events, err)
		}
		h, err := st.Head(ctx)
		if err != nil || h.NextSeq != 1 || h.AckedThrough != 0 || h.HasHistory || h.OpenEpoch != uuid.Nil || h.LedgerEpoch != uuid.Nil {
			t.Fatalf("new cursor %+v %v", h, err)
		}
		if i == 0 {
			if err := st.Close(); err != nil {
				t.Fatal(err)
			}
			// Same usage.db and new identity only. No retired key/certificate exists.
			st, err = Open(path, testNode, nextInc, Options{})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := st.ApplyRetiredACK(ctx, testInc, BatchACK{Highest: 1, Results: []EventResult{{Seq: 1, Status: StatusAccepted}}}); err != nil {
		t.Fatal(err)
	}
	streams, _ := st.RetiredStreams(ctx)
	h, _ := st.Head(ctx)
	if streams[0].AckedThrough != 1 || streams[0].Pending != 0 || h.AckedThrough != 0 {
		t.Fatal("ACK crossed streams")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path, testNode, testInc, Options{RetirePreviousAt: at}); !errors.Is(err, ErrIdentity) {
		t.Fatalf("reused retired identity: %v", err)
	}
	if _, err := Open(path, uuid.New(), uuid.New(), Options{RetirePreviousAt: at}); !errors.Is(err, ErrIdentity) {
		t.Fatalf("foreign node: %v", err)
	}
}

func TestAT16_RetirementTransactionCrash(t *testing.T) {
	for _, step := range []string{StepRetireBeforeCommit, StepRetireAfterCommit} {
		t.Run(step, func(t *testing.T) {
			s, ev := handoffStore(t, Options{})
			path := s.Path()
			_ = s.Close()
			opt := Options{RetirePreviousAt: ev.End, Hook: func(at string) error {
				if at == step {
					return ErrInjectedCrash
				}
				return nil
			}}
			if _, err := Open(path, testNode, nextInc, opt); !errors.Is(err, ErrInjectedCrash) {
				t.Fatalf("crash: %v", err)
			}
			opt.Hook = nil
			st, err := Open(path, testNode, nextInc, opt)
			if err != nil {
				t.Fatal(err)
			}
			defer st.Close()
			streams, err := st.RetiredStreams(context.Background())
			if err != nil || len(streams) != 1 {
				t.Fatalf("rotation duplicated: %+v %v", streams, err)
			}
			events, err := st.RetiredUnacked(context.Background(), testInc, 1, 1)
			if err != nil || len(events) != 1 || !bytes.Equal(events[0].Payload, ev.Payload) {
				t.Fatal("crash lost tail")
			}
		})
	}
}

func TestD105_RejectionAuditAndACKAtomic(t *testing.T) {
	ctx := context.Background()
	s, ev := handoffStore(t, Options{})
	path := s.Path()
	_ = s.Close()
	fail := true
	st, err := Open(path, testNode, nextInc, Options{RetirePreviousAt: ev.End, Hook: func(step string) error {
		if fail && step == StepACKBeforeCommit {
			return ErrInjectedCrash
		}
		return nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ack := BatchACK{Results: []EventResult{{Seq: 1, Status: StatusRejected, Code: "VALIDATION_FAILED"}}}
	if err := st.ApplyRetiredACK(ctx, testInc, ack); !errors.Is(err, ErrInjectedCrash) {
		t.Fatal(err)
	}
	audit, err := st.HandoffRejections(ctx)
	if err != nil || len(audit) != 0 {
		t.Fatal("partial rejection commit")
	}
	pending, _ := st.RetiredUnacked(ctx, testInc, 100, 512000)
	if len(pending) != 1 {
		t.Fatal("lost pending event on rollback")
	}
	fail = false
	for i := 0; i < 2; i++ {
		if err := st.ApplyRetiredACK(ctx, testInc, ack); err != nil {
			t.Fatal(err)
		}
	}
	audit, err = st.HandoffRejections(ctx)
	if err != nil || len(audit) != 1 || audit[0].Origin != testInc || audit[0].Presenter != nextInc || audit[0].Seq != 1 || audit[0].Code != "VALIDATION_FAILED" {
		t.Fatalf("audit %+v %v", audit, err)
	}
	var retained []byte
	if err := st.db.QueryRow(`SELECT payload FROM retired_events WHERE incarnation_id = ? AND seq = 1`, testInc.String()).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retained, ev.Payload) {
		t.Fatal("rejection deleted or rewrote evidence")
	}
}

func TestD105_MigrateV1(t *testing.T) {
	s, ev := handoffStore(t, Options{})
	path := s.Path()
	// The v1 tables are unchanged. Remove only the v2 additions to model a
	// real retained database from the previous release.
	for _, sql := range []string{"DROP TABLE handoff_rejections", "DROP TABLE retired_events", "DROP TABLE retired_history", "DROP TABLE retired_replay_unavailable", "DROP TABLE retired_streams", "PRAGMA user_version = 1"} {
		if _, err := s.db.Exec(sql); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ReadSnapshot(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	st, err := Open(path, testNode, nextInc, Options{RetirePreviousAt: ev.End})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	got, err := st.RetiredUnacked(context.Background(), testInc, 1, 512000)
	if err != nil || len(got) != 1 || !bytes.Equal(got[0].Payload, ev.Payload) {
		t.Fatal("v1 migration lost event")
	}
}

func TestD105_RetainedCapacityAndArchive(t *testing.T) {
	ctx := context.Background()
	s, ev := handoffStore(t, Options{})
	path := s.Path()
	_ = s.Close()
	at := ev.End
	st, err := Open(path, testNode, nextInc, Options{RetirePreviousAt: at, Now: func() time.Time { return at }, UnackedMaxBytes: uint64(len(ev.Payload))})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	status, err := st.OutboxStatus(ctx)
	if err != nil || status.UnackedEventCount != 0 {
		t.Fatalf("current status %+v %v", status, err)
	}
	health, err := st.EvaluateGate(ctx)
	if err != nil || health.ForwardingAllowed || health.Reason != ReasonOutboxBytes || health.UnackedCount != 1 {
		t.Fatalf("retired backlog escaped budget: %+v %v", health, err)
	}
	if err := st.ApplyRetiredACK(ctx, testInc, BatchACK{Highest: 1, Results: []EventResult{{Seq: 1, Status: StatusAccepted}}}); err != nil {
		t.Fatal(err)
	}
	health, err = st.EvaluateGate(ctx)
	if err != nil || !health.ForwardingAllowed || health.UnackedCount != 0 || health.ArchiveBytes != uint64(len(ev.Payload)) {
		t.Fatalf("archive accounting %+v %v", health, err)
	}
	if n, err := st.PurgeArchive(ctx, at.Add(DefaultArchiveRetain-time.Second)); err != nil || n != 0 {
		t.Fatalf("early purge %d %v", n, err)
	}
	if n, err := st.PurgeArchive(ctx, at.Add(DefaultArchiveRetain+time.Second)); err != nil || n != 1 {
		t.Fatalf("expired archive %d %v", n, err)
	}
	streams, err := st.RetiredStreams(ctx)
	if err != nil || streams[0].AckedThrough != 1 {
		t.Fatal("purge lost original ACK")
	}
}

func TestD105_RecoveryRangesBelongToOrigin(t *testing.T) {
	ctx := context.Background()
	s, ev := handoffStore(t, Options{})
	// TEST ONLY: model a previously verified recovery gap before this payload.
	if _, err := s.db.Exec(`UPDATE events SET seq = 3`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE meta SET next_seq = 4`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO replay_unavailable VALUES (1,2,'retention_expired')`); err != nil {
		t.Fatal(err)
	}
	path := s.Path()
	_ = s.Close()
	st, err := Open(path, testNode, nextInc, Options{RetirePreviousAt: ev.End})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ranges, err := st.RetiredReplayUnavailable(ctx, testInc)
	if err != nil || len(ranges) != 1 || ranges[0].StartSeq != 1 || ranges[0].EndSeq != 2 {
		t.Fatalf("ranges %+v %v", ranges, err)
	}
	var history string
	if err := st.db.QueryRow(`SELECT body FROM retired_history WHERE incarnation_id = ? AND kind = 'replay_unavailable'`, testInc.String()).Scan(&history); err != nil {
		t.Fatal(err)
	}
	if history != `[{"start_seq":"1","end_seq":"2","reason":"retention_expired"}]` {
		t.Fatal("missing retirement-time gap evidence", history)
	}
	if err := st.ApplyRetiredACK(ctx, testInc, BatchACK{Highest: 3, Results: []EventResult{{Seq: 3, Status: StatusAccepted}}}); err != nil {
		t.Fatal(err)
	}
	streams, _ := st.RetiredStreams(ctx)
	h, _ := st.Head(ctx)
	if streams[0].AckedThrough != 3 || h.AckedThrough != 0 {
		t.Fatal("gap advanced wrong stream")
	}
	ranges, _ = st.RetiredReplayUnavailable(ctx, testInc)
	if len(ranges) != 0 {
		t.Fatal("ACKed gap remained pending")
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = Open(path, testNode, nextInc, Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	var after string
	if err := st.db.QueryRow(`SELECT body FROM retired_history WHERE incarnation_id = ? AND kind = 'replay_unavailable'`, testInc.String()).Scan(&after); err != nil || after != history {
		t.Fatal("ACK/restart lost gap audit evidence", after, err)
	}
	ranges, err = st.RetiredReplayUnavailable(ctx, testInc)
	if err != nil || len(ranges) != 0 {
		t.Fatal("restart recreated reported gaps", ranges, err)
	}
}

func TestD105_HandoffACKClearsUnconfirmedOldBlocker(t *testing.T) {
	ctx := context.Background()
	s, ev := handoffStore(t, Options{})
	blocked := uint64(1)
	if err := s.ApplyACK(ctx, BatchACK{Blocked: &blocked, Results: []EventResult{{Seq: 1, Status: StatusRejected}}}); err != nil {
		t.Fatal(err)
	}
	path := s.Path()
	_ = s.Close()
	st, err := Open(path, testNode, nextInc, Options{RetirePreviousAt: ev.End})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	streams, _ := st.RetiredStreams(ctx)
	if streams[0].BlockedSeq == nil {
		t.Fatal("old blocker was not retained")
	}
	// Central now accepts this original event. The earlier live rejection
	// must not turn into a fabricated final D-105 receipt or pin the old ACK.
	if err := st.ApplyRetiredACK(ctx, testInc, BatchACK{Highest: 1, Results: []EventResult{{Seq: 1, Status: StatusAccepted}}}); err != nil {
		t.Fatal(err)
	}
	streams, _ = st.RetiredStreams(ctx)
	if streams[0].AckedThrough != 1 || streams[0].BlockedSeq != nil || streams[0].Pending != 0 {
		t.Fatalf("stale live blocker %+v", streams[0])
	}
	audit, _ := st.HandoffRejections(ctx)
	if len(audit) != 0 {
		t.Fatal("invented final hand-off rejection")
	}
}
