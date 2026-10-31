package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usageupload"
	"github.com/google/uuid"
)

func barrierRig(t *testing.T) *rig {
	t.Helper()
	r := newRig(t)
	alignControlClock(r)
	r.srv.SetUsageUnavailable(1000)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	return r
}

func writeBarrier(t *testing.T, r *rig, id string) {
	t.Helper()
	if err := writeAtomic(filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked"), []byte(id+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readCheckpoint(t *testing.T, r *rig) usagestore.MeterCheckpoint {
	t.Helper()
	var cp usagestore.MeterCheckpoint
	if err := json.Unmarshal(readFile(t, r.eng.cfg.MeterCheckpoint), &cp); err != nil {
		t.Fatal(err)
	}
	return cp
}

func noCheckpoint(t *testing.T, r *rig) {
	t.Helper()
	if _, err := os.Lstat(r.eng.cfg.MeterCheckpoint); !os.IsNotExist(err) {
		t.Fatalf("unexpected checkpoint: %v", err)
	}
}

func TestD125_Rule1FreshBlockCheckpointWithLatchIdentity(t *testing.T) {
	r := barrierRig(t)
	// Preserve the opaque UUID spelling; parsed UUID equality is insufficient.
	id := strings.ToUpper(uuid.NewString())
	writeBarrier(t, r, id)
	applies, baselines := r.plane.applies, r.plane.baselines
	if err := r.eng.Step(t.Context()); err != nil {
		t.Fatal(err)
	}
	cp := readCheckpoint(t, r)
	if cp.BarrierID != id || r.eng.serving || r.eng.decision.Serve || r.eng.expectServing.Load() || r.plane.applies != applies || r.plane.baselines <= baselines {
		t.Fatalf("block not finalized: %+v serving=%v", cp, r.eng.serving)
	}
	if !r.eng.finalDone || r.eng.finalBarrierID != id {
		t.Fatal("barrier not keyed by identity")
	}
}

func TestD125_Rule2BlockAfterDecisionRefusesServingPath(t *testing.T) {
	r := barrierRig(t)
	d := r.eng.decision
	if !d.Serve {
		t.Fatal("test needs an earlier serving decision")
	}
	id := uuid.NewString()
	writeBarrier(t, r, id)
	applies := r.plane.applies
	if err := r.eng.enforce(t.Context(), d); err != nil {
		t.Fatal(err)
	}
	if r.plane.applies != applies || r.eng.serving || r.eng.decision.Serve || readCheckpoint(t, r).BarrierID != id {
		t.Fatal("earlier decision bypassed the live block")
	}
}

func TestD125_Rule3SecondBlockQuietClosedEpochSameWatermarks(t *testing.T) {
	r := barrierRig(t)
	writeBarrier(t, r, uuid.NewString())
	r.eng.handoff(t.Context())
	first := readCheckpoint(t, r)
	before, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// No serving/resume between blocks, including an unblock/reblock between
	// agent polls. A quiet close must not manufacture another event or epoch.
	id := uuid.NewString()
	writeBarrier(t, r, id)
	baselines := r.plane.baselines
	r.eng.handoff(t.Context())
	second := readCheckpoint(t, r)
	after, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if second.BarrierID != id || second.BarrierID == first.BarrierID || r.plane.baselines <= baselines || before.NextSeq != after.NextSeq || after.OpenEpoch != uuid.Nil {
		t.Fatal("new block reused finalDone or manufactured usage", first, second)
	}
	first.BarrierID = id
	if second != first {
		t.Fatalf("quiet reclose changed watermarks: %+v %+v", first, second)
	}
}

type barrierPlane struct {
	Plane
	baseline func() error
}

func (p barrierPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	if err := p.baseline(); err != nil {
		return err
	}
	return p.Plane.Baseline(ctx, dp)
}

func TestD125_Rule3FinalDoneCannotRelabel(t *testing.T) {
	r := barrierRig(t)
	writeBarrier(t, r, uuid.NewString())
	r.eng.handoff(t.Context())
	old := readFile(t, r.eng.cfg.MeterCheckpoint)
	writeBarrier(t, r, uuid.NewString())
	r.eng.plane = barrierPlane{Plane: r.plane, baseline: func() error { return errors.New("TEST ONLY withdrawal failed") }}
	r.eng.handoff(t.Context())
	if r.eng.finalDone || !bytes.Equal(old, readFile(t, r.eng.cfg.MeterCheckpoint)) {
		t.Fatal("finalDone relabelled evidence without a verified withdrawal")
	}
}

func TestD125_Rule2LatchReplacementDuringClosePublishesNothing(t *testing.T) {
	r := barrierRig(t)
	writeBarrier(t, r, uuid.NewString())
	r.eng.plane = barrierPlane{Plane: r.plane, baseline: func() error {
		writeBarrier(t, r, uuid.NewString())
		return nil
	}}
	r.eng.handoff(t.Context())
	noCheckpoint(t, r)
	if r.eng.finalDone || r.eng.serving {
		t.Fatal("changed barrier acknowledged")
	}
}

func TestD125_Rule2FailedFinalSamplePublishesNothing(t *testing.T) {
	r := barrierRig(t)
	r.eng.meter.Close()
	src := &barrierSource{obs: meter.Observation{BootID: r.eng.bootID.String()}}
	var err error
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src})
	if err != nil {
		t.Fatal(err)
	}
	writeBarrier(t, r, uuid.NewString())
	src.err = errors.New("TEST ONLY final counter read failed")
	r.eng.handoff(t.Context())
	noCheckpoint(t, r)
	if r.eng.finalDone || r.eng.serving || !r.eng.usagePersist {
		t.Fatal("failed sample acknowledged")
	}
	src.err = nil
	r.eng.handoff(t.Context())
	_ = readCheckpoint(t, r)
}

