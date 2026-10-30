package agent

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func alignControlClock(r *rig) {
	r.srv.SetClock(func() time.Time { return r.clock.Wall() })
}

func restartEngine(t *testing.T, r *rig, plane Plane) {
	t.Helper()
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := Start(Options{
		Config: r.eng.cfg, Clock: r.clock, Plane: plane,
		MeterSource: staticSource{boot: r.eng.bootID.String()},
		Prober:      &verProbe{version: "0.76.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	r.eng = next
	if err := next.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func leaseCheck(t *testing.T, checks []api.NodeCheck) api.NodeCheck {
	t.Helper()
	c, ok := checkBy(checks, api.NodeCheckNameLease)
	if !ok {
		t.Fatal("missing lease check")
	}
	return c
}

func threeBindings(now time.Time) []api.DeviceBinding {
	out := make([]api.DeviceBinding, 3)
	for i := range out {
		b := oneBinding(now)
		b.DeviceId = uuid.MustParse(fmt.Sprintf("019a0000-0000-7000-8000-0000000000a%d", i+1))
		b.MemberId = uuid.MustParse(fmt.Sprintf("019a0000-0000-7000-8000-0000000000b%d", i+1))
		b.PeerId = fmt.Sprintf("peer-m%d", i+1)
		b.AllowedIpv4 = []string{fmt.Sprintf("100.64.1.%d/32", i+1)}
		out[i] = b
	}
	return out
}

func fixtureWithBindings(gen uint64, bindings []api.DeviceBinding) fakecontrol.Fixture {
	f := generationFixture(gen, bindings[0])
	f.Bindings = bindings
	return f
}

func TestD067_ProvisionalWaitBounds(t *testing.T) {
	e := &Engine{cfg: Config{HeartbeatSeconds: 10}}
	if e.provisionalWait(1) != 10*time.Second || e.provisionalWait(2) != 20*time.Second || e.provisionalWait(3) != 40*time.Second || e.provisionalWait(4) != 60*time.Second || e.provisionalWait(5) != 60*time.Second {
		t.Fatalf("10s heartbeat waits: %s %s %s %s %s", e.provisionalWait(1), e.provisionalWait(2), e.provisionalWait(3), e.provisionalWait(4), e.provisionalWait(5))
	}
	e.cfg.HeartbeatSeconds = 90
	if e.provisionalWait(1) != 90*time.Second || e.provisionalWait(2) != 60*time.Second {
		t.Fatalf("long heartbeat: first %s later %s", e.provisionalWait(1), e.provisionalWait(2))
	}
}

func TestD067_GrantlessRestartRenewsOnHeartbeatSchedule(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	if len(r.eng.lease.Members) == 0 {
		t.Fatal("setup lease has no grants")
	}
	gen := uint64(r.eng.desired.ConfigGeneration)
	r.srv.SetMemberGrantsReady(false)
	before := r.srv.LeaseRenewals()
	restartEngine(t, r, r.plane)
	if r.srv.LeaseRenewals() != before {
		t.Fatalf("boot renewed: %d -> %d", before, r.srv.LeaseRenewals())
	}
	stepOK(t, r)
	assertGrantless := func(streak int) {
		t.Helper()
		if r.eng.serving || uint64(r.eng.lease.ConfigGeneration) != gen || uint64(r.eng.desired.ConfigGeneration) != gen || len(r.eng.lease.Members) != 0 || r.eng.provisionalStreak != streak {
			t.Fatalf("grant-less: serving=%v gen lease=%s desired=%s members=%d streak=%d reasons=%v", r.eng.serving, r.eng.lease.ConfigGeneration, r.eng.desired.ConfigGeneration, len(r.eng.lease.Members), r.eng.provisionalStreak, r.eng.decision.Reasons)
		}
		if !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonLeaseExpired) || r.eng.decision.Safety != api.SafetyStatusPassed {
			t.Fatalf("reason/safety: %v %s", r.eng.decision.Reasons, r.eng.decision.Safety)
		}
		lease := leaseCheck(t, r.eng.decision.Checks)
		if lease.Status != api.HealthCheckStatusFailed || codeOf(t, lease) != api.ErrorCodeLEASEEXPIRED || detailValue(lease) != leaseGrantsPendingDetail {
			t.Fatalf("lease check: status=%s code=%s detail=%q", lease.Status, codeOf(t, lease), detailValue(lease))
		}
	}
	if r.srv.LeaseRenewals() != before+1 {
		t.Fatalf("restart renewed %d times", r.srv.LeaseRenewals()-before)
	}
	assertGrantless(1)

	stepOK(t, r)
	r.clock.Advance(9 * time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 {
		t.Fatalf("renewed inside the heartbeat: %d", r.srv.LeaseRenewals()-before)
	}
	r.clock.Advance(time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+2 {
		t.Fatalf("heartbeat slot renewed %d times", r.srv.LeaseRenewals()-before)
	}
	assertGrantless(2)

	r.srv.SetMemberGrantsReady(true)
	r.clock.Advance(19 * time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+2 || r.eng.serving {
		t.Fatalf("renewed before the doubled wait: count=%d serving=%v", r.srv.LeaseRenewals()-before, r.eng.serving)
	}
	r.clock.Advance(time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+3 || !r.eng.serving || len(r.eng.lease.Members) == 0 || uint64(r.eng.lease.ConfigGeneration) != gen || r.eng.appliedGen != gen || r.eng.provisionalStreak != 0 {
		t.Fatalf("grants did not restore this generation: renewals=%d serving=%v members=%d lease=%s applied=%d streak=%d reasons=%v", r.srv.LeaseRenewals()-before, r.eng.serving, len(r.eng.lease.Members), r.eng.lease.ConfigGeneration, r.eng.appliedGen, r.eng.provisionalStreak, r.eng.decision.Reasons)
	}
}

func TestD067_EmptyBindingsDoNotRenewProvisionally(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
	stepOK(t, r)
	if r.eng.serving || len(r.eng.lease.Members) != 0 || uint64(r.eng.lease.ConfigGeneration) != 3 || r.eng.provisionalStreak != 1 {
		t.Fatalf("provisional setup: serving=%v members=%d gen=%s streak=%d", r.eng.serving, len(r.eng.lease.Members), r.eng.lease.ConfigGeneration, r.eng.provisionalStreak)
	}
	empty := generationFixture(3, oneBinding(r.clock.Wall()))
	empty.Bindings = nil
	r.srv.SetFixture(empty)
	count := r.srv.LeaseRenewals()
	stepOK(t, r)
	if len(r.eng.desired.DeviceBindings) != 0 || r.eng.provisionalStreak != 0 || r.eng.provisionalRetryAt != 0 {
		t.Fatalf("schedule survived empty bindings: n=%d streak=%d retry=%s", len(r.eng.desired.DeviceBindings), r.eng.provisionalStreak, r.eng.provisionalRetryAt)
	}
	for range 3 {
		r.clock.Advance(30 * time.Second)
		stepOK(t, r)
	}
	if r.srv.LeaseRenewals() != count {
		t.Fatalf("empty bindings renewed %d times", r.srv.LeaseRenewals()-count)
	}
}

func TestD067_GrantedLeaseKeepsRenewalLead(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	count := r.srv.LeaseRenewals()
	r.clock.Advance(30 * time.Second)
	stepOK(t, r)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count || !r.eng.serving || len(r.eng.lease.Members) == 0 {
		t.Fatalf("granted lease renewed inside the lead: count=%d serving=%v members=%d", r.srv.LeaseRenewals()-count, r.eng.serving, len(r.eng.lease.Members))
	}
}

func TestD067_StaleGenerationThenEmptyMembershipConverges(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	bindings := threeBindings(r.clock.Wall())
	r.srv.SetFixture(fixtureWithBindings(12, bindings))
	r.srv.SetLeaseGeneration(8)
	stepOK(t, r)
	if r.eng.serving || uint64(r.eng.lease.ConfigGeneration) != 8 || r.eng.appliedGen != 12 || uint64(r.eng.desired.ConfigGeneration) != 12 {
		t.Fatalf("stale phase: serving=%v lease=%s applied=%d desired=%s reasons=%v", r.eng.serving, r.eng.lease.ConfigGeneration, r.eng.appliedGen, r.eng.desired.ConfigGeneration, r.eng.decision.Reasons)
	}
	if !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("stale reasons: %v", r.eng.decision.Reasons)
	}
	lease := leaseCheck(t, r.eng.decision.Checks)
	if lease.Status != api.HealthCheckStatusFailed || codeOf(t, lease) != api.ErrorCodeSTALEGENERATION {
		t.Fatalf("stale lease check: status=%s code=%s detail=%q", lease.Status, codeOf(t, lease), detailValue(lease))
	}

	r.srv.SetLeaseGeneration(0)
	r.srv.SetMemberGrantsReady(false)
	count := r.srv.LeaseRenewals()
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count+1 || r.eng.serving || uint64(r.eng.lease.ConfigGeneration) != 12 || len(r.eng.lease.Members) != 0 || r.eng.provisionalStreak != 1 {
		t.Fatalf("empty membership: renewals=%d serving=%v lease=%s members=%d streak=%d reasons=%v", r.srv.LeaseRenewals()-count, r.eng.serving, r.eng.lease.ConfigGeneration, len(r.eng.lease.Members), r.eng.provisionalStreak, r.eng.decision.Reasons)
	}
	if !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonLeaseExpired) || hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("empty-membership reasons: %v", r.eng.decision.Reasons)
	}
	lease = leaseCheck(t, r.eng.decision.Checks)
	if lease.Status != api.HealthCheckStatusFailed || codeOf(t, lease) != api.ErrorCodeLEASEEXPIRED || detailValue(lease) != leaseGrantsPendingDetail {
		t.Fatalf("grants detail: status=%s code=%s detail=%q", lease.Status, codeOf(t, lease), detailValue(lease))
	}
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count+1 {
		t.Fatalf("provisional storm: %d", r.srv.LeaseRenewals()-count)
	}

	r.srv.SetMemberGrantsReady(true)
	r.clock.Advance(10 * time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count+2 || !r.eng.serving || uint64(r.eng.desired.ConfigGeneration) != 12 || uint64(r.eng.lease.ConfigGeneration) != 12 || len(r.eng.lease.Members) != 3 || r.eng.appliedGen != 12 {
		t.Fatalf("did not converge on generation 12: renewals=%d serving=%v desired=%s lease=%s members=%d applied=%d reasons=%v", r.srv.LeaseRenewals()-count, r.eng.serving, r.eng.desired.ConfigGeneration, r.eng.lease.ConfigGeneration, len(r.eng.lease.Members), r.eng.appliedGen, r.eng.decision.Reasons)
	}
}

func TestParsePendingRevocationsLegacyAndKeyed(t *testing.T) {
	flat, err := parsePendingRevocations(`["100.64.1.1"]`)
	if err != nil || len(flat) != 1 || flat[0].Addr != netip.MustParseAddr("100.64.1.1") || flat[0].known() {
		t.Fatalf("legacy: %+v %v", flat, err)
	}
	device := uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	member := uuid.MustParse("019a0000-0000-7000-8000-00000000000a")
	raw := fmt.Sprintf(`[{"addr":"100.64.1.2","device_id":"%s","member_id":"%s"}]`, device, member)
	keyed, err := parsePendingRevocations(raw)
	if err != nil || len(keyed) != 1 || keyed[0].Addr != netip.MustParseAddr("100.64.1.2") || keyed[0].DeviceID != device || keyed[0].MemberID != member {
		t.Fatalf("keyed: %+v %v", keyed, err)
	}
	if got, err := parsePendingRevocations("null"); err != nil || got != nil {
		t.Fatalf("null: %+v %v", got, err)
	}
}

type lookPlane struct {
	*memPlane
	missing error
}

func (p *lookPlane) ConntrackLookPath() error { return p.missing }

func TestM3_MissingConntrackFailsClosedImmediately(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &lookPlane{memPlane: r.plane, missing: errors.New("TEST ONLY: conntrack missing")}
	r.eng.plane = p
	stepOK(t, r)
	if r.eng.serving || r.eng.conntrackAttempts != 0 || !r.eng.conntrackBlocked || r.eng.conntrackDetail != conntrackMissingDetail {
		t.Fatalf("missing binary: serving=%v attempts=%d blocked=%v detail=%q", r.eng.serving, r.eng.conntrackAttempts, r.eng.conntrackBlocked, r.eng.conntrackDetail)
	}
	if !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonSafetyCheckFailed) || r.eng.decision.Safety != api.SafetyStatusFailed {
		t.Fatalf("reasons=%v safety=%s", r.eng.decision.Reasons, r.eng.decision.Safety)
	}
	iso, ok := checkBy(r.eng.decision.Checks, api.NodeCheckNameInternetOnly)
	if !ok || iso.Status != api.HealthCheckStatusFailed || codeOf(t, iso) != api.ErrorCodeISOLATIONPOLICYINVALID || detailValue(iso) != conntrackMissingDetail {
		t.Fatalf("internet_only: %+v", iso)
	}
	stepOK(t, r)
	if r.eng.serving || r.eng.conntrackAttempts != 0 {
		t.Fatalf("second tick served or counted a missing binary: serving=%v attempts=%d", r.eng.serving, r.eng.conntrackAttempts)
	}
	p.missing = nil
	stepOK(t, r)
	if !r.eng.serving || r.eng.conntrackBlocked || r.eng.conntrackAttempts != 0 {
		t.Fatalf("binary returned before the bound and stayed closed: serving=%v blocked=%v attempts=%d reasons=%v", r.eng.serving, r.eng.conntrackBlocked, r.eng.conntrackAttempts, r.eng.decision.Reasons)
	}
}

type flushPlane struct {
	*memPlane
	failed []netip.Addr
	calls  int
}

func (p *flushPlane) FlushConntrack(_ context.Context, _ []netip.Addr) ([]netip.Addr, error) {
	p.calls++
	return append([]netip.Addr(nil), p.failed...), nil
}

func TestM2_FlushKeepsOnlyFailedAddresses(t *testing.T) {
	r := newRig(t)
	device := uuid.MustParse("019a0000-0000-7000-8000-0000000000a1")
	member := uuid.MustParse("019a0000-0000-7000-8000-00000000000a")
	other := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	r.eng.pendingRevoke = []addrID{
		{Addr: netip.MustParseAddr("100.64.1.1"), DeviceID: device, MemberID: member},
		{Addr: netip.MustParseAddr("100.64.1.2"), DeviceID: other, MemberID: member},
	}
	p := &flushPlane{memPlane: r.plane, failed: []netip.Addr{netip.MustParseAddr("100.64.1.2")}}
	r.eng.plane = p
	if r.eng.flushPendingConntrack(context.Background()) {
		t.Fatal("partial flush reported success")
	}
	if p.calls != 1 || len(r.eng.pendingRevoke) != 1 || r.eng.pendingRevoke[0].Addr != netip.MustParseAddr("100.64.1.2") || r.eng.pendingRevoke[0].DeviceID != other {
		t.Fatalf("kept %+v calls=%d", r.eng.pendingRevoke, p.calls)
	}
	p.failed = nil
	if !r.eng.flushPendingConntrack(context.Background()) || len(r.eng.pendingRevoke) != 0 {
		t.Fatalf("successful flush left %+v", r.eng.pendingRevoke)
	}
}

func TestM2_UnpublishedTransitionStillDeletesSameIP(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)
	if len(r.eng.addresses) != 1 || r.eng.addresses[0].Addr != netip.MustParseAddr("100.64.1.1") {
		t.Fatalf("tracked %+v", r.eng.addresses)
	}
	rows, err := r.eng.store.Addresses(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Addr != "100.64.1.1" || rows[0].DeviceID != "019a0000-0000-7000-8000-0000000000a1" || rows[0].MemberID != "019a0000-0000-7000-8000-00000000000a" {
		t.Fatalf("persisted identity %+v %v", rows, err)
	}
	r.srv.SetPublication(api.PublicationIntentUnpublished)
	stepOK(t, r)
	if r.eng.serving || len(r.eng.addresses) != 1 {
		t.Fatalf("unpublished cleared the tracked set: serving=%v %+v", r.eng.serving, r.eng.addresses)
	}
	bindingB := oneBinding(r.clock.Wall())
	bindingB.DeviceId = uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	bindingB.PeerId = "peer-b2"
	bindingB.AllowedIpv4 = []string{"100.64.1.1/32"}
	r.srv.SetFixture(generationFixture(4, bindingB))
	stepOK(t, r)
	last := p.models[len(p.models)-1]
	want := netip.MustParseAddr("100.64.1.1")
	found := false
	for _, addr := range last.Revoke {
		if addr == want {
			found = true
		}
	}
	deleted := false
	for _, line := range p.ex.deletes {
		if strings.Contains(line, want.String()) {
			deleted = true
		}
	}
	if !r.eng.serving || !found || !deleted {
		t.Fatalf("same IP after unpublish: serving=%v revoke=%v deletes=%v", r.eng.serving, last.Revoke, p.ex.deletes)
	}
}

func TestM3_ConntrackAttemptBoundSurvivesRestart(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)
	binding := oneBinding(r.clock.Wall())
	binding.AllowedIpv4 = []string{"100.64.1.2/32"}
	r.srv.SetFixture(generationFixture(3, binding))
	p.ex.failDelete = true
	for range maxConntrackDeleteAttempts - 1 {
		stepOK(t, r)
	}
	if !r.eng.serving || r.eng.conntrackAttempts != maxConntrackDeleteAttempts-1 || r.eng.conntrackBlocked {
		t.Fatalf("under the bound: serving=%v attempts=%d blocked=%v", r.eng.serving, r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	restartEngine(t, r, p)
	if r.eng.conntrackAttempts != maxConntrackDeleteAttempts-1 || r.eng.conntrackBlocked {
		t.Fatalf("restart reset the window: attempts=%d blocked=%v", r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	stepOK(t, r)
	if r.eng.serving || !r.eng.conntrackBlocked || r.eng.conntrackAttempts != maxConntrackDeleteAttempts || r.eng.conntrackDetail != conntrackBoundDetail {
		t.Fatalf("twelfth failure: serving=%v attempts=%d blocked=%v detail=%q", r.eng.serving, r.eng.conntrackAttempts, r.eng.conntrackBlocked, r.eng.conntrackDetail)
	}
	if !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonSafetyCheckFailed) || r.eng.decision.Safety != api.SafetyStatusFailed {
		t.Fatalf("reasons=%v safety=%s", r.eng.decision.Reasons, r.eng.decision.Safety)
	}
	iso, ok := checkBy(r.eng.decision.Checks, api.NodeCheckNameInternetOnly)
	if !ok || iso.Status != api.HealthCheckStatusFailed || codeOf(t, iso) != api.ErrorCodeISOLATIONPOLICYINVALID || detailValue(iso) != conntrackBoundDetail {
		t.Fatalf("internet_only: %+v", iso)
	}
}
