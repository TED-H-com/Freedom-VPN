package installer

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestPKG01_AT07ShippedAgentStopBudget(t *testing.T) {
	dir := t.TempDir()
	env := baseEnv(t, dir)
	fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
	if _, err := Install(context.Background(), env, mustReq(t, dir, fx)); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(unitDir(t), UnitAgent),
		filepath.Join(env.Layout.unitSourceDir(fx.ReleaseID), UnitAgent),
		filepath.Join(env.Layout.SystemdDir, UnitAgent),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		section, count := "", 0
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[") {
				section = line
			}
			if strings.HasPrefix(line, "TimeoutStopSec=") {
				count++
				if section != "[Service]" || line != "TimeoutStopSec=15s" {
					t.Fatalf("%s: %s in %s", path, line, section)
				}
			}
		}
		if count != 1 {
			t.Fatalf("%s: %d stop timeout directives", path, count)
		}
	}
}

func TestPKG01_StopWaitsThroughUnitBudget(t *testing.T) {
	for _, mode := range []string{"completed", "caller_deadline", "stop_deadline", "already_canceled"} {
		t.Run(mode, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx := context.Background()
				switch mode {
				case "caller_deadline":
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, time.Second)
					defer cancel()
				case "already_canceled":
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					cancel()
				}
				called := false
				env := &Env{Command: func(stopCtx context.Context, name string, args ...string) ([]byte, error) {
					called = true
					if name != "systemctl" || strings.Join(args, " ") != "stop "+UnitAgent {
						t.Fatalf("unexpected command %s %v", name, args)
					}
					deadline, ok := stopCtx.Deadline()
					if !ok || time.Until(deadline) != 20*time.Second {
						t.Fatalf("stop must have its own 20 s deadline: %v %v", deadline, ok)
					}
					if mode == "stop_deadline" {
						<-stopCtx.Done()
						// exec.CommandContext often returns a process error, not ctx.Err().
						return nil, errors.New("signal: killed")
					}
					time.Sleep(16 * time.Second)
					if stopCtx.Err() != nil {
						t.Fatal("stop abandoned before unit budget plus margin")
					}
					return nil, nil
				}}
				start := time.Now()
				err := stopForBinaryReplace(ctx, systemdManager{env: env}, UnitAgent)
				wantDuration := 16 * time.Second
				var wantErr error
				switch mode {
				case "caller_deadline":
					wantErr = context.DeadlineExceeded
				case "stop_deadline":
					wantDuration, wantErr = 20*time.Second, context.DeadlineExceeded
				case "already_canceled":
					wantDuration, wantErr = 0, context.Canceled
				}
				if !errors.Is(err, wantErr) || time.Since(start) != wantDuration || called != (mode != "already_canceled") {
					t.Fatalf("stop: err=%v duration=%v called=%v", err, time.Since(start), called)
				}
			})
		})
	}
}

func TestPKG01_AT25ReplaceStopFailureIsJournaled(t *testing.T) {
	for _, failure := range []error{context.DeadlineExceeded, errors.New("systemctl stop failed")} {
		t.Run(failure.Error(), func(t *testing.T) {
			dir := t.TempDir()
			env := baseEnv(t, dir)
			fx := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c1", "1")
			if _, err := Install(context.Background(), env, mustReq(t, dir, fx)); err != nil {
				t.Fatal(err)
			}
			oldInstalled := hashFile(t, env.Layout.installedFile())
			oldAgent := hashFile(t, filepath.Join(env.Layout.SystemdDir, UnitAgent))
			next := mustRelease(t, dir, "019a0000-0000-7000-8000-0000000000c2", "2")
			command := env.Command
			env.Services = systemdManager{env: env}
			var stops []string
			env.Command = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				if name == "systemctl" && args[0] == "stop" {
					stops = append(stops, args[1])
					if args[1] == UnitAgent {
						return nil, failure
					}
				}
				return command(ctx, name, args...)
			}
			res, err := Install(context.Background(), env, mustReq(t, dir, next))
			if res != nil || err == nil || AsError(err).Exit != ExitFailed || !strings.Contains(err.Error(), failure.Error()) {
				t.Fatalf("replacement stop: result=%+v err=%v", res, err)
			}
			if stops[len(stops)-1] != UnitAgent {
				t.Fatalf("continued stopping after failure: %v", stops)
			}
			if target, err := os.Readlink(env.Layout.CurrentLink); err != nil || target != "releases/"+fx.ReleaseID {
				t.Fatalf("current changed: %s %v", target, err)
			}
			if oldInstalled != hashFile(t, env.Layout.installedFile()) || oldAgent != hashFile(t, filepath.Join(env.Layout.SystemdDir, UnitAgent)) {
				t.Fatal("failed stop changed installed release or unit")
			}
			if _, err := os.Stat(env.Layout.releaseDir(next.ReleaseID)); !os.IsNotExist(err) {
				t.Fatalf("failed stop extracted candidate: %v", err)
			}
			raw, err := os.ReadFile(env.Layout.journalFile())
			if err != nil {
				t.Fatal(err)
			}
			var journal journalFile
			if err := json.Unmarshal(raw, &journal); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, rec := range journal.Records {
				if rec.Action == "stop-failed" && rec.Path == UnitAgent && strings.Contains(rec.Note, failure.Error()) {
					found = true
				}
			}
			if !found || journal.Phase != "rolled_back" {
				t.Fatalf("missing durable stop failure: %s", raw)
			}
		})
	}
}
