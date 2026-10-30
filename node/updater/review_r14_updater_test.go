package updater

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/google/uuid"
)

// TestReviewR14_ProbeOmissionEventSurvivesMetaDelete checks the prune
// follow-up 12 added. DeleteProbeOmissions removes the meta row and leaves
// the append-only txn_events copy, which is the evidence a later challenge
// on the same release can still read from the journal.
func TestReviewR14_ProbeOmissionEventSurvivesMetaDelete(t *testing.T) {
	ctx := context.Background()
	j, err := updatejournal.Open(t.TempDir(), updatejournal.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = j.Close() })
	releaseID := uuid.Must(uuid.NewV7()).String()
	ch := healthattest.Challenge{
		TxnID:     uuid.Must(uuid.NewV7()).String(),
		ReleaseID: releaseID,
		Nonce:     strings.Repeat("ab", 32),
		IssuedAt:  time.Now().UTC(),
		ProbeOmitted: &healthattest.ProbeOmission{
			PeerID: "test-peer",
			Reason: "node_addresses_unavailable",
		},
	}
	if _, err := j.BeginTxn(ctx, updatejournal.Txn{
		TxnID: ch.TxnID, JobID: "job", Attempt: 1, Generation: 1, ReleaseID: releaseID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := j.RecordProbeOmission(ctx, ch, "candidate"); err != nil {
		t.Fatal(err)
	}
	if err := j.DeleteProbeOmissions(ctx, ch.TxnID); err != nil {
		t.Fatal(err)
	}
	records, err := j.ProbeOmissions(ctx, ch.TxnID)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("meta rows remained after DeleteProbeOmissions: %+v", records)
	}
	events, err := j.TxnEvents(ctx, ch.TxnID)
	if err != nil {
		t.Fatal(err)
	}
	for _, ev := range events {
		if strings.Contains(ev.Detail, ch.Nonce) && strings.Contains(ev.Detail, releaseID) {
			return
		}
	}
	t.Fatalf("txn_events lost the probe omission: %+v", events)
}
