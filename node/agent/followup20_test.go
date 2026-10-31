package agent

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// Exercise the same delay calculation Run uses, with time spent receiving a
// lease and finishing a control pass. Polling must not round a renewal up to
// another full tick; RenewLeadSeconds=120 must not defer a 60s lease to expiry.
func TestD067_EngineRenewalTiming(t *testing.T) {
	for _, tc := range []struct {
		name   string
		window time.Duration
		hb     int
		poll   time.Duration
		want   []time.Duration
	}{
		{"applying_60s_lease", time.Minute, 10, 10 * time.Second, []time.Duration{10, 20, 30, 30, 30}},
		{"long_lease", time.Hour, 10, 10 * time.Second, []time.Duration{10, 20, 40, 60, 60}},
		{"heartbeat_over_cap", time.Hour, 90, 90 * time.Second, []time.Duration{60, 60, 60}},
		{"short_lease_long_heartbeat", time.Minute, 90, 90 * time.Second, []time.Duration{30, 30, 30}},
		{"unaligned_poll", time.Hour, 10, 7 * time.Second, []time.Duration{10, 20, 40, 60, 60}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Hour
			e := renewFixture(1, 1, start, start, time.Now().UTC(), tc.window)
			e.enrolled = true
			e.cfg.HeartbeatSeconds = tc.hb
			e.lease.Members = nil
			clock := e.clock.(*timeanchor.Manual)
			for i, seconds := range tc.want {
				sent := clock.Mono()
				e.leaseMono = sent
				e.leaseAttemptAt = sent
				e.leaseAttempted = true
				// Response parsing takes 2s; remaining step work takes 3s.
				clock.Advance(2 * time.Second)
				e.noteProvisionalSchedule(sent)
				clock.Advance(3 * time.Second)
				ready := clock.Mono() // ready observation accepted here
				for ticks := 0; ; ticks++ {
					if ticks > 20 {
						t.Fatal("renewal never became due")
					}
					delay := e.nextStepDelay(tc.poll)
					if delay < 0 || (delay == 0 && !e.needRenew()) {
						t.Fatalf("unexpected busy loop at %s", clock.Mono()-sent)
					}
					clock.Advance(delay)
					e.leaseAttempted = false
					if e.needRenew() {
						break
					}
					clock.Advance(3 * time.Second) // another control pass
				}
				gap := clock.Mono() - sent
				// A control pass already in progress may cross the slot by
				// its 3s work duration; Run must add no further sleep.
				if gap < seconds*time.Second || gap > seconds*time.Second+3*time.Second || gap > time.Minute || clock.Mono()-ready > time.Minute {
					t.Fatalf("renewal %d: request gap=%s ready gap=%s want=%ss", i+1, gap, clock.Mono()-ready, seconds)
				}
				t.Logf("renewal %d: request gap=%s ready gap=%s", i+1, gap, clock.Mono()-ready)
			}
		})
	}
}

// A ready central starts granting on the next scheduled request without a
// desired-generation change or agent restart, even with heartbeat/poll >60s.
func TestD067_ReadyRenewsWithin60Seconds(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.cfg.HeartbeatSeconds = 90
	r.eng.cfg.PollSeconds = 90
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	stepOK(t, r)
	assertProvisionalDecision(t, r.eng.decision)
	sent := r.eng.leaseMono
	before := r.srv.LeaseRenewals()
	// Model the rest of that pass and readiness arriving five seconds later.
	r.clock.Advance(5 * time.Second)
	ready := r.clock.Mono()
	r.srv.SetMemberGrantsReady(true)
	delay := r.eng.nextStepDelay(90 * time.Second)
	if delay != 55*time.Second {
		t.Fatalf("next engine wake=%s, want 55s", delay)
	}
	r.clock.Advance(delay - time.Nanosecond)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before || r.eng.serving {
		t.Fatal("renewed before the scheduled slot or served without grants")
	}
	r.clock.Advance(time.Nanosecond)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 || !r.eng.serving || len(r.eng.lease.Members) != 1 || r.eng.appliedGen != 3 {
		t.Fatalf("renewals=%d serving=%v members=%d gen=%d", r.srv.LeaseRenewals()-before, r.eng.serving, len(r.eng.lease.Members), r.eng.appliedGen)
	}
	t.Logf("grant-less request to populated request=%s; ready to populated request=%s", r.clock.Mono()-sent, r.clock.Mono()-ready)
}

func TestD067_GrantlessFailureDoesNotSpinOrOverrideRetryAfter(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	stepOK(t, r)
	r.clock.Advance(10 * time.Second)
	r.srv.FailLeases(1)
	before := r.srv.LeaseRenewals()
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 || r.eng.serving {
		t.Fatal("failure did not stay closed with one attempt per step")
	}
	// The fake's failure carries a 1s Retry-After. Waking at that deadline
	// is allowed, but a zero-delay retry loop is not.
	if delay := r.eng.nextStepDelay(10 * time.Second); delay <= 0 || delay > 10*time.Second {
		t.Fatalf("failed renewal wake=%s, want a bounded positive delay", delay)
	}
	r.eng.leaseRetryAt = 0
	if r.eng.nextStepDelay(10*time.Second) != 10*time.Second {
		t.Fatal("failure without Retry-After did not use the control cadence")
	}
	// HTTP Retry-After is an independent lower bound, including >60s.
	r.eng.leaseRetryAt = r.clock.Mono() + 90*time.Second
	before = r.srv.LeaseRenewals()
	for range 8 {
		r.clock.Advance(r.eng.nextStepDelay(10 * time.Second))
		stepOK(t, r)
	}
	if r.srv.LeaseRenewals() != before || r.eng.serving {
		t.Fatal("renewed during Retry-After or served without grants")
	}
	r.clock.Advance(r.eng.nextStepDelay(10 * time.Second))
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 || r.eng.serving {
		t.Fatal("did not retry at Retry-After or served without grants")
	}
}

func TestD067_GrantlessReplayKeepsDeadline(t *testing.T) {
	start := time.Hour
	e := renewFixture(1, 1, start, start, time.Now().UTC(), time.Minute)
	e.enrolled = true
	e.lease.Members = nil
	e.cfg.HeartbeatSeconds = 10
	clock := e.clock.(*timeanchor.Manual)
	e.noteProvisionalSchedule(start)
	clock.Advance(10 * time.Second)
	e.noteProvisionalSchedule(clock.Mono()) // same lease id, original leaseMono
	clock.Advance(20 * time.Second)
	e.noteProvisionalSchedule(clock.Mono())
	if e.leaseMono != start || e.provisionalRetryAt != start+30*time.Second || !e.needRenew() {
		t.Fatal("replay moved the signed lease renewal lead")
	}
	e.leaseAttempted = true
	e.leaseAttemptAt = clock.Mono()
	if e.nextStepDelay(10*time.Second) != 10*time.Second {
		t.Fatal("replay of a lease past its lead caused a busy loop")
	}
	// A granted lease exits the grant-less schedule and keeps its usual lead.
	e.lease.Members = []api.MemberLease{{}}
	e.noteProvisionalSchedule(clock.Mono())
	if e.provisionalRetryAt != 0 || e.provisionalStreak != 0 {
		t.Fatal("granted lease retained the provisional schedule")
	}
}
