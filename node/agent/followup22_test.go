package agent

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestD088_IdentityFrozenTLSAndRestart(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	before := r.srv.LeaseRenewals()
	frozen := true
	r.srv.SetIdentityFrozen(&frozen)
	r.eng.lastObs = time.Time{}
	// An already consumed renewal slot and Retry-After do not defer the freeze.
	r.eng.leaseRetryAt = r.clock.Mono() + time.Hour
	stepOK(t, r)
	assertFrozen := func() {
		t.Helper()
		b := mustBeat(t, r)
		if b.Serving || b.PolicyExpires != 0 || b.Reason != "identity_frozen" || !r.eng.identityFrozen || len(r.plane.last.Bindings) != 0 {
			t.Fatalf("freeze not withdrawn: %+v", b)
		}
		rep, ok := r.srv.LastObservation()
		if !ok || rep.MemberForwardingEnabled || !rep.AppliedConfigGeneration.IsNull() || !hasMessage(rep.Errors, "identity_frozen") {
			t.Fatalf("frozen wire report: %+v", rep)
		}
	}
	assertFrozen()
	if r.srv.LeaseRenewals() != before+1 || len(r.eng.lease.Members) != 0 || r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt) != 60*time.Second {
		t.Fatal("freeze did not immediately renew grant-less 60s lease")
	}
	if v, _, err := r.eng.store.Meta(t.Context(), "identity_frozen"); err != nil || v != "1" {
		t.Fatal("freeze not durable", err)
	}
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 {
		t.Fatal("repeated flag bypassed schedule again")
	}
	r.clock.Advance(10 * time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+2 {
		t.Fatal("frozen renewal did not follow D-087")
	}
	// A newer desired state and a populated signed lease cannot unfreeze this
	// incarnation, even after the optional flag is removed and the agent restarts.
	r.srv.SetIdentityFrozen(nil)
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	restartEngine(t, r, r.plane)
	stepOK(t, r)
	assertFrozen()
	if r.eng.appliedGen != 2 {
		t.Fatal("frozen identity ACKed new generation")
	}
	t.Log("TLS true -> same-step baseline/heartbeat/observation, immediate grant-less 60s renewal; scheduled retry; restart and populated envelope stay frozen without generation ACK")
}

func TestD088_AbsentAndFalseDoNotChangeIdentity(t *testing.T) {
	f := false
	for _, flag := range []*bool{nil, &f} {
		name := "absent"
		if flag != nil {
			name = "false"
		}
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			before := r.srv.LeaseRenewals()
			r.srv.SetIdentityFrozen(flag)
			r.eng.lastObs = time.Time{}
			stepOK(t, r)
			if r.eng.identityFrozen || !mustBeat(t, r).Serving || r.srv.LeaseRenewals() != before {
				t.Fatal("absent/false changed live identity")
			}
			// False is no change for a latched identity as well.
			r.eng.freezeIdentity(t.Context())
			r.eng.lastObs = time.Time{}
			stepOK(t, r)
			if !r.eng.identityFrozen || mustBeat(t, r).Serving {
				t.Fatal("absent/false cleared freeze")
			}
		})
	}
}

type quarantinePlane struct {
	*livePlane
	quarantines  []enforce.Model
	qerr         error
	configured   bool
	beforeQ      func()
	applyForeign bool
}

func (p *quarantinePlane) Quarantine(ctx context.Context, model enforce.Model) (enforce.QuarantineEvidence, error) {
	if p.beforeQ != nil {
		p.beforeQ()
	}
	p.quarantines = append(p.quarantines, model)
	q := enforce.QuarantineEvidence{Quarantined: p.qerr == nil, Recorded: true,
		Links:    []enforce.QuarantineLink{{Interface: model.Datapath.MemberIngress, Before: &enforce.LinkState{AdminUp: true, OperState: "UP"}, After: &enforce.LinkState{AdminUp: p.qerr != nil, OperState: "DOWN"}}},
		Baseline: enforce.BaselineEvidence{Installed: true, Verified: true}}
	if p.qerr != nil {
		q.Links[0].Error = "down_verify_failed"
	}
	return q, p.qerr
}
func (p *quarantinePlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	ev, err := p.livePlane.VerifyServing(ctx, m, a)
	if p.foreign {
		ev.TC = []enforce.TCReadEvidence{{Interface: m.Datapath.MemberIngress, Attachment: "ingress", Objects: []enforce.TCObject{{Kind: "matchall", Actions: []string{"mirred"}}}}}
	}
	return ev, err
}
func (p *quarantinePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if p.applyForeign {
		p.foreign = true
		q, _ := p.Quarantine(ctx, m)
		ev, err := p.VerifyServing(ctx, m, nil)
		return enforce.ApplyResult{Evidence: ev, Quarantine: &q}, err
	}
	return p.livePlane.Apply(ctx, m, prev)
}
func (p *quarantinePlane) MemberPath(context.Context, enforce.Model, *MemberPath) (MemberPath, error) {
	if !p.configured {
		return MemberPath{}, errMemberPath
	}
	return MemberPath{}, nil
}

