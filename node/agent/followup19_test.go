package agent

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/contracts"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

const zeroUsageJSON = `{"acked_archive_bytes":"0","blocked_seq":null,"central_acked_through":"0","last_durable_event_seq":"0","ledger_epoch":null,"meter_epoch_id":null,"oldest_unacked_at":null,"unacked_event_count":0,"unacked_payload_bytes":"0"}`

func TestSECOPS19_StoreWatermark(t *testing.T) {
	ctx := context.Background()
	node := uuid.MustParse("93bc18d9-26c2-4ec4-bf5b-0c70df89b0cb")
	inc := uuid.MustParse("d1c5d642-f6ac-4ca4-bcd6-5460d491a2e0")
	ledger := uuid.MustParse("7612b7d9-1e70-4390-b86b-7c3456e4df27")
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
	s, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), node, inc, usagestore.Options{Trust: trust})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	now := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	raw := signUsageCheckpoint(t, priv, "recovery-test", node, inc, api.UsageCheckpoint{
		LedgerEpoch: ledger, IssuedAt: now,
		HighestContiguousSeq: 0, MaxSeenSeq: 0,
	})
	if err := s.ApplyCheckpoint(ctx, raw, now); err != nil {
		t.Fatal(err)
	}
	meterEpoch := seedDurableEvents(t, s, 3)
	ackThrough(t, s, ledger, 2)
	for _, seq := range []uint64{1, 2} {
		ev, err := s.Event(ctx, seq)
		if err != nil || ev.State != "archived" {
			t.Fatalf("seq %d state %q err %v", seq, ev.State, err)
		}
	}
	ev, err := s.Event(ctx, 3)
	if err != nil || ev.State != "outbox" || len(ev.Payload) == 0 {
		t.Fatalf("unacked event %+v %v", ev, err)
	}
	var archived int
	for _, seq := range []uint64{1, 2} {
		row, err := s.Event(ctx, seq)
		if err != nil {
			t.Fatal(err)
		}
		archived += len(row.Payload)
	}

	var buf bytes.Buffer
	e := statusEngine(&buf)
	e.usageStatus = s
	wantErrs := append([]api.NodeError(nil), e.decision.Errors...)
	rep, err := e.report(ctx, now.Add(45*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	direct, err := s.OutboxStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if usageJSON(t, rep.Usage) != usageJSON(t, direct) {
		t.Fatalf("report %s\nstore %s", usageJSON(t, rep.Usage), usageJSON(t, direct))
	}
	assertWatermark(t, rep.Usage, ledger, meterEpoch, len(ev.Payload), archived)
	gotOld, err := rep.Usage.OldestUnackedAt.Get()
	if err != nil || !gotOld.Equal(ev.CreatedAt) {
		t.Fatalf("oldest %s want %s err %v", gotOld, ev.CreatedAt, err)
	}
	rawUsage, err := json.Marshal(rep.Usage)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawUsage), `"last_durable_event_seq":"3"`) || !strings.Contains(string(rawUsage), `"central_acked_through":"2"`) {
		t.Fatalf("usage json %s", rawUsage)
	}
	assertUntouched(t, e, rep, wantErrs)
	requireObservationSchema(t, rep)
	if !strings.Contains(buf.String(), "usage status available") {
		t.Fatalf("log %s", buf.String())
	}
}

