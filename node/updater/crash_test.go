package updater

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// TestMain runs a crash child before the test harness. The parent execs the
// test binary with FVPN_UPDATER_CHILD=1. The child exits before m.Run so it
// does not re-enter the suite. Exit 86 means the journal stage was durable
// and the process stopped before that stage's side effect (§13.5, AT25).
func TestMain(m *testing.M) {
	if os.Getenv("FVPN_UPDATER_CHILD") == "1" {
		os.Exit(runCrashChild())
	}
	os.Exit(m.Run())
}

type crashScenario struct {
	Layout         Layout        `json:"layout"`
	Node           Node          `json:"node"`
	Platform       Platform      `json:"platform"`
	Work           Work          `json:"work"`
	Trust          []byte        `json:"trust"`
	CertPEM        []byte        `json:"cert_pem"`
	Now            time.Time     `json:"now"`
	Fail           []string      `json:"fail"`
	FileCheckpoint string        `json:"file_checkpoint,omitempty"`
	MeterWait      time.Duration `json:"meter_wait,omitempty"`
}

func runCrashChild() int {
	raw, err := os.ReadFile(os.Getenv("FVPN_UPDATER_SCENARIO"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "scenario: %v\n", err)
		return 1
	}
	var sc crashScenario
	if err := json.Unmarshal(raw, &sc); err != nil {
		fmt.Fprintf(os.Stderr, "scenario json: %v\n", err)
		return 1
	}
	trust, err := envelope.ParseNodeTrustStoreJSON(sc.Trust)
	if err != nil {
		fmt.Fprintf(os.Stderr, "trust: %v\n", err)
		return 1
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(sc.CertPEM) {
		fmt.Fprintf(os.Stderr, "cert: empty\n")
		return 1
	}
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs: pool, MinVersion: tls.VersionTLS12,
	}}}
	fail := map[string]bool{}
	for _, id := range sc.Fail {
		fail[id] = true
	}
	src := &memSource{}
	src.set(sc.Work)
	eng := &Engine{
		Layout: sc.Layout, Node: sc.Node, Platform: sc.Platform,
		ReleaseTrust: trust, Source: src, HTTP: client,
		Meter: newMeter(), Services: &fakeSvc{}, Health: &fakeHealth{fail: fail},
		Gate: &fakeGate{}, Migrator: &fakeMig{},
		Download: DownloadConfig{
			MaxAttempts: 4, Initial: time.Millisecond, MaxWait: 5 * time.Millisecond,
			AttemptTimeout: 5 * time.Second, ContentRetries: 1,
		},
		Now: func() time.Time { return sc.Now }, Sleep: func(time.Duration) {}, Jitter: func() float64 { return 0 },
		AllowLoopback: true,
	}
	crash := os.Getenv("FVPN_UPDATER_CRASH")
	if sc.FileCheckpoint != "" {
		eng.Meter = FileMeter{Path: sc.FileCheckpoint, Wait: sc.MeterWait}
		eng.Gate = FileGate{Path: filepath.Join(sc.Layout.Journal, "forwarding.blocked")}
	}
	eng.Hooks.AfterStage = func(stage string) error {
		if crash == "" || stage != crash {
			return nil
		}
		// Close before Exit. Exit skips defers, and the parent must be able
		// to open the journal (§13.5).
		_ = eng.Close()
		os.Exit(86)
		return nil
	}
	err = eng.Run(context.Background())
	_ = eng.Close()
	if err == nil {
		return 0
	}
	if errors.Is(err, ErrMembersBlocked) {
		return 2
	}
	fmt.Fprintf(os.Stderr, "child: %v\n", err)
	return 1
}

