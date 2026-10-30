package agent

import (
	"context"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

type revocationPlane struct {
	*memPlane
	ex                  revocationExecutor
	models              []enforce.Model
	results             []enforce.ApplyResult
	errors              []error
	baselinesAfterApply int
	unverified          bool
}

func (p *revocationPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	p.ex.model = m
	p.models = append(p.models, m)
	p.baselinesAfterApply = 0
	res, err := enforce.Apply(ctx, &p.ex, m, prev)
	if p.unverified {
		res.Verified = false
	}
	p.results = append(p.results, res)
	p.errors = append(p.errors, err)
	return res, err
}

func (p *revocationPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	p.baselinesAfterApply++
	return p.memPlane.Baseline(ctx, dp)
}

func TestISO01_PendingConntrackDeleteKeepsServingAndRetries(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)
	if !r.eng.serving || !p.results[0].Verified || p.errors[0] != nil {
		t.Fatalf("fixture did not install verified ruleset: %v", p.errors)
	}
	previousCounters := cloneCounters(r.eng.counters)
	binding := oneBinding(r.clock.Wall())
	binding.AllowedIpv4 = []string{"100.64.1.2/32"}
	r.srv.SetFixture(generationFixture(3, binding))
	p.ex.failDelete = true
	stepOK(t, r)
	wantRevoke := []netip.Addr{netip.MustParseAddr("100.64.1.1")}
	assertPending := func() {
		t.Helper()
		last := len(p.results) - 1
		if !p.results[last].Verified || !enforce.IsConntrackDeleteFailure(p.errors[last]) {
			t.Fatalf("not a verified conntrack failure: %+v %v", p.results[last], p.errors[last])
		}
		if !reflect.DeepEqual(p.models[last].Revoke, wantRevoke) || p.baselinesAfterApply != 0 {
			t.Fatalf("pending revoke lost or baseline restored: revoke=%v baselines=%d", p.models[last].Revoke, p.baselinesAfterApply)
		}
		if !reflect.DeepEqual(r.eng.counters, previousCounters) {
			t.Fatal("partial apply result discarded prior counters")
		}
		// Force the normal outbox delivery so the fake control sees this tick.
		r.eng.lastObs = time.Time{}
		if err := r.eng.observe(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := r.eng.flush(context.Background()); err != nil {
			t.Fatal(err)
		}
		rep, ok := r.srv.LastObservation()
		gen, err := rep.AppliedConfigGeneration.Get()
		if !ok || err != nil || gen != 3 || r.eng.appliedGen != 3 || !rep.MemberForwardingEnabled || !r.eng.decision.Serve || len(rep.ForwardingBlockReasons) != 0 || rep.SafetyStatus != api.SafetyStatusPassed {
			t.Fatalf("pending delete withdrew verified generation: serving=%v gen=%d reasons=%v errors=%v", rep.MemberForwardingEnabled, gen, rep.ForwardingBlockReasons, rep.Errors)
		}
		for _, check := range rep.Checks {
			if check.Status != api.HealthCheckStatusPassed {
				t.Fatalf("pending delete failed check: %+v", check)
			}
		}
		if len(rep.Errors) != 1 || rep.Errors[0].Code != api.ErrorCode(enforce.ReasonIsolation) || rep.Errors[0].Stage != "apply" || rep.Errors[0].Message != "conntrack delete pending; serving ruleset verified" {
			t.Fatalf("pending entry: %+v", rep.Errors)
		}
		if beat := mustBeat(t, r); !beat.Serving || beat.Reason != "serving" || beat.Generation != 3 {
			t.Fatalf("pending heartbeat: %+v", beat)
		}
	}
	assertPending()
	stored, ok, err := r.eng.store.Desired(context.Background())
	if err != nil || !ok || !stored.HasApplied || stored.Applied != 3 {
		t.Fatalf("generation not durably acknowledged: %+v %v", stored, err)
	}
	// Reload durable runtime state as a replacement process would. The new
	// binding addresses alone no longer identify the pending old address.
	r.eng.pendingRevoke = nil
	if err := r.eng.loadRuntime(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(p.ex.deletes) != 1 {
		t.Fatalf("delete retried within tick: %v", p.ex.deletes)
	}
	for range 2 {
		applies, deletes := len(p.models), len(p.ex.deletes)
		stepOK(t, r)
		assertPending()
		if len(p.models) != applies+1 || len(p.ex.deletes) != deletes+1 {
			t.Fatal("retry did not wait for next apply tick")
		}
	}
	// Renewal remains possible while the pending entry is present.
	renewals := r.srv.LeaseRenewals()
	r.eng.cfg.RenewLeadSeconds = 3600
	advance := r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt) * 3 / 4
	r.clock.Advance(advance)
	r.srv.SetClockOffset(advance)
	stepOK(t, r)
	assertPending()
	if r.srv.LeaseRenewals() != renewals+1 {
		t.Fatal("pending delete suppressed renewal")
	}
	p.ex.failDelete = false
	stepOK(t, r)
	if !reflect.DeepEqual(p.models[len(p.models)-1].Revoke, wantRevoke) {
		t.Fatal("successful retry lost revoke list")
	}
	r.eng.lastObs = time.Time{}
	if err := r.eng.observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	rep, _ := r.srv.LastObservation()
	if len(rep.Errors) != 0 || !rep.MemberForwardingEnabled || !r.eng.decision.Serve {
		t.Fatalf("successful delete did not clear pending entry: %+v", rep.Errors)
	}
	if err := r.eng.loadRuntime(context.Background()); err != nil {
		t.Fatal(err)
	}
	stepOK(t, r)
	if len(p.models[len(p.models)-1].Revoke) != 0 {
		t.Fatal("successful deletes remained queued")
	}
}

func TestISO01_UnverifiedConntrackFailureStillRestoresBaseline(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane, unverified: true, ex: revocationExecutor{failDelete: true}}
	r.eng.plane = p
	binding := oneBinding(r.clock.Wall())
	binding.AllowedIpv4 = []string{"100.64.1.2/32"}
	r.srv.SetFixture(generationFixture(3, binding))
	stepOK(t, r)
	if !enforce.IsConntrackDeleteFailure(p.errors[0]) || p.results[0].Verified || r.eng.serving || r.eng.decision.Serve || p.baselinesAfterApply == 0 || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonSafetyCheckFailed) {
		t.Fatal("unverified typed delete failure did not fail closed")
	}
}

