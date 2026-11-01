package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

type cancelAfterReadPlane struct {
	*memPlane
	cancel context.CancelFunc
}

type cancelLocalWorkPlane struct {
	*memPlane
	cancel context.CancelFunc
	phase  string
	fail   bool
}

func (p *cancelLocalWorkPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if p.phase == "apply" {
		p.cancel()
	}
	if err := ctx.Err(); err != nil {
		return enforce.ApplyResult{}, err
	}
	if p.phase == "apply" {
		if _, bounded := ctx.Deadline(); !bounded || controlContext(ctx).Err() == nil {
			return enforce.ApplyResult{}, errors.New("local completion must be bounded while control is canceled")
		}
	}
	if p.fail {
		return enforce.ApplyResult{}, &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter}
	}
	return p.memPlane.Apply(ctx, m, prev)
}

func (p *cancelLocalWorkPlane) MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error) {
	if p.phase == "member_path" {
		p.cancel()
	}
	if err := ctx.Err(); err != nil {
		return MemberPath{}, err
	}
	return p.memPlane.MemberPath(ctx, m, expected)
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

func TestF31_AT13_CancellationDuringLocalWorkKeepsRestartAuthorization(t *testing.T) {
	for _, phase := range []string{"apply", "member_path"} {
		t.Run(phase, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			r.eng.plane = &cancelLocalWorkPlane{memPlane: r.plane, cancel: cancel, phase: phase}
			baselines, lease, deadline := r.plane.baselines, r.eng.lease.LeaseId, r.eng.policyExpires
			_ = r.eng.Step(ctx)
			if !r.eng.storageOK || r.eng.usagePersist || !r.eng.serving || r.eng.applyError != nil {
				t.Fatalf("cancellation interrupted local %s: storage=%v persist=%v serving=%v error=%v", phase, r.eng.storageOK, r.eng.usagePersist, r.eng.serving, r.eng.applyError)
			}
			restartEngine(t, r, r.plane)
			if !r.eng.serving || r.plane.baselines != baselines || r.eng.lease.LeaseId != lease || r.eng.policyExpires != deadline {
				t.Fatal("restart lost original valid authorization")
			}
		})
	}
}

func TestF31_AT13_CancellationDoesNotMaskRealApplyFailure(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	r.eng.plane = &cancelLocalWorkPlane{memPlane: r.plane, cancel: cancel, phase: "apply", fail: true}
	before := r.plane.baselines
	_ = r.eng.Step(ctx)
	if r.eng.serving || r.eng.applyError == nil || r.plane.baselines <= before || !r.eng.quarantinePending {
		t.Fatal("cancellation hid a real Apply/foreign-tc failure")
	}
}
