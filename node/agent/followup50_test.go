package agent

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// TEST ONLY: rows 557–564's ordering, with a manual monotonic clock. The fake
// issuer's readiness switch stands for the synchronous observation transaction;
// it is changed only after inspecting the clean report accepted by that issuer.
func f50Recovery(t *testing.T, r *rig, p *r27BootMissingInterfacePlane, gen uint64) {
	t.Helper()
	p.foreignTC = true
	stepOK(t, r)
	if r.eng.serving || r.eng.applyError == nil {
		t.Fatal("quarantine did not close forwarding")
	}
	r.srv.SetMemberGrantsReady(false)
	p.foreignTC = false
	r.srv.SetFixture(generationFixture(gen, oneBinding(r.clock.Wall())))
	stepOK(t, r) // 557: empty lease, then 558: verified closed Apply.
	if !r.eng.closedRecoveryHeld || r.eng.serving || r.eng.appliedGen != gen {
		t.Fatal("missing closed recovery")
	}
}

func f50Rig(t *testing.T) (*rig, *r27BootMissingInterfacePlane) {
	t.Helper()
	r := newRig(t)
	alignControlClock(r)
	r.eng.cfg.HeartbeatSeconds, r.eng.cfg.PollSeconds = 5, 5
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.srv.SetFixture(generationFixture(13, oneBinding(r.clock.Wall())))
	advanceToRenewal(r)
	stepOK(t, r)
	p := &r27BootMissingInterfacePlane{memPlane: r.plane}
	r.eng.plane = p
	return r, p
}

func f50ReadyReport(t *testing.T, r *rig) {
	t.Helper()
	rep, ok := r.srv.LastObservation()
	gen, err := rep.AppliedConfigGeneration.Get()
	if !ok || err != nil || uint64(gen) != r.eng.appliedGen || rep.ProvisioningState != api.ProvisioningStateReady || rep.SafetyStatus != api.SafetyStatusPassed || rep.MemberForwardingEnabled {
		t.Fatalf("central has not accepted clean recovery ACK: %+v", rep)
	}
}

func TestD157_AT02_ClosedRecoveryRenewsAfterAcceptedReadyReport(t *testing.T) {
	r, p := f50Rig(t)
	f50Recovery(t, r, p, 14)
	sent, slot := r.eng.leaseAttemptAt, r.eng.provisionalRetryAt
	// Assert the old scheduling defect first, so the before-change failure is
	// precisely the five-second wait, not a changed reporting assertion.
	if delay := r.eng.runStepDelay(sent, 5*time.Second); delay != 0 {
		t.Fatalf("closed recovery waits %s for D-067 slot; want immediate wake after accepted ready report", delay)
	}
	f50ReadyReport(t, r)
	requests := r.srv.LeaseRequests()
	beforeReq := requests[len(requests)-1]
	if !beforeReq.AppliedConfigGeneration.IsNull() {
		t.Fatal("quarantined request ACKed an unverified generation")
	}
	r.srv.SetMemberGrantsReady(true)
	before, baselines := r.srv.LeaseRenewals(), p.baselineCalls
	stepOK(t, r) // 563/564, without advancing to the slot.
	if r.clock.Mono() >= slot || r.srv.LeaseRenewals() != before+1 || !r.eng.serving || len(r.eng.lease.Members) != 1 || p.baselineCalls != baselines {
		t.Fatalf("early grant did not serve via verified Apply: serving=%v renewals=%d baseline=%d", r.eng.serving, r.srv.LeaseRenewals(), p.baselineCalls)
	}
	requests = r.srv.LeaseRequests()
	early := requests[len(requests)-1]
	ack, err := early.AppliedConfigGeneration.Get()
	if err != nil || ack != 14 || early.RenewalRequestId == beforeReq.RenewalRequestId {
		t.Fatal("early request missing fresh ID or applied ACK")
	}
	for range 3 {
		stepOK(t, r)
	}
	if r.srv.LeaseRenewals() != before+1 {
		t.Fatal("recovery renewed twice")
	}
}

