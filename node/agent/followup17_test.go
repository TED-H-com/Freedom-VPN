package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// failedSubsetPlane verifies a rendered serving apply and reports a
// conntrack failure for the revoke addresses marked in fail. The typed
// error is what enforce returns once those deletes fail; this plane does
// not run conntrack.
type failedSubsetPlane struct {
	*memPlane
	fail   map[netip.Addr]bool
	models []enforce.Model
	errs   []error
}

func (p *failedSubsetPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.memPlane.Apply(ctx, m, prev)
	p.models = append(p.models, m)
	if err != nil {
		p.errs = append(p.errs, err)
		return res, err
	}
	var failed []netip.Addr
	for _, addr := range m.Revoke {
		if p.fail[addr] {
			failed = append(failed, addr)
		}
	}
	res.Verified = true
	if len(failed) == 0 {
		p.errs = append(p.errs, nil)
		return res, nil
	}
	ct := &enforce.ConntrackDeleteError{Failed: append([]netip.Addr(nil), failed...), Pending: len(failed)}
	p.errs = append(p.errs, ct)
	return res, ct
}

// invalidRevokePlane returns the plain isolation CheckError for an invalid
// revoke address, with the serving result verified. That is the delete-loop
// result when the address never reaches conntrack -D. Render rejects the
// address before that loop; realApply records the error Apply returns today.
type invalidRevokePlane struct {
	*memPlane
	ex      revocationExecutor
	err     error
	real    enforce.ApplyResult
	realErr error
}

func (p *invalidRevokePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	copied := m
	copied.Revoke = []netip.Addr{{}}
	p.ex.model = m
	p.real, p.realErr = enforce.Apply(ctx, &p.ex, copied, prev)
	res, err := p.memPlane.Apply(ctx, m, prev)
	if err != nil {
		p.err = err
		return res, err
	}
	res.Verified = true
	p.err = &enforce.CheckError{Reason: enforce.ReasonIsolation, Detail: "revoke address is invalid"}
	return res, p.err
}

func serveTwoBindings(t *testing.T, r *rig) {
	t.Helper()
	alignControlClock(r)
	r.srv.StickyLease(false)
	r.eng.prober = &verProbe{version: "0.76.2"}
	enrollBoot(t, r)
	stepOK(t, r)
	r.srv.SetFixture(fixtureWithBindings(2, threeBindings(r.clock.Wall())[:2]))
	stepOK(t, r)
	if !r.eng.serving || len(r.eng.addresses) != 2 {
		t.Fatalf("setup: serving=%v addresses=%d reasons=%v errors=%v", r.eng.serving, len(r.eng.addresses), r.eng.decision.Reasons, r.eng.decision.Errors)
	}
}

func replaceWithThird(t *testing.T, r *rig) {
	t.Helper()
	r.srv.SetFixture(fixtureWithBindings(3, threeBindings(r.clock.Wall())[2:]))
}

func fu17ID(addr, device, member string) addrID {
	return addrID{
		Addr:     netip.MustParseAddr(addr),
		DeviceID: uuid.MustParse(device),
		MemberID: uuid.MustParse(member),
	}
}

func assertPendingEqual(t *testing.T, got []addrID, want ...addrID) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("pending %+v, want %+v", got, want)
	}
}

