package agent

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// servePair reaches a verified generation-3 model with two member grants.
// The second binding is the grant later documents remove.
func servePair(t *testing.T, r *rig) (keep, drop api.DeviceBinding) {
	t.Helper()
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	both := threeBindings(r.clock.Wall())[:2]
	r.srv.SetFixture(fixtureWithBindings(3, both))
	stepOK(t, r)
	if !r.eng.serving || r.eng.appliedGen != 3 || len(r.plane.last.Bindings) != 2 {
		t.Fatalf("pair setup serving=%v applied=%d bindings=%d reasons=%v err=%v",
			r.eng.serving, r.eng.appliedGen, len(r.plane.last.Bindings), r.eng.decision.Reasons, r.eng.applyError)
	}
	return both[0], both[1]
}

func modelHasPrefix(m enforce.Model, prefix string) bool {
	want := netip.MustParsePrefix(prefix)
	for _, b := range m.Bindings {
		for _, p := range b.AllowedV4 {
			if p == want {
				return true
			}
		}
	}
	return false
}

func callsWithPrefix(calls []planeCall, prefix string) int {
	n := 0
	for _, c := range calls {
		if c.kind == "apply" && modelHasPrefix(c.model, prefix) {
			n++
		}
	}
	return n
}

func cacheStickyLease(t *testing.T, r *rig) uuid.UUID {
	t.Helper()
	before := r.srv.LeaseRenewals()
	r.eng.leaseRefreshPending = true
	r.eng.leaseAttempted = false
	r.srv.StickyLease(true)
	if err := r.eng.syncLease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.eng.lease == nil || r.srv.LeaseRenewals() == before {
		t.Fatal("sticky lease was not renewed")
	}
	return r.eng.lease.LeaseId
}

func assertClosedWithout(t *testing.T, r *rig, dropped string, mark int) {
	t.Helper()
	if r.eng.serving || modelHasPrefix(r.plane.last, dropped) || callsWithPrefix(r.plane.calls[mark:], dropped) != 0 {
		t.Fatalf("removed grant reachable: serving=%v last=%v applies_with_grant=%d reasons=%v",
			r.eng.serving, r.plane.last.Bindings, callsWithPrefix(r.plane.calls[mark:], dropped), r.eng.decision.Reasons)
	}
}

func TestReviewR18_Q1_OrdinaryStalePathDoesNotForwardRemovedGrant(t *testing.T) {
	r := newRig(t)
	_, drop := servePair(t, r)
	dropped := drop.AllowedIpv4[0]
	oldLease := r.eng.lease.LeaseId
	applies, baselines := r.plane.applies, r.plane.baselines
	mark := len(r.plane.calls)
	r.eng.leaseAttempted = false
	r.srv.SetFixture(fixtureWithBindings(4, threeBindings(r.clock.Wall())[:1]))
	if err := r.eng.syncDesired(context.Background()); err != nil {
		t.Fatal(err)
	}
	if r.eng.needRenew() {
		t.Fatal("desired ahead of the applied lease must not be renewal-due before enforce")
	}
	if r.eng.applyError != nil {
		t.Fatal("ordinary path retained an apply error")
	}
	r.eng.evaluate(context.Background())
	if !leaseGenerationStale(r.eng.input()) || !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("stale lease not classified policy_missing: reasons=%v", r.eng.decision.Reasons)
	}
	if err := r.eng.enforce(context.Background(), r.eng.decision); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || r.eng.appliedGen != 4 || r.eng.lease.LeaseId != oldLease ||
		r.plane.applies != applies || r.plane.baselines <= baselines || r.eng.applyError != nil {
		t.Fatalf("ordinary stale path applied a candidate or skipped the baseline: serving=%v applied=%d lease=%s applies=%d baselines=%d err=%v",
			r.eng.serving, r.eng.appliedGen, r.eng.lease.LeaseId, r.plane.applies, r.plane.baselines, r.eng.applyError)
	}
	assertClosedWithout(t, r, dropped, mark)
	if err := r.eng.renewAppliedLease(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.serving || uint64(r.eng.lease.ConfigGeneration) != 4 || r.eng.lease.LeaseId == oldLease ||
		modelHasPrefix(r.plane.last, dropped) || callsWithPrefix(r.plane.calls[mark:], dropped) != 0 {
		t.Fatalf("matching renewal forwarded the removed grant or did not serve: serving=%v leaseGen=%s idChanged=%v",
			r.eng.serving, r.eng.lease.ConfigGeneration, r.eng.lease.LeaseId != oldLease)
	}
}

