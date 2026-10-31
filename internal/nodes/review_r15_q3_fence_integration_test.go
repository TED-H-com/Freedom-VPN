//go:build integration

package nodes_test

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestReviewR15_QuietPeerReleasesTheRevokeAfterNinetyOneSeconds is the r14
// fence at its boundary. A peer that has been quiet for 91s no longer holds
// member_device.revoke. The next compile has no binding. The peer row can
// still say forwarding with the isolation error until that peer reports again.
func TestReviewR15_QuietPeerReleasesTheRevokeAfterNinetyOneSeconds(t *testing.T) {
	w := newR14World(t)
	group := w.group("grp-r15-91s", "r15 ninety one seconds")
	peer := w.enroll()
	ready, _ := w.prepare(peer, false, false)
	w.observe(ready)
	w.publish(w.primary.NodeId, group)
	w.publish(peer.NodeId, group)
	member := w.bindDevice("r15-91s", "100.64.0.61", []uuid.UUID{group})
	if _, n := w.prepare(w.primary, true, false); n != 1 {
		t.Fatalf("clean bindings = %d, want 1", n)
	}
	if _, n := w.prepare(peer, true, false); n != 1 {
		t.Fatalf("peer bindings = %d, want 1", n)
	}
	opID := w.revoke(member)
	dirty, n := w.prepare(peer, true, true)
	if n != 0 {
		t.Fatalf("peer still bound after revoke: %d", n)
	}
	w.observe(dirty)
	if op := w.operation(opID); op.Status == api.OperationStatusSucceeded {
		t.Fatal("fresh dirty peer did not hold the revoke; the 91s release has nothing to observe")
	}

	w.now = w.now.Add(nodes.DefaultStaleTimeout + time.Second)
	clean, n := w.prepare(w.primary, true, false)
	if n != 0 {
		t.Fatalf("clean node still bound 91s later: %d", n)
	}
	w.observe(clean)
	op := w.operation(opID)
	st, ok := nodesStep(op)
	if !ok || op.Status != api.OperationStatusSucceeded || st.Status != api.OperationStepStatusSucceeded {
		t.Fatalf("peer quiet for %s; member_device.revoke %s is %s (%s)", nodes.DefaultStaleTimeout+time.Second, opID, op.Status, stepText(st))
	}

	var forwarding bool
	var errors []byte
	if err := w.r.Pool.QueryRow(w.t.Context(), `SELECT coalesce(member_forwarding_observed,false), recent_errors FROM nodes WHERE id=$1`, peer.NodeId).Scan(&forwarding, &errors); err != nil {
		t.Fatal(err)
	}
	again, n := w.prepare(peer, false, false)
	_ = again
	t.Logf("exposure: stale timeout %s released the revoke; peer row still forwarding=%v recent_errors=%s; next compile bindings=%d; offline lease cap %ds; conntrack delete bound is 12 attempts (node/agent maxConntrackDeleteAttempts)",
		nodes.DefaultStaleTimeout, forwarding, errors, n, leases.DefaultMaxOfflineLeaseSeconds)
	if n != 0 {
		t.Fatalf("next desired state still binds the revoked device: %d", n)
	}
}

// TestReviewR15_RevokedPlanRevisionStillHoldsTheFence grants a plan in group
// OLD, revokes that grant, and grants a plan whose exit_group_ids is empty.
// The projector writes the active projection. A peer published only in OLD
// must not keep member_device.revoke once the active projection is empty.
func TestReviewR15_RevokedPlanRevisionStillHoldsTheFence(t *testing.T) {
	w := newR14World(t)
	ctx := w.t.Context()
	oldGroup := w.group("grp-r15-old", "r15 old plan group")
	peer := w.enroll()
	ready, _ := w.prepare(peer, false, false)
	w.observe(ready)
	w.publish(peer.NodeId, oldGroup)
	member := w.bindDevice("r15-old", "100.64.0.62", []uuid.UUID{oldGroup})

	var grantID uuid.UUID
	if err := w.r.Pool.QueryRow(ctx, `SELECT id FROM entitlement_grants WHERE member_id=$1 AND revoked_at IS NULL`, member.id).Scan(&grantID); err != nil {
		t.Fatal(err)
	}
	store, err := entitlement.NewStore(entitlement.TestConfig())
	if err != nil {
		t.Fatal(err)
	}
	store.Now = func() time.Time { return w.now }
	membersSvc := &members.Service{Pool: w.r.Pool, Entitlements: store}
	cmd := entitlement.Command{Actor: entitlement.Actor{Role: "admin", ID: w.admin, RequestID: uuid.New()}, Key: uuid.New()}
	if err = membersSvc.RevokeGrant(ctx, cmd, member.id, grantID, "TEST ONLY revoke the plan that named the old exit group"); err != nil {
		t.Fatal(err)
	}

	// A plan revision with empty exit_group_ids is rejected by validPlan, so
	// the active projection goes empty because the only grant was revoked
	// and Recompute ran. The revoked grant's plan_revisions row still names
	// the old group.
	var groups []byte
	if err = w.r.Pool.QueryRow(ctx, `SELECT coalesce(projection->'exit_group_ids', '[]'::jsonb) FROM entitlements WHERE member_id=$1`, member.id).Scan(&groups); err != nil {
		t.Fatal(err)
	}
	if string(groups) != "[]" {
		t.Fatalf("active projection exit_group_ids = %s after revoking the only grant, want [] before claiming the fence (setup, not a finding)", groups)
	}

	dirty, n := w.prepare(peer, true, true)
	if n != 0 {
		t.Fatalf("peer compiled %d bindings from an empty active projection", n)
	}
	w.observe(dirty)
	opID := w.revoke(member)
	clean, n := w.prepare(w.primary, true, false)
	if n != 0 {
		t.Fatalf("clean node compiled %d bindings from an empty active projection", n)
	}
	w.observe(clean)
	op := w.operation(opID)
	st, ok := nodesStep(op)
	if ok && op.Status == api.OperationStatusSucceeded && st.Status == api.OperationStepStatusSucceeded {
		return
	}
	t.Fatalf("TODO(review): active projection exit_group_ids is [] after the grant that named group %s was revoked, and the peer %s is published only in that group. member_device.revoke %s stayed %s (%s). otherNodePendingIso matches entitlement_grants joined to plan_revisions with no revoked_at IS NULL, so the stale revision still holds the fence. The projector itself filters revoked_at.",
		oldGroup, peer.NodeId, opID, op.Status, stepText(st))
}
