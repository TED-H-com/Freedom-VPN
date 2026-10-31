package agent

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type livePlane struct {
	*memPlane
	foreign     bool
	injectApply bool
	read        func(context.Context) error
	models      []enforce.Model
	assignments []map[uuid.UUID]uint16
}

func (p *livePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if p.foreign {
		return enforce.ApplyResult{}, &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter}
	}
	res, err := p.memPlane.Apply(ctx, m, prev)
	p.foreign = p.injectApply
	return res, err
}

func (p *livePlane) VerifyServing(ctx context.Context, m enforce.Model, assignments map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	p.models = append(p.models, m)
	p.assignments = append(p.assignments, maps.Clone(assignments))
	if p.read != nil {
		if err := p.read(ctx); err != nil {
			return enforce.ServingEvidence{}, err
		}
	}
	if p.foreign {
		return enforce.ServingEvidence{}, &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter}
	}
	return enforce.ServingEvidence{}, nil
}

func assertLiveWithdrawn(t *testing.T, r *rig, generation uint64, message string) {
	t.Helper()
	b := mustBeat(t, r)
	if b.Serving || b.PolicyExpires != 0 || b.Reason != "safety_check_failed" || !b.BaselineOK || b.Generation != generation {
		t.Fatalf("withdrawn heartbeat: %+v", b)
	}
	if r.eng.serving || r.eng.policyExpires != 0 || r.eng.expectServing.Load() {
		t.Fatal("retained serving evidence survived withdrawal")
	}
	wantCode := api.ErrorCodeHEALTHCHECKFAILED
	if message == "TC_FOREIGN_FILTER" {
		wantCode = api.ErrorCodeTCFOREIGNFILTER
	}
	if r.eng.applyError == nil || r.eng.applyError.Code != wantCode || r.eng.applyError.Message != message {
		t.Fatalf("verification error: %+v", r.eng.applyError)
	}
	rep, ok := r.srv.LastObservation()
	gen, err := rep.AppliedConfigGeneration.Get()
	if !ok || err != nil || uint64(gen) != generation || rep.MemberForwardingEnabled || !hasMessage(rep.Errors, message) {
		t.Fatalf("control observation: forwarding=%v gen=%v errors=%v", rep.MemberForwardingEnabled, gen, rep.Errors)
	}
	durable, ok, err := r.eng.store.Desired(t.Context())
	if err != nil || !ok || durable.Applied != generation {
		t.Fatalf("durable acknowledgment advanced: %+v %v", durable, err)
	}
}

func TestD085_ISO01_BetweenApplyInjection(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	p := &livePlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)
	if len(p.models) != 3 { // before reconciliation, before ACK, before heartbeat
		t.Fatalf("steady step live reads=%d", len(p.models))
	}
	oldModel, oldAssignments := *r.eng.appliedModel, maps.Clone(r.eng.appliedAssignments)
	p.models = nil
	p.assignments = nil
	p.foreign = true // after the previous successful Apply and heartbeat
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	before := r.plane.applies
	stepOK(t, r)
	assertLiveWithdrawn(t, r, 2, "TC_FOREIGN_FILTER")
	if r.plane.applies != before || len(p.models) != 2 || !reflect.DeepEqual(p.models[0], oldModel) || !reflect.DeepEqual(p.assignments[0], oldAssignments) {
		t.Fatal("drift was overwritten by Apply or checked against a desired model")
	}
	// The ordinary renewal slot supplies a lease for the offered generation.
	// No stale generation ACK is used to force that renewal while degraded.
	advanceToRenewal(r)
	stepOK(t, r) // the offered generation is accepted, but still cannot be ACKed
	if r.eng.desired.ConfigGeneration != 3 {
		t.Fatal("control access lost while degraded")
	}
	assertLiveWithdrawn(t, r, 2, "TC_FOREIGN_FILTER")
	p.foreign = false
	stepOK(t, r)
	if !mustBeat(t, r).Serving || r.eng.appliedGen != 3 || r.eng.applyError != nil || r.plane.applies != before+1 {
		t.Fatalf("clean full Apply did not restore serving: serving=%v gen=%d error=%+v applies=%d before=%d reasons=%v lease=%d", r.eng.serving, r.eng.appliedGen, r.eng.applyError, r.plane.applies, before, r.eng.decision.Reasons, r.eng.lease.ConfigGeneration)
	}
	t.Log("generation 2 serving -> filter injected -> next heartbeat non-serving/safety_check_failed; TC_FOREIGN_FILTER -> desired 3 accepted, ACK remains 2 -> filter removed, full Apply -> generation 3 serving")
}

