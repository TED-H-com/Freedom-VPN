package usageupload

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// Retire a real store after a signed recovery checkpoint discovers that seq 1
// has expired from its archive. Remaining events keep their original bytes.
func retiredRecovery(t *testing.T, count int) (*usagestore.Store, []usagestore.StoredEvent) {
	t.Helper()
	ctx := t.Context()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	trust, err := envelope.NewNodeTrustStore(envelope.TrustedKey{
		KeyID: "recovery-test", PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRecovery},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	s := seed(t, count, usagestore.Options{Trust: trust, Now: func() time.Time { return now }})
	cp := api.UsageCheckpoint{
		NodeId: upNode, NodeIncarnationId: upInc, LedgerEpoch: uuid.New(), IssuedAt: now,
		HighestContiguousSeq: 1, MaxSeenSeq: decstr.U64(count), MissingRanges: []api.SeqRange{},
		RecoveryRecordId: presence.NewNullNullable[uuid.UUID](),
	}
	apply := func() {
		t.Helper()
		env, err := envelope.Sign(cp, priv, "recovery-test")
		if err != nil {
			t.Fatal(err)
		}
		raw, err := env.MarshalCanonical()
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ApplyCheckpoint(ctx, raw, now); err != nil {
			t.Fatal(err)
		}
	}
	apply()
	if n, err := s.PurgeArchive(ctx, now.Add(usagestore.DefaultArchiveRetain+time.Second)); err != nil || n != 1 {
		t.Fatalf("purge: %d %v", n, err)
	}
	cp.LedgerEpoch, cp.HighestContiguousSeq = epochA, 0
	cp.MissingRanges = []api.SeqRange{{StartSeq: 1, EndSeq: 1}}
	apply()
	held, err := s.Unacked(ctx, 0, 100, 512000)
	if err != nil || len(held) != count-1 {
		t.Fatalf("retained tail: %d %v", len(held), err)
	}
	path := s.Path()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = usagestore.Open(path, upNode, presentingInc, usagestore.Options{Trust: trust, RetirePreviousAt: now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, held
}

func reopenHandoff(t *testing.T, s *usagestore.Store) *usagestore.Store {
	t.Helper()
	path := s.Path()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := usagestore.Open(path, upNode, presentingInc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestD105_RetiredGapsReportedNeverRefetched(t *testing.T) {
	for _, lostACK := range []bool{false, true} {
		t.Run(map[bool]string{false: "acknowledged", true: "lost-ACK-restart"}[lostACK], func(t *testing.T) {
			ctx := t.Context()
			s, held := retiredRecovery(t, 3)
			head, err := s.Head(ctx)
			if err != nil || head.NextSeq != 1 || head.AckedThrough != 0 {
				t.Fatalf("new cursor: %+v %v", head, err)
			}
			currentEvent(t, s)
			current, err := s.Event(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			gets, reports, currentPosts := 0, 0, 0
			tr := &fakeTR{getFn: func() (int, []byte, error) { gets++; return 404, nil, nil }}
			tr.postFn = func(_ string, body []byte) (int, []byte, time.Duration, error) {
				var batch wireBatch
				if err := json.Unmarshal(body, &batch); err != nil {
					t.Fatal(err)
				}
				if batch.NodeID != upNode || batch.NodeIncarnationID != presentingInc || len(batch.Events) != 1 {
					t.Fatal("mixed stream or wrong presenter", string(body))
				}
				if batch.OriginIncarnationID == nil {
					streams, err := s.RetiredStreams(ctx)
					if err != nil || streams[0].AckedThrough != 3 || streams[0].Pending != 0 {
						t.Fatal("current sent before retired tail", streams, err)
					}
					if len(batch.Replay) != 0 || !bytes.Equal(batch.Events[0], current.Payload) {
						t.Fatal("retired gaps/events leaked into current stream")
					}
					currentPosts++
					return 200, currentResponse(), 0, nil
				}
				if *batch.OriginIncarnationID != upInc {
					t.Fatal("wrong origin")
				}
				var event api.UsageEvent
				if err := json.Unmarshal(batch.Events[0], &event); err != nil {
					t.Fatal(err)
				}
				seq := uint64(event.EventSeq)
				if seq < 2 || seq > 3 || !bytes.Equal(batch.Events[0], held[seq-2].Payload) || event.PayloadSha256 != held[seq-2].Hash {
					t.Fatal("hand-off re-minted an event")
				}
				if seq == 2 {
					if len(batch.Replay) != 1 || batch.Replay[0] != (api.ReplayUnavailableRange{StartSeq: 1, EndSeq: 1, Reason: api.ReplayUnavailableRangeReasonRetentionExpired}) {
						t.Fatal("missing origin recovery gap", batch.Replay)
					}
					reports++
				} else if len(batch.Replay) != 0 {
					t.Fatal("acknowledged gap reported again")
				}
				return 200, resultBody(event.EventSeq.String(), nil, [][2]string{{event.EventSeq.String(), "accepted"}}), 0, nil
			}
			u := newUp(t, s, tr, (&sleeper{}).Sleep)
			u.maxEv = 1
			if lostACK {
				u.hook = func(step string) error {
					if step == StepApply {
						return errors.New("lost ACK")
					}
					return nil
				}
				if err := u.Once(ctx); err == nil {
					t.Fatal("expected lost ACK")
				}
				s = reopenHandoff(t, s)
				u = newUp(t, s, tr, (&sleeper{}).Sleep)
				u.maxEv = 1
			}
			if err := u.Once(ctx); err != nil {
				t.Fatal(err)
			}
			head, err = s.Head(ctx)
			streams, streamErr := s.RetiredStreams(ctx)
			if err != nil || streamErr != nil || head.AckedThrough != 0 || streams[0].AckedThrough != 2 {
				t.Fatal("gap ACK crossed streams", head, streams, err, streamErr)
			}
			if lostACK {
				calls := tr.calls()
				if calls[0].key != calls[1].key || !bytes.Equal(calls[0].body, calls[1].body) {
					t.Fatal("lost-ACK replay changed body/key")
				}
			}
			// Restart after the gap ACK, before the remaining tail and live event.
			s = reopenHandoff(t, s)
			u = newUp(t, s, tr, (&sleeper{}).Sleep)
			u.maxEv = 1
			for range 4 {
				if err := u.Once(ctx); err != nil {
					t.Fatal(err)
				}
			}
			wantReports := 1
			if lostACK {
				wantReports++ // same hand-off retry until its ACK, not a new report
			}
			head, err = s.Head(ctx)
			if err != nil || head.NextSeq != 2 || head.AckedThrough != 1 || currentPosts != 1 || reports != wantReports || gets != 0 || u.NeedsRestore() {
				t.Fatalf("head=%+v reports=%d current=%d gets=%d restore=%v err=%v", head, reports, currentPosts, gets, u.NeedsRestore(), err)
			}
			if due, err := u.due(ctx); err != nil || due {
				t.Fatal("completed hand-off scheduled again", due, err)
			}
		})
	}
}

func TestD105_RetiredEpochMismatchNeverFetchesCheckpoint(t *testing.T) {
	for _, status := range []int{409, 200} {
		t.Run(map[int]string{409: "HTTP-mismatch", 200: "ACK-mismatch"}[status], func(t *testing.T) {
			ctx := t.Context()
			s, held := retiredRecovery(t, 3)
			currentEvent(t, s)
			gets := 0
			tr := &fakeTR{getFn: func() (int, []byte, error) { gets++; return 503, nil, nil }}
			tr.postFn = func(_ string, body []byte) (int, []byte, time.Duration, error) {
				var batch wireBatch
				if err := json.Unmarshal(body, &batch); err != nil {
					t.Fatal(err)
				}
				if batch.OriginIncarnationID == nil {
					return 200, currentResponse(), 0, nil
				}
				resp := []byte(`{"error":{"code":"LEDGER_EPOCH_MISMATCH"}}`)
				if status == 200 {
					resp = bytes.ReplaceAll(resultBody("3", nil, [][2]string{{"2", "accepted"}, {"3", "accepted"}}), []byte(epochA.String()), []byte("22222222-2222-4222-8222-222222222222"))
				}
				return status, resp, 0, nil
			}
			for range 2 {
				u := newUp(t, s, tr, (&sleeper{}).Sleep)
				if err := u.Once(ctx); !errors.Is(err, usagestore.ErrLedgerEpoch) {
					t.Fatalf("retired mismatch: %v", err)
				}
				if gets != 0 || u.NeedsRestore() {
					t.Fatal("retired mismatch initiated restore")
				}
				streams, err := s.RetiredStreams(ctx)
				if err != nil || streams[0].LedgerEpoch != epochA || streams[0].AckedThrough != 0 || streams[0].Pending != len(held) {
					t.Fatal("mismatch changed original state", streams, err)
				}
				ranges, err := s.RetiredReplayUnavailable(ctx, upInc)
				if err != nil || len(ranges) != 1 {
					t.Fatal("mismatch lost gap evidence", ranges, err)
				}
				// Current upload still gets its turn, independently of origin recovery.
				if err := u.Once(ctx); err != nil {
					// On the second pass the live event is already ACKed, so the
					// empty live turn retries the still-pending retired tail.
					if !errors.Is(err, usagestore.ErrLedgerEpoch) {
						t.Fatal(err)
					}
				}
				head, err := s.Head(ctx)
				if err != nil || head.AckedThrough != 1 || gets != 0 || u.NeedsRestore() {
					t.Fatal("current upload starved or restore scheduled", head, gets, err)
				}
				s = reopenHandoff(t, s)
			}
		})
	}
}

func TestD105_RetiredRangeWithoutEventsStaysLocal(t *testing.T) {
	ctx := t.Context()
	s, _ := retiredRecovery(t, 1)
	gets := 0
	tr := &fakeTR{getFn: func() (int, []byte, error) { gets++; return 404, nil, nil }, postFn: func(string, []byte) (int, []byte, time.Duration, error) {
		t.Fatal("revision 14 requires a non-empty events array; do not fabricate an event")
		return 0, nil, 0, nil
	}}
	for range 2 {
		u := newUp(t, s, tr, (&sleeper{}).Sleep)
		if err := u.Once(ctx); err != nil {
			t.Fatal(err)
		}
		if due, err := u.due(ctx); err != nil || due || gets != 0 || u.NeedsRestore() {
			t.Fatal("range-only origin scheduled transport", due, gets, err)
		}
		ranges, err := s.RetiredReplayUnavailable(ctx, upInc)
		if err != nil || len(ranges) != 1 {
			t.Fatal("local gap evidence lost", ranges, err)
		}
		s = reopenHandoff(t, s)
	}
}
