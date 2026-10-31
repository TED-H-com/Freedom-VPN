//go:build integration

package desiredstate

import (
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

func reviewHold(t *testing.T, f *restoreFixture, node uuid.UUID) bool {
	t.Helper()
	_, err := f.compiler.Compile(t.Context(), node)
	var hold *apierr.Error
	return errors.As(err, &hold) && hold.Code == apierr.DependencyUnavailable && hold.Message == usage.RestoreObservationStatus
}

// TestReviewR16_RestoreHoldBoundaries proves the D-095 cutoff is the central
// receipt time of the current incarnation. A replayed pre-freeze receipt, a
// new receipt whose central clock is still behind the cutoff, and another
// node's fresh receipt do not release this node. A node created after the
// freeze still compiles, and a device revocation is not blocked.
func TestReviewR16_RestoreHoldBoundaries(t *testing.T) {
	f := newRestoreFixture(t)
	other, otherInc, _, _ := seedCompilerTestNode(t, t.Context(), f.pool, uuid.Nil, api.PublicationIntentUnpublished, nil)
	f.freeze()
	var cutoff time.Time
	if err := f.pool.QueryRow(t.Context(), `SELECT created_at FROM operations WHERE resource_id=$1 AND principal=$2`, f.node, usage.RestoreObservationPrincipal).Scan(&cutoff); err != nil {
		t.Fatal(err)
	}
	f.begin()
	// A node with no freeze operation is held at the recovery record, which
	// BeginRecovery stamps with the clock after freeze. Keep that instant.
	recoveryAt := f.now
	f.submit(f.oldReport)
	if !reviewHold(t, f, f.node) || !reviewHold(t, f, other) {
		t.Fatal("replay of the pre-freeze receipt released a hold")
	}

	// Central receipt time comes from the service clock. Move it behind the
	// cutoff while the node claims to have observed a day later.
	f.now = cutoff.Add(-time.Second)
	stale := f.report(11)
	stale.ObservedAt = cutoff.Add(24 * time.Hour)
	f.submit(stale)
	if !reviewHold(t, f, f.node) {
		t.Fatal("receipt behind the cutoff released the hold because the node clock was ahead")
	}

	f.now = cutoff.Add(time.Second)
	freshOther := f.report(11)
	freshOther.NodeId, freshOther.NodeIncarnationId = other, otherInc
	if _, err := f.nodes.SubmitObservation(t.Context(), other, otherInc, freshOther); err != nil {
		t.Fatal(err)
	}
	if reviewHold(t, f, other) {
		t.Fatal("other node's own fresh receipt did not release its hold")
	}
	if !reviewHold(t, f, f.node) {
		t.Fatal("another node's fresh receipt released this node")
	}

	later, laterInc, _, _ := seedCompilerTestNode(t, t.Context(), f.pool, uuid.Nil, api.PublicationIntentUnpublished, nil)
	if !reviewHold(t, f, later) {
		t.Fatal("a node created during recovery compiled before its own fresh receipt")
	}
	if _, err := f.pool.Exec(t.Context(), `UPDATE exits SET publication_intent='published', resource_version=resource_version+1 WHERE node_id=$1`, later); err != nil {
		t.Fatalf("hold blocked publication of an unrelated node: %v", err)
	}

	member, device := uuid.New(), uuid.New()
	if _, err := f.pool.Exec(t.Context(), `INSERT INTO members (id, email, display_name, status) VALUES ($1, $2, $2, 'active')`, member, member.String()+"@example.invalid"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(t.Context(), `
		INSERT INTO member_devices (id, member_id, deployment_id, peer_id, status)
		VALUES ($1, $2, 'nb-main', $3, 'pending')`, device, member, "peer-"+device.String()[:8]); err != nil {
		t.Fatal(err)
	}
	tag, err := f.pool.Exec(t.Context(), `UPDATE member_devices SET status='revoked', revoked_at=clock_timestamp(), resource_version=resource_version+1 WHERE id=$1 AND status='pending'`, device)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("hold blocked device revocation: %v rows=%d", err, tag.RowsAffected())
	}
	if !reviewHold(t, f, f.node) || !reviewHold(t, f, later) {
		t.Fatal("device revocation released a node hold")
	}
	f.now = recoveryAt.Add(time.Second)
	freshLater := f.report(1)
	freshLater.NodeId, freshLater.NodeIncarnationId = later, laterInc
	if _, err := f.nodes.SubmitObservation(t.Context(), later, laterInc, freshLater); err != nil {
		t.Fatal(err)
	}
	if reviewHold(t, f, later) {
		t.Fatal("the new node's own fresh receipt left it held")
	}
	if !reviewHold(t, f, f.node) {
		t.Fatal("the new node's receipt released the restored node")
	}

	f.now = recoveryAt.Add(2 * time.Second)
	f.submit(f.report(11))
	if reviewHold(t, f, f.node) {
		t.Fatal("fresh central receipt of the current incarnation left the hold in place")
	}
}
