package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/fakecontrol"
	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/meter"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func TestISO_F6_ForeignStillPresentWithholdsNewACK(t *testing.T) {
	for _, newer := range []bool{false, true} {
		t.Run(map[bool]string{false: "same_generation", true: "new_generation"}[newer], func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			p := &revocationPlane{memPlane: r.plane}
			r.eng.plane = p
			stepOK(t, r)
			p.ex.foreignTC = true
			stepOK(t, r)
			if r.eng.containment == nil || len(r.eng.containment.TC) == 0 {
				t.Fatal("missing rejected read")
			}
			at := r.eng.containment.ReadAt
			if newer {
				r.srv.SetFixture(generationFixture(3, oneBinding(r.clock.Wall())))
			}
			for range 3 {
				r.clock.Advance(time.Second)
				stepOK(t, r)
			}
			if r.eng.serving || r.eng.appliedGen != 2 {
				t.Fatal("foreign state served or ACKed new generation")
			}
			p.ex.foreignTC = false
			for range 3 {
				r.clock.Advance(time.Second)
				stepOK(t, r)
			}
			if !r.eng.serving || r.eng.applyError != nil {
				t.Fatalf("clean retry failed: %+v", r.eng.decision.Errors)
			}
			rep := mustReport(t, r, t.Context())
			found := false
			for _, row := range rep.Errors {
				if row.Stage == "tc_evidence" {
					found = true
					if !row.ObservedAt.Equal(at) {
						t.Fatal("historical rejection relabeled as current")
					}
				}
			}
			if !found || r.eng.containment.Generation != 2 {
				t.Fatal("recovery discarded rejection provenance")
			}
		})
	}
}

type evidencePlane struct {
	*memPlane
	ev  enforce.ServingEvidence
	err error
}

func (p *evidencePlane) VerifyServing(context.Context, enforce.Model, map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	return p.ev, p.err
}

func TestISO_F6_ObservationEvidenceRevision13(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	p := &evidencePlane{memPlane: r.plane, ev: enforce.ServingEvidence{TCSummary: enforce.TCSummary{Generation: 2, Phase: "verify", Reads: 8, Objects: 4, SHA256: strings.Repeat("a", 64)}}}
	r.eng.plane = p
	if !r.eng.verifyServingModel(t.Context(), *r.eng.appliedModel, r.eng.appliedAssignments) {
		t.Fatal("clean read failed")
	}
	at := r.clock.Wall()
	r.clock.Advance(time.Second)
	rep := mustReport(t, r, t.Context())
	requireObservationSchema(t, rep)
	q, _ := checkBy(rep.Checks, api.NodeCheckNameQos)
	if q.Status != api.HealthCheckStatusPassed || q.Detail == nil || !strings.Contains(*q.Detail, at.Format(time.RFC3339Nano)) || !strings.Contains(*q.Detail, `"generation":"2"`) || len(*q.Detail) > 500 || len(rep.Errors) != 0 {
		t.Fatalf("clean evidence: %+v errors=%v", q, rep.Errors)
	}
	p.ev.TC = []enforce.TCReadEvidence{{Interface: "fv-fh", Attachment: "ingress", Objects: []enforce.TCObject{{Kind: "matchall", Actions: []string{"mirred"}}}}}
	p.err = &enforce.CheckError{Reason: enforce.ReasonTCForeignFilter}
	if r.eng.verifyServingModel(t.Context(), *r.eng.appliedModel, r.eng.appliedAssignments) {
		t.Fatal("rejected read passed")
	}
	at = r.clock.Wall()
	rep = mustReport(t, r, t.Context())
	requireObservationSchema(t, rep)
	q, _ = checkBy(rep.Checks, api.NodeCheckNameQos)
	if q.Status != api.HealthCheckStatusFailed || codeOf(t, q) != api.ErrorCodeTCFOREIGNFILTER || !strings.Contains(*q.Detail, "verified=false") {
		t.Fatalf("rejection check: %+v", q)
	}
	// A rollback audit is not current serving verification and cannot erase
	// the deciding rejected object, candidate, timestamp or check failure.
	p.ev = enforce.ServingEvidence{}
	p.err = &enforce.CheckError{Reason: enforce.ReasonGenerationMismatch}
	r.clock.Advance(5 * time.Second)
	r.eng.auditBlocked(t.Context())
	post := mustReport(t, r, t.Context())
	requireObservationSchema(t, post)
	q, _ = checkBy(post.Checks, api.NodeCheckNameQos)
	if !strings.Contains(*q.Detail, at.Format(time.RFC3339Nano)) || q.Status != api.HealthCheckStatusFailed {
		t.Fatal("rollback replaced rejection")
	}
	for _, row := range post.Errors {
		if row.Stage == "tc_evidence" && !row.ObservedAt.Equal(at) {
			t.Fatal("rollback changed historical timestamp")
		}
	}
}

