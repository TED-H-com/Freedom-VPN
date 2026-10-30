//go:build integration

package usage_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// testEnv holds seeded entities and a configured usage.Service for integration tests.
type testEnv struct {
	pool            *pgxpool.Pool
	svc             *usage.Service
	trustStore      *envelope.TrustStore
	epochTrustStore *envelope.TrustStore
	now             time.Time

	memberA uuid.UUID
	memberB uuid.UUID
	deviceA uuid.UUID
	deviceB uuid.UUID

	node1     uuid.UUID
	inc1      uuid.UUID
	identity1 usage.NodeIdentity

	node2     uuid.UUID
	inc2      uuid.UUID
	identity2 usage.NodeIdentity
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	pool := dbtest.New(t)

	now := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	const keyID = "test-recovery-key-1"
	signer := usage.NewRecoverySigner(keyID, priv)
	ts, err := envelope.NewTrustStore(envelope.TrustedKey{
		KeyID:     keyID,
		PublicKey: pub,
		Purposes:  []envelope.Purpose{envelope.PurposeRecovery},
		NotBefore: now.Add(-24 * time.Hour),
		NotAfter:  now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("NewTrustStore: %v", err)
	}

	epochPub, epochPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ledger-epoch ed25519 key: %v", err)
	}
	const epochKeyID = "test-ledger-epoch-key-1"
	epochSigner := usage.NewLedgerEpochSigner(epochKeyID, epochPriv)
	epochTS, err := envelope.NewTrustStore(envelope.TrustedKey{
		KeyID:     epochKeyID,
		PublicKey: epochPub,
		Purposes:  []envelope.Purpose{envelope.PurposeLedgerEpoch},
		NotBefore: now.Add(-24 * time.Hour),
		NotAfter:  now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("NewTrustStore(epoch): %v", err)
	}

	env := &testEnv{
		pool:            pool,
		trustStore:      ts,
		epochTrustStore: epochTS,
		now:             now,
		memberA:         uuid.New(),
		memberB:         uuid.New(),
		deviceA:         uuid.New(),
		deviceB:         uuid.New(),
		node1:           uuid.New(),
		inc1:            uuid.New(),
		node2:           uuid.New(),
		inc2:            uuid.New(),
	}
	env.identity1 = usage.NodeIdentity{NodeID: env.node1, NodeIncarnationID: env.inc1}
	env.identity2 = usage.NodeIdentity{NodeID: env.node2, NodeIncarnationID: env.inc2}

	svc, err := usage.NewService(usage.Config{
		Pool:              pool,
		Signer:            signer,
		LedgerEpochSigner: epochSigner,
		Clock:             func() time.Time { return env.now },
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	env.svc = svc

	// Seed members, entitlements (required by F-05 before authorizing devices),
	// devices, historical device_bindings, nodes, and node_incarnations.
	mustExec(t, pool, `INSERT INTO members(id, status) VALUES ($1, 'active'), ($2, 'active')`, env.memberA, env.memberB)
	mustExec(t, pool, `
		INSERT INTO entitlements(member_id, revision, device_limit, valid_from, effective_service_end)
		VALUES ($1, 1, 5, now() - interval '1 year', now() + interval '1 year'),
		       ($2, 1, 5, now() - interval '1 year', now() + interval '1 year')
	`, env.memberA, env.memberB)
	mustExec(t, pool, `
		INSERT INTO member_devices(id, member_id, deployment_id, peer_id, status)
		VALUES ($1, $2, 'dep-1', 'peer-a', 'authorized'),
		       ($3, $4, 'dep-1', 'peer-b', 'authorized')
	`, env.deviceA, env.memberA, env.deviceB, env.memberB)

	// Historical bindings:
	// deviceA revision 1: valid [2026-01-01Z, 2026-12-01Z) on 100.64.0.10
	// deviceB revision 1: valid [2026-01-01Z, 2026-12-01Z) on 100.64.0.11
	mustExec(t, pool, `
		INSERT INTO device_bindings(
			device_id, member_id, overlay_scope, overlay_ip,
			binding_revision, valid_from, valid_until, peer_ref
		) VALUES
		($1, $2, 'overlay-main', '100.64.0.10', 1, '2026-01-01T00:00:00Z', '2026-12-01T00:00:00Z', 'peer-a'),
		($3, $4, 'overlay-main', '100.64.0.11', 1, '2026-01-01T00:00:00Z', '2026-12-01T00:00:00Z', 'peer-b')
	`, env.deviceA, env.memberA, env.deviceB, env.memberB)

	mustExec(t, pool, `
		INSERT INTO nodes(id, lifecycle_state, provisioning_state)
		VALUES ($1, 'registered', 'ready'),
		       ($2, 'registered', 'ready')
	`, env.node1, env.node2)
	mustExec(t, pool, `
		INSERT INTO node_incarnations(id, node_id, public_key_fingerprint, deployment_id, peer_id, os, arch)
		VALUES ($1, $2, decode(repeat('11', 32), 'hex'), 'dep-1', 'node-peer-1', 'ubuntu', 'amd64'),
		       ($3, $4, decode(repeat('22', 32), 'hex'), 'dep-1', 'node-peer-2', 'ubuntu', 'arm64')
	`, env.inc1, env.node1, env.inc2, env.node2)
	mustExec(t, pool, `UPDATE nodes SET current_incarnation_id = $2, resource_version = resource_version + 1 WHERE id = $1`, env.node1, env.inc1)
	mustExec(t, pool, `UPDATE nodes SET current_incarnation_id = $2, resource_version = resource_version + 1 WHERE id = $1`, env.node2, env.inc2)

	return env
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec failed: %v\nSQL: %s", err, sql)
	}
}

func makeSignedEvent(
	t *testing.T,
	node usage.NodeIdentity,
	seq uint64,
	meterEpoch uuid.UUID,
	start, end time.Time,
	quality api.UsageQuality,
	gap *api.UsageGap,
	records []api.UsageRecord,
) api.UsageEvent {
	t.Helper()
	ev := api.UsageEvent{
		SchemaVersion:     api.UsageEventSchemaVersionN1,
		NodeId:            node.NodeID,
		NodeIncarnationId: node.NodeIncarnationID,
		MeterEpochId:      meterEpoch,
		EventSeq:          decstr.U64(seq),
		IntervalStart:     start.UTC(),
		IntervalEnd:       end.UTC(),
		Quality:           quality,
		Gap:               gap,
		MeteringVersion:   api.MeteringVersion(usage.MeteringVersionForwardAdmissionV1),
		Records:           records,
	}
	withHash, err := usage.WithComputedHash(ev)
	if err != nil {
		t.Fatalf("WithComputedHash seq=%d: %v", seq, err)
	}
	return withHash
}

func TestAT15_DedupeOutOfOrderAckAndConflict(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch1 := uuid.New()
	meterEpoch2 := uuid.New()
	base := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

	state, err := env.svc.GetLedgerState(ctx)
	if err != nil {
		t.Fatalf("GetLedgerState: %v", err)
	}
	epoch := state.LedgerEpoch

	ev1 := makeSignedEvent(t, env.identity1, 1, meterEpoch1, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     1000,
			DownloadBytes:   2000,
		},
	})
	ev2 := makeSignedEvent(t, env.identity1, 2, meterEpoch1, base.Add(time.Minute), base.Add(2*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     300,
			DownloadBytes:   700,
		},
	})
	ev3 := makeSignedEvent(t, env.identity1, 3, meterEpoch2, base.Add(2*time.Minute), base.Add(3*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     500,
			DownloadBytes:   500,
		},
	})

	// 1. Ingest event 1 and event 3 out-of-order (event 2 missing).
	// Event 1 is in meterEpoch1, event 3 is in meterEpoch2.
	res, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(epoch),
		Events:            []api.UsageEvent{ev1, ev3},
	})
	if err != nil {
		t.Fatalf("IngestBatch(1, 3): %v", err)
	}
	if res.HighestContiguousSeq != 1 || res.MaxSeenSeq != 3 || !res.BlockedSeq.IsNull() {
		t.Fatalf("unexpected watermark after (1, 3): hcs=%d max=%d blocked=%v", res.HighestContiguousSeq, res.MaxSeenSeq, res.BlockedSeq)
	}
	if res.Results[0].Status != api.UsageEventResultStatusAccepted || res.Results[1].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("unexpected results after (1, 3): %+v", res.Results)
	}

	// Verify checkpoint reports missing_ranges = [2, 2].
	cpEnv, err := env.svc.GetCheckpoint(ctx, env.identity1)
	if err != nil {
		t.Fatalf("GetCheckpoint: %v", err)
	}
	if cpEnv.Signed.HighestContiguousSeq != 1 || cpEnv.Signed.MaxSeenSeq != 3 {
		t.Fatalf("checkpoint watermark mismatch: %+v", cpEnv.Signed)
	}
	if len(cpEnv.Signed.MissingRanges) != 1 || cpEnv.Signed.MissingRanges[0].StartSeq != 2 || cpEnv.Signed.MissingRanges[0].EndSeq != 2 {
		t.Fatalf("checkpoint missing_ranges mismatch: %+v", cpEnv.Signed.MissingRanges)
	}

	// 2. Simulate ACK loss: resend event 1 and fill the gap with event 2.
	res, err = env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(epoch),
		Events:            []api.UsageEvent{ev1, ev2},
	})
	if err != nil {
		t.Fatalf("IngestBatch(1 dup, 2): %v", err)
	}
	if res.Results[0].Status != api.UsageEventResultStatusDuplicate || res.Results[1].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("want (duplicate, accepted), got %+v", res.Results)
	}
	if res.HighestContiguousSeq != 3 || res.MaxSeenSeq != 3 {
		t.Fatalf("want hcs=3 max=3 after filling gap 2, got hcs=%d max=%d", res.HighestContiguousSeq, res.MaxSeenSeq)
	}

	// Verify total bytes counted once: 1000+300+500 = 1800 up, 2000+700+500 = 3200 down, 5000 total.
	tot, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal: %v", err)
	}
	if tot.UploadBytes != 1800 || tot.DownloadBytes != 3200 || tot.TotalBytes != 5000 {
		t.Fatalf("unexpected totals after dedupe: up=%d down=%d total=%d", tot.UploadBytes, tot.DownloadBytes, tot.TotalBytes)
	}

	// 3. Store event 5 out-of-order (while 4 is not sent yet), then send a batch containing:
	//    - event 4 with invalid payload hash (422 rejected)
	//    - conflicting event 5 (same seq 5, different bytes/hash -> USAGE_EVENT_CONFLICT)
	//    - valid event 6 (accepted, must NOT be deleted or rolled back by 4's rejection or 5's conflict!)
	ev5Orig := makeSignedEvent(t, env.identity1, 5, meterEpoch2, base.Add(4*time.Minute), base.Add(5*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     100,
			DownloadBytes:   100,
		},
	})
	if _, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(epoch),
		Events:            []api.UsageEvent{ev5Orig},
	}); err != nil {
		t.Fatalf("IngestBatch(5): %v", err)
	}

	ev4BadHash := makeSignedEvent(t, env.identity1, 4, meterEpoch2, base.Add(3*time.Minute), base.Add(4*time.Minute), api.UsageQualityComplete, nil, nil)
	ev4BadHash.PayloadSha256 = "0000000000000000000000000000000000000000000000000000000000000000"

	ev5Conflict := makeSignedEvent(t, env.identity1, 5, meterEpoch2, base.Add(4*time.Minute), base.Add(5*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     999999,
			DownloadBytes:   999999,
		},
	})

	ev6Valid := makeSignedEvent(t, env.identity1, 6, meterEpoch2, base.Add(5*time.Minute), base.Add(6*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     200,
			DownloadBytes:   300,
		},
	})

	res, err = env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(epoch),
		Events:            []api.UsageEvent{ev4BadHash, ev5Conflict, ev6Valid},
	})
	if err != nil {
		t.Fatalf("IngestBatch(4 bad, 5 conflict, 6 valid): %v", err)
	}
	if res.Results[0].Status != api.UsageEventResultStatusRejected || (!res.Results[0].Error.IsSpecified() || res.Results[0].Error.IsNull()) || nullableValue(res.Results[0].Error).Code != api.ErrorCodeVALIDATIONFAILED {
		t.Fatalf("event 4 expected rejected VALIDATION_FAILED, got %+v", res.Results[0])
	}
	if res.Results[1].Status != api.UsageEventResultStatusConflict || (!res.Results[1].Error.IsSpecified() || res.Results[1].Error.IsNull()) || nullableValue(res.Results[1].Error).Code != api.ErrorCodeUSAGEEVENTCONFLICT {
		t.Fatalf("event 5 expected conflict USAGE_EVENT_CONFLICT, got %+v", res.Results[1])
	}
	if res.Results[2].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("event 6 expected accepted despite earlier conflict/rejection, got %+v", res.Results[2])
	}
	if res.HighestContiguousSeq != 3 {
		t.Fatalf("expected hcs=3, got %d", res.HighestContiguousSeq)
	}
	if (!res.BlockedSeq.IsSpecified() || res.BlockedSeq.IsNull()) || nullableValue(res.BlockedSeq) != 4 {
		t.Fatalf("expected blocked_seq=4, got %v", res.BlockedSeq)
	}

	// Verify both contents of event 5 are preserved (authoritative in usage_event_payloads, conflicting in usage_event_conflicts),
	// and conflicting bytes (999999) were NOT added to the rollup.
	var conflictCount int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM usage_event_conflicts WHERE node_id = $1 AND incarnation_id = $2 AND event_seq = 5`, env.node1, env.inc1).Scan(&conflictCount); err != nil || conflictCount != 1 {
		t.Fatalf("expected 1 conflict record for seq 5, got %d (err=%v)", conflictCount, err)
	}
	var rejectionCount int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM usage_event_rejections WHERE node_id = $1 AND incarnation_id = $2 AND event_seq = 4`, env.node1, env.inc1).Scan(&rejectionCount); err != nil || rejectionCount != 1 {
		t.Fatalf("expected 1 rejection record for seq 4, got %d (err=%v)", rejectionCount, err)
	}

	tot, err = env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal: %v", err)
	}
	// 1800 + 100 (ev5Orig) + 200 (ev6Valid) = 2100 up; 3200 + 100 + 300 = 3600 down.
	if tot.UploadBytes != 2100 || tot.DownloadBytes != 3600 || tot.TotalBytes != 5700 {
		t.Fatalf("unexpected totals after conflict/rejection batch: up=%d down=%d total=%d", tot.UploadBytes, tot.DownloadBytes, tot.TotalBytes)
	}

	// 4. Now send valid event 4: all receipts 1..6 are now durable and contiguous in usage_events
	// (since ev5Orig was already accepted before ev5Conflict arrived), so F-04 requires
	// highest_contiguous_seq to advance to 6.
	ev4Valid := makeSignedEvent(t, env.identity1, 4, meterEpoch2, base.Add(3*time.Minute), base.Add(4*time.Minute), api.UsageQualityComplete, nil, nil)
	res, err = env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(epoch),
		Events:            []api.UsageEvent{ev4Valid},
	})
	if err != nil {
		t.Fatalf("IngestBatch(4 valid): %v", err)
	}
	if res.HighestContiguousSeq != 6 || res.MaxSeenSeq != 6 {
		t.Fatalf("expected hcs=6 max=6 (all durable receipts 1..6 contiguous per F-04), got hcs=%d max=%d", res.HighestContiguousSeq, res.MaxSeenSeq)
	}
	if !res.BlockedSeq.IsNull() {
		t.Fatalf("expected blocked_seq=nil once 1..6 are contiguous, got %v", res.BlockedSeq)
	}

	// 5. mTLS mismatch must reject the whole batch with 403 INCARNATION_MISMATCH and no ledger change.
	ev7 := makeSignedEvent(t, env.identity1, 7, meterEpoch2, base.Add(6*time.Minute), base.Add(7*time.Minute), api.UsageQualityComplete, nil, nil)
	_, err = env.svc.IngestBatch(ctx, env.identity2, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(epoch),
		Events:            []api.UsageEvent{ev7},
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.HTTPStatus() != http.StatusForbidden || ae.Code != apierr.IncarnationMismatch {
		t.Fatalf("expected 403 INCARNATION_MISMATCH, got %v", err)
	}
	var seq7Exists bool
	if err := env.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM usage_events WHERE node_id = $1 AND event_seq = 7)`, env.node1).Scan(&seq7Exists); err != nil || seq7Exists {
		t.Fatalf("mTLS mismatch must not write to usage_events")
	}
}

func TestAT16_EventSeqContinuityAndNodeReportedGap(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	base := time.Date(2026, 10, 28, 13, 0, 0, 0, time.UTC)

	epoch1 := uuid.New()
	epoch2 := uuid.New()

	// Seq 1 in meter epoch 1.
	ev1 := makeSignedEvent(t, env.identity1, 1, epoch1, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     500,
			DownloadBytes:   1500,
		},
	})
	// Power loss / counter rebuild happens on the node: seq 2 is a gap event in meter epoch 2, with 0 records.
	lastCP := base.Add(time.Minute)
	ev2Gap := makeSignedEvent(t, env.identity1, 2, epoch2, base.Add(time.Minute), base.Add(4*time.Minute), api.UsageQualityGap, &api.UsageGap{
		Reason:           api.UsageGapReasonPowerLoss,
		LastCheckpointAt: presence.NewNullableWithValue(lastCP),
	}, nil)
	// Seq 3 continues in meter epoch 2 without resetting event_seq to 1.
	ev3 := makeSignedEvent(t, env.identity1, 3, epoch2, base.Add(4*time.Minute), base.Add(5*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     250,
			DownloadBytes:   750,
		},
	})

	res, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{ev1, ev2Gap, ev3},
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	})
	if err != nil {
		t.Fatalf("IngestBatch(1, 2 gap, 3): %v", err)
	}
	if res.HighestContiguousSeq != 3 || res.MaxSeenSeq != 3 {
		t.Fatalf("want hcs=3 max=3 across meter epochs, got hcs=%d max=%d", res.HighestContiguousSeq, res.MaxSeenSeq)
	}

	u, err := env.svc.GetMemberUsage(ctx, usage.MemberUsageQuery{
		MemberID:    env.memberA,
		Period:      "2026-10",
		Granularity: usage.GranularityDay,
	})
	if err != nil {
		t.Fatalf("GetMemberUsage: %v", err)
	}
	if u.Totals.UploadBytes != 750 || u.Totals.DownloadBytes != 2250 || u.Totals.TotalBytes != 3000 {
		t.Fatalf("unexpected totals: %+v", u.Totals)
	}
	if !u.Quality.Gap {
		t.Fatalf("expected Quality.Gap=true after node-reported gap event")
	}
	if len(u.Days) != 1 || !u.Days[0].Quality.Gap || u.Days[0].TotalBytes != 3000 {
		t.Fatalf("expected day row with gap=true and 3000 bytes, got %+v", u.Days)
	}
}

func TestAT17_MonthBoundaryLateEventsAndHistoricalBinding(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()

	// 1. Cross-month boundary event: [2026-10-31T23:59:30Z, 2026-11-01T00:00:30Z) (30s in Oct, 30s in Nov).
	// Upload = 101 bytes (Oct gets 50, Nov gets remainder 51), Download = 200 bytes (Oct gets 100, Nov gets 100).
	bStart := time.Date(2026, 10, 31, 23, 59, 30, 0, time.UTC)
	bEnd := time.Date(2026, 11, 1, 0, 0, 30, 0, time.UTC)
	evBoundary := makeSignedEvent(t, env.identity1, 1, meterEpoch, bStart, bEnd, api.UsageQualityBoundaryEstimated, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     101,
			DownloadBytes:   200,
		},
	})

	// 2. Late previous-month event arriving in November for [2026-10-31T23:50:00Z, 2026-10-31T23:55:00Z).
	env.now = time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	lateStart := time.Date(2026, 10, 31, 23, 50, 0, 0, time.UTC)
	lateEnd := time.Date(2026, 10, 31, 23, 55, 0, 0, time.UTC)
	evLateOct := makeSignedEvent(t, env.identity1, 2, meterEpoch, lateStart, lateEnd, api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     1000,
			DownloadBytes:   2000,
		},
	})

	// 3. Invalid negative-duration interval event (seq 3) must be rejected (422) and never counted.
	evNegInterval := makeSignedEvent(t, env.identity1, 3, meterEpoch, bEnd, bStart, api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     50000,
			DownloadBytes:   50000,
		},
	})

	res, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{evBoundary, evLateOct, evNegInterval},
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	})
	if err != nil {
		t.Fatalf("IngestBatch: %v", err)
	}
	if res.Results[0].Status != api.UsageEventResultStatusAccepted || res.Results[1].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("expected events 1 and 2 accepted, got %+v", res.Results)
	}
	if res.Results[2].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("expected negative interval event 3 rejected, got %+v", res.Results[2])
	}

	octUsage, err := env.svc.GetMemberUsage(ctx, usage.MemberUsageQuery{
		MemberID:    env.memberA,
		Period:      "2026-10",
		Granularity: usage.GranularityMonth,
	})
	if err != nil {
		t.Fatalf("GetMemberUsage(2026-10): %v", err)
	}
	// Oct has 50 + 1000 = 1050 up, 100 + 2000 = 2100 down, 3150 total, estimated=true.
	if octUsage.Totals.UploadBytes != 1050 || octUsage.Totals.DownloadBytes != 2100 || octUsage.Totals.TotalBytes != 3150 || !octUsage.Quality.Estimated {
		t.Fatalf("unexpected 2026-10 usage: %+v quality=%+v", octUsage.Totals, octUsage.Quality)
	}

	novUsage, err := env.svc.GetMemberUsage(ctx, usage.MemberUsageQuery{
		MemberID:    env.memberA,
		Period:      "2026-11",
		Granularity: usage.GranularityMonth,
	})
	if err != nil {
		t.Fatalf("GetMemberUsage(2026-11): %v", err)
	}
	// Nov has only the 51 up, 100 down from the boundary split (late Oct event did NOT pollute Nov!).
	if novUsage.Totals.UploadBytes != 51 || novUsage.Totals.DownloadBytes != 100 || novUsage.Totals.TotalBytes != 151 || !novUsage.Quality.Estimated {
		t.Fatalf("unexpected 2026-11 usage: %+v quality=%+v", novUsage.Totals, novUsage.Quality)
	}

	// 4. Historical device_bindings attribution (§10.2, §15.2):
	// Reassign IP 100.64.0.10 from deviceA (ending at 2026-11-05T00:00:00Z) to deviceB (revision 2 starting at 2026-11-05T00:00:00Z).
	mustExec(t, env.pool, `UPDATE device_bindings SET valid_until = '2026-11-05T00:00:00Z' WHERE device_id = $1 AND binding_revision = 1`, env.deviceA)
	mustExec(t, env.pool, `
		INSERT INTO device_bindings(
			device_id, member_id, overlay_scope, overlay_ip,
			binding_revision, valid_from, valid_until, peer_ref
		) VALUES ($1, $2, 'overlay-main', '100.64.0.10', 2, '2026-11-05T00:00:00Z', '2026-12-01T00:00:00Z', 'peer-b')
	`, env.deviceB, env.memberB)

	// An event for [2026-11-06T00:00:00Z, 2026-11-06T00:01:00Z) claiming deviceA at binding_revision 1 must be rejected
	// because binding_revision 1 expired at 2026-11-05T00:00:00Z.
	afterReassignStart := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	afterReassignEnd := time.Date(2026, 11, 6, 0, 1, 0, 0, time.UTC)
	evExpiredBinding := makeSignedEvent(t, env.identity1, 4, meterEpoch, afterReassignStart, afterReassignEnd, api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-11",
			UploadBytes:     777,
			DownloadBytes:   888,
		},
	})
	res, err = env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{evExpiredBinding},
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	})
	if err != nil {
		t.Fatalf("IngestBatch(evExpiredBinding): %v", err)
	}
	if res.Results[0].Status != api.UsageEventResultStatusRejected {
		t.Fatalf("expected expired binding event to be rejected, got %+v", res.Results[0])
	}
}

func TestAT19_MultiNodeAggregationDrillDownAdjustmentsAndRebuild(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)

	// Give memberA a second device (deviceA2) so we test multi-node + multi-device drill-down.
	deviceA2 := uuid.New()
	mustExec(t, env.pool, `INSERT INTO member_devices(id, member_id, deployment_id, peer_id, status) VALUES ($1, $2, 'dep-1', 'peer-a2', 'authorized')`, deviceA2, env.memberA)
	mustExec(t, env.pool, `
		INSERT INTO device_bindings(
			device_id, member_id, overlay_scope, overlay_ip,
			binding_revision, valid_from, valid_until, peer_ref
		) VALUES ($1, $2, 'overlay-main', '100.64.0.12', 1, '2026-01-01T00:00:00Z', '2026-12-01T00:00:00Z', 'peer-a2')
	`, deviceA2, env.memberA)

	// Node 1 reports usage for memberA on deviceA (1000 up, 2000 down).
	evNode1 := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     1000,
			DownloadBytes:   2000,
		},
	})
	if _, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{evNode1},
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	}); err != nil {
		t.Fatalf("IngestBatch node1: %v", err)
	}

	// Node 2 reports usage for memberA on deviceA (400 up, 600 down) and deviceA2 (500 up, 1500 down).
	evNode2 := makeSignedEvent(t, env.identity2, 1, meterEpoch, base.Add(time.Minute), base.Add(2*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     400,
			DownloadBytes:   600,
		},
		{
			MemberId:        env.memberA,
			DeviceId:        deviceA2,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     500,
			DownloadBytes:   1500,
		},
	})
	if _, err := env.svc.IngestBatch(ctx, env.identity2, api.UsageBatchRequest{
		NodeId:            env.node2,
		NodeIncarnationId: env.inc2,
		Events:            []api.UsageEvent{evNode2},
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	}); err != nil {
		t.Fatalf("IngestBatch node2: %v", err)
	}

	// Check cross-node period total and watermarks (§11.2 step 1).
	tot, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal: %v", err)
	}
	if tot.UploadBytes != 1900 || tot.DownloadBytes != 4100 || tot.TotalBytes != 6000 {
		t.Fatalf("want (1900, 4100, 6000), got (%d, %d, %d)", tot.UploadBytes, tot.DownloadBytes, tot.TotalBytes)
	}
	if len(tot.Watermarks) != 2 {
		t.Fatalf("expected 2 node stream watermarks, got %d", len(tot.Watermarks))
	}

	// Admin drill-down by node and device.
	dd, err := env.svc.GetMemberDrillDown(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberDrillDown: %v", err)
	}
	if len(dd.ByNode) != 2 || len(dd.ByDevice) != 2 || len(dd.ByNodeDevice) != 3 {
		t.Fatalf("unexpected drill-down counts: nodes=%d devices=%d entries=%d", len(dd.ByNode), len(dd.ByDevice), len(dd.ByNodeDevice))
	}

	// Apply an append-only adjustment (-200 upload, -100 download) and verify effective totals.
	adj, err := env.svc.CreateAdjustment(ctx, usage.AdjustmentCreate{
		MemberID:      env.memberA,
		Period:        "2026-10",
		UploadBytes:   -200,
		DownloadBytes: -100,
		Reason:        "Duplicate test traffic credit",
		Actor:         "admin-1",
	})
	if err != nil {
		t.Fatalf("CreateAdjustment: %v", err)
	}
	if adj.UploadBytes != -200 || adj.DownloadBytes != -100 {
		t.Fatalf("unexpected adjustment: %+v", adj)
	}

	totAfterAdj, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal after adjustment: %v", err)
	}
	if totAfterAdj.RawTotalBytes != 6000 || totAfterAdj.TotalBytes != 5700 {
		t.Fatalf("want raw=6000 effective=5700, got raw=%d effective=%d", totAfterAdj.RawTotalBytes, totAfterAdj.TotalBytes)
	}

	// Rebuild projection generation (§10.9, §15.4) and verify generation switches from 1 to 2
	// with identical totals and without mutating generation 1.
	rebuildRes, err := env.svc.RebuildProjection(ctx)
	if err != nil {
		t.Fatalf("RebuildProjection: %v", err)
	}
	if rebuildRes.PreviousGeneration != 1 || rebuildRes.NewGeneration != 2 || rebuildRes.ProjectedSamples != 3 {
		t.Fatalf("unexpected RebuildProjectionResult: %+v", rebuildRes)
	}

	totAfterRebuild, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal after rebuild: %v", err)
	}
	if totAfterRebuild.ProjectionGeneration != 2 || totAfterRebuild.RawTotalBytes != 6000 || totAfterRebuild.TotalBytes != 5700 {
		t.Fatalf("unexpected total after rebuild: %+v", totAfterRebuild)
	}
}

func TestAT30_RestoreSnapshotRecoveryReplayAndReplayUnavailable(t *testing.T) {
	// We use two isolated databases to model a central database restore:
	// - envLive receives events 1..5 and ACKs them to the node (which keeps 1..3 in its archive, while 4..5 are later lost locally).
	// - envRestored has only events 1..2 (the backup snapshot taken before 3..5 arrived).
	envLive := setupTestEnv(t)
	envRestored := setupTestEnv(t)
	ctx := context.Background()

	// Align IDs in envRestored with envLive so the same signed node events apply to both.
	envRestored.memberA = envLive.memberA
	envRestored.deviceA = envLive.deviceA
	envRestored.node1 = envLive.node1
	envRestored.inc1 = envLive.inc1
	envRestored.identity1 = envLive.identity1
	mustExec(t, envRestored.pool, `INSERT INTO members(id, status) VALUES ($1, 'active')`, envRestored.memberA)
	mustExec(t, envRestored.pool, `
		INSERT INTO entitlements(member_id, revision, device_limit, valid_from, effective_service_end)
		VALUES ($1, 1, 5, now() - interval '1 year', now() + interval '1 year')
	`, envRestored.memberA)
	mustExec(t, envRestored.pool, `INSERT INTO member_devices(id, member_id, deployment_id, peer_id, status) VALUES ($1, $2, 'dep-live', 'peer-live-a', 'authorized')`, envRestored.deviceA, envRestored.memberA)
	mustExec(t, envRestored.pool, `
		INSERT INTO device_bindings(
			device_id, member_id, overlay_scope, overlay_ip,
			binding_revision, valid_from, valid_until, peer_ref
		) VALUES ($1, $2, 'overlay-live', '100.64.1.10', 1, '2026-01-01T00:00:00Z', '2026-12-01T00:00:00Z', 'peer-live-a')
	`, envRestored.deviceA, envRestored.memberA)
	mustExec(t, envRestored.pool, `INSERT INTO nodes(id, lifecycle_state, provisioning_state) VALUES ($1, 'registered', 'ready')`, envRestored.node1)
	mustExec(t, envRestored.pool, `INSERT INTO node_incarnations(id, node_id, public_key_fingerprint, deployment_id, peer_id, os, arch) VALUES ($1, $2, decode(repeat('33', 32), 'hex'), 'dep-live', 'node-peer-live', 'ubuntu', 'amd64')`, envRestored.inc1, envRestored.node1)
	mustExec(t, envRestored.pool, `UPDATE nodes SET current_incarnation_id = $2, resource_version = resource_version + 1 WHERE id = $1`, envRestored.node1, envRestored.inc1)

	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 28, 14, 0, 0, 0, time.UTC)

	var nodeArchive []api.UsageEvent
	for seq := uint64(1); seq <= 5; seq++ {
		ev := makeSignedEvent(t, envLive.identity1, seq, meterEpoch,
			base.Add(time.Duration(seq-1)*time.Minute),
			base.Add(time.Duration(seq)*time.Minute),
			api.UsageQualityComplete, nil,
			[]api.UsageRecord{
				{
					MemberId:        envLive.memberA,
					DeviceId:        envLive.deviceA,
					BindingRevision: 1,
					PeriodId:        "2026-10",
					UploadBytes:     decstr.U64(seq * 100),
					DownloadBytes:   decstr.U64(seq * 200),
				},
			},
		)
		nodeArchive = append(nodeArchive, ev)
	}

	// Pre-snapshot: events 1..2 committed on both envLive and envRestored.
	if _, err := envLive.svc.IngestBatch(ctx, envLive.identity1, api.UsageBatchRequest{
		NodeId:            envLive.node1,
		NodeIncarnationId: envLive.inc1,
		Events:            nodeArchive[:2],
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	}); err != nil {
		t.Fatalf("envLive IngestBatch(1..2): %v", err)
	}
	if _, err := envRestored.svc.IngestBatch(ctx, envRestored.identity1, api.UsageBatchRequest{
		NodeId:            envRestored.node1,
		NodeIncarnationId: envRestored.inc1,
		Events:            nodeArchive[:2],
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	}); err != nil {
		t.Fatalf("envRestored IngestBatch(1..2): %v", err)
	}

	// Post-snapshot: events 3..5 arrive on envLive and get ACKed through 5, then envLive is lost and replaced by envRestored.
	liveRes, err := envLive.svc.IngestBatch(ctx, envLive.identity1, api.UsageBatchRequest{
		NodeId:            envLive.node1,
		NodeIncarnationId: envLive.inc1,
		Events:            nodeArchive[2:5],
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	})
	if err != nil || liveRes.HighestContiguousSeq != 5 {
		t.Fatalf("envLive IngestBatch(3..5): hcs=%d err=%v", liveRes.HighestContiguousSeq, err)
	}

	preRestoreState, err := envRestored.svc.GetLedgerState(ctx)
	if err != nil {
		t.Fatalf("envRestored GetLedgerState: %v", err)
	}
	oldEpoch := preRestoreState.LedgerEpoch

	// Step 1 (§17.3): Freeze ingest on the restored database.
	if err := envRestored.svc.FreezeIngest(ctx, "ops-admin", "Point-in-time restore drill"); err != nil {
		t.Fatalf("FreezeIngest: %v", err)
	}
	// While frozen, IngestBatch must return 503 and never ACK.
	_, err = envRestored.svc.IngestBatch(ctx, envRestored.identity1, api.UsageBatchRequest{
		NodeId:            envRestored.node1,
		NodeIncarnationId: envRestored.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(oldEpoch),
		Events:            nodeArchive[2:3],
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.HTTPStatus() != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 while frozen, got %v", err)
	}

	// Step 2 (§17.3): BeginRecovery creates a new ledger_epoch and signed recovery record,
	// recomputes watermarks from committed rows (hcs=2), and marks rollups reconciliation_in_progress.
	recState, err := envRestored.svc.BeginRecovery(ctx, usage.RecoveryRequest{
		RestoredTo:    base.Add(2 * time.Minute),
		AffectedSince: base,
		Reason:        "Restore from 14:02Z snapshot",
		Actor:         "ops-admin",
	})
	if err != nil {
		t.Fatalf("BeginRecovery: %v", err)
	}
	if recState.NewLedgerEpoch == oldEpoch || recState.Mode != usage.LedgerModeReconciling {
		t.Fatalf("unexpected RecoveryState: %+v", recState)
	}
	rawRecEnv, err := recState.SignedRecord.MarshalCanonical()
	if err != nil {
		t.Fatalf("marshal signed recovery record: %v", err)
	}
	verifiedRec, _, err := usage.VerifyRecoveryRecord(rawRecEnv, envRestored.epochTrustStore, envRestored.now)
	if err != nil {
		t.Fatalf("VerifyRecoveryRecord under ledger-epoch purpose failed: %v", err)
	}
	if verifiedRec.NewLedgerEpoch != recState.NewLedgerEpoch || verifiedRec.PreviousLedgerEpoch != oldEpoch {
		t.Fatalf("unexpected verified recovery record payload: %+v", verifiedRec)
	}
	if _, _, err := envelope.Verify(rawRecEnv, envRestored.trustStore, envelope.PurposeRecovery, envRestored.now); err == nil {
		t.Fatalf("expected recovery record verification against node recovery trust bundle to fail")
	}

	// Verify rollups are marked reconciliation_in_progress = true.
	uDuringRec, err := envRestored.svc.GetMemberUsage(ctx, usage.MemberUsageQuery{
		MemberID: envRestored.memberA,
		Period:   "2026-10",
	})
	if err != nil {
		t.Fatalf("GetMemberUsage during recovery: %v", err)
	}
	if !uDuringRec.Quality.ReconciliationInProgress {
		t.Fatalf("expected ReconciliationInProgress=true during recovery")
	}

	// Step 3: Node sending a batch with oldEpoch gets 409 LEDGER_EPOCH_MISMATCH.
	_, err = envRestored.svc.IngestBatch(ctx, envRestored.identity1, api.UsageBatchRequest{
		NodeId:            envRestored.node1,
		NodeIncarnationId: envRestored.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(oldEpoch),
		Events:            nodeArchive[2:3],
	})
	if !errors.As(err, &ae) || ae.HTTPStatus() != http.StatusConflict || ae.Code != apierr.LedgerEpochMismatch {
		t.Fatalf("expected 409 LEDGER_EPOCH_MISMATCH, got %v", err)
	}
	if ae.Details["current_ledger_epoch"] != recState.NewLedgerEpoch.String() {
		t.Fatalf("expected current_ledger_epoch detail %s, got %+v", recState.NewLedgerEpoch, ae.Details)
	}

	// Step 4: Node fetches signed checkpoint and verifies the recovery signature with its trust store.
	cpEnv, err := envRestored.svc.GetCheckpoint(ctx, envRestored.identity1)
	if err != nil {
		t.Fatalf("GetCheckpoint after BeginRecovery: %v", err)
	}
	rawCPEnv, err := json.Marshal(cpEnv)
	if err != nil {
		t.Fatalf("marshal checkpoint envelope: %v", err)
	}
	if _, _, err := envelope.Verify(rawCPEnv, envRestored.trustStore, envelope.PurposeRecovery, envRestored.now); err != nil {
		t.Fatalf("node verification of signed checkpoint failed: %v", err)
	}
	if cpEnv.Signed.HighestContiguousSeq != 2 || cpEnv.Signed.LedgerEpoch != recState.NewLedgerEpoch {
		t.Fatalf("unexpected checkpoint signed content: %+v", cpEnv.Signed)
	}
	if (!cpEnv.Signed.RecoveryRecordId.IsSpecified() || cpEnv.Signed.RecoveryRecordId.IsNull()) || nullableValue(cpEnv.Signed.RecoveryRecordId) != recState.RecoveryID {
		t.Fatalf("expected recovery_record_id=%s, got %v", recState.RecoveryID, cpEnv.Signed.RecoveryRecordId)
	}

	// Step 5: Node replays events 1..3 under the new ledger_epoch; 1..2 are deduplicated (no double count), 3 is accepted.
	newEpoch := recState.NewLedgerEpoch
	replayRes, err := envRestored.svc.IngestBatch(ctx, envRestored.identity1, api.UsageBatchRequest{
		NodeId:            envRestored.node1,
		NodeIncarnationId: envRestored.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(newEpoch),
		Events:            nodeArchive[:3],
	})
	if err != nil {
		t.Fatalf("IngestBatch replay(1..3): %v", err)
	}
	if replayRes.Results[0].Status != api.UsageEventResultStatusDuplicate ||
		replayRes.Results[1].Status != api.UsageEventResultStatusDuplicate ||
		replayRes.Results[2].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("unexpected replay results: %+v", replayRes.Results)
	}
	if replayRes.HighestContiguousSeq != 3 {
		t.Fatalf("expected hcs=3 after replaying 1..3, got %d", replayRes.HighestContiguousSeq)
	}

	// Step 6: Suppose sequences 4..5 were purged from the node's archive before the restore;
	// node reports replay_unavailable for [4, 5] and sends new event 6.
	ev6 := makeSignedEvent(t, envRestored.identity1, 6, meterEpoch,
		base.Add(5*time.Minute), base.Add(6*time.Minute),
		api.UsageQualityComplete, nil,
		[]api.UsageRecord{
			{
				MemberId:        envRestored.memberA,
				DeviceId:        envRestored.deviceA,
				BindingRevision: 1,
				PeriodId:        "2026-10",
				UploadBytes:     600,
				DownloadBytes:   1200,
			},
		},
	)
	unavail := []api.ReplayUnavailableRange{
		{
			StartSeq: 4,
			EndSeq:   5,
			Reason:   api.ReplayUnavailableRangeReasonRetentionExpired,
		},
	}
	gapRes, err := envRestored.svc.IngestBatch(ctx, envRestored.identity1, api.UsageBatchRequest{
		NodeId:            envRestored.node1,
		NodeIncarnationId: envRestored.inc1,
		LedgerEpoch:       presence.NewNullableWithValue(newEpoch),
		ReplayUnavailable: &unavail,
		Events:            []api.UsageEvent{ev6},
	})
	if err != nil {
		t.Fatalf("IngestBatch with replay_unavailable(4..5) and event 6: %v", err)
	}
	if gapRes.HighestContiguousSeq != 6 || gapRes.MaxSeenSeq != 6 {
		t.Fatalf("expected hcs=6 max=6 after replay_unavailable(4..5) + event 6, got hcs=%d max=%d", gapRes.HighestContiguousSeq, gapRes.MaxSeenSeq)
	}

	// Step 7: CompleteRecovery clears reconciliation_in_progress while keeping gap=true and never inventing bytes for 4..5.
	if err := envRestored.svc.CompleteRecovery(ctx, "ops-admin"); err != nil {
		t.Fatalf("CompleteRecovery: %v", err)
	}

	finalUsage, err := envRestored.svc.GetMemberUsage(ctx, usage.MemberUsageQuery{
		MemberID: envRestored.memberA,
		Period:   "2026-10",
	})
	if err != nil {
		t.Fatalf("GetMemberUsage after CompleteRecovery: %v", err)
	}
	// Bytes must be exact sum of events 1, 2, 3, and 6:
	// Upload = 100 + 200 + 300 + 600 = 1200; Download = 200 + 400 + 600 + 1200 = 2400; Total = 3600.
	if finalUsage.Totals.UploadBytes != 1200 || finalUsage.Totals.DownloadBytes != 2400 || finalUsage.Totals.TotalBytes != 3600 {
		t.Fatalf("unexpected totals after recovery: %+v", finalUsage.Totals)
	}
	if !finalUsage.Quality.Gap {
		t.Fatalf("expected Quality.Gap=true after replay_unavailable")
	}
	if finalUsage.Quality.ReconciliationInProgress {
		t.Fatalf("expected ReconciliationInProgress=false after CompleteRecovery")
	}

	// Verify recovery audit rows use fleet_config:00000000-0000-0000-0000-000000000001 and carry
	// usage_ledger_id (and recovery_id) in after_diff metadata (Decision D-051).
	auditRows, err := envRestored.pool.Query(ctx, `
		SELECT action, resource, after_diff
		FROM audit_events
		WHERE action IN ('usage_ledger.freeze', 'usage_ledger.begin_recovery', 'usage_ledger.complete_recovery')
		ORDER BY occurred_at ASC, id ASC
	`)
	if err != nil {
		t.Fatalf("query recovery audit_events: %v", err)
	}
	defer auditRows.Close()
	var seenAuditActions []string
	for auditRows.Next() {
		var action, resource string
		var afterRaw []byte
		if err := auditRows.Scan(&action, &resource, &afterRaw); err != nil {
			t.Fatalf("scan recovery audit row: %v", err)
		}
		seenAuditActions = append(seenAuditActions, action)
		if resource != "fleet_config:00000000-0000-0000-0000-000000000001" {
			t.Fatalf("%s audit resource = %q, want fleet_config:00000000-0000-0000-0000-000000000001 (D-051)", action, resource)
		}
		var meta map[string]any
		if err := json.Unmarshal(afterRaw, &meta); err != nil {
			t.Fatalf("unmarshal %s after_diff: %v", action, err)
		}
		if got, _ := meta["usage_ledger_id"].(string); got != "00000000-0000-0000-0000-000000000130" {
			t.Fatalf("%s metadata usage_ledger_id = %q, want 00000000-0000-0000-0000-000000000130", action, got)
		}
		if action != "usage_ledger.freeze" {
			if got, _ := meta["recovery_id"].(string); got != recState.RecoveryID.String() {
				t.Fatalf("%s metadata recovery_id = %q, want %s", action, got, recState.RecoveryID)
			}
		}
	}
	if err := auditRows.Err(); err != nil {
		t.Fatalf("iterate recovery audit rows: %v", err)
	}
	if len(seenAuditActions) != 3 {
		t.Fatalf("expected 3 recovery audit actions, got %v", seenAuditActions)
	}
}

func TestMTR04_PrecisionAbove2Pow53EndToEnd(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

	// 2^53 + 1 = 9007199254740993, 2^53 + 3 = 9007199254740995.
	// Sum = 18014398509481988 (which would round to 18014398509481984 if passed through float64!).
	const (
		upLarge   uint64 = 9007199254740993
		downLarge uint64 = 9007199254740995
		sumLarge  uint64 = upLarge + downLarge
	)

	ev := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     decstr.U64(upLarge),
			DownloadBytes:   decstr.U64(downLarge),
		},
	})

	if _, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{ev},
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	}); err != nil {
		t.Fatalf("IngestBatch(>2^53): %v", err)
	}

	u, err := env.svc.GetMemberUsage(ctx, usage.MemberUsageQuery{
		MemberID:    env.memberA,
		Period:      "2026-10",
		Granularity: usage.GranularityDay,
	})
	if err != nil {
		t.Fatalf("GetMemberUsage: %v", err)
	}
	if u.Totals.UploadBytes.String() != "9007199254740993" ||
		u.Totals.DownloadBytes.String() != "9007199254740995" ||
		u.Totals.TotalBytes.String() != "18014398509481988" {
		t.Fatalf("precision lost in Totals: %+v", u.Totals)
	}
	if len(u.Days) != 1 || u.Days[0].TotalBytes.String() != "18014398509481988" {
		t.Fatalf("precision lost in Days: %+v", u.Days)
	}

	if _, err := env.svc.RebuildProjection(ctx); err != nil {
		t.Fatalf("RebuildProjection(>2^53): %v", err)
	}
	tot, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal after rebuild: %v", err)
	}
	if uint64(tot.TotalBytes) != sumLarge {
		t.Fatalf("precision lost after rebuild: got %s, want %d", tot.TotalBytes.String(), sumLarge)
	}
}

func TestMTR03_ConcurrentSameStreamIngest(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)

	const totalEvents = 12
	events := make([]api.UsageEvent, totalEvents)
	for i := 0; i < totalEvents; i++ {
		seq := uint64(i + 1)
		events[i] = makeSignedEvent(t, env.identity1, seq, meterEpoch,
			base.Add(time.Duration(i)*time.Minute),
			base.Add(time.Duration(i+1)*time.Minute),
			api.UsageQualityComplete, nil,
			[]api.UsageRecord{
				{
					MemberId:        env.memberA,
					DeviceId:        env.deviceA,
					BindingRevision: 1,
					PeriodId:        "2026-10",
					UploadBytes:     100,
					DownloadBytes:   200,
				},
			},
		)
	}

	// Launch 4 concurrent workers submitting overlapping subsets of the same stream.
	batches := [][]api.UsageEvent{
		{events[0], events[2], events[4], events[6], events[8], events[10]},
		{events[1], events[3], events[5], events[7], events[9], events[11]},
		{events[0], events[1], events[2], events[3], events[4], events[5]},
		{events[6], events[7], events[8], events[9], events[10], events[11]},
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(batches))
	for _, b := range batches {
		bCopy := b
		wg.Go(func() {
			_, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
				NodeId:            env.node1,
				NodeIncarnationId: env.inc1,
				Events:            bCopy,
				LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
			})
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent IngestBatch failed: %v", err)
		}
	}

	cpEnv, err := env.svc.GetCheckpoint(ctx, env.identity1)
	if err != nil {
		t.Fatalf("GetCheckpoint: %v", err)
	}
	if cpEnv.Signed.HighestContiguousSeq != totalEvents || cpEnv.Signed.MaxSeenSeq != totalEvents || len(cpEnv.Signed.MissingRanges) != 0 {
		t.Fatalf("unexpected checkpoint after concurrent ingest: %+v", cpEnv.Signed)
	}

	tot, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal: %v", err)
	}
	if tot.UploadBytes != decstr.U64(totalEvents*100) || tot.DownloadBytes != decstr.U64(totalEvents*200) {
		t.Fatalf("expected (%d, %d), got (%d, %d)", totalEvents*100, totalEvents*200, tot.UploadBytes, tot.DownloadBytes)
	}

	// Verify fmt is used if needed.
	_ = fmt.Sprintf
}

func TestF04_ReplayUnavailableTombstonesAndRetentionPurge(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)

	ev1 := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     500,
			DownloadBytes:   1500,
		},
	})
	ev4 := makeSignedEvent(t, env.identity1, 4, meterEpoch, base.Add(3*time.Minute), base.Add(4*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     700,
			DownloadBytes:   1300,
		},
	})

	// 1. Ingest events 1 and 4 with gap at [2, 3].
	res, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{ev1, ev4},
		LedgerEpoch:       presence.NewNullNullable[uuid.UUID](),
	})
	if err != nil {
		t.Fatalf("IngestBatch(1, 4): %v", err)
	}
	if res.HighestContiguousSeq != 1 || res.MaxSeenSeq != 4 {
		t.Fatalf("expected hcs=1 max=4, got hcs=%d max=%d", res.HighestContiguousSeq, res.MaxSeenSeq)
	}

	// 2. Decision D-027: In normal mode, replay_unavailable must be rejected with 409 STATE_CONFLICT.
	state, err := env.svc.GetLedgerState(ctx)
	if err != nil {
		t.Fatalf("GetLedgerState: %v", err)
	}
	err = env.svc.RecordReplayUnavailable(ctx, env.identity1, state.LedgerEpoch, []api.ReplayUnavailableRange{
		{
			StartSeq: 2,
			EndSeq:   3,
			Reason:   api.ReplayUnavailableRangeReasonRetentionExpired,
		},
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.HTTPStatus() != http.StatusConflict || ae.Code != apierr.StateConflict {
		t.Fatalf("expected 409 STATE_CONFLICT for replay_unavailable in normal mode, got %v", err)
	}

	// Enter recovery (reconciling mode) and record replay_unavailable for [2, 3]:
	// inserts 0-byte tombstones (record_count = 0, 0 usage_samples rows, 0 usage_event_payloads rows)
	// and advances checkpoint to hcs=4, max=4.
	// This proves 0-byte tombstones satisfy F-04's check_usage_event_contents and check_usage_checkpoint triggers.
	rec1, err := env.svc.BeginRecovery(ctx, usage.RecoveryRequest{
		RestoredTo:    env.now,
		AffectedSince: base,
		Reason:        "Enter reconciling mode for replay_unavailable",
		Actor:         "ops-admin",
	})
	if err != nil {
		t.Fatalf("BeginRecovery(1): %v", err)
	}
	if err := env.svc.RecordReplayUnavailable(ctx, env.identity1, rec1.NewLedgerEpoch, []api.ReplayUnavailableRange{
		{
			StartSeq: 2,
			EndSeq:   3,
			Reason:   api.ReplayUnavailableRangeReasonRetentionExpired,
		},
	}); err != nil {
		t.Fatalf("RecordReplayUnavailable([2, 3]) in reconciling mode: %v", err)
	}

	var tombstoneCount, tombstoneSamples, tombstonePayloads int
	if err := env.pool.QueryRow(ctx, `
		SELECT count(*),
		       (SELECT count(*) FROM usage_samples s JOIN usage_events e ON e.id = s.event_id WHERE e.node_id = $1 AND e.kind = 'replay_unavailable'),
		       (SELECT count(*) FROM usage_event_payloads p JOIN usage_events e ON e.id = p.event_id WHERE e.node_id = $1 AND e.kind = 'replay_unavailable')
		FROM usage_events
		WHERE node_id = $1 AND kind = 'replay_unavailable' AND record_count = 0
	`, env.node1).Scan(&tombstoneCount, &tombstoneSamples, &tombstonePayloads); err != nil {
		t.Fatalf("query tombstones: %v", err)
	}
	if tombstoneCount != 2 || tombstoneSamples != 0 || tombstonePayloads != 0 {
		t.Fatalf("expected 2 zero-byte tombstones with 0 samples and 0 payloads, got count=%d samples=%d payloads=%d",
			tombstoneCount, tombstoneSamples, tombstonePayloads)
	}

	cpEnv, err := env.svc.GetCheckpoint(ctx, env.identity1)
	if err != nil {
		t.Fatalf("GetCheckpoint: %v", err)
	}
	if cpEnv.Signed.HighestContiguousSeq != 4 || cpEnv.Signed.MaxSeenSeq != 4 || len(cpEnv.Signed.MissingRanges) != 0 {
		t.Fatalf("expected checkpoint hcs=4 max=4 missing=[], got %+v", cpEnv.Signed)
	}

	// 3. Rotate ledger_epoch via BeginRecovery while tombstones are present;
	// proves resetStreamCheckpointForNewEpochInTx satisfies F-04 on epoch rotation.
	recState, err := env.svc.BeginRecovery(ctx, usage.RecoveryRequest{
		RestoredTo:    env.now,
		AffectedSince: base,
		Reason:        "F-04 epoch rotation with tombstones",
		Actor:         "ops-admin",
	})
	if err != nil {
		t.Fatalf("BeginRecovery with tombstones: %v", err)
	}
	if recState.RecomputedStreams != 1 {
		t.Fatalf("expected 1 recomputed stream, got %d", recState.RecomputedStreams)
	}
	if err := env.svc.CompleteRecovery(ctx, "ops-admin"); err != nil {
		t.Fatalf("CompleteRecovery: %v", err)
	}

	// 4. Ingest a clean historical event on node2 in 2026-01 (more than 90 days ago)
	// and verify F-03 retention purge behavior via PurgeRetainedPayloads.
	oldTime := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	env.now = oldTime
	evOld := makeSignedEvent(t, env.identity2, 1, uuid.New(), oldTime, oldTime.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberB,
			DeviceId:        env.deviceB,
			BindingRevision: 1,
			PeriodId:        "2026-01",
			UploadBytes:     1000,
			DownloadBytes:   2000,
		},
	})
	if _, err := env.svc.IngestBatch(ctx, env.identity2, api.UsageBatchRequest{
		NodeId:            env.node2,
		NodeIncarnationId: env.inc2,
		LedgerEpoch:       presence.NewNullableWithValue(recState.NewLedgerEpoch),
		Events:            []api.UsageEvent{evOld},
	}); err != nil {
		t.Fatalf("IngestBatch(evOld on node2): %v", err)
	}
	env.now = time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)

	// Cutoff within 90 days of clock_timestamp() is rejected by F-03 retention policy.
	if _, err := env.svc.PurgeRetainedPayloads(ctx, time.Now().UTC(), "too recent cutoff"); err == nil {
		t.Fatalf("expected PurgeRetainedPayloads to reject cutoff inside 90-day retention window")
	}

	// Cutoff at 2026-02-01Z (>90 days ago) purges the clean 2026-01 payload while leaving
	// the 2026-10 gap-period payloads intact.
	purged, err := env.svc.PurgeRetainedPayloads(ctx, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), "scheduled 90-day raw usage payload retention")
	if err != nil {
		t.Fatalf("PurgeRetainedPayloads(2026-02-01Z): %v", err)
	}
	if purged != 1 {
		t.Fatalf("expected 1 clean historical payload purged, got %d", purged)
	}

	// 5. Verify direct DELETE on usage_checkpoints and usage_rollups is rejected (F-02, F-04).
	if _, err := env.pool.Exec(ctx, `DELETE FROM usage_checkpoints WHERE node_id = $1`, env.node1); err == nil {
		t.Fatalf("expected DELETE FROM usage_checkpoints to be rejected by F-04 trigger")
	}
	if _, err := env.pool.Exec(ctx, `DELETE FROM usage_rollups WHERE member_id = $1`, env.memberA); err == nil {
		t.Fatalf("expected DELETE FROM usage_rollups to be rejected by F-02 trigger")
	}
}

func TestMTR03_CheckpointInitRaceDeterministic(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)

	ev1 := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     100,
			DownloadBytes:   200,
		},
	})
	ev2 := makeSignedEvent(t, env.identity1, 2, meterEpoch, base.Add(time.Minute), base.Add(2*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     300,
			DownloadBytes:   400,
		},
	})
	ev3 := makeSignedEvent(t, env.identity1, 3, meterEpoch, base.Add(2*time.Minute), base.Add(3*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     500,
			DownloadBytes:   600,
		},
	})

	// Deterministically reproduce the fresh-stream checkpoint init race (D-024):
	// 1. Slow worker (ingesting ev2) sees pgx.ErrNoRows on the initial selectCheckpointForUpdateInTx
	//    and pauses at AfterInitialMiss.
	// 2. While slow worker is paused at AfterInitialMiss, fast worker 1 (ingesting ev1) creates the
	//    checkpoint row (1, 1) and commits ev1 (seq=1).
	// 3. Slow worker is released and acquires node_incarnations.
	//    - With the post-#21 re-check of selectCheckpointForUpdateInTx: slow worker finds and locks
	//      the existing checkpoint row (1, 1), skips BeforeInsert, commits ev2 (seq=2), and the
	//      stream checkpoint immediately reaches highest_contiguous_seq=2, max_seen_seq=2 with no backfill.
	//    - Without the re-check (pre-#21): slow worker computes watermark (1, 1) from usage_events and
	//      enters BeforeInsert, where fast worker 2 (ingesting ev3 via the fast path) commits seq=3 and
	//      advances max_seen_seq to 3; slow worker's BEFORE INSERT check_usage_checkpoint trigger then
	//      fails with SQLSTATE 23514.
	initialMissEntered := make(chan struct{})
	releaseInitialMiss := make(chan struct{})
	var missOnce, insertOnce sync.Once

	tracedCtx := usage.WithCheckpointInitTrace(ctx, &usage.CheckpointInitTrace{
		AfterInitialMiss: func() {
			missOnce.Do(func() {
				close(initialMissEntered)
				<-releaseInitialMiss
			})
		},
		BeforeInsert: func() {
			insertOnce.Do(func() {
				if _, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
					NodeId:            env.node1,
					NodeIncarnationId: env.inc1,
					Events:            []api.UsageEvent{ev3},
				}); err != nil {
					t.Errorf("concurrent fast-path IngestBatch(ev3) in BeforeInsert failed: %v", err)
				}
			})
		},
	})

	errCh := make(chan error, 1)
	go func() {
		_, err := env.svc.IngestBatch(tracedCtx, env.identity1, api.UsageBatchRequest{
			NodeId:            env.node1,
			NodeIncarnationId: env.inc1,
			Events:            []api.UsageEvent{ev2},
		})
		errCh <- err
	}()

	<-initialMissEntered
	// While ev2 is paused after its initial checkpoint miss, ingest ev1 so that ev1 creates
	// and commits the stream's usage_checkpoints row (1, 1) and usage_events row (seq=1).
	if _, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{ev1},
	}); err != nil {
		t.Fatalf("IngestBatch(ev1) during AfterInitialMiss: %v", err)
	}
	close(releaseInitialMiss)

	if err := <-errCh; err != nil {
		t.Fatalf("deterministic checkpoint init race failed: %v", err)
	}

	cpEnv, err := env.svc.GetCheckpoint(ctx, env.identity1)
	if err != nil {
		t.Fatalf("GetCheckpoint after race: %v", err)
	}
	if cpEnv.Signed.HighestContiguousSeq != 2 || cpEnv.Signed.MaxSeenSeq != 2 {
		t.Fatalf("expected hcs=2 max=2 directly after ev1+ev2 race, got hcs=%d max=%d", cpEnv.Signed.HighestContiguousSeq, cpEnv.Signed.MaxSeenSeq)
	}
}

func Test17_3_FreezeFencesInFlightIngest(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

	ev := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     10,
			DownloadBytes:   20,
		},
	})

	var freezeErr error
	traced := usage.WithCheckpointInitTrace(ctx, &usage.CheckpointInitTrace{
		AfterInitialMiss: func() {
			freezeErr = env.svc.FreezeIngest(ctx, "ops-admin", "fence in-flight ingest")
		},
	})

	_, err := env.svc.IngestBatch(traced, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{ev},
	})
	if freezeErr != nil {
		t.Fatalf("FreezeIngest during checkpoint init: %v", freezeErr)
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.DependencyUnavailable {
		t.Fatalf("expected 503 DEPENDENCY_UNAVAILABLE when frozen mid-ingest, got %v", err)
	}
	var eventCount int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM usage_events WHERE node_id = $1`, env.node1).Scan(&eventCount); err != nil {
		t.Fatalf("count usage_events: %v", err)
	}
	if eventCount != 0 {
		t.Fatalf("expected 0 committed usage_events after mid-ingest freeze, got %d", eventCount)
	}
}

