package agent

// Adversarial security review r21 proofs for agent-core follow-up 37
// (05b2c2f2..7b3d7bf8). Tests that fail against 7b3d7bf8 because of a real
// defect uncovered by r21 are marked with `// TODO(review r21): <ID>`.
// Passing tests prove the "What held" claims across Attack Questions 1-6.
// TEST ONLY fixtures; no test touches host networking or the kernel.

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

type r21MemberRev struct {
	member uuid.UUID
	rev    uint64
}

type r21LedgerSummary struct {
	byMemberRev map[r21MemberRev]uint64
	byRevision  map[uint64]uint64
	gaps        []api.UsageEvent
	complete    []api.UsageEvent
}

func r21ReadLedger(t *testing.T, cfg Config, node, inc uuid.UUID) r21LedgerSummary {
	t.Helper()
	st, err := usagestore.Open(cfg.UsageDBPath(), node, inc, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	events, err := st.Unacked(t.Context(), 0, 1000, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	out := r21LedgerSummary{
		byMemberRev: map[r21MemberRev]uint64{},
		byRevision:  map[uint64]uint64{},
	}
	for _, ev := range events {
		var p api.UsageEvent
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			t.Fatal(err)
		}
		switch p.Quality {
		case api.UsageQualityGap:
			out.gaps = append(out.gaps, p)
		case api.UsageQualityComplete, api.UsageQualityBoundaryEstimated:
			out.complete = append(out.complete, p)
		}
		for _, rec := range p.Records {
			rev := uint64(rec.BindingRevision)
			up := uint64(rec.UploadBytes)
			out.byRevision[rev] += up
			out.byMemberRev[r21MemberRev{member: rec.MemberId, rev: rev}] += up
		}
	}
	return out
}

// H-1: Follow-up 37 claims that a crash between plane.Apply and the
// retained_applied write in agent.db "records a gap, not a misattribution."
// That is false whenever no background sample committed in the milliseconds
// between e.plane.Apply (engine.go:1128) and the crash before
// e.store.SetMeta(ctx, "retained_applied", ...) (engine.go:1178).
//
// Process 1 serves generation 2 (DeviceID=D, MemberID=M1, BindingRevision=1)
// and samples 4096 B into usage.db. Generation 3 rebinds DeviceID=D to
// (MemberID=M2, BindingRevision=2). Process 1's plane.Apply installs
// generation 3 in the kernel (preserving counter u_<D>), and Process 1 crashes
// before retained_applied is updated in agent.db. While generation 3 is active
// in the kernel before Process 2's Boot installs Baseline, 4096 B are
// forwarded for Member M2 (BindingRevision=2), advancing u_<D> to 8192.
// Process 2 boots on the same boot, loads the stale retained_applied (M1,
// rev 1), refuses adoption, installs Baseline, and runs settleRetainedUsage.
// Because both head.Counters and retained_applied carry (M1, rev 1),
// settleRetainedUsage emits a UsageQualityComplete event charging Member M1
// (rev 1) for Member M2's (rev 2) 4096 B.
func TestR21_H1_CrashBetweenApplyAndRetainedAppliedMisattributesToPreviousMember(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}

	// Process 1 serves generation 2 for Member M1 (BindingRevision 1) and
	// samples 4096 B into usage.db.
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	b1 := r.eng.appliedModel.Bindings[0]
	member1 := b1.MemberID
	upName := meter.CounterName(b1.DeviceID, meter.DirUpload)

	r.eng.meter.Close()
	src1 := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		bindings: r.eng.meterBindings,
		counters: map[string]enforce.CounterValue{upName: {Packets: 1, Bytes: 4096}},
	}
	var err error
	r.eng.meter, err = meter.New(meter.Config{
		NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}

	// Generation 3 rebinds the same DeviceID to Member M2 (BindingRevision 2).
	member2 := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	b2 := oneBinding(r.clock.Wall())
	b2.MemberId = member2
	b2.BindingRevision = 2
	r.srv.SetFixture(generationFixture(3, b2))

	// Process 1 accepts desired generation 3 and runs plane.Apply
	// (engine.go:1128, installing generation 3 in the kernel), then crashes
	// before SetMeta("retained_applied") at engine.go:1178. While generation 3
	// is active in the kernel before Process 2's Boot installs Baseline,
	// 4096 B are forwarded for Member M2 (BindingRevision 2), advancing u_<D>
	// from 4096 to 8192.
	ctx := t.Context()
	if err := r.eng.syncDesired(ctx); err != nil || r.eng.desired.ConfigGeneration != 3 {
		t.Fatalf("precondition: syncDesired generation 3 failed: %v", err)
	}
	model := *r.eng.appliedModel
	model.Generation = 3
	model.Bindings = append([]enforce.Binding(nil), model.Bindings...)
	model.Bindings[0].MemberID = member2
	model.Bindings[0].BindingRevision = 2
	model.Leases = append([]enforce.MemberLease(nil), model.Leases...)
	model.Leases[0].MemberID = member2
	model.QoS.Members = append([]enforce.MemberQoS(nil), model.QoS.Members...)
	model.QoS.Members[0].MemberID = member2
	model.QoS.Assignments = nil
	if _, err := r.plane.Apply(ctx, model, r.eng.counters); err != nil {
		t.Fatal(err)
	}
	r.eng.setMeterBindings(model.Bindings)
	r20Crash(r.eng)

	// Process 2 boots on the same boot. Kernel counter u_<D> is at 8192 (4096 B
	// from M1 rev 1 + 4096 B from M2 rev 2).
	src2 := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		counters: map[string]enforce.CounterValue{upName: {Packets: 2, Bytes: 8192}},
	}
	next := r20Successor(t, r, r.eng.cfg, src2, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.serving {
		t.Fatal("expected Process 2 Boot to refuse adoption after unacknowledged generation 3 install")
	}

	ledger := r21ReadLedger(t, r.eng.cfg, next.nodeID, next.incID)
	m1Bytes := ledger.byMemberRev[r21MemberRev{member: member1, rev: 1}]
	// TODO(review r21): H-1 — settleRetainedUsage must not bill Member M1
	// (BindingRevision 1) for the 4096 B that the kernel counted under
	// generation 3 for Member M2 (BindingRevision 2).
	if m1Bytes != 4096 {
		t.Fatalf("H-1: Member M1 (rev 1) was billed %d bytes as UsageQualityComplete (want 4096; the extra %d bytes belonged to Member M2 rev 2 under generation 3)",
			m1Bytes, m1Bytes-4096)
	}
}

