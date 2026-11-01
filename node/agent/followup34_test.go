package agent

import (
	"context"
	"encoding/json"
	"net/netip"
	"strconv"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/agentstore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// A real Step can report g while desired/lease h are ready: a due renewal
// obtains h, then a busy node lock keeps the verified g model installed.
func TestF34_ReportBeforeNewApplyOnBusyLock(t *testing.T) {
	r := newRig(t)
	_, drop := servePair(t, r)
	g, baselines, applies := r.eng.appliedGen, r.plane.baselines, r.plane.applies
	// Same-boot adoption legitimately makes renewal due before the next Apply.
	restartEngine(t, r, r.plane)
	r.manualUsage(t)
	if !r.eng.leaseRefreshPending {
		t.Fatal("adoption did not schedule renewal")
	}
	r.srv.SetFixture(fixtureWithBindings(g+1, threeBindings(r.clock.Wall())[:1]))
	lock, err := nodelock.Acquire(context.Background(), r.eng.cfg.LockPath, 1, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	r.eng.lastObs = time.Time{}
	stepOK(t, r)
	rep, ok := r.srv.LastObservation()
	ack, err := rep.AppliedConfigGeneration.Get()
	if !ok || err != nil || uint64(ack) != g || !rep.MemberForwardingEnabled ||
		uint64(r.eng.desired.ConfigGeneration) != g+1 || uint64(r.eng.lease.ConfigGeneration) != g+1 ||
		!modelHasPrefix(r.plane.last, drop.AllowedIpv4[0]) || r.plane.baselines != baselines || r.plane.applies != applies {
		t.Fatalf("window not reached: ack=%d serving=%v desired=%s lease=%s baselines=%d applies=%d log=%s", ack, rep.MemberForwardingEnabled, r.eng.desired.ConfigGeneration, r.eng.lease.ConfigGeneration, r.plane.baselines, r.plane.applies, r.log.String())
	}
}

func TestF34_AppliedSnapshotSurvivesNewDesiredAndRestart(t *testing.T) {
	r := newRig(t)
	servePair(t, r)
	want := snapshotOf(r.eng.desired)
	r.srv.SetFixture(fixtureWithBindings(4, threeBindings(r.clock.Wall())[:1]))
	if err := r.eng.syncDesired(context.Background()); err != nil {
		t.Fatal(err)
	}
	restartEngine(t, r, r.plane)
	if r.eng.appliedSnapshot == nil || *r.eng.appliedSnapshot != want {
		t.Fatalf("snapshot=%+v want=%+v", r.eng.appliedSnapshot, want)
	}
	rep, err := r.eng.report(context.Background(), r.clock.Wall())
	refs, rerr := rep.AppliedPolicyRefs.Get()
	bind, berr := rep.AppliedBindingRevision.Get()
	if err != nil || rerr != nil || berr != nil || refs != want.PolicyRefs || bind != want.BindingRevision {
		t.Fatalf("historical ACK lost refs: %+v %v", rep, err)
	}
	r.eng.identityFrozen = true
	rep, err = r.eng.report(context.Background(), r.clock.Wall())
	if err != nil || !rep.AppliedConfigGeneration.IsNull() || !rep.AppliedPolicyRefs.IsNull() || !rep.AppliedBindingRevision.IsNull() {
		t.Fatal("frozen ACK exposed provenance")
	}
}

func TestF34_MissingOrMismatchedSnapshotDoesNotBorrowNewDesired(t *testing.T) {
	for _, stored := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "mismatch"}[stored], func(t *testing.T) {
			r := newRig(t)
			servePair(t, r)
			r.eng.appliedSnapshot = nil
			r.eng.appliedGen--
			if stored {
				raw, _ := json.Marshal(snapshotOf(r.eng.desired))
				if err := r.eng.store.SetMeta(context.Background(), agentstore.AppliedSnapshotKey, string(raw)); err != nil {
					t.Fatal(err)
				}
			} else {
				// Legacy absence is exercised using a fresh metadata store.
				disk, err := agentstore.Open(t.TempDir() + "/legacy.db")
				if err != nil {
					t.Fatal(err)
				}
				defer disk.Close()
				orig := r.eng.store
				r.eng.store = disk
				defer func() { r.eng.store = orig }()
			}
			if err := r.eng.loadAppliedSnapshot(context.Background()); err != nil {
				t.Fatal(err)
			}
			rep, err := r.eng.report(context.Background(), r.clock.Wall())
			if err != nil || rep.AppliedConfigGeneration.IsNull() || !rep.AppliedBindingRevision.IsNull() || !rep.AppliedPolicyRefs.IsNull() {
				t.Fatal("missing history borrowed newer desired refs")
			}
		})
	}
}

func TestF34_ReservationFailurePreventsApply(t *testing.T) {
	r := newRig(t)
	servePair(t, r)
	model := *r.eng.appliedModel
	model.Bindings = model.Bindings[:1]
	applies, baselines := r.plane.applies, r.plane.baselines
	if err := r.eng.store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.enforce(context.Background(), Decision{Serve: true, Model: model, PolicyExpires: r.eng.policyExpires}); err == nil {
		t.Fatal("reservation failure hidden")
	}
	if r.eng.serving || r.eng.storageOK || r.plane.applies != applies || r.plane.baselines <= baselines {
		t.Fatal("failed reservation reached Apply or kept forwarding")
	}
}

func TestF34_GenerationChangeKeepsPendingReservationCount(t *testing.T) {
	r := newRig(t)
	keep, drop := servePair(t, r)
	addr := netip.MustParsePrefix(drop.AllowedIpv4[0]).Addr()
	p := &failedSubsetPlane{memPlane: r.plane, fail: map[netip.Addr]bool{addr: true}}
	r.eng.plane = p
	for gen := uint64(4); gen <= 5; gen++ {
		r.srv.SetFixture(fixtureWithBindings(gen, []api.DeviceBinding{keep}))
		stepOK(t, r)
		raw, ok, err := r.eng.store.Meta(context.Background(), conntrackAttemptsKey)
		if err != nil || !ok || raw != strconv.FormatUint(gen-3, 10) || r.eng.conntrackAttempts != int(gen-3) || len(r.eng.pendingRevoke) != 1 || r.eng.pendingRevoke[0].Addr != addr {
			t.Fatalf("generation=%d attempts=%d durable=%q pending=%v err=%v", gen, r.eng.conntrackAttempts, raw, r.eng.pendingRevoke, err)
		}
	}
	restartEngine(t, r, p)
	if r.eng.conntrackAttempts != 2 || len(r.eng.pendingRevoke) != 1 {
		t.Fatal("restart lost cross-generation accounting")
	}
}
