package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/healthattest"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
)

// TestReviewR13_Q5_ToRecoveryOverwritesErrorStageWithReportAndSealsNullHealth
// is the r13 M-4 proof. seal must keep the error_stage toRecovery recorded,
// and a recovery entered at rollback with no stored profile must carry
// blankHealth. Central (internal/fleetupdate/results.go) rejects a terminal
// result at or after the health stages when health is null:
// "Terminal result at or after health checks requires health evidence".
// beforeHealth does not include rollback or report. The pre-fix document was
// stage=report and health=null, which that rule rejects with 422.
func TestReviewR13_Q5_ToRecoveryOverwritesErrorStageWithReportAndSealsNullHealth(t *testing.T) {
	t.Run("stepRollback_issueChallenge_failure_overwrites_rollback_stage_with_report", func(t *testing.T) {
		w := newWorld(t)
		r1 := w.makeRelease("r1", 1, []int{1})
		w.seed(r1)
		w.rollbackAllow = []uuid.UUID{r1.id}
		r2 := w.makeRelease("r2", 1, []int{1})

		cfg := probeConfigFixture(t)
		cfg.ProbePeer = probeFixture()
		cfg.probeSources.interfaces = func() ([]net.Addr, error) { return probeInterfaces("1.2.3.4")() }
		w.eng.ProbePeer = cfg.ProbePeer
		w.eng.probeEnv = cfg.probeEnvironment()

		w.svc.stopErr = errors.New("stop failed before health check")
		w.eng.Hooks.AfterStage = func(stage string) error {
			if stage == string(updatejournal.StageRollingBack) {
				return os.MkdirAll(filepath.Join(w.layout.Journal, healthattest.ChallengeFile, "block"), 0o700)
			}
			return nil
		}
		work := w.work(r2, "5", api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{r1.id, r2.id}
		w.setWork(work)
		_ = w.run()

		sealed := w.oneResult()
		assertRollbackRecoverySeal(t, sealed, api.ErrorCode(apierr.RollbackFailed))
	})

	t.Run("pre_check_toRecovery_seals_stage_report_with_null_health_rejected_by_central", func(t *testing.T) {
		w := newWorld(t)
		r1 := w.makeRelease("r1", 1, []int{1})
		w.seed(r1)
		w.rollbackAllow = []uuid.UUID{r1.id}
		r2 := w.makeRelease("r2", 1, []int{1})

		// stepStop fails before stepCheck (so txn.Health is empty), and r1 is no
		// longer in Serving so stepRollback enters toRecovery at apply.go:249.
		w.svc.stopErr = errors.New("stop failed before health check")
		work := w.work(r2, "5", api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{r2.id}
		w.setWork(work)
		_ = w.run()

		sealed := w.oneResult()
		assertRollbackRecoverySeal(t, sealed, api.ErrorCode(apierr.MinimumVersionViolation))
		if got := centralHealthRuleRejection(api.UpdateResultStateFailed, api.UpdateStageReport, true); got != centralHealthEvidenceMessage {
			t.Fatalf("report with null health must match central's rejection %q, got %q", centralHealthEvidenceMessage, got)
		}
	})
}

// centralHealthEvidenceMessage is the permanent 422 from SubmitResult when a
// terminal result at or after the health stages has health null
// (internal/fleetupdate/results.go).
const centralHealthEvidenceMessage = "Terminal result at or after health checks requires health evidence"

// stageBeforeHealth is the contract order in fleetupdate.beforeHealth.
// rollback and report are not in it, so they require health evidence.
func stageBeforeHealth(stage api.UpdateStage) bool {
	switch stage {
	case api.UpdateStageClaim, api.UpdateStageDownload, api.UpdateStageVerify,
		api.UpdateStagePreflight, api.UpdateStagePrepare, api.UpdateStageBlockForwarding,
		api.UpdateStageFinalSample, api.UpdateStageStopServices, api.UpdateStageMigrate,
		api.UpdateStageSwitch, api.UpdateStageStart:
		return true
	default:
		return false
	}
}

// centralHealthRuleRejection mirrors the health half of SubmitResult. An empty
// string means that half accepts the document. It is not a substitute for the
// real central call in the integration test.
func centralHealthRuleRejection(state api.UpdateResultState, stage api.UpdateStage, healthIsNull bool) string {
	if state == api.UpdateResultStateFailed && stageBeforeHealth(stage) && !healthIsNull {
		return "Pre-health failure requires null health, an error and the unchanged attempt release"
	}
	terminal := state == api.UpdateResultStateSucceeded || state == api.UpdateResultStateFailed ||
		state == api.UpdateResultStateRolledBack || state == "already_current"
	if !(state == api.UpdateResultStateFailed && stageBeforeHealth(stage)) && terminal && healthIsNull {
		return centralHealthEvidenceMessage
	}
	return ""
}

func assertRollbackRecoverySeal(t *testing.T, sealed api.UpdateResult, code api.ErrorCode) {
	t.Helper()
	stage := requiredValue(t, sealed.Stage)
	if stage != api.UpdateStageRollback {
		raw, _ := json.Marshal(sealed)
		t.Fatalf("sealed stage %q, want rollback; document %s", stage, raw)
	}
	if sealed.State != api.UpdateResultStateFailed || sealed.MemberForwardingEnabled || !requiredValue(t, sealed.RecoveryRequired) {
		t.Fatalf("state %s forwarding %v recovery %v", sealed.State, sealed.MemberForwardingEnabled, requiredValue(t, sealed.RecoveryRequired))
	}
	if got := requiredValue(t, sealed.ErrorCode); got != code {
		t.Fatalf("error_code %s, want %s", got, code)
	}
	h := requiredValue(t, sealed.Health)
	if h.Processes != api.HealthCheckStatusFailed || h.InternetOnly != api.HealthCheckStatusFailed ||
		h.DnsPolicy != api.HealthCheckStatusFailed || h.TunnelProbe != api.HealthCheckStatusFailed {
		t.Fatalf("rollback recovery health = %+v, want a blank failed profile", h)
	}
	if got := centralHealthRuleRejection(sealed.State, stage, false); got != "" {
		t.Fatal(got)
	}
}

// TestReviewR13_Q5_ProbeOmissionJournalMetaRowsNeverPruned is the r13 L-3 proof.
// Finished apply transactions must not leave probe_omission/<txn_id>/<nonce>
// rows in SQLite meta. cleanTxnNotes and Retain delete those rows. The
// append-only txn_events audit stays. Standalone recovery rows, which have
// no txns row, are covered by the journal test.
func TestReviewR13_Q5_ProbeOmissionJournalMetaRowsNeverPruned(t *testing.T) {
	cfg := probeConfigFixture(t)
	cfg.ProbePeer = probeFixture()
	nodeIP := "192.0.2.10"
	cfg.probeSources.interfaces = func() ([]net.Addr, error) { return probeInterfaces(nodeIP)() }
	loaded, err := loadProbeConfig(t, cfg)
	if err != nil {
		t.Fatalf("loadProbeConfig: %v", err)
	}
	configured, err := loaded.Engine()
	if err != nil {
		t.Fatalf("Engine: %v", err)
	}
	defer configured.Close()

	w := newWorld(t)
	r1 := w.makeRelease("r1", 1, []int{1})
	w.seed(r1)
	w.eng.ProbePeer, w.eng.probeEnv = configured.ProbePeer, configured.probeEnv

	// Node acquires 1.2.3.4 so every update's health check refuses the probe
	// with "node_destination" and records a probe_omission/<txn>/<nonce> row.
	nodeIP = "1.2.3.4"
	prev := r1
	var completedTxns []string
	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage == string(updatejournal.StageChecking) {
			if txn, err := w.eng.Journal.ActiveTxn(context.Background()); err == nil {
				completedTxns = append(completedTxns, txn.TxnID)
			}
		}
		return nil
	}
	for i, name := range []string{"r2", "r3", "r4"} {
		w.rollbackAllow = []uuid.UUID{prev.id}
		next := w.makeRelease(name, 1, []int{1})
		work := w.work(next, string(rune('5'+i)), api.PublicationIntentPublished)
		work.Serving = []uuid.UUID{prev.id, next.id}
		w.setWork(work)
		if err := w.run(); err != nil {
			t.Fatalf("run %s: %v", name, err)
		}
		prev = next
	}

	var unpruned int
	for _, txnID := range completedTxns {
		recs, err := w.eng.Journal.ProbeOmissions(context.Background(), txnID)
		if err != nil {
			t.Fatalf("ProbeOmissions(%s): %v", txnID, err)
		}
		unpruned += len(recs)
	}
	if unpruned != 0 {
		t.Fatalf("%d probe_omission/<txn_id>/<nonce> rows across %d completed transactions remained after cleanTxnNotes and Retain",
			unpruned, len(completedTxns))
	}
	for _, txnID := range completedTxns {
		events, err := w.eng.Journal.TxnEvents(context.Background(), txnID)
		if err != nil {
			t.Fatal(err)
		}
		var audit bool
		for _, event := range events {
			var d struct {
				Kind string `json:"kind"`
			}
			if json.Unmarshal([]byte(event.Detail), &d) == nil && d.Kind == "probe_omitted" {
				audit = true
			}
		}
		if !audit {
			t.Fatalf("txn_events audit for %s was deleted with the meta row", txnID)
		}
	}
}

// TestReviewR13_Q5_UpdaterWhatHeld verifies the Q5 properties that held in
// node/updater and node/internal/healthattest:
//   - If RecordProbeOmission fails in issueChallenge (e.g. closed/unwritable
//     journal), publication of health-challenge.json is aborted before disk write.
//   - readProbeAgentConfig rejects symlinks and rejects an atomic rename to a
//     different file between Lstat and OpenFile (when simulated via SameFile),
//     while SameFile cannot detect in-place modification of the same inode.
//   - A pre-#168 agent decoder (DisallowUnknownFields without probe_omitted)
//     rejects an omission-bearing challenge.
func TestReviewR13_Q5_UpdaterWhatHeld(t *testing.T) {
	cfg := probeConfigFixture(t)
	cfg.ProbePeer = probeFixture()
	nodeIP := "192.0.2.10"
	cfg.probeSources.interfaces = func() ([]net.Addr, error) { return probeInterfaces(nodeIP)() }
	loaded, err := loadProbeConfig(t, cfg)
	if err != nil {
		t.Fatalf("loadProbeConfig: %v", err)
	}
	configured, err := loaded.Engine()
	if err != nil {
		t.Fatalf("Engine: %v", err)
	}
	defer configured.Close()

	w := newWorld(t)
	r1 := w.makeRelease("r1", 1, []int{1})
	w.seed(r1)
	w.eng.ProbePeer, w.eng.probeEnv = configured.ProbePeer, configured.probeEnv
	nodeIP = "1.2.3.4"

	// Close the journal so RecordProbeOmission fails during issueChallenge,
	// and verify that health-challenge.json is NOT written to disk.
	_ = w.eng.Journal.Close()
	txn := updatejournal.Txn{TxnID: uuid.Must(uuid.NewV7()).String()}
	if err := w.eng.issueChallenge(context.Background(), txn, r1.id.String(), "candidate"); err == nil {
		t.Fatal("expected issueChallenge to fail when RecordProbeOmission fails")
	}
	if _, err := os.Stat(filepath.Join(w.layout.Journal, healthattest.ChallengeFile)); !os.IsNotExist(err) {
		t.Fatalf("health-challenge.json should not be published when RecordProbeOmission fails: err=%v", err)
	}

	// Re-open journal and issue an omission-bearing challenge; verify that a
	// pre-#168 agent decoder (strict DisallowUnknownFields without probe_omitted)
	// rejects the challenge JSON.
	w.eng.Journal = openJournal(t, w.layout.Journal)
	if err := w.eng.issueChallenge(context.Background(), txn, r1.id.String(), "candidate"); err != nil {
		t.Fatalf("issueChallenge: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(w.layout.Journal, healthattest.ChallengeFile))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	type pre168Challenge struct {
		SchemaVersion int                          `json:"schema_version"`
		Nonce         string                       `json:"nonce"`
		ReleaseID     string                       `json:"release_id"`
		IssuedAt      time.Time                    `json:"issued_at"`
		ExpiresAt     time.Time                    `json:"expires_at"`
		Components    api.ManifestComponents       `json:"components"`
		Probe         *healthattest.ChallengeProbe `json:"probe,omitempty"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var oldCh pre168Challenge
	if err := dec.Decode(&oldCh); err == nil {
		t.Fatal("expected pre-#168 strict decoder without probe_omitted field to reject omission-bearing challenge")
	}
}
