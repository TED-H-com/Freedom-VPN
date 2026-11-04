package updater

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/nodelock"
)

// This child holds a real flock until the parent closes stdin. Readiness is
// sent only after holder metadata is durable; no startup sleep guesses.
func TestD158_HolderProcess(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "--d158-holder" {
		return
	}
	lk, err := nodelock.TryAcquire(os.Args[len(os.Args)-1])
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("ready")
	_, _ = io.Copy(io.Discard, os.Stdin)
	if err := lk.Release(); err != nil {
		t.Fatal(err)
	}
}

func d158Holder(t *testing.T, path string) func() {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestD158_HolderProcess$", "--", "--d158-holder", path)
	in, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	release := func() { _ = in.Close() }
	t.Cleanup(func() {
		release()
		err := cmd.Wait()
		cancel()
		if err != nil {
			t.Errorf("holder: %v; %s", err, &stderr)
		}
	})
	if ready, err := bufio.NewReader(out).ReadString('\n'); err != nil || ready != "ready\n" {
		t.Fatalf("holder readiness: %q %v", ready, err)
	}
	return release
}

func d158Record(t *testing.T, log *bytes.Buffer, command string, acquired bool, minimum time.Duration) {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(log.String()), "\n") {
		var record map[string]any
		if json.Unmarshal([]byte(line), &record) == nil && record["msg"] == "update lock wait" {
			records = append(records, record)
		}
	}
	if len(records) != 1 {
		t.Fatalf("want one wait record, got %d: %s", len(records), log)
	}
	r := records[0]
	waited, ok := r["waited_ms"].(float64)
	if !ok || waited < float64(minimum.Milliseconds()) || r["command"] != command || r["acquired"] != acquired || r["budget_ms"] != float64(10000) {
		t.Fatalf("wait record: %v", r)
	}
	raw, _ := json.Marshal(r)
	t.Logf("wait record: %s", raw)
}