func persistedPending(t *testing.T, r *rig) []addrID {
	t.Helper()
	raw, ok, err := r.eng.store.Meta(context.Background(), pendingRevocationsKey)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		return nil
	}
	parsed, err := parsePendingRevocations(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func assertAppliedOnly(t *testing.T, r *rig, addr, device, member string) {
	t.Helper()
	rows, err := r.eng.store.Addresses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Addr != addr || rows[0].DeviceID != device || rows[0].MemberID != member {
		t.Fatalf("applied rows %+v", rows)
	}
}

func assertNoOverlayAddrs(t *testing.T, label, blob string) {
	t.Helper()
	for _, addr := range []string{"100.64.1.1", "100.64.1.2", "100.64.1.3"} {
		if strings.Contains(blob, addr) {
			t.Fatalf("%s contains %s", label, addr)
		}
	}
}

func assertPendingObservation(t *testing.T, r *rig, n int) {
	t.Helper()
	if !r.eng.serving || !r.eng.decision.Serve || r.eng.decision.Safety != api.SafetyStatusPassed || len(r.eng.decision.Reasons) != 0 {
		t.Fatalf("under the bound: serving=%v serve=%v safety=%s reasons=%v", r.eng.serving, r.eng.decision.Serve, r.eng.decision.Safety, r.eng.decision.Reasons)
	}
	if len(r.eng.decision.Errors) != 1 || r.eng.decision.Errors[0].Message != "conntrack delete pending; serving ruleset verified" || r.eng.decision.Errors[0].Stage != "apply" || r.eng.decision.Errors[0].Code != api.ErrorCode(enforce.ReasonIsolation) {
		t.Fatalf("decision errors %+v", r.eng.decision.Errors)
	}
	for _, check := range r.eng.decision.Checks {
		if check.Status != api.HealthCheckStatusPassed {
			t.Fatalf("check %+v", check)
		}
	}
	if len(r.eng.pendingRevoke) != n {
		t.Fatalf("pending count %d, want %d", len(r.eng.pendingRevoke), n)
	}
	rep, ok := r.srv.LastObservation()
	if !ok || !rep.MemberForwardingEnabled || rep.SafetyStatus != api.SafetyStatusPassed || len(rep.ForwardingBlockReasons) != 0 {
		t.Fatalf("observation forwarding=%v safety=%s reasons=%v ok=%v", rep.MemberForwardingEnabled, rep.SafetyStatus, rep.ForwardingBlockReasons, ok)
	}
	if len(rep.Errors) != 1 || rep.Errors[0].Message != "conntrack delete pending; serving ruleset verified" || rep.Errors[0].Stage != "apply" || rep.Errors[0].Code != api.ErrorCode(enforce.ReasonIsolation) {
		t.Fatalf("report errors %+v", rep.Errors)
	}
	for _, check := range rep.Checks {
		if check.Status != api.HealthCheckStatusPassed {
			t.Fatalf("report check %+v", check)
		}
	}
	raw, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlayAddrs(t, "report", string(raw))
	assertNoOverlayAddrs(t, "log", r.log.String())
	errs, err := json.Marshal(r.eng.decision.Errors)
	if err != nil {
		t.Fatal(err)
	}
	assertNoOverlayAddrs(t, "decision", string(errs))
}

// TestFollowup17_SecondFailedAddressIsTheOnlyRetry is case (a): a two-address
// revoke whose second delete fails persists only that address, retires the
// first address's rows, and passes exactly that address as Revoke next time.
func TestFollowup17_SecondFailedAddressIsTheOnlyRetry(t *testing.T) {
	r := newRig(t)
	serveTwoBindings(t, r)
	second := netip.MustParseAddr("100.64.1.2")
	p := &failedSubsetPlane{memPlane: r.plane, fail: map[netip.Addr]bool{second: true}}
	r.eng.plane = p
	replaceWithThird(t, r)
	stepOK(t, r)

	want := fu17ID("100.64.1.2", "019a0000-0000-7000-8000-0000000000a2", "019a0000-0000-7000-8000-0000000000b2")
	assertPendingEqual(t, r.eng.pendingRevoke, want)
	assertPendingEqual(t, persistedPending(t, r), want)
	assertAppliedOnly(t, r, "100.64.1.3", "019a0000-0000-7000-8000-0000000000a3", "019a0000-0000-7000-8000-0000000000b3")
	if r.eng.conntrackAttempts != 1 || r.eng.conntrackBlocked {
		t.Fatalf("attempts=%d blocked=%v", r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	last := len(p.errs) - 1
	var ct *enforce.ConntrackDeleteError
	if last < 0 || !errors.As(p.errs[last], &ct) || ct.Pending != 1 || len(ct.Failed) != 1 || ct.Failed[0] != second {
		t.Fatalf("apply error %+v pending=%d", p.errs, ct.Pending)
	}
	if len(r.eng.pendingRevoke) != len(ct.Failed) || len(r.eng.pendingRevoke) != ct.Pending {
		t.Fatalf("pending %d failed %d reported %d", len(r.eng.pendingRevoke), len(ct.Failed), ct.Pending)
	}
	if !slices.Equal(p.models[last].Revoke, []netip.Addr{netip.MustParseAddr("100.64.1.1"), second}) {
		t.Fatalf("first revoke %v", p.models[last].Revoke)
	}
	assertPendingObservation(t, r, 1)

	stepOK(t, r)
	got := p.models[len(p.models)-1].Revoke
	if !slices.Equal(got, []netip.Addr{second}) {
		t.Fatalf("retry revoke %v", got)
	}
	assertPendingEqual(t, r.eng.pendingRevoke, want)
	assertPendingEqual(t, persistedPending(t, r), want)
	assertAppliedOnly(t, r, "100.64.1.3", "019a0000-0000-7000-8000-0000000000a3", "019a0000-0000-7000-8000-0000000000b3")
	if r.eng.conntrackAttempts != 2 {
		t.Fatalf("attempts=%d", r.eng.conntrackAttempts)
	}
}

// TestFollowup17_PartialSuccessShrinksThePendingSet is case (b): both
// addresses fail, then one succeeds, and the pending set shrinks to the
// address that is still failing. The attempt count is not reset.
func TestFollowup17_PartialSuccessShrinksThePendingSet(t *testing.T) {
	r := newRig(t)
	serveTwoBindings(t, r)
	first := netip.MustParseAddr("100.64.1.1")
	second := netip.MustParseAddr("100.64.1.2")
	p := &failedSubsetPlane{memPlane: r.plane, fail: map[netip.Addr]bool{first: true, second: true}}
	r.eng.plane = p
	replaceWithThird(t, r)
	stepOK(t, r)
	both := []addrID{
		fu17ID("100.64.1.1", "019a0000-0000-7000-8000-0000000000a1", "019a0000-0000-7000-8000-0000000000b1"),
		fu17ID("100.64.1.2", "019a0000-0000-7000-8000-0000000000a2", "019a0000-0000-7000-8000-0000000000b2"),
	}
	assertPendingEqual(t, r.eng.pendingRevoke, both...)
	assertPendingEqual(t, persistedPending(t, r), both...)
	if r.eng.conntrackAttempts != 1 {
		t.Fatalf("attempts=%d", r.eng.conntrackAttempts)
	}

	p.fail = map[netip.Addr]bool{second: true}
	stepOK(t, r)
	want := fu17ID("100.64.1.2", "019a0000-0000-7000-8000-0000000000a2", "019a0000-0000-7000-8000-0000000000b2")
	assertPendingEqual(t, r.eng.pendingRevoke, want)
	assertPendingEqual(t, persistedPending(t, r), want)
	assertAppliedOnly(t, r, "100.64.1.3", "019a0000-0000-7000-8000-0000000000a3", "019a0000-0000-7000-8000-0000000000b3")
	if r.eng.conntrackAttempts != 2 || r.eng.conntrackBlocked {
		t.Fatalf("attempts=%d blocked=%v", r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	if !slices.Equal(p.models[len(p.models)-1].Revoke, []netip.Addr{first, second}) {
		t.Fatalf("shrink step still owed both addresses: %v", p.models[len(p.models)-1].Revoke)
	}
	assertPendingObservation(t, r, 1)
}

// TestFollowup17_FullSuccessClearsTheSetAndTheBound is case (c): once every
// pending delete succeeds the set is empty, the attempt count is reset, and
// a restart does not resurrect the revoked addresses.
func TestFollowup17_FullSuccessClearsTheSetAndTheBound(t *testing.T) {
	r := newRig(t)
	serveTwoBindings(t, r)
	first := netip.MustParseAddr("100.64.1.1")
	second := netip.MustParseAddr("100.64.1.2")
	p := &failedSubsetPlane{memPlane: r.plane, fail: map[netip.Addr]bool{first: true, second: true}}
	r.eng.plane = p
	replaceWithThird(t, r)
	stepOK(t, r)
	if r.eng.conntrackAttempts != 1 || len(r.eng.pendingRevoke) != 2 {
		t.Fatalf("setup attempts=%d pending=%d", r.eng.conntrackAttempts, len(r.eng.pendingRevoke))
	}

	p.fail = nil
	stepOK(t, r)
	if len(r.eng.pendingRevoke) != 0 || len(persistedPending(t, r)) != 0 || r.eng.conntrackAttempts != 0 || r.eng.conntrackBlocked {
		t.Fatalf("after success pending=%+v attempts=%d blocked=%v", r.eng.pendingRevoke, r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	if len(r.eng.decision.Errors) != 0 || !r.eng.serving {
		t.Fatalf("success left errors=%v serving=%v", r.eng.decision.Errors, r.eng.serving)
	}
	assertAppliedOnly(t, r, "100.64.1.3", "019a0000-0000-7000-8000-0000000000a3", "019a0000-0000-7000-8000-0000000000b3")

	restartEngine(t, r, p)
	if len(r.eng.pendingRevoke) != 0 || r.eng.conntrackAttempts != 0 || r.eng.conntrackBlocked {
		t.Fatalf("restart resurrected pending=%+v attempts=%d blocked=%v", r.eng.pendingRevoke, r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	before := len(p.models)
	stepOK(t, r)
	if len(p.models) != before+1 || len(p.models[before].Revoke) != 0 {
		t.Fatalf("post-restart revoke %v", p.models[before].Revoke)
	}
	if len(r.eng.pendingRevoke) != 0 || r.eng.conntrackAttempts != 0 {
		t.Fatalf("post-restart step pending=%+v attempts=%d", r.eng.pendingRevoke, r.eng.conntrackAttempts)
	}
}

// TestFollowup17_RestartRetriesOnlyThePersistedSet is case (d): the pending
// set saved mid-bound is the Revoke list of the first Apply after restart.
func TestFollowup17_RestartRetriesOnlyThePersistedSet(t *testing.T) {
	r := newRig(t)
	serveTwoBindings(t, r)
	second := netip.MustParseAddr("100.64.1.2")
	p := &failedSubsetPlane{memPlane: r.plane, fail: map[netip.Addr]bool{second: true}}
	r.eng.plane = p
	replaceWithThird(t, r)
	stepOK(t, r)
	want := fu17ID("100.64.1.2", "019a0000-0000-7000-8000-0000000000a2", "019a0000-0000-7000-8000-0000000000b2")
	assertPendingEqual(t, persistedPending(t, r), want)
	if r.eng.conntrackAttempts != 1 {
		t.Fatalf("attempts=%d", r.eng.conntrackAttempts)
	}

	restartEngine(t, r, p)
	assertPendingEqual(t, r.eng.pendingRevoke, want)
	if r.eng.conntrackAttempts != 1 || r.eng.conntrackBlocked {
		t.Fatalf("restart attempts=%d blocked=%v", r.eng.conntrackAttempts, r.eng.conntrackBlocked)
	}
	before := len(p.models)
	stepOK(t, r)
	if len(p.models) != before+1 || !slices.Equal(p.models[before].Revoke, []netip.Addr{second}) {
		t.Fatalf("first post-restart revoke %v", p.models[before].Revoke)
	}
	assertPendingEqual(t, r.eng.pendingRevoke, want)
	assertPendingEqual(t, persistedPending(t, r), want)
	if r.eng.conntrackAttempts != 2 {
		t.Fatalf("attempts=%d", r.eng.conntrackAttempts)
	}
}

// TestFollowup17_InvalidRevokeAddressIsNotAConntrackAttempt is case (e):
// an invalid revoke address is the plain CheckError and does not spend a
// conntrack bound attempt or write a pending set.
func TestFollowup17_InvalidRevokeAddressIsNotAConntrackAttempt(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &invalidRevokePlane{memPlane: r.plane}
	r.eng.plane = p
	binding := oneBinding(r.clock.Wall())
	binding.AllowedIpv4 = []string{"100.64.1.2/32"}
	r.srv.SetFixture(generationFixture(3, binding))
	baselines := r.plane.baselines
	stepOK(t, r)

	var ce *enforce.CheckError
	if p.err == nil || !errors.As(p.err, &ce) || ce.Detail != "revoke address is invalid" || enforce.IsConntrackDeleteFailure(p.err) {
		t.Fatalf("error %v", p.err)
	}
	var ct *enforce.ConntrackDeleteError
	if errors.As(p.err, &ct) {
		t.Fatal("invalid address returned ConntrackDeleteError")
	}
	if p.real.Verified || p.realErr == nil || enforce.IsConntrackDeleteFailure(p.realErr) || errors.As(p.realErr, &ct) {
		t.Fatalf("real apply verified=%v err=%v", p.real.Verified, p.realErr)
	}
	if len(p.ex.deletes) != 0 {
		t.Fatalf("conntrack ran: %v", p.ex.deletes)
	}
	if r.eng.serving || r.eng.decision.Serve || r.eng.conntrackAttempts != 0 || r.eng.conntrackBlocked || len(r.eng.pendingRevoke) != 0 || len(persistedPending(t, r)) != 0 {
		t.Fatalf("serving=%v attempts=%d blocked=%v pending=%+v", r.eng.serving, r.eng.conntrackAttempts, r.eng.conntrackBlocked, r.eng.pendingRevoke)
	}
	if r.plane.baselines <= baselines {
		t.Fatalf("baseline was not restored: %d -> %d", baselines, r.plane.baselines)
	}
	for _, err := range r.eng.decision.Errors {
		if err.Message == "conntrack delete pending; serving ruleset verified" {
			t.Fatal("invalid address took the pending-delete observation")
		}
	}
	assertNoOverlayAddrs(t, "log", r.log.String())
}
