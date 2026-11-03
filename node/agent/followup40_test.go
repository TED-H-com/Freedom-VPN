package agent

// TEST ONLY disk SQLite and injected data planes; no host networking.
import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/google/uuid"
)

func f40Live(t *testing.T) (*rig, *r20DynamicSource, string) {
	t.Helper()
	r := barrierRig(t)
	b := r.eng.appliedModel.Bindings[0]
	up := meter.CounterName(b.DeviceID, meter.DirUpload)
	src := &r20DynamicSource{boot: r.eng.bootID.String(), bindings: r.eng.meterBindings, counters: map[string]enforce.CounterValue{up: {Packets: 1, Bytes: 4096}}}
	r.eng.meter.Close()
	var err error
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	src.setCounter(up, enforce.CounterValue{Packets: 2, Bytes: 8192})
	return r, src, up
}

type f40MarkerError struct{ *memPlane }

func (p f40MarkerError) RestartGeneration(context.Context) (uint64, error) {
	return 0, errors.New("TEST ONLY transient marker read failure")
}

func TestR23_H1c_MarkerReadFailureKeepsIntentAcrossRestart(t *testing.T) {
	r, src, _ := f40Live(t)
	// No legacy bound-counter evidence: the new device has counted only since
	// the real pre-Apply baseline. The intent must supply the sole candidate.
	b := oneBinding(r.clock.Wall())
	b.DeviceId, b.MemberId = uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	b.BindingRevision, b.AllowedIpv4 = 2, []string{"100.64.0.12/32"}
	r.srv.SetFixture(generationFixture(3, b))
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	src.setCounter(meter.CounterName(b.DeviceId, meter.DirUpload), enforce.CounterValue{Packets: 1, Bytes: 2048})
	r20Crash(r.eng)
	next := r20Successor(t, r, r20ChangedSets(r), src, f40MarkerError{r.plane})
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || u.byRevision[2] != 2048 || len(u.gaps) != 0 {
		t.Fatalf("intent lost: %+v", u)
	}
	r.eng = next
	r20Crash(next)
	next = r20Successor(t, r, r.eng.cfg, src, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u = r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || u.byRevision[2] != 2048 || len(u.gaps) != 0 {
		t.Fatalf("restart recounted: %+v", u)
	}
}

func TestR23_H1_IntentCannotCrossSettledBoundaryOrFailedIntentWrite(t *testing.T) {
	r, src, _ := f40Live(t)
	old := r.eng.restartIntent
	r.eng.handoff(t.Context())
	if !r.eng.finalDone {
		t.Fatal("handoff failed")
	}
	if err := r.eng.meter.PrepareSampling(t.Context()); err != nil {
		t.Fatal(err)
	}
	h, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ok, err := r.eng.usage.SameCountingInterval(t.Context(), old.Epoch, h.OpenEpoch, src.boot)
	if err != nil || ok {
		t.Fatalf("settled intent crosses baseline: %v %v", ok, err)
	}
	// Actual failed metadata write before any later Apply. The old intent is
	// durable, but cannot label this unbound epoch, even on the same OS boot.
	db := f40AgentDB(t, r.eng.cfg.DBPath())
	if _, err := db.Exec(`CREATE TRIGGER f40_intent_fail BEFORE UPDATE ON meta WHEN NEW.key='usage_apply_intent' BEGIN SELECT RAISE(FAIL,'TEST ONLY intent failure'); END`); err != nil {
		t.Fatal(err)
	}
	model := *r.eng.appliedModel
	model.Generation++
	model.Bindings = append([]enforce.Binding(nil), model.Bindings...)
	model.Bindings[0].MemberID = uuid.Must(uuid.NewV7())
	if err := r.eng.recordUsageIntent(t.Context(), model); err == nil {
		t.Fatal("intent write did not fail")
	}
	r20Crash(r.eng)
	next := r20Successor(t, r, r20ChangedSets(r), src, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	u := r21ReadLedger(t, next.cfg, next.nodeID, next.incID)
	if u.byRevision[1] != 8192 || len(u.gaps) != 0 {
		t.Fatalf("settled bytes changed: %+v", u)
	}
}

// A production reset/nudge retains the renderer's idle zero counters. This
// variant exercises the same switch/restart as the original missing-counter
// guard, without claiming that absent counters are a healthy verified state.
func TestR23_H1b_IdleZeroCounterSurvivesSwitchAndRestart(t *testing.T) {
	r23H1b(t, true)
}

func f40AgentDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
