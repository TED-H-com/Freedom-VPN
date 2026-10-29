package updater

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
	"github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal"
	"github.com/TED-H-com/Freedom-VPN/node/updater/fsutil"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
)

// stockHealth uses NodeChecks for the two layers the updater can prove and
// passes the three layers the agent owns. Persistent data must survive a
// real FileMeter transaction (D-034c).
type stockHealth struct{}

func (stockHealth) Processes(ctx context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	return NodeChecks{}.Processes(ctx, v)
}
func (stockHealth) PersistentData(ctx context.Context, v ReleaseView) (api.HealthCheckStatus, error) {
	return NodeChecks{}.PersistentData(ctx, v)
}
func (stockHealth) Isolation(context.Context, ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusPassed, nil
}
func (stockHealth) DNSPolicy(context.Context, ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusPassed, nil
}
func (stockHealth) TunnelProbe(context.Context, ReleaseView) (api.HealthCheckStatus, error) {
	return api.HealthCheckStatusPassed, nil
}

func TestUPD_ExtractRejectsSymlinkChain(t *testing.T) {
	dest := t.TempDir()
	err := ExtractArchive(packTar(t, []tarMember{
		{name: "a", typ: tar.TypeSymlink, link: "b"},
		{name: "b", typ: tar.TypeSymlink, link: "c"},
		{name: "c", body: "secret"},
		{name: "hard", typ: tar.TypeLink, link: "c"},
	}), dest, ExtractOptions{})
	if err == nil {
		t.Fatal("symlink chain extracted")
	}
	if asOp(err) == nil || asOp(err).Code != apierr.ValidationFailed {
		t.Fatal(err)
	}
	if _, statErr := os.Lstat(filepath.Join(dest, "a")); !os.IsNotExist(statErr) {
		t.Fatalf("symlink left behind: %v", statErr)
	}
	tree := t.TempDir()
	if err := os.WriteFile(filepath.Join(tree, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("file", filepath.Join(tree, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := fsutil.TreeHash(tree); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("TreeHash accepted a symlink: %v", err)
	}
}

func TestUPD_RecoveryRequiredIsNotAlreadyCurrent(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	w.health.fail[next.id.String()] = true
	work := w.work(next, "14", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{next.id}
	work.Revoked = []uuid.UUID{old.id}
	w.setWork(work)
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	res := w.oneResult()
	if !requiredValue(t, res.RecoveryRequired) || res.State == api.UpdateResultStateAlreadyCurrent {
		t.Fatalf("first result state %s recovery %v", res.State, requiredValue(t, res.RecoveryRequired))
	}
	work.Intent.Attempt = 2
	w.setWork(work)
	err := w.run()
	if !errors.Is(err, ErrMembersBlocked) {
		t.Fatalf("attempt 2: %v", err)
	}
	for _, outcome := range w.outcomes() {
		if outcome == string(api.UpdateResultStateAlreadyCurrent) || outcome == "ack:"+string(api.UpdateResultStateAlreadyCurrent) {
			t.Fatalf("recovery_required was treated as already_current: %v", w.outcomes())
		}
	}
	blocked, _, _ := w.gate.snapshot()
	if !blocked {
		t.Fatal("gate opened for a release in recovery_required")
	}
	on, _ := w.eng.Journal.BlocksForwarding(context.Background())
	if !on {
		t.Fatal("journal stopped blocking")
	}
}

func TestUPD_StaleLastTxnCannotUnblock(t *testing.T) {
	w := newWorld(t)
	cur := w.makeRelease("cur", 1, []int{1})
	w.seed(cur)
	w.setWork(w.work(cur, "3", api.PublicationIntentPublished))
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	if blocked, _, _ := w.gate.snapshot(); blocked {
		t.Fatal("published release stayed blocked")
	}
	w.setWork(w.work(cur, "4", api.PublicationIntentDisabled))
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	if blocked, _, _ := w.gate.snapshot(); !blocked {
		t.Fatal("disabled publication did not block")
	}
	// The latch file is cleared. LastTxn is still the published install.
	// Recover must re-block from the gate record.
	w.gate.mu.Lock()
	w.gate.blocked = false
	w.gate.mu.Unlock()
	err := w.eng.Recover(context.Background())
	if !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	if blocked, _, _ := w.gate.snapshot(); !blocked {
		t.Fatal("Recover unblocked from a stale published transaction")
	}
	rec, err := w.eng.Journal.GateRecord(context.Background())
	if err != nil || rec.PublicationIntent != string(api.PublicationIntentDisabled) || !rec.ForwardingBlocked {
		t.Fatalf("gate record %+v %v", rec, err)
	}
}

func TestUPD_MinimumVersionViolationBlocksGate(t *testing.T) {
	w := newWorld(t)
	cur := w.makeRelease("cur", 1, []int{1})
	w.seed(cur)
	w.setWork(w.work(cur, "3", api.PublicationIntentPublished))
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	work := w.work(cur, "4", api.PublicationIntentPublished)
	work.Serving = nil
	w.setWork(work)
	err := w.run()
	if asOp(err) == nil || asOp(err).Code != apierr.MinimumVersionViolation {
		t.Fatal(err)
	}
	if errors.Is(err, ErrMembersBlocked) {
		t.Fatal("minimum version was reported as members blocked")
	}
	blocked, _, _ := w.gate.snapshot()
	on, _ := w.eng.Journal.BlocksForwarding(context.Background())
	rec, gerr := w.eng.Journal.GateRecord(context.Background())
	if !blocked || !on || gerr != nil || rec.TerminalOutcome != updatejournal.OutcomeFailed {
		t.Fatalf("blocked %v journal %v record %+v err %v", blocked, on, rec, gerr)
	}
	stops, _ := w.svc.counts()
	if stops != 0 {
		t.Fatal("services restarted")
	}
}

func TestUPD_PersistentDataSurvivesFileMeter(t *testing.T) {
	w := newWorld(t)
	_, usagePath := touchPersistent(t, w.layout)
	cp := filepath.Join(w.layout.State, "meter.json")
	epoch := uuid.Must(uuid.NewV7()).String()
	body := []byte(`{"closed_meter_epoch_id":"` + epoch + `","last_durable_event_seq":"4","central_acked_through":"3"}`)
	if err := os.WriteFile(cp, body, 0o600); err != nil {
		t.Fatal(err)
	}
	w.eng.Meter = FileMeter{Path: cp}
	w.eng.Health = stockHealth{}
	var sawEpochNote bool
	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage != string(updatejournal.StageChecking) {
			return nil
		}
		f, err := os.OpenFile(usagePath, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte("-mutated"))
		_ = f.Close()
		if err != nil {
			return err
		}
		entries, err := os.ReadDir(w.layout.Journal)
		if err != nil {
			return err
		}
		for _, ent := range entries {
			if strings.HasPrefix(ent.Name(), "epoch-") {
				sawEpochNote = true
			}
		}
		matches, _ := filepath.Glob(filepath.Join(w.layout.State, "*.epoch"))
		if len(matches) != 0 {
			return errors.New("epoch sidecar written into state")
		}
		return nil
	}
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	work := w.work(next, "6", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	if !sawEpochNote {
		t.Fatal("journal epoch note was missing during the transaction")
	}
	res := w.oneResult()
	if res.State != api.UpdateResultStateSucceeded {
		t.Fatalf("state %s", res.State)
	}
	left, err := os.ReadFile(cp)
	if err != nil || string(left) != string(body) {
		t.Fatalf("checkpoint changed: %s %v", left, err)
	}
	entries, err := os.ReadDir(w.layout.Journal)
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range entries {
		if strings.HasPrefix(ent.Name(), "epoch-") {
			t.Fatalf("epoch note %s survived the transaction", ent.Name())
		}
	}
	matches, _ := filepath.Glob(filepath.Join(w.layout.State, "*.epoch"))
	if len(matches) != 0 {
		t.Fatalf("state epoch sidecar %v", matches)
	}
	if !strings.HasSuffix(string(fileBytes(t, usagePath)), "-mutated") {
		t.Fatal("state was not mutated before the persistent-data check")
	}
}

func TestUPD_RecoverRefusedWhileLockHeld(t *testing.T) {
	if os.Getenv("FVPN_HOLD_LOCK") == "1" {
		lk, err := nodelock.TryAcquire(os.Getenv("FVPN_LOCK_PATH"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "child lock: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(os.Getenv("FVPN_LOCK_READY"), []byte("1"), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "ready: %v\n", err)
			os.Exit(1)
		}
		_, _ = io.Copy(io.Discard, os.Stdin)
		_ = lk.Release()
		os.Exit(0)
	}
	w := newWorld(t)
	ready := filepath.Join(w.root, "lock-ready")
	cmd := exec.Command(os.Args[0], "-test.run=^TestUPD_RecoverRefusedWhileLockHeld$")
	cmd.Env = append(os.Environ(), "FVPN_HOLD_LOCK=1", "FVPN_LOCK_PATH="+w.layout.Lock, "FVPN_LOCK_READY="+ready)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waitReady(t, ready)
	err = w.eng.Recover(context.Background())
	_ = stdin.Close()
	_ = cmd.Wait()
	op := asOp(err)
	if op == nil || op.Code != apierr.UpdateBusy {
		t.Fatal(err)
	}
	if errors.Is(err, ErrMembersBlocked) {
		t.Fatal("a held lock was reported as members blocked")
	}
}

func waitReady(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("lock holder did not become ready")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestUPD_PartialMetaBeforeBodyAndIfRange(t *testing.T) {
	body := bytes.Repeat([]byte("abc"), 20000)
	sum := sha256.Sum256(body)
	var attempt int
	var sawRange, sawIfRange string
	var metaExisted bool
	destDir := t.TempDir()
	dest := filepath.Join(destDir, "artifact")
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			hj, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "no hijack", 500)
				return
			}
			conn, buf, err := hj.Hijack()
			if err != nil {
				return
			}
			half := body[:len(body)/2]
			_, _ = buf.WriteString("HTTP/1.1 200 OK\r\nETag: \"etag-v1\"\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\nConnection: close\r\n\r\n")
			_, _ = buf.Write(half)
			_ = buf.Flush()
			_ = conn.Close()
			return
		}
		matches, _ := filepath.Glob(dest + ".partial.meta")
		metaExisted = len(matches) > 0
		sawRange = r.Header.Get("Range")
		sawIfRange = r.Header.Get("If-Range")
		half := len(body) / 2
		w.Header().Set("ETag", `"etag-v1"`)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", half, len(body)-1, len(body)))
		w.Header().Set("Content-Length", strconv.Itoa(len(body)-half))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(body[half:])
	}))
	srv.EnableHTTP2 = false
	srv.StartTLS()
	t.Cleanup(srv.Close)
	eng := &Engine{
		HTTP: srv.Client(), AllowLoopback: true,
		Download: DownloadConfig{MaxAttempts: 4, Initial: time.Millisecond, MaxWait: 5 * time.Millisecond, AttemptTimeout: 5 * time.Second, ContentRetries: 1},
		Now:      func() time.Time { return fixedNow() }, Sleep: func(time.Duration) {}, Jitter: func() float64 { return 0 },
	}
	err := eng.download(context.Background(), srv.URL+"/art", dest, hexEncode(sum[:]), hashRaw, int64(len(body)), int64(len(body)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !metaExisted {
		t.Fatal(".partial.meta was missing after the interrupted download")
	}
	if sawRange == "" || sawIfRange != `"etag-v1"` {
		t.Fatalf("resume Range %q If-Range %q", sawRange, sawIfRange)
	}
	if !bytes.Equal(fileBytes(t, dest), body) {
		t.Fatal("resumed body mismatch")
	}
}

func TestUPD_ManifestDownloadCap(t *testing.T) {
	var read bool
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		read = true
		w.Header().Set("Content-Length", strconv.FormatInt(canonjson.MaxBytes+1, 10))
		w.WriteHeader(http.StatusOK)
	}))
	srv.EnableHTTP2 = false
	srv.StartTLS()
	t.Cleanup(srv.Close)
	eng := &Engine{
		HTTP: srv.Client(), AllowLoopback: true,
		Download: DownloadConfig{MaxAttempts: 2, Initial: time.Millisecond, MaxWait: time.Millisecond, AttemptTimeout: 5 * time.Second},
		Now:      func() time.Time { return fixedNow() }, Sleep: func(time.Duration) {}, Jitter: func() float64 { return 0 },
	}
	dest := filepath.Join(t.TempDir(), "manifest.json")
	err := eng.download(context.Background(), srv.URL+"/m", dest, strings.Repeat("ab", 32), hashCanonical, 0, 0, nil)
	if asOp(err) == nil || asOp(err).Code != apierr.SizeMismatch {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatal("oversize manifest was stored")
	}
	if !read {
		t.Fatal("server was not reached")
	}
}