func TestReviewR18_Q1_RecoveryCandidateDoesNotForwardRemovedGrant(t *testing.T) {
	r := newRig(t)
	keep, drop := servePair(t, r)
	dropped := drop.AllowedIpv4[0]
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
	if err := r.eng.syncDesired(context.Background()); err != nil {
		t.Fatal(err)
	}
	leaseGen := uint64(r.eng.lease.ConfigGeneration)
	r.eng.applyError = &api.NodeError{
		Code: api.ErrorCodeHEALTHCHECKFAILED, Message: "apply failed",
		ObservedAt: r.clock.Wall().UTC(), Stage: "apply",
	}
	mark := len(r.plane.calls)
	r.eng.evaluate(context.Background())
	if err := r.eng.enforce(context.Background(), r.eng.decision); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || uint64(r.eng.lease.ConfigGeneration) != leaseGen {
		t.Fatalf("recovery rewrote the stored lease or served: serving=%v leaseGen=%s want %d",
			r.eng.serving, r.eng.lease.ConfigGeneration, leaseGen)
	}
	assertClosedWithout(t, r, dropped, mark)
	// The generation rewrite is safe only on a cleared copy. Applying it to the
	// real lease would authorize a new device of a member the old lease still names.
	desired := *r.eng.desired
	extra := keep
	extra.DeviceId = uuid.MustParse("019a0000-0000-7000-8000-0000000000a9")
	extra.PeerId = "peer-a9"
	extra.AllowedIpv4 = []string{"100.64.9.9/32"}
	desired.DeviceBindings = append(append([]api.DeviceBinding{}, desired.DeviceBindings...), extra)
	lease := *r.eng.lease
	lease.ConfigGeneration = desired.ConfigGeneration
	in := r.eng.input()
	in.Desired, in.Lease, in.ApplyError = &desired, &lease, nil
	in.DesiredOK, in.LeaseFresh = true, true
	counter := Decide(in)
	if !counter.Serve || !modelHasPrefix(counter.Model, "100.64.9.9/32") {
		t.Fatal("counterfactual generation rewrite did not admit the new device; the hazard check is invalid")
	}
	if modelHasPrefix(r.plane.last, "100.64.9.9/32") {
		t.Fatal("engine installed the counterfactual device")
	}
}

