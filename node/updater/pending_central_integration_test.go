//go:build integration

package updater

import (
	"context"
	"errors"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// D-080 currently has no central job-registration contract. Prove the local
// result exists and is not mistaken for acceptance by the real result service.
func TestD080_InstallerResultNeedsCentralJobRegistration(t *testing.T) {
	l := newCentralLab(t)
	id := l.node("amd64", "disabled", "registered", true)
	w := newWorld(t)
	quietDesired(w)
	w.eng.Node.ID = id.NodeID
	w.eng.Node.Incarnation = id.IncarnationID
	rel := w.makeRelease("installer", 1, []int{1})
	pending := stageInstaller(t, w, rel)
	if err := w.eng.ApplyPending(context.Background(), pending); err != nil {
		t.Fatal(err)
	}
	raw, result := lastPayload(t, w)
	ack, err := l.svc.SubmitResult(l.ctx, id, raw)
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.Code != apierr.ResourceNotFound || ack.Accepted {
		t.Fatalf("unexpected central response: %+v %v", ack, err)
	}
	out, err := w.eng.Journal.Outbox(context.Background())
	if err != nil || len(out) != 1 || out[0].ResultID != result.ResultId.String() || out[0].Status != updatejournal.OutboxPending {
		t.Fatalf("lost unaccepted result: %+v %v", out, err)
	}
}
