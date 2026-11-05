package agent

// TEST ONLY adversarial review proof tests for follow-up 38 (r22).
// Every counter and ruleset belongs to the in-memory plane; no host networking
// is read or modified.
import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// r22 H-1: outgoing bytes commit before another identity can count. Exercise
// both absent and nil-bound heads, a stable neighbor, and live/restart reads.
func TestR22_H1_UnsettledGenerationTransitionAndHandoffMisattributeUsage(t *testing.T) {
	for _, headKind := range []string{"nil_bound", "absent"} {
		for _, restart := range []bool{false, true} {
			t.Run(fmt.Sprintf("head_%s_restart_%v", headKind, restart), func(t *testing.T) {
				r := newRig(t)
				alignControlClock(r)
				r.srv.SetUsageUnavailable(1000)
				r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
				serveBoundGeneration(t, r)
				r.manualUsage(t)

				m1 := r.eng.appliedModel.Bindings[0].MemberID
				m2 := uuid.Must(uuid.NewV7())
				devA := r.eng.appliedModel.Bindings[0].DeviceID
				devB := uuid.Must(uuid.NewV7())
				upA := meter.CounterName(devA, meter.DirUpload)
				upB := meter.CounterName(devB, meter.DirUpload)

				src := &r20DynamicSource{
					boot:     r.eng.bootID.String(),
					bindings: r.eng.meterBindings,
					counters: map[string]enforce.CounterValue{
						upA: {Packets: 1, Bytes: 1000},
						upB: {Packets: 0, Bytes: 0},
					},
				}
				if headKind == "absent" {
					delete(src.counters, upB)
				}
				r.eng.meter.Close()
				var err error
				r.eng.meter, err = meter.New(meter.Config{
					NodeID:        r.eng.nodeID,
					IncarnationID: r.eng.incID,
					Store:         r.eng.usage,
					Source:        src,
				})
				if err != nil {
					t.Fatal(err)
				}
				if err := r.eng.meter.Sample(t.Context()); err != nil {
					t.Fatal(err)
				}

				// Generation 3: Device A on (M1, rev 1), Device B added on (M1, rev 2).
				// Advance into the lease renewal lead window (or post-adoption refresh)
				// so syncLease at engine.go:713 renews in the same pass as syncDesired.
				r.clock.Advance(time.Second)
				r.eng.leaseRefreshPending = true
				alignControlClock(r)
				bindA := oneBinding(r.clock.Wall())
				bindA.MemberId = m1
				bindA.DeviceId = devA
				bindA.PeerId = "peer-a1"
				bindA.BindingRevision = 1
				bindA.AllowedIpv4 = []string{"100.64.1.1/32"}

				bindB := oneBinding(r.clock.Wall())
				bindB.MemberId = m1
				bindB.DeviceId = devB
				bindB.PeerId = "peer-b"
				bindB.BindingRevision = 2
				bindB.AllowedIpv4 = []string{"100.64.1.11/32"}

				f3 := generationFixture(3, bindA)
				f3.Bindings = []api.DeviceBinding{bindA, bindB}
				r.srv.SetFixture(f3)
				stepOK(t, r)
				if !r.eng.serving || r.eng.appliedGen != 3 {
					t.Fatalf("setup: expected gen 3 serving, got serving=%v gen=%d reasons=%v", r.eng.serving, r.eng.appliedGen, r.eng.decision.Reasons)
				}

				// Under Generation 3 (before the async poked sample of Gen 3 commits),
				// Device B forwards 4096 B for Member M1 (rev 2).
				src.setCounter(upA, enforce.CounterValue{Packets: 2, Bytes: 3000})
				src.setCounter(upB, enforce.CounterValue{Packets: 1, Bytes: 4096})

				// Generation 4 rebinds Device B to Member M2 (rev 3); Device A stays on (M1, rev 1).
				// Advance into the lease renewal lead window (or post-adoption refresh)
				// so syncLease at engine.go:713 renews to Gen 4 before evaluate/enforce.
				r.clock.Advance(time.Second)
				r.eng.leaseRefreshPending = true
				alignControlClock(r)
				bindA = oneBinding(r.clock.Wall())
				bindA.MemberId = m1
				bindA.DeviceId = devA
				bindA.PeerId = "peer-a1"
				bindA.BindingRevision = 1
				bindA.AllowedIpv4 = []string{"100.64.1.1/32"}

				bindBRebound := oneBinding(r.clock.Wall())
				bindBRebound.MemberId = m2
				bindBRebound.DeviceId = devB
				bindBRebound.PeerId = "peer-b"
				bindBRebound.BindingRevision = 3
				bindBRebound.AllowedIpv4 = []string{"100.64.1.11/32"}
				f4 := generationFixture(4, bindA)
				f4.Bindings = []api.DeviceBinding{bindA, bindBRebound}
				r.srv.SetFixture(f4)

				// Step applies Generation 4 and writes retained_applied = Gen 4 without
				// sampling Generation 3 first. Process 1 then crashes after the
				// retained_applied write, before the first poked sample of Gen 4 commits.
				stepOK(t, r)
				if !r.eng.serving || r.eng.appliedGen != 4 {
					t.Fatalf("setup: expected gen 4 serving, got serving=%v gen=%d reasons=%v", r.eng.serving, r.eng.appliedGen, r.eng.decision.Reasons)
				}
				var u r21LedgerSummary
				if restart {
					r20Crash(r.eng)
					r.clock.Advance(time.Second)
					cfg := r20ChangedSets(r)
					next := r20Successor(t, r, cfg, src, r.plane)
					if err := next.Boot(t.Context()); err != nil {
						t.Fatal(err)
					}
					u = r21ReadLedger(t, cfg, next.nodeID, next.incID)
				} else {
					if err := r.eng.meter.Sample(t.Context()); err != nil {
						t.Fatal(err)
					}
					u = r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
				}
				if u.byRevision[1] != 3000 || u.byRevision[2] != 4096 || len(u.gaps) != 0 {
					t.Fatalf("outgoing or unchanged device lost bytes: %v gaps=%d", u.byRevision, len(u.gaps))
				}
				// Regression r22: H-1 — 4096 B forwarded by Device B under Generation 3
				// (Member M1, rev 2) must NEVER be billed as UsageQualityComplete to
				// Member M2 (rev 3) after Generation 4 overwrote retained_applied before
				// a sample of Generation 3 committed.
				if gotM2 := u.byMemberRev[r21MemberRev{member: m2, rev: 3}]; gotM2 != 0 {
					t.Fatalf("H-1: Member M2 (rev 3) was billed %d B as UsageQualityComplete for traffic counted under Member M1 (rev 2) in Generation 3 (byMemberRev=%v byRevision=%v gaps=%d)",
						gotM2, u.byMemberRev, u.byRevision, len(u.gaps))
				}
			})

		}
	}

	t.Run("failed_handoff_finalsample_does_not_block_rebinding_apply_in_next_step", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		r.srv.SetUsageUnavailable(1000)
		serveBoundGeneration(t, r)
		r.manualUsage(t)

		m1 := r.eng.appliedModel.Bindings[0].MemberID
		m2 := uuid.Must(uuid.NewV7())
		devA := r.eng.appliedModel.Bindings[0].DeviceID
		devB := uuid.Must(uuid.NewV7())
		upA := meter.CounterName(devA, meter.DirUpload)
		upB := meter.CounterName(devB, meter.DirUpload)

		src := &r20DynamicSource{
			boot:     r.eng.bootID.String(),
			bindings: r.eng.meterBindings,
			counters: map[string]enforce.CounterValue{
				upA: {Packets: 1, Bytes: 1000},
				upB: {Packets: 0, Bytes: 0},
			},
		}
		r.clock.Advance(5 * time.Second)
		alignControlClock(r)
		r.eng.meter.Close()
		var err error
		r.eng.meter, err = meter.New(meter.Config{
			NodeID:        r.eng.nodeID,
			IncarnationID: r.eng.incID,
			Store:         r.eng.usage,
			Source:        src,
			Now:           r.clock.Wall,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}

		// Step 1: serve Generation 3 adding Device B bound to Member M1 (rev 2).
		bindA := oneBinding(r.clock.Wall())
		bindB := oneBinding(r.clock.Wall())
		bindB.MemberId = m1
		bindB.DeviceId = devB
		bindB.PeerId = "peer-b"
		bindB.BindingRevision = 2
		bindB.AllowedIpv4 = []string{"100.64.1.11/32"}
		f3 := generationFixture(3, bindA)
		f3.Bindings = []api.DeviceBinding{bindA, bindB}
		r.srv.SetFixture(f3)
		r.clock.Advance(time.Second)
		alignControlClock(r)
		stepOK(t, r)

		// Device B forwards 4096 B under Generation 3 (Member M1, rev 2).
		src.setCounter(upB, enforce.CounterValue{Packets: 1, Bytes: 4096})

		// Step 2: node is unpublished in Generation 4 at the same wall-clock
		// instant as head.LastSample (or under sub-interval NTP jitter), so
		// handoff's FinalSample returns meter.ErrHold after installing Baseline.
		fUnpub := generationFixture(4, bindA)
		fUnpub.Publication = api.PublicationIntentUnpublished
		r.srv.SetFixture(fUnpub)
		r.clock.JumpWall(-time.Second)
		stepOK(t, r)
		if r.eng.serving || r.eng.finalDone {
			t.Fatalf("precondition: expected handoff to leave serving=false finalDone=false on ErrHold, got serving=%v finalDone=%v",
				r.eng.serving, r.eng.finalDone)
		}

		// At this point, the kernel is on Baseline with 4096 B of unsettled
		// Generation 3 traffic for (M1, rev 2), and the epoch is still open.
		// Step 3 advances into the renewal lead window and publishes Generation 5
		// rebinding Device B to Member M2 (rev 3). Because retainedPending was
		// never set by handoff, Step 3 clears usagePersist in readMeter, skips
		// PrepareSampling (!finalDone), and runs a rebinding Apply.
		r.clock.Advance(59 * time.Minute)
		alignControlClock(r)
		bindA = oneBinding(r.clock.Wall())
		bindBRebound := oneBinding(r.clock.Wall())
		bindBRebound.MemberId = m2
		bindBRebound.DeviceId = devB
		bindBRebound.PeerId = "peer-b"
		bindBRebound.BindingRevision = 3
		bindBRebound.AllowedIpv4 = []string{"100.64.1.11/32"}
		f5 := generationFixture(5, bindA)
		f5.Bindings = []api.DeviceBinding{bindA, bindBRebound}
		r.srv.SetFixture(f5)
		stepOK(t, r)

		// Settle/sample the open epoch.
		r.clock.Advance(time.Second)
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}

		u := r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
		// Regression r22: H-1 — A rebinding Apply must never run while the
		// withdrawn generation's delta is unsettled; Member M2 (rev 3) must not
		// be billed for the 4096 B forwarded under Member M1 (rev 2) before
		// handoff installed Baseline.
		if gotM2 := u.byMemberRev[r21MemberRev{member: m2, rev: 3}]; gotM2 != 0 {
			t.Fatalf("H-1: failed handoff FinalSample allowed next Step to run rebinding Apply over unsettled delta, billing Member M2 (rev 3) %d B for Member M1's (rev 2) traffic (byMemberRev=%v gaps=%d)",
				gotM2, u.byMemberRev, len(u.gaps))
		}
	})
}

