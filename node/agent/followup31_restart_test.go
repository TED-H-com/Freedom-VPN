package agent

import (
	"context"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

type cancelAfterReadPlane struct {
	*memPlane
	cancel context.CancelFunc
}

func (p *cancelAfterReadPlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	ev, err := p.memPlane.VerifyServing(ctx, m, a)
	p.cancel()
	return ev, err
}

func TestF31_AT13_CancellationAfterLiveReadKeepsRestartAuthorization(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	r.eng.plane = &cancelAfterReadPlane{r.plane, cancel}
	baselines, lease, deadline := r.plane.baselines, r.eng.lease.LeaseId, r.eng.policyExpires
	if err := r.eng.Step(ctx); err != context.Canceled {
		t.Fatalf("Step returned %v, want cancellation", err)
	}
	if !r.eng.storageOK || r.eng.usagePersist || !r.eng.serving {
		t.Fatalf("canceled Step poisoned healthy storage/serving: storage=%v persist=%v serving=%v; %s", r.eng.storageOK, r.eng.usagePersist, r.eng.serving, r.log.String())
	}
	restartEngine(t, r, r.plane)
	if !r.eng.serving || r.plane.baselines != baselines || r.eng.lease.LeaseId != lease || r.eng.policyExpires != deadline {
		t.Fatal("restart lost original valid authorization")
	}
}
