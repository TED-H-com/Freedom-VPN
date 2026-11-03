package agent

// TEST ONLY: disk SQLite and synthetic counters, no network or timing run.
import (
	"encoding/json"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/google/uuid"
)

func f42Rotated(t *testing.T) (*rig, *r20DynamicSource) {
	t.Helper()
	r, src, up := f40Live(t)
	r.clock.Advance(time.Second)
	r.eng.leaseRefreshPending = true
	stepOK(t, r)
	r.clock.Advance(time.Second)
	src.setCounter(up, enforce.CounterValue{Packets: 1, Bytes: 1024})
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	signedAt := r.eng.desired.IssuedAt.Add(time.Minute)
	r.srv.SetClock(func() time.Time { return signedAt })
	h, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if r.eng.restartIntent.Epoch == h.OpenEpoch {
		t.Fatal("no discontinuity")
	}
	return r, src
}

func TestMTR03_F42_RetainedEpochIntentIsDurable(t *testing.T) {
	r, src := f42Rotated(t)
	before := r.plane.applies
	r.clock.Advance(10 * time.Second)
	stepOK(t, r)
	h, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	raw, ok, err := r.eng.store.Meta(t.Context(), usageIntentKey)
	if err != nil || !ok {
		t.Fatal(err)
	}
	var intent usageIntent
	if err := json.Unmarshal([]byte(raw), &intent); err != nil {
		t.Fatal(err)
	}
	if r.plane.applies != before || intent.Epoch != h.OpenEpoch || r.eng.restartIntent.Epoch != intent.Epoch {
		t.Fatal("skip did not durably refresh epoch")
	}
	r20Crash(r.eng)
	next := r20Successor(t, r, r.eng.cfg, src, r.plane)
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if next.restartIntent == nil || next.restartIntent.Epoch != intent.Epoch {
		t.Fatal("restart lost refreshed intent")
	}
}

func TestMTR03_F42_RetentionRejectsCountingBoundaries(t *testing.T) {
	for _, boundary := range []string{"closed", "final_sample", "boot"} {
		t.Run(boundary, func(t *testing.T) {
			r, src := f42Rotated(t)
			if boundary == "boot" {
				src.mu.Lock()
				src.boot = uuid.Must(uuid.NewV7()).String()
				src.mu.Unlock()
				if err := r.eng.meter.Sample(t.Context()); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := r.eng.meter.FinalSample(t.Context()); err != nil {
					t.Fatal(err)
				}
				if boundary == "final_sample" {
					if err := r.eng.meter.PrepareSampling(t.Context()); err != nil {
						t.Fatal(err)
					}
				}
			}
			lock, err := nodelock.TryAcquire(r.eng.cfg.LockPath)
			if err != nil {
				t.Fatal(err)
			}
			defer lock.Release()
			if r.eng.canRetainServing(t.Context(), Decide(r.eng.input())) {
				t.Fatal("retained across", boundary)
			}
		})
	}
}

func TestMTR03_F42_IntentRefreshFailureWithdraws(t *testing.T) {
	r, _ := f42Rotated(t)
	before, old := r.plane.applies, *r.eng.restartIntent
	raw, _, _ := r.eng.store.Meta(t.Context(), usageIntentKey)
	db := f40AgentDB(t, r.eng.cfg.DBPath())
	if _, err := db.Exec(`CREATE TRIGGER f42_intent_fail BEFORE UPDATE ON meta WHEN NEW.key='usage_apply_intent' BEGIN SELECT RAISE(FAIL,'TEST ONLY intent failure'); END`); err != nil {
		t.Fatal(err)
	}
	stepOK(t, r)
	after, _, _ := r.eng.store.Meta(t.Context(), usageIntentKey)
	if r.plane.applies != before || r.eng.serving || r.eng.restartIntent.Epoch != old.Epoch || after != raw || mustBeat(t, r).Reason != "meter_storage_failed" {
		t.Fatal("failed intent refresh admitted or diverged")
	}
}
