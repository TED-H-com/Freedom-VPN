package agent

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Exercise repeated hung control requests through the signed deadline, using
// the production probe and Step. The manual clock represents lease time only.
func TestF48_AT13_HungControlPreservesLocalChecksUntilDeadline(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.StickyLease(false)
	var probeErr error
	r.eng.prober = ctxAwareNetbirdProber(&probeErr)
	enrollBoot(t, r)
	b := oneBinding(r.clock.Wall())
	b.ValidUntil = r.clock.Wall().Add(time.Hour)
	fix := generationFixture(2, b)
	fix.TTL = time.Hour
	r.srv.SetFixture(fix)
	r.srv.SetMaxOfflineSeconds(60)
	stepOK(t, r)
	r.manualUsage(t)
	deadline := r.eng.policyExpires
	baselines := r.plane.baselines
	r.eng.cfg.WatchdogStaleSeconds = 1
	r.srv.SetDelay(5 * time.Second)
	defer r.srv.SetDelay(0)
	for _, remaining := range []time.Duration{46 * time.Second, 4 * time.Second, 0} {
		r.clock.Advance(deadline - r.clock.Mono() - remaining)
		stepOK(t, r)
		if probeErr != nil || r.eng.decision.Safety != api.SafetyStatusPassed {
			t.Fatalf("local safety inherited hung control: %v %+v", probeErr, r.eng.decision)
		}
		if remaining > 0 {
			if !r.eng.serving || r.plane.baselines != baselines {
				t.Fatalf("withdrawn before D: %+v", mustBeat(t, r))
			}
		} else if r.eng.serving || mustBeat(t, r).Reason != "lease_expired" {
			t.Fatalf("did not expire at D: %+v", mustBeat(t, r))
		}
	}
}

func TestF48_LocalWorkBudgetSeparatesStepAndBoot(t *testing.T) {
	control, stop := context.WithCancel(t.Context())
	stop()
	step, done := newLocalBudget(context.WithValue(t.Context(), controlBudgetKey{}, control), time.Second)
	defer done()
	if err := localWorkContext(step).Err(); err != nil {
		t.Fatalf("Step local work canceled: %v", err)
	}
	boot, finish := bootLocal(control, time.Second)
	defer finish()
	if err := localWorkContext(boot).Err(); err != context.Canceled {
		t.Fatalf("Boot adoption ignored caller: %v", err)
	}
	if err := localContext(boot).Err(); err != nil {
		t.Fatalf("Boot cleanup canceled: %v", err)
	}
}
