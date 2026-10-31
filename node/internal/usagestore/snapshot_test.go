package usagestore

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

func TestSnapshotFieldSetAndIdentical(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	zero := []Counter{
		counterAt(testDev, testMem, "upload", 0, 0, 7),
		counterAt(testDev2, testMem2, "download", 0, 0, 9),
	}
	epoch := openEpoch(t, s, zero, at)
	const big = uint64(9007199254740993)
	next := []Counter{
		counterAt(testDev, testMem, "upload", 2, big, 7),
		counterAt(testDev2, testMem2, "download", 3, 20, 9),
	}
	seq, err := s.Commit(ctx, Commit{
		Epoch: epoch, Counters: next, BootID: "boot-1", SampledAt: at.Add(10 * time.Second),
		Event: &Draft{
			EpochID: epoch, Start: at, End: at.Add(10 * time.Second), Quality: "complete",
			Records: []DraftRecord{
				{MemberID: testMem, DeviceID: testDev, BindingRevision: 7, PeriodID: "2026-10", Upload: big},
				{MemberID: testMem2, DeviceID: testDev2, BindingRevision: 9, PeriodID: "2026-10", Download: 20},
			},
		},
	})
	if err != nil || seq != 1 {
		t.Fatalf("seq %d %v", seq, err)
	}
	if _, err := s.Commit(ctx, Commit{
		Epoch: epoch, CloseEpoch: true, CloseReason: "handoff", Counters: next,
		BootID: "boot-1", SampledAt: at.Add(20 * time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	epoch2 := uuid.Must(uuid.NewV7())
	if _, err := s.Commit(ctx, Commit{
		Counters: next, Open: &NewEpoch{ID: epoch2, Reason: "baseline", BootID: "boot-1"},
		BootID: "boot-1", SampledAt: at.Add(20 * time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	before, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	mode := mustMode(t, s.Path())
	first, err := ReadSnapshot(ctx, s.Path())
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReadSnapshot(ctx, s.Path())
	if err != nil {
		t.Fatal(err)
	}
	j1, err := first.JSON()
	if err != nil {
		t.Fatal(err)
	}
	j2, err := second.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(j1, j2) || first.Text() != second.Text() {
		t.Fatalf("snapshots diverged\n%s\n%s", j1, j2)
	}
	after, err := os.ReadFile(s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("snapshot wrote the database file")
	}
	if mustMode(t, s.Path()) != mode {
		t.Fatal("snapshot changed the file mode")
	}
	if _, err := os.Stat(s.Path() + "-missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}

	if first.SchemaVersion != 1 || first.NodeID != testNode.String() || first.NodeIncarnationID != testInc.String() {
		t.Fatalf("identity %+v", first)
	}
	if first.LedgerEpoch != nil || first.LastEventSeq != "1" || first.AckedThrough != "0" || first.UnackedCount != 1 {
		t.Fatalf("cursor %+v", first)
	}
	if len(first.MeterEpochs) != 2 || first.MeterEpochs[0].MeterEpochID != epoch.String() || first.MeterEpochs[0].State != "closed" {
		t.Fatalf("epochs %+v", first.MeterEpochs)
	}
	if first.MeterEpochs[1].MeterEpochID != epoch2.String() || first.MeterEpochs[1].State != "current" {
		t.Fatalf("current %+v", first.MeterEpochs)
	}
	if len(first.UnackedEvents) != 1 {
		t.Fatalf("events %+v", first.UnackedEvents)
	}
	ev := first.UnackedEvents[0]
	stored, err := s.Event(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := canonjson.HashHex(stored.Payload, "payload_sha256")
	if err != nil || ev.PayloadSHA256 != sum || ev.PayloadSHA256 != stored.Hash {
		t.Fatalf("hash snap %s stored %s canon %s err %v", ev.PayloadSHA256, stored.Hash, sum, err)
	}
	if ev.EventSeq != "1" || len(ev.PeriodIDs) != 1 || ev.PeriodIDs[0] != "2026-10" || len(ev.Records) != 2 {
		t.Fatalf("event %+v", ev)
	}
	if ev.Records[0].MemberID != testMem.String() || ev.Records[0].DeviceID != testDev.String() ||
		ev.Records[0].BindingRevision != "7" || ev.Records[0].UploadBytes != "9007199254740993" || ev.Records[0].DownloadBytes != "0" {
		t.Fatalf("record0 %+v", ev.Records[0])
	}
	if ev.Records[1].MemberID != testMem2.String() || ev.Records[1].DeviceID != testDev2.String() ||
		ev.Records[1].UploadBytes != "0" || ev.Records[1].DownloadBytes != "20" {
		t.Fatalf("record1 %+v", ev.Records[1])
	}
	raw := string(j1)
	if !bytes.Contains(j1, []byte(`"upload_bytes":"9007199254740993"`)) || bytes.Contains(j1, []byte(`"last_event_seq":1`)) {
		t.Fatalf("decimal was not a string: %s", raw)
	}
	if !bytes.Contains(j1, []byte(`"unacked_payload_bytes":"`+decstr.U64(uint64(len(stored.Payload))).String()+`"`)) {
		t.Fatalf("byte total %s payload %d", raw, len(stored.Payload))
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(j1), &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"schema_version", "node_id", "node_incarnation_id", "ledger_epoch", "meter_epochs",
		"last_event_seq", "acked_through", "unacked_count", "unacked_payload_bytes", "unacked_events",
	} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing %s in %s", key, raw)
		}
	}
	if string(decoded["ledger_epoch"]) != "null" {
		t.Fatalf("ledger_epoch %s", decoded["ledger_epoch"])
	}
	if !bytes.Contains(j1, []byte("\n")) || j1[len(j1)-1] != '\n' {
		t.Fatal("json does not end with one newline")
	}
}

func TestSnapshotNotTornDuringACK(t *testing.T) {
	ctx := context.Background()
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	closeRelease := func() { once.Do(func() { close(release) }) }
	t.Cleanup(closeRelease)
	s := openTest(t, Options{Hook: func(step string) error {
		if step == StepACKBeforeCommit {
			close(entered)
			<-release
		}
		return nil
	}})
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 0, 0, 7)}, at)
	for i := 1; i <= 3; i++ {
		counters := []Counter{counterAt(testDev, testMem, "upload", uint64(i), uint64(i*10), 7)}
		end := at.Add(time.Duration(i) * 10 * time.Second)
		if _, err := s.Commit(ctx, Commit{
			Epoch: epoch, Counters: counters, BootID: "boot-1", SampledAt: end,
			Event: &Draft{
				EpochID: epoch, Start: end.Add(-10 * time.Second), End: end, Quality: "complete",
				Records: []DraftRecord{{
					MemberID: testMem, DeviceID: testDev, BindingRevision: 7,
					PeriodID: "2026-10", Upload: 10,
				}},
			},
		}); err != nil {
			t.Fatal(err)
		}
	}
	before, err := ReadSnapshot(ctx, s.Path())
	if err != nil {
		t.Fatal(err)
	}
	beforeJSON, err := before.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if before.AckedThrough != "0" || before.UnackedCount != 3 || before.LastEventSeq != "3" {
		t.Fatalf("before %+v", before)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.ApplyACK(ctx, BatchACK{
			Highest: 3,
			Results: []EventResult{
				{Seq: 1, Status: StatusAccepted},
				{Seq: 2, Status: StatusAccepted},
				{Seq: 3, Status: StatusAccepted},
			},
		})
	}()
	select {
	case <-entered:
	case err := <-errCh:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("ack did not reach the commit boundary")
	}
	mid, err := ReadSnapshot(ctx, s.Path())
	if err != nil {
		closeRelease()
		t.Fatal(err)
	}
	midJSON, err := mid.JSON()
	if err != nil {
		closeRelease()
		t.Fatal(err)
	}
	if !bytes.Equal(beforeJSON, midJSON) {
		closeRelease()
		t.Fatalf("torn snapshot\nbefore %s\nmid %s", beforeJSON, midJSON)
	}
	closeRelease()
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	after, err := ReadSnapshot(ctx, s.Path())
	if err != nil {
		t.Fatal(err)
	}
	if after.AckedThrough != "3" || after.UnackedCount != 0 || len(after.UnackedEvents) != 0 || after.LastEventSeq != "3" {
		t.Fatalf("after %+v", after)
	}
	if after.SchemaVersion != before.SchemaVersion || len(after.MeterEpochs) != len(before.MeterEpochs) {
		t.Fatalf("epochs changed %+v %+v", before.MeterEpochs, after.MeterEpochs)
	}
	// A view is never half-applied: the new watermark has an empty outbox,
	// and the old watermark still listed every event.
	if (mid.AckedThrough == "3") != (mid.UnackedCount == 0) {
		t.Fatalf("torn fields %+v", mid)
	}
}

func TestSnapshotAbsentAndCorrupt(t *testing.T) {
	ctx := context.Background()
	missing := filepath.Join(t.TempDir(), "missing", "usage.db")
	if _, err := ReadSnapshot(ctx, missing); !errors.Is(err, ErrAbsent) {
		t.Fatalf("absent %v", err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("absent snapshot created a file")
	}
	garbage := filepath.Join(t.TempDir(), "usage.db")
	if err := os.WriteFile(garbage, []byte("not a database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSnapshot(ctx, garbage); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("garbage %v", err)
	}

	s := openTest(t, Options{})
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 0, 0, 7)}, at)
	if _, err := s.Commit(ctx, Commit{
		Epoch: epoch, Counters: []Counter{counterAt(testDev, testMem, "upload", 1, 5, 7)},
		BootID: "boot-1", SampledAt: at.Add(10 * time.Second),
		Event: &Draft{
			EpochID: epoch, Start: at, End: at.Add(10 * time.Second), Quality: "complete",
			Records: []DraftRecord{{
				MemberID: testMem, DeviceID: testDev, BindingRevision: 7, PeriodID: "2026-10", Upload: 5,
			}},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE events SET payload_sha256 = ? WHERE seq = 1`, "0000000000000000000000000000000000000000000000000000000000000000"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadSnapshot(ctx, s.Path()); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("tampered hash %v", err)
	}
}

func TestSnapshotLocked(t *testing.T) {
	ctx := context.Background()
	s := openTest(t, Options{})
	path := s.Path()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	holder, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = holder.Close() })
	holder.SetMaxOpenConns(1)
	if _, err := holder.ExecContext(ctx, `PRAGMA locking_mode=EXCLUSIVE`); err != nil {
		t.Fatal(err)
	}
	if _, err := holder.ExecContext(ctx, `BEGIN EXCLUSIVE`); err != nil {
		t.Fatal(err)
	}
	if _, err := holder.ExecContext(ctx, `UPDATE meta SET boot_id = boot_id`); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = readSnapshot(ctx, path, 150*time.Millisecond)
	elapsed := time.Since(start)
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("locked %v after %s", err, elapsed)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("waited %s, want the bounded timeout", elapsed)
	}
}

func mustMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return st.Mode()
}