func TestUPD_StageGraph(t *testing.T) {
	// Happy path of §13.6 steps 6-12. rolling_back is the extra journal
	// stage for §13.9; every non-terminal stage may enter it.
	happy := []updatejournal.Stage{
		updatejournal.StagePrepared,
		updatejournal.StageBlocked,
		updatejournal.StageFinalSampled,
		updatejournal.StageStopped,
		updatejournal.StageMigrated,
		updatejournal.StageSwitched,
		updatejournal.StageChecking,
		updatejournal.StageLocallySucceeded,
	}
	for i := 0; i < len(happy)-1; i++ {
		if !updatejournal.CanTransition(happy[i], happy[i+1]) {
			t.Fatalf("happy path %s -> %s refused", happy[i], happy[i+1])
		}
	}
	for _, s := range happy[:len(happy)-1] {
		if !updatejournal.CanTransition(s, updatejournal.StageRollingBack) {
			t.Fatalf("%s cannot roll back", s)
		}
		if !updatejournal.CanTransition(s, updatejournal.StageRecoveryRequired) {
			t.Fatalf("%s cannot enter recovery_required", s)
		}
		if s.Terminal() {
			t.Fatalf("%s is not a terminal stage", s)
		}
	}
	for _, s := range []updatejournal.Stage{updatejournal.StageLocallySucceeded, updatejournal.StageRolledBack, updatejournal.StageRecoveryRequired} {
		if !s.Terminal() || updatejournal.CanTransition(s, updatejournal.StagePrepared) {
			t.Fatalf("%s must be a closed terminal", s)
		}
	}
}

func TestAT25_CrashEachStageThenSucceed(t *testing.T) {
	stages := []string{
		string(updatejournal.StagePrepared),
		string(updatejournal.StageBlocked),
		string(updatejournal.StageFinalSampled),
		string(updatejournal.StageStopped),
		string(updatejournal.StageMigrated),
		string(updatejournal.StageSwitched),
		string(updatejournal.StageChecking),
	}
	for _, stage := range stages {
		t.Run(stage, func(t *testing.T) {
			w, idPath, usagePath, beforeID, beforeUsage, work := crashWorld(t, false)
			code, out := runChild(t, w, work, nil, stage)
			if code != 86 {
				t.Fatalf("exit %d, want 86\n%s", code, out)
			}
			j := reopen(t, w.layout.Journal)
			txn, err := j.ActiveTxn(context.Background())
			if err != nil || string(txn.Stage) != stage {
				t.Fatalf("stage %s err %v, want %s", txn.Stage, err, stage)
			}
			if blocked, why := j.BlocksForwarding(context.Background()); !blocked || why == "" {
				t.Fatal("members were not held while the transaction was open")
			}
			box, _ := j.Outbox(context.Background())
			if len(box) != 0 {
				t.Fatal("a result was enqueued before the transaction finished")
			}
			mustClose(t, j)

			code, out = runChild(t, w, work, nil, "")
			if code != 0 {
				t.Fatalf("recover exit %d\n%s", code, out)
			}
			j = reopen(t, w.layout.Journal)
			defer mustClose(t, j)
			last, err := j.LastTxn(context.Background())
			if err != nil || last.Stage != updatejournal.StageLocallySucceeded {
				t.Fatalf("ended %s %v", last.Stage, err)
			}
			if blocked, _ := j.BlocksForwarding(context.Background()); blocked {
				t.Fatal("members stayed blocked after checks passed")
			}
			if cur, _ := w.layout.currentID(); cur != work.Intent.ReleaseId.String() {
				t.Fatalf("current %s", cur)
			}
			assertSameBytes(t, idPath, beforeID)
			assertSameBytes(t, usagePath, beforeUsage)
			box, err = j.Outbox(context.Background())
			if err != nil || len(box) != 1 || box[0].State != string(api.UpdateResultStateSucceeded) {
				t.Fatalf("outbox %+v %v", box, err)
			}
		})
	}
}

