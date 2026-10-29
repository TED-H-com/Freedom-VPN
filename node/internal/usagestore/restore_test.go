package usagestore

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func testTrust(t *testing.T) (ed25519.PrivateKey, *envelope.TrustStore) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := envelope.NewNodeTrustStore(envelope.TrustedKey{
		KeyID:     "recovery-test",
		PublicKey: pub,
		Purposes:  []envelope.Purpose{envelope.PurposeRecovery},
		NotBefore: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	return priv, ts
}

func signCheckpoint(t *testing.T, priv ed25519.PrivateKey, cp api.UsageCheckpoint) []byte {
	t.Helper()
	cp.NodeId = testNode
	cp.NodeIncarnationId = testInc
	cp.RecoveryRecordId = presence.NewNullNullable[uuid.UUID]()
	if cp.MissingRanges == nil {
		cp.MissingRanges = []api.SeqRange{}
	}
	if cp.IssuedAt.IsZero() {
		cp.IssuedAt = time.Date(2026, 10, 28, 18, 0, 0, 0, time.UTC)
	}
	env, err := envelope.Sign(cp, priv, "recovery-test")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := env.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestAT30_LedgerEpochRestore(t *testing.T) {
	ctx := context.Background()
	priv, trust := testTrust(t)
	now := time.Date(2026, 10, 28, 18, 0, 0, 0, time.UTC)
	s := openTest(t, Options{Trust: trust, Now: func() time.Time { return now }})
	at := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)
	epoch := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
	payloads := map[uint64]string{}
	var from uint64
	for i := 1; i <= 6; i++ {
		to := from + 3
		seq, err := s.Commit(ctx, delta(epoch, at.Add(time.Duration(i)*10*time.Second), from, to))
		if err != nil {
			t.Fatal(err)
		}
		ev, err := s.Event(ctx, seq)
		if err != nil {
			t.Fatal(err)
		}
		payloads[seq] = string(ev.Payload)
		from = to
	}
	epochA := uuid.MustParse("7612b7d9-1e70-4390-b86b-7c3456e4df27")
	epochB := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	rawA := signCheckpoint(t, priv, api.UsageCheckpoint{
		LedgerEpoch: epochA, IssuedAt: now,
		HighestContiguousSeq: decstr.U64(6), MaxSeenSeq: decstr.U64(6),
	})
	if err := s.ApplyCheckpoint(ctx, rawA, now); err != nil {
		t.Fatal(err)
	}
	if got, err := s.AckedThrough(ctx); err != nil || got != 6 {
		t.Fatalf("acked %d %v", got, err)
	}
	out, arch, err := s.CountEvents(ctx)
	if err != nil || out != 0 || arch != 6 {
		t.Fatalf("out %d arch %d err %v", out, arch, err)
	}

	rawLow := signCheckpoint(t, priv, api.UsageCheckpoint{
		LedgerEpoch: epochA, IssuedAt: now.Add(time.Minute),
		HighestContiguousSeq: decstr.U64(4), MaxSeenSeq: decstr.U64(6),
		MissingRanges: []api.SeqRange{{StartSeq: 5, EndSeq: 6}},
	})
	if err := s.ApplyCheckpoint(ctx, rawLow, now); !errors.Is(err, ErrAckRegress) {
		t.Fatalf("same-epoch lower ACK err = %v", err)
	}
	if got, _ := s.AckedThrough(ctx); got != 6 {
		t.Fatalf("acked moved to %d", got)
	}

	rawB := signCheckpoint(t, priv, api.UsageCheckpoint{
		LedgerEpoch: epochB, IssuedAt: now.Add(time.Hour),
		HighestContiguousSeq: decstr.U64(3), MaxSeenSeq: decstr.U64(6),
		MissingRanges: []api.SeqRange{{StartSeq: 4, EndSeq: 6}},
	})
	if err := s.ApplyCheckpoint(ctx, rawB, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.AckedThrough(ctx); got != 3 {
		t.Fatalf("acked %d", got)
	}
	for seq := uint64(1); seq <= 3; seq++ {
		ev, err := s.Event(ctx, seq)
		if err != nil || ev.State != "archived" {
			t.Fatalf("seq %d %+v %v", seq, ev, err)
		}
	}
	for seq := uint64(4); seq <= 6; seq++ {
		ev, err := s.Event(ctx, seq)
		if err != nil || ev.State != "outbox" || string(ev.Payload) != payloads[seq] {
			t.Fatalf("replay seq %d changed: %s", seq, ev.Payload)
		}
	}
	epochs, err := s.LedgerEpochs(ctx)
	if err != nil || len(epochs) != 2 || epochs[0] != epochA || epochs[1] != epochB {
		t.Fatalf("epochs %v %v", epochs, err)
	}
	head, err := s.Head(ctx)
	if err != nil || head.LedgerEpoch != epochB || head.NextSeq != 7 {
		t.Fatalf("head %+v %v", head, err)
	}
	if err := s.ApplyCheckpoint(ctx, rawA, now.Add(2*time.Hour)); !errors.Is(err, ErrEpochReplay) {
		t.Fatalf("historical epoch err = %v", err)
	}
	if got, _ := s.AckedThrough(ctx); got != 3 {
		t.Fatalf("historical epoch moved ACK to %d", got)
	}

	ranges, err := s.ReplayUnavailable(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranges) != 0 {
		t.Fatalf("local events were marked unavailable: %+v", ranges)
	}
}

func TestCheckpointAdvancesPastAHole(t *testing.T) {
	ctx := context.Background()
	priv, trust := testTrust(t)
	now := time.Date(2026, 10, 28, 18, 0, 0, 0, time.UTC)
	s := openTest(t, Options{Trust: trust})
	raw := signCheckpoint(t, priv, api.UsageCheckpoint{
		LedgerEpoch:          uuid.MustParse("7612b7d9-1e70-4390-b86b-7c3456e4df27"),
		IssuedAt:             now,
		HighestContiguousSeq: 400,
		MaxSeenSeq:           417,
		MissingRanges:        []api.SeqRange{{StartSeq: 401, EndSeq: 409}},
	})
	if err := s.ApplyCheckpoint(ctx, raw, now); err != nil {
		t.Fatal(err)
	}
	head, err := s.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if head.AckedThrough != 400 || head.NextSeq != 418 {
		t.Fatalf("head %+v", head)
	}
	ranges, err := s.ReplayUnavailable(ctx)
	if err != nil || len(ranges) != 1 || uint64(ranges[0].StartSeq) != 401 || uint64(ranges[0].EndSeq) != 409 {
		t.Fatalf("%+v %v", ranges, err)
	}
	if ranges[0].Reason != api.ReplayUnavailableRangeReasonLocalDataLost {
		t.Fatalf("reason %s", ranges[0].Reason)
	}
	at := now
	meter := openEpoch(t, s, []Counter{counterAt(testDev, testMem, "upload", 1, 0, 7)}, at)
	seq, err := s.Commit(ctx, delta(meter, at, 0, 8))
	if err != nil {
		t.Fatal(err)
	}
	if seq != 418 {
		t.Fatalf("reused or skipped wrong, seq %d", seq)
	}
	if err := s.ApplyCheckpoint(ctx, raw[:len(raw)-1], now); !errors.Is(err, ErrBadCheckpoint) {
		t.Fatalf("tamper err %v", err)
	}
	bare := openTest(t, Options{})
	if err := bare.ApplyCheckpoint(ctx, raw, now); !errors.Is(err, ErrNoTrust) {
		t.Fatalf("trust err %v", err)
	}
}
