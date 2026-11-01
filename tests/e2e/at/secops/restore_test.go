//go:build e2e && integration

package secops

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Executed only inside the network-none restoration container. The immutable
// backup and retained batch originate from the running lab, not a seeded copy.
// §17.3 / AT30: physical mode also verifies the WAL recovery boundary before writes.
func TestAT30_IsolatedRestoreReplayer(t *testing.T) {
	if os.Getenv("FVPN_SECOPS_RESTORE") != "network-none" {
		t.Skip("requires isolated restore container")
	}
	ctx := t.Context()
	start := time.Now()
	password, err := os.ReadFile("/fixture/password")
	if err != nil {
		t.Fatal("restore password")
	}
	cfg, err := pgxpool.ParseConfig("host=127.0.0.1 port=5432 user=fvpn dbname=fvpn sslmode=disable")
	if err != nil {
		t.Fatal("restore connection settings")
	}
	cfg.ConnConfig.Password = strings.TrimSpace(string(password))
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal("restore pool")
	}
	defer pool.Close()
	raw, err := os.ReadFile("/fixture/retained.json")
	if err != nil {
		t.Fatal("retained archive")
	}
	var batch api.UsageBatchRequest
	if json.Unmarshal(raw, &batch) != nil {
		t.Fatal("retained archive decode")
	}
	prov, err := signing.OpenFileProvider("/fixture/vault")
	if err != nil {
		t.Fatal("restore signing provider")
	}
	signer := usage.NewProviderSigner(prov)
	svc, err := usage.NewService(usage.Config{Pool: pool, Signer: signer, LedgerEpochSigner: signer})
	if err != nil {
		t.Fatal(err)
	}
	identity := usage.NodeIdentity{NodeID: batch.NodeId, NodeIncarnationID: batch.NodeIncarnationId}
	before, err := svc.GetCheckpoint(ctx, identity)
	if err != nil {
		t.Fatal("restore checkpoint")
	}
	expectedACK := decstr.U64(5)
	restoredTo := before.Signed.IssuedAt
	physical := map[string]any{}
	if raw, readErr := os.ReadFile("/fixture/pitr-manifest.json"); readErr == nil {
		var manifest pitrManifest
		if json.Unmarshal(raw, &manifest) != nil {
			t.Fatal("PITR manifest decode")
		}
		expectedACK = manifest.TargetACK
		restoredTo = manifest.Target
		physical = verifyPhysicalRestore(t, pool, manifest)
	} else if !os.IsNotExist(readErr) {
		t.Fatal("PITR manifest read")
	}
	if before.Signed.HighestContiguousSeq != expectedACK {
		t.Fatal("restored checkpoint differs from target")
	}
	actor := "00000000-0000-0000-0000-000000000003"
	var entitlementBefore string
	err = pool.QueryRow(ctx, `SELECT string_agg(member_id::text||effective_service_end::text,',' ORDER BY member_id) FROM entitlements`).Scan(&entitlementBefore)
	if err != nil {
		t.Fatal("entitlement baseline")
	}
	if svc.FreezeIngest(ctx, actor, "TEST ONLY isolated backup restore") != nil {
		t.Fatal("freeze")
	}
	if _, err = svc.IngestBatch(ctx, identity, batch); err == nil {
		t.Fatal("frozen ingestion accepted")
	}
	state, err := svc.BeginRecovery(ctx, usage.RecoveryRequest{Actor: actor, Reason: "TEST ONLY retained window replay", RestoredTo: restoredTo, AffectedSince: batch.Events[0].IntervalStart})
	if err != nil {
		t.Fatal("begin recovery")
	}
	batch.LedgerEpoch = presence.NewNullableWithValue(state.NewLedgerEpoch)
	first, err := svc.IngestBatch(ctx, identity, batch)
	if err != nil {
		t.Fatal("replay failed")
	}
	again, err := svc.IngestBatch(ctx, identity, batch)
	if err != nil {
		t.Fatal("duplicate replay failed")
	}
	accepted, duplicate, repeat := 0, 0, 0
	for _, r := range first.Results {
		switch r.Status {
		case api.UsageEventResultStatusAccepted:
			accepted++
		case api.UsageEventResultStatusDuplicate:
			duplicate++
		}
	}
	for _, r := range again.Results {
		if r.Status == api.UsageEventResultStatusDuplicate {
			repeat++
		}
	}
	if accepted != 10-int(expectedACK) || duplicate != int(expectedACK) || repeat != 10 || again.HighestContiguousSeq != 10 {
		t.Fatalf("replay counts accepted=%d duplicate=%d repeat=%d ack=%s", accepted, duplicate, repeat, again.HighestContiguousSeq)
	}
	if svc.CompleteRecovery(ctx, actor) != nil {
		t.Fatal("complete recovery")
	}
	// The ten retained one-second events can occupy two UTC months.
	wantByPeriod := map[api.PeriodId]decstr.U64{}
	for _, event := range batch.Events {
		for _, record := range event.Records {
			wantByPeriod[record.PeriodId] += record.UploadBytes + record.DownloadBytes
		}
	}
	var totalBytes decstr.U64
	for period, want := range wantByPeriod {
		total, err := svc.GetMemberPeriodTotal(ctx, batch.Events[0].Records[0].MemberId, string(period))
		if err != nil || total.TotalBytes != want || total.Gap || total.ReconciliationInProgress {
			t.Fatalf("replayed member rollup for %s must total exactly %s bytes without gap or pending reconciliation", period, want)
		}
		totalBytes += total.TotalBytes
	}
	if totalBytes != 3000 {
		t.Fatal("replayed member rollups must total exactly 3000 bytes across all retained periods")
	}
	var entitlementAfter string
	err = pool.QueryRow(ctx, `SELECT string_agg(member_id::text||effective_service_end::text,',' ORDER BY member_id) FROM entitlements`).Scan(&entitlementAfter)
	if err != nil || entitlementBefore != entitlementAfter {
		t.Fatal("entitlements extended during replay")
	}
	var count int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM usage_events WHERE node_id=$1 AND incarnation_id=$2`, batch.NodeId, batch.NodeIncarnationId).Scan(&count)
	if err != nil || count != 10 {
		t.Fatal("ledger rows lost or duplicated")
	}
	for _, event := range batch.Events {
		var matches bool
		err = pool.QueryRow(ctx, `SELECT encode(payload_hash,'hex')=$3 FROM usage_events WHERE node_id=$1 AND incarnation_id=$2 AND event_seq=$4::numeric`, batch.NodeId, batch.NodeIncarnationId, event.PayloadSha256, event.EventSeq.String()).Scan(&matches)
		if err != nil || !matches {
			t.Fatal("retained canonical hash changed")
		}
	}
	report := map[string]any{"kind": "logical snapshot restore and service replay", "status": "passed", "utc": time.Now().UTC(), "restored_ack": before.Signed.HighestContiguousSeq, "previous_ack": 10, "epoch_rotated": state.PreviousLedgerEpoch != state.NewLedgerEpoch, "replayed_accepted": accepted, "replayed_duplicate": duplicate, "second_replay_duplicate": repeat, "final_ack": again.HighestContiguousSeq, "ledger_rows": count, "entitlements_unchanged": true, "canonical_hashes_preserved": true, "protocol_seconds": time.Since(start).Seconds(), "external_reconciliation": false, "archive_source": "host-retained synthetic signed-node upload; not actual node SQLite archive"}
	if len(physical) > 0 {
		report["kind"] = "physical pgBackRest WAL/PITR and service replay"
		report["physical_boundary"] = physical
	}
	report["final_member_total_bytes"] = totalBytes
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if os.WriteFile("/output/restore.json", append(encoded, '\n'), 0644) != nil {
		t.Fatal("restore evidence")
	}
}