type evidenceClockExecutor struct {
	renderedExecutor
	advance func(time.Duration)
}

func (x *evidenceClockExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	x.advance(time.Millisecond)
	return x.renderedExecutor.Output(ctx, name, args...)
}

func TestISO_F6_ApplyRejectionTimestampPrecedesRollback(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	x := &evidenceClockExecutor{advance: r.clock.Advance}
	x.model = *r.eng.appliedModel
	p := SystemPlane{Executor: x}
	if _, err := p.Apply(t.Context(), x.model, nil); err != nil {
		t.Fatal(err)
	}
	x.foreignTC = true
	timing := &applyReadTiming{now: r.clock.Wall}
	res, err := p.Apply(context.WithValue(t.Context(), applyReadKey{}, timing), x.model, nil)
	if err == nil || res.Verified || res.Evidence.TCSummary.Phase != "pre_apply" || timing.at.IsZero() || !timing.at.Before(r.clock.Wall()) {
		t.Fatalf("missing pre-rollback read: result=%+v at=%s err=%v", res.Evidence.TCSummary, timing.at, err)
	}
	r.eng.retainServingRead(res.Evidence, x.model.Generation, timing.at, "apply", false)
	r.eng.recordApplyQuarantine(res, err)
	r.eng.rejectServing(t.Context(), x.model, res.Evidence, err)
	r.clock.Advance(time.Second)
	rep := mustReport(t, r, t.Context())
	requireObservationSchema(t, rep)
	q, _ := checkBy(rep.Checks, api.NodeCheckNameQos)
	if q.Status != api.HealthCheckStatusFailed || !strings.Contains(*q.Detail, "source=apply") || !strings.Contains(*q.Detail, timing.at.Format(time.RFC3339Nano)) {
		t.Fatal("cached Apply mislabeled", q)
	}
	for _, row := range rep.Errors {
		if row.Stage == "tc_evidence" && !row.ObservedAt.Equal(timing.at) {
			t.Fatal("post-rollback read time substituted")
		}
	}
}

type barrierSource struct {
	obs      meter.Observation
	err      error
	bindings func() []enforce.Binding
}

func (s *barrierSource) Read(context.Context) (meter.Observation, error) {
	obs := s.obs
	if s.bindings != nil {
		obs.Bindings = s.bindings()
	}
	return obs, s.err
}

type admissionPlane struct {
	Plane
	admit     func()
	verifyErr error
	applyErr  error
}

func (p admissionPlane) Apply(ctx context.Context, model enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	res, err := p.Plane.Apply(ctx, model, prev)
	if err == nil {
		p.admit()
		if p.applyErr != nil {
			res.Verified = false
			res.Evidence.TCSummary.Phase = "verify"
			return res, p.applyErr
		}
	}
	return res, err
}

func (p admissionPlane) VerifyServing(ctx context.Context, model enforce.Model, assignments map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	if p.verifyErr != nil {
		return enforce.ServingEvidence{}, p.verifyErr
	}
	return p.Plane.VerifyServing(ctx, model, assignments)
}

