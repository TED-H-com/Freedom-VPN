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

func TestMTR03_UPD05_DurableFileMeterBarrier(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	head, err := r.eng.usage.Head(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (updater.FileMeter{Path: r.eng.cfg.MeterCheckpoint}).FinalSample(t.Context()); err == nil {
		t.Fatal("open epoch exposed checkpoint")
	}
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