func TestISO01_VerifyFailureStillRestoresBaseline(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane, ex: revocationExecutor{failVerify: true}}
	r.eng.plane = p
	stepOK(t, r)
	if p.results[0].Verified || p.errors[0] == nil || enforce.IsConntrackDeleteFailure(p.errors[0]) || r.eng.serving || r.eng.decision.Serve || p.baselinesAfterApply == 0 {
		t.Fatalf("verify failure did not fail closed: result=%+v err=%v", p.results[0], p.errors[0])
	}
	if len(p.ex.scripts) < 2 || !strings.Contains(p.ex.scripts[len(p.ex.scripts)-1], "fvpn_baseline=1") || len(p.ex.deletes) != 0 {
		t.Fatal("enforce did not restore baseline before any delete")
	}
}

func TestLEASE01_RetryAfterPreservesFreshLeaseUntilDeadline(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(map[bool]string{false: "renew", true: "expire"}[expire], func(t *testing.T) {
			r := newRig(t)
			serveBoundGeneration(t, r)
			r.eng.cfg.RenewLeadSeconds = 3600
			advance := r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt) * 3 / 4
			r.clock.Advance(advance)
			r.srv.SetClockOffset(advance)
			wait := 30 * time.Second
			if expire {
				wait = r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt) + time.Minute
			}
			r.srv.SetLeaseRetryAfter(1, int(wait/time.Second))
			count := r.srv.LeaseRenewals()
			assertServing := func() {
				t.Helper()
				lease, _ := checkBy(r.eng.decision.Checks, api.NodeCheckNameLease)
				if !r.eng.serving || !r.eng.decision.Serve || lease.Status != api.HealthCheckStatusPassed || lease.Detail == nil || !strings.Contains(*lease.Detail, "Retry-After") || mustBeat(t, r).Reason != "serving" {
					t.Fatalf("backoff invalidated fresh lease: serving=%v reasons=%v check=%+v", r.eng.serving, r.eng.decision.Reasons, lease)
				}
				if r.srv.LeaseRenewals() != count+1 {
					t.Fatal("renewed before Retry-After")
				}
			}
			stepOK(t, r)
			assertServing()
			stepOK(t, r)
			assertServing()
			if expire {
				deadline := r.eng.leaseMono + r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt)
				r.clock.Advance(deadline - r.clock.Mono())
				stepOK(t, r)
				lease, _ := checkBy(r.eng.decision.Checks, api.NodeCheckNameLease)
				if r.eng.serving || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonLeaseExpired) || lease.Status != api.HealthCheckStatusFailed || codeOf(t, lease) != api.ErrorCodeLEASEEXPIRED || r.srv.LeaseRenewals() != count+1 {
					t.Fatal("backoff extended expired lease or retried early")
				}
				return
			}
			r.clock.Advance(wait - time.Second)
			stepOK(t, r)
			assertServing()
			r.clock.Advance(time.Second)
			stepOK(t, r)
			if r.srv.LeaseRenewals() != count+2 || !r.eng.serving || r.eng.leaseRetryAt != 0 {
				t.Fatal("renewal did not recover at Retry-After")
			}
		})
	}
}
