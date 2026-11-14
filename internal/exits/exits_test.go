package exits

import (
	"context"
	"errors"
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

func TestFakeNetBirdAdapter_ReadExitPage(t *testing.T) {
	ctx := context.Background()
	nb := NewFakeNetBirdAdapter()
	served := uuid.New()
	other := uuid.New()
	nb.InjectRouteState(served, ObservedExitState{Exists: true, Enabled: true, ExternalID: "nb-route-" + served.String(), PeerID: "peer"})
	nb.SetExitEvidence(other, PublicationEvidenceError("route_ambiguous"))
	nb.SetUnservedDeployment("other-deployment")

	if _, err := nb.ReadExitPage(ctx, "other-deployment", []ExitPageRef{{ExitID: served}}); !errors.Is(err, ErrDeploymentNotServed) || nb.PageCalls() != 0 {
		t.Fatalf("unserved deployment call=%d err=%v", nb.PageCalls(), err)
	}
	page := []ExitPageRef{{ExitID: served, ExternalID: "nb-route-" + served.String()}, {ExitID: other}}
	got, err := nb.ReadExitPage(ctx, "nb-main", page)
	if err != nil || nb.PageCalls() != 1 || len(got) != 2 || !got[0].State.Exists || got[0].Evidence != nil {
		t.Fatalf("page=(%+v, %v) calls=%d", got, err, nb.PageCalls())
	}
	var evidence PublicationEvidenceError
	if !errors.As(got[1].Evidence, &evidence) || string(evidence) != "route_ambiguous" {
		t.Fatalf("evidence=%v", got[1].Evidence)
	}
	nb.SetPageError(errors.New("TEST ONLY page failed"))
	if _, err = nb.ReadExitPage(ctx, "nb-main", page); err == nil || nb.PageCalls() != 2 {
		t.Fatalf("page error call=%d err=%v", nb.PageCalls(), err)
	}
	seen := nb.PageExits()
	if len(seen) != 2 || len(seen[0]) != 2 || seen[0][0] != served || seen[0][1] != other {
		t.Fatalf("page exits=%v", seen)
	}
}
