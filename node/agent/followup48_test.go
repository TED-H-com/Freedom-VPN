package agent

import (
	"context"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"strings"
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

// Even an unhealthy probe cannot waive a read failure on a present interface.
func TestF48_ReadFailureNeedsIndependentAbsenceEvidence(t *testing.T) {
	for _, tc := range []struct {
		name        string
		exists      bool
		err         error
		foreign     bool
		wantMissing bool
	}{
		{"absent", false, nil, false, true},
		{"present", true, nil, false, false},
		{"unreadable_inventory", false, context.DeadlineExceeded, false, false},
		{"real_filter", false, nil, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			p := &f48InterfacePlane{r27BootMissingInterfacePlane: &r27BootMissingInterfacePlane{memPlane: r.plane, missingInterface: !tc.foreign, foreignTC: tc.foreign}, exists: tc.exists, err: tc.err}
			r.eng.plane = p
			r.eng.withdraw(t.Context())
			r.eng.auditBlocked(t.Context())
			if r.eng.serving {
				t.Fatal("audit opened forwarding")
			}
			if tc.wantMissing {
				if r.eng.applyError != nil || r.eng.containment != nil || r.eng.quarantinePending {
					t.Fatal("absence latched containment")
				}
			} else if r.eng.applyError == nil || r.eng.applyError.Code != api.ErrorCodeTCFOREIGNFILTER || !r.eng.quarantinePending {
				t.Fatalf("unproven absence waived containment: %+v", r.eng.applyError)
			}
		})
	}
}

type f48InterfacePlane struct {
	*r27BootMissingInterfacePlane
	exists bool
	err    error
}

func (p *f48InterfacePlane) InterfaceExists(context.Context, string) (bool, error) {
	return p.exists, p.err
}

type f48LinkInventory struct {
	enforce.Executor
	raw string
	err error
}

func (x f48LinkInventory) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	if name != "ip" || strings.Join(args, " ") != "-j link show" {
		return nil, errors.New("unexpected inventory command")
	}
	return []byte(x.raw), x.err
}
func TestF48_SystemPlaneInterfaceInventory(t *testing.T) {
	for _, tc := range []struct {
		raw          string
		exists, fail bool
	}{
		{`[]`, false, false}, {`[{"ifname":"fv-fh"}]`, true, false},
		{`[{"ifname":"fv-other"}]`, false, false}, {`null`, false, true}, {`[{}]`, false, true}, {`broken`, false, true},
	} {
		p := SystemPlane{Executor: f48LinkInventory{raw: tc.raw}}
		exists, err := p.InterfaceExists(t.Context(), "fv-fh")
		if exists != tc.exists || (err != nil) != tc.fail {
			t.Fatalf("%s: exists=%v err=%v", tc.raw, exists, err)
		}
	}
}
