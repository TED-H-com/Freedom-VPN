package updatejournal

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/google/uuid"
)

func TestD063_ProbeOmissionAtomicAndIdempotent(t *testing.T) {
	ctx := context.Background()
	j, _ := openTest(t)
	ch := healthattest.Challenge{TxnID: uuid.Must(uuid.NewV7()).String(), ReleaseID: uuid.Must(uuid.NewV7()).String(),
		Nonce: strings.Repeat("a", 64), IssuedAt: time.Now().UTC(), ProbeOmitted: &healthattest.ProbeOmission{PeerID: "test-peer", Reason: "node_addresses_unavailable"}}
	if _, err := j.BeginTxn(ctx, Txn{TxnID: ch.TxnID, JobID: "job", Attempt: 1, Generation: 1, ReleaseID: ch.ReleaseID}); err != nil {
		t.Fatal(err)
	}
	// A failing event insert must roll back the meta evidence as well.
	if _, err := j.db.Exec(`CREATE TRIGGER refuse_probe_event BEFORE INSERT ON txn_events BEGIN SELECT RAISE(ABORT, 'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := j.RecordProbeOmission(ctx, ch, "candidate"); err == nil {
		t.Fatal("event failure ignored")
	}
	if records, err := j.ProbeOmissions(ctx, ch.TxnID); err != nil || len(records) != 0 {
		t.Fatal("partial audit survived rollback", err)
	}
	if _, err := j.db.Exec(`DROP TRIGGER refuse_probe_event`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := j.RecordProbeOmission(ctx, ch, "candidate"); err != nil {
			t.Fatal(err)
		}
	}
	if events, err := j.TxnEvents(ctx, ch.TxnID); err != nil || len(events) != 2 {
		t.Fatal("duplicate or missing audit", err)
	}
	ch.ProbeOmitted = &healthattest.ProbeOmission{PeerID: "test-peer", Reason: "node_source"}
	if err := j.RecordProbeOmission(ctx, ch, "candidate"); err == nil {
		t.Fatal("same issuance changed its evidence")
	}
	if records, err := j.ProbeOmissions(ctx, ch.TxnID); err != nil || len(records) != 1 || records[0].ProbeOmitted.Reason != "node_addresses_unavailable" {
		t.Fatal("conflict overwrote evidence", err)
	}
	// Standalone revocation rollback has no row in txns. Its fresh recovery
	// transaction/nonce still identifies durable evidence after consumption.
	ch.TxnID = uuid.Must(uuid.NewV7()).String()
	if err := j.RecordProbeOmission(ctx, ch, "rollback"); err != nil {
		t.Fatal(err)
	}
	if records, err := j.ProbeOmissions(ctx, ch.TxnID); err != nil || len(records) != 1 || records[0].Phase != "rollback" {
		t.Fatal("recovery omission missing", err)
	}
	var version int
	if err := j.db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil || version != 2 {
		t.Fatal("unnecessary schema change", err)
	}
}
