package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

type failVerifiedPlane struct {
	*memPlane
	fail bool
}

func (p *failVerifiedPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.memPlane.Apply(ctx, m, prev)
	if err == nil && p.fail {
		return res, &enforce.CheckError{Reason: "DEPENDENCY_UNAVAILABLE", Detail: "TEST ONLY: conntrack missing after ruleset read-back"}
	}
	return res, err
}

func TestUPD05_ApplyFailureObservationCoherence(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	p := &failVerifiedPlane{memPlane: r.plane, fail: true}
	r.eng.plane = p
	renewals := r.srv.LeaseRenewals()
	stepOK(t, r)
	rep, ok := r.srv.LastObservation()
	if !ok || rep.MemberForwardingEnabled || r.eng.serving || r.eng.decision.Serve || !hasReason(rep.ForwardingBlockReasons, api.ForwardingBlockReasonSafetyCheckFailed) {
		t.Fatalf("incoherent apply failure: observation serving=%v reasons=%v decision serving=%v errors=%v", rep.MemberForwardingEnabled, rep.ForwardingBlockReasons, r.eng.decision.Serve, rep.Errors)
	}
	for _, name := range []api.NodeCheckName{api.NodeCheckNameSourceBinding, api.NodeCheckNameQos, api.NodeCheckNameLease} {
		check, ok := checkBy(rep.Checks, name)
		if !ok || check.Status != api.HealthCheckStatusFailed || codeOf(t, check) != api.ErrorCodeDEPENDENCYUNAVAILABLE {
			t.Fatalf("apply failure check %s: %+v", name, check)
		}
	}
	if len(r.plane.last.Bindings) != 0 || mustBeat(t, r).Reason == "serving" || r.srv.LeaseRenewals() != renewals {
		t.Fatal("apply failure restored members, claimed serving, or renewed an unserved generation")
	}
	// Retrying a verified serving apply is the only way to clear this failure.
	p.fail = false
	stepOK(t, r)
	if !r.eng.serving || !r.eng.decision.Serve || hasCode(r.eng.decision.Errors, api.ErrorCodeDEPENDENCYUNAVAILABLE) {
		t.Fatalf("successful retry did not clear apply failure: %+v", r.eng.decision)
	}
}

type preflightPlane struct {
	*memPlane
	missing []string
}

func (p preflightPlane) MissingExecutables() []string { return p.missing }

func TestUPD05_MissingExecutableFirstObservation(t *testing.T) {
	for _, name := range []string{"nft", "tc", "ip", "conntrack"} {
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			r.eng.plane = preflightPlane{memPlane: r.plane, missing: []string{name}}
			enrollBoot(t, r)
			stepOK(t, r)
			rep, ok := r.srv.LastObservation()
			if !ok || rep.MemberForwardingEnabled || r.eng.decision.Serve || r.plane.applies != 0 || !hasReason(rep.ForwardingBlockReasons, api.ForwardingBlockReasonSafetyCheckFailed) || !hasCode(rep.Errors, api.ErrorCodeDEPENDENCYUNAVAILABLE) {
				t.Fatalf("missing %s: serving=%v applies=%d reasons=%v errors=%v", name, rep.MemberForwardingEnabled, r.plane.applies, rep.ForwardingBlockReasons, rep.Errors)
			}
			if !strings.Contains(r.log.String(), "executable="+name) || mustBeat(t, r).Reason == "serving" {
				t.Fatalf("preflight log/heartbeat missing %s: %s", name, r.log.String())
			}
		})
	}
}

// Keep the lookup error deliberately sensitive; only the executable name
// may reach the agent log when this fixture is used by SystemPlane.
var errLookupFixture = errors.New("TEST ONLY: private PATH lookup detail")

type lookupExecutor struct {
	enforce.Executor
	missing string
	looked  []string
}

func (e *lookupExecutor) LookPath(name string) (string, error) {
	e.looked = append(e.looked, name)
	if name == e.missing {
		return "", errLookupFixture
	}
	return "/test-only/" + name, nil
}

func TestUPD05_EnforcementExecutorPreflight(t *testing.T) {
	for _, missing := range []string{"", "nft", "tc", "ip", "conntrack"} {
		ex := &lookupExecutor{missing: missing}
		got := (SystemPlane{Executor: ex}).MissingExecutables()
		if strings.Join(ex.looked, ",") != "nft,tc,ip,conntrack" || strings.Join(got, ",") != missing {
			t.Fatalf("lookup %q: called %v missing %v", missing, ex.looked, got)
		}
	}
}

