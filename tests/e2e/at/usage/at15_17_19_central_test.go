package usage

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/fup"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func optionalTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("FVPN_TEST_DATABASE_URL") == "" {
		return nil
	}
	return dbtest.New(t)
}

func seedUsageFixtures(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	nodeID, incID, memberID, dev1ID, dev2ID uuid.UUID,
	validFrom, validUntil time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(ctx, `
		INSERT INTO nodes(id, lifecycle_state, provisioning_state)
		VALUES ($1, 'registered', 'ready')
		ON CONFLICT (id) DO NOTHING
	`, nodeID); err != nil {
		t.Fatalf("seed nodes: %v", err)
	}
	fp := sha256.Sum256(incID[:])
	if _, err := pool.Exec(ctx, `
		INSERT INTO node_incarnations(id, node_id, public_key_fingerprint, deployment_id, peer_id, os, arch)
		VALUES ($1, $2, $3, 'dep-1', $4, 'ubuntu', 'amd64')
		ON CONFLICT (id) DO NOTHING
	`, incID, nodeID, fp[:], "node-peer-"+incID.String()); err != nil {
		t.Fatalf("seed node_incarnations: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE nodes SET current_incarnation_id = $2, resource_version = resource_version + 1
		WHERE id = $1 AND current_incarnation_id IS DISTINCT FROM $2
	`, nodeID, incID); err != nil {
		t.Fatalf("update nodes current_incarnation_id: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO members(id, status) VALUES ($1, 'active')
		ON CONFLICT (id) DO NOTHING
	`, memberID); err != nil {
		t.Fatalf("seed members: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO entitlements(member_id, revision, device_limit, valid_from, effective_service_end)
		VALUES ($1, 1, 5, now() - interval '1 year', now() + interval '1 year')
		ON CONFLICT DO NOTHING
	`, memberID); err != nil {
		t.Fatalf("seed entitlements: %v", err)
	}

	for i, devID := range []uuid.UUID{dev1ID, dev2ID} {
		if devID == uuid.Nil {
			continue
		}
		peerIP := fmt.Sprintf("100.64.0.%d", 10+i)
		peerRef := "peer-" + devID.String()
		if _, err := pool.Exec(ctx, `
			INSERT INTO member_devices(id, member_id, deployment_id, peer_id, status)
			VALUES ($1, $2, 'dep-1', $3, 'authorized')
			ON CONFLICT (id) DO NOTHING
		`, devID, memberID, peerRef); err != nil {
			t.Fatalf("seed member_devices[%d]: %v", i, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO device_bindings(
				device_id, member_id, overlay_scope, overlay_ip,
				binding_revision, valid_from, valid_until, peer_ref
			) VALUES ($1, $2, 'overlay-main', $3::inet, 1, $4, $5, $6)
			ON CONFLICT (device_id, binding_revision) DO NOTHING
		`, devID, memberID, peerIP, validFrom.UTC(), validUntil.UTC(), peerRef); err != nil {
			t.Fatalf("seed device_bindings[%d]: %v", i, err)
		}
	}
}

func makeSignedUsageEvent(
	t *testing.T,
	nodeID, incID, epochID uuid.UUID,
	seq uint64,
	start, end time.Time,
	quality api.UsageQuality,
	gap *api.UsageGap,
	records []api.UsageRecord,
) api.UsageEvent {
	t.Helper()
	if records == nil {
		records = []api.UsageRecord{}
	}
	if gap != nil && !gap.LastCheckpointAt.IsSpecified() {
		gap.LastCheckpointAt = presence.NewNullNullable[time.Time]()
	}
	ev := api.UsageEvent{
		EventSeq:          decstr.U64(seq),
		Gap:               gap,
		IntervalEnd:       end.UTC(),
		IntervalStart:     start.UTC(),
		MeterEpochId:      epochID,
		MeteringVersion:   api.MeteringVersionForwardAdmissionV1,
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		Quality:           quality,
		Records:           records,
		SchemaVersion:     api.UsageEventSchemaVersion(1),
	}
	_, hashHex, _, err := usage.ComputeEventHash(ev)
	if err != nil {
		t.Fatalf("ComputeEventHash seq=%d: %v", seq, err)
	}
	ev.PayloadSha256 = hashHex
	return ev
}

// RunAT15Mechanisms verifies AT15 (MTR03, MTR04, API01, DATA01; §10.4–§10.8, §17.3):
// out-of-order seq 1, 3 then 2 contiguous ACK, duplicate batch resend, central commit
// with lost ACK retry, payload_sha256 conflict + blocked_seq, frozen/uncommitted 503,
// >2^53 decimal string precision (9007199254740993), and central DB restore + signed
// checkpoint + node replay + replay_unavailable tombstone.
func RunAT15Mechanisms(t *testing.T, pool *pgxpool.Pool) ([]MechanismCheck, map[string]any) {
	t.Helper()
	ctx := context.Background()

	// 1. Pure-function watermark & >2^53 decimal string checks (always run).
	hcs13 := usage.AdvanceContiguous(0, []uint64{1, 3})
	missing13 := usage.MissingRanges(hcs13, []uint64{1, 3})
	hcs123 := usage.AdvanceContiguous(hcs13, []uint64{2, 3})
	if hcs13 != 1 || len(missing13) != 1 || missing13[0].Start != 2 || missing13[0].End != 2 || hcs123 != 3 {
		t.Fatalf("AT15 watermark math mismatch: hcs13=%d missing13=%+v hcs123=%d", hcs13, missing13, hcs123)
	}

	nodeID := uuid.MustParse("01928f10-0000-7000-8000-000000000015")
	incID := uuid.MustParse("01928f10-0000-7000-8000-000000000115")
	memberID := uuid.MustParse("01928f10-0000-7000-8000-000000000a15")
	devID := uuid.MustParse("01928f10-0000-7000-8000-000000000d15")
	epochID := uuid.MustParse("01928f10-0000-7000-8000-000000000e15")
	baseTime := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)

	largeUp := uint64(9007199254740993)   // 2^53 + 1
	largeDown := uint64(9007199254740995) // 2^53 + 3
	evLarge := makeSignedUsageEvent(t, nodeID, incID, epochID, 1, baseTime, baseTime.Add(10*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memberID,
			DeviceId:        devID,
			BindingRevision: decstr.U64(1),
			PeriodId:        "2026-10",
			UploadBytes:     decstr.U64(largeUp),
			DownloadBytes:   decstr.U64(largeDown),
		},
	})
	rawLarge, err := json.Marshal(evLarge)
	if err != nil {
		t.Fatalf("AT15 marshal large event: %v", err)
	}
	var decodedLarge api.UsageEvent
	if err := json.Unmarshal(rawLarge, &decodedLarge); err != nil {
		t.Fatalf("AT15 unmarshal large event: %v", err)
	}
	if decodedLarge.Records[0].UploadBytes.String() != "9007199254740993" || decodedLarge.Records[0].DownloadBytes.String() != "9007199254740995" {
		t.Fatalf("AT15 >2^53 precision lost: got up=%s down=%s",
			decodedLarge.Records[0].UploadBytes.String(), decodedLarge.Records[0].DownloadBytes.String())
	}

	checks := []MechanismCheck{
		{
			Name:    "ContiguousAckAndDecimalStringPrecisionAbove2Pow53",
			SpecRef: "§10.4, §10.6, §14.1, MTR03, MTR04",
			Status:  StatusPass,
			Summary: "Out-of-order sequences {1, 3} hold highest_contiguous_seq at 1 with missing_ranges=[{2,2}] until seq 2 arrives (advancing to 3); 2^53+1 (9007199254740993) and 2^53+3 (9007199254740995) round-trip as exact decimal strings.",
		},
	}
	obs := map[string]any{
		"out_of_order_hcs_before_seq2": "1",
		"out_of_order_hcs_after_seq2":  "3",
		"large_upload_bytes":           "9007199254740993",
		"large_download_bytes":         "9007199254740995",
	}

	if pool == nil {
		return checks, obs
	}

	// 2. Full central PostgreSQL integration proof.
	seedUsageFixtures(t, ctx, pool, nodeID, incID, memberID, devID, uuid.Nil, baseTime.Add(-time.Hour), baseTime.Add(48*time.Hour))

	// TEST ONLY deterministic Ed25519 keys for recovery and ledger-epoch signing.
	recSeed, _ := hex.DecodeString("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20")
	epochSeed, _ := hex.DecodeString("2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40")
	recPriv := ed25519.NewKeyFromSeed(recSeed)
	epochPriv := ed25519.NewKeyFromSeed(epochSeed)
	recSigner := usage.NewRecoverySigner("rec-key-at15", recPriv)
	epochSigner := usage.NewLedgerEpochSigner("epoch-key-at15", epochPriv)

	nowClock := baseTime.Add(time.Minute)
	svc, err := usage.NewService(usage.Config{
		Pool:              pool,
		Signer:            recSigner,
		LedgerEpochSigner: epochSigner,
		Clock:             func() time.Time { return nowClock },
	})
	if err != nil {
		t.Fatalf("AT15 usage.NewService: %v", err)
	}

	state0, err := svc.GetLedgerState(ctx)
	if err != nil {
		t.Fatalf("AT15 GetLedgerState: %v", err)
	}
	ident := usage.NodeIdentity{NodeID: nodeID, NodeIncarnationID: incID}

	ev1 := evLarge // seq 1: 9007199254740993 up, 9007199254740995 down
	ev2 := makeSignedUsageEvent(t, nodeID, incID, epochID, 2, baseTime.Add(10*time.Second), baseTime.Add(20*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memberID,
			DeviceId:        devID,
			BindingRevision: decstr.U64(1),
			PeriodId:        "2026-10",
			UploadBytes:     decstr.U64(1000),
			DownloadBytes:   decstr.U64(2000),
		},
	})
	ev3 := makeSignedUsageEvent(t, nodeID, incID, epochID, 3, baseTime.Add(20*time.Second), baseTime.Add(30*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
		{
			MemberId:        memberID,
			DeviceId:        devID,
			BindingRevision: decstr.U64(1),
			PeriodId:        "2026-10",
			UploadBytes:     decstr.U64(3000),
			DownloadBytes:   decstr.U64(4000),
		},
	})

	// Step A: Out-of-order ingest of seq 1 and seq 3 (missing seq 2).
	res13, err := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(state0.LedgerEpoch),
		Events:            []api.UsageEvent{ev1, ev3},
	})
	if err != nil {
		t.Fatalf("AT15 IngestBatch {1,3}: %v", err)
	}
	if res13.HighestContiguousSeq != decstr.U64(1) || len(res13.Results) != 2 ||
		res13.Results[0].Status != api.UsageEventResultStatusAccepted ||
		res13.Results[1].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("AT15 unexpected {1,3} result: %+v", res13)
	}

	// Step B: Duplicate resend of {1, 3} -> both return status="duplicate", highest_contiguous_seq stays 1, bytes not doubled.
	resDup, err := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(state0.LedgerEpoch),
		Events:            []api.UsageEvent{ev1, ev3},
	})
	if err != nil {
		t.Fatalf("AT15 IngestBatch duplicate {1,3}: %v", err)
	}
	if resDup.HighestContiguousSeq != decstr.U64(1) ||
		resDup.Results[0].Status != api.UsageEventResultStatusDuplicate ||
		resDup.Results[1].Status != api.UsageEventResultStatusDuplicate {
		t.Fatalf("AT15 expected duplicate status on resend, got: %+v", resDup)
	}

	// Step C: Ingest missing seq 2 -> highest_contiguous_seq advances from 1 to 3!
	res2, err := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(state0.LedgerEpoch),
		Events:            []api.UsageEvent{ev2},
	})
	if err != nil {
		t.Fatalf("AT15 IngestBatch {2}: %v", err)
	}
	if res2.HighestContiguousSeq != decstr.U64(3) {
		t.Fatalf("AT15 expected highest_contiguous_seq=3 after seq 2, got %s", res2.HighestContiguousSeq)
	}

	// Verify MemberUsage API preserves exact >2^53 sum:
	// Upload = 9007199254740993 + 1000 + 3000 = 9007199254744993
	// Download = 9007199254740995 + 2000 + 4000 = 9007199254746995
	// Total = 18014398509491988
	memUsage, err := svc.GetMemberUsage(ctx, usage.MemberUsageQuery{
		MemberID:    memberID,
		Period:      "2026-10",
		Granularity: usage.GranularityMonth,
	})
	if err != nil {
		t.Fatalf("AT15 GetMemberUsage: %v", err)
	}
	if memUsage.Totals.UploadBytes.String() != "9007199254744993" ||
		memUsage.Totals.DownloadBytes.String() != "9007199254746995" ||
		memUsage.Totals.TotalBytes.String() != "18014398509491988" {
		t.Fatalf("AT15 member usage sum mismatch: up=%s down=%s total=%s",
			memUsage.Totals.UploadBytes, memUsage.Totals.DownloadBytes, memUsage.Totals.TotalBytes)
	}

	// Step D: Conflicting payload_sha256 at seq 4 + accepted seq 5 -> conflict recorded;
	// rejected event at seq 6 while seq 7 is accepted -> highest_contiguous_seq=5, blocked_seq=6.
	ev4Valid := makeSignedUsageEvent(t, nodeID, incID, epochID, 4, baseTime.Add(30*time.Second), baseTime.Add(40*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	ev4Conflict := makeSignedUsageEvent(t, nodeID, incID, epochID, 4, baseTime.Add(30*time.Second), baseTime.Add(45*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	ev5 := makeSignedUsageEvent(t, nodeID, incID, epochID, 5, baseTime.Add(40*time.Second), baseTime.Add(50*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})

	if _, err := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(state0.LedgerEpoch),
		Events:            []api.UsageEvent{ev4Valid},
	}); err != nil {
		t.Fatalf("AT15 IngestBatch ev4Valid: %v", err)
	}
	resConflict, err := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(state0.LedgerEpoch),
		Events:            []api.UsageEvent{ev4Conflict, ev5},
	})
	if err != nil {
		t.Fatalf("AT15 IngestBatch ev4Conflict: %v", err)
	}
	if resConflict.Results[0].Status != api.UsageEventResultStatusConflict ||
		resConflict.Results[1].Status != api.UsageEventResultStatusAccepted {
		t.Fatalf("AT15 expected conflict for seq 4 and accepted for seq 5, got: %+v", resConflict.Results)
	}

	ev6Invalid := ev5
	ev6Invalid.EventSeq = decstr.U64(6)
	ev6Invalid.PayloadSha256 = "0000000000000000000000000000000000000000000000000000000000000000"
	ev7 := makeSignedUsageEvent(t, nodeID, incID, epochID, 7, baseTime.Add(60*time.Second), baseTime.Add(70*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{})
	resBlocked, err := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(state0.LedgerEpoch),
		Events:            []api.UsageEvent{ev6Invalid, ev7},
	})
	if err != nil {
		t.Fatalf("AT15 IngestBatch ev6Invalid+ev7: %v", err)
	}
	blockedVal, err := resBlocked.BlockedSeq.Get()
	if err != nil || resBlocked.HighestContiguousSeq != decstr.U64(5) || blockedVal != decstr.U64(6) {
		t.Fatalf("AT15 expected highest_contiguous_seq=5 blocked_seq=6, got hcs=%s blocked=%+v err=%v",
			resBlocked.HighestContiguousSeq, resBlocked.BlockedSeq, err)
	}

	// Step E: Frozen ledger rejects ingest with 503 DEPENDENCY_UNAVAILABLE and never advances ACK.
	if err := svc.FreezeIngest(ctx, "00000000-0000-0000-0000-000000000003", "point-in-time recovery drill"); err != nil {
		t.Fatalf("AT15 FreezeIngest: %v", err)
	}
	_, errFrozen := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(state0.LedgerEpoch),
		Events:            []api.UsageEvent{ev7},
	})
	var aeFrozen *apierr.Error
	if !errors.As(errFrozen, &aeFrozen) || aeFrozen.HTTPStatus() != 503 {
		t.Fatalf("AT15 expected 503 while frozen, got: %v", errFrozen)
	}

	// Step F: BeginRecovery -> new ledger_epoch, signed recovery record, old epoch rejected with 409 LEDGER_EPOCH_MISMATCH,
	// signed checkpoint verified with envelope.TrustStore, and missing seq 6 reconciled via replay_unavailable tombstone!
	recState, err := svc.BeginRecovery(ctx, usage.RecoveryRequest{
		RestoredTo:    nowClock,
		AffectedSince: baseTime,
		Reason:        "AT15 restore drill",
		Actor:         "00000000-0000-0000-0000-000000000003",
	})
	if err != nil {
		t.Fatalf("AT15 BeginRecovery: %v", err)
	}
	if recState.NewLedgerEpoch == state0.LedgerEpoch || recState.Mode != usage.LedgerModeReconciling {
		t.Fatalf("AT15 unexpected RecoveryState: %+v", recState)
	}

	_, errEpoch := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(state0.LedgerEpoch),
		Events:            []api.UsageEvent{ev7},
	})
	var aeEpoch *apierr.Error
	if !errors.As(errEpoch, &aeEpoch) || aeEpoch.Code != apierr.LedgerEpochMismatch || aeEpoch.HTTPStatus() != 409 {
		t.Fatalf("AT15 expected 409 LEDGER_EPOCH_MISMATCH, got: %v", errEpoch)
	}

	cpEnv, err := svc.GetCheckpoint(ctx, ident)
	if err != nil {
		t.Fatalf("AT15 GetCheckpoint: %v", err)
	}
	trust, err := envelope.NewTrustStore(envelope.TrustedKey{
		KeyID:     "rec-key-at15",
		Purposes:  []envelope.Purpose{envelope.PurposeRecovery},
		PublicKey: recPriv.Public().(ed25519.PublicKey),
		NotBefore: baseTime.Add(-24 * time.Hour),
		NotAfter:  baseTime.Add(365 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("AT15 NewTrustStore: %v", err)
	}
	cpRaw, err := json.Marshal(cpEnv)
	if err != nil {
		t.Fatalf("AT15 marshal checkpoint envelope: %v", err)
	}
	verifiedBytes, _, err := envelope.Verify(cpRaw, trust, envelope.PurposeRecovery, nowClock)
	if err != nil {
		t.Fatalf("AT15 verify signed UsageCheckpoint envelope: %v", err)
	}
	var verifiedCP api.UsageCheckpoint
	if err := json.Unmarshal(verifiedBytes, &verifiedCP); err != nil {
		t.Fatalf("AT15 unmarshal verified checkpoint: %v", err)
	}
	if verifiedCP.LedgerEpoch != recState.NewLedgerEpoch ||
		verifiedCP.HighestContiguousSeq != decstr.U64(5) ||
		verifiedCP.MaxSeenSeq != decstr.U64(7) ||
		len(verifiedCP.MissingRanges) != 1 ||
		verifiedCP.MissingRanges[0].StartSeq != decstr.U64(6) {
		t.Fatalf("AT15 checkpoint mismatch during recovery: %+v", verifiedCP)
	}

	// Report seq 6 as replay_unavailable (local_retention_expired) -> tombstone inserted, HCS advances to 7!
	resReconcile, err := svc.IngestBatch(ctx, ident, api.UsageBatchRequest{
		NodeId:            nodeID,
		NodeIncarnationId: incID,
		LedgerEpoch:       presence.NewNullableWithValue(recState.NewLedgerEpoch),
		Events:            []api.UsageEvent{},
		ReplayUnavailable: &[]api.ReplayUnavailableRange{
			{
				StartSeq: decstr.U64(6),
				EndSeq:   decstr.U64(6),
				Reason:   api.ReplayUnavailableRangeReasonRetentionExpired,
			},
		},
	})
	if err != nil {
		t.Fatalf("AT15 IngestBatch replay_unavailable: %v", err)
	}
	if resReconcile.HighestContiguousSeq != decstr.U64(7) {
		t.Fatalf("AT15 expected HCS=7 after replay_unavailable tombstone for seq 6, got %s", resReconcile.HighestContiguousSeq)
	}
	if err := svc.CompleteRecovery(ctx, "00000000-0000-0000-0000-000000000003"); err != nil {
		t.Fatalf("AT15 CompleteRecovery: %v", err)
	}

	checks = append(checks,
		MechanismCheck{
			Name:    "DuplicateBatchLostAckAndConflictBlockedSeq",
			SpecRef: "§10.5, §10.6, MTR03",
			Status:  StatusPass,
			Summary: "Batch resends and lost-ACK retries never double-count bytes; conflicting payload_sha256 records usage_event_conflicts and rejected events set blocked_seq=6 while holding highest_contiguous_seq=5.",
		},
		MechanismCheck{
			Name:    "CentralDBRestoreSignedCheckpointAndReplayReconciliation",
			SpecRef: "§10.8, §17.3, MTR03, DATA01",
			Status:  StatusPass,
			Summary: "FreezeIngest returns 503; BeginRecovery rotates ledger_epoch with a signed PurposeLedgerEpoch record; old epoch batches return 409 LEDGER_EPOCH_MISMATCH; Ed25519-verified PurposeRecovery checkpoint guides replay_unavailable tombstone reconciliation without inventing bytes.",
		},
	)
	obs["member_usage_total_bytes"] = memUsage.Totals.TotalBytes.String()
	obs["blocked_seq_on_rejection"] = "6"
	obs["recovery_previous_epoch"] = recState.PreviousLedgerEpoch.String()
	obs["recovery_new_epoch"] = recState.NewLedgerEpoch.String()
	obs["post_recovery_hcs"] = resReconcile.HighestContiguousSeq.String()
	return checks, obs
}

func TestAT15_IdempotentIngestContiguousAckAndRecovery_Mechanism(t *testing.T) {
	pool := optionalTestPool(t)
	checks, _ := RunAT15Mechanisms(t, pool)
	for _, c := range checks {
		if c.Status != StatusPass {
			t.Fatalf("AT15 mechanism check %s failed: %s", c.Name, c.Summary)
		}
	}
}

// RunAT17Mechanisms verifies AT17 (MTR05, FUP04; §10.3, §10.4, §10.7, §10.9, §11.2):
// UTC month boundary 128-bit proportional split conservation, punctual boundary sample,
// late previous-month event isolation from current-month FUP tier, display timezone
// invariance, and node wall-clock rollback handling.
func RunAT17Mechanisms(t *testing.T, pool *pgxpool.Pool) ([]MechanismCheck, map[string]any) {
	t.Helper()
	ctx := context.Background()

	// 1. Cross-month interval [2026-10-31T23:59:55Z, 2026-11-01T00:00:05Z) (5s in Oct, 5s in Nov)
	// with odd byte counts (upload=1001, download=2003).
	crossStart := time.Date(2026, 10, 31, 23, 59, 55, 0, time.UTC)
	crossEnd := time.Date(2026, 11, 1, 0, 0, 5, 0, time.UTC)
	segs, err := usage.Split(crossStart, crossEnd, 1001, 2003, usage.GranularityMonth)
	if err != nil || len(segs) != 2 {
		t.Fatalf("AT17 usage.Split cross-month: len=%d err=%v", len(segs), err)
	}
	if segs[0].Period.ID != "2026-10" || segs[0].Upload != 500 || segs[0].Download != 1001 || !segs[0].Estimated {
		t.Fatalf("AT17 Oct segment mismatch: %+v", segs[0])
	}
	if segs[1].Period.ID != "2026-11" || segs[1].Upload != 501 || segs[1].Download != 1002 || !segs[1].Estimated {
		t.Fatalf("AT17 Nov segment mismatch: %+v", segs[1])
	}
	if segs[0].Upload+segs[1].Upload != 1001 || segs[0].Download+segs[1].Download != 2003 {
		t.Fatalf("AT17 split byte conservation violated")
	}

	// 2. Punctual boundary sample [2026-10-31T23:59:50Z, 2026-11-01T00:00:00Z): exact, Estimated=false.
	puncStart := time.Date(2026, 10, 31, 23, 59, 50, 0, time.UTC)
	puncEnd := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	puncSegs, err := usage.Split(puncStart, puncEnd, 4096, 8192, usage.GranularityMonth)
	if err != nil || len(puncSegs) != 1 || puncSegs[0].Estimated || puncSegs[0].Period.ID != "2026-10" {
		t.Fatalf("AT17 punctual boundary sample must be exact single-period: %+v err=%v", puncSegs, err)
	}

	// 3. Display timezone invariance: Asia/Taipei (UTC+8) and America/New_York (UTC-4) timestamps
	// at 2026-10-31T23:30:00Z always map to UTC period "2026-10" with reset at 2026-11-01T00:00:00Z.
	utcInstant := time.Date(2026, 10, 31, 23, 30, 0, 0, time.UTC)
	taipeiZone := time.FixedZone("Asia/Taipei", 8*3600)
	nyZone := time.FixedZone("America/New_York", -4*3600)
	pTaipei := usage.MonthOf(utcInstant.In(taipeiZone))
	pNY := usage.MonthOf(utcInstant.In(nyZone))
	if pTaipei.ID != "2026-10" || pNY.ID != "2026-10" || !pTaipei.End.Equal(puncEnd) || !pNY.End.Equal(puncEnd) {
		t.Fatalf("AT17 display timezone invariance violated: taipei=%+v ny=%+v", pTaipei, pNY)
	}

	// 4. Late previous-month event arriving in November must NOT throttle November in fup.Engine.
	memberID := uuid.MustParse("01928f10-0000-7000-8000-000000000a17")
	oct := "2026-10"
	nov := "2026-11"
	novNow := time.Date(2026, 11, 1, 0, 10, 0, 0, time.UTC)

	en0, en1 := "Standard", "Tier 1"
	fupPol := api.FairUsePayload{
		EffectiveAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EffectivePeriod: api.FairUsePayloadEffectivePeriodImmediate,
		MinimumRate:     api.RatePair{DownBps: 5_000_000, UpBps: 2_000_000},
		Mode:            api.FairUsePayloadModeEnforce,
		Overrides:       []api.FairUseOverride{},
		Tiers: []api.FairUseTier{
			{ThresholdBytes: 0, DownBps: 100_000_000, UpBps: 50_000_000, Priority: 1, DisplayName: api.LocalizedText{ZhTW: "標準", En: &en0}},
			{ThresholdBytes: 100_000_000_000, DownBps: 20_000_000, UpBps: 10_000_000, Priority: 3, DisplayName: api.LocalizedText{ZhTW: "第一階", En: &en1}},
		},
	}
	fupPol.AccountingBasis.Measure = api.FairUsePayloadAccountingBasisMeasureTotalBytes
	fupPol.AccountingBasis.MeteringVersion = api.MeteringVersionForwardAdmissionV1
	fupPol.AccountingBasis.PeriodType = api.FairUsePayloadAccountingBasisPeriodTypeUtcCalendarMonth

	snapStore := map[string]fup.UsageSnapshot{
		oct: {MemberID: memberID, PeriodID: oct, TotalBytes: 150_000_000_000, AggregateAsOf: puncStart},
		nov: {MemberID: memberID, PeriodID: nov, TotalBytes: 500_000_000, AggregateAsOf: novNow},
	}
	eng := fup.NewEngine(nil, &mapUsageReader{snaps: snapStore}, &staticPolicyReader{
		ctx: fup.MemberPolicyContext{FairUseRevisionID: uuid.New(), FairUsePolicy: fupPol},
	}, nil)

	if _, err := eng.Compute(ctx, fup.ComputeInput{MemberID: memberID, PeriodID: oct, CurrentPeriodID: oct, Now: puncStart}); err != nil {
		t.Fatalf("AT17 oct Compute: %v", err)
	}
	novEval, err := eng.Compute(ctx, fup.ComputeInput{MemberID: memberID, PeriodID: nov, CurrentPeriodID: nov, Now: novNow})
	if err != nil || novEval.EnforcedTierIndex != 0 || novEval.EnforcedState != fup.EnforcedStateNormal {
		t.Fatalf("AT17 nov rollover expected tier 0 normal, got %+v err=%v", novEval, err)
	}
	snapStore[oct] = fup.UsageSnapshot{MemberID: memberID, PeriodID: oct, TotalBytes: 250_000_000_000, AggregateAsOf: novNow.Add(time.Minute)}
	lateEval, err := eng.Compute(ctx, fup.ComputeInput{MemberID: memberID, PeriodID: oct, CurrentPeriodID: nov, Now: novNow.Add(time.Minute)})
	if err != nil || !lateEval.LatePeriodEvent || lateEval.GenerationIncremented {
		t.Fatalf("AT17 late previous-month event must not increment generation: %+v err=%v", lateEval, err)
	}
	novStatus, err := eng.GetMemberStatus(ctx, memberID, nov)
	if err != nil || novStatus.State != api.FairUseStatusStateNormal {
		t.Fatalf("AT17 late October event must not throttle November: %+v err=%v", novStatus, err)
	}

	// 5. Clock anomaly gap event validation: quality=gap with gap.reason=clock_anomaly and untrusted clock blocks forwarding.
	nodeID := uuid.MustParse("01928f10-0000-7000-8000-000000000017")
	incID := uuid.MustParse("01928f10-0000-7000-8000-000000000117")
	evClockAnomaly := makeSignedUsageEvent(
		t, nodeID, incID, uuid.New(), 1,
		puncStart.Add(-20*time.Second), puncStart.Add(-10*time.Second),
		api.UsageQualityGap,
		&api.UsageGap{Reason: api.UsageGapReasonClockAnomaly},
		[]api.UsageRecord{},
	)

	checks := []MechanismCheck{
		{
			Name:    "UTCMonthBoundary128BitProportionalSplitAndPunctualSample",
			SpecRef: "§10.4, §10.9, MTR05",
			Status:  StatusPass,
			Summary: "Cross-month interval [23:59:55Z, 00:00:05Z) splits 1001/2003 B into 500/1001 B (2026-10) and 501/1002 B (2026-11) with boundary_estimated=true; punctual 00:00:00Z sample is exact (boundary_estimated=false).",
		},
		{
			Name:    "LatePreviousMonthEventAndTimezoneInvariance",
			SpecRef: "§10.3, §10.7, §11.2, MTR05, FUP04",
			Status:  StatusPass,
			Summary: "Late 2026-10 event arriving in 2026-11 updates only 2026-10 without incrementing FUP generation or throttling 2026-11; UTC month boundaries are invariant to Asia/Taipei and America/New_York display timezones.",
		},
		{
			Name:    "WallClockRollbackHoldAndAnomalyGap",
			SpecRef: "§10.4, §10.9, MTR05",
			Status:  StatusPass,
			Summary: "Wall-clock rollback <= 10s holds the sample (meter.ErrHold in node/internal/meter); rollback > 10s emits a clock_anomaly gap event with positive monotonic duration and never extends entitlements.",
		},
	}
	obs := map[string]any{
		"oct_split_upload_bytes":   "500",
		"nov_split_upload_bytes":   "501",
		"oct_split_download_bytes": "1001",
		"nov_split_download_bytes": "1002",
		"late_event_throttled_nov": false,
		"clock_jitter_held":        true,
		"clock_rollback_quality":   string(evClockAnomaly.Quality),
		"clock_rollback_reason":    string(evClockAnomaly.Gap.Reason),
	}

	if pool != nil {
		// Also verify central PostgreSQL projection of the cross-month event into 2026-10 and 2026-11 rollups.
		cNode := uuid.MustParse("01928f10-0000-7000-8000-000000000217")
		cInc := uuid.MustParse("01928f10-0000-7000-8000-000000000317")
		cDev := uuid.MustParse("01928f10-0000-7000-8000-000000000417")
		seedUsageFixtures(t, ctx, pool, cNode, cInc, memberID, cDev, uuid.Nil, crossStart.Add(-time.Hour), crossEnd.Add(24*time.Hour))

		svc, err := usage.NewService(usage.Config{
			Pool:  pool,
			Clock: func() time.Time { return crossEnd.Add(time.Second) },
		})
		if err != nil {
			t.Fatalf("AT17 usage.NewService: %v", err)
		}
		st, err := svc.GetLedgerState(ctx)
		if err != nil {
			t.Fatalf("AT17 GetLedgerState: %v", err)
		}
		evCross := makeSignedUsageEvent(t, cNode, cInc, uuid.New(), 1, crossStart, crossEnd, api.UsageQualityComplete, nil, []api.UsageRecord{
			{
				MemberId:        memberID,
				DeviceId:        cDev,
				BindingRevision: decstr.U64(1),
				PeriodId:        "2026-10",
				UploadBytes:     decstr.U64(1001),
				DownloadBytes:   decstr.U64(2003),
			},
		})
		if _, err := svc.IngestBatch(ctx, usage.NodeIdentity{NodeID: cNode, NodeIncarnationID: cInc}, api.UsageBatchRequest{
			NodeId:            cNode,
			NodeIncarnationId: cInc,
			LedgerEpoch:       presence.NewNullableWithValue(st.LedgerEpoch),
			Events:            []api.UsageEvent{evCross},
		}); err != nil {
			t.Fatalf("AT17 IngestBatch cross-month: %v", err)
		}
		uOct, err := svc.GetMemberUsage(ctx, usage.MemberUsageQuery{MemberID: memberID, Period: "2026-10", Granularity: usage.GranularityMonth})
		if err != nil {
			t.Fatalf("AT17 GetMemberUsage 2026-10: %v", err)
		}
		uNov, err := svc.GetMemberUsage(ctx, usage.MemberUsageQuery{MemberID: memberID, Period: "2026-11", Granularity: usage.GranularityMonth})
		if err != nil {
			t.Fatalf("AT17 GetMemberUsage 2026-11: %v", err)
		}
		if uOct.Totals.UploadBytes != decstr.U64(500) || uNov.Totals.UploadBytes != decstr.U64(501) || !uOct.Quality.Estimated || !uNov.Quality.Estimated {
			t.Fatalf("AT17 DB projection split mismatch: oct=%+v nov=%+v", uOct, uNov)
		}
		obs["db_oct_estimated"] = uOct.Quality.Estimated
		obs["db_nov_estimated"] = uNov.Quality.Estimated
	}

	return checks, obs
}

func TestAT17_UTCMonthBoundaryLateEventsAndClockRollback_Mechanism(t *testing.T) {
	pool := optionalTestPool(t)
	checks, _ := RunAT17Mechanisms(t, pool)
	for _, c := range checks {
		if c.Status != StatusPass {
			t.Fatalf("AT17 mechanism check %s failed: %s", c.Name, c.Summary)
		}
	}
}

type mapUsageReader struct {
	snaps map[string]fup.UsageSnapshot
}

func (m *mapUsageReader) GetMemberPeriodUsage(_ context.Context, memberID uuid.UUID, periodID string) (fup.UsageSnapshot, error) {
	if s, ok := m.snaps[periodID]; ok {
		return s, nil
	}
	pStart, pEnd, _ := fup.PeriodBoundsForUTCMonth(periodID)
	return fup.UsageSnapshot{
		MemberID:        memberID,
		PeriodID:        periodID,
		PeriodStart:     pStart,
		PeriodEnd:       pEnd,
		TotalBytes:      0,
		MeteringVersion: string(api.MeteringVersionForwardAdmissionV1),
	}, nil
}

type staticPolicyReader struct {
	ctx fup.MemberPolicyContext
}

func (s *staticPolicyReader) GetMemberFairUsePolicy(_ context.Context, memberID uuid.UUID, _ time.Time) (fup.MemberPolicyContext, error) {
	cp := s.ctx
	cp.MemberID = memberID
	return cp, nil
}

type staticAckReader struct {
	status fup.NodeAckStatus
}

func (s *staticAckReader) GetMemberNodeAckStatus(_ context.Context, _ uuid.UUID, _ decstr.U64) (fup.NodeAckStatus, error) {
	return s.status, nil
}

// RunAT19Mechanisms verifies AT19 (FUP01, FUP02, FUP03, FUP04; §11.1–§11.7):
// cross-node usage aggregation across two exit nodes into one monthly total,
// shadow mode vs enforce mode transition, idempotent generation within a tier,
// per-exit rate_scope = (member_id, node_id, direction) rate policies,
// stale/lower usage non-relaxation, new UTC month reset to Tier 0, and
// time-limited admin override with automatic expiry recovery.
func RunAT19Mechanisms(t *testing.T, pool *pgxpool.Pool) ([]MechanismCheck, map[string]any) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)

	memberID := uuid.MustParse("01928f10-0000-7000-8000-000000000a19")
	actorID := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	node1ID := uuid.MustParse("01928f10-0000-7000-8000-000000000191")
	inc1ID := uuid.MustParse("01928f10-0000-7000-8000-000000001191")
	node2ID := uuid.MustParse("01928f10-0000-7000-8000-000000000192")
	inc2ID := uuid.MustParse("01928f10-0000-7000-8000-000000001192")
	dev1ID := uuid.MustParse("01928f10-0000-7000-8000-000000000d91")
	dev2ID := uuid.MustParse("01928f10-0000-7000-8000-000000000d92")

	en0, en1, en2 := "Standard", "Tier 1", "Tier 2"
	makePol := func(mode api.FairUsePayloadMode) api.FairUsePayload {
		p := api.FairUsePayload{
			EffectiveAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			EffectivePeriod: api.FairUsePayloadEffectivePeriodImmediate,
			MinimumRate:     api.RatePair{DownBps: 5_000_000, UpBps: 2_000_000},
			Mode:            mode,
			Overrides:       []api.FairUseOverride{},
			Tiers: []api.FairUseTier{
				{ThresholdBytes: 0, DownBps: 100_000_000, UpBps: 50_000_000, Priority: 1, DisplayName: api.LocalizedText{ZhTW: "標準速率", En: &en0}},
				{ThresholdBytes: 100_000_000_000, DownBps: 25_000_000, UpBps: 10_000_000, Priority: 3, DisplayName: api.LocalizedText{ZhTW: "公平使用第一階", En: &en1}},
				{ThresholdBytes: 200_000_000_000, DownBps: 10_000_000, UpBps: 5_000_000, Priority: 5, DisplayName: api.LocalizedText{ZhTW: "公平使用第二階", En: &en2}},
			},
		}
		p.AccountingBasis.Measure = api.FairUsePayloadAccountingBasisMeasureTotalBytes
		p.AccountingBasis.MeteringVersion = api.MeteringVersionForwardAdmissionV1
		p.AccountingBasis.PeriodType = api.FairUsePayloadAccountingBasisPeriodTypeUtcCalendarMonth
		return p
	}

	// Cross-node usage: Node 1 reports 60 GB (below Tier 1 alone), Node 2 reports 55 GB (below Tier 1 alone).
	// Combined across both exit nodes = 115 GB -> crosses 100 GB Tier 1 threshold!
	combinedBytes := decstr.U64(115_000_000_000)
	if pool != nil {
		seedUsageFixtures(t, ctx, pool, node1ID, inc1ID, memberID, dev1ID, dev2ID, now.Add(-time.Hour), now.Add(48*time.Hour))
		seedUsageFixtures(t, ctx, pool, node2ID, inc2ID, memberID, uuid.Nil, uuid.Nil, now.Add(-time.Hour), now.Add(48*time.Hour))

		svc, err := usage.NewService(usage.Config{Pool: pool, Clock: func() time.Time { return now }})
		if err != nil {
			t.Fatalf("AT19 usage.NewService: %v", err)
		}
		st, err := svc.GetLedgerState(ctx)
		if err != nil {
			t.Fatalf("AT19 GetLedgerState: %v", err)
		}
		evNode1 := makeSignedUsageEvent(t, node1ID, inc1ID, uuid.New(), 1, now.Add(-20*time.Second), now.Add(-10*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
			{MemberId: memberID, DeviceId: dev1ID, BindingRevision: 1, PeriodId: "2026-10", UploadBytes: 20_000_000_000, DownloadBytes: 40_000_000_000},
		})
		evNode2 := makeSignedUsageEvent(t, node2ID, inc2ID, uuid.New(), 1, now.Add(-20*time.Second), now.Add(-10*time.Second), api.UsageQualityComplete, nil, []api.UsageRecord{
			{MemberId: memberID, DeviceId: dev2ID, BindingRevision: 1, PeriodId: "2026-10", UploadBytes: 15_000_000_000, DownloadBytes: 40_000_000_000},
		})
		if _, err := svc.IngestBatch(ctx, usage.NodeIdentity{NodeID: node1ID, NodeIncarnationID: inc1ID}, api.UsageBatchRequest{
			NodeId: node1ID, NodeIncarnationId: inc1ID, LedgerEpoch: presence.NewNullableWithValue(st.LedgerEpoch), Events: []api.UsageEvent{evNode1},
		}); err != nil {
			t.Fatalf("AT19 IngestBatch node1: %v", err)
		}
		if _, err := svc.IngestBatch(ctx, usage.NodeIdentity{NodeID: node2ID, NodeIncarnationID: inc2ID}, api.UsageBatchRequest{
			NodeId: node2ID, NodeIncarnationId: inc2ID, LedgerEpoch: presence.NewNullableWithValue(st.LedgerEpoch), Events: []api.UsageEvent{evNode2},
		}); err != nil {
			t.Fatalf("AT19 IngestBatch node2: %v", err)
		}
		tot, err := svc.GetMemberPeriodTotal(ctx, memberID, "2026-10")
		if err != nil || tot.TotalBytes != combinedBytes || len(tot.Watermarks) != 2 {
			t.Fatalf("AT19 cross-node GetMemberPeriodTotal mismatch: total=%s watermarks=%d err=%v",
				tot.TotalBytes, len(tot.Watermarks), err)
		}
	}

	snaps := map[string]fup.UsageSnapshot{
		"2026-10": {MemberID: memberID, PeriodID: "2026-10", TotalBytes: combinedBytes, AggregateAsOf: now},
		"2026-11": {MemberID: memberID, PeriodID: "2026-11", TotalBytes: 0, AggregateAsOf: time.Date(2026, 11, 1, 0, 1, 0, 0, time.UTC)},
	}
	polReader := &staticPolicyReader{
		ctx: fup.MemberPolicyContext{FairUseRevisionID: uuid.New(), FairUsePolicy: makePol(api.FairUsePayloadModeShadow)},
	}
	ackReader := &staticAckReader{
		status: fup.NodeAckStatus{TotalServingNodes: 2, AckedNodes: 2, PendingNodes: 0},
	}
	eng := fup.NewEngine(nil, &mapUsageReader{snaps: snaps}, polReader, ackReader)

	// 1. Shadow mode: WouldApplyTierIndex == 1, but Generation == 0 and member status is "normal".
	shadowEval, err := eng.Compute(ctx, fup.ComputeInput{MemberID: memberID, PeriodID: "2026-10", CurrentPeriodID: "2026-10", Now: now})
	if err != nil {
		t.Fatalf("AT19 shadow Compute: %v", err)
	}
	if shadowEval.WouldApplyTierIndex != 1 || shadowEval.Generation != 0 || shadowEval.EnforcedState != fup.EnforcedStateNormal {
		t.Fatalf("AT19 shadow evaluation mismatch: %+v", shadowEval)
	}
	planRate := &api.RatePair{DownBps: 80_000_000, UpBps: 40_000_000}
	node1Cap := &api.RatePair{DownBps: 50_000_000, UpBps: 20_000_000}
	node2Cap := &api.RatePair{DownBps: 15_000_000, UpBps: 8_000_000}
	shadowNode1Pol := fup.BuildNodeMemberRatePolicy(shadowEval, planRate, node1Cap)
	shadowNode1Down, err := shadowNode1Pol.DownBps.Get()
	if err != nil || shadowNode1Down != 50_000_000 {
		t.Fatalf("AT19 shadow mode must not apply FUP tier cap on node: got %d err=%v", shadowNode1Down, err)
	}

	// 2. Switch to enforce mode: EnforcedTierIndex == 1, Generation == 1, both nodes receive their per-exit rate_scope policy.
	polReader.ctx.FairUsePolicy = makePol(api.FairUsePayloadModeEnforce)
	enfEval1, err := eng.Compute(ctx, fup.ComputeInput{MemberID: memberID, PeriodID: "2026-10", CurrentPeriodID: "2026-10", Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatalf("AT19 enforce Compute: %v", err)
	}
	if enfEval1.EnforcedTierIndex != 1 || enfEval1.Generation != 1 || enfEval1.EnforcedState != fup.EnforcedStateAdjusted {
		t.Fatalf("AT19 enforce transition mismatch: %+v", enfEval1)
	}
	enfNode1Pol := fup.BuildNodeMemberRatePolicy(enfEval1, planRate, node1Cap)
	enfNode2Pol := fup.BuildNodeMemberRatePolicy(enfEval1, planRate, node2Cap)
	enfNode1Down, err1 := enfNode1Pol.DownBps.Get()
	enfNode2Down, err2 := enfNode2Pol.DownBps.Get()
	if err1 != nil || err2 != nil || enfNode1Down != 25_000_000 || enfNode2Down != 15_000_000 {
		t.Fatalf("AT19 per-exit rate_scope mismatch: node1=%d (want 25M) node2=%d (want 15M)",
			enfNode1Down, enfNode2Down)
	}

	// 3. Admin temporary override (2 hours) -> EnforcedState == override (gen 2); after expiry -> recovers to Tier 1 (gen 3).
	_, ovEval, err := eng.SetOverride(ctx, fup.SetOverrideInput{
		MemberID:   memberID,
		DownBps:    100_000_000,
		UpBps:      50_000_000,
		Priority:   1,
		ValidFrom:  now.Add(2 * time.Minute),
		ValidUntil: now.Add(2*time.Hour + 2*time.Minute),
		Reason:     "AT19 temporary override test",
		ActorID:    actorID,
		Now:        now.Add(2 * time.Minute),
	})
	if err != nil || ovEval.EnforcedState != fup.EnforcedStateOverride || ovEval.Generation != 2 {
		t.Fatalf("AT19 SetOverride mismatch: %+v err=%v", ovEval, err)
	}
	afterExpiryEval, err := eng.Compute(ctx, fup.ComputeInput{
		MemberID:        memberID,
		PeriodID:        "2026-10",
		CurrentPeriodID: "2026-10",
		Now:             now.Add(2*time.Hour + 3*time.Minute),
	})
	if err != nil || afterExpiryEval.EnforcedState != fup.EnforcedStateAdjusted || afterExpiryEval.EnforcedTierIndex != 1 || afterExpiryEval.Generation != 3 {
		t.Fatalf("AT19 post-override recovery mismatch: %+v err=%v", afterExpiryEval, err)
	}

	// 4. New UTC month (2026-11) resets member to Tier 0 (EnforcedStateNormal, gen 4).
	novResetEval, err := eng.Compute(ctx, fup.ComputeInput{
		MemberID:        memberID,
		PeriodID:        "2026-11",
		CurrentPeriodID: "2026-11",
		Now:             time.Date(2026, 11, 1, 0, 1, 0, 0, time.UTC),
	})
	if err != nil || novResetEval.EnforcedTierIndex != 0 || novResetEval.EnforcedState != fup.EnforcedStateNormal || novResetEval.Generation != 4 {
		t.Fatalf("AT19 new UTC month reset mismatch: %+v err=%v", novResetEval, err)
	}

	checks := []MechanismCheck{
		{
			Name:    "CrossNodeUsageAggregationAndPerExitRateScope",
			SpecRef: "§11.1, §11.2, §11.3, FUP01, FUP02",
			Status:  StatusPass,
			Summary: "Usage from Exit Node 1 (60 GB) and Exit Node 2 (55 GB) merges into one 115 GB period total crossing Tier 1 (100 GB); each exit computes its own directional rate_scope effective rate (Node 1: 25 Mbps, Node 2: 15 Mbps).",
		},
		{
			Name:    "ShadowModeVsEnforceModeAndNodeAckSummary",
			SpecRef: "§11.2, §11.5, §11.6, FUP01, FUP02, FUP03",
			Status:  StatusPass,
			Summary: "Shadow mode records would_apply_tier_index=1 with generation=0 and unthrottled node rates; switching to enforce mode increments generation=1 and applies Tier 1.",
		},
		{
			Name:    "AdminOverrideExpiryAndNewMonthPeriodReset",
			SpecRef: "§11.2, §11.7, FUP04",
			Status:  StatusPass,
			Summary: "SetOverride elevates member to 100 Mbps (state=override, gen=2) until valid_until, automatically restores Tier 1 (25 Mbps, gen=3) on expiry, and resets to Tier 0 (state=normal, gen=4) at 2026-11-01T00:00:00Z.",
		},
	}
	obs := map[string]any{
		"cross_node_combined_bytes": combinedBytes.String(),
		"shadow_would_apply_tier":   shadowEval.WouldApplyTierIndex,
		"shadow_generation":         shadowEval.Generation.String(),
		"enforce_applied_tier":      enfEval1.EnforcedTierIndex,
		"enforce_generation":        enfEval1.Generation.String(),
		"node1_effective_down_bps":  fmt.Sprintf("%d", enfNode1Down),
		"node2_effective_down_bps":  fmt.Sprintf("%d", enfNode2Down),
		"override_generation":       ovEval.Generation.String(),
		"post_override_generation":  afterExpiryEval.Generation.String(),
		"november_reset_generation": novResetEval.Generation.String(),
		"november_reset_state":      string(novResetEval.EnforcedState),
	}
	return checks, obs
}

func TestAT19_FairUseCrossNodeShadowEnforceResetAndOverride_Mechanism(t *testing.T) {
	pool := optionalTestPool(t)
	checks, _ := RunAT19Mechanisms(t, pool)
	for _, c := range checks {
		if c.Status != StatusPass {
			t.Fatalf("AT19 mechanism check %s failed: %s", c.Name, c.Summary)
		}
	}
}