func TestReviewR18_Q1_StaleWindowStretchStaysClosed(t *testing.T) {
	r := newRig(t)
	keep, drop := servePair(t, r)
	dropped := drop.AllowedIpv4[0]
	cached := cacheStickyLease(t, r)
	next := uint64(r.eng.desired.ConfigGeneration) + 1
	r.srv.SetFixture(fixtureWithBindings(next, []api.DeviceBinding{keep}))
	if err := r.eng.syncDesired(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !r.eng.serving || !modelHasPrefix(r.plane.last, dropped) {
		t.Fatal("setup left the window before the restart")
	}
	mark := len(r.plane.calls)
	restartEngine(t, r, r.plane)
	assertClosedWithout(t, r, dropped, mark)

	// Renewals keep returning the cached previous generation, which still names the removed member.
	for i := 0; i < 5; i++ {
		stepOK(t, r)
		if r.eng.lease == nil || r.eng.lease.LeaseId != cached {
			t.Fatalf("step %d did not keep the replayed lease", i)
		}
		assertClosedWithout(t, r, dropped, mark)
	}
	r.srv.FailLeases(5)
	for i := 0; i < 3; i++ {
		stepOK(t, r)
		assertClosedWithout(t, r, dropped, mark)
	}
	for _, gen := range []uint64{next + 1, next + 2, next + 3} {
		r.srv.SetFixture(fixtureWithBindings(gen, []api.DeviceBinding{keep}))
		stepOK(t, r)
		assertClosedWithout(t, r, dropped, mark)
	}
	r.clock.JumpWall(-time.Hour)
	stepOK(t, r)
	assertClosedWithout(t, r, dropped, mark)
	r.clock.JumpWall(30 * time.Minute)
	stepOK(t, r)
	assertClosedWithout(t, r, dropped, mark)
	r.clock.Set(r.clock.Wall(), r.clock.Mono()+2*time.Hour)
	stepOK(t, r)
	assertClosedWithout(t, r, dropped, mark)
}

func TestReviewR18_Q1_FailedApplyDoesNotRestorePreviousGeneration(t *testing.T) {
	r := newRig(t)
	keep, drop := servePair(t, r)
	dropped := drop.AllowedIpv4[0]
	p := &failApplyPlane{memPlane: r.plane, fail: 4}
	r.eng.plane = p
	mark := len(r.plane.calls)
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
	stepOK(t, r)
	stepOK(t, r)
	if p.fail > 2 {
		t.Fatalf("apply was not retried after the failed generation: remaining=%d", p.fail)
	}
	assertClosedWithout(t, r, dropped, mark)
	if len(r.plane.last.Bindings) != 0 {
		t.Fatalf("failed apply restored bindings: %+v", r.plane.last.Bindings)
	}
}

type failApplyPlane struct {
	*memPlane
	fail int
}

func (p *failApplyPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if p.fail > 0 {
		p.fail--
		return enforce.ApplyResult{}, &enforce.CheckError{Reason: enforce.ReasonIsolation}
	}
	return p.memPlane.Apply(ctx, m, prev)
}

func TestReviewR18_Q1_MemberPathReasonDoesNotInvertServing(t *testing.T) {
	r := newRig(t)
	keep, drop := servePair(t, r)
	dropped := drop.AllowedIpv4[0]
	r.eng.rejectMemberPath(context.Background())
	if r.eng.serving || !r.eng.memberPathPending || r.eng.quarantinePending {
		t.Fatalf("path rejection did not latch a closed pending path: serving=%v pending=%v quarantine=%v",
			r.eng.serving, r.eng.memberPathPending, r.eng.quarantinePending)
	}
	r.srv.SetLeaseGeneration(uint64(r.eng.lease.ConfigGeneration))
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
	mark := len(r.plane.calls)
	stepOK(t, r)
	beat := mustBeat(t, r)
	assertClosedWithout(t, r, dropped, mark)
	if beat.Serving || r.eng.serving {
		t.Fatalf("TODO(review): member_path_unconfigured left serving reported or forwarding open (beat=%+v serving=%v)", beat, r.eng.serving)
	}
	if beat.Reason == "member_path_unconfigured" && hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("TODO(review): heartbeat reason %q replaced %v while serving=%v and the installed model is the baseline; memberPathPending survives the stale !Serve branch (engine.go beat, memberpath.go rejectMemberPath)",
			beat.Reason, r.eng.decision.Reasons, beat.Serving)
	}
}

