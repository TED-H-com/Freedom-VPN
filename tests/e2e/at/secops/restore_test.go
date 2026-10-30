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
	"github.com/jackc/pgx/v5/pgxpool"
)

// Executed only inside the network-none restoration container. The immutable
// dump and retained batch originate from the running lab, not a seeded copy.
// This is the logical-restore subset of §17.3 / AT30, not a WAL/PITR claim.
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
	if before.Signed.HighestContiguousSeq != 5 {
		t.Fatal("restore did not return earlier checkpoint 5")
	}
	actor := "00000000-0000-0000-0000-000000000003"
	var entitlementBefore string
	err = pool.QueryRow(ctx, `SELECT string_agg(member_id::text||effective_service_end::text,',' ORDER BY member_id) FROM entitlements`).Scan(&entitlementBefore)
	if err != nil {
		t.Fatal("entitlement baseline")
	}
	if svc.FreezeIngest(ctx, actor, "TEST ONLY isolated logical restore") != nil {
		t.Fatal("freeze")
	}
	if _, err = svc.IngestBatch(ctx, identity, batch); err == nil {
		t.Fatal("frozen ingestion accepted")
	}
	state, err := svc.BeginRecovery(ctx, usage.RecoveryRequest{Actor: actor, Reason: "TEST ONLY retained window replay", RestoredTo: before.Signed.IssuedAt, AffectedSince: batch.Events[0].IntervalStart})
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
	if accepted != 5 || duplicate != 5 || repeat != 10 || again.HighestContiguousSeq != 10 {
		t.Fatalf("replay counts accepted=%d duplicate=%d repeat=%d ack=%s", accepted, duplicate, repeat, again.HighestContiguousSeq)
	}
	if svc.CompleteRecovery(ctx, actor) != nil {
		t.Fatal("complete recovery")
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
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if os.WriteFile("/output/restore.json", append(encoded, '\n'), 0644) != nil {
		t.Fatal("restore evidence")
	}
}