func d158Engine(t *testing.T) *Engine {
	t.Helper()
	e := &Engine{Layout: testLayout(t.TempDir()), Source: &memSource{err: ErrNoWork}, Sender: HTTPSink{}}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func d158Command(e *Engine, name string) func(context.Context) error {
	switch name {
	case "run":
		return e.Run
	case "seed":
		return func(ctx context.Context) error { return e.Seed(ctx, "absent-manifest", "absent-tree") }
	case "apply":
		return func(ctx context.Context) error { return e.ApplyPending(ctx, "absent-pending") }
	default:
		return e.Flush
	}
}

func TestD158_CommandsProceedAfter600msHolder(t *testing.T) {
	for _, command := range []string{"run", "seed", "apply", "flush"} {
		t.Run(command, func(t *testing.T) {
			w := newWorld(t)
			quietDesired(w)
			call := w.eng.Run
			check := func() {}
			switch command {
			case "seed":
				r := w.makeRelease("installed", 1, []int{1})
				manifest := installSeedTree(t, w, r, true)
				call = func(ctx context.Context) error { return w.eng.Seed(ctx, manifest, w.layout.releaseDir(r.id.String())) }
				check = func() {
					if snap := seedSnapshot(t, w); snap.SeededBaseline == nil || snap.CurrentRelease != r.id.String() {
						t.Fatal("seed did not record the verified baseline")
					}
				}
			case "apply":
				touchPersistent(t, w.layout)
				r := w.makeRelease("staged", 1, []int{1})
				art, tree, _, cmd := stageCopy(t, w, r)
				pending := filepath.Join(w.layout.Staging, PendingReleaseFile)
				writePendingFile(t, pending, r, art, tree, cmd, "", r.env, false)
				call = func(ctx context.Context) error { return w.eng.ApplyPending(ctx, pending) }
				check = func() { w.mustCurrent(r.id); assertConsumed(t, pending, "succeeded") }
			case "flush":
				w.eng.Sender = HTTPSink{}
				call = w.eng.Flush // Standalone, with a real journal and empty outbox.
			}
			var log bytes.Buffer
			w.eng.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
			release := d158Holder(t, w.layout.Lock)
			started := time.Now()
			timer := time.AfterFunc(600*time.Millisecond, release)
			defer timer.Stop()
			err := call(t.Context())
			elapsed := time.Since(started)
			t.Logf("command=%s elapsed=%s exit=%d", command, elapsed, ExitCode(err))
			if err != nil || elapsed < 600*time.Millisecond {
				t.Fatalf("did not wait for release and proceed: %v (%s)", err, elapsed)
			}
			check()
			d158Record(t, &log, command, true, 500*time.Millisecond)
		})
	}
}

func d158StaleMetadata(t *testing.T, path, detail string) []byte {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(fileBytes(t, path), &fields); err != nil {
		t.Fatal(err)
	}
	switch detail {
	case "boot id":
		fields["boot_id"] = "TEST ONLY different boot"
	case "pid":
		fields["pid"] = 0
	case "process":
		fields["pid"] = 1 << 30
	case "start time":
		fields["start_time"] = 0
	}
	b, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestD158_StaleDetailsKeepPromptWindow(t *testing.T) {
	for _, command := range []string{"run", "seed", "apply", "flush"} {
		for _, detail := range []string{"boot id", "pid", "process", "start time"} {
			t.Run(command+"/"+detail, func(t *testing.T) {
				e := d158Engine(t)
				d158Holder(t, e.Layout.Lock)
				if err := os.WriteFile(e.Layout.Lock, d158StaleMetadata(t, e.Layout.Lock, detail), 0o600); err != nil {
					t.Fatal(err)
				}
				started := time.Now()
				err := d158Command(e, command)(t.Context())
				elapsed := time.Since(started)
				if ExitCode(err) != 1 || err.Error() != "updater: UPDATE_BUSY: lock "+detail || elapsed < 80*time.Millisecond || elapsed > time.Second {
					t.Fatalf("stale: %v elapsed=%s", err, elapsed)
				}
				if e.Journal != nil {
					t.Fatal("stale contention opened journal")
				}
				t.Logf("exit=1 elapsed=%s", elapsed)
			})
		}
	}
}

func TestD158_StaleWindowStartsAtFirstStaleAnswer(t *testing.T) {
	e := d158Engine(t)
	d158Holder(t, e.Layout.Lock)
	stale := d158StaleMetadata(t, e.Layout.Lock, "pid")
	changed := make(chan error, 1)
	timer := time.AfterFunc(300*time.Millisecond, func() { changed <- os.WriteFile(e.Layout.Lock, stale, 0o600) })
	defer timer.Stop()
	started := time.Now()
	err := e.Run(t.Context())
	elapsed := time.Since(started)
	if ExitCode(err) != 1 || err.Error() != "updater: UPDATE_BUSY: lock pid" || elapsed < 380*time.Millisecond || elapsed > time.Second {
		t.Fatalf("late stale answer: %v elapsed=%s", err, elapsed)
	}
	if err := <-changed; err != nil {
		t.Fatal(err)
	}
	t.Logf("live then stale: exit=1 elapsed=%s", elapsed)
}

func TestD158_RecoverAndRetainKeepPromptWindow(t *testing.T) {
	for _, command := range []string{"recover", "retain"} {
		t.Run(command, func(t *testing.T) {
			e := d158Engine(t)
			d158Holder(t, e.Layout.Lock)
			var log bytes.Buffer
			e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
			call := e.Recover
			if command == "retain" {
				call = e.Retain
			}
			started := time.Now()
			err := call(t.Context())
			elapsed := time.Since(started)
			if ExitCode(err) != 3 || elapsed < 80*time.Millisecond || elapsed > time.Second || log.Len() != 0 {
				t.Fatalf("prompt %s: %v elapsed=%s log=%s", command, err, elapsed, &log)
			}
			t.Logf("exit=3 elapsed=%s", elapsed)
		})
	}
}

func TestD158_CancellationAndUnreadableMetadata(t *testing.T) {
	for _, unreadable := range []bool{false, true} {
		t.Run(fmt.Sprint(unreadable), func(t *testing.T) {
			e := d158Engine(t)
			d158Holder(t, e.Layout.Lock)
			if unreadable {
				if err := os.WriteFile(e.Layout.Lock, []byte("{"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			var log bytes.Buffer
			e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			started := time.Now()
			timer := time.AfterFunc(300*time.Millisecond, cancel)
			defer timer.Stop()
			err := e.Run(ctx)
			elapsed := time.Since(started)
			if !errors.Is(err, context.Canceled) || elapsed < 300*time.Millisecond || elapsed > time.Second || e.Journal != nil {
				t.Fatalf("cancel: %v elapsed=%s journal=%v", err, elapsed, e.Journal)
			}
			d158Record(t, &log, "run", false, 280*time.Millisecond)
		})
	}
}

func TestD158_NestedCallsDoNotWaitOrLog(t *testing.T) {
	e := d158Engine(t)
	var log bytes.Buffer
	e.UseLog(slog.New(slog.NewJSONHandler(&log, nil)))
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	err := e.withLock(ctx, func() error {
		for _, call := range []func(context.Context) error{e.Run, e.Recover, e.Flush, e.Retain} {
			if err := call(ctx); err != nil {
				return err
			}
		}
		other, err := nodelock.TryAcquire(e.Layout.Lock)
		if other != nil {
			_ = other.Release()
		}
		if !errors.Is(err, nodelock.ErrBusy) {
			return fmt.Errorf("outer ownership lost: %v", err)
		}
		return nil
	})
	if err != nil || log.Len() != 0 {
		t.Fatalf("nested callers: %v log=%s", err, &log)
	}
}