// H-2: Follow-up 37 claims that if settleRetainedUsage's sample fails (e.g.
// clock hold), "a rebinding Apply records an honest gap rather than a
// misattribution." That is false whenever head.Counters in usage.db does not
// yet carry the retained (MemberID=M1, BindingRevision=1) binding (because
// Process 1 opened the baseline epoch before Apply and crashed before its
// first post-Apply sample committed). When Process 2's settleRetainedUsage
// hits meter.ErrHold and the first Step runs a rebinding Apply for Member M2
// (BindingRevision=2), e.setMeterBindings overwrites e.meterSnap with (M2,
// rev 2) and the next Sample sees hadOld==false, billing Member M2 (rev 2)
// as UsageQualityComplete for Member M1's (rev 1) 4096 B.
func TestR21_H2_FailedSettlementThenRebindingApplyMisattributesToNewMember(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}

	// Process 1 serves generation 2 (Member M1, BindingRevision 1). Its
	// baseline epoch was opened in ensureUsage before Apply, and Process 1
	// crashes after forwarding 4096 B before the first post-Apply Sample
	// commits to usage.db.
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	b1 := r.eng.appliedModel.Bindings[0]
	member1 := b1.MemberID
	upName := meter.CounterName(b1.DeviceID, meter.DirUpload)

	// Re-create the exact state where ensureUsage opened the baseline epoch at
	// counter=0 before bindings were set, then 4096 B were forwarded under
	// (M1, rev 1) before Process 1 crashed without completing a post-Apply
	// sample. Follow-up 37's settleRetainedUsage hits ErrHold when LastSample
	// is 2s ahead of time.Now() (within the 10s backward-skew hold window).
	r.eng.meter.Close()
	_ = r.eng.usage.Close()
	_ = os.Remove(r.eng.cfg.UsageDBPath())
	st, err := usagestore.Open(r.eng.cfg.UsageDBPath(), r.eng.nodeID, r.eng.incID, usagestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	r.eng.usage = st
	r.eng.usageStatus = st
	r.eng.setMeterBindings(nil)
	initSrc := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		bindings: r.eng.meterBindings,
		counters: map[string]enforce.CounterValue{upName: {Packets: 0, Bytes: 0}},
	}
	m1, err := meter.New(meter.Config{
		NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: st, Source: initSrc,
		Now: func() time.Time { return time.Now().UTC().Add(2 * time.Second) },
	})
	if err != nil {
		t.Fatal(err)
	}
	// Take one quiet sample before Apply so head.LastSample is set to now+2s
	// while head.Counters[upName] has MemberID=uuid.Nil, BindingRevision=0.
	if err := m1.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	r.eng.meter = m1
	r.eng.setMeterBindings([]enforce.Binding{b1})
	r20Crash(r.eng)

	// Process 2 starts on the same boot with changed ExtraDeniedV4 so adoption
	// is refused. settleRetainedUsage calls Sample(), which returns ErrHold
	// because head.LastSample is 2s in the future (< 10s backward hold limit).
	cfg := r20ChangedSets(r)
	src2 := &r20DynamicSource{
		boot:     r.eng.bootID.String(),
		counters: map[string]enforce.CounterValue{upName: {Packets: 1, Bytes: 4096}},
	}
	next := r20Successor(t, r, cfg, src2, r.plane)
	r.eng = next
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.serving {
		t.Fatal("expected Boot to refuse adoption")
	}

	// First Step applies generation 3, which rebinds the same DeviceID to
	// Member M2 (BindingRevision 2). No traffic flows under generation 3 yet.
	member2 := uuid.MustParse("019a0000-0000-7000-8000-0000000000b2")
	b2 := oneBinding(r.clock.Wall())
	b2.MemberId = member2
	b2.BindingRevision = 2
	r.srv.SetFixture(generationFixture(3, b2))
	stepOK(t, r)
	r.manualUsage(t)

	// Once the 2s clock hold elapses, the next Sample runs.
	next.meter.Close()
	next.meter, err = meter.New(meter.Config{
		NodeID: next.nodeID, IncarnationID: next.incID, Store: next.usage, Source: src2,
		Now: func() time.Time { return time.Now().UTC().Add(5 * time.Second) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := next.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}

	ledger := r21ReadLedger(t, cfg, next.nodeID, next.incID)
	m2Bytes := ledger.byMemberRev[r21MemberRev{member: member2, rev: 2}]
	// TODO(review r21): H-2 — Member M2 (BindingRevision 2) must not be billed
	// as UsageQualityComplete for the 4096 B consumed by Member M1
	// (BindingRevision 1) before the refused adoption.
	if m2Bytes != 0 || ledger.byMemberRev[r21MemberRev{member: member1, rev: 1}] == 0 && len(ledger.gaps) == 0 {
		t.Fatalf("H-2: Member M2 (rev 2) was billed %d bytes as UsageQualityComplete for Member M1's (rev 1) traffic (byMemberRev=%v gaps=%d)",
			m2Bytes, ledger.byMemberRev, len(ledger.gaps))
	}
}

// M-1: After Boot refuses adoption and installs Baseline, member forwarding is
// stopped and the restart-gap delta (8192 B under BindingRevision 1) is settled.
// However, Boot leaves head.Counters[upName] in usage.db stamped with the
// retired BindingRevision=1 in an open epoch across Baseline (both when
// settleRetainedUsage samples an open epoch and when loadRuntime populates
// e.meterSnap before ensureUsage's planBaseline opens a new epoch after a
// prior FinalSample). When the first Step runs a rebinding Apply for
// generation 3 (BindingRevision=2) and packets flow under generation 3 before
// the first post-Apply sample, attribute() compares head.Counters (rev 1)
// against obs.Bindings (rev 2) and drops the post-Baseline generation 3
// traffic into a false mapping-conflict UsageQualityGap.
func TestR21_M1_RefusedAdoptionLeavesOpenRetainedBindingCausingFalseConflictOnRebind(t *testing.T) {
	for _, tc := range []string{"open_epoch_settled", "closed_epoch_reopened_by_load_runtime"} {
		t.Run(tc, func(t *testing.T) {
			r, src, upName := r20ServedPendingDelta(t)
			if tc == "closed_epoch_reopened_by_load_runtime" {
				// Close the epoch cleanly in usage.db (as after a prior handoff /
				// FinalSample) before Process 2 boots. Because loadRuntime calls
				// e.setMeterBindings(retained.Model.Bindings) (rev 1) before
				// ensureUsage opens the meter, planBaseline opens the new
				// baseline epoch with head.Counters[upName] pre-stamped with the
				// retired BindingRevision=1.
				st, err := usagestore.Open(r.eng.cfg.UsageDBPath(), r.eng.nodeID, r.eng.incID, usagestore.Options{})
				if err != nil {
					t.Fatal(err)
				}
				mClose, err := meter.New(meter.Config{
					NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: st, Source: src,
					Now: func() time.Time { return time.Now().UTC().Add(time.Second) },
				})
				if err != nil {
					_ = st.Close()
					t.Fatal(err)
				}
				if err := mClose.FinalSample(t.Context()); err != nil {
					mClose.Close()
					_ = st.Close()
					t.Fatal(err)
				}
				mClose.Close()
				_ = st.Close()
			}

			cfg := r20ChangedSets(r)
			next := r20Successor(t, r, cfg, src, r.plane)
			r.eng = next
			if err := next.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			if next.serving {
				t.Fatal("expected Boot to refuse same-boot adoption on changed ExtraDeniedV4")
			}

			// Generation 3 rotates the device's BindingRevision from 1 to 2.
			// As soon as plane.Apply installs generation 3 in the kernel (before the
			// asynchronous sampler's next tick), 1024 B of generation 3 traffic are
			// forwarded under BindingRevision 2 (advancing u_<D> from 8192 to 9216).
			b := oneBinding(r.clock.Wall())
			b.BindingRevision = 2
			r.srv.SetFixture(generationFixture(3, b))
			src.setCounter(upName, enforce.CounterValue{Packets: 3, Bytes: 9216})
			stepOK(t, r)
			r.manualUsage(t)

			// Advance wall time past head.LastSample so Sample() commits the interval.
			next.meter.Close()
			var err error
			next.meter, err = meter.New(meter.Config{
				NodeID: next.nodeID, IncarnationID: next.incID, Store: next.usage, Source: src,
				Now: func() time.Time { return time.Now().UTC().Add(5 * time.Second) },
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := next.meter.Sample(t.Context()); err != nil {
				t.Fatal(err)
			}

			u := r20ReadUsage(t, cfg, next.nodeID, next.incID)
			// TODO(review r21): M-1 — 8192 B belonged to revision 1 before Baseline
			// and 1024 B belonged to revision 2 after generation 3 Apply; the 1024 B
			// must be attributed to revision 2 without a mapping-conflict gap.
			if len(u.gaps) != 0 || u.byRevision[1] != 8192 || u.byRevision[2] != 1024 {
				t.Fatalf("M-1 (%s): post-Baseline traffic under generation 3 (rev 2) was dropped into a mapping-conflict gap: byRevision=%v gaps=%+v (want rev1=8192, rev2=1024, gaps=0)",
					tc, u.byRevision, u.gaps)
			}
		})
	}
}

// M-2: Boot under SIGTERM violates the cooperative stop contract in three
// engine-level ways (in addition to the cmd/fvpn-agent exit-code defect proved
// in node/agent/cmd/fvpn-agent/review_r21_test.go):
//  1. ensureUsage(ctx) unconditionally spawns `go up.Run(e.loopCtx)` even when
//     caller.Err() != nil (SIGTERM already delivered). Because e.loopCtx is
//     derived from context.Background() without caller's controlBudgetKey{},
//     up.Run immediately starts an uncanceled outbound HTTP POST
//     /api/v1/agent/usage-batches after SIGTERM.
//  2. e.recoverOnce(caller) at engine.go:449 passes the canceled caller context
//     instead of the bootLocal context to `fvpn-updater recover`, causing a
//     healthy updater binary to fail with context.Canceled and latch
//     e.recoverBlocked = true (heartbeat Reason="recovery_required").
//  3. Boot allocates two separate 10s bootLocal budgets back-to-back (load at
//     engine.go:440 and ctx at engine.go:463) plus Close's 3s budget (23s
//     total), exceeding the 13s cooperative stop budget and TimeoutStopSec=15s.
func r21ServerUsageEvents(r *rig) int {
	var total int
	for _, n := range r.srv.UsageReport().Nodes {
		total += n.Events
	}
	return total
}

func TestR21_M2_BootUnderSIGTERMUncanceledUploadAndRecoverBlocked(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	cfg := r20ChangedSets(r)
	// Configure a healthy UpdaterRecover executable (/bin/true) as in production.
	if _, err := os.Stat("/bin/true"); err == nil {
		cfg.UpdaterRecover = "/bin/true"
		cfg.UpdaterConfig = filepath.Join(r.dir, "updater.json")
	}

	// Re-enable usage uploads on fakecontrol and record any POST that arrives
	// after SIGTERM has already canceled caller.
	r.srv.SetUsageUnavailable(0)
	beforeBatches := r21ServerUsageEvents(r)

	next := r20Successor(t, r, cfg, src, r.plane)
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // SIGTERM already delivered before Boot

	if err := next.Boot(ctx); err != nil {
		t.Fatal(err)
	}
	// Give the background uploader goroutine spawned by ensureUsage a brief
	// moment before Close() cancels e.loopCtx.
	time.Sleep(50 * time.Millisecond)
	afterBatches := r21ServerUsageEvents(r)
	hb := mustBeat(t, r)

	// TODO(review r21): M-2 — Boot under SIGTERM must not start uncanceled
	// background usage-batch HTTP uploads after caller is canceled, and must
	// not fail recoverOnce on the canceled caller context (latching
	// recoverBlocked=true / Reason="recovery_required").
	if afterBatches > beforeBatches || next.recoverBlocked || hb.Reason == string(api.ForwardingBlockReasonRecoveryRequired) {
		t.Fatalf("M-2: Boot under SIGTERM sent %d uncanceled usage batch HTTP POST(s) after SIGTERM and latched recoverBlocked=%v heartbeat.Reason=%q",
			afterBatches-beforeBatches, next.recoverBlocked, hb.Reason)
	}
}

type r21FailMemberPathPlane struct {
	*memPlane
	failMemberPath bool
}

func (p *r21FailMemberPathPlane) MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error) {
	if p.failMemberPath {
		return MemberPath{}, errMemberPath
	}
	return p.memPlane.MemberPath(ctx, m, expected)
}

// L-1: When adoptStoredServing passes verifyServingModel (restart.go:97) and
// fails checkMemberPath (restart.go:100), rejectMemberPath sets both
// e.memberPathPending = true and e.applyError = member_path_unconfigured.
// Boot (engine.go:473-480) then clears e.applyError = nil after Baseline
// succeeds without clearing e.memberPathPending. On a non-serving node (e.g.
// unpublished generation 3), enforce's !d.Serve branch never clears
// e.memberPathPending, so e.beat() (engine.go:1348) writes
// Reason="member_path_unconfigured" on Boot and on every subsequent non-serving
// Step even though e.decision.Safety is ok and e.decision.Errors is empty.
func TestR21_L1_RefusedAdoptionMemberPathLeavesOrphanMemberPathPending(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}

	plane := &r21FailMemberPathPlane{memPlane: r.plane, failMemberPath: true}
	next, err := Start(Options{
		Config:      r.eng.cfg,
		Clock:       r.clock,
		Plane:       plane,
		MeterSource: staticSource{boot: r.eng.bootID.String()},
		Prober:      passProbe{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	r.eng = next

	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.serving {
		t.Fatal("expected Boot to refuse adoption when checkMemberPath fails")
	}

	// Transition to an unpublished generation 3.
	b := oneBinding(r.clock.Wall())
	f := generationFixture(3, b)
	f.Publication = api.PublicationIntentUnpublished
	r.srv.SetFixture(f)
	stepOK(t, r)

	hb := mustBeat(t, r)
	// TODO(review r21): L-1 — after Boot clears the refused adoption's
	// applyError and Step acknowledges unpublished generation 3 with Safety=passed,
	// e.memberPathPending must not remain latched and override heartbeat.Reason
	// with "member_path_unconfigured".
	if next.memberPathPending || hb.Reason == "member_path_unconfigured" {
		t.Fatalf("L-1: orphan memberPathPending=%v after Boot cleared applyError=%v; unpublished generation %d heartbeat.Reason=%q (decision.Reasons=%v, decision.Safety=%s)",
			next.memberPathPending, next.applyError, next.appliedGen, hb.Reason, next.decision.Reasons, next.decision.Safety)
	}
}

// What held (Attack Question 1):
//  1. The state exercised by TestR20_M3_RefusedAdoptionErrorDoesNotBlockNonServingAck
//     (a fresh serving heartbeat on disk while the live kernel ruleset is at
//     generation 0 / mismatched) is reachable in production without clock rewind
//     both via out-of-band ruleset replacement during the restart gap and via a
//     process crash in withdraw/handoff immediately after e.baseline replaces
//     the kernel ruleset and before e.beat() updates HeartbeatPath.
//  2. Both L-1 guards hold: sameRestartSets rejects non-nil empty DeniedV4 on
//     installed, candidate, or both; adoptStoredServing refuses when
//     installed.Generation != e.appliedGen even if desired/lease/heartbeat were
//     forged to match installed.Generation.
func TestR21_Held_Q1_OutOfBandRulesetReplacementReachableAndL1Guards(t *testing.T) {
	t.Run("crash_after_withdraw_baseline_before_beat_reaches_m3_state", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)

		// Simulate a process crash during withdraw/handoff right after
		// e.baseline(ctx) replaced the kernel ruleset with generation 0 and
		// before finishServingStep/shutdownUsage wrote the updated heartbeat.
		if err := r.eng.baseline(t.Context()); err != nil {
			t.Fatal(err)
		}
		r20Crash(r.eng)

		hbBefore := mustBeat(t, r)
		if !hbBefore.Serving || hbBefore.Generation != 2 || r.plane.last.Generation != 0 {
			t.Fatalf("precondition: expected Serving=true gen=2 heartbeat with baseline kernel (gen=0), got hb=%+v kernelGen=%d",
				hbBefore, r.plane.last.Generation)
		}

		// Successor boots on the same boot with a strict plane that rejects
		// VerifyServing because the kernel has generation 0 != 2.
		strict := &r20GenVerifyPlane{memPlane: r.plane}
		next, err := Start(Options{
			Config:      r.eng.cfg,
			Clock:       r.clock,
			Plane:       strict,
			MeterSource: staticSource{boot: r.eng.bootID.String()},
			Prober:      passProbe{},
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = next.Close() })
		r.eng = next

		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if next.serving || next.applyError != nil {
			t.Fatalf("expected refused adoption to install baseline and clear applyError, got serving=%v applyError=%+v",
				next.serving, next.applyError)
		}

		// Subsequent unpublished generation 3 is acknowledged cleanly.
		b := oneBinding(r.clock.Wall())
		f := generationFixture(3, b)
		f.Publication = api.PublicationIntentUnpublished
		r.srv.SetFixture(f)
		stepOK(t, r)
		if next.appliedGen != 3 || next.decision.Safety != api.SafetyStatusPassed {
			t.Fatalf("expected unpublished generation 3 ACK with Safety=passed, got gen=%d safety=%s",
				next.appliedGen, next.decision.Safety)
		}
	})

	t.Run("l1_guards_hold", func(t *testing.T) {
		valid := enforce.Model{
			DeniedV4:      enforce.SpecDeniedV4(),
			ExtraDeniedV4: []netip.Prefix{netip.MustParsePrefix("192.0.2.1/32")},
			HostV4:        []netip.Prefix{netip.MustParsePrefix("198.51.100.1/32")},
		}
		empty := valid
		empty.DeniedV4 = []netip.Prefix{}
		if sameRestartSets(empty, valid) || sameRestartSets(valid, empty) || sameRestartSets(empty, empty) {
			t.Fatal("sameRestartSets accepted non-nil empty DeniedV4")
		}
	})
}

// What held (Attack Question 2):
// When retained_applied matches the kernel generation installed by Process 1
// and settleRetainedUsage succeeds, repeated refused adoptions on the same boot
// attribute the pending delta exactly once (idempotent across crashes before
// Step), and a reboot closes the old epoch as power_loss without carrying
// same-named counter bytes across boots.
func TestR21_Held_Q2_DoubleRefusedAdoptionAndRebootIdempotency(t *testing.T) {
	r, src, upName := r20ServedPendingDelta(t)
	cfg := r20ChangedSets(r)

	p2 := r20Successor(t, r, cfg, src, r.plane)
	if err := p2.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u1 := r20ReadUsage(t, cfg, p2.nodeID, p2.incID)
	if len(u1.gaps) != 0 || u1.byRevision[1] != 8192 {
		t.Fatalf("after p2 Boot: got byRevision=%v gaps=%d, want rev1=8192 gaps=0", u1.byRevision, len(u1.gaps))
	}
	r20Crash(p2)

	// Process 3 boots on the same boot and refuses adoption again with the same
	// counter reading (8192); settleRetainedUsage sees a zero delta.
	p3 := r20Successor(t, r, cfg, src, r.plane)
	if err := p3.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u2 := r20ReadUsage(t, cfg, p3.nodeID, p3.incID)
	if len(u2.gaps) != 0 || u2.byRevision[1] != 8192 {
		t.Fatalf("after p3 Boot: got byRevision=%v gaps=%d, want rev1=8192 gaps=0", u2.byRevision, len(u2.gaps))
	}
	r20Crash(p3)

	// Process 4 boots after a host reboot (new boot_id) with a non-zero counter.
	newBoot := uuid.Must(uuid.NewV7()).String()
	if err := os.WriteFile(cfg.BootIDFile, []byte(newBoot+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	src.boot = newBoot
	src.setCounter(upName, enforce.CounterValue{Packets: 5, Bytes: 16384})
	p4 := r20Successor(t, r, cfg, src, r.plane)
	if err := p4.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u3 := r20ReadUsage(t, cfg, p4.nodeID, p4.incID)
	if u3.byRevision[1] != 8192 || len(u3.gaps) != 1 || u3.gaps[0].Gap == nil || u3.gaps[0].Gap.Reason != api.UsageGapReasonPowerLoss {
		t.Fatalf("after reboot Boot: got byRevision=%v gaps=%+v, want rev1=8192 and 1 power_loss gap", u3.byRevision, u3.gaps)
	}
}

// What held (Attack Question 3):
// When SIGTERM arrives during adoptStoredServing (waiting on nodelock, inside
// Prober.Probe, or inside ResolverDocument), adoptStoredServing refuses
// adoption, Boot installs the fail-closed Baseline under bootLocal(caller),
// settles retained usage, writes a non-serving heartbeat with BaselineOK=true,
// and skips answerHealthOnBoot(caller) even when a matching health challenge
// is present on disk.
func TestR21_Held_Q3_BootSIGTERMRefusesAdoptionAndSkipsHealthAnswer(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	cfg := r.eng.cfg

	// Write a live matching health challenge into JournalDir.
	if err := os.MkdirAll(cfg.JournalDir, 0o700); err != nil {
		t.Fatal(err)
	}
	chBytes, err := IssueHealthChallenge(
		uuid.Must(uuid.NewV7()).String(),
		cfg.ReleaseID,
		r.eng.nodeID.String(),
		r.eng.incID.String(),
		r.eng.bootID.String(),
		r.clock.Wall().Add(-time.Second),
		30*time.Second,
		cfg.AgentRelease,
		"0.0.0-test",
		"0.0.0-test",
		"1",
	)
	if err != nil {
		t.Fatal(err)
	}
	chPath := filepath.Join(cfg.JournalDir, healthattest.ChallengeFile)
	if err := os.WriteFile(chPath, chBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	prober := &r20CancelProber{cancel: cancel}

	next, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		MeterSource: src,
		Prober:      prober,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	src.setBindings(next.meterBindings)
	r.eng = next

	if err := next.Boot(ctx); err != nil {
		t.Fatal(err)
	}
	if next.serving || !next.baselineOK {
		t.Fatalf("expected canceled adoption to fall back to verified baseline, got serving=%v baselineOK=%v", next.serving, next.baselineOK)
	}
	hb := mustBeat(t, r)
	if hb.Serving || !hb.BaselineOK {
		t.Fatalf("expected heartbeat Serving=false BaselineOK=true, got %+v", hb)
	}
	// answerHealthOnBoot must not write health-attest.json when caller was canceled.
	if _, err := os.Stat(filepath.Join(cfg.JournalDir, healthattest.AttestationFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no health-attest.json when Boot caller was canceled, stat err=%v", err)
	}
	// Retained usage delta was still settled under bootLocal(caller).
	u := r20ReadUsage(t, cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || len(u.gaps) != 0 {
		t.Fatalf("expected 8192 settled bytes under bootLocal, got byRevision=%v gaps=%d", u.byRevision, len(u.gaps))
	}
}

// What held (Attack Question 4):
//  1. Clock-sample validation (NowMono < leaseMono, NowMono > now, Now !=
//     lease.IssuedAt + (NowMono - leaseMono)) and sameRestartSets run before
//     nodelock.Acquire, Prober.Probe, and ResolverDocument.
//  2. e.planeLocked prevents updateLockHeld() self-contention during
//     adoptStoredServing, prevents repairGeneration re-entrancy, allows
//     quarantine to execute without re-acquiring LockPath on
//     ReasonTCForeignFilter, and is always reset to false when adoption returns.
func TestR21_Held_Q4_AdoptionOrderingClockSampleSetChecksAndPlaneLocked(t *testing.T) {
	t.Run("tampered_now_and_nowmono_refused_before_lock_and_probe", func(t *testing.T) {
		for _, tc := range []string{"mono_before_lease", "mono_after_now", "wall_mismatch"} {
			t.Run(tc, func(t *testing.T) {
				r := newRig(t)
				alignControlClock(r)
				serveBoundGeneration(t, r)
				r.manualUsage(t)

				raw, ok, err := r.eng.store.Meta(t.Context(), "retained_applied")
				if err != nil || !ok {
					t.Fatalf("retained_applied meta: ok=%v err=%v", ok, err)
				}
				var retained retainedApplied
				if err := json.Unmarshal([]byte(raw), &retained); err != nil {
					t.Fatal(err)
				}
				switch tc {
				case "mono_before_lease":
					retained.Model.NowMono = r.eng.leaseMono - time.Millisecond
				case "mono_after_now":
					retained.Model.NowMono = r.clock.Mono() + time.Minute
				case "wall_mismatch":
					retained.Model.Now = retained.Model.Now.Add(time.Second)
				}
				patched, err := json.Marshal(retained)
				if err != nil {
					t.Fatal(err)
				}
				if err := r.eng.store.SetMeta(t.Context(), "retained_applied", string(patched)); err != nil {
					t.Fatal(err)
				}
				if err := r.eng.Close(); err != nil {
					t.Fatal(err)
				}

				// Hold the node lock externally so any attempt to acquire it
				// inside adoptStoredServing would block for ~2s and fail the
				// fast refusal check.
				lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
				if err != nil {
					t.Fatal(err)
				}
				defer lock.Release()

				prober := &r20CountingProber{}
				next, err := Start(Options{
					Config:      r.eng.cfg,
					Clock:       r.clock,
					Plane:       r.plane,
					MeterSource: staticSource{boot: r.eng.bootID.String()},
					Prober:      prober,
				})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = next.Close() })

				t0 := time.Now()
				if err := next.Boot(t.Context()); err != nil {
					t.Fatal(err)
				}
				if elapsed := time.Since(t0); elapsed > 500*time.Millisecond {
					t.Fatalf("expected early refusal before nodelock wait, took %v", elapsed)
				}
				if next.serving || prober.calls != 0 || next.planeLocked {
					t.Fatalf("expected serving=false prober.calls=0 planeLocked=false, got serving=%v calls=%d planeLocked=%v",
						next.serving, prober.calls, next.planeLocked)
				}
			})
		}
	})

	t.Run("foreign_tc_during_adoption_quarantines_under_held_lock_and_keeps_error", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		r.manualUsage(t)
		if err := r.eng.Close(); err != nil {
			t.Fatal(err)
		}

		qPlane := &r21ForeignTCPlane{memPlane: r.plane}
		next, err := Start(Options{
			Config:      r.eng.cfg,
			Clock:       r.clock,
			Plane:       qPlane,
			MeterSource: staticSource{boot: r.eng.bootID.String()},
			Prober:      passProbe{},
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = next.Close() })
		r.eng = next

		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if next.serving || !next.quarantinePending || next.applyError == nil || qPlane.quarantines.Load() != 1 || next.planeLocked {
			t.Fatalf("expected quarantine under held lock to keep applyError and clear planeLocked: serving=%v quarantinePending=%v applyError=%+v quarantines=%d planeLocked=%v",
				next.serving, next.quarantinePending, next.applyError, qPlane.quarantines.Load(), next.planeLocked)
		}
		// While quarantined, Boot must NOT clear applyError, so a subsequent
		// unpublished generation 3 is NOT acknowledged.
		b := oneBinding(r.clock.Wall())
		f := generationFixture(3, b)
		f.Publication = api.PublicationIntentUnpublished
		r.srv.SetFixture(f)
		stepOK(t, r)
		if next.appliedGen == 3 {
			t.Fatalf("expected quarantinePending applyError to block unpublished generation 3 ACK, got appliedGen=%d", next.appliedGen)
		}
	})
}

type r21ForeignTCPlane struct {
	*memPlane
	quarantines atomic.Int32
}

func (p *r21ForeignTCPlane) VerifyServing(_ context.Context, _ enforce.Model, _ map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	return enforce.ServingEvidence{}, &enforce.CheckError{
		Reason: enforce.ReasonTCForeignFilter,
		Detail: "foreign filter on root",
	}
}

func (p *r21ForeignTCPlane) Quarantine(ctx context.Context, model enforce.Model) (enforce.QuarantineEvidence, error) {
	p.quarantines.Add(1)
	return p.memPlane.Quarantine(ctx, model)
}

// What held (Attack Question 5):
//  1. When shutdownUsage takes !preserveServingOnStop (e.g. updater barrier is
//     active) and handoff's Baseline succeeds, shutdownUsage writes a
//     non-serving heartbeat (Serving=false, BaselineOK=true).
//  2. Conversely, if handoff's e.baseline(ctx) fails during shutdownUsage,
//     e.serving remains true so shutdownUsage does NOT overwrite the serving
//     heartbeat with a false Serving=false claim while the serving rules are
//     still in the kernel.
func TestR21_Held_Q5_QuarantineRetainsApplyErrorAndShutdownUsageHeartbeat(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)

	// Place an active updater barrier in JournalDir so preserveServingOnStop
	// returns false, and make Baseline fail during Close()'s handoff.
	if err := os.MkdirAll(r.eng.cfg.JournalDir, 0o700); err != nil {
		t.Fatal(err)
	}
	barrier := map[string]any{
		"version": 1,
		"state":   "closing_meter",
		"txn_id":  uuid.Must(uuid.NewV7()).String(),
	}
	raw, _ := json.Marshal(barrier)
	if err := os.WriteFile(filepath.Join(r.eng.cfg.JournalDir, "state.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}

	failPlane := &r21FailBaselinePlane{memPlane: r.plane}
	r.eng.plane = failPlane
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}

	// Because Baseline failed during handoff, the serving ruleset is still in
	// the kernel and the serving heartbeat must remain Serving=true.
	hb, err := watchdog.ReadHeartbeat(r.eng.cfg.HeartbeatPath)
	if err != nil {
		t.Fatal(err)
	}
	if !hb.Serving || hb.Generation != uint64(decstr.U64(2)) {
		t.Fatalf("expected failed handoff baseline to preserve Serving=true heartbeat for generation 2, got %+v", hb)
	}
}

type r21FailBaselinePlane struct {
	*memPlane
}

func (p *r21FailBaselinePlane) Baseline(context.Context, enforce.Datapath) error {
	return errors.New("simulated nft baseline failure")
}