func TestMTR03_FirstBytesAfterFinalBarrierAreDurable(t *testing.T) {
	for _, scenario := range []string{"first_packet", "read_failure_then_recovery", "post_apply_verification_failure", "apply_verification_rollback"} {
		t.Run(scenario, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			r.eng.meter.Close()
			name := meter.CounterName(r.eng.appliedModel.Bindings[0].DeviceID, meter.DirUpload)
			src := &barrierSource{obs: meter.Observation{BootID: r.eng.bootID.String(), Bindings: r.eng.appliedModel.Bindings, Counters: map[string]enforce.CounterValue{name: {}}}}
			var err error
			r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src})
			if err != nil {
				t.Fatal(err)
			}
			r.eng.withdraw(t.Context())
			closed, err := r.eng.usage.ClosedCheckpoint(t.Context())
			if err != nil || !r.eng.finalDone {
				t.Fatal("missing real closing barrier", err)
			}
			// A restarted process has no prior in-memory attribution. Match
			// the production source, which reads the engine's binding snapshot.
			src.bindings = r.eng.meterBindings
			r.eng.setMeterBindings(nil)
			var verifyErr error
			if scenario == "post_apply_verification_failure" {
				verifyErr = os.ErrPermission
			}
			var applyErr error
			if scenario == "apply_verification_rollback" {
				applyErr = os.ErrPermission
			}
			wantServe := verifyErr == nil && applyErr == nil
			admitted := 0
			r.eng.plane = admissionPlane{Plane: r.plane, verifyErr: verifyErr, applyErr: applyErr, admit: func() {
				head, err := r.eng.usage.Head(t.Context())
				if err != nil || head.OpenEpoch == uuid.Nil || head.OpenEpoch.String() == closed.ClosedMeterEpochID {
					t.Fatal("forwarding opened before the next epoch was durable", err)
				}
				if _, err := os.Stat(r.eng.cfg.MeterCheckpoint); !os.IsNotExist(err) {
					t.Fatal("forwarding opened with a stale updater checkpoint", err)
				}
				// The very first bytes arrive inside Apply, before the sampler
				// resumes. They must not become the next epoch's baseline.
				admitted++
				src.obs.Counters[name] = enforce.CounterValue{Packets: 1, Bytes: 4096}
			}}
			if scenario == "read_failure_then_recovery" {
				src.err = os.ErrPermission
				if err := r.eng.enforce(t.Context(), Decide(r.eng.input())); err == nil {
					t.Fatal("unreadable meter permitted admission")
				}
				if admitted != 0 || r.eng.serving {
					t.Fatal("unreadable meter forwarded")
				}
				src.err = nil
				// A successful closing retry clears the meter read failure.
				r.eng.readMeter(t.Context())
				if err := r.eng.enforce(t.Context(), Decide(r.eng.input())); err != nil {
					t.Fatal(err)
				}
				r.eng.readMeter(t.Context())
			}
			if err := r.eng.enforce(t.Context(), Decide(r.eng.input())); err != nil {
				t.Fatal(err)
			}
			if admitted != 1 || r.eng.serving != wantServe {
				t.Fatalf("did not resume once: admitted=%d serving=%v reasons=%v", admitted, r.eng.serving, r.eng.decision.Reasons)
			}
			if wantServe {
				if err := r.eng.meter.Sample(t.Context()); err != nil {
					t.Fatal(err)
				}
			} else if !r.eng.finalDone || r.eng.appliedGen != 2 {
				t.Fatal("verification failure did not finish its real closing barrier")
			}
			head, _ := r.eng.usage.Head(t.Context())
			var bytes uint64
			for seq := uint64(closed.LastDurableEventSeq) + 1; seq < head.NextSeq; seq++ {
				row, err := r.eng.usage.Event(t.Context(), seq)
				if err != nil {
					t.Fatal(err)
				}
				var event api.UsageEvent
				if err := json.Unmarshal(row.Payload, &event); err != nil {
					t.Fatal(err)
				}
				for _, rec := range event.Records {
					bytes += uint64(rec.UploadBytes)
				}
			}
			if bytes != 4096 {
				t.Fatalf("first admitted bytes missing or duplicated: %d", bytes)
			}
		})
	}
}