func TestD085_InjectionAfterApplyBeforeAck(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	p := &livePlane{memPlane: r.plane, injectApply: true}
	r.eng.plane = p
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)
	assertLiveWithdrawn(t, r, 2, "TC_FOREIGN_FILTER")
}

func advanceToRenewal(r *rig) {
	window := r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt)
	lead := r.eng.renewLead(window)
	r.clock.Advance(window - lead + lead/10 + time.Second)
}

func TestD085_FinalReadWithBusyApplyLock(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &livePlane{memPlane: r.plane}
	p.read = func(context.Context) error {
		// First read is clean. Inject after it, before the deferred heartbeat.
		p.foreign = p.foreign || len(p.models) == 2
		return nil
	}
	r.eng.plane = p
	lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	before := r.plane.applies
	stepOK(t, r)
	assertLiveWithdrawn(t, r, 2, "TC_FOREIGN_FILTER")
	if r.plane.applies != before || len(p.models) != 3 {
		t.Fatal("busy Apply bypassed the final live read")
	}
}

func TestD085_ReadTimeoutAndCanceledStep(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "timeout", true: "canceled"}[canceled], func(t *testing.T) {
			r := newRig(t)
			serveBoundGeneration(t, r)
			r.eng.cfg.HeartbeatSeconds = 10
			r.eng.cfg.PollSeconds = 1
			p := &livePlane{memPlane: r.plane}
			p.read = func(ctx context.Context) error {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 500*time.Millisecond {
					t.Fatal("read not bounded below shorter cadence")
				}
				<-ctx.Done()
				return nil // late success must still fail closed
			}
			r.eng.plane = p
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			message := "serving verification timed out"
			if canceled {
				cancel()
				// Process cancellation still permits an independent bounded
				// kernel read. This deliberately stalled read times out.
			}
			started := time.Now()
			err := r.eng.Step(ctx)
			if (canceled && !errors.Is(err, context.Canceled)) || (!canceled && err != nil) {
				t.Fatal(err)
			}
			if time.Since(started) > 2*time.Second {
				t.Fatal("verification did not respect timeout")
			}
			assertLiveWithdrawn(t, r, 2, message)
		})
	}
}

func TestD085_ShorterCadenceSubtractsStepWork(t *testing.T) {
	for _, cfg := range []Config{{HeartbeatSeconds: 10, PollSeconds: 1}, {HeartbeatSeconds: 1, PollSeconds: 10}} {
		r := newRig(t)
		r.eng.cfg.HeartbeatSeconds, r.eng.cfg.PollSeconds = cfg.HeartbeatSeconds, cfg.PollSeconds
		start := r.clock.Mono()
		r.clock.Advance(200 * time.Millisecond)
		if r.eng.stepInterval() != time.Second || r.eng.servingReadTimeout() != 500*time.Millisecond || r.eng.remainingStepDelay(start, r.eng.stepInterval()) != 800*time.Millisecond {
			t.Fatal("step work added another full heartbeat/poll sleep")
		}
		r.clock.Advance(time.Second)
		if r.eng.remainingStepDelay(start, r.eng.stepInterval()) != 0 {
			t.Fatal("overdue step did not wake immediately")
		}
	}
}

// Includes context creation/deadline/cancellation and the fake live read.
// This is an overhead measurement, not a kernel/netlink latency claim.
func TestD085_LiveReadCostFake(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	const reads = 10000
	start := time.Now()
	for range reads {
		if !r.eng.verifyCurrentServing(t.Context()) {
			t.Fatal("clean fake live read failed")
		}
	}
	perRead := time.Since(start) / reads
	t.Logf("fake bounded live read: n=%d mean=%s; steady serving step adds 3 reads ~= %s", reads, perRead, 3*perRead)
}
