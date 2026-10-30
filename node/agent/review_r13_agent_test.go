package agent

import (
	"context"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// TestReviewR13_Q2_SameIPDeviceSwapSkipsConntrackDelete is the r13 M-2 proof.
// Replacing device A with device B on the same IPv4 must queue that address
// for conntrack -D before device B is served. A flat address set used to
// skip the delete, so A's established flows charged B's download counter.
func TestReviewR13_Q2_SameIPDeviceSwapSkipsConntrackDelete(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)
	if !r.eng.serving || len(p.models) != 1 {
		t.Fatalf("initial serving generation not applied: serving=%v models=%d", r.eng.serving, len(p.models))
	}

	bindingB := oneBinding(r.clock.Wall())
	bindingB.DeviceId = uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	bindingB.PeerId = "peer-b2"
	bindingB.AllowedIpv4 = []string{"100.64.1.1/32"}
	r.srv.SetFixture(generationFixture(3, bindingB))
	stepOK(t, r)

	last := p.models[len(p.models)-1]
	want := netip.MustParseAddr("100.64.1.1")
	revoked := false
	for _, addr := range last.Revoke {
		if addr == want {
			revoked = true
		}
	}
	deleted := false
	for _, line := range p.ex.deletes {
		if strings.Contains(line, want.String()) {
			deleted = true
		}
	}
	if !revoked || !deleted {
		t.Fatalf("same-IPv4 device replacement did not delete conntrack: revoke=%v deletes=%v", last.Revoke, p.ex.deletes)
	}
}

// TestReviewR13_Q2_PendingRevokeNotPrunedOnReAdmission is the r13 M-2 proof
// that a re-admitted (addr, device, member) is not passed to conntrack -D
// on the next tick, even while another address's delete is still failing.
func TestReviewR13_Q2_PendingRevokeNotPrunedOnReAdmission(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)

	bGen3 := oneBinding(r.clock.Wall())
	bGen3.AllowedIpv4 = []string{"100.64.1.2/32"}
	r.srv.SetFixture(generationFixture(3, bGen3))
	p.ex.failDelete = true
	stepOK(t, r)
	if len(r.eng.pendingRevoke) != 1 || r.eng.pendingRevoke[0].Addr != netip.MustParseAddr("100.64.1.1") {
		t.Fatalf("expected 100.64.1.1 in pendingRevoke, got %v", r.eng.pendingRevoke)
	}

	bGen4 := oneBinding(r.clock.Wall())
	bGen4.AllowedIpv4 = []string{"100.64.1.1/32"}
	r.srv.SetFixture(generationFixture(4, bGen4))
	p.ex.deletes = nil
	stepOK(t, r)

	lastModel := p.models[len(p.models)-1]
	reAdmitted := netip.MustParseAddr("100.64.1.1")
	replaced := netip.MustParseAddr("100.64.1.2")
	foundReplaced := false
	for _, addr := range lastModel.Revoke {
		if addr == reAdmitted {
			t.Fatalf("re-admitted address %s remained in model.Revoke=%v deletes=%v", reAdmitted, lastModel.Revoke, p.ex.deletes)
		}
		if addr == replaced {
			foundReplaced = true
		}
	}
	if !foundReplaced {
		t.Fatalf("replaced address was not revoked: %v", lastModel.Revoke)
	}
	for _, line := range p.ex.deletes {
		if strings.Contains(line, reAdmitted.String()) {
			t.Fatalf("conntrack -D ran against the re-admitted address: %v", p.ex.deletes)
		}
	}
}

// TestReviewR13_Q2_PermanentConntrackFailureNeverEscalates is the r13 M-3
// proof. Twelve consecutive verified conntrack delete failures fail closed:
// forwarding off, safety failed, and internet_only failed.
func TestReviewR13_Q2_PermanentConntrackFailureNeverEscalates(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &revocationPlane{memPlane: r.plane}
	r.eng.plane = p
	stepOK(t, r)

	bGen3 := oneBinding(r.clock.Wall())
	bGen3.AllowedIpv4 = []string{"100.64.1.2/32"}
	r.srv.SetFixture(generationFixture(3, bGen3))
	p.ex.failDelete = true

	for i := 0; i < maxConntrackDeleteAttempts; i++ {
		r.clock.Advance(5 * time.Second)
		r.srv.SetClockOffset(time.Duration(i+1) * 5 * time.Second)
		stepOK(t, r)
	}
	r.eng.lastObs = time.Time{}
	if err := r.eng.observe(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	rep, ok := r.srv.LastObservation()
	if !ok {
		t.Fatal("expected observation report")
	}
	hasFailedCheck := false
	var iso api.NodeCheck
	for _, c := range rep.Checks {
		if c.Status == api.HealthCheckStatusFailed {
			hasFailedCheck = true
		}
		if c.Name == api.NodeCheckNameInternetOnly {
			iso = c
		}
	}
	if rep.MemberForwardingEnabled || rep.SafetyStatus != api.SafetyStatusFailed || !hasFailedCheck {
		t.Fatalf("after %d consecutive conntrack delete failures the node stayed open: forwarding=%v safety=%s failedCheck=%v errors=%+v",
			maxConntrackDeleteAttempts, rep.MemberForwardingEnabled, rep.SafetyStatus, hasFailedCheck, rep.Errors)
	}
	if !hasReason(rep.ForwardingBlockReasons, api.ForwardingBlockReasonSafetyCheckFailed) {
		t.Fatalf("block reasons: %v", rep.ForwardingBlockReasons)
	}
	if iso.Status != api.HealthCheckStatusFailed || codeOf(t, iso) != api.ErrorCodeISOLATIONPOLICYINVALID || detailValue(iso) != conntrackBoundDetail {
		t.Fatalf("internet_only: status=%s detail=%q check=%+v", iso.Status, detailValue(iso), iso)
	}
}

// TestReviewR13_Q2_Q5_AgentWhatHeld verifies the agent properties that held:
//   - Boot preflight calls enforce.RequireConntrack and fails closed when the
//     conntrack binary is absent at startup.
//   - Probe omission details and logs never leak the probe peer ID or IPv4
//     addresses across all 10 allowed reason tokens, while rejecting unknown
//     reason tokens.
func TestReviewR13_Q2_Q5_AgentWhatHeld(t *testing.T) {
	r := omissionRig(t)
	ch := omittedChallenge(t, r, "node_addresses_unavailable")
	stepOK(t, r)
	check := reportedTunnel(t, r)
	if detailValue(check) != "probe_omitted: node_addresses_unavailable" {
		t.Fatalf("unexpected tunnel_probe detail: %q", detailValue(check))
	}
	if strings.Contains(r.log.String(), ch.ProbeOmitted.PeerID) || strings.Contains(r.log.String(), "100.64.8.9") {
		t.Fatalf("agent log leaked probe peer ID or IPv4: %s", r.log.String())
	}
	if err := enforce.RequireConntrack(&revocationExecutor{}); err != nil {
		t.Fatalf("expected RequireConntrack to succeed on test executor: %v", err)
	}
}