func TestMTR03_GenerationRejectionRetainsGapBeforeFinalSample(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.meter.Close()
	name := meter.CounterName(r.eng.appliedModel.Bindings[0].DeviceID, meter.DirUpload)
	state := enforce.State{Table: true, Chains: map[string]bool{"forward": true}, Generation: 2, Counters: map[string]enforce.CounterValue{name: {}}}
	k := &kernelSource{boot: r.eng.bootID.String(), bindings: r.eng.meterBindings, expect: r.eng.expectServing.Load,
		read: func(context.Context) (enforce.State, error) { return state, nil }}
	var err error
	r.eng.kernel = k
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: k})
	if err != nil {
		t.Fatal(err)
	}
	state.Counters[name] = enforce.CounterValue{Packets: 10, Bytes: 4096}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	before, _ := r.eng.usage.Head(t.Context())
	state.Generation = 0 // The live read wins the race against the sampler.
	r.eng.rejectServing(t.Context(), *r.eng.appliedModel, enforce.ServingEvidence{Counters: state.Counters}, &enforce.CheckError{Reason: enforce.ReasonGenerationMismatch})
	after, _ := r.eng.usage.Head(t.Context())
	gap := false
	for seq := before.NextSeq; seq < after.NextSeq; seq++ {
		row, err := r.eng.usage.Event(t.Context(), seq)
		if err != nil {
			t.Fatal(err)
		}
		var event api.UsageEvent
		if err := json.Unmarshal(row.Payload, &event); err != nil {
			t.Fatal(err)
		}
		if event.Gap != nil && event.Gap.Reason == api.UsageGapReasonCounterDiscontinuity {
			gap = true
		}
	}
	if !gap || r.eng.serving || !r.eng.finalDone || after.OpenEpoch != uuid.Nil {
		t.Fatal("withdrawal erased the real discontinuity or left the barrier incomplete")
	}
	if _, err := (updater.FileMeter{Path: r.eng.cfg.MeterCheckpoint}).FinalSample(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestMTR03_UPD05_DurableFileMeterBarrier(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	// Drive the production sampler with a controlled admission-counter read.
	// The final bytes must create a real durable event before publication.
	r.eng.meter.Close()
	name := meter.CounterName(r.eng.appliedModel.Bindings[0].DeviceID, meter.DirUpload)
	src := &barrierSource{obs: meter.Observation{BootID: r.eng.bootID.String(), Bindings: r.eng.appliedModel.Bindings, Counters: map[string]enforce.CounterValue{name: {}}}}
	var err error
	r.eng.meter, err = meter.New(meter.Config{NodeID: r.eng.nodeID, IncarnationID: r.eng.incID, Store: r.eng.usage, Source: src})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.eng.meter.Sample(t.Context()); err != nil {
		t.Fatal(err)
	}
	head, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (updater.FileMeter{Path: r.eng.cfg.MeterCheckpoint}).FinalSample(t.Context()); err == nil {
		t.Fatal("open epoch exposed checkpoint")
	}
	src.obs.Counters[name] = enforce.CounterValue{Packets: 1, Bytes: 4096}
	r.eng.handoff(t.Context())
	cp, err := (updater.FileMeter{Path: r.eng.cfg.MeterCheckpoint}).FinalSample(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	after, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if cp.ClosedMeterEpochID != head.OpenEpoch.String() || after.OpenEpoch != uuid.Nil || cp.LastDurableEventSeq != decstr.U64(after.NextSeq-1) || cp.CentralAckedThrough != decstr.U64(after.AckedThrough) || cp.NewMeterEpochID != "" {
		t.Fatalf("not real cursor: %+v head=%+v", cp, after)
	}
	if after.NextSeq <= head.NextSeq {
		t.Fatal("final counter delta was not durably recorded")
	}
	raw := readFile(t, r.eng.cfg.MeterCheckpoint)
	r.eng.handoff(t.Context())
	if string(raw) != string(readFile(t, r.eng.cfg.MeterCheckpoint)) {
		t.Fatal("repeated barrier changed checkpoint")
	}
	if err := r.eng.invalidateMeterCheckpoint(); err != nil {
		t.Fatal(err)
	}
	if _, err := (updater.FileMeter{Path: r.eng.cfg.MeterCheckpoint}).FinalSample(t.Context()); err == nil {
		t.Fatal("stale checkpoint survived resume")
	}
	// A failed destination write leaves the barrier retryable; no made-up
	// epoch or sequence is used to make FileMeter pass.
	r.eng.finalDone = false
	r.eng.cfg.MeterCheckpoint = filepath.Join(t.TempDir(), "directory")
	if err := os.Mkdir(r.eng.cfg.MeterCheckpoint, 0o700); err != nil {
		t.Fatal(err)
	}
	r.eng.handoff(t.Context())
	if r.eng.finalDone || !r.eng.usagePersist {
		t.Fatal("checkpoint write failure marked barrier complete")
	}
}

func TestUPD05_CandidateIdentityAndHealth(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	old := r.eng.cfg.ReleaseUUID()
	next := uuid.New()
	root := t.TempDir()
	dir := filepath.Join(root, next.String())
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(dir, "fvpn-agent")
	if err := os.WriteFile(executable, []byte("TEST ONLY executable identity"), 0o700); err != nil {
		t.Fatal(err)
	}
	r.eng.cfg.CurrentReleasePath = filepath.Join(root, "current")
	if err := os.Symlink(dir, r.eng.cfg.CurrentReleasePath); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"signed": map[string]any{"release_id": next.String(), "components": map[string]string{"node_agent": "2.0.0-test"}}})
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := canonjson.HashHex(raw)
	journal, err := updatejournal.Open(r.eng.cfg.JournalDir, updatejournal.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	if err := journal.PutRelease(t.Context(), updatejournal.Release{ReleaseID: next.String(), ManifestEnvelope: raw, ManifestSHA256: hash, StateSchemaTo: 1}); err != nil {
		t.Fatal(err)
	}
	if err := journal.SetPointers(t.Context(), updatejournal.Pointers{CurrentRelease: next.String()}); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.loadActiveRelease(t.Context(), executable); err != nil {
		t.Fatal(err)
	}
	if r.eng.cfg.ReleaseUUID() != next || r.eng.cfg.AgentRelease != "2.0.0-test" {
		t.Fatal("static config won over candidate")
	}
	issueChallenge(t, r, func(in *healthattest.Input) { in.ReleaseID = old.String() })
	if _, ok := r.eng.matchingChallenge(); ok {
		t.Fatal("accepted old release challenge")
	}
	ch := issueChallenge(t, r, nil)
	r.eng.prober = &verProbe{version: "0.76.2"}
	r.eng.refreshProbe(t.Context())
	r.eng.answerHealth(t.Context())
	att := requireAttest(t, r, ch)
	if att.ReleaseID != next.String() || att.AgentVersion != "2.0.0-test" {
		t.Fatal("wrong candidate answer")
	}
	foreign := filepath.Join(root, "old-agent")
	if err := os.WriteFile(foreign, []byte("TEST ONLY old process"), 0o700); err != nil {
		t.Fatal(err)
	}
	if r.eng.loadActiveRelease(t.Context(), foreign) == nil {
		t.Fatal("old executable impersonated switched current")
	}
}

func TestD087_ObservationRetryAfterCoalescesAndRetainsTransition(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.cfg.ObservationSeconds = 1
	r.srv.SetObservationRetryAfter(1, 10)
	r.eng.lastObs = time.Time{}
	stepOK(t, r)
	posts := r.srv.ObservationPosts()
	pending, err := r.eng.store.Pending(t.Context())
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending=%d err=%v", len(pending), err)
	}
	deciding := pending[0].ReportID
	for range 9 {
		r.clock.Advance(time.Second)
		stepOK(t, r)
	}
	if r.srv.ObservationPosts() != posts {
		t.Fatal("flush bypassed Retry-After")
	}
	pending, _ = r.eng.store.Pending(t.Context())
	if len(pending) != 1 || pending[0].ReportID != deciding {
		t.Fatal("dropped deciding body or failed to coalesce")
	}
	r.clock.Advance(time.Second)
	stepOK(t, r)
	if r.srv.ObservationPosts() <= posts {
		t.Fatal("retry did not resume")
	}
	// No serving transition occurs here: a grant-less lease must force its
	// ready/passed provisional observation despite a long report interval.
	r.eng.cfg.ObservationSeconds = 3600
	r.srv.SetMemberGrantsReady(false)
	r.eng.leaseFresh = false
	stepOK(t, r)
	rep, _ := r.srv.LastObservation()
	if rep.ProvisioningState != api.ProvisioningStateReady || rep.SafetyStatus != api.SafetyStatusPassed || len(rep.Errors) != 1 || rep.Errors[0].Code != api.ErrorCodeAWAITINGMEMBERGRANTS {
		t.Fatalf("provisional transition: %+v", rep.Errors)
	}
}