// r22 H-2: candidate generations are bounded by live kernel evidence and
// epoch-scoped pre-Apply intent. Converted crash points run the real enforce
// path; a removed device's last packets precede its removal (D-141).
func TestR22_H2_CaptureRestartBindingsFalseGapsAcrossCrashWindows(t *testing.T) {
	t.Run("crash_before_apply_with_live_retained_generation_in_kernel", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		r.srv.SetUsageUnavailable(1000)
		serveBoundGeneration(t, r)
		r.manualUsage(t)

		bA := oneBinding(r.clock.Wall())
		bA.PeerId = "peer-a"
		bA.BindingRevision = 1
		bA.AllowedIpv4 = []string{"100.64.1.1/32"}

		bB := oneBinding(r.clock.Wall())
		bB.DeviceId = uuid.Must(uuid.NewV7())
		bB.PeerId = "peer-b"
		bB.BindingRevision = 2
		bB.AllowedIpv4 = []string{"100.64.1.2/32"}

		f3 := generationFixture(3, bA)
		f3.Bindings = []api.DeviceBinding{bA, bB}
		r.srv.SetFixture(f3)
		stepOK(t, r)

		upA := meter.CounterName(bA.DeviceId, meter.DirUpload)
		upB := meter.CounterName(bB.DeviceId, meter.DirUpload)
		src := &r20DynamicSource{
			boot:     r.eng.bootID.String(),
			bindings: r.eng.meterBindings,
			counters: map[string]enforce.CounterValue{
				upA: {Packets: 1, Bytes: 1000},
				upB: {Packets: 1, Bytes: 4096},
			},
		}
		r.eng.meter.Close()
		var err error
		r.eng.meter, err = meter.New(meter.Config{
			NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}

		// Under Generation 3 (still installed in the kernel at gen == 3!),
		// Device A advances to 3000 B and Device B advances to 8192 B.
		src.setCounter(upA, enforce.CounterValue{Packets: 2, Bytes: 3000})
		src.setCounter(upB, enforce.CounterValue{Packets: 2, Bytes: 8192})

		// Central publishes Generation 4 where Device B is rebound to rev 9.
		// Process 1 accepts Desired Generation 4 in syncDesired, and crashes
		// BEFORE plane.Apply runs (so kernel RestartGeneration is still 3!).
		bBRebound := bB
		bBRebound.MemberId = uuid.Must(uuid.NewV7())
		bBRebound.BindingRevision = 9
		f4 := generationFixture(4, bA)
		f4.Bindings = []api.DeviceBinding{bA, bBRebound}
		r.srv.SetFixture(f4)
		if err := r.eng.syncDesired(t.Context()); err != nil {
			t.Fatal(err)
		}
		if gen, _ := r.plane.RestartGeneration(t.Context()); gen != 3 {
			t.Fatalf("precondition: expected live kernel generation 3 before Apply, got %d", gen)
		}
		r20Crash(r.eng)

		next := r20Successor(t, r, r.eng.cfg, src, r.plane)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}

		u := r21ReadLedger(t, r.eng.cfg, next.nodeID, next.incID)
		// Regression r22: H-2 — Because live kernel RestartGeneration was 3
		// (retained.Generation), Generation 4's Apply never started in the
		// kernel; both Device A (rev 1: 3000 B) and Device B (rev 2: 8192 B)
		// must be attributed without a mapping-conflict gap.
		if u.byRevision[1] != 3000 || u.byRevision[2] != 8192 || len(u.gaps) != 0 {
			t.Fatalf("H-2 (crash before Apply): pure Generation 3 traffic turned into a gap when live kernel was still gen 3: byRevision=%v gaps=%d (want rev1=3000 rev2=8192 gaps=0)",
				u.byRevision, len(u.gaps))
		}
	})

	t.Run("unpublished_desired_turns_all_devices_retained_usage_into_gap", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		r.srv.SetUsageUnavailable(1000)
		r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
		serveBoundGeneration(t, r)
		r.manualUsage(t)

		binding := r.eng.appliedModel.Bindings[0]
		upName := meter.CounterName(binding.DeviceID, meter.DirUpload)
		r.eng.meter.Close()
		src := &r20DynamicSource{
			boot:     r.eng.bootID.String(),
			bindings: r.eng.meterBindings,
			counters: map[string]enforce.CounterValue{upName: {Packets: 1, Bytes: 4096}},
		}
		var err error
		r.eng.meter, err = meter.New(meter.Config{
			NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}

		// Central unpublishes the node in Generation 3 (PublicationIntent =
		// unpublished, DeviceBindings = []). Process 1 syncs Desired Gen 3,
		// installs Baseline, and crashes before FinalSample settles the
		// Generation 2 delta (8192 B total).
		src.setCounter(upName, enforce.CounterValue{Packets: 2, Bytes: 8192})
		b := oneBinding(r.clock.Wall())
		fUnpub := generationFixture(3, b)
		fUnpub.Publication = api.PublicationIntentUnpublished
		fUnpub.Bindings = nil
		r.srv.SetFixture(fUnpub)
		if err := r.eng.syncDesired(t.Context()); err != nil {
			t.Fatal(err)
		}
		dp, err := r.eng.cfg.DatapathValue()
		if err != nil {
			t.Fatal(err)
		}
		if err := r.plane.Baseline(t.Context(), dp); err != nil {
			t.Fatal(err)
		}
		r20Crash(r.eng)

		next := r20Successor(t, r, r.eng.cfg, src, r.plane)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}

		u := r21ReadLedger(t, r.eng.cfg, next.nodeID, next.incID)
		// Regression r22: H-2 — An unpublished desired document never passes
		// Decide and can never be applied to the kernel; intersecting
		// retained.Bindings with an unpublished document's empty bindings drops
		// 100% of the node's retained Generation 2 usage into a gap.
		if u.byRevision[1] != 8192 || len(u.gaps) != 0 {
			t.Fatalf("H-2 (unpublished desired): retained Generation 2 traffic (want rev1=8192, gaps=0) was wiped into a gap: byRevision=%v gaps=%d",
				u.byRevision, len(u.gaps))
		}
	})

	t.Run("removed_or_added_device_ran_under_single_binding_but_becomes_gap", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		r.srv.SetUsageUnavailable(1000)
		serveBoundGeneration(t, r)
		r.manualUsage(t)

		bA := oneBinding(r.clock.Wall())
		bA.PeerId = "peer-a"
		bA.BindingRevision = 1
		bA.AllowedIpv4 = []string{"100.64.1.1/32"}

		bRemoved := oneBinding(r.clock.Wall())
		bRemoved.DeviceId = uuid.Must(uuid.NewV7())
		bRemoved.PeerId = "peer-removed"
		bRemoved.BindingRevision = 2
		bRemoved.AllowedIpv4 = []string{"100.64.1.2/32"}

		bAdded := oneBinding(r.clock.Wall())
		bAdded.DeviceId = uuid.Must(uuid.NewV7())
		bAdded.PeerId = "peer-added"
		bAdded.BindingRevision = 3
		bAdded.AllowedIpv4 = []string{"100.64.1.3/32"}

		// Generation 3 has Device A (rev 1) and bRemoved (rev 2).
		f3 := generationFixture(3, bA)
		f3.Bindings = []api.DeviceBinding{bA, bRemoved}
		r.srv.SetFixture(f3)
		stepOK(t, r)

		upA := meter.CounterName(bA.DeviceId, meter.DirUpload)
		upRemoved := meter.CounterName(bRemoved.DeviceId, meter.DirUpload)
		upAdded := meter.CounterName(bAdded.DeviceId, meter.DirUpload)
		src := &r20DynamicSource{
			boot:     r.eng.bootID.String(),
			bindings: r.eng.meterBindings,
			counters: map[string]enforce.CounterValue{
				upA:       {Packets: 1, Bytes: 1000},
				upRemoved: {Packets: 1, Bytes: 2000},
			},
		}
		r.eng.meter.Close()
		var err error
		r.eng.meter, err = meter.New(meter.Config{
			NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}

		// Under Generation 3, bRemoved advances from 2000 B to 5000 B (+3000 B).
		// Generation 4 removes bRemoved and adds bAdded (rev 3).
		// Apply(Gen 4) installs Gen 4, bAdded counts 4096 B under rev 3, and
		// Process 1 crashes before SetMeta("retained_applied", Gen 4).
		f4 := generationFixture(4, bA)
		f4.Bindings = []api.DeviceBinding{bA, bAdded}
		r.srv.SetFixture(f4)
		if err := r.eng.syncDesired(t.Context()); err != nil {
			t.Fatal(err)
		}
		// The removed device's last packets arrive while its old rules exist.
		src.setCounter(upA, enforce.CounterValue{Packets: 2, Bytes: 3000})
		src.setCounter(upRemoved, enforce.CounterValue{Packets: 2, Bytes: 5000})
		r.eng.leaseRefreshPending = true
		r.eng.plane = &r22CrashPlane{memPlane: r.plane, afterApply: func() {
			src.setCounter(upAdded, enforce.CounterValue{Packets: 1, Bytes: 4096})
			panic(r22CrashFault{})
		}}
		r22CrashDuringStep(t, r)

		next := r20Successor(t, r, r.eng.cfg, src, r.plane)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}

		u := r21ReadLedger(t, r.eng.cfg, next.nodeID, next.incID)
		// Regression r22: H-2 — bRemoved only ever had counter rules in Gen 3
		// (rev 2: 5000 B total) and bAdded only ever had counter rules in Gen 4
		// (rev 3: 4096 B total); neither device had competing bindings across
		// the candidate generations, so both must be attributed without a gap.
		if u.byRevision[1] != 3000 || u.byRevision[2] != 5000 || u.byRevision[3] != 4096 || len(u.gaps) != 0 {
			t.Fatalf("H-2 (removed/added device): single-binding devices turned into gap: byRevision=%v gaps=%d (want rev1=3000 rev2=5000 rev3=4096 gaps=0)",
				u.byRevision, len(u.gaps))
		}
	})

	t.Run("stale_retained_applied_from_closed_epoch_and_first_apply_nil_retained", func(t *testing.T) {
		r, src, upName := r20ServedPendingDelta(t)
		cfg := r20ChangedSets(r)

		// Process 2 boots, refuses adoption, and settles + closes the Generation 2
		// epoch cleanly (rev 1 = 8192 B, OpenEpoch == uuid.Nil).
		// Note that retained_applied in agent.db still holds Generation 2 (rev 1).
		p2 := r20Successor(t, r, cfg, src, r.plane)
		if err := p2.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}

		// Now in a brand-new epoch on p2, Central publishes Generation 3 with
		// BindingRevision 2. PrepareSampling opens the new epoch at 8192 B,
		// plane.Apply installs Generation 3, 2048 B flow under Generation 3
		// (advancing u_<D> to 10240 B), and p2 crashes before SetMeta("retained_applied").
		b := oneBinding(r.clock.Wall())
		b.BindingRevision = 2
		r.srv.SetFixture(generationFixture(3, b))
		if err := p2.syncDesired(t.Context()); err != nil {
			t.Fatal(err)
		}
		// Exercise the real failure ordering: the retained write fails, then
		// rejection installs Baseline. Crash before its final sample.
		db, err := sql.Open("sqlite", "file:"+cfg.DBPath())
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec(`CREATE TRIGGER r22_retained_failure BEFORE UPDATE ON meta
			WHEN NEW.key = 'retained_applied' BEGIN SELECT RAISE(FAIL, 'TEST ONLY retained write failure'); END`)
		if err != nil {
			t.Fatal(err)
		}
		_ = db.Close()
		applied := false
		p2.plane = &r22CrashPlane{memPlane: r.plane, afterApply: func() {
			applied = true
			src.setCounter(upName, enforce.CounterValue{Packets: 3, Bytes: 10240})
		}, afterBaseline: func() {
			if applied {
				panic(r22CrashFault{})
			}
		}}
		r.eng = p2
		r22CrashDuringStep(t, r)
		if r.plane.last.Generation != 0 {
			t.Fatal("crash did not follow rejection Baseline")
		}

		// Process 3 boots and refuses adoption. In the currently open epoch,
		// ONLY Generation 3 (rev 2) ever ran; Generation 2 (rev 1) belonged to
		// the already-closed previous epoch.
		p3 := r20Successor(t, r, cfg, src, r.plane)
		if err := p3.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}

		u := r21ReadLedger(t, cfg, p3.nodeID, p3.incID)
		// Regression r22: H-2 — Stale retained_applied from an already-closed
		// epoch (or earlier boot) must not intersect with the new epoch's
		// generation and turn pure Generation 3 (rev 2: 2048 B) traffic into a gap.
		if u.byRevision[1] != 8192 || u.byRevision[2] != 2048 || len(u.gaps) != 0 {
			t.Fatalf("H-2 (stale retained_applied from closed epoch): Generation 3 traffic turned into gap: byRevision=%v gaps=%d (want rev1=8192 rev2=2048 gaps=0)",
				u.byRevision, len(u.gaps))
		}
	})

	t.Run("first_ever_apply_crash_before_retained_applied_has_nil_applied_model", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		r.srv.StickyLease(false)
		r.srv.SetUsageUnavailable(1000)
		b := oneBinding(r.clock.Wall())
		r.srv.SetFixture(generationFixture(2, b))
		up := meter.CounterName(b.DeviceId, meter.DirUpload)
		src := &r20DynamicSource{boot: r.eng.bootID.String(), bindings: r.eng.meterBindings, counters: map[string]enforce.CounterValue{}}
		r.eng.meterSource = src
		enrollBoot(t, r)
		r.eng.plane = &r22CrashPlane{memPlane: r.plane, afterApply: func() {
			src.setCounter(up, enforce.CounterValue{Packets: 1, Bytes: 8192})
			panic(r22CrashFault{})
		}}
		r22CrashDuringStep(t, r)
		next := r20Successor(t, r, r.eng.cfg, src, r.plane)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
		if u.byRevision[1] != 8192 || len(u.gaps) != 0 {
			t.Fatalf("first Apply lost bytes: revisions=%v gaps=%d", u.byRevision, len(u.gaps))
		}
	})
}