func TestReviewR18_Q2_UnverifiedDesiredAndExpiredLeaseStayClosed(t *testing.T) {
	t.Run("corrupt", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		gen, bindings := r.eng.appliedGen, len(r.plane.last.Bindings)
		r.srv.CorruptSignatures(true)
		r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
		stepOK(t, r)
		if !r.eng.serving || r.eng.appliedGen != gen || len(r.plane.last.Bindings) != bindings {
			t.Fatalf("unverified desired changed forwarding: serving=%v applied=%d bindings=%d",
				r.eng.serving, r.eng.appliedGen, len(r.plane.last.Bindings))
		}
	})
	t.Run("lease_expires_during_stale_window", func(t *testing.T) {
		r := newRig(t)
		keep, drop := servePair(t, r)
		dropped := drop.AllowedIpv4[0]
		cacheStickyLease(t, r)
		r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
		mark := len(r.plane.calls)
		r.clock.Set(r.clock.Wall(), r.clock.Mono()+2*time.Hour)
		stepOK(t, r)
		assertClosedWithout(t, r, dropped, mark)
	})
	t.Run("freeze", func(t *testing.T) {
		r := newRig(t)
		keep, drop := servePair(t, r)
		dropped := drop.AllowedIpv4[0]
		r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
		if err := r.eng.syncDesired(context.Background()); err != nil {
			t.Fatal(err)
		}
		mark := len(r.plane.calls)
		r.eng.freezeIdentity(context.Background())
		if r.eng.serving || modelHasPrefix(r.plane.last, dropped) {
			t.Fatal("freeze during the stale window left the removed grant installed")
		}
		stepOK(t, r)
		assertClosedWithout(t, r, dropped, mark)
	})
	t.Run("new_device_not_admitted_from_desired_alone", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		b := oneBinding(r.clock.Wall())
		extra := b
		extra.DeviceId = uuid.MustParse("019a0000-0000-7000-8000-0000000000a9")
		extra.PeerId = "peer-a9"
		extra.AllowedIpv4 = []string{"100.64.9.9/32"}
		r.srv.SetLeaseGeneration(2)
		r.srv.SetFixture(fixtureWithBindings(3, []api.DeviceBinding{b, extra}))
		mark := len(r.plane.calls)
		stepOK(t, r)
		assertClosedWithout(t, r, "100.64.9.9/32", mark)
		if r.eng.serving {
			t.Fatal("stale lease served a desired document the lease does not authorize")
		}
	})
}

// d133Gate is the pass/fail rule in the follow-up 31 note (lines 511–517):
// no later revocation cutoff, no new binding, no larger expiry bound, and
// stricter restrictive-transition percentiles. It does not compare QoS ceilings.
func d133Gate(base, cand struct {
	p50, p90, p99, max float64
	cutoffOK           bool
	newBindings        int
	expiry             time.Duration
}) bool {
	if !cand.cutoffOK || cand.newBindings != 0 || cand.expiry > base.expiry {
		return false
	}
	if cand.p50 > base.p50 || cand.p90 > base.p90 || cand.p99 > base.p99 || cand.max > base.max {
		return false
	}
	return cand.p90 < base.p90 && cand.p99 < base.p99
}

func TestReviewR18_Q2_AcceptanceGateMissesQoSCeiling(t *testing.T) {
	base := struct {
		p50, p90, p99, max float64
		cutoffOK           bool
		newBindings        int
		expiry             time.Duration
	}{100, 200, 300, 400, true, 0, time.Hour}
	looser := base
	looser.p50, looser.p90, looser.p99, looser.max = 80, 150, 200, 250
	if !d133Gate(base, looser) {
		t.Fatal("gate rejected a percentile improvement with the same bindings and expiry")
	}
	// Option (b) requires the stricter QoS ceiling (note line 411). The
	// candidate below raises only that ceiling. The written gate still accepts.
	const baseCeil, candCeil = 1_000_000, 5_000_000
	if candCeil <= baseCeil {
		t.Fatal("ceiling fixture did not loosen")
	}
	if d133Gate(base, looser) {
		t.Fatalf("TODO(review): D-133 acceptance at docs/node/followup31-generation-change-availability-2026-11-01.md:511-517 accepts a candidate whose only regression is a sibling ceil from %d to %d bps; percentiles improve and bindings, cutoff and expiry are unchanged, so both B legs would pass", baseCeil, candCeil)
	}
}