func TestD088_FailedFreezeAcceptanceStaysReplayable(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	db, err := sql.Open("sqlite", "file:"+r.eng.cfg.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TRIGGER fail_freeze BEFORE INSERT ON meta WHEN NEW.key = 'identity_frozen' BEGIN SELECT RAISE(FAIL, 'TEST ONLY freeze failure'); END`); err != nil {
		t.Fatal(err)
	}
	frozen := true
	r.srv.SetIdentityFrozen(&frozen)
	r.eng.lastObs = time.Time{}
	stepOK(t, r)
	rows, err := r.eng.store.Pending(t.Context())
	if err != nil || len(rows) == 0 || !r.eng.identityFrozen || r.eng.freezeDurable || r.eng.serving {
		t.Fatalf("failed freeze not pending: rows=%d err=%v", len(rows), err)
	}
	if _, err := db.Exec(`DROP TRIGGER fail_freeze`); err != nil {
		t.Fatal(err)
	}
	stepOK(t, r)
	if !r.eng.freezeDurable {
		t.Fatal("Step did not retry durable freeze")
	}
	restartEngine(t, r, r.plane)
	stepOK(t, r)
	if !r.eng.identityFrozen || r.eng.serving {
		t.Fatal("restart lost freeze")
	}
}

func TestD088_FreezeIntentRetainsIncarnationBeforeBoot(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	db, err := sql.Open("sqlite", "file:"+r.eng.cfg.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TRIGGER fail_freeze BEFORE INSERT ON meta WHEN NEW.key = 'identity_frozen' BEGIN SELECT RAISE(FAIL, 'TEST ONLY freeze failure'); END`); err != nil {
		t.Fatal(err)
	}
	r.eng.freezeIdentity(t.Context())
	cfg, inc, boot := r.eng.cfg, r.eng.incID, r.eng.bootID
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		e, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, MeterSource: staticSource{boot: boot.String()}, Prober: passProbe{}})
		if err != nil {
			t.Fatal(err)
		}
		raw, readErr := os.ReadFile(filepath.Join(cfg.StateDir, "identity-freeze.intent"))
		if !e.identityFrozen || e.freezeDurable || readErr != nil || strings.TrimSpace(string(raw)) != inc.String() {
			_ = e.Close()
			t.Fatal("startup replaced the failed-persist intent with another incarnation", readErr)
		}
		// Simulate another process exit before Boot can retry persistence.
		if err := e.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestD088_LostFreezeResponseBlocksRestartUntilReplay(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	frozen := true
	r.srv.SetIdentityFrozen(&frozen)
	r.eng.lastObs = time.Time{}
	if err := r.eng.observe(t.Context()); err != nil {
		t.Fatal(err)
	}
	rows, err := r.eng.store.Pending(t.Context())
	if err != nil || len(rows) != 1 {
		t.Fatal("pending report", err)
	}
	row := rows[0]
	var rep api.ObservationReport
	if err := json.Unmarshal(row.Payload, &rep); err != nil {
		t.Fatal(err)
	}
	if err := r.eng.store.BumpAttempt(t.Context(), row.ReportID, "in_flight"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.eng.client.Observe(t.Context(), row.ReportID, rep); err != nil {
		t.Fatal(err)
	}
	// Lose the acceptance before any sidecar or meta write, just as a process
	// crash or a full disk could. The pending attempted body is the intent.
	restartEngine(t, r, r.plane)
	if !r.eng.freezeUncertain || r.eng.serving || r.eng.input().StorageOK {
		t.Fatal("restart authorized before uncertain response replay")
	}
	stepOK(t, r)
	if !r.eng.identityFrozen || !r.eng.freezeDurable || r.eng.serving {
		t.Fatal("replay failed to recover freeze")
	}
}

func TestUPD05_UpgradeHealthFixtureUsesSignedDesiredAndRealTCP(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	target := api.EgressProbeTarget{Address: "127.0.0.1", Port: ln.Addr().(*net.TCPAddr).Port, Protocol: api.EgressProbeTargetProtocolTcp}
	r.srv.SetFixture(fakecontrol.UpgradeHealthFixture(r.eng.cfg.ReleaseUUID(), target))
	r.eng.prober = netbirdProber{run: func(context.Context) ([]byte, error) {
		return []byte(`{"daemonVersion":"0.76.2","management":{"connected":true},"relays":{"total":1,"available":1}}`), nil
	}}
	enrollBoot(t, r)
	stepOK(t, r)
	ch := issueChallenge(t, r, nil)
	r.clock.Advance(time.Second)
	stepOK(t, r)
	att := requireAttest(t, r, ch)
	if att.Checks.TunnelProbe.Status != api.HealthCheckStatusPassed || r.eng.desired == nil || r.eng.desired.PublicationIntent != api.PublicationIntentUnpublished || r.eng.serving {
		t.Fatal("fixture did not run genuine signed unpublished health")
	}
	_ = ln.Close()
	stepOK(t, r)
	att = requireAttest(t, r, ch)
	if att.Checks.TunnelProbe.Status != api.HealthCheckStatusFailed {
		t.Fatal("unreachable probe was fabricated as success")
	}
}