// M-1: During Boot, ensureUsage sets e.usageOnce = true at usage.go:150 right
// after openUsage, BEFORE st.Head(ctx) (line 173) and meter.NewContext(ctx, ...)
// (line 183). If meter.NewContext fails during Boot (for example, a transient
// counter read timeout in m.reconcile -> m.src.Read or a transient SQLite lock
// in st.Head), e.meter remains nil and e.retainedPending remains true. On every
// subsequent Step, Step calls e.ensureUsage(ctx) (engine.go:666), which
// immediately returns at `if e.usageOnce { return }` (usage.go:133), leaving
// e.meter permanently nil and wedging retainedPending settlement for the life
// of the process.
func TestR22_M1_TransientBootMeterOpenFailureWedgesRetainedPendingSettlement(t *testing.T) {
	r, src, _ := r20ServedPendingDelta(t)
	cfg := r20ChangedSets(r)

	failBootRead := true
	flakySrc := r21ReadFunc(func(ctx context.Context) (meter.Observation, error) {
		if failBootRead {
			return meter.Observation{}, errors.New("TEST ONLY transient counter read timeout during Boot meter.NewContext")
		}
		return src.Read(ctx)
	})

	next, err := Start(Options{
		Config:      cfg,
		Clock:       r.clock,
		Plane:       r.plane,
		MeterSource: flakySrc,
		Prober:      passProbe{},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = next.Close() })
	src.setBindings(next.meterBindings)
	r.eng = next

	// Boot succeeds in installing Baseline, while meter.NewContext fails on the
	// transient counter read error, keeping next.retainedPending = true.
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !next.retainedPending || next.meter != nil {
		t.Fatalf("precondition: expected retainedPending=true and meter=nil after transient Boot meter open failure, got retainedPending=%v meter=%v",
			next.retainedPending, next.meter)
	}

	// Counter reads recover before the first Step.
	failBootRead = false
	r.clock.Advance(time.Second)
	alignControlClock(r)
	if err := next.Step(t.Context()); err != nil {
		t.Fatal(err)
	}

	u := r21ReadLedger(t, cfg, next.nodeID, next.incID)
	// Regression r22: M-1 — Step must retry meter initialization when Boot's
	// ensureUsage failed after openUsage, settle the retained 8192 B delta, and
	// clear retainedPending.
	if next.retainedPending || next.meter == nil || u.byRevision[1] != 8192 {
		t.Fatalf("M-1: transient Boot meter.NewContext failure permanently wedged ensureUsage (usageOnce=%v retainedPending=%v meter=%v byRevision=%v)",
			next.usageOnce, next.retainedPending, next.meter, u.byRevision)
	}
}