func TestSECOPS19_UsageReadDoesNotChangeSafety(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 10, 30, 18, 45, 0, 0, time.UTC)
	var buf bytes.Buffer
	e := statusEngine(&buf)
	wantErrs := append([]api.NodeError(nil), e.decision.Errors...)

	rep, err := e.report(ctx, at)
	if err != nil {
		t.Fatal(err)
	}
	assertZeroUsage(t, rep.Usage)
	assertUntouched(t, e, rep, wantErrs)
	requireObservationSchema(t, rep)
	if got := usageJSON(t, rep.Usage); got != zeroUsageJSON {
		t.Fatalf("zero json %s", got)
	}
	if _, err := e.report(ctx, at); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(buf.String(), "usage status unavailable: not_open"); n != 1 {
		t.Fatalf("not_open logs %d\n%s", n, buf.String())
	}

	e.usageStatus = scriptUsage{fn: func(context.Context) (api.UsageOutboxStatus, error) {
		return api.UsageOutboxStatus{}, errUsagePath
	}}
	rep, err = e.report(ctx, at)
	if err != nil {
		t.Fatal(err)
	}
	assertZeroUsage(t, rep.Usage)
	assertUntouched(t, e, rep, wantErrs)
	requireObservationSchema(t, rep)
	if strings.Contains(buf.String(), "/var/lib/vpn-node/state/usage.db") {
		t.Fatalf("path logged\n%s", buf.String())
	}
	if n := strings.Count(buf.String(), "usage status unavailable: read_failed"); n != 1 {
		t.Fatalf("read_failed logs %d\n%s", n, buf.String())
	}

	good := sampleStatus()
	e.usageStatus = scriptUsage{fn: func(context.Context) (api.UsageOutboxStatus, error) {
		return good, nil
	}}
	rep, err = e.report(ctx, at)
	if err != nil {
		t.Fatal(err)
	}
	if usageJSON(t, rep.Usage) != usageJSON(t, good) {
		t.Fatalf("good %s", usageJSON(t, rep.Usage))
	}
	assertUntouched(t, e, rep, wantErrs)
	requireObservationSchema(t, rep)
	if _, err := e.report(ctx, at); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(buf.String(), "usage status available"); n != 1 {
		t.Fatalf("available logs %d\n%s", n, buf.String())
	}

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	e.usageStatusBound = time.Nanosecond
	e.usageStatus = stuckUsage{release: release}
	rep, err = e.report(ctx, at)
	if err != nil {
		t.Fatal(err)
	}
	if usageJSON(t, rep.Usage) != usageJSON(t, good) {
		t.Fatalf("timeout lost the last good status %s", usageJSON(t, rep.Usage))
	}
	assertUntouched(t, e, rep, wantErrs)
	requireObservationSchema(t, rep)
	if _, err := e.report(ctx, at); err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(buf.String(), "usage status unavailable: deadline"); n != 1 {
		t.Fatalf("deadline logs %d\n%s", n, buf.String())
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	e.usageStatus = scriptUsage{fn: func(ctx context.Context) (api.UsageOutboxStatus, error) {
		<-ctx.Done()
		return api.UsageOutboxStatus{}, ctx.Err()
	}}
	rep, err = e.report(canceled, at)
	if err != nil {
		t.Fatal(err)
	}
	assertZeroUsage(t, rep.Usage)
	assertUntouched(t, e, rep, wantErrs)
	requireObservationSchema(t, rep)
	if n := strings.Count(buf.String(), "usage status unavailable: canceled"); n != 1 {
		t.Fatalf("canceled logs %d\n%s", n, buf.String())
	}

	var fresh bytes.Buffer
	cold := statusEngine(&fresh)
	cold.usageStatusBound = time.Nanosecond
	cold.usageStatus = stuckUsage{release: release}
	rep, err = cold.report(ctx, at)
	if err != nil {
		t.Fatal(err)
	}
	assertZeroUsage(t, rep.Usage)
	assertUntouched(t, cold, rep, wantErrs)
	requireObservationSchema(t, rep)
	if n := strings.Count(fresh.String(), "usage status unavailable: deadline"); n != 1 {
		t.Fatalf("cold deadline logs %d\n%s", n, fresh.String())
	}
}

