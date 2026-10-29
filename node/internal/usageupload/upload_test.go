package usageupload

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

var (
	upNode = uuid.MustParse("93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb")
	upInc  = uuid.MustParse("d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0")
	upDev  = uuid.MustParse("2df6c7df-e7e8-4bd4-9f2c-a5836dd0d8ea")
	upMem  = uuid.MustParse("dc01a74b-9e6f-41ac-81cd-03291d0a88c0")
	epochA = uuid.MustParse("7612b7d9-1e70-4390-b86b-7c3456e4df27")
)

type fakeTR struct {
	mu     sync.Mutex
	posts  []postCall
	postFn func(key string, body []byte) (int, []byte, time.Duration, error)
	getFn  func() (int, []byte, error)
}

type postCall struct {
	key  string
	body []byte
}

func (f *fakeTR) PostUsageBatch(ctx context.Context, key string, body []byte) (int, []byte, time.Duration, error) {
	f.mu.Lock()
	f.posts = append(f.posts, postCall{key: key, body: append([]byte(nil), body...)})
	fn := f.postFn
	f.mu.Unlock()
	return fn(key, body)
}

func (f *fakeTR) GetUsageCheckpoint(ctx context.Context) (int, []byte, error) {
	return f.getFn()
}

func (f *fakeTR) calls() []postCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]postCall, len(f.posts))
	copy(out, f.posts)
	return out
}

type sleeper struct {
	mu sync.Mutex
	ds []time.Duration
}

func (s *sleeper) Sleep(ctx context.Context, d time.Duration) error {
	s.mu.Lock()
	s.ds = append(s.ds, d)
	s.mu.Unlock()
	return nil
}

