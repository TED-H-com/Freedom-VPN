package agent

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// TEST ONLY: exercise the real executor's observation branch with a harmless
// child, then stop before parsing or mutating kernel state. No nft/tc is run.
type lifecycleProbeExecutor struct{ enforce.Executor }

func (lifecycleProbeExecutor) Output(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	if _, err := (enforce.System{}).Output(ctx, "true"); err != nil {
		return nil, err
	}
	return nil, errors.New("TEST ONLY stop before kernel access")
}

type lifecycleProbePlane struct {
	*memPlane
	timing *enforce.TimingEvidence
}

func (p *lifecycleProbePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, _ := enforce.Apply(ctx, lifecycleProbeExecutor{}, m, prev)
	p.timing = res.Evidence.TimingEvidence()
	return p.memPlane.Apply(ctx, m, prev)
}

func TestF31_CommandLifecycleRequiresBothTimingGates(t *testing.T) {
	for _, tc := range []struct {
		name       string
		mode, path bool
	}{
		{"disabled", false, false}, {"mode_only", true, false},
		{"path_only", false, true}, {"enabled", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			serveBoundGeneration(t, r)
			p := &lifecycleProbePlane{memPlane: r.plane}
			r.eng.plane = p
			r.eng.cfg.TestMode = tc.mode
			if tc.path {
				r.eng.cfg.TestTimingPath = filepath.Join(r.dir, "operations.jsonl")
			}
			stepOK(t, r)
			if !r.eng.serving || p.timing == nil {
				t.Fatal("Apply caller was not exercised")
			}
			if !tc.mode || !tc.path {
				if len(p.timing.Commands) != 0 {
					t.Fatal("disabled caller collected commands")
				}
				return
			}
			if len(p.timing.Commands) != 1 {
				t.Fatalf("commands=%d, want harmless child observation", len(p.timing.Commands))
			}
			c := p.timing.Commands[0]
			if !c.Started || !c.OK || !c.StartAttempt.Available || !c.WaitReturn.Available || c.ChildStartExact.Available {
				t.Fatalf("lost lifecycle/availability semantics: %+v", c)
			}
		})
	}
}

func TestF31_StaleGenerationBaselinePrecedesRenewalWithoutCandidateApply(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	oldLease, applies, baselines := r.eng.lease.LeaseId, r.plane.applies, r.plane.baselines
	r.eng.leaseAttempted = false // beginning of the next Step
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	if err := r.eng.syncDesired(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r.eng.needRenew() {
		t.Fatal("fixture must exercise desired-ahead-of-applied renewal ordering")
	}
	r.eng.evaluate(t.Context())
	if err := r.eng.enforce(t.Context(), r.eng.decision); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || r.eng.appliedGen != 3 || r.eng.lease.LeaseId != oldLease ||
		r.plane.applies != applies || r.plane.baselines <= baselines || r.eng.applyError != nil {
		t.Fatal("ordinary stale generation must baseline and ACK without candidate Apply")
	}
	if err := r.eng.renewAppliedLease(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.serving || r.eng.lease.LeaseId == oldLease || r.eng.lease.ConfigGeneration != 3 || r.plane.applies != applies+1 {
		t.Fatal("matching renewal requires a subsequent serving Apply")
	}
}