func TestAT16_GapEventDoesNotFlagUnrelatedMembers(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

	gap := &api.UsageGap{
		LastCheckpointAt: presence.NewNullNullable[time.Time](),
		Reason:           api.UsageGapReasonPowerLoss,
	}
	ev := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityGap, gap, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     0,
			DownloadBytes:   0,
		},
	})
	res, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{ev},
	})
	if err != nil {
		t.Fatalf("IngestBatch(gap): %v", err)
	}
	if res.Results[0].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("expected gap event accepted, got %s", res.Results[0].Status)
	}

	// Member A was listed on the gap event -> has gap=true.
	totA, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal(memberA): %v", err)
	}
	if !totA.Gap {
		t.Fatalf("expected memberA Gap=true")
	}

	// Member B has an overlapping device binding in the database, but node1 never saw memberB ->
	// must NOT have a zero-byte rollup row or Gap=true.
	var rollupCountB int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM usage_rollups WHERE member_id = $1`, env.memberB).Scan(&rollupCountB); err != nil {
		t.Fatalf("count memberB rollups: %v", err)
	}
	if rollupCountB != 0 {
		t.Fatalf("expected 0 rollup rows for unrelated memberB, got %d", rollupCountB)
	}
	totB, err := env.svc.GetMemberPeriodTotal(ctx, env.memberB, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal(memberB): %v", err)
	}
	if totB.Gap {
		t.Fatalf("expected unrelated memberB Gap=false, got true")
	}
}

func Test17_3_ReplayUnavailableModeAndMemberScoping(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

	// Member A has usage on node1 only; node2 has never seen member A.
	ev := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     100,
			DownloadBytes:   50,
		},
	})
	if _, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{ev},
	}); err != nil {
		t.Fatalf("seed memberA on node1: %v", err)
	}

	state, err := env.svc.GetLedgerState(ctx)
	if err != nil {
		t.Fatalf("GetLedgerState: %v", err)
	}
	ranges := []api.ReplayUnavailableRange{{
		StartSeq: 1,
		EndSeq:   1,
		Reason:   api.ReplayUnavailableRangeReasonLocalDataLost,
	}}

	// 1. Decision D-027: In normal mode, IngestBatch with replay_unavailable is rejected with 409 STATE_CONFLICT.
	_, err = env.svc.IngestBatch(ctx, env.identity2, api.UsageBatchRequest{
		NodeId:            env.node2,
		NodeIncarnationId: env.inc2,
		LedgerEpoch:       presence.NewNullableWithValue(state.LedgerEpoch),
		ReplayUnavailable: &ranges,
	})
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.HTTPStatus() != http.StatusConflict || ae.Code != apierr.StateConflict {
		t.Fatalf("expected 409 STATE_CONFLICT for replay_unavailable in normal mode, got %v", err)
	}
	var tombstones int
	if err := env.pool.QueryRow(ctx, `SELECT count(*) FROM usage_events WHERE node_id = $1`, env.node2).Scan(&tombstones); err != nil {
		t.Fatalf("count node2 events: %v", err)
	}
	if tombstones != 0 {
		t.Fatalf("expected 0 tombstones stored after rejected normal-mode replay_unavailable, got %d", tombstones)
	}

	// 2. Enter reconciling mode; node2 reports replay_unavailable. Because memberA never used node2,
	// memberA's rollup and period total must NOT be flagged with gap=true.
	rec, err := env.svc.BeginRecovery(ctx, usage.RecoveryRequest{
		RestoredTo:    env.now,
		AffectedSince: base,
		Reason:        "Test replay_unavailable member scoping",
		Actor:         "ops-admin",
	})
	if err != nil {
		t.Fatalf("BeginRecovery: %v", err)
	}
	res, err := env.svc.IngestBatch(ctx, env.identity2, api.UsageBatchRequest{
		NodeId:            env.node2,
		NodeIncarnationId: env.inc2,
		LedgerEpoch:       presence.NewNullableWithValue(rec.NewLedgerEpoch),
		ReplayUnavailable: &ranges,
	})
	if err != nil {
		t.Fatalf("IngestBatch(replay_unavailable in reconciling mode): %v", err)
	}
	if res.HighestContiguousSeq != 1 {
		t.Fatalf("expected node2 hcs=1 after replay_unavailable in reconciling mode, got %s", res.HighestContiguousSeq)
	}

	totA, err := env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal(memberA): %v", err)
	}
	if totA.Gap {
		t.Fatalf("memberA Gap=true after node2 replay_unavailable; memberA never used node2")
	}
}

func TestMTR04_Uint64OverflowFailsClosedOnRead(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

	// 1. Ingest a single sample at math.MaxUint64 for memberB, then add a positive adjustment of +1:
	// reading memberB's period total must fail closed with decstr.ErrOverflow and usage_uint64_overflow.
	evB := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberB,
			DeviceId:        env.deviceB,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     decstr.U64(^uint64(0)),
			DownloadBytes:   0,
		},
	})
	if _, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{evB},
	}); err != nil {
		t.Fatalf("IngestBatch(evB max uint64): %v", err)
	}
	// Exact MaxUint64 without overflow succeeds and returns 18446744073709551615.
	totBExact, err := env.svc.GetMemberPeriodTotal(ctx, env.memberB, "2026-10")
	if err != nil {
		t.Fatalf("GetMemberPeriodTotal(memberB exact MaxUint64): %v", err)
	}
	if totBExact.UploadBytes != decstr.U64(^uint64(0)) {
		t.Fatalf("expected exact MaxUint64 upload_bytes, got %s", totBExact.UploadBytes)
	}
	if _, err := env.svc.CreateAdjustment(ctx, usage.AdjustmentCreate{
		MemberID:      env.memberB,
		Period:        "2026-10",
		UploadBytes:   1,
		DownloadBytes: 0,
		Reason:        "push adjusted upload total above uint64",
		Actor:         "ops-admin",
	}); err != nil {
		t.Fatalf("CreateAdjustment(+1): %v", err)
	}
	_, err = env.svc.GetMemberPeriodTotal(ctx, env.memberB, "2026-10")
	if !errors.Is(err, decstr.ErrOverflow) {
		t.Fatalf("expected errors.Is(err, decstr.ErrOverflow) on adjusted overflow, got %v", err)
	}
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.InternalError || ae.Details["detail_code"] != usage.ErrCodeUint64Overflow {
		t.Fatalf("expected InternalError with detail_code=%s, got %+v", usage.ErrCodeUint64Overflow, err)
	}

	// 2. Two samples of math.MaxUint64 for memberA accumulate in NUMERIC(40,0) on ingest,
	// and fail closed with decstr.ErrOverflow and usage_uint64_overflow on read.
	evA1 := makeSignedEvent(t, env.identity1, 2, meterEpoch, base.Add(time.Minute), base.Add(2*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     decstr.U64(^uint64(0)),
			DownloadBytes:   0,
		},
	})
	evA2 := makeSignedEvent(t, env.identity1, 3, meterEpoch, base.Add(2*time.Minute), base.Add(3*time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        env.memberA,
			DeviceId:        env.deviceA,
			BindingRevision: 1,
			PeriodId:        "2026-10",
			UploadBytes:     decstr.U64(^uint64(0)),
			DownloadBytes:   0,
		},
	})
	if _, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{evA1, evA2},
	}); err != nil {
		t.Fatalf("IngestBatch(evA1, evA2): %v", err)
	}
	_, err = env.svc.GetMemberPeriodTotal(ctx, env.memberA, "2026-10")
	if !errors.Is(err, decstr.ErrOverflow) {
		t.Fatalf("expected errors.Is(err, decstr.ErrOverflow) on rollup sum overflow, got %v", err)
	}
	if !errors.As(err, &ae) || ae.Code != apierr.InternalError || ae.Details["detail_code"] != usage.ErrCodeUint64Overflow {
		t.Fatalf("expected InternalError with detail_code=%s, got %+v", usage.ErrCodeUint64Overflow, err)
	}
}

func Test17_3_LedgerEpochSigningPurposeSeparation(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// A service configured only with a recovery-purpose signer must reject BeginRecovery
	// because recovery keys are not authorized for envelope.PurposeLedgerEpoch.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key: %v", err)
	}
	_ = pub
	recoveryOnlySvc, err := usage.NewService(usage.Config{
		Pool:   env.pool,
		Signer: usage.NewRecoverySigner("recovery-only-key", priv),
		Clock:  func() time.Time { return env.now },
	})
	if err != nil {
		t.Fatalf("NewService(recoveryOnly): %v", err)
	}
	if _, err := recoveryOnlySvc.BeginRecovery(ctx, usage.RecoveryRequest{
		Reason: "attempt epoch rotation with recovery-only signer",
		Actor:  "ops-admin",
	}); err == nil {
		t.Fatalf("expected BeginRecovery to fail when only a recovery-purpose signer is configured")
	}

	// With LedgerEpochSigner configured on env.svc, BeginRecovery succeeds and signs under PurposeLedgerEpoch.
	rec, err := env.svc.BeginRecovery(ctx, usage.RecoveryRequest{
		Reason: "authorized epoch rotation with ledger-epoch signer",
		Actor:  "ops-admin",
	})
	if err != nil {
		t.Fatalf("BeginRecovery with LedgerEpochSigner: %v", err)
	}
	raw, err := rec.SignedRecord.MarshalCanonical()
	if err != nil {
		t.Fatalf("MarshalCanonical: %v", err)
	}
	payload, keyID, err := usage.VerifyRecoveryRecord(raw, env.epochTrustStore, env.now)
	if err != nil {
		t.Fatalf("VerifyRecoveryRecord: %v", err)
	}
	if keyID != "test-ledger-epoch-key-1" || payload.NewLedgerEpoch != rec.NewLedgerEpoch {
		t.Fatalf("unexpected verified payload=%+v keyID=%s", payload, keyID)
	}
}

func Test10_6_EmptyCompleteEventAdvancesACK(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()
	meterEpoch := uuid.New()
	base := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

	ev := makeSignedEvent(t, env.identity1, 1, meterEpoch, base, base.Add(time.Minute), api.UsageQualityComplete, nil, []api.UsageRecord{})
	res, err := env.svc.IngestBatch(ctx, env.identity1, api.UsageBatchRequest{
		NodeId:            env.node1,
		NodeIncarnationId: env.inc1,
		Events:            []api.UsageEvent{ev},
	})
	if err != nil {
		t.Fatalf("IngestBatch(empty complete event): %v", err)
	}
	if res.Results[0].Status != api.UsageEventResultStatusAccepted || res.HighestContiguousSeq != 1 {
		t.Fatalf("expected empty complete event accepted with hcs=1, got status=%s hcs=%s", res.Results[0].Status, res.HighestContiguousSeq)
	}
}
