package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

type barrierCounterSource struct{ obs meter.Observation }

func (s *barrierCounterSource) Read(context.Context) (meter.Observation, error) { return s.obs, nil }

// The fake agent uses the real meter, durable usage store, snapshot type and
// atomic/fsynced publication order from agent-core 24. Only kernel reads and
// forwarding are fixtures; no JSON checkpoint or sequence is fabricated.
func agentBarrierBytes(t *testing.T, boot uuid.UUID) []byte {
	t.Helper()
	node, inc, dev, member := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := fixedNow()
	s, err := usagestore.Open(filepath.Join(t.TempDir(), "usage.db"), node, inc, usagestore.Options{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	src := &barrierCounterSource{obs: meter.Observation{
		BootID: boot.String(), Bindings: []enforce.Binding{{DeviceID: dev, MemberID: member, BindingRevision: 1}},
		Counters: map[string]enforce.CounterValue{
			meter.CounterName(dev, meter.DirUpload): {}, meter.CounterName(dev, meter.DirDownload): {},
		},
	}}
	m, err := meter.New(meter.Config{NodeID: node, IncarnationID: inc, Store: s, Source: src, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	now = now.Add(time.Second)
	src.obs.Counters[meter.CounterName(dev, meter.DirUpload)] = enforce.CounterValue{Bytes: 123, Packets: 1}
	if err := m.Barrier(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := m.FinalSample(t.Context()); err != nil {
		t.Fatal(err)
	}
	cp, err := s.ClosedCheckpoint(t.Context())
	if err != nil || cp.LastDurableEventSeq == 0 || cp.NewMeterEpochID != "" {
		t.Fatalf("real final sample: %+v %v", cp, err)
	}
	events, err := s.Unacked(t.Context(), 0, 10, 1<<20)
	if err != nil || len(events) != 1 || !bytes.Contains(events[0].Payload, []byte(`"upload_bytes":"123"`)) {
		t.Fatalf("durable nonzero usage missing: events=%d err=%v", len(events), err)
	}
	raw, err := json.Marshal(cp)
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

func TestUPD03_AgentBarrierSnapshotAndReceiptValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meter.checkpoint")
	raw := agentBarrierBytes(t, uuid.New())
	if err := fsutil.WriteFileAtomic(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	m := FileMeter{Path: path}
	a, err := m.FinalSample(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.FinalSample(t.Context())
	if err != nil || a != b {
		t.Fatal("repeated read changed the completed close", err)
	}
	cp, err := checkpointFromUsage(a, "")
	if err != nil || cp.LastDurableEventSeq != a.LastDurableEventSeq || cp.CentralAckedThrough != a.CentralAckedThrough || requiredValue(t, cp.ClosedMeterEpochId).String() != a.ClosedMeterEpochID || !cp.NewMeterEpochId.IsNull() {
		t.Fatalf("receipt conversion changed snapshot: %+v %v", cp, err)
	}
}

func TestAT25_FileMeterWaitFailsClosedWithoutCorrelation(t *testing.T) {
	for _, scenario := range []string{"older_boot_epoch", "missing", "late_within_bound", "after_bound", "withdrawn", "withdrawn_reblocked"} {
		t.Run(scenario, func(t *testing.T) {
			w := newWorld(t)
			t.Cleanup(func() { _ = w.eng.Close() })
			old := w.makeRelease("baseline", 1, nil)
			w.seed(old)
			next := w.makeRelease("candidate", 1, nil)
			path := stageInstaller(t, w, next)
			g := FileGate{Path: filepath.Join(w.layout.Journal, "forwarding.blocked")}
			m := FileMeter{Path: filepath.Join(w.layout.State, "meter.checkpoint"), Wait: 250 * time.Millisecond}
			w.eng.Gate, w.eng.Meter = g, m
			raw := agentBarrierBytes(t, uuid.New()) // distinct previous boot
			if scenario == "older_boot_epoch" {
				if err := fsutil.WriteFileAtomic(m.Path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan error, 1)
			var started time.Time
			w.eng.Hooks.AfterStage = func(stage string) error {
				if stage != string(updatejournal.StageBlocked) {
					return nil
				}
				started = time.Now()
				if scenario == "late_within_bound" || scenario == "withdrawn" || scenario == "withdrawn_reblocked" {
					go func() {
						time.Sleep(25 * time.Millisecond)
						if scenario == "late_within_bound" {
							done <- fsutil.WriteFileAtomic(m.Path, raw, 0600)
							return
						}
						if err := g.Unblock(context.Background()); err != nil {
							done <- err
							return
						}
						if scenario == "withdrawn_reblocked" {
							done <- g.Block(context.Background())
						} else {
							done <- nil
						}
					}()
				} else {
					done <- nil
				}
				return nil
			}
			err := w.eng.ApplyPending(t.Context(), path)
			if err == nil {
				t.Fatal("uncorrelated checkpoint allowed apply")
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if time.Since(started) > 2*time.Second {
				t.Fatal("wait exceeded bound")
			}
			if scenario == "after_bound" {
				if err := fsutil.WriteFileAtomic(m.Path, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			code, reason := apierr.CheckpointFailed, "meter_checkpoint_timeout"
			if scenario == "older_boot_epoch" || scenario == "late_within_bound" {
				reason = "meter_checkpoint_uncorrelated"
			}
			if scenario == "withdrawn" || scenario == "withdrawn_reblocked" {
				code, reason = apierr.StateConflict, "meter_barrier_withdrawn"
			}
			assertMeterFailure(t, w, old.id, code, reason)
			before, _ := w.eng.Journal.Outbox(t.Context())
			if err := w.eng.Close(); err != nil {
				t.Fatal(err)
			}
			w.eng.Journal = openJournal(t, w.layout.Journal)
			w.eng.Hooks = Hooks{}
			if err := w.eng.Recover(t.Context()); !errors.Is(err, ErrMembersBlocked) {
				t.Fatalf("sealed failure reopened: %v", err)
			}
			after, _ := w.eng.Journal.Outbox(t.Context())
			if len(after) != 1 || !bytes.Equal(before[0].Payload, after[0].Payload) {
				t.Fatal("restart resealed failed receipt")
			}
			assertMeterFailure(t, w, old.id, code, reason)
		})
	}
}

func assertMeterFailure(t *testing.T, w *world, baseline uuid.UUID, code apierr.Code, detail string) {
	t.Helper()
	w.mustCurrent(baseline)
	txn, err := w.eng.Journal.LastTxn(t.Context())
	if err != nil || txn.ErrorCode != string(code) || txn.ErrorDetail != detail || txn.Destructive {
		t.Fatalf("failure journal: %+v %v", txn, err)
	}
	res := w.oneResult()
	if res.State != api.UpdateResultStateFailed || requiredValue(t, res.Stage) != api.UpdateStageFinalSample || requiredValue(t, res.ErrorCode) != api.ErrorCode(code) || !res.UsageCheckpoint.IsNull() || res.MemberForwardingEnabled {
		t.Fatalf("discarded sample receipt: %+v", res)
	}
	if stops, starts := w.svc.counts(); stops != 0 || starts != 0 {
		t.Fatal("failed meter wait stopped/restarted services")
	}
	g, _ := w.eng.fileGate()
	if !g.Blocked() {
		t.Fatal("failure unblocked members")
	}
}

func TestAT25_FileMeterWaitRestartRechecksCurrentFile(t *testing.T) {
	for _, change := range []string{"missing", "replaced", "withdrawn", "new_boot", "expired"} {
		t.Run(change, func(t *testing.T) {
			w := newWorld(t)
			t.Cleanup(func() { _ = w.eng.Close() })
			old := w.makeRelease("baseline", 1, nil)
			w.seed(old)
			next := w.makeRelease("candidate", 1, nil)
			path := stageInstaller(t, w, next)
			g := FileGate{Path: filepath.Join(w.layout.Journal, "forwarding.blocked")}
			m := FileMeter{Path: filepath.Join(w.layout.State, "meter.checkpoint"), Wait: 250 * time.Millisecond}
			w.eng.Gate, w.eng.Meter = g, m
			raw := agentBarrierBytes(t, w.node.BootID)
			w.eng.Hooks.AfterStage = func(stage string) error {
				if stage == string(updatejournal.StageBlocked) {
					if err := fsutil.WriteFileAtomic(m.Path, raw, 0600); err != nil {
						return err
					}
					// A valid pre-restart read must not be cached as permission.
					if _, err := m.FinalSample(t.Context()); err != nil {
						return err
					}
					return ErrCrashed
				}
				return nil
			}
			if err := w.eng.ApplyPending(t.Context(), path); !errors.Is(err, ErrCrashed) {
				t.Fatal(err)
			}
			txn, _ := w.eng.Journal.ActiveTxn(t.Context())
			notePath := filepath.Join(w.layout.Journal, meterWaitPrefix+txn.TxnID)
			noteRaw := fileBytes(t, notePath)
			var note meterWait
			if err := json.Unmarshal(noteRaw, &note); err != nil {
				t.Fatal(err)
			}
			if err := w.eng.Close(); err != nil {
				t.Fatal(err)
			}
			// Recreate the engine and file adapters as well as the SQLite handle.
			nextEngine := *w.eng
			nextEngine.Journal = openJournal(t, w.layout.Journal)
			nextEngine.Hooks = Hooks{}
			nextEngine.Meter = FileMeter{Path: m.Path, Wait: time.Minute} // must not renew deadline
			w.eng = &nextEngine
			code, detail := apierr.CheckpointFailed, "meter_checkpoint_timeout"
			switch change {
			case "missing":
				if err := os.Remove(m.Path); err != nil {
					t.Fatal(err)
				}
			case "replaced":
				if err := fsutil.WriteFileAtomic(m.Path, agentBarrierBytes(t, uuid.New()), 0600); err != nil {
					t.Fatal(err)
				}
				detail = "meter_checkpoint_uncorrelated"
			case "withdrawn":
				if err := g.Unblock(t.Context()); err != nil {
					t.Fatal(err)
				}
				code, detail = apierr.StateConflict, "meter_barrier_withdrawn"
			case "new_boot":
				w.eng.Node.BootID = uuid.New()
				detail = "meter_wait_invalid"
			case "expired":
				time.Sleep(time.Until(note.Deadline) + time.Millisecond)
			}
			start := time.Now()
			if err := w.eng.Recover(t.Context()); !errors.Is(err, ErrMembersBlocked) {
				t.Fatal(err)
			}
			if time.Since(start) > time.Second {
				t.Fatal("restart renewed the original wait deadline")
			}
			assertMeterFailure(t, w, old.id, code, detail)
		})
	}
}

func TestUPD02_MeterWaitConfiguration(t *testing.T) {
	for _, tc := range []struct {
		ms   int
		want time.Duration
		bad  bool
	}{
		{0, 30 * time.Second, false}, {1, time.Millisecond, false}, {300000, 5 * time.Minute, false}, {-1, 0, true}, {300001, 0, true},
	} {
		got, err := meterWaitDuration(tc.ms)
		if (err != nil) != tc.bad || got != tc.want {
			t.Fatalf("wait %d: %s %v", tc.ms, got, err)
		}
	}
}