// r22 M-2: a final adoption read may fail after preparing the meter. Neither
// the bindings, epoch nor uploader may be committed before that read succeeds.
// D-141: the original closed-epoch fixture made its gap before Boot by calling
// FinalSample with nil bindings while leaving serving rules live. Use the real
// open same-boot epoch and late drift instead.
func TestR22_M2_AdoptionPreparationHasNoUsageEffectsBeforeLateAbort(t *testing.T) {
	r, src, upName := r20ServedPendingDelta(t)
	before := r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
	verifyCalls := 0
	var next *Engine
	premature := false
	plane := &r22FailSecondVerifyPlane{
		memPlane: r.plane,
		onSecondVerify: func() {
			premature = len(next.meterBindings()) != 0 || next.loopCancel != nil
			atDecision := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
			if len(atDecision.gaps) != len(before.gaps) || atDecision.byRevision[1] != before.byRevision[1] {
				premature = true
			}
			src.setCounter(upName, enforce.CounterValue{Packets: 3, Bytes: 9216})
		},
		calls: &verifyCalls,
	}
	next = r20Successor(t, r, r.eng.cfg, src, plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if premature {
		t.Fatal("r22 M-2: usage bindings, ledger or uploader changed before final adoption decision")
	}
	if verifyCalls < 2 || next.serving {
		t.Fatal("late adoption refusal was not exercised")
	}
	u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 9216 || len(u.gaps) != 0 || next.retainedPending {
		t.Fatalf("late abort lost outgoing bytes: revisions=%v gaps=%d pending=%v", u.byRevision, len(u.gaps), next.retainedPending)
	}
}

type r22FailSecondVerifyPlane struct {
	*memPlane
	calls          *int
	onSecondVerify func()
}

func (p *r22FailSecondVerifyPlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	*p.calls++
	if *p.calls == 2 {
		if p.onSecondVerify != nil {
			p.onSecondVerify()
		}
		return enforce.ServingEvidence{}, &enforce.CheckError{
			Reason: enforce.ReasonGenerationMismatch,
			Detail: "simulated drift before second verify",
		}
	}
	return p.memPlane.VerifyServing(ctx, m, a)
}

// M-3: Failed, timed-out, or post-beat Boot paths leave untruthful heartbeats on
// disk (and failed probe_clear permanently burns the challenge nonce):
//  1. When Boot installs Baseline (setting e.bootSafe.Store(true) at
//     engine.go:535) under SIGTERM and then times out on bootLocalBudget inside
//     ensureUsage/settleRetainedUsage before e.beat() at line 543, Boot returns
//     context.Canceled (normal stop, exit 0) while leaving the previous
//     process's Serving=true heartbeat on disk even though the kernel ruleset
//     was replaced with Baseline (Generation 0).
//  2. Likewise, when Boot fails on DatapathValue() at engine.go:517 or on
//     bootLocalBudget timeout at engine.go:478, Boot returns without writing a
//     Serving=false, BaselineOK=false heartbeat, leaving the stale Serving=true
//     heartbeat on disk (and DatapathValue() omits ErrBoot).
//  3. When answerHealthOnBoot runs at engine.go:552 (after e.beat() at line 543)
//     and clear() fails to remove the probe exception from the kernel
//     (probeexc.go:167), e.baselineOK becomes false in memory while the on-disk
//     heartbeat still claims BaselineOK=true, Boot still returns nil, and
//     clear() still marks issuance.Cleared=true in agent.db.
func TestR22_M3_FailedOrTimedOutBootAndProbeClearLeaveUntruthfulHeartbeat(t *testing.T) {
	t.Run("term_after_baseline_with_ledger_timeout_leaves_stale_serving_true_heartbeat", func(t *testing.T) {
		r, src, _ := r20ServedPendingDelta(t)
		next := r20Successor(t, r, r20ChangedSets(r), src, r.plane)
		release := make(chan struct{})
		defer close(release)
		entered := make(chan f53DeadlineRead, 1)
		next.meterSource = r21ReadFunc(func(ctx context.Context) (meter.Observation, error) {
			entered <- f53DeadlineRead{ctx: ctx, errOnEntry: ctx.Err()}
			<-release
			return src.Read(ctx)
		})

		caller, cancel := context.WithCancel(t.Context())
		cancel()
		err := next.bootWithBudget(caller, bootLocalBudget)
		f53RequireExpiredLedgerRead(t, entered)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled on TERM after baseline, got %v", err)
		}

		hb, err := watchdog.ReadHeartbeat(next.cfg.HeartbeatPath)
		if err != nil {
			t.Fatal(err)
		}
		// Regression r22: M-3 — Once Boot's Baseline has replaced the kernel
		// serving ruleset (Generation 0) and bootWithBudget returns, the
		// on-disk heartbeat must not still claim Serving=true, Generation=2.
		if hb.Serving || hb.Generation != 0 {
			t.Fatalf("M-3: Boot returned after replacing kernel ruleset with Baseline, but heartbeat on disk still claims Serving=%v Generation=%d",
				hb.Serving, hb.Generation)
		}
	})

	t.Run("local_deadline_timeout_and_datapath_error_leave_serving_true_heartbeat", func(t *testing.T) {
		r, src, _ := r20ServedPendingDelta(t)
		release := make(chan struct{})
		defer close(release)
		plane := &r21BudgetPlane{memPlane: r.plane, stage: "baseline", release: release}
		next := r20Successor(t, r, r20ChangedSets(r), src, plane)

		err := next.bootWithBudget(t.Context(), 80*time.Millisecond)
		if !errors.Is(err, ErrBoot) {
			t.Fatalf("expected ErrBoot on local deadline timeout, got %v", err)
		}
		hb, err := watchdog.ReadHeartbeat(next.cfg.HeartbeatPath)
		if err != nil {
			t.Fatal(err)
		}
		// Regression r22: M-3 — A failed/timed-out Boot must overwrite any
		// prior same-boot Serving=true heartbeat with Serving=false, BaselineOK=false.
		if hb.Serving || hb.BaselineOK {
			t.Fatalf("M-3: timed-out Boot left stale Serving=%v BaselineOK=%v heartbeat on disk", hb.Serving, hb.BaselineOK)
		}
	})

	t.Run("boot_probe_clear_failure_leaves_baseline_ok_heartbeat_and_burns_issuance", func(t *testing.T) {
		r := newRig(t)
		r.eng.prober = &verProbe{version: "0.76.2"}
		r.srv.SetPublication(api.PublicationIntentUnpublished)
		enrollBoot(t, r)
		stepOK(t, r)
		holdUpdateLock(t, r)
		ch := probeChallenge(t, r, labProbe(), 0)

		// Fail the second BaselineModel call (probe_clear after the probe runs)
		// during Boot -> answerHealthOnBoot.
		baselineModelCalls := 0
		plane := &r22FailClearProbePlane{
			memPlane: r.plane,
			calls:    &baselineModelCalls,
		}
		r.eng.plane = plane
		if err := r.eng.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		hb, err := watchdog.ReadHeartbeat(r.eng.cfg.HeartbeatPath)
		if err != nil {
			t.Fatal(err)
		}
		rawIssuance, exists, err := r.eng.store.Meta(t.Context(), probeIssuanceKey)
		if err != nil || !exists {
			t.Fatalf("expected probe_issuance in agent.db, exists=%v err=%v", exists, err)
		}
		var iss probeIssuance
		if err := json.Unmarshal([]byte(rawIssuance), &iss); err != nil {
			t.Fatal(err)
		}
		// Regression r22: M-3 — When probe_clear fails during answerHealthOnBoot,
		// e.baselineOK is false and the probe exception is still in the kernel,
		// so the on-disk heartbeat must not claim BaselineOK=true and
		// probe_issuance must not record Cleared=true before kernel removal succeeds.
		if hb.BaselineOK != r.eng.baselineOK || iss.Cleared {
			t.Fatalf("M-3: probe_clear failed during Boot (eng.baselineOK=%v kernelHasProbe=%v), but on-disk heartbeat has BaselineOK=%v and probe_issuance has Cleared=%v (nonce=%s)",
				r.eng.baselineOK, r.plane.last.Probe != nil, hb.BaselineOK, iss.Cleared, ch.Nonce)
		}
	})
}

