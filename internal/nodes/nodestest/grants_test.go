package nodestest

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/leases"
	"github.com/google/uuid"
)

func TestD087_GrantEligibilitySnapshotKeepsProductionLocking(t *testing.T) {
	g := NewMemberGrantEligibility()
	on := g.ReaderFactory(nil)
	if _, ok := on.(leases.DBEntitlementReader); !ok {
		t.Fatal("ready must preserve the concrete production reader and its share-lock path")
	}
	g.SetMemberGrantsReady(false)
	off := g.ReaderFactory(nil)
	g.SetMemberGrantsReady(true)
	// A flip cannot change either of the two reads within an in-flight renewal.
	for range 2 {
		members, err := off.ListLeaseEntitlementsForGroups(t.Context(), nil, []uuid.UUID{uuid.New()}, time.Now())
		if err != nil || len(members) != 0 {
			t.Fatalf("withheld snapshot changed after flip: %v %v", members, err)
		}
	}
	if _, ok := g.ReaderFactory(nil).(leases.DBEntitlementReader); !ok {
		t.Fatal("next renewal must restore production eligibility and locking")
	}
}
