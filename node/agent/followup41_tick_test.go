package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TEST ONLY instruments the real Step, lock and heartbeat without networking.
type f41ReadPlane struct {
	*memPlane
	lock        string
	lockedReads int
	failLocked  bool
}

func (p *f41ReadPlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	lock, err := nodelock.TryAcquire(p.lock)
	if errors.Is(err, nodelock.ErrBusy) {
		p.lockedReads++
		if p.failLocked {
			return enforce.ServingEvidence{}, errors.New("TEST ONLY live read failed")
		}
	} else if err == nil {
		lock.Release()
	} else {
		return enforce.ServingEvidence{}, err
	}
	return p.memPlane.VerifyServing(ctx, m, a)
}

func TestAT10_F41_SkipVerifiesUnderLockAndHeartbeats(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	// A setup renewal supplies the exact model held by the current lease.
	r.clock.Advance(time.Second)
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	p := &f41ReadPlane{memPlane: r.plane, lock: r.eng.cfg.LockPath}
	r.eng.plane = p
	applies := r.plane.applies
	expires := r.eng.policyExpires
	for range 3 {
		reads := p.lockedReads
		r.clock.Advance(10 * time.Second)
		stepOK(t, r)
		b := mustBeat(t, r)
		if r.plane.applies != applies || p.lockedReads != reads+1 || !b.Serving || !b.BaselineOK || b.Mono != r.clock.Mono() || b.PolicyExpires != expires || b.Reason != "serving" {
			t.Fatalf("skip/read/beat: applies=%d/%d reads=%d/%d beat=%+v", r.plane.applies, applies, p.lockedReads, reads, b)
		}
	}
	p.failLocked = true
	r.clock.Advance(10 * time.Second)
	stepOK(t, r)
	if b := mustBeat(t, r); b.Serving || b.PolicyExpires != 0 || r.eng.applyError == nil {
		t.Fatalf("failed live read claimed serving: %+v", b)
	}
}

func TestAT13_F41_TimeDrivenModelChanges(t *testing.T) {
	for _, kind := range []string{"valid_from", "valid_until", "sibling_expiry", "member_expiry"} {
		t.Run(kind, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			now := r.clock.Wall()
			a := oneBinding(now)
			b := a
			b.DeviceId = uuid.Must(uuid.NewV7())
			b.AllowedIpv4 = []string{"100.64.0.12/32"}
			if kind == "member_expiry" {
				b.MemberId = uuid.Must(uuid.NewV7())
			}
			if kind == "valid_from" {
				b.ValidFrom = now.Add(20 * time.Second)
			} else {
				b.ValidUntil = now.Add(20 * time.Second)
			}
			bindings := []api.DeviceBinding{a, b}
			if kind == "valid_until" {
				bindings = []api.DeviceBinding{b}
			}
			r.srv.SetFixture(fixtureWithBindings(3, bindings))
			stepOK(t, r)
			// Exercise an actual shorter member grant separately from binding expiry.
			if kind == "member_expiry" {
				r.eng.desired.DeviceBindings[1].ValidUntil = a.ValidUntil
				for i := range r.eng.lease.Members {
					if r.eng.lease.Members[i].MemberId == b.MemberId {
						r.eng.lease.Members[i].LeaseEnd = now.Add(20 * time.Second)
					}
				}
				stepOK(t, r)
			}
			before := r.plane.applies
			r.clock.Advance(10 * time.Second)
			stepOK(t, r)
			if r.plane.applies != before {
				t.Fatalf("applied before boundary: %d -> %d", before, r.plane.applies)
			}
			r.clock.Advance(10 * time.Second)
			stepOK(t, r)
			want := 1
			if kind == "valid_from" {
				want = 2
			}
			if kind == "valid_until" {
				want = 0
			}
			if want == 0 {
				if r.eng.serving || len(r.plane.last.Bindings) != 0 {
					t.Fatal("expired only device still serving")
				}
			} else if !r.eng.serving || r.plane.applies != before+1 || len(r.plane.last.Bindings) != want {
				t.Fatalf("boundary not applied on tick: serving=%v applies=%d/%d bindings=%d want=%d reasons=%v", r.eng.serving, r.plane.applies, before, len(r.plane.last.Bindings), want, r.eng.decision.Reasons)
			}
		})
	}
}

func TestAT13_F41_LeaseRenewalApplies(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	before := r.plane.applies
	old := r.eng.lease.LeaseId
	r.clock.Advance(time.Minute)
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	if r.eng.lease.LeaseId == old || r.plane.applies != before+1 || !r.eng.serving {
		t.Fatal("renewal did not apply")
	}
	r.clock.Advance(10 * time.Second)
	stepOK(t, r)
	if r.plane.applies != before+1 {
		t.Fatal("unchanged renewed model applied again")
	}
}
