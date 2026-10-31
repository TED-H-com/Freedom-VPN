package agent

import (
	"context"
	"net/netip"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/google/uuid"
)

// TestReviewR15_ReassignedAddressDeleteIsOnceAndAddressOnly is the collateral
// of the nil-UUID migration shield. conntrack -D is keyed by address only, so
// the new device's established flows on that address are deleted. The delete
// is queued once, not on every later apply, and an address that was never in
// the previous or pending identity is not queued.
func TestReviewR15_ReassignedAddressDeleteIsOnceAndAddressOnly(t *testing.T) {
	never := netip.MustParseAddr("100.64.9.9")
	neverBound := []enforce.Binding{{
		DeviceID:  uuid.Must(uuid.NewV7()),
		MemberID:  uuid.Must(uuid.NewV7()),
		AllowedV4: []netip.Prefix{netip.MustParsePrefix(never.String() + "/32")},
	}}
	if got := planRevokes(nil, nil, neverBound); len(got) != 0 {
		t.Fatalf("planRevokes queued %v for an address that was never in prev or pending", got)
	}

	addr := netip.MustParseAddr("100.64.1.1")
	nextDevice := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	nextMember := uuid.MustParse("019a0000-0000-7000-8000-0000000000c2")
	otherMember := uuid.MustParse("019a0000-0000-7000-8000-0000000000c3")
	legacy := []addrID{{Addr: addr}}
	rebound := []enforce.Binding{{
		DeviceID: nextDevice, MemberID: nextMember,
		AllowedV4: []netip.Prefix{netip.MustParsePrefix("100.64.1.1/32")},
	}}
	got := planRevokes(nil, legacy, rebound)
	if len(got) != 1 || got[0] != addr {
		t.Fatalf("planRevokes = %v, want the reassigned address once", got)
	}
	// A different member on the same address is still that address. The tuple
	// the kernel delete sees is the address, not (address, member).
	rebound[0].MemberID = otherMember
	got = planRevokes(nil, legacy, rebound)
	if len(got) != 1 || got[0] != addr {
		t.Fatalf("member change did not keep the address-only delete: %v", got)
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
	next := oneBinding(r.clock.Wall())
	next.DeviceId = nextDevice
	next.MemberId = nextMember
	next.PeerId = "peer-b2"
	next.AllowedIpv4 = []string{"100.64.1.1/32"}
	r.srv.SetFixture(generationFixture(4, next))
	stepOK(t, r)
	if !revokeContains(r.plane.last.Revoke, addr) {
		t.Fatalf("first apply did not queue conntrack -D for %s (revoke=%v)", addr, r.plane.last.Revoke)
	}
	stepOK(t, r)
	if revokeContains(r.plane.last.Revoke, addr) {
		t.Fatalf("second apply queued conntrack -D again for %s (revoke=%v)", addr, r.plane.last.Revoke)
	}
}

func revokeContains(addrs []netip.Addr, want netip.Addr) bool {
	for _, a := range addrs {
		if a == want {
			return true
		}
	}
	return false
}
