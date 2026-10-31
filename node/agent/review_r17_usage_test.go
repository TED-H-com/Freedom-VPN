package agent

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usageupload"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

func TestReviewR17_UsageLedgerDoesNotRemint(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.db")
	nodeID, incID := uuid.New(), uuid.New()
	s, err := usagestore.Open(path, nodeID, incID, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	dev, mem, epoch := uuid.New(), uuid.New(), uuid.New()
	if _, err = s.Commit(ctx, usagestore.Commit{
		Open: &usagestore.NewEpoch{ID: epoch, Reason: "baseline", BootID: "boot"},
		Counters: []usagestore.Counter{{
			Name: "upload", DeviceID: dev, Direction: "upload", MemberID: mem, BindingRevision: 42,
		}},
		BootID: "boot", SampledAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(ctx, usagestore.Commit{
		Epoch: epoch, SampledAt: at.Add(10 * time.Second), BootID: "boot",
		Counters: []usagestore.Counter{{
			Name: "upload", DeviceID: dev, Direction: "upload", Packets: 1, Bytes: 100, MemberID: mem, BindingRevision: 42,
		}},
		Event: &usagestore.Draft{
			EpochID: epoch, Start: at, End: at.Add(10 * time.Second), Quality: "complete",
			Records: []usagestore.DraftRecord{{
				MemberID: mem, DeviceID: dev, BindingRevision: 42, PeriodID: "2026-10", Upload: 100,
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	next := uuid.New()
	if _, err = usagestore.Open(path, nodeID, next, usagestore.Options{}); !errors.Is(err, usagestore.ErrIdentity) {
		t.Fatalf("incarnation switch without retirement: %v", err)
	}
	if _, err = usagestore.Open(path, uuid.New(), next, usagestore.Options{RetirePreviousAt: at}); !errors.Is(err, usagestore.ErrIdentity) {
		t.Fatalf("foreign node with a retirement time: %v", err)
	}
	stepped := at.Add(-24 * time.Hour)
	st, err := usagestore.Open(path, nodeID, next, usagestore.Options{RetirePreviousAt: stepped})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	streams, err := st.RetiredStreams(ctx)
	if err != nil || len(streams) != 1 || streams[0].IncarnationID != incID || streams[0].LastSeq != 1 || !streams[0].RetiredAt.Equal(stepped) {
		t.Fatalf("retired stream %+v %v", streams, err)
	}
	events, err := st.RetiredUnacked(ctx, incID, 10, 1<<20)
	if err != nil || len(events) != 1 || events[0].Seq != 1 {
		t.Fatalf("tail was re-minted: %+v %v", events, err)
	}
	head, err := st.Head(ctx)
	if err != nil || head.NextSeq != 1 || head.AckedThrough != 0 || head.HasHistory {
		t.Fatalf("new stream continued the old cursor: %+v %v", head, err)
	}
	if err = st.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err = usagestore.Open(path, nodeID, incID, usagestore.Options{RetirePreviousAt: at}); !errors.Is(err, usagestore.ErrIdentity) {
		t.Fatalf("retired incarnation was reused: %v", err)
	}
}

func TestReviewR17_TwoUsageStoresDoNotDoubleCommit(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "usage.db")
	nodeID, incID := uuid.New(), uuid.New()
	open := func() *usagestore.Store {
		t.Helper()
		s, err := usagestore.Open(path, nodeID, incID, usagestore.Options{})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	a, b := open(), open()
	t.Cleanup(func() { _ = a.Close(); _ = b.Close() })
	epoch := uuid.New()
	dev, mem := uuid.New(), uuid.New()
	at := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	commit := usagestore.Commit{
		Open: &usagestore.NewEpoch{ID: epoch, Reason: "baseline", BootID: "boot"},
		Counters: []usagestore.Counter{{
			Name: "upload", DeviceID: dev, Direction: "upload", MemberID: mem, BindingRevision: 1,
		}},
		BootID: "boot", SampledAt: at,
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, s := range []*usagestore.Store{a, b} {
		wg.Add(1)
		go func(i int, s *usagestore.Store) {
			defer wg.Done()
			_, errs[i] = s.Commit(ctx, commit)
		}(i, s)
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		}
	}
	if ok > 1 {
		t.Fatal("TODO(review): two stores sharing one usage.db both committed the same epoch")
	}
	if ok == 0 {
		t.Fatalf("neither writer committed: %v", errs)
	}
}

type scriptUsageTransport struct {
	mu     sync.Mutex
	keys   []string
	bodies [][]byte
	reply  func(n int, key string, body []byte) (int, []byte, error)
}

func (s *scriptUsageTransport) PostUsageBatch(_ context.Context, key string, body []byte) (int, []byte, time.Duration, error) {
	s.mu.Lock()
	s.keys = append(s.keys, key)
	s.bodies = append(s.bodies, append([]byte(nil), body...))
	n := len(s.keys)
	s.mu.Unlock()
	status, resp, err := s.reply(n, key, body)
	return status, resp, 0, err
}

func (s *scriptUsageTransport) GetUsageCheckpoint(context.Context) (int, []byte, error) {
	return 404, []byte(`{"error":{"code":"RESOURCE_NOT_FOUND","message":"none","details":{},"request_id":"00000000-0000-7000-8000-000000000001","retryable":false}}`), nil
}

func TestReviewR17_RetiredUploadReplayKeepsTheOriginACK(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "usage.db")
	nodeID, incID, next := uuid.New(), uuid.New(), uuid.New()
	s, err := usagestore.Open(path, nodeID, incID, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	dev, mem, epoch := uuid.New(), uuid.New(), uuid.New()
	if _, err = s.Commit(ctx, usagestore.Commit{
		Open:     &usagestore.NewEpoch{ID: epoch, Reason: "baseline", BootID: "boot"},
		Counters: []usagestore.Counter{{Name: "upload", DeviceID: dev, Direction: "upload", MemberID: mem, BindingRevision: 1}},
		BootID:   "boot", SampledAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(ctx, usagestore.Commit{
		Epoch: epoch, SampledAt: at.Add(10 * time.Second), BootID: "boot",
		Counters: []usagestore.Counter{{Name: "upload", DeviceID: dev, Direction: "upload", Packets: 1, Bytes: 40, MemberID: mem, BindingRevision: 1}},
		Event: &usagestore.Draft{EpochID: epoch, Start: at, End: at.Add(10 * time.Second), Quality: "complete", Records: []usagestore.DraftRecord{{
			MemberID: mem, DeviceID: dev, BindingRevision: 1, PeriodID: "2026-10", Upload: 40,
		}}},
	}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := usagestore.Open(path, nodeID, next, usagestore.Options{RetirePreviousAt: at.Add(10 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	tr := &scriptUsageTransport{reply: func(int, string, []byte) (int, []byte, error) {
		return 0, nil, errors.New("connection reset")
	}}
	up, err := usageupload.New(usageupload.Config{Store: st, Transport: tr, Interval: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err = up.Once(ctx); err == nil {
		t.Fatal("transport failure was ignored")
	}
	ledger := uuid.New()
	tr.reply = func(n int, key string, body []byte) (int, []byte, error) {
		if n == 1 {
			return 0, nil, errors.New("connection reset before the response was stored")
		}
		return 200, reviewUsageACK(nodeID, incID, ledger), nil
	}
	// The first call above already posted once. Reset the script and drive
	// the crash window again: post, lose the response, then the stored key.
	tr2 := &scriptUsageTransport{}
	var posted int
	tr2.reply = func(int, string, []byte) (int, []byte, error) {
		posted++
		if posted == 1 {
			return 0, nil, errors.New("lost response")
		}
		return 200, reviewUsageACK(nodeID, incID, ledger), nil
	}
	up, err = usageupload.New(usageupload.Config{Store: st, Transport: tr2, Interval: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err = up.Once(ctx); err == nil {
		t.Fatal("lost response was treated as an ACK")
	}
	if err = up.Once(ctx); err != nil {
		t.Fatal(err)
	}
	if len(tr2.keys) != 2 || tr2.keys[0] != tr2.keys[1] || string(tr2.bodies[0]) != string(tr2.bodies[1]) {
		t.Fatalf("replay key/body changed: %v", tr2.keys)
	}
	streams, err := st.RetiredStreams(ctx)
	if err != nil || streams[0].AckedThrough != 1 || streams[0].IncarnationID != incID {
		t.Fatalf("origin ACK was not applied: %+v %v", streams, err)
	}
	head, err := st.Head(ctx)
	if err != nil || head.AckedThrough != 0 {
		t.Fatalf("origin ACK moved the current stream: %+v %v", head, err)
	}

	// A permanent validation failure retries. A state conflict retires the tail.
	fresh := retireOneEvent(t)
	bad := &scriptUsageTransport{reply: func(int, string, []byte) (int, []byte, error) {
		return 422, []byte(`{"error":{"code":"VALIDATION_FAILED","message":"no","details":{},"request_id":"00000000-0000-7000-8000-000000000002","retryable":false}}`), nil
	}}
	up, err = usageupload.New(usageupload.Config{Store: fresh.store, Transport: bad, Interval: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err = up.Once(ctx); err == nil {
		t.Fatal("422 retired or succeeded")
	}
	left, err := fresh.store.RetiredUnacked(ctx, fresh.origin, 10, 1<<20)
	if err != nil || len(left) != 1 {
		t.Fatalf("422 retired the tail: %+v %v", left, err)
	}
	conflict := &scriptUsageTransport{reply: func(int, string, []byte) (int, []byte, error) {
		return 409, []byte(`{"error":{"code":"` + string(apierr.StateConflict) + `","message":"no","details":{},"request_id":"00000000-0000-7000-8000-000000000003","retryable":false}}`), nil
	}}
	up, err = usageupload.New(usageupload.Config{Store: fresh.store, Transport: conflict, Interval: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err = up.Once(ctx); err != nil {
		t.Fatal(err)
	}
	streams, err = fresh.store.RetiredStreams(ctx)
	if err != nil || streams[0].FinalCode != string(apierr.StateConflict) {
		t.Fatalf("state conflict did not retire the tail: %+v %v", streams, err)
	}
	if err = up.Once(ctx); err != nil {
		t.Fatal(err)
	}
	if len(conflict.keys) != 1 {
		t.Fatalf("retired tail was uploaded again: %d", len(conflict.keys))
	}
}

func reviewUsageACK(nodeID, incID, ledger uuid.UUID) []byte {
	res := api.UsageBatchResult{
		NodeId: nodeID, NodeIncarnationId: incID, LedgerEpoch: ledger,
		HighestContiguousSeq: 1,
		Results: []api.UsageEventResult{{
			EventSeq: 1, Status: api.UsageEventResultStatusAccepted,
		}},
	}
	res.BlockedSeq.SetNull()
	res.Results[0].Error.SetNull()
	raw, err := json.Marshal(res)
	if err != nil {
		panic(err)
	}
	return raw
}

type retiredFixture struct {
	store  *usagestore.Store
	origin uuid.UUID
}

func retireOneEvent(t *testing.T) retiredFixture {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "usage.db")
	nodeID, incID, next := uuid.New(), uuid.New(), uuid.New()
	s, err := usagestore.Open(path, nodeID, incID, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	dev, mem, epoch := uuid.New(), uuid.New(), uuid.New()
	if _, err = s.Commit(ctx, usagestore.Commit{
		Open:     &usagestore.NewEpoch{ID: epoch, Reason: "baseline", BootID: "boot"},
		Counters: []usagestore.Counter{{Name: "upload", DeviceID: dev, Direction: "upload", MemberID: mem, BindingRevision: 1}},
		BootID:   "boot", SampledAt: at,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Commit(ctx, usagestore.Commit{
		Epoch: epoch, SampledAt: at.Add(10 * time.Second), BootID: "boot",
		Counters: []usagestore.Counter{{Name: "upload", DeviceID: dev, Direction: "upload", Packets: 1, Bytes: 8, MemberID: mem, BindingRevision: 1}},
		Event: &usagestore.Draft{EpochID: epoch, Start: at, End: at.Add(10 * time.Second), Quality: "complete", Records: []usagestore.DraftRecord{{
			MemberID: mem, DeviceID: dev, BindingRevision: 1, PeriodID: "2026-10", Upload: 8,
		}}},
	}); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := usagestore.Open(path, nodeID, next, usagestore.Options{RetirePreviousAt: at.Add(10 * time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return retiredFixture{store: st, origin: incID}
}