func TestUPD_ExtractEntryCap(t *testing.T) {
	members := make([]tarMember, maxArchiveEntries+1)
	for i := range members {
		members[i] = tarMember{name: fmt.Sprintf("f%d", i), body: "x"}
	}
	err := ExtractArchive(packTar(t, members), t.TempDir(), ExtractOptions{})
	if asOp(err) == nil || asOp(err).Code != apierr.ValidationFailed || !strings.Contains(err.Error(), "entry cap") {
		t.Fatal(err)
	}
}

func TestUPD_PathsOverlapSymmetric(t *testing.T) {
	if !pathsOverlap("/var/lib/state", "/var/lib/state/child") || !pathsOverlap("/var/lib/state/child", "/var/lib/state") {
		t.Fatal("containment was not symmetric")
	}
	if !pathsOverlap("/var/lib/state", "/var/lib/state") {
		t.Fatal("equal paths do not overlap")
	}
	if pathsOverlap("", "/var/lib/state") || pathsOverlap("/var/lib/state", "") || pathsOverlap("/var/lib/state", "/var/lib/staging") {
		t.Fatal("empty or sibling paths overlapped")
	}
	w := newWorld(t)
	id := uuid.Must(uuid.NewV7())
	dir := filepath.Join(w.root, "overlap", id.String())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "keep")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.eng.Layout.Releases = filepath.Join(w.root, "overlap")
	w.eng.Layout.State = dir
	err := w.eng.installTree(context.Background(), claimed{Work: Work{Intent: api.UpdateIntent{ReleaseId: id}}}, api.ReleaseManifest{}, nil, api.ManifestArtifact{}, filepath.Join(w.root, "missing-artifact"))
	if asOp(err) == nil || asOp(err).Code != apierr.ValidationFailed || !strings.Contains(err.Error(), "overlaps") {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(marker); statErr != nil {
		t.Fatal("release directory was removed before the overlap check")
	}
}