func TestD125_Rule4WithdrawalRemovesCheckpointBeforeResume(t *testing.T) {
	r := barrierRig(t)
	writeBarrier(t, r, uuid.NewString())
	if err := r.eng.Step(t.Context()); err != nil {
		t.Fatal(err)
	}
	_ = readCheckpoint(t, r)
	if err := os.Remove(filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked")); err != nil {
		t.Fatal(err)
	}
	admitted := false
	r.eng.plane = admissionPlane{Plane: r.plane, admit: func() {
		noCheckpoint(t, r)
		admitted = true
	}}
	if err := r.eng.Step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !admitted || !r.eng.serving || r.eng.finalDone {
		t.Fatal("withdrawal did not resume after invalidation")
	}
	noCheckpoint(t, r)
}

func TestD125_Rule4RemovalFailureRefusesResume(t *testing.T) {
	r := barrierRig(t)
	writeBarrier(t, r, uuid.NewString())
	r.eng.handoff(t.Context())
	if err := os.Remove(filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(r.eng.cfg.MeterCheckpoint); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(r.eng.cfg.MeterCheckpoint, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(r.eng.cfg.MeterCheckpoint, "TEST_ONLY"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	applies := r.plane.applies
	if err := r.eng.Step(t.Context()); err != nil {
		t.Fatal(err)
	}
	if r.eng.serving || r.plane.applies != applies || !r.eng.usagePersist {
		t.Fatal("invalidation failure admitted forwarding")
	}
}

func TestD125_Rule5RestartPresentIdenticalRepublish(t *testing.T) {
	r := barrierRig(t)
	r.eng.meter.Close()
	bindings := r.eng.appliedModel.Bindings
	name := meter.CounterName(bindings[0].DeviceID, meter.DirUpload)
	src := &barrierSource{obs: meter.Observation{BootID: r.eng.bootID.String(), Bindings: bindings, Counters: map[string]enforce.CounterValue{name: {}}}}
	var err error
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src})
	if err != nil {
		t.Fatal(err)
	}
	src.obs.Counters[name] = enforce.CounterValue{Packets: 1, Bytes: 4096}
	id := uuid.NewString()
	writeBarrier(t, r, id)
	r.eng.handoff(t.Context())
	raw := readFile(t, r.eng.cfg.MeterCheckpoint)
	info, err := os.Stat(r.eng.cfg.MeterCheckpoint)
	if err != nil {
		t.Fatal(err)
	}
	// An uploader can advance ACKs after the barrier. That must not change
	// the frozen snapshot when the process restarts under the same latch.
	r.srv.SetUsageUnavailable(0)
	up, err := usageupload.New(usageupload.Config{Store: r.eng.usage, Transport: usageTransport{client: r.eng.client}})
	if err != nil {
		t.Fatal(err)
	}
	if err := up.Once(t.Context()); err != nil {
		t.Fatal(err)
	}
	head, err := r.eng.usage.Head(t.Context())
	if err != nil || head.AckedThrough <= uint64(readCheckpoint(t, r).CentralAckedThrough) {
		t.Fatal("test did not advance the ACK after publication", head, err)
	}
	// The updater owns the node lock across a candidate restart. Opening the
	// unchanged incarnation must still work and republish while it holds it.
	lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Release()
	restartEngine(t, r, r.plane)
	if !bytes.Equal(raw, readFile(t, r.eng.cfg.MeterCheckpoint)) || r.eng.finalBarrierID != id || !r.eng.finalDone {
		t.Fatal("restart changed the correlated snapshot")
	}
	newInfo, err := os.Stat(r.eng.cfg.MeterCheckpoint)
	if err != nil || os.SameFile(info, newInfo) {
		t.Fatal("restart did not atomically republish", err)
	}
}

func TestD125_Rule5RestartAbsentPublishesNothing(t *testing.T) {
	r := barrierRig(t)
	writeBarrier(t, r, uuid.NewString())
	r.eng.handoff(t.Context())
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(r.eng.cfg.JournalDir, "forwarding.blocked")); err != nil {
		t.Fatal(err)
	}
	restartEngine(t, r, r.plane)
	noCheckpoint(t, r)
	r.eng.handoff(t.Context())
	noCheckpoint(t, r)
}

