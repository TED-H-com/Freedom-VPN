package agent

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestReviewR14_GrantlessControlRenewsForeverAtCap is the D-067 schedule.
// A control that keeps returning grant-less same-generation leases makes the
// agent renew at one heartbeat, then double, then every 60s, while the node
// serves nothing. The operator signal is the lease check, not a separate state.
// Agent follow-up 16 (awaiting_member_grants, no LEASE_EXPIRED while the lease
// is in force) is not on this branch; this test records the detail that is.
func TestReviewR14_GrantlessControlRenewsForeverAtCap(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	// An in-force granted lease is not replaced just because the next
	// renewal would be grant-less. A restart loads that lease and the
	// first step renews it, which is how the provisional schedule starts.
	r.srv.SetMemberGrantsReady(false)
	before := r.srv.LeaseRenewals()
	restartEngine(t, r, r.plane)
	if r.srv.LeaseRenewals() != before {
		t.Fatalf("boot renewed: %d -> %d", before, r.srv.LeaseRenewals())
	}
	stepOK(t, r)
	if r.srv.LeaseRenewals() != before+1 || r.eng.serving || r.eng.provisionalStreak != 1 || len(r.eng.lease.Members) != 0 {
		t.Fatalf("first grant-less renewal: renewals=%d serving=%v streak=%d members=%d", r.srv.LeaseRenewals()-before, r.eng.serving, r.eng.provisionalStreak, len(r.eng.lease.Members))
	}
	lease := leaseCheck(t, r.eng.decision.Checks)
	if lease.Status != api.HealthCheckStatusFailed || codeOf(t, lease) != api.ErrorCodeLEASEEXPIRED || detailValue(lease) != leaseGrantsPendingDetail {
		t.Fatalf("operator signal: status=%s code=%s detail=%q reasons=%v safety=%s", lease.Status, codeOf(t, lease), detailValue(lease), r.eng.decision.Reasons, r.eng.decision.Safety)
	}
	if r.eng.decision.Safety != api.SafetyStatusPassed || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonLeaseExpired) {
		t.Fatalf("safety/reason: %s %v", r.eng.decision.Safety, r.eng.decision.Reasons)
	}
	for i, gap := range []time.Duration{10 * time.Second, 20 * time.Second, 40 * time.Second, 60 * time.Second, 60 * time.Second, 60 * time.Second} {
		count := r.srv.LeaseRenewals()
		r.clock.Advance(gap - time.Second)
		stepOK(t, r)
		if r.srv.LeaseRenewals() != count || r.eng.serving {
			t.Fatalf("gap %d renewed early: count=%d serving=%v", i, r.srv.LeaseRenewals()-before, r.eng.serving)
		}
		r.clock.Advance(time.Second)
		stepOK(t, r)
		if r.srv.LeaseRenewals() != count+1 || r.eng.serving || len(r.eng.lease.Members) != 0 {
			t.Fatalf("gap %d (%s): renewals=%d serving=%v members=%d streak=%d", i, gap, r.srv.LeaseRenewals()-before, r.eng.serving, len(r.eng.lease.Members), r.eng.provisionalStreak)
		}
	}
}