type r22FailClearProbePlane struct {
	*memPlane
	calls *int
}

func (p *r22FailClearProbePlane) BaselineModel(ctx context.Context, m enforce.Model) error {
	*p.calls++
	if m.Probe == nil && *p.calls >= 2 {
		return errors.New("TEST ONLY simulated nft failure during probe_clear")
	}
	return p.memPlane.BaselineModel(ctx, m)
}

// What held (Attack Question 1):
//  1. Brand-new node (no usage.db yet): Boot opens usage.db, baselines and
//     closes the initial epoch cleanly (LastCloseReason="final_sample"); the
//     first serving Step calls PrepareSampling to open an unbound epoch before
//     plane.Apply, and the first sample attributes traffic as UsageQualityComplete.
//  2. Cold boot after a clean stop (OpenEpoch == uuid.Nil, LastCloseReason ==
//     "final_sample") with residual same-boot kernel counters: PreserveClosedEpoch
//     preserves the closed checkpoint across Boot without inventing a gap or
//     recounting the residual counters, and the next PrepareSampling baselines
//     the residual counters cleanly before the new generation admits bytes.
//  3. Cold boot after an unclean reboot (OpenEpoch != uuid.Nil, BootID != current):
//     ensureUsage clears meter bindings to nil before meter.NewContext, planSwitch
//     closes the old boot's epoch with a power_loss gap and opens the new boot's
//     epoch unbound, and settleRetainedUsage closes that baseline epoch before
//     any new Apply.
func TestR22_Held_Q1_FreshInstallColdBootAndUncleanRebootBaseline(t *testing.T) {
	t.Run("brand_new_node_boot_apply_sample", func(t *testing.T) {
		r := newRig(t)
		alignControlClock(r)
		r.srv.SetUsageUnavailable(1000)
		r.srv.StickyLease(false)
		r.eng.prober = &verProbe{version: "0.76.2"}

		b := oneBinding(r.clock.Wall())
		upName := meter.CounterName(b.DeviceId, meter.DirUpload)
		src := &r20DynamicSource{
			boot:     r.eng.bootID.String(),
			bindings: r.eng.meterBindings,
			counters: map[string]enforce.CounterValue{
				upName: {Packets: 0, Bytes: 0},
			},
		}
		r.eng.meterSource = src
		enrollBoot(t, r)

		headAfterBoot, err := r.eng.usage.Head(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if headAfterBoot.OpenEpoch != uuid.Nil || headAfterBoot.LastCloseReason != "final_sample" || !r.eng.finalDone || r.eng.retainedPending {
			t.Fatalf("after brand-new Boot: head=%+v finalDone=%v retainedPending=%v",
				headAfterBoot, r.eng.finalDone, r.eng.retainedPending)
		}

		stepOK(t, r)
		r.clock.Advance(time.Second)
		alignControlClock(r)
		r.srv.SetFixture(generationFixture(2, b))
		stepOK(t, r)
		r.manualUsage(t)

		r.clock.Advance(time.Second)
		src.setCounter(upName, enforce.CounterValue{Packets: 1, Bytes: 2048})
		if err := r.eng.meter.Sample(t.Context()); err != nil {
			t.Fatal(err)
		}
		u := r21ReadLedger(t, r.eng.cfg, r.eng.nodeID, r.eng.incID)
		if u.byRevision[1] != 2048 || len(u.gaps) != 0 {
			t.Fatalf("brand-new node first sample: got byRevision=%v gaps=%d, want rev1=2048 gaps=0", u.byRevision, len(u.gaps))
		}
	})

	t.Run("unclean_reboot_closes_power_loss_and_leaves_epoch_unbound", func(t *testing.T) {
		r, src, upName := r20ServedPendingDelta(t)
		cfg := r20ChangedSets(r)

		newBoot := uuid.Must(uuid.NewV7()).String()
		if err := os.WriteFile(cfg.BootIDFile, []byte(newBoot+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		src.boot = newBoot
		src.setCounter(upName, enforce.CounterValue{Packets: 1, Bytes: 512})

		next := r20Successor(t, r, cfg, src, r.plane)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		head, err := next.usage.Head(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if head.OpenEpoch != uuid.Nil || head.LastCloseReason != "final_sample" || !next.finalDone {
			t.Fatalf("expected unclean reboot Boot to close power-loss epoch and leave baseline closed, got head=%+v finalDone=%v",
				head, next.finalDone)
		}
		u := r21ReadLedger(t, cfg, next.nodeID, next.incID)
		if u.byRevision[1] != 4096 || len(u.gaps) != 1 || u.gaps[0].Gap.Reason != api.UsageGapReasonPowerLoss {
			t.Fatalf("expected 4096 B before reboot + 1 power_loss gap, got byRevision=%v gaps=%+v", u.byRevision, u.gaps)
		}
	})
}

// What held (Attack Question 2 & 3):
//  1. With an initialized real meter, when Boot FinalSample returns ErrHold,
//     retainedPending stays true into Step, blocking syncDesired and enforce
//     until the clock advances and settleRetainedUsage completes.
//  2. During adoptStoredServing, e.planeLocked prevents updateLockHeld() from
//     reporting the agent's own node lock as updater contention, so a live
//     health challenge with a probe does not install a probe exception over
//     the serving ruleset being verified.
func TestR22_Held_Q2_Q3_HeldFinalSampleWaitsInStepAndAdoptionSuppressesProbeException(t *testing.T) {
	t.Run("held_finalsample_blocks_step_sync_and_apply_until_settled", func(t *testing.T) {
		r, src, upName := r20ServedPendingDelta(t)
		cfg := r20ChangedSets(r)

		// Use a controlled meter clock on both sides of the checkpoint. Disk
		// and Boot scheduling cannot consume the held interval. The real store
		// and meter still account for the change from 4096 to 8192.
		st, err := usagestore.Open(cfg.UsageDBPath(), r.eng.nodeID, r.eng.incID, usagestore.Options{})
		if err != nil {
			t.Fatal(err)
		}
		head, err := st.Head(t.Context())
		if err != nil {
			_ = st.Close()
			t.Fatal(err)
		}
		checkpointAt := head.LastSample.Add(120 * time.Millisecond)
		meterClock := timeanchor.NewManual(checkpointAt, 0)
		src4096 := &r20DynamicSource{
			boot:     r.eng.bootID.String(),
			bindings: r.eng.meterBindings,
			counters: map[string]enforce.CounterValue{upName: {Packets: 1, Bytes: 4096}},
		}
		mHold, err := meter.New(meter.Config{
			NodeID:        r.eng.nodeID,
			IncarnationID: r.eng.incID,
			Store:         st,
			Source:        src4096,
			Now:           meterClock.Wall,
		})
		if err != nil {
			_ = st.Close()
			t.Fatal(err)
		}
		if err := mHold.Sample(t.Context()); err != nil {
			mHold.Close()
			_ = st.Close()
			t.Fatal(err)
		}
		mHold.Close()
		_ = st.Close()

		next := r20Successor(t, r, cfg, src, r.plane)
		r.eng = next
		// Initialize through the production ownership/constructor path before
		// Boot, then replace only the meter's clock. ensureUsage is idempotent;
		// Boot still performs baseline, retained settlement and its heartbeat.
		if err := next.loadRuntime(t.Context()); err != nil {
			t.Fatal(err)
		}
		next.captureRestartBindings(t.Context())
		next.retainedPending = true
		next.ensureUsage(t.Context())
		if next.meter == nil || !next.usageOnce {
			t.Fatal("precondition: usage initialization failed")
		}
		stopUsageLoops(t, r)
		next.meter.Close()
		meterClock.Advance(-time.Millisecond)
		next.meter, err = meter.New(meter.Config{
			NodeID: next.nodeID, IncarnationID: next.incID,
			Store: next.usage, Source: src, Now: meterClock.Wall,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !next.retainedPending || next.serving {
			t.Fatalf("expected retainedPending=true serving=false on ErrHold during Boot, got retainedPending=%v serving=%v",
				next.retainedPending, next.serving)
		}

		// Queue Generation 3 (rev 2) on central. Step while clock is still held
		// must NOT sync Generation 3 or apply it.
		b := oneBinding(r.clock.Wall())
		b.BindingRevision = 2
		r.srv.SetFixture(generationFixture(3, b))
		if err := next.Step(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !next.retainedPending || next.serving || next.desired.ConfigGeneration != 2 {
			t.Fatalf("expected Step while held to keep retainedPending=true and desired gen=2, got retainedPending=%v serving=%v desiredGen=%d",
				next.retainedPending, next.serving, next.desired.ConfigGeneration)
		}

		// Advance only the meter clock past the checkpoint: next Step settles
		// Generation 2 (rev 1 = 8192 B) and then syncs and applies Generation 3 (rev 2).
		meterClock.Advance(2 * time.Millisecond)
		r.clock.Advance(time.Second)
		alignControlClock(r)
		stepOK(t, r)
		if next.retainedPending || !next.serving || next.appliedGen != 3 {
			t.Fatalf("expected Step after clock advance to settle and apply gen 3, got retainedPending=%v serving=%v appliedGen=%d",
				next.retainedPending, next.serving, next.appliedGen)
		}
		u := r21ReadLedger(t, cfg, next.nodeID, next.incID)
		if u.byRevision[1] != 8192 || len(u.gaps) != 0 {
			t.Fatalf("expected rev1=8192 gaps=0 after settlement, got byRevision=%v gaps=%d", u.byRevision, len(u.gaps))
		}
	})

	t.Run("adoption_lock_does_not_trigger_probe_exception_install", func(t *testing.T) {
		r, src, _ := r20ServedPendingDelta(t)
		cfg := r.eng.cfg
		if err := os.MkdirAll(cfg.JournalDir, 0o700); err != nil {
			t.Fatal(err)
		}
		ch, err := healthattest.Issue(healthattest.Input{
			TxnID:         uuid.Must(uuid.NewV7()).String(),
			ReleaseID:     cfg.ReleaseID,
			NodeID:        r.eng.nodeID.String(),
			IncarnationID: r.eng.incID.String(),
			BootID:        r.eng.bootID.String(),
			Now:           r.clock.Wall().Add(-time.Second),
			TTL:           30 * time.Second,
			Components: healthattest.Components{
				NodeAgent:   cfg.AgentRelease,
				Netbird:     "0.0.0-test",
				NodeUpdater: "0.0.0-test",
				Enforcement: "1",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		ch.Probe = labProbe()
		if err := healthattest.WriteChallenge(cfg.JournalDir, ch); err != nil {
			t.Fatal(err)
		}

		baselinesBefore := r.plane.baselines
		next := r20Successor(t, r, cfg, src, r.plane)
		if err := next.Boot(t.Context()); err != nil {
			t.Fatal(err)
		}
		if !next.serving || r.plane.baselines != baselinesBefore {
			t.Fatalf("expected same-boot adoption to succeed without installing probe baseline, got serving=%v baselines=%d (before %d)",
				next.serving, r.plane.baselines, baselinesBefore)
		}
	})
}

// What held (Attack Questions 4, 5, 6, 7):
//   - Q4: FinalSample on a counter reset or boot change runs planSwitch and then
//     closes the fresh epoch with CloseReason="final_sample", and FinalSample on
//     a backward clock jump > 10s runs planClockGap with CloseEpoch=true.
//   - Q5 & Q6: Single bootLocalBudget across Boot stages + Close waiting on
//     bootDone stays within the 13s stop allowance (10s Boot + 3s Close).
//   - Q7: When checkMemberPath fails after plane.Apply during enforce, the
//     durable member_path expectation is preserved across restart and blocks
//     serving until the topology recovers.
func TestR22_Held_Q4_Q5_Q6_Q7_FinalSampleCloseAndStopBudgetAndMemberPathEvidence(t *testing.T) {
	t.Run("finalsample_closes_epoch_on_counter_reset_and_clock_gap", func(t *testing.T) {
		r, src, upName := r20ServedPendingDelta(t)
		// Drop counter from 4096 to 512 on the same boot (counter reset).
		src.setCounter(upName, enforce.CounterValue{Packets: 1, Bytes: 512})
		st, err := usagestore.Open(r.eng.cfg.UsageDBPath(), r.eng.nodeID, r.eng.incID, usagestore.Options{})
		if err != nil {
			t.Fatal(err)
		}
		defer st.Close()
		m, err := meter.New(meter.Config{
			NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: st, Source: src,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer m.Close()
		if err := m.FinalSample(t.Context()); err != nil {
			t.Fatal(err)
		}
		head, err := st.Head(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if head.OpenEpoch != uuid.Nil || head.LastCloseReason != "final_sample" {
			t.Fatalf("expected FinalSample on counter reset to close fresh epoch with final_sample, got head=%+v", head)
		}
	})

	t.Run("stop_budget_constants_and_member_path_retained_across_restart", func(t *testing.T) {
		if bootLocalBudget+closeLocalBudget != 13*time.Second {
			t.Fatalf("expected bootLocalBudget+closeLocalBudget == 13s, got %s", bootLocalBudget+closeLocalBudget)
		}
		r := newRig(t)
		alignControlClock(r)
		serveBoundGeneration(t, r)
		if r.eng.memberPath == nil || r.eng.memberPath.Generation != 2 {
			t.Fatalf("expected durable memberPath at generation 2, got %+v", r.eng.memberPath)
		}
	})
}