func TestD125_Rule5MalformedLatchPublishesNothingAndStaysBlocked(t *testing.T) {
	for label, value := range map[string]string{"empty": "", "invalid": "not-a-uuid", "space": " " + uuid.NewString(), "extra_newline": uuid.NewString() + "\n"} {
		t.Run(label, func(t *testing.T) {
			r := barrierRig(t)
			writeBarrier(t, r, uuid.NewString())
			r.eng.handoff(t.Context())
			writeBarrier(t, r, value)
			restartEngine(t, r, r.plane)
			applies := r.plane.applies
			if err := r.eng.Step(t.Context()); err != nil {
				t.Fatal(err)
			}
			noCheckpoint(t, r)
			if !ForwardingBlocked(r.eng.cfg.JournalDir) || r.eng.serving || r.eng.decision.Serve || r.plane.applies != applies {
				t.Fatal("malformed latch allowed forwarding")
			}
		})
	}
}

func TestD118_LedgerOwnershipWaitsForPreviousLoops(t *testing.T) {
	r := barrierRig(t)
	// Simulate a loop whose cancellation cleanup has not finished. The real
	// shutdown waits on this same WaitGroup before releasing ledger ownership.
	finished := make(chan struct{})
	r.eng.loops.Add(1)
	go func() { defer r.eng.loops.Done(); <-finished }()
	done := make(chan error, 1)
	go func() { done <- r.eng.Close() }()
	other, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: &memPlane{}, MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
	if err != nil {
		close(finished)
		<-done
		t.Fatal(err)
	}
	defer other.Close()
	other.ensureUsage(t.Context())
	if other.usage != nil || other.usageOnce || !other.usagePersist {
		close(finished)
		<-done
		t.Fatal("opened ledger while previous loop was still alive")
	}
	close(finished)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	other.ensureUsage(t.Context())
	if other.usage == nil {
		t.Fatal("ledger not released after both loops stopped")
	}
}