// TestReviewR14_StaleDesiredDoesNotMakeNewGenerationProvisional checks a
// grant-less lease whose generation is not the desired generation. That is
// the stale-generation path, not the provisional doubling schedule. Empty
// desired bindings are not provisional either.
func TestReviewR14_StaleDesiredDoesNotMakeNewGenerationProvisional(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	if len(r.eng.desired.DeviceBindings) == 0 {
		t.Fatal("setup desired has no bindings")
	}
	gen := uint64(r.eng.desired.ConfigGeneration)
	// Desired stays on this generation, bindings included. The next lease
	// is grant-less and numbered gen+7. Entering the renew lead is what
	// makes the agent accept that lease; SetLeaseGeneration alone does not.
	r.srv.SetMemberGrantsReady(false)
	r.srv.SetLeaseGeneration(gen + 7)
	r.eng.cfg.RenewLeadSeconds = 3600
	window := r.eng.lease.ExpiresAt.Sub(r.eng.lease.IssuedAt)
	advance := window * 3 / 4
	r.clock.Advance(advance)
	count := r.srv.LeaseRenewals()
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count+1 || uint64(r.eng.lease.ConfigGeneration) != gen+7 || len(r.eng.lease.Members) != 0 || uint64(r.eng.desired.ConfigGeneration) != gen || len(r.eng.desired.DeviceBindings) == 0 {
		t.Fatalf("setup did not accept the mismatched grant-less lease: renewals=%d lease=%s members=%d desired=%s bindings=%d", r.srv.LeaseRenewals()-count, r.eng.lease.ConfigGeneration, len(r.eng.lease.Members), r.eng.desired.ConfigGeneration, len(r.eng.desired.DeviceBindings))
	}
	if r.eng.provisionalStreak != 0 || r.eng.provisionalRetryAt != 0 {
		t.Fatalf("stale desired bindings made the newer grant-less lease provisional: streak=%d retry=%s lease=%s desired=%s", r.eng.provisionalStreak, r.eng.provisionalRetryAt, r.eng.lease.ConfigGeneration, r.eng.desired.ConfigGeneration)
	}
	r.clock.Advance(60 * time.Second)
	stepOK(t, r)
	if r.eng.provisionalStreak != 0 || r.eng.provisionalRetryAt != 0 {
		t.Fatalf("later mismatch armed the provisional schedule: streak=%d retry=%s renewals=%d", r.eng.provisionalStreak, r.eng.provisionalRetryAt, r.srv.LeaseRenewals()-count)
	}

	empty := generationFixture(gen+7, oneBinding(r.clock.Wall()))
	empty.Bindings = nil
	r.srv.SetLeaseGeneration(0)
	r.srv.SetFixture(empty)
	stepOK(t, r)
	held := r.srv.LeaseRenewals()
	if len(r.eng.desired.DeviceBindings) != 0 || r.eng.provisionalStreak != 0 || r.eng.provisionalRetryAt != 0 {
		t.Fatalf("empty membership was provisional: bindings=%d streak=%d retry=%s", len(r.eng.desired.DeviceBindings), r.eng.provisionalStreak, r.eng.provisionalRetryAt)
	}
	r.clock.Advance(60 * time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != held {
		t.Fatalf("empty membership renewed %d times", r.srv.LeaseRenewals()-held)
	}
}

// TestReviewR14_ProvisionalTimerFollowsMonotonicClock checks the in-memory
// provisionalRetryAt deadline. A wall jump does not fire it. A monotonic
// advance does. A restart drops the deadline; the next step renews at once.
// Production Mono is CLOCK_MONOTONIC, which stops across suspend on Linux,
// the same clock as the lease deadline, so the two stay aligned. This process
// cannot suspend the host.
func TestReviewR14_ProvisionalTimerFollowsMonotonicClock(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.srv.SetMemberGrantsReady(false)
	restartEngine(t, r, r.plane)
	stepOK(t, r)
	armed := r.eng.provisionalRetryAt
	if armed == 0 || r.eng.provisionalStreak != 1 || !r.eng.anchorOK {
		t.Fatalf("timer not armed: retry=%s streak=%d anchor=%v", armed, r.eng.provisionalStreak, r.eng.anchorOK)
	}
	count := r.srv.LeaseRenewals()
	// Inside DefaultForwardSkew (60s). A larger jump is an anchor anomaly
	// and needRenew returns before it consults provisionalRetryAt.
	r.clock.JumpWall(20 * time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count || r.eng.provisionalRetryAt != armed || r.eng.provisionalStreak != 1 || !r.eng.anchorOK {
		t.Fatalf("wall jump fired the timer: renewals=%d retry=%s streak=%d anchor=%v", r.srv.LeaseRenewals()-count, r.eng.provisionalRetryAt, r.eng.provisionalStreak, r.eng.anchorOK)
	}
	monoBefore := r.clock.Mono()
	r.clock.Advance(10 * time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count+1 || r.eng.provisionalStreak != 2 || r.clock.Mono() <= monoBefore {
		t.Fatalf("monotonic advance did not renew: renewals=%d streak=%d", r.srv.LeaseRenewals()-count, r.eng.provisionalStreak)
	}

	restartEngine(t, r, r.plane)
	if r.eng.provisionalRetryAt != 0 || r.eng.provisionalStreak != 0 {
		t.Fatalf("restart kept the in-memory timer: retry=%s streak=%d", r.eng.provisionalRetryAt, r.eng.provisionalStreak)
	}
	held := r.srv.LeaseRenewals()
	stepOK(t, r)
	if r.srv.LeaseRenewals() != held+1 || r.eng.provisionalStreak != 1 {
		t.Fatalf("restart did not renew on the first step: renewals=%d streak=%d", r.srv.LeaseRenewals()-held, r.eng.provisionalStreak)
	}
}

// TestReviewR14_ConntrackBoundIsPerApplyAndSurvivesPartialSuccess drives twelve
// verified applies. Each apply deletes two addresses and the second delete
// fails. The counter advances once per apply, the successful address stays
// queued, a restart between 11 and 12 keeps the count, and apply 12 fail-closes
// the node. The report built here is the shape central must accept for D-070.
func TestReviewR14_ConntrackBoundIsPerApplyAndSurvivesPartialSuccess(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	p := &stickyPlane{memPlane: r.plane, ex: stickyExec{sticky: "100.64.1.2"}}
	r.eng.plane = p
	stepOK(t, r)
	if !r.eng.serving || len(p.models) != 1 || p.errors[0] != nil {
		t.Fatalf("fixture install: serving=%v err=%v", r.eng.serving, p.errors)
	}

	now := r.clock.Wall()
	keep := oneBinding(now)
	dropA := oneBinding(now)
	dropA.DeviceId = uuid.MustParse("019a0000-0000-7000-8000-0000000000a2")
	dropA.MemberId = uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	dropA.PeerId = "peer-a2"
	dropA.AllowedIpv4 = []string{"100.64.1.2/32"}
	dropB := oneBinding(now)
	dropB.DeviceId = uuid.MustParse("019a0000-0000-7000-8000-0000000000a3")
	dropB.MemberId = uuid.MustParse("019a0000-0000-7000-8000-0000000000b3")
	dropB.PeerId = "peer-a3"
	dropB.AllowedIpv4 = []string{"100.64.1.3/32"}
	r.srv.SetFixture(fixtureWithBindings(3, []api.DeviceBinding{keep, dropA, dropB}))
	stepOK(t, r)
	if !r.eng.serving || len(r.eng.addresses) != 3 || r.eng.conntrackAttempts != 0 {
		t.Fatalf("three addresses: serving=%v n=%d attempts=%d reasons=%v err=%v", r.eng.serving, len(r.eng.addresses), r.eng.conntrackAttempts, r.eng.decision.Reasons, p.errors)
	}

	r.srv.SetFixture(generationFixture(4, keep))
	for n := 1; n < maxConntrackDeleteAttempts; n++ {
		before := len(p.ex.deletes)
		stepOK(t, r)
		if !r.eng.serving || r.eng.conntrackAttempts != n || r.eng.conntrackBlocked {
			t.Fatalf("tick %d: serving=%v attempts=%d blocked=%v", n, r.eng.serving, r.eng.conntrackAttempts, r.eng.conntrackBlocked)
		}
		if r.eng.conntrackAttempts != n {
			t.Fatalf("tick %d advanced the counter by more than one apply", n)
		}
		sawDrop := false
		sawKeepFail := false
		for _, line := range p.ex.deletes[before:] {
			if strings.Contains(line, "100.64.1.3") {
				sawDrop = true
			}
			if strings.Contains(line, p.ex.sticky) {
				sawKeepFail = true
			}
		}
		if !sawDrop || !sawKeepFail {
			t.Fatalf("tick %d deletes=%v", n, p.ex.deletes[before:])
		}
		pending := map[string]bool{}
		for _, id := range r.eng.pendingRevoke {
			pending[id.Addr.String()] = true
		}
		if !pending["100.64.1.2"] || !pending["100.64.1.3"] {
			t.Fatalf("tick %d dropped a partially deleted identity: %+v", n, r.eng.pendingRevoke)
		}
	}

	restartEngine(t, r, p)
	if r.eng.conntrackAttempts != maxConntrackDeleteAttempts-1 || r.eng.conntrackBlocked || len(r.eng.pendingRevoke) != 2 {
		t.Fatalf("restart between 11 and 12: attempts=%d blocked=%v pending=%+v", r.eng.conntrackAttempts, r.eng.conntrackBlocked, r.eng.pendingRevoke)
	}
	stepOK(t, r)
	if r.eng.serving || !r.eng.conntrackBlocked || r.eng.conntrackAttempts != maxConntrackDeleteAttempts || r.eng.conntrackDetail != conntrackBoundDetail {
		t.Fatalf("twelfth apply: serving=%v attempts=%d blocked=%v detail=%q", r.eng.serving, r.eng.conntrackAttempts, r.eng.conntrackBlocked, r.eng.conntrackDetail)
	}
	iso, ok := checkBy(r.eng.decision.Checks, api.NodeCheckNameInternetOnly)
	if !ok || iso.Status != api.HealthCheckStatusFailed || codeOf(t, iso) != api.ErrorCodeISOLATIONPOLICYINVALID || detailValue(iso) != conntrackBoundDetail {
		t.Fatalf("internet_only: %+v", iso)
	}

	rep, err := r.eng.report(context.Background(), r.clock.Wall())
	if err != nil {
		t.Fatal(err)
	}
	if rep.MemberForwardingEnabled || rep.SafetyStatus != api.SafetyStatusFailed || rep.ProvisioningState != api.ProvisioningStateReady {
		t.Fatalf("fail-closed report: forwarding=%v safety=%s provisioning=%s", rep.MemberForwardingEnabled, rep.SafetyStatus, rep.ProvisioningState)
	}
	gotISO := false
	for _, e := range rep.Errors {
		if e.Code == api.ErrorCodeISOLATIONPOLICYINVALID && e.Stage == "apply" && e.Message == conntrackBoundDetail {
			gotISO = true
		}
	}
	if !gotISO {
		t.Fatalf("fail-closed errors: %+v", rep.Errors)
	}
	net, ok := checkBy(rep.Checks, api.NodeCheckNameInternetOnly)
	if !ok || net.Status != api.HealthCheckStatusFailed || codeOf(t, net) != api.ErrorCodeISOLATIONPOLICYINVALID || detailValue(net) != conntrackBoundDetail {
		t.Fatalf("reported internet_only: %+v", net)
	}
}

// stickyPlane is revocationPlane with one address whose conntrack -D fails.
// Every other delete succeeds. The attempt counter still moves once per apply.
type stickyPlane struct {
	*memPlane
	ex     stickyExec
	models []enforce.Model
	errors []error
}

func (p *stickyPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	p.ex.model = m
	p.models = append(p.models, m)
	res, err := enforce.Apply(ctx, &p.ex, m, prev)
	p.errors = append(p.errors, err)
	return res, err
}

func (p *stickyPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	return p.memPlane.Baseline(ctx, dp)
}

type stickyExec struct {
	renderedExecutor
	sticky  string
	tcGen   uint64
	tcReads int
}

func (ex *stickyExec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "conntrack" && strings.Contains(strings.Join(args, " "), ex.sticky) {
		ex.deletes = append(ex.deletes, strings.Join(args, " "))
		return nil, fmt.Errorf("TEST ONLY: delete failed")
	}
	// The shared tc fixture lists one member class. A three-member apply
	// has to show one fq_codel leaf and one HTB class per member or
	// VerifyQoS fails before conntrack -D runs.
	if name == "nft" && ex.installed && len(ex.model.Bindings) > 1 {
		// The accumulated script fold keeps class minors from the previous
		// generation's state. VerifyServing classifies from the model, so
		// the listing has to be a fresh render of this model.
		return ex.listingForModel()
	}
	if name == "tc" && len(args) > 1 && args[0] == "-j" && ex.installed && len(ex.model.QoS.Members) > 1 {
		// loadTC runs four reads before Render and reserves every leaf it
		// sees. Those reads stay parent-only so the minors stay the ones
		// VerifyServing computes. The later reads are VerifyQoS.
		if ex.tcGen != ex.model.Generation {
			ex.tcGen = ex.model.Generation
			ex.tcReads = 0
		}
		ex.tcReads++
		if ex.tcReads <= 4 {
			return ex.parentTC(args)
		}
		return ex.memberTC(args)
	}
	return ex.renderedExecutor.Output(ctx, name, args...)
}

func (ex *stickyExec) parentTC(args []string) ([]byte, error) {
	if args[1] == "qdisc" {
		return []byte(`[{"kind":"htb","handle":"1:","root":true,"options":{"default":"fffe"}}]`), nil
	}
	q := ex.model.QoS
	parent, host := q.Up, q.HostUp
	if args[len(args)-1] == ex.model.Datapath.MemberIngress {
		parent, host = q.Down, q.HostDown
	}
	return fmt.Appendf(nil, `[{"kind":"htb","handle":"1:1","options":{"rate":"%dbit","ceil":"%dbit"}},{"kind":"htb","handle":"1:fffe","options":{"rate":"%dbit","ceil":"%dbit"}}]`, parent.RateBPS, parent.CeilBPS, host.RateBPS, host.CeilBPS), nil
}

func (ex *stickyExec) listingForModel() ([]byte, error) {
	plan, err := enforce.Render(ex.model, enforce.State{}, enforce.TCState{})
	if err != nil {
		return nil, err
	}
	var table listedTable
	table.init()
	if err := table.applyScript(plan.NFT); err != nil {
		return nil, err
	}
	return table.listingJSON()
}

func (ex *stickyExec) memberTC(args []string) ([]byte, error) {
	plan, err := enforce.Render(ex.model, enforce.State{}, enforce.TCState{})
	if err != nil {
		return nil, err
	}
	if args[1] == "qdisc" {
		b := strings.Builder{}
		b.WriteString(`[{"kind":"htb","handle":"1:","root":true,"options":{"default":"fffe"}}`)
		for _, mem := range ex.model.QoS.Members {
			minor := plan.Assignments[mem.MemberID]
			fmt.Fprintf(&b, `,{"kind":"fq_codel","handle":"%x:","parent":"1:%x"}`, minor, minor)
		}
		b.WriteByte(']')
		return []byte(b.String()), nil
	}
	q := ex.model.QoS
	parent, host := q.Up, q.HostUp
	down := args[len(args)-1] == ex.model.Datapath.MemberIngress
	if down {
		parent, host = q.Down, q.HostDown
	}
	b := strings.Builder{}
	fmt.Fprintf(&b, `[{"kind":"htb","handle":"1:1","options":{"rate":"%dbit","ceil":"%dbit"}},{"kind":"htb","handle":"1:fffe","options":{"rate":"%dbit","ceil":"%dbit"}}`, parent.RateBPS, parent.CeilBPS, host.RateBPS, host.CeilBPS)
	for _, mem := range q.Members {
		minor := plan.Assignments[mem.MemberID]
		rate := mem.Up
		if down {
			rate = mem.Down
		}
		fmt.Fprintf(&b, `,{"kind":"htb","handle":"1:%x","options":{"rate":"%dbit","ceil":"%dbit"}}`, minor, rate.RateBPS, rate.CeilBPS)
	}
	b.WriteByte(']')
	return []byte(b.String()), nil
}

// TestReviewR14_NilUUIDMigrationShieldsReassignedAddress is the v2 row shape
// migrateAppliedAddressIdentity writes: both device_id and member_id nil.
// planRevokes will not queue that address while any device holds it, and the
// next successful apply replaces the row, so the delete is never retried.
func TestReviewR14_NilUUIDMigrationShieldsReassignedAddress(t *testing.T) {
	addr := netip.MustParseAddr("100.64.1.1")
	nextDevice := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	nextMember := uuid.MustParse("019a0000-0000-7000-8000-0000000000c2")
	legacy := []addrID{{Addr: addr}}
	rebound := []enforce.Binding{{
		DeviceID:  nextDevice,
		MemberID:  nextMember,
		AllowedV4: []netip.Prefix{netip.MustParsePrefix("100.64.1.1/32")},
	}}
	if got := planRevokes(nil, legacy, rebound); len(got) != 1 || got[0] != addr {
		t.Errorf("TODO(review): planRevokes did not queue conntrack -D for a nil-UUID address reassigned to device %s (revoke=%v)", nextDevice, got)
	}

	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	nilID := uuid.Nil.String()
	if err := r.eng.store.SaveAddresses(context.Background(), []agentstore.AppliedAddress{{
		Addr: addr.String(), DeviceID: nilID, MemberID: nilID,
	}}); err != nil {
		t.Fatal(err)
	}
	restartEngine(t, r, r.plane)
	if len(r.eng.addresses) != 1 || r.eng.addresses[0].Addr != addr || r.eng.addresses[0].known() {
		t.Fatalf("boot did not reload the migrated row: %+v", r.eng.addresses)
	}
	next := oneBinding(r.clock.Wall())
	next.DeviceId = nextDevice
	next.MemberId = nextMember
	next.PeerId = "peer-b2"
	next.AllowedIpv4 = []string{"100.64.1.1/32"}
	r.srv.SetFixture(generationFixture(4, next))
	stepOK(t, r)
	queued := false
	for _, a := range r.plane.last.Revoke {
		if a == addr {
			queued = true
		}
	}
	if !queued {
		t.Errorf("TODO(review): engine apply did not queue conntrack -D after the migrated address moved to device %s (revoke=%v serving=%v)", nextDevice, r.plane.last.Revoke, r.eng.serving)
	}
	rows, err := r.eng.store.Addresses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Addr == addr.String() && row.DeviceID == nilID && row.MemberID == nilID {
			return
		}
	}
	if !queued {
		t.Errorf("TODO(review): the nil-UUID row was replaced by the new identity without a conntrack delete (rows=%+v)", rows)
	}
}