func TestD157_EmptyEarlyAnswerKeepsD067AndSecondRecoveryGetsOne(t *testing.T) {
	r, p := f50Rig(t)
	for _, gen := range []uint64{14, 15} {
		f50Recovery(t, r, p, gen)
		f50ReadyReport(t, r)
		before, baseline := r.srv.LeaseRenewals(), p.baselineCalls
		sent := r.clock.Mono()
		stepOK(t, r) // Keep issuer grant-less for this one early answer.
		if r.srv.LeaseRenewals() != before+1 || r.eng.serving {
			t.Fatal("missing single early renewal")
		}
		if r.eng.provisionalRetryAt != sent+10*time.Second {
			t.Fatalf("D-067 second gap=%s", r.eng.provisionalRetryAt-sent)
		}
		for range 9 {
			r.clock.Advance(time.Second)
			stepOK(t, r)
			if r.srv.LeaseRenewals() != before+1 || p.baselineCalls != baseline || r.eng.serving {
				t.Fatal("repeat early renewal, baseline or forwarding")
			}
		}
		r.srv.SetMemberGrantsReady(true)
		r.clock.Advance(time.Second)
		stepOK(t, r)
		if !r.eng.serving || r.srv.LeaseRenewals() != before+2 {
			t.Fatal("ordinary scheduled grant did not restore serving")
		}
	}
}

func TestD157_RecoveryWaitsForObservationAcceptance(t *testing.T) {
	r, p := f50Rig(t)
	// Reject only the recovery reports, after the quarantine report was accepted.
	p.foreignTC = true
	stepOK(t, r)
	p.foreignTC = false
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetFixture(generationFixture(14, oneBinding(r.clock.Wall())))
	r.srv.SetObservationRetryAfter(1, 2)
	stepOK(t, r)
	before := r.srv.LeaseRenewals()
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before {
		t.Fatal("renewed before ready report acceptance")
	}
	r.clock.Advance(r.eng.observationRetryAt - r.clock.Mono())
	stepOK(t, r)
	f50ReadyReport(t, r)
	// A successful early flush may dispatch in this Step; otherwise Run wakes
	// immediately. Both must be before the original five-second slot.
	if r.srv.LeaseRenewals() == before {
		stepOK(t, r)
	}
	if r.srv.LeaseRenewals() != before+1 {
		t.Fatal("accepted report did not permit exactly one early renewal")
	}
}

func TestD157_EarlyRenewalHonorsRetryAfterAndDoesNotRearmOnFailure(t *testing.T) {
	r, p := f50Rig(t)
	f50Recovery(t, r, p, 14)
	f50ReadyReport(t, r)
	before := r.srv.LeaseRenewals()
	// A pre-existing server lower bound still wins over the one-shot wake.
	r.eng.leaseRetryAt = r.clock.Mono() + 2*time.Second
	if got := r.eng.runStepDelay(r.clock.Mono(), 5*time.Second); got != 2*time.Second {
		t.Fatalf("Retry-After wake=%s", got)
	}
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before {
		t.Fatal("ignored Retry-After")
	}
	r.clock.Advance(2 * time.Second)
	r.srv.SetLeaseRetryAfter(1, 3)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 || r.eng.earlyRecoveryRenewal() {
		t.Fatal("failed early request did not consume opportunity")
	}
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 || r.eng.serving {
		t.Fatal("early failure retried or served")
	}
}

func TestD157_OldOrUnsafeReportCannotArmRecoveryRenewal(t *testing.T) {
	for _, kind := range []string{"old_id", "failed_safety", "wrong_generation", "peer_job", "frozen", "wrong_receipt"} {
		t.Run(kind, func(t *testing.T) {
			r, p := f50Rig(t)
			f50Recovery(t, r, p, 14)
			rep, _ := r.srv.LastObservation()
			// A fresh opportunity has no report bound yet, just like the next clean Apply.
			r.eng.closedRecoveryRenewal = recoveryRenewal{armed: true}
			accepted := api.ObservationAccepted{Accepted: true, ReportId: rep.ReportId}
			switch kind {
			case "failed_safety":
				rep.SafetyStatus = api.SafetyStatusFailed
			case "wrong_generation":
				rep.AppliedConfigGeneration.Set(13)
			case "peer_job":
				rep.Netbird.PeerId.Set("test-peer")
			case "frozen":
				v := true
				accepted.IdentityFrozen = &v
			case "wrong_receipt":
				accepted.ReportId = r.eng.bootID
			}
			if kind != "old_id" {
				r.eng.noteRecoveryReport(rep)
			}
			r.eng.acceptRecoveryReport(rep, accepted)
			if r.eng.earlyRecoveryRenewal() {
				t.Fatal("unsafe or unbound report authorized early renewal")
			}
		})
	}
}