func TestAT25_CrashAtLocalSuccessStillReports(t *testing.T) {
	w, idPath, usagePath, beforeID, beforeUsage, work := crashWorld(t, false)
	code, out := runChild(t, w, work, nil, string(updatejournal.StageLocallySucceeded))
	if code != 86 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	j := reopen(t, w.layout.Journal)
	last, err := j.LastTxn(context.Background())
	if err != nil || last.Stage != updatejournal.StageLocallySucceeded {
		t.Fatalf("stage %s %v", last.Stage, err)
	}
	if blocked, _ := j.BlocksForwarding(context.Background()); blocked {
		t.Fatal("a passed check must not stay blocked in the journal")
	}
	box, _ := j.Outbox(context.Background())
	if len(box) != 0 {
		t.Fatal("enqueue ran before the crash hook")
	}
	mustClose(t, j)

	code, out = runChild(t, w, work, nil, "")
	if code != 0 {
		t.Fatalf("recover exit %d\n%s", code, out)
	}
	j = reopen(t, w.layout.Journal)
	defer mustClose(t, j)
	box, err = j.Outbox(context.Background())
	if err != nil || len(box) != 1 || box[0].State != string(api.UpdateResultStateSucceeded) {
		t.Fatalf("outbox %+v %v", box, err)
	}
	assertSameBytes(t, idPath, beforeID)
	assertSameBytes(t, usagePath, beforeUsage)
}

func TestAT25_CrashDuringRollback(t *testing.T) {
	w, idPath, usagePath, beforeID, beforeUsage, work := crashWorld(t, false)
	old := mustPointer(t, w)
	code, out := runChild(t, w, work, []string{work.Intent.ReleaseId.String()}, string(updatejournal.StageRollingBack))
	if code != 86 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	j := reopen(t, w.layout.Journal)
	txn, err := j.ActiveTxn(context.Background())
	if err != nil || txn.Stage != updatejournal.StageRollingBack {
		t.Fatalf("stage %s %v", txn.Stage, err)
	}
	if blocked, _ := j.BlocksForwarding(context.Background()); !blocked {
		t.Fatal("rollback left members unblocked")
	}
	mustClose(t, j)

	code, out = runChild(t, w, work, []string{work.Intent.ReleaseId.String()}, "")
	if code != 0 {
		t.Fatalf("recover exit %d\n%s", code, out)
	}
	j = reopen(t, w.layout.Journal)
	defer mustClose(t, j)
	last, err := j.LastTxn(context.Background())
	if err != nil || last.Stage != updatejournal.StageRolledBack {
		t.Fatalf("ended %s %v", last.Stage, err)
	}
	if blocked, _ := j.BlocksForwarding(context.Background()); blocked {
		t.Fatal("members stayed blocked after a verified rollback")
	}
	if cur, _ := w.layout.currentID(); cur != old {
		t.Fatalf("current %s, want last_good %s", cur, old)
	}
	box, _ := j.Outbox(context.Background())
	if len(box) != 1 || box[0].State != string(api.UpdateResultStateRolledBack) {
		t.Fatalf("outbox %+v", box)
	}
	assertSameBytes(t, idPath, beforeID)
	assertSameBytes(t, usagePath, beforeUsage)
}

func TestAT25_CrashAtRecoveryRequiredStaysBlocked(t *testing.T) {
	w, _, _, _, _, work := crashWorld(t, true)
	code, out := runChild(t, w, work, []string{work.Intent.ReleaseId.String()}, string(updatejournal.StageRecoveryRequired))
	if code != 86 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	j := reopen(t, w.layout.Journal)
	last, err := j.LastTxn(context.Background())
	if err != nil || last.Stage != updatejournal.StageRecoveryRequired {
		t.Fatalf("stage %s %v", last.Stage, err)
	}
	if blocked, why := j.BlocksForwarding(context.Background()); !blocked || why == "" {
		t.Fatal("recovery_required did not hold members")
	}
	mustClose(t, j)

	code, out = runChild(t, w, work, []string{work.Intent.ReleaseId.String()}, "")
	if code != 2 {
		t.Fatalf("recover exit %d, want 2\n%s", code, out)
	}
	j = reopen(t, w.layout.Journal)
	defer mustClose(t, j)
	if blocked, _ := j.BlocksForwarding(context.Background()); !blocked {
		t.Fatal("restart cleared recovery_required")
	}
	box, err := j.Outbox(context.Background())
	if err != nil || len(box) != 1 || !strings.Contains(string(box[0].Payload), `"recovery_required":true`) {
		t.Fatalf("outbox %+v %v", box, err)
	}
}