func assertTCWire(t *testing.T, r *rig) {
	t.Helper()
	rep, ok := r.srv.LastObservation()
	if !ok || rep.MemberForwardingEnabled {
		t.Fatal("missing non-serving TLS report")
	}
	found, checked := false, false
	for _, e := range rep.Errors {
		if e.Code == api.ErrorCodeTCFOREIGNFILTER {
			found = true
		}
		if e.Code == api.ErrorCodeHEALTHCHECKFAILED && strings.Contains(e.Message, "TC_FOREIGN_FILTER") {
			t.Fatal("legacy fallback on wire")
		}
		if len(e.Message) > 500 {
			t.Fatal("oversized wire evidence")
		}
	}
	for _, c := range rep.Checks {
		if code, err := c.ErrorCode.Get(); err == nil && code == api.ErrorCodeTCFOREIGNFILTER {
			checked = true
		}
	}
	if !found || !checked {
		t.Fatalf("wire codes: errors=%v checks=%v", rep.Errors, rep.Checks)
	}
	raw, _ := json.Marshal(rep)
	if !strings.Contains(string(raw), "mirred") || !strings.Contains(string(raw), "quarantined") {
		t.Fatal("wire evidence absent")
	}
	var ev containmentEvidence
	if err := json.Unmarshal([]byte(mustBeat(t, r).Evidence), &ev); err != nil || ev.Code != api.ErrorCodeTCFOREIGNFILTER {
		t.Fatal("heartbeat evidence", err)
	}
}

func TestD090_QuarantineRetainedModelEveryStepAndD096Recovery(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	old := *r.eng.appliedModel
	p := &quarantinePlane{livePlane: &livePlane{memPlane: r.plane}, configured: true}
	p.beforeQ = func() {
		lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
		if err == nil {
			lock.Release()
			t.Fatal("quarantine ran outside Apply lock")
		}
		if !errors.Is(err, nodelock.ErrBusy) {
			t.Fatal(err)
		}
	}
	r.eng.plane = p
	p.foreign = true
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	stepOK(t, r)
	assertTCWire(t, r)
	if r.eng.appliedModel == nil || !reflect.DeepEqual(*r.eng.appliedModel, old) || len(p.quarantines) == 0 {
		t.Fatal("baseline cleared retained applied model")
	}
	for _, m := range p.quarantines {
		if !reflect.DeepEqual(m, old) {
			t.Fatal("quarantine used a desired/cleared model")
		}
	}
	// Canceled and already frozen returns must still re-contain a daemon raise.
	p.qerr = errors.New("SECRET RAW OUTPUT 203.0.113.99")
	before := len(p.quarantines)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := r.eng.Step(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(p.quarantines) <= before || mustBeat(t, r).Serving {
		t.Fatal("canceled return skipped quarantine")
	}
	assertTCWire(t, r)
	raw, _ := json.Marshal(r.eng.containment)
	if strings.Contains(string(raw), "SECRET") || strings.Contains(string(raw), "203.0.113.99") {
		t.Fatal("raw error leaked")
	}
	if r.eng.containment.Error != "quarantine_incomplete" {
		t.Fatal("partial error lost")
	}
	before = len(p.quarantines)
	r.eng.identityFrozen = true
	stepOK(t, r)
	if len(p.quarantines) <= before {
		t.Fatal("already blocked step skipped quarantine")
	}
	r.eng.identityFrozen = false // test only; production never clears this latch
	p.foreign = false
	p.qerr = nil
	p.configured = false
	advanceToRenewal(r)
	stepOK(t, r)
	if mustBeat(t, r).Serving || mustBeat(t, r).Reason != "member_path_unconfigured" || r.eng.appliedGen != 2 {
		t.Fatalf("unconfigured recovery advertised: %+v", mustBeat(t, r))
	}
	before = len(p.quarantines)
	p.configured = true
	stepOK(t, r)
	if !mustBeat(t, r).Serving || r.eng.appliedGen != 3 || len(p.quarantines) != before {
		t.Fatal("configured clean Apply did not recover")
	}
}

func TestD090_ApplyQuarantineEvidenceTLS(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	p := &quarantinePlane{livePlane: &livePlane{memPlane: r.plane}, configured: true, applyForeign: true}
	r.eng.plane = p
	stepOK(t, r)
	assertTCWire(t, r)
	rep, _ := r.srv.LastObservation()
	found := false
	for _, e := range rep.Errors {
		found = found || e.Stage == "apply_quarantine"
	}
	if !found || r.eng.containment.ApplyQuarantine == nil {
		t.Fatal("ApplyResult.Quarantine not reported")
	}
}