func restartProbeRig(t *testing.T, r *rig) {
	t.Helper()
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: r.plane,
		MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: &verProbe{version: "0.76.2"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	r.eng = next
	if err := next.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestUPD05_ProbeIssuanceSurvivesRestart(t *testing.T) {
	for _, cleared := range []bool{false, true} {
		t.Run(fmt.Sprintf("cleared=%v", cleared), func(t *testing.T) {
			r := newRig(t)
			r.srv.SetPublication(api.PublicationIntentUnpublished)
			r.eng.prober = &verProbe{version: "0.76.2"}
			enrollBoot(t, r)
			stepOK(t, r)
			holdUpdateLock(t, r)
			ch := probeChallenge(t, r, labProbe(), 300*time.Second)
			clear, installed := r.eng.installProbeException(context.Background(), ch)
			if !installed {
				t.Fatal("initial exception not installed")
			}
			first := r.plane.last.Probe.TimeoutCeilSeconds
			raw, ok, err := r.eng.store.Meta(context.Background(), probeIssuanceKey)
			var saved probeIssuance
			if err != nil || !ok || json.Unmarshal([]byte(raw), &saved) != nil || (saved.Nonce != ch.Nonce || saved.Hash == "" || strings.Contains(raw, ch.Probe.SourceV4) || strings.Contains(raw, ch.Probe.Destinations[0].AddrV4)) {
				t.Fatalf("issuance not durable: ok=%v err=%v", ok, err)
			}
			if cleared {
				clear()
			}
			// A newly observed anchor can move trusted time slightly backward.
			// Its expiry calculation must not extend the stored original cap.
			anchor, found, err := r.eng.store.Anchor(context.Background())
			if err != nil || !found {
				t.Fatal("missing anchor", err)
			}
			anchor.Server = anchor.Server.Add(-time.Second)
			if err := r.eng.store.SaveAnchor(context.Background(), anchor); err != nil {
				t.Fatal(err)
			}
			r.clock.Advance(80 * time.Second)
			// A replacement process reads the durable nonce, not the old pointer.
			// Boot may answer the challenge and clear a re-capped exception.
			before := len(r.plane.calls)
			restartProbeRig(t, r)
			count := 0
			for _, call := range r.plane.calls[before:] {
				if call.model.Probe == nil {
					continue
				}
				count++
				cap := call.model.Probe.TimeoutCeilSeconds
				if cleared || call.kind != "baseline_model" || len(call.model.Bindings) != 0 || cap > first-80 || cap > uint64(ch.ExpiresAt.Sub(call.model.Now)/time.Second) {
					t.Fatalf("restart extended/reopened issuance: cleared=%v original=%d current=%d", cleared, first, cap)
				}
			}
			if !cleared && count != 1 {
				t.Fatalf("restart did not recap surviving issuance: installs=%d", count)
			}
			if r.plane.last.Probe != nil {
				t.Fatal("restart answer left an exception")
			}
			if _, installed := r.eng.installProbeException(context.Background(), ch); installed {
				t.Fatal("answered issuance was reopened")
			}
		})
	}
}

func TestUPD05_NoTrustedClockNoProbeOrAnswer(t *testing.T) {
	for _, anomaly := range []bool{false, true} {
		t.Run(fmt.Sprintf("anomaly=%v", anomaly), func(t *testing.T) {
			r := newRig(t)
			r.srv.SetPublication(api.PublicationIntentUnpublished)
			enrollBoot(t, r)
			stepOK(t, r)
			holdUpdateLock(t, r)
			ch := probeChallenge(t, r, labProbe(), 300*time.Second)
			if anomaly {
				r.clock.JumpWall(-80 * time.Second)
			} else {
				r.eng.anchor.Trusted = false
			}
			before := len(r.plane.calls)
			if _, installed := r.eng.installProbeException(context.Background(), ch); installed {
				t.Fatal("untrusted clock opened exception")
			}
			r.eng.answerHealth(context.Background())
			if _, err := os.Stat(attestPath(r)); !os.IsNotExist(err) {
				t.Fatalf("untrusted clock answered: %v", err)
			}
			if !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonClockUntrusted) {
				t.Fatalf("missing clock_untrusted: %v", r.eng.decision.Reasons)
			}
			for _, call := range r.plane.calls[before:] {
				if call.model.Probe != nil {
					t.Fatal("untrusted probe reached plane")
				}
			}
		})
	}
}

func TestUPD05_LeaseRetryAfterAcrossSteps(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	r.srv.SetLeaseRetryAfter(1, 30)
	r.srv.SetFixture(generationFixture(3, oneBinding(time.Now().UTC())))
	count := r.srv.LeaseRenewals()
	// The engine schedules the retry without blocking its heartbeat or
	// observations. The same step and later steps share this deadline.
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count+1 || r.eng.serving || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("429 state: count=%d serving=%v reasons=%v", r.srv.LeaseRenewals()-count, r.eng.serving, r.eng.decision.Reasons)
	}
	for range 3 {
		stepOK(t, r)
	}
	r.clock.Advance(29 * time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count+1 || r.eng.serving {
		t.Fatal("retried or served inside Retry-After")
	}
	rep, ok := r.srv.LastObservation()
	if !ok || rep.MemberForwardingEnabled || !hasReason(rep.ForwardingBlockReasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("backoff report: serving=%v reasons=%v", rep.MemberForwardingEnabled, rep.ForwardingBlockReasons)
	}
	r.clock.Advance(time.Second)
	stepOK(t, r)
	if r.srv.LeaseRenewals() != count+2 || !r.eng.serving {
		t.Fatalf("deadline did not release exactly one renewal: count=%d reasons=%v", r.srv.LeaseRenewals()-count, r.eng.decision.Reasons)
	}
}