func TestAT25_CrashAfterRollbackStillReports(t *testing.T) {
	w, _, _, _, _, work := crashWorld(t, false)
	code, out := runChild(t, w, work, []string{work.Intent.ReleaseId.String()}, string(updatejournal.StageRolledBack))
	if code != 86 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	j := reopen(t, w.layout.Journal)
	last, err := j.LastTxn(context.Background())
	if err != nil || last.Stage != updatejournal.StageRolledBack {
		t.Fatalf("stage %s %v", last.Stage, err)
	}
	box, _ := j.Outbox(context.Background())
	if len(box) != 0 {
		t.Fatal("result enqueued before the seal hook")
	}
	mustClose(t, j)
	code, out = runChild(t, w, work, []string{work.Intent.ReleaseId.String()}, "")
	if code != 0 {
		t.Fatalf("recover exit %d\n%s", code, out)
	}
	j = reopen(t, w.layout.Journal)
	defer mustClose(t, j)
	box, _ = j.Outbox(context.Background())
	if len(box) != 1 || box[0].State != string(api.UpdateResultStateRolledBack) {
		t.Fatalf("outbox %+v", box)
	}
	if blocked, _ := j.BlocksForwarding(context.Background()); blocked {
		t.Fatal("reported rollback still blocks members")
	}
}

func crashWorld(t *testing.T, dropLastGood bool) (*world, string, string, []byte, []byte, Work) {
	t.Helper()
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	idPath, usagePath := touchPersistent(t, w.layout)
	beforeID, beforeUsage := fileBytes(t, idPath), fileBytes(t, usagePath)
	work := w.work(next, "40", api.PublicationIntentPublished)
	if !dropLastGood {
		work.Serving = append(work.Serving, old.id)
	}
	// The parent must not hold the journal while the child runs.
	if err := w.eng.Journal.Close(); err != nil {
		t.Fatal(err)
	}
	w.eng.Journal = nil
	return w, idPath, usagePath, beforeID, beforeUsage, work
}

func mustPointer(t *testing.T, w *world) string {
	t.Helper()
	j := reopen(t, w.layout.Journal)
	defer mustClose(t, j)
	p, err := j.Pointers(context.Background())
	if err != nil || p.CurrentRelease == "" {
		t.Fatal(err)
	}
	return p.CurrentRelease
}

func reopen(t *testing.T, dir string) *updatejournal.Journal {
	t.Helper()
	j, err := updatejournal.Open(dir, updatejournal.Options{Now: fixedNow})
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func mustClose(t *testing.T, j *updatejournal.Journal) {
	t.Helper()
	if err := j.Close(); err != nil {
		t.Fatal(err)
	}
}

func runChild(t *testing.T, w *world, work Work, fail []string, crash string) (int, string) {
	t.Helper()
	trust, err := json.Marshal(w.trust)
	if err != nil {
		t.Fatal(err)
	}
	sc := crashScenario{
		Layout: w.layout, Node: w.node, Platform: testPlatform(), Work: work,
		Trust: trust, CertPEM: w.srv.CertPEM, Now: w.now, Fail: fail,
	}
	if m, ok := w.eng.fileMeter(); ok {
		sc.FileCheckpoint, sc.MeterWait = m.Path, m.Wait
	}
	raw, err := json.Marshal(sc)
	if err != nil {
		t.Fatal(err)
	}
	path := w.root + "/crash-scenario.json"
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0])
	cmd.Env = childEnv(path, crash)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	if err == nil {
		return 0, buf.String()
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), buf.String()
	}
	return -1, buf.String() + err.Error()
}

func childEnv(scenario, crash string) []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "FVPN_UPDATER_") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "FVPN_UPDATER_CHILD=1", "FVPN_UPDATER_SCENARIO="+scenario)
	if crash != "" {
		env = append(env, "FVPN_UPDATER_CRASH="+crash)
	}
	return env
}
