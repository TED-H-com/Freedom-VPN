package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// Real enforcement producer over an emulated kernel. The replay clock drives
// Engine decisions only; TimingEvidence is always sampled by enforce itself.
type replayPlane struct {
	*memPlane
	x         renderedExecutor
	model     enforce.Model
	failApply bool
}

func (p *replayPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	p.model = kernelClockModel(m)
	p.x.model = p.model
	if p.failApply {
		p.x.failVerify = true
	}
	res, err := (SystemPlane{Executor: &p.x}).Apply(ctx, p.model, prev)
	if err == nil {
		_, _ = p.memPlane.Apply(ctx, m, prev)
	}
	return res, err
}

func (p *replayPlane) VerifyServing(ctx context.Context, m enforce.Model, a map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	expected := p.model
	expected.Generation = m.Generation
	return (SystemPlane{Executor: &p.x}).VerifyServing(ctx, expected, a)
}

func (p *replayPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	_ = p.memPlane.Baseline(ctx, dp)
	return (SystemPlane{Executor: &p.x}).Baseline(ctx, dp)
}

func measurements(t *testing.T, path string) []Measurement {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []Measurement
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		var m Measurement
		if err := json.Unmarshal(line, &m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	return out
}

func TestAT13_ISOF603_ExactProducerReplay955To961(t *testing.T) {
	var trace struct {
		Deadline time.Duration `json:"deadline_ns,string"`
		Rows     []Measurement `json:"rows"`
	}
	if err := json.Unmarshal(readFile(t, "testdata/f28-replay.json"), &trace); err != nil {
		t.Fatal(err)
	}
	for _, failRepair := range []bool{false, true} {
		t.Run(map[bool]string{false: "same_lease_repair", true: "repair_fails_closed"}[failRepair], func(t *testing.T) {
			r := newRig(t)
			// Exact captured request anchor and 60s duration. No captured identity
			// or signing material: fake control signs this test's own generation.
			r.clock.Set(time.Date(2026, 10, 31, 15, 7, 17, 994901292, time.UTC), trace.Deadline-time.Minute)
			alignControlClock(r)
			r.srv.SetMaxOfflineSeconds(60)
			r.srv.StickyLease(false)
			r.srv.SetFixture(generationFixture(30, oneBinding(r.clock.Wall())))
			enrollBoot(t, r)
			stepOK(t, r)
			if !r.eng.serving {
				t.Fatal("replay setup", r.eng.decision.Reasons)
			}
			r.manualUsage(t)
			r.srv.StickyLease(true)
			lease, anchor := r.eng.lease.LeaseId, r.eng.leaseMono
			p := &replayPlane{memPlane: r.plane}
			r.eng.plane = p
			r.eng.cfg.TestMode = true
			r.eng.cfg.TestTimingPath = filepath.Join(r.dir, "operations.jsonl")
			for _, row := range trace.Rows {
				r.clock.Set(row.Anchor.Wall, row.Anchor.MonoBefore)
				switch row.Sequence {
				case 955:
					r.eng.evaluate(t.Context())
					if err := r.eng.enforce(t.Context(), r.eng.decision); err != nil {
						t.Fatal(err)
					}
				case 956, 957:
					if !r.eng.verifyCurrentServing(t.Context()) {
						t.Fatal("captured good read failed", row.Sequence)
					}
				case 958:
					// Inject the observed generation-0 state. This intentionally
					// does not identify the unknown writer in the historical run.
					if err := (SystemPlane{Executor: &p.x}).Baseline(t.Context(), p.model.Datapath); err != nil {
						t.Fatal(err)
					}
					p.failApply = failRepair
					before := p.baselines
					if r.eng.verifyCurrentServing(t.Context()) {
						t.Fatal("mismatch reported as success")
					}
					if !failRepair && (!r.eng.serving || p.baselines != before || p.last.Generation != 30) {
						t.Fatal("generation mismatch added baseline/wait instead of immediate same-generation repair")
					}
					if failRepair && (r.eng.serving || p.baselines == before) {
						t.Fatal("failed repair did not close")
					}
					pending, err := r.eng.store.Pending(t.Context())
					if err != nil {
						t.Fatal(err)
					}
					found := false
					for _, item := range pending {
						var rep api.ObservationReport
						if json.Unmarshal(item.Payload, &rep) != nil {
							t.Fatal("invalid report")
						}
						for _, c := range rep.Checks {
							code, _ := c.ErrorCode.Get()
							if c.Name == api.NodeCheckNameLease && code == api.ErrorCodeGENERATIONMISMATCH && !rep.MemberForwardingEnabled {
								found = true
							}
						}
					}
					if !found {
						t.Fatal("repair erased failed verifier wire report")
					}
				case 959:
					r.eng.finishServingStep(t.Context())
				case 960:
					if err := r.eng.syncLease(t.Context()); err != nil {
						t.Fatal(err)
					}
				case 961:
					if !failRepair && !r.eng.serving {
						t.Fatal("still waiting for historical recovery slot")
					}
				}
				if r.eng.lease.LeaseId != lease || r.eng.leaseMono != anchor || anchor+time.Minute != trace.Deadline {
					t.Fatal("replay extended or replaced lease")
				}
			}
			var failed, applied bool
			for _, m := range measurements(t, r.eng.cfg.TestTimingPath) {
				if m.Timing == nil {
					continue
				}
				if m.Timing.Start <= 0 || m.Timing.End < m.Timing.Start || m.Timing.End > m.Anchor.MonoBefore {
					t.Fatal("fabricated or unordered producer timing")
				}
				failed = failed || m.Timing.Operation == "verify" && !m.Timing.Succeeded
				applied = applied || m.Timing.Operation == "apply" && m.Timing.Succeeded
			}
			if !failed || !applied {
				t.Fatal("missing successful Apply or failed producer read")
			}
			r.clock.Set(trace.Rows[len(trace.Rows)-1].Anchor.Wall.Add(trace.Deadline-r.clock.Mono()), trace.Deadline)
			r.eng.finishServingStep(t.Context())
			if r.eng.serving {
				t.Fatal("deadline extended")
			}
		})
	}
}

func TestAT13_ISOF603_SameBootRestartKeepsRulesAndDeadline(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	deadline, lease, anchor := r.eng.policyExpires, r.eng.lease.LeaseId, r.eng.leaseMono
	baselines, applies := r.plane.baselines, r.plane.applies
	head, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	r.clock.Advance(time.Second)
	restartEngine(t, r, r.plane)
	if !r.eng.serving || r.plane.baselines != baselines || r.plane.applies != applies || r.eng.policyExpires != deadline || r.eng.lease.LeaseId != lease || r.eng.leaseMono != anchor {
		t.Fatal("restart changed rules/lease/deadline")
	}
	after, err := r.eng.usage.Head(t.Context())
	if err != nil || after.OpenEpoch != head.OpenEpoch {
		t.Fatal("restart lost meter continuity", err)
	}
}

func TestAT13_ISOF603_ControlStallKeepsVerifiedRules(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.cfg.WatchdogStaleSeconds = 3
	r.srv.SetDelay(10 * time.Second)
	defer r.srv.SetDelay(0)
	before, deadline := r.plane.applies, r.eng.policyExpires
	start := time.Now()
	stepOK(t, r)
	if time.Since(start) >= 3*time.Second || !r.eng.serving || r.plane.applies != before || r.eng.policyExpires != deadline {
		t.Fatal("control stall rewrote valid rules or starved heartbeat")
	}
	beat := mustBeat(t, r)
	if !beat.Serving || beat.PolicyExpires != deadline {
		t.Fatal("control timeout lost verified heartbeat")
	}
}

func TestAT13_ISOF603_RestartInvalidAuthorizationCloses(t *testing.T) {
	for _, reason := range []string{"reboot", "expired", "clock_jump", "frozen", "barrier"} {
		t.Run(reason, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			if err := r.eng.Close(); err != nil {
				t.Fatal(err)
			}
			switch reason {
			case "reboot":
				if err := os.WriteFile(r.eng.cfg.BootIDFile, []byte(uuid.NewString()), 0600); err != nil {
					t.Fatal(err)
				}
			case "expired":
				r.clock.Advance(24 * time.Hour)
			case "clock_jump":
				r.clock.JumpWall(-time.Hour)
			case "frozen":
				if err := writeAtomic(filepath.Join(r.eng.cfg.StateDir, "identity-freeze.intent"), []byte(r.eng.incID.String()+"\n"), 0600); err != nil {
					t.Fatal(err)
				}
			case "barrier":
				writeBarrier(t, r, uuid.NewString())
			}
			next, err := Start(Options{Config: r.eng.cfg, Clock: r.clock, Plane: r.plane, MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = next.Close() })
			if err := next.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			if next.serving || len(r.plane.last.Bindings) != 0 {
				t.Fatal("invalid restart retained forwarding")
			}
		})
	}
}