func TestUPD_AllowedReleaseIdsEnforced(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	w.rollbackAllow = []uuid.UUID{uuid.Must(uuid.NewV7())}
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	w.health.fail[next.id.String()] = true
	work := w.work(next, "16", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); !errors.Is(err, ErrMembersBlocked) {
		t.Fatal(err)
	}
	res := w.oneResult()
	if res.State == api.UpdateResultStateRolledBack || !requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("state %s recovery %v", res.State, requiredValue(t, res.RecoveryRequired))
	}
	w.mustCurrent(next.id)
	in := w.eng.evalLastGood(context.Background(), old.id.String(), w.eng.claimFrom(work), &next.manifest)
	plan := DecideRollback(in)
	if plan.Restore || plan.Code != apierr.MinimumVersionViolation {
		t.Fatalf("plan %+v input %+v", plan, in)
	}
}

func TestUPD_SchemaReadableNotOverridden(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 2, []int{1})
	w.seed(old)
	if err := w.eng.Journal.SetStateSchema(context.Background(), 2); err != nil {
		t.Fatal(err)
	}
	if !next.manifest.Rollback.StateReadableByPrevious {
		t.Fatal("fixture lost StateReadableByPrevious")
	}
	work := w.work(next, "1", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	in := w.eng.evalLastGood(context.Background(), old.id.String(), w.eng.claimFrom(work), &next.manifest)
	if in.SchemaReadable {
		t.Fatal("StateReadableByPrevious overrode an unreadable schema")
	}
	if plan := DecideRollback(in); plan.Restore || plan.Code != apierr.SchemaIncompatible {
		t.Fatalf("plan %+v", plan)
	}
}

