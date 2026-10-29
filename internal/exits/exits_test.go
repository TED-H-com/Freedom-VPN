package exits

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestFakeNetBirdAdapter_IdempotentRouteCreation(t *testing.T) {
	ctx := context.Background()
	nb := NewFakeNetBirdAdapter()
	exitID := uuid.New()
	grpID := uuid.New()

	ev1, err := nb.PublishExit(ctx, PublishExitSpec{
		DeploymentID:   "nb-main",
		ExitID:         exitID,
		NodeID:         uuid.New(),
		PeerID:         "peer-1",
		MemberGroupIDs: []uuid.UUID{grpID},
	})
	if err != nil {
		t.Fatalf("PublishExit #1: %v", err)
	}
	ev2, err := nb.PublishExit(ctx, PublishExitSpec{
		DeploymentID:   "nb-main",
		ExitID:         exitID,
		NodeID:         uuid.New(),
		PeerID:         "peer-1",
		MemberGroupIDs: []uuid.UUID{grpID},
		ExistingRef:    ev1.ExternalID,
	})
	if err != nil {
		t.Fatalf("PublishExit #2: %v", err)
	}
	if ev1.ExternalID != ev2.ExternalID || nb.RouteCount() != 1 {
		t.Fatalf("expected 1 route with same ExternalID, got (%q, %q, count=%d)", ev1.ExternalID, ev2.ExternalID, nb.RouteCount())
	}

	if _, err := nb.WithdrawExit(ctx, "nb-main", exitID, ev1.ExternalID); err != nil {
		t.Fatalf("WithdrawExit: %v", err)
	}
	st, err := nb.ReadExitState(ctx, "nb-main", exitID, ev1.ExternalID)
	if err != nil || st.Enabled {
		t.Fatalf("ReadExitState after withdraw = (%+v, %v), want Enabled=false", st, err)
	}
}