func TestSECOPS19_FakecontrolWatermark(t *testing.T) {
	ctx := context.Background()
	r := newRig(t)
	ledger, err := r.srv.SetUsageCheckpoint("same", 0)
	if err != nil {
		t.Fatal(err)
	}
	enrollBoot(t, r)
	stopUsageLoops(t, r)
	head, err := r.eng.usage.Head(ctx)
	if err != nil || head.LedgerEpoch != ledger {
		t.Fatalf("ledger %s want %s err %v\n%s", head.LedgerEpoch, ledger, err, r.log.String())
	}
	meterEpoch := seedDurableEvents(t, r.eng.usage, 3)
	ackThrough(t, r.eng.usage, ledger, 2)
	ev, err := r.eng.usage.Event(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	var archived int
	for _, seq := range []uint64{1, 2} {
		row, err := r.eng.usage.Event(ctx, seq)
		if err != nil || row.State != "archived" {
			t.Fatalf("seq %d %+v %v", seq, row, err)
		}
		archived += len(row.Payload)
	}
	safety, prov, serving := r.eng.decision.Safety, r.eng.decision.Provisioning, r.eng.serving
	errN := len(r.eng.decision.Errors)

	if err := r.eng.observe(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.flush(ctx); err != nil {
		t.Fatal(err)
	}
	if r.eng.decision.Safety != safety || r.eng.decision.Provisioning != prov || r.eng.serving != serving || len(r.eng.decision.Errors) != errN {
		t.Fatal("usage read changed safety, provisioning, forwarding, or errors")
	}
	first, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no observation")
	}
	assertWatermark(t, first.Usage, ledger, meterEpoch, len(ev.Payload), archived)
	if first.SafetyStatus != safety || first.ProvisioningState != prov || first.MemberForwardingEnabled != serving || len(first.Errors) != errN {
		t.Fatalf("report safety %s provisioning %s forwarding %v errors %d", first.SafetyStatus, first.ProvisioningState, first.MemberForwardingEnabled, len(first.Errors))
	}
	requireObservationSchema(t, first)
	if !strings.Contains(r.log.String(), "usage status available") {
		t.Fatalf("log %s", r.log.String())
	}

	// The second process has to see the same boot id and the same counter
	// snapshot. An empty reading is a disappeared counter, and the meter
	// would open a new epoch and append a gap event before the report.
	head, err = r.eng.usage.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	counters := map[string]enforce.CounterValue{}
	for _, c := range head.Counters {
		counters[c.Name] = enforce.CounterValue{Packets: c.Packets, Bytes: c.Bytes}
	}
	r.srv.SetUsageUnavailable(8)
	restartUsageEngine(t, r, fixedSource{boot: head.BootID, counters: counters})
	stopUsageLoops(t, r)
	if err := r.eng.observe(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.flush(ctx); err != nil {
		t.Fatal(err)
	}
	second, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("no second observation")
	}
	assertWatermark(t, second.Usage, ledger, meterEpoch, len(ev.Payload), archived)
	if !second.Usage.OldestUnackedAt.IsSpecified() || second.Usage.OldestUnackedAt.IsNull() {
		t.Fatal("second observation lost oldest_unacked_at")
	}
	firstOld, err := first.Usage.OldestUnackedAt.Get()
	if err != nil {
		t.Fatal(err)
	}
	secondOld, err := second.Usage.OldestUnackedAt.Get()
	if err != nil || !secondOld.Equal(firstOld) {
		t.Fatalf("oldest %s then %s", firstOld, secondOld)
	}
	if second.NodeId != first.NodeId || second.NodeIncarnationId != first.NodeIncarnationId {
		t.Fatalf("identity %s/%s then %s/%s", first.NodeId, first.NodeIncarnationId, second.NodeId, second.NodeIncarnationId)
	}
	requireObservationSchema(t, second)
}

func seedDurableEvents(t *testing.T, s *usagestore.Store, n int) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	at := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	head, err := s.Head(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dev := uuid.MustParse("2df6c7df-e7e8-4bd4-9f2c-a5836dd0d8ea")
	mem := uuid.MustParse("dc01a74b-9e6f-41ac-81cd-03291d0a88c0")
	name := "u_" + strings.ReplaceAll(dev.String(), "-", "")
	boot := head.BootID
	if boot == "" {
		boot = "boot"
	}
	epoch := head.OpenEpoch
	base := append([]usagestore.Counter(nil), head.Counters...)
	var have uint64
	for _, c := range base {
		if c.Name == name {
			have = c.Bytes
		}
	}
	if epoch == uuid.Nil {
		epoch = uuid.Must(uuid.NewV7())
		zero := usagestore.Counter{
			Name: name, DeviceID: dev, Direction: "upload", Packets: 1,
			MemberID: mem, BindingRevision: 7,
		}
		if _, err := s.Commit(ctx, usagestore.Commit{
			Counters:  []usagestore.Counter{zero},
			Open:      &usagestore.NewEpoch{ID: epoch, Reason: "baseline", BootID: boot},
			BootID:    boot,
			SampledAt: at,
		}); err != nil {
			t.Fatal(err)
		}
		base = []usagestore.Counter{zero}
		have = 0
	}
	start := head.NextSeq
	var last uint64
	for i := 1; i <= n; i++ {
		to := have + 10
		end := at.Add(time.Duration(i) * 10 * time.Second)
		base = advanceUpload(base, name, dev, mem, to)
		seq, err := s.Commit(ctx, usagestore.Commit{
			Epoch: epoch, Counters: base, BootID: boot, SampledAt: end,
			Event: &usagestore.Draft{
				EpochID: epoch, Start: end.Add(-10 * time.Second), End: end, Quality: "complete",
				Records: []usagestore.DraftRecord{{
					MemberID: mem, DeviceID: dev, BindingRevision: 7,
					PeriodID: "2026-10", Upload: to - have,
				}},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		last = seq
		have = to
	}
	if last != start+uint64(n)-1 {
		t.Fatalf("last durable seq %d after %d events from %d", last, n, start)
	}
	return epoch
}

func advanceUpload(base []usagestore.Counter, name string, dev, mem uuid.UUID, bytes uint64) []usagestore.Counter {
	out := append([]usagestore.Counter(nil), base...)
	for i := range out {
		if out[i].Name != name {
			continue
		}
		out[i].Bytes = bytes
		if out[i].Packets == 0 {
			out[i].Packets = 1
		}
		return out
	}
	return append(out, usagestore.Counter{
		Name: name, DeviceID: dev, Direction: "upload", Packets: 1, Bytes: bytes,
		MemberID: mem, BindingRevision: 7,
	})
}

// fixedSource is the counter snapshot a restarted agent must read so the
// meter's continuity check does not open another epoch.
type fixedSource struct {
	boot     string
	counters map[string]enforce.CounterValue
}

func (f fixedSource) Read(context.Context) (meter.Observation, error) {
	return meter.Observation{Counters: f.counters, BootID: f.boot}, nil
}

func restartUsageEngine(t *testing.T, r *rig, src meter.Source) {
	t.Helper()
	cfg := r.eng.cfg
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(r.log, &slog.HandlerOptions{Level: slog.LevelDebug}))
	next, err := Start(Options{
		Config: cfg, Clock: r.clock, Plane: r.plane, Log: log,
		MeterSource: src, Prober: passProbe{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	r.eng = next
	if err := next.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func ackThrough(t *testing.T, s *usagestore.Store, epoch uuid.UUID, highest uint64) {
	t.Helper()
	results := make([]usagestore.EventResult, 0, highest)
	for seq := uint64(1); seq <= highest; seq++ {
		results = append(results, usagestore.EventResult{Seq: seq, Status: usagestore.StatusAccepted})
	}
	if err := s.ApplyACK(context.Background(), usagestore.BatchACK{
		LedgerEpoch: epoch, Highest: highest, Results: results,
	}); err != nil {
		t.Fatal(err)
	}
}

func signUsageCheckpoint(t *testing.T, priv ed25519.PrivateKey, keyID string, node, inc uuid.UUID, cp api.UsageCheckpoint) []byte {
	t.Helper()
	cp.NodeId = node
	cp.NodeIncarnationId = inc
	cp.RecoveryRecordId = presence.NewNullNullable[uuid.UUID]()
	if cp.MissingRanges == nil {
		cp.MissingRanges = []api.SeqRange{}
	}
	env, err := envelope.Sign(cp, priv, keyID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := env.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func stopUsageLoops(t *testing.T, r *rig) {
	t.Helper()
	if r.eng.loopCancel == nil {
		t.Fatal("usage loops have not started")
	}
	r.eng.loopCancel()
	r.eng.loops.Wait()
}

type scriptUsage struct {
	fn func(context.Context) (api.UsageOutboxStatus, error)
}

func (s scriptUsage) OutboxStatus(ctx context.Context) (api.UsageOutboxStatus, error) {
	return s.fn(ctx)
}

// stuckUsage does not return until release is closed. The observation bound
// has to win without this method observing the context.
type stuckUsage struct{ release <-chan struct{} }

func (s stuckUsage) OutboxStatus(context.Context) (api.UsageOutboxStatus, error) {
	<-s.release
	return api.UsageOutboxStatus{}, errUsagePath
}

var errUsagePath = pathError{}

type pathError struct{}

func (pathError) Error() string { return "open /var/lib/vpn-node/state/usage.db: locked" }

func statusEngine(buf *bytes.Buffer) *Engine {
	return &Engine{
		cfg: Config{
			AgentRelease: "0.0.0-test", NetbirdBackend: "kernel",
			ReleaseID: "019a0000-0000-7000-8000-0000000000c1",
		},
		nodeID:  uuid.MustParse("019a0000-0000-7000-8000-0000000000e1"),
		incID:   uuid.MustParse("019a0000-0000-7000-8000-0000000000e2"),
		bootID:  uuid.MustParse("019a0000-0000-7000-8000-0000000000e3"),
		log:     slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})),
		serving: true,
		decision: Decision{
			Safety: api.SafetyStatusPassed, Provisioning: api.ProvisioningStateReady,
			DataHealth: api.DataHealthHealthy,
			Errors: []api.NodeError{{
				Code: api.ErrorCodeSTOREUNAVAILABLE, Message: "kept", Stage: "store",
				ObservedAt: time.Date(2026, 10, 30, 18, 45, 0, 0, time.UTC),
			}},
			Reasons: []api.ForwardingBlockReason{},
			Checks:  []api.NodeCheck{},
		},
	}
}

func sampleStatus() api.UsageOutboxStatus {
	return api.UsageOutboxStatus{
		AckedArchiveBytes:   decstr.U64(20),
		BlockedSeq:          presence.NewNullNullable[decstr.U64](),
		CentralAckedThrough: decstr.U64(2),
		LastDurableEventSeq: decstr.U64(3),
		LedgerEpoch:         presence.NewNullableWithValue(uuid.MustParse("7612b7d9-1e70-4390-b86b-7c3456e4df27")),
		MeterEpochId:        presence.NewNullableWithValue(uuid.MustParse("11111111-1111-4111-8111-111111111111")),
		OldestUnackedAt:     presence.NewNullableWithValue(time.Date(2026, 10, 30, 18, 1, 0, 0, time.UTC)),
		UnackedEventCount:   1,
		UnackedPayloadBytes: decstr.U64(10),
	}
}

func assertWatermark(t *testing.T, st api.UsageOutboxStatus, ledger, meter uuid.UUID, unackedBytes, archivedBytes int) {
	t.Helper()
	if st.LastDurableEventSeq != 3 || st.CentralAckedThrough != 2 {
		t.Fatalf("seqs last %s acked %s", st.LastDurableEventSeq, st.CentralAckedThrough)
	}
	if st.UnackedEventCount != 1 || st.UnackedPayloadBytes != decstr.U64(unackedBytes) || unackedBytes == 0 {
		t.Fatalf("unacked count %d bytes %s want %d", st.UnackedEventCount, st.UnackedPayloadBytes, unackedBytes)
	}
	if st.AckedArchiveBytes != decstr.U64(archivedBytes) || archivedBytes == 0 {
		t.Fatalf("archive %s want %d", st.AckedArchiveBytes, archivedBytes)
	}
	if !st.BlockedSeq.IsNull() {
		t.Fatalf("blocked %v", st.BlockedSeq)
	}
	gotLedger, err := st.LedgerEpoch.Get()
	if err != nil || gotLedger != ledger {
		t.Fatalf("ledger %s want %s err %v", gotLedger, ledger, err)
	}
	gotMeter, err := st.MeterEpochId.Get()
	if err != nil || gotMeter != meter {
		t.Fatalf("meter %s want %s err %v", gotMeter, meter, err)
	}
	if _, err := st.OldestUnackedAt.Get(); err != nil {
		t.Fatal(err)
	}
}

func assertZeroUsage(t *testing.T, st api.UsageOutboxStatus) {
	t.Helper()
	if got := usageJSON(t, st); got != zeroUsageJSON {
		t.Fatalf("usage %s", got)
	}
}

func assertUntouched(t *testing.T, e *Engine, rep api.ObservationReport, wantErrs []api.NodeError) {
	t.Helper()
	if rep.SafetyStatus != e.decision.Safety || rep.ProvisioningState != e.decision.Provisioning || rep.MemberForwardingEnabled != e.serving {
		t.Fatalf("report safety %s provisioning %s forwarding %v", rep.SafetyStatus, rep.ProvisioningState, rep.MemberForwardingEnabled)
	}
	if e.decision.Safety != api.SafetyStatusPassed || e.decision.Provisioning != api.ProvisioningStateReady || !e.serving {
		t.Fatal("engine safety, provisioning, or forwarding changed")
	}
	if !reflect.DeepEqual(rep.Errors, wantErrs) || !reflect.DeepEqual(e.decision.Errors, wantErrs) {
		t.Fatalf("errors report %#v engine %#v", rep.Errors, e.decision.Errors)
	}
}

func usageJSON(t *testing.T, st api.UsageOutboxStatus) string {
	t.Helper()
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

var (
	obsSchemaOnce sync.Once
	obsSchema     *openapi3.Schema
	obsSchemaErr  error
)

func observationSchema(t *testing.T) *openapi3.Schema {
	t.Helper()
	obsSchemaOnce.Do(func() {
		loader := openapi3.NewLoader()
		doc, err := loader.LoadFromData(contracts.OpenAPI())
		if err != nil {
			obsSchemaErr = err
			return
		}
		ref, ok := doc.Components.Schemas["ObservationReport"]
		if !ok || ref.Value == nil {
			obsSchemaErr = errors.New("ObservationReport schema missing")
			return
		}
		obsSchema = ref.Value
	})
	if obsSchemaErr != nil {
		t.Fatal(obsSchemaErr)
	}
	return obsSchema
}

func requireObservationSchema(t *testing.T, rep api.ObservationReport) {
	t.Helper()
	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	var again api.ObservationReport
	if err := json.Unmarshal(raw, &again); err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	opts := []openapi3.SchemaValidationOption{
		openapi3.MultiErrors(),
		openapi3.EnableFormatValidation(),
		openapi3.WithStringFormatValidator("uuid", openapi3.NewRegexpFormatValidator(openapi3.FormatOfStringForUUIDOfRFC9562)),
	}
	if err := observationSchema(t).VisitJSON(v, opts...); err != nil {
		t.Fatalf("observation schema: %v", err)
	}
}