func TestUPD_CheckCapBeginsRollback(t *testing.T) {
	w := newWorld(t)
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 1, []int{1})
	w.seed(old)
	w.eng.Hooks.AfterStage = func(stage string) error {
		if stage != string(updatejournal.StageChecking) {
			return nil
		}
		txn, err := w.eng.Journal.ActiveTxn(context.Background())
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(w.layout.Journal, "checks-"+txn.TxnID), []byte("8"), 0o600)
	}
	work := w.work(next, "17", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	w.mustCurrent(old.id)
	res := w.oneResult()
	if res.State != api.UpdateResultStateRolledBack || requiredValue(t, res.RecoveryRequired) {
		t.Fatalf("state %s recovery %v", res.State, requiredValue(t, res.RecoveryRequired))
	}
	txn, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil || txn.ErrorCode != string(apierr.HealthCheckFailed) || txn.Stage != updatejournal.StageRolledBack {
		t.Fatalf("txn stage %s code %s err %v", txn.Stage, txn.ErrorCode, err)
	}
}

func TestUPD_SemVerPrereleaseOrdering(t *testing.T) {
	if versionAtLeast("1.0.0-alpha.1", "1.0.0") {
		t.Fatal("pre-release compared as a release")
	}
	if !versionAtLeast("1.0.0", "1.0.0-alpha.1") || !versionAtLeast("1.10.0", "1.9.0") {
		t.Fatal("numeric or release ordering")
	}
	w := newWorld(t)
	w.eng.Platform.UpdaterVersion = "1.0.0-alpha.1"
	next := w.makeRelease("new", 1, []int{1})
	w.setWork(w.work(next, "1", api.PublicationIntentPublished))
	err := w.run()
	if err == nil {
		t.Fatal("pre-release updater installed a release")
	}
	res := w.oneResult()
	if res.State == api.UpdateResultStateSucceeded || res.State == api.UpdateResultStateAlreadyCurrent {
		t.Fatalf("state %s", res.State)
	}
}

