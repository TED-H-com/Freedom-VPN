package agent

import (
	"context"
	"errors"
	"net/netip"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/google/uuid"
)

// TestReviewR14_NilUUIDMigrationShieldsReassignedAddress is the v2 row
// migrateAppliedAddressIdentity writes: both device_id and member_id nil.
// That identity is not the device that later binds the address, so the
// address is queued for conntrack -D before the new device is served.
func TestReviewR14_NilUUIDMigrationShieldsReassignedAddress(t *testing.T) {
	addr := netip.MustParseAddr("100.64.1.1")
	nextDevice := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	nextMember := uuid.MustParse("019a0000-0000-7000-8000-0000000000c2")
	legacy := []addrID{{Addr: addr}}
	rebound := []enforce.Binding{{
		DeviceID:  nextDevice,
		MemberID:  nextMember,
		AllowedV4: []netip.Prefix{netip.MustParsePrefix("100.64.1.1/32")},
	}}
	if got := planRevokes(nil, legacy, rebound); len(got) != 1 || got[0] != addr {
		t.Fatalf("planRevokes did not queue conntrack -D for a nil-UUID address reassigned to device %s (revoke=%v)", nextDevice, got)
	}

	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	nilID := uuid.Nil.String()
	if err := r.eng.store.SaveAddresses(context.Background(), []agentstore.AppliedAddress{{
		Addr: addr.String(), DeviceID: nilID, MemberID: nilID,
	}}); err != nil {
		t.Fatal(err)
	}
	restartEngine(t, r, r.plane)
	if len(r.eng.addresses) != 1 || r.eng.addresses[0].Addr != addr || r.eng.addresses[0].known() {
		t.Fatalf("boot did not reload the migrated row: %+v", r.eng.addresses)
	}
	next := oneBinding(r.clock.Wall())
	next.DeviceId = nextDevice
	next.MemberId = nextMember
	next.PeerId = "peer-b2"
	next.AllowedIpv4 = []string{"100.64.1.1/32"}
	r.srv.SetFixture(generationFixture(4, next))
	stepOK(t, r)
	queued := false
	for _, a := range r.plane.last.Revoke {
		if a == addr {
			queued = true
		}
	}
	if !queued {
		t.Fatalf("engine apply did not queue conntrack -D after the migrated address moved to device %s (revoke=%v serving=%v)", nextDevice, r.plane.last.Revoke, r.eng.serving)
	}
	rows, err := r.eng.store.Addresses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Addr == addr.String() && row.DeviceID == nilID && row.MemberID == nilID {
			return
		}
	}
	if !queued {
		t.Fatalf("the nil-UUID row was replaced by the new identity without a conntrack delete (rows=%+v)", rows)
	}
}

// TestReviewR14_NilUUIDMigrationKeepsPendingUntilDeleteSucceeds is the
// failed-delete half of the same migration. The nil identity stays in
// pending_revocations, and ConntrackDeleteError.Failed lists the address,
// until a later verified apply deletes it.
func TestReviewR14_NilUUIDMigrationKeepsPendingUntilDeleteSucceeds(t *testing.T) {
	addr := netip.MustParseAddr("100.64.1.1")
	nextDevice := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	nextMember := uuid.MustParse("019a0000-0000-7000-8000-0000000000c2")
	nilID := uuid.Nil.String()
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	if err := r.eng.store.SaveAddresses(context.Background(), []agentstore.AppliedAddress{{
		Addr: addr.String(), DeviceID: nilID, MemberID: nilID,
	}}); err != nil {
		t.Fatal(err)
	}
	p := &failedSubsetPlane{memPlane: r.plane, fail: map[netip.Addr]bool{addr: true}}
	restartEngine(t, r, p)
	if len(r.eng.addresses) != 1 || r.eng.addresses[0].known() {
		t.Fatalf("boot did not reload the migrated row: %+v", r.eng.addresses)
	}
	next := oneBinding(r.clock.Wall())
	next.DeviceId = nextDevice
	next.MemberId = nextMember
	next.PeerId = "peer-b2"
	next.AllowedIpv4 = []string{"100.64.1.1/32"}
	r.srv.SetFixture(generationFixture(4, next))
	stepOK(t, r)

	want := addrID{Addr: addr}
	assertPendingEqual(t, r.eng.pendingRevoke, want)
	assertPendingEqual(t, persistedPending(t, r), want)
	if r.eng.conntrackAttempts != 1 || r.eng.conntrackBlocked {
		t.Fatalf("attempts=%d blocked=%v", r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	last := len(p.errs) - 1
	var ct *enforce.ConntrackDeleteError
	if last < 0 || !errors.As(p.errs[last], &ct) || ct.Pending != 1 || len(ct.Failed) != 1 || ct.Failed[0] != addr {
		t.Fatalf("apply error %+v", p.errs)
	}
	assertAppliedOnly(t, r, addr.String(), nextDevice.String(), nextMember.String())

	p.fail = nil
	stepOK(t, r)
	if len(r.eng.pendingRevoke) != 0 || len(persistedPending(t, r)) != 0 || r.eng.conntrackAttempts != 0 {
		t.Fatalf("success left pending=%+v attempts=%d", r.eng.pendingRevoke, r.eng.conntrackAttempts)
	}
	rows, err := r.eng.store.Addresses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Addr == addr.String() && row.DeviceID == nilID && row.MemberID == nilID {
			t.Fatalf("nil identity survived a successful delete: %+v", rows)
		}
	}
}