func TestReviewR18_Q3_ProductionConfigCannotEnableObservation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent.json")
	body := `{
		"state_dir": "` + filepath.Join(dir, "state") + `",
		"identity_dir": "` + filepath.Join(dir, "identity") + `",
		"journal_dir": "` + filepath.Join(dir, "journal") + `",
		"lock_path": "` + filepath.Join(dir, "update.lock") + `",
		"control_url": "https://control.example",
		"server_ca_file": "` + filepath.Join(dir, "ca.pem") + `",
		"trust_bundle": "` + filepath.Join(dir, "trust.json") + `",
		"datapath": {"member_ingress": "fv-fh", "uplink": "fv-ih", "snat": "192.0.2.37"},
		"agent_release": "0.1.0",
		"release_id": "019a0000-0000-7000-8000-0000000000c1",
		"host_reserve_bps": 1000000,
		"netbird_backend": "kernel"
	}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FVPN_TEST_MODE", "true")
	t.Setenv("FVPN_TEST_TIMING_PATH", "/tmp/fvpn-r18-should-not-apply.jsonl")
	t.Setenv("TEST_MODE", "true")
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	eng := &Engine{cfg: cfg}
	if cfg.TestMode || cfg.TestTimingPath != "" || eng.measurementEnabled() {
		t.Fatalf("production file or environment enabled observation: mode=%v path=%q", cfg.TestMode, cfg.TestTimingPath)
	}
	timed := strings.TrimRight(body, "}\n") + `, "test_timing_path": "/tmp/fvpn-r18.jsonl"}`
	if err := os.WriteFile(path, []byte(timed), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("test_timing_path without test_mode was accepted")
	}
	src, err := os.ReadFile("engine.go")
	if err != nil {
		t.Fatal(err)
	}
	gate := strings.Index(string(src), "if e.measurementEnabled()")
	call := strings.Index(string(src), "WithCommandLifecycleForTest")
	if gate < 0 || call < gate || call > gate+400 {
		t.Fatal("command lifecycle caller is not inside measurementEnabled")
	}
	if strings.Contains(string(src), "WithApplyCaptureForTest") {
		t.Fatal("agent attaches render capture")
	}
	inst, err := os.ReadFile(filepath.Join("..", "..", "packaging", "installer", "config.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(inst)
	if strings.Contains(text, "TestMode:") || strings.Contains(text, "TestTimingPath:") || strings.Contains(text, "test_timing_path") || strings.Contains(text, "test_mode") {
		t.Fatal("installer config writer names the test observation gate")
	}
}

func TestReviewR18_Q4_TornCheckpointFailsClosed(t *testing.T) {
	t.Run("graceful_restart_keeps_valid_lease", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		gen, lease := r.eng.appliedGen, r.eng.lease.LeaseId
		restartEngine(t, r, r.plane)
		if !r.eng.serving || r.eng.appliedGen != gen || r.eng.lease.LeaseId != lease || len(r.plane.last.Bindings) == 0 {
			t.Fatalf("valid checkpoint was not adopted: serving=%v applied=%d lease=%s bindings=%d",
				r.eng.serving, r.eng.appliedGen, r.eng.lease.LeaseId, len(r.plane.last.Bindings))
		}
	})
	t.Run("partial_heartbeat", func(t *testing.T) {
		assertBootClosed(t, func(r *rig) {
			if err := os.WriteFile(r.eng.cfg.HeartbeatPath, []byte("v1\nboot_id "), 0o644); err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("stale_serving_heartbeat", func(t *testing.T) {
		assertBootClosed(t, func(r *rig) {
			err := watchdog.WriteHeartbeat(r.eng.cfg.HeartbeatPath, watchdog.Beat{
				BootID: r.eng.bootID, Mono: r.clock.Mono(), Wall: r.clock.Wall().UTC(),
				PolicyExpires: r.clock.Mono() + time.Hour, BaselineOK: true, Serving: true,
				Generation: 1, Reason: "serving",
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	})
	t.Run("torn_database", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		if err := r.eng.Close(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(r.eng.cfg.DBPath(), []byte("not a sqlite database"), 0o600); err != nil {
			t.Fatal(err)
		}
		next, err := Start(Options{
			Config: r.eng.cfg, Clock: r.clock, Plane: r.plane,
			MeterSource: staticSource{boot: r.eng.bootID.String()},
			Prober:      &verProbe{version: "0.76.2"},
		})
		if err != nil {
			return // process did not start; it did not accept the torn lease
		}
		t.Cleanup(func() { _ = next.Close() })
		r.eng = next
		if err := next.Boot(context.Background()); err != nil && !next.serving {
			return
		}
		if next.serving || len(r.plane.last.Bindings) != 0 {
			t.Fatalf("TODO(review): torn agent.db was adopted serving=%v bindings=%d", next.serving, len(r.plane.last.Bindings))
		}
	})
}

func assertBootClosed(t *testing.T, breakStore func(*rig)) {
	t.Helper()
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	// Close rewrites a valid heartbeat. The crash artifact has to land after that.
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	breakStore(r)
	next, err := Start(Options{
		Config: r.eng.cfg, Clock: r.clock, Plane: r.plane,
		MeterSource: staticSource{boot: r.eng.bootID.String()},
		Prober:      &verProbe{version: "0.76.2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	if err := next.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if next.serving || len(r.plane.last.Bindings) != 0 {
		t.Fatalf("TODO(review): next start accepted a partial or stale checkpoint serving=%v bindings=%d", next.serving, len(r.plane.last.Bindings))
	}
}

type sigtermObsPlane struct {
	*memPlane
	cancel   context.CancelFunc
	verifies int
}

func (p *sigtermObsPlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	p.verifies++
	if p.verifies == 1 {
		p.cancel()
		return enforce.ServingEvidence{}, nil
	}
	return enforce.ServingEvidence{}, &enforce.CheckError{Reason: enforce.ReasonIsolation}
}

func TestReviewR18_Q4_ControlRequestAfterSIGTERM(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &sigtermObsPlane{memPlane: r.plane, cancel: cancel}
	r.eng.plane = p
	before := r.srv.ObservationPosts()
	if err := r.eng.Step(parent); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if parent.Err() == nil || p.verifies < 2 {
		t.Fatalf("setup did not cancel across the shutdown read: err=%v verifies=%d", parent.Err(), p.verifies)
	}
	if r.eng.serving {
		t.Fatal("shutdown read left forwarding open")
	}
	after := r.srv.ObservationPosts()
	if after > before {
		rep, ok := r.srv.LastObservation()
		t.Fatalf("TODO(review): observation POST completed after the Step context was canceled (posts %d -> %d, report_ok=%v forwarding=%v). finishServingStep is deferred on the pre-budget context (engine.go:495) and observe/flush use controlContext, which falls back to that uncancellable context (serving.go:161-182, serving.go:232-236)",
			before, after, ok, rep.MemberForwardingEnabled)
	}
}

type budgetPlane struct {
	*memPlane
	q int
}

func (p *budgetPlane) VerifyServing(context.Context, enforce.Model, map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	return enforce.ServingEvidence{}, &enforce.CheckError{Reason: enforce.ReasonIsolation}
}

func (p *budgetPlane) Quarantine(ctx context.Context, model enforce.Model) (enforce.QuarantineEvidence, error) {
	p.q++
	<-ctx.Done()
	return enforce.QuarantineEvidence{}, ctx.Err()
}

func TestReviewR18_Q4_ShutdownExceedsLocalBound(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	if r.eng.appliedModel == nil {
		t.Fatal("no applied model to quarantine")
	}
	p := &budgetPlane{memPlane: r.plane}
	r.eng.plane = p
	r.eng.quarantinePending = true
	r.srv.SetDelay(2 * time.Second)
	unit, err := os.ReadFile(filepath.Join("..", "..", "deploy", "node", "systemd", "fvpn-agent.service"))
	if err != nil {
		t.Fatal(err)
	}
	timeoutStop := strings.Contains(string(unit), "TimeoutStopSec")
	bound := r.eng.controlPassTimeout() + 5*time.Second
	start := time.Now()
	if err := r.eng.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if p.q < 2 {
		t.Fatalf("quarantine budgets did not stack: %d", p.q)
	}
	if r.eng.serving {
		t.Fatal("stacked shutdown left forwarding open")
	}
	if elapsed > bound {
		t.Fatalf("TODO(review): Step spent %s after a serving-read failure with quarantinePending set; advertised local bound is %s (controlPassTimeout+5s). Quarantine ran %d times on independent 5s contexts (rejectServing then finishServingStep) and the following observation used another fresh 5s context. TimeoutStopSec set=%v. fvpn-agent.service has no TimeoutStopSec; systemd's default is 90s and this run did not approach it",
			elapsed, bound, p.q, timeoutStop)
	}
}

type shutdownRevokePlane struct {
	*memPlane
	cancel  context.CancelFunc
	sawLive bool
}

func (p *shutdownRevokePlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	if p.cancel != nil {
		p.sawLive = ctx.Err() == nil
		p.cancel()
		p.cancel = nil
	}
	return p.memPlane.Baseline(ctx, dp)
}

func TestReviewR18_Q4_RevocationDuringShutdownFailsClosed(t *testing.T) {
	r := newRig(t)
	keep, drop := servePair(t, r)
	dropped := drop.AllowedIpv4[0]
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := &shutdownRevokePlane{memPlane: r.plane, cancel: cancel}
	r.eng.plane = p
	r.eng.pendingRevoke = []addrID{{
		Addr: netip.MustParseAddr("100.64.1.2"), MemberID: drop.MemberId, DeviceID: drop.DeviceId,
	}}
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
	mark := len(r.plane.calls)
	_ = r.eng.Step(parent)
	if parent.Err() == nil || !p.sawLive {
		t.Fatalf("baseline did not run on the local context after cancel: parent=%v live=%v", parent.Err(), p.sawLive)
	}
	assertClosedWithout(t, r, dropped, mark)
	if r.eng.serving || modelHasPrefix(r.plane.last, dropped) {
		t.Fatalf("pending revocation ended open: serving=%v pending=%d", r.eng.serving, len(r.eng.pendingRevoke))
	}
}

type cancelSensitivePlane struct{ *memPlane }

func (p *cancelSensitivePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return enforce.ApplyResult{}, err
	}
	return p.memPlane.Apply(ctx, m, prev)
}

func TestReviewR18_Q4_CanceledApplyIsThePreFixWithdrawal(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.plane = &cancelSensitivePlane{r.plane}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.eng.enforce(ctx, r.eng.decision); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || r.eng.applyError == nil {
		t.Fatalf("canceled enforce context did not withdraw: serving=%v err=%v", r.eng.serving, r.eng.applyError)
	}

	r = newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	parent, stop := context.WithCancel(context.Background())
	defer stop()
	live := false
	r.eng.leaseRefreshPending = true
	r.eng.plane = cancelThenApply{memPlane: r.plane, cancel: stop, live: &live}
	if err := r.eng.Step(parent); err != nil && parent.Err() == nil {
		t.Fatal(err)
	}
	if parent.Err() == nil || !live || !r.eng.serving {
		t.Fatalf("current Step withdrew a live generation on caller cancel: parent=%v live=%v serving=%v err=%v",
			parent.Err(), live, r.eng.serving, r.eng.applyError)
	}
}

type cancelThenApply struct {
	*memPlane
	cancel context.CancelFunc
	live   *bool
}

func (p cancelThenApply) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	*p.live = ctx.Err() == nil
	if err := ctx.Err(); err != nil {
		return enforce.ApplyResult{}, err
	}
	return p.memPlane.Apply(ctx, m, prev)
}

func TestReviewR18_Q5_BaselineHoldHasNoCapOrDistinctAlarm(t *testing.T) {
	r := newRig(t)
	keep, drop := servePair(t, r)
	dropped := drop.AllowedIpv4[0]
	cached := cacheStickyLease(t, r)
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
	mark := len(r.plane.calls)
	distinct := false
	for i := 0; i < 40; i++ {
		stepOK(t, r)
		assertClosedWithout(t, r, dropped, mark)
		if r.eng.lease == nil || r.eng.lease.LeaseId != cached {
			t.Fatalf("step %d left the replayed lease", i)
		}
		if !hasReason(r.eng.decision.Reasons, api.ForwardingBlockReasonPolicyMissing) ||
			r.eng.decision.Safety != api.SafetyStatusPassed ||
			r.eng.decision.Provisioning != api.ProvisioningStateReady {
			distinct = true
			break
		}
	}
	if !distinct {
		t.Fatalf("TODO(review): 40 stale-lease steps stayed on the node-wide baseline (applied=%d serving=%v) with block reason policy_missing, safety %s and provisioning %s. No cap, hysteresis or distinct alarm fired. DeadlineInstallBound is %s and does not cover this outage; the reason token is the same one Decide uses when the document is absent (decide.go leaseGenerationStale)",
			r.eng.appliedGen, r.eng.serving, r.eng.decision.Safety, r.eng.decision.Provisioning, enforce.DeadlineInstallBound)
	}
}