func TestUPD_RejectsPrivateAndCGNAT(t *testing.T) {
	for _, raw := range []string{
		"https://10.1.2.3/a",
		"https://192.168.0.1/a",
		"https://172.16.0.1/a",
		"https://100.64.0.1/a",
		"https://100.127.255.1/a",
		"http://10.1.2.3/a",
		"http://100.64.12.1/a",
	} {
		if err := checkURL(raw, true); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if err := checkURL("https://100.128.0.1/a", false); err != nil {
		t.Fatal(err)
	}
	if err := checkURL("https://8.8.8.8/a", false); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, addr := range []string{"10.1.2.3:443", "100.64.0.5:443", "192.168.1.1:443"} {
		if _, err := dialChecked(ctx, "tcp", addr, true); err == nil {
			t.Fatalf("dialed %s", addr)
		}
	}
	if _, err := dialChecked(ctx, "tcp", "127.0.0.1:1", false); err == nil || asOp(err) == nil {
		t.Fatalf("loopback without the test exception: %v", err)
	}
	_, err := dialChecked(ctx, "tcp", "127.0.0.1:1", true)
	if err == nil || asOp(err) != nil {
		t.Fatalf("allowLoopback refused loopback before dial: %v", err)
	}
	var netErr *net.OpError
	if !errors.As(err, &netErr) {
		t.Fatalf("loopback dial: %v", err)
	}
}

func TestUPD_NoMigratorRollsBackBeforeStop(t *testing.T) {
	w := newWorld(t)
	w.eng.Migrator = noMigrator{}
	old := w.makeRelease("old", 1, []int{1})
	next := w.makeRelease("new", 2, []int{1})
	w.seed(old)
	work := w.work(next, "18", api.PublicationIntentPublished)
	work.Serving = []uuid.UUID{old.id, next.id}
	w.setWork(work)
	if err := w.run(); err != nil {
		t.Fatal(err)
	}
	stops, _ := w.svc.counts()
	if stops != 0 {
		t.Fatalf("services stopped before the migrator refused: %d", stops)
	}
	w.mustCurrent(old.id)
	txn, err := w.eng.Journal.LastTxn(context.Background())
	if err != nil || txn.ErrorCode != string(apierr.MigrationFailed) || txn.Destructive {
		t.Fatalf("txn %+v %v", txn, err)
	}
	if blocked, _ := w.eng.Journal.BlocksForwarding(context.Background()); blocked {
		t.Fatal("members blocked after a non-destructive migration refusal")
	}
}