func seed(t *testing.T, n int, opt usagestore.Options) *usagestore.Store {
	t.Helper()
	s, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), upNode, upInc, opt)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	id := uuid.Must(uuid.NewV7())
	if _, err := s.Commit(context.Background(), usagestore.Commit{
		Counters:  []usagestore.Counter{upCounter(0)},
		Open:      &usagestore.NewEpoch{ID: id, Reason: "baseline", BootID: "boot"},
		BootID:    "boot",
		SampledAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	var from uint64
	for i := 1; i <= n; i++ {
		to := from + 10
		end := at.Add(time.Duration(i) * 10 * time.Second)
		if _, err := s.Commit(context.Background(), usagestore.Commit{
			Epoch: id, Counters: []usagestore.Counter{upCounter(to)}, BootID: "boot", SampledAt: end,
			Event: &usagestore.Draft{
				EpochID: id, Start: end.Add(-10 * time.Second), End: end, Quality: "complete",
				Records: []usagestore.DraftRecord{{
					MemberID: upMem, DeviceID: upDev, BindingRevision: 7,
					PeriodID: "2026-10", Upload: to - from,
				}},
			},
		}); err != nil {
			t.Fatal(err)
		}
		from = to
	}
	return s
}

func upCounter(bytes uint64) usagestore.Counter {
	return usagestore.Counter{
		Name:     "u_" + strings.ReplaceAll(upDev.String(), "-", ""),
		DeviceID: upDev, Direction: "upload", Packets: 1, Bytes: bytes,
		MemberID: upMem, BindingRevision: 7,
	}
}

func resultBody(highest string, blocked *string, results [][2]string) []byte {
	b := "null"
	if blocked != nil {
		b = `"` + *blocked + `"`
	}
	var rows []string
	for _, r := range results {
		rows = append(rows, `{"event_seq":"`+r[0]+`","status":"`+r[1]+`","error":null}`)
	}
	return []byte(`{
		"blocked_seq": ` + b + `,
		"highest_contiguous_seq": "` + highest + `",
		"ledger_epoch": "` + epochA.String() + `",
		"node_id": "` + upNode.String() + `",
		"node_incarnation_id": "` + upInc.String() + `",
		"received_at": "2026-10-28T15:00:00Z",
		"results": [` + strings.Join(rows, ",") + `]
	}`)
}

func newUp(t *testing.T, s *usagestore.Store, tr Transport, sleep Sleeper) *Uploader {
	t.Helper()
	u, err := New(Config{Store: s, Transport: tr, Sleep: sleep, Interval: time.Minute, Rand: func() float64 { return 0.5 }})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestAT15_ContiguousAck(t *testing.T) {
	ctx := context.Background()
	t.Run("later seq does not skip a hole", func(t *testing.T) {
		s := seed(t, 3, usagestore.Options{})
		saved, err := s.Event(ctx, 3)
		if err != nil {
			t.Fatal(err)
		}
		round := 0
		tr := &fakeTR{postFn: func(key string, body []byte) (int, []byte, time.Duration, error) {
			round++
			if round == 1 {
				// The server claims 3 is contiguous but never accepted 2.
				return 200, resultBody("3", nil, [][2]string{{"1", "accepted"}, {"3", "accepted"}}), 0, nil
			}
			return 200, resultBody("3", nil, [][2]string{{"2", "accepted"}, {"3", "duplicate"}}), 0, nil
		}}
		u := newUp(t, s, tr, (&sleeper{}).Sleep)
		if err := u.Once(ctx); err != nil {
			t.Fatal(err)
		}
		h, err := s.Head(ctx)
		if err != nil || h.AckedThrough != 1 {
			t.Fatalf("acked %+v %v", h, err)
		}
		for _, seq := range []uint64{2, 3} {
			ev, err := s.Event(ctx, seq)
			if err != nil || ev.State != "outbox" {
				t.Fatalf("seq %d %+v %v", seq, ev, err)
			}
		}
		if err := u.Once(ctx); err != nil {
			t.Fatal(err)
		}
		h, _ = s.Head(ctx)
		if h.AckedThrough != 3 {
			t.Fatalf("acked %d", h.AckedThrough)
		}
		ev, err := s.Event(ctx, 3)
		if err != nil || ev.State != "archived" || string(ev.Payload) != string(saved.Payload) {
			t.Fatal("resend rebuilt the payload")
		}
		calls := tr.calls()
		if len(calls) != 2 || !strings.Contains(string(calls[1].body), string(saved.Payload)) {
			t.Fatal("second batch did not resend the stored event")
		}
	})

	t.Run("lost ack retries the same bytes and key", func(t *testing.T) {
		s := seed(t, 1, usagestore.Options{})
		n := 0
		tr := &fakeTR{postFn: func(key string, body []byte) (int, []byte, time.Duration, error) {
			n++
			if n == 1 {
				return 0, nil, 0, errors.New("connection reset")
			}
			return 200, resultBody("1", nil, [][2]string{{"1", "duplicate"}}), 0, nil
		}}
		u := newUp(t, s, tr, (&sleeper{}).Sleep)
		if err := u.Once(ctx); err == nil {
			t.Fatal("dropped response was treated as success")
		}
		if got, _ := s.AckedThrough(ctx); got != 0 {
			t.Fatalf("acked %d", got)
		}
		if err := u.Once(ctx); err != nil {
			t.Fatal(err)
		}
		calls := tr.calls()
		if len(calls) != 2 || calls[0].key != calls[1].key || string(calls[0].body) != string(calls[1].body) {
			t.Fatalf("retry changed identity %s %s", calls[0].key, calls[1].key)
		}
		if _, err := uuid.Parse(calls[0].key); err != nil {
			t.Fatal(err)
		}
		ev, _ := s.Event(ctx, 1)
		if ev.State != "archived" {
			t.Fatalf("state %s", ev.State)
		}
		out, arch, _ := s.CountEvents(ctx)
		if out != 0 || arch != 1 {
			t.Fatalf("out %d arch %d", out, arch)
		}
	})

	t.Run("hash conflict blocks and keeps the payload", func(t *testing.T) {
		s := seed(t, 2, usagestore.Options{})
		before, _ := s.Event(ctx, 1)
		tr := &fakeTR{postFn: func(string, []byte) (int, []byte, time.Duration, error) {
			return 200, resultBody("0", strPtr("1"), [][2]string{{"1", "conflict"}, {"2", "accepted"}}), 0, nil
		}}
		u := newUp(t, s, tr, (&sleeper{}).Sleep)
		if err := u.Once(ctx); err != nil {
			t.Fatal(err)
		}
		h, _ := s.Head(ctx)
		if h.AckedThrough != 0 || h.BlockedSeq == nil || *h.BlockedSeq != 1 {
			t.Fatalf("%+v", h)
		}
		ev, _ := s.Event(ctx, 1)
		ev2, _ := s.Event(ctx, 2)
		if ev.State != "outbox" || string(ev.Payload) != string(before.Payload) || ev2.State != "outbox" {
			t.Fatalf("conflict dropped or archived later seqs")
		}
	})

	t.Run("409 conflict", func(t *testing.T) {
		s := seed(t, 1, usagestore.Options{})
		tr := &fakeTR{postFn: func(string, []byte) (int, []byte, time.Duration, error) {
			return 409, []byte(`{"error":{"code":"USAGE_EVENT_CONFLICT","message":"hash","request_id":"00000000-0000-0000-0000-000000000010","retryable":false,"details":{}}}`), 0, nil
		}}
		u := newUp(t, s, tr, (&sleeper{}).Sleep)
		if err := u.Once(ctx); err == nil {
			t.Fatal("conflict returned success")
		}
		h, _ := s.Head(ctx)
		if h.AckedThrough != 0 || h.BlockedSeq == nil || *h.BlockedSeq != 1 {
			t.Fatalf("%+v", h)
		}
	})

	t.Run("503 does not advance", func(t *testing.T) {
		s := seed(t, 1, usagestore.Options{})
		n := 0
		tr := &fakeTR{postFn: func(string, []byte) (int, []byte, time.Duration, error) {
			n++
			if n == 1 {
				return 503, []byte(`{"error":{"code":"LEDGER_FROZEN","message":"frozen","request_id":"00000000-0000-0000-0000-000000000010","retryable":true,"details":{}}}`), 0, nil
			}
			return 200, resultBody("1", nil, [][2]string{{"1", "accepted"}}), 0, nil
		}}
		u := newUp(t, s, tr, (&sleeper{}).Sleep)
		if err := u.Once(ctx); err == nil {
			t.Fatal("503 succeeded")
		}
		if got, _ := s.AckedThrough(ctx); got != 0 {
			t.Fatalf("acked %d", got)
		}
		if err := u.Once(ctx); err != nil {
			t.Fatal(err)
		}
		if got, _ := s.AckedThrough(ctx); got != 1 {
			t.Fatalf("acked %d", got)
		}
	})

	t.Run("throttle", func(t *testing.T) {
		s := seed(t, 1, usagestore.Options{})
		sl := &sleeper{}
		tr := &fakeTR{postFn: func(string, []byte) (int, []byte, time.Duration, error) {
			return 429, []byte(`{"error":{"code":"RATE_LIMITED","message":"slow","request_id":"00000000-0000-0000-0000-000000000010","retryable":true,"details":{}}}`), 2 * time.Second, nil
		}}
		u := newUp(t, s, tr, sl.Sleep)
		if err := u.Once(ctx); err == nil {
			t.Fatal("429 succeeded")
		}
		sl.mu.Lock()
		defer sl.mu.Unlock()
		if len(sl.ds) != 1 || sl.ds[0] < 2*time.Second {
			t.Fatalf("sleep %v", sl.ds)
		}
		if got, _ := s.AckedThrough(ctx); got != 0 {
			t.Fatalf("acked %d", got)
		}
	})
}

func TestEpochMismatchFetchesCheckpoint(t *testing.T) {
	ctx := context.Background()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := envelope.NewNodeTrustStore(envelope.TrustedKey{
		KeyID: "recovery-test", PublicKey: pub,
		Purposes:  []envelope.Purpose{envelope.PurposeRecovery},
		NotBefore: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 28, 18, 0, 0, 0, time.UTC)
	s := seed(t, 1, usagestore.Options{Trust: trust, Now: func() time.Time { return now }})
	cp := api.UsageCheckpoint{
		LedgerEpoch: epochA, IssuedAt: now,
		HighestContiguousSeq: 0, MaxSeenSeq: 0,
		MissingRanges:    []api.SeqRange{},
		RecoveryRecordId: presence.NewNullNullable[uuid.UUID](),
		NodeId:           upNode, NodeIncarnationId: upInc,
	}
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
	other := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	cp.LedgerEpoch = other
	cp.IssuedAt = now.Add(time.Minute)
	env, err = envelope.Sign(cp, priv, "recovery-test")
	if err != nil {
		t.Fatal(err)
	}
	next, err := env.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	tr := &fakeTR{
		postFn: func(string, []byte) (int, []byte, time.Duration, error) {
			return 409, []byte(`{"error":{"code":"LEDGER_EPOCH_MISMATCH","message":"epoch","request_id":"00000000-0000-0000-0000-000000000010","retryable":false,"details":{"current_ledger_epoch":"` + other.String() + `"}}}`), 0, nil
		},
		getFn: func() (int, []byte, error) { return 200, next, nil },
	}
	u := newUp(t, s, tr, (&sleeper{}).Sleep)
	if err := u.Once(ctx); err != nil {
		t.Fatal(err)
	}
	h, err := s.Head(ctx)
	if err != nil || h.LedgerEpoch != other || h.AckedThrough != 0 {
		t.Fatalf("%+v %v", h, err)
	}
	ev, err := s.Event(ctx, 1)
	if err != nil || ev.State != "outbox" {
		t.Fatal("epoch fetch archived the event")
	}
	epochs, err := s.LedgerEpochs(ctx)
	if err != nil || len(epochs) != 2 {
		t.Fatalf("epochs %v %v", epochs, err)
	}
}

func TestBatchPreservesDecimalPayload(t *testing.T) {
	ctx := context.Background()
	s, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), upNode, upInc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	id := uuid.Must(uuid.NewV7())
	const big = uint64(9007199254740993)
	if _, err := s.Commit(ctx, usagestore.Commit{
		Counters: []usagestore.Counter{upCounter(0)},
		Open:     &usagestore.NewEpoch{ID: id, Reason: "baseline", BootID: "boot"},
		BootID:   "boot", SampledAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	c := upCounter(big)
	c.Packets = 4
	if _, err := s.Commit(ctx, usagestore.Commit{
		Epoch: id, Counters: []usagestore.Counter{c}, BootID: "boot", SampledAt: at.Add(10 * time.Second),
		Event: &usagestore.Draft{
			EpochID: id, Start: at, End: at.Add(10 * time.Second), Quality: "complete",
			Records: []usagestore.DraftRecord{{
				MemberID: upMem, DeviceID: upDev, BindingRevision: 7,
				PeriodID: "2026-10", Upload: big,
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	var posted []byte
	tr := &fakeTR{postFn: func(key string, body []byte) (int, []byte, time.Duration, error) {
		posted = append([]byte(nil), body...)
		var batch api.UsageBatchRequest
		if err := json.Unmarshal(body, &batch); err != nil {
			t.Errorf("batch decode: %v", err)
		}
		return 200, resultBody("1", nil, [][2]string{{"1", "accepted"}}), 0, nil
	}}
	u := newUp(t, s, tr, (&sleeper{}).Sleep)
	if err := u.Once(ctx); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(posted), `"upload_bytes":"9007199254740993"`) {
		t.Fatalf("body %s", posted)
	}
	var batch api.UsageBatchRequest
	if err := json.Unmarshal(posted, &batch); err != nil {
		t.Fatal(err)
	}
	if !batch.LedgerEpoch.IsNull() {
		t.Fatal("ledger epoch was not null before a checkpoint")
	}
	if !strings.Contains(string(posted), decstr.U64(big).String()) {
		t.Fatal("batch lost the decimal")
	}
	if len(batch.Events) != 1 || uint64(batch.Events[0].Records[0].UploadBytes) != big {
		t.Fatalf("decoded upload %v", batch.Events)
	}
	ev, _ := s.Event(ctx, 1)
	if !strings.Contains(string(posted), string(ev.Payload)) {
		t.Fatal("batch did not embed the stored payload")
	}
}

func TestRestoreSchedulesAndClears(t *testing.T) {
	ctx := context.Background()
	s := seed(t, 0, usagestore.Options{})
	tr := &fakeTR{
		postFn: func(string, []byte) (int, []byte, time.Duration, error) {
			return 500, nil, 0, nil
		},
		getFn: func() (int, []byte, error) {
			return 503, []byte("no"), nil
		},
	}
	u := newUp(t, s, tr, (&sleeper{}).Sleep)
	if err := u.Restore(ctx); err == nil {
		t.Fatal("503 restore succeeded")
	}
	if !u.NeedsRestore() {
		t.Fatal("failed restore did not ask for another")
	}
	tr.getFn = func() (int, []byte, error) { return 404, nil, nil }
	if err := u.Restore(ctx); err != nil {
		t.Fatal(err)
	}
	if u.NeedsRestore() {
		t.Fatal("404 left a restore pending")
	}
	h, err := s.Head(ctx)
	if err != nil || h.LedgerEpoch != uuid.Nil {
		t.Fatalf("404 stored an epoch: %+v %v", h, err)
	}
}

func strPtr(s string) *string { return &s }
