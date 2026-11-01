package enforce

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestLifecycleHelper(t *testing.T) {
	switch os.Getenv("FVPN_TEST_COMMAND_HELPER") {
	case "ok":
		os.Exit(0)
	case "exit":
		os.Exit(23)
	case "wait":
		time.Sleep(time.Hour)
		os.Exit(0)
	}
}

func helperCommand(ctx context.Context) *exec.Cmd {
	return exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLifecycleHelper$")
}

func TestD133_CommandLifecycleOutcomes(t *testing.T) {
	for _, mode := range []string{"ok", "exit", "wait", "canceled", "missing"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("FVPN_TEST_COMMAND_HELPER", mode)
			ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
			defer cancel()
			if mode == "canceled" {
				cancel()
			}
			var records []CommandLifecycle
			ctx = commandPhase(beginCommandMeasurement(WithCommandLifecycleForTest(ctx), &records), "install")
			cmd := helperCommand(ctx)
			if mode == "missing" {
				cmd = exec.CommandContext(ctx, t.TempDir()+"/absent")
			}
			err := runCommand(ctx, cmd, "nft")
			if len(records) != 1 {
				t.Fatal(records)
			}
			r := records[0]
			if r.OK != (mode == "ok") || (err == nil) != r.OK || r.Kind != "nft" || r.Phase != "install" {
				t.Fatalf("%+v %v", r, err)
			}
			if !r.ContextDeadline.Before.Available || r.ContextDeadline.Before.NS > r.ContextDeadline.After.NS || !r.StartAttempt.Available || r.StartAttempt.NS > r.StartReturn.NS || r.Before.Bracket.After.NS > r.StartAttempt.NS || r.After.Bracket.Before.NS < r.StartReturn.NS {
				t.Fatalf("invalid brackets: %+v", r)
			}
			if r.ChildStartExact.Available {
				t.Fatal("invented exact child creation")
			}
			started := mode != "canceled" && mode != "missing"
			if r.Started != started || r.CPUAvailable != started || r.ExitAvailable != started || r.WaitReturn.Available != started || r.ChildStartedBy.Available != started {
				t.Fatalf("start failure/child exit confused: %+v", r)
			}
			if started && (r.ChildStartedBy.NS < r.StartAttempt.NS || r.WaitReturn.NS < r.ChildStartedBy.NS || r.After.Bracket.Before.NS < r.WaitReturn.NS) {
				t.Fatalf("unordered child: %+v", r)
			}
			switch mode {
			case "ok":
				if r.ExitCode != 0 || r.Signal != 0 || r.ContextAfter != "none" {
					t.Fatal(r)
				}
			case "exit":
				var exit *exec.ExitError
				if !errors.As(err, &exit) || r.ExitCode != 23 || r.Signal != 0 || r.CancelObserved.Available {
					t.Fatalf("exit lost: %+v %v", r, err)
				}
			case "wait":
				if !r.CancelObserved.Available || r.CancelObserved.NS < r.StartAttempt.NS || r.CancelObserved.NS > r.WaitReturn.NS || r.ContextAtCancel != "deadline_exceeded" || r.ContextAfter != "deadline_exceeded" || !r.SignalAvailable || r.Signal != 9 || r.ExitCode != -1 {
					t.Fatalf("cancel lost: %+v", r)
				}
			case "canceled":
				if !errors.Is(err, context.Canceled) || r.ContextBefore != "canceled" || r.CancelObserved.Available {
					t.Fatalf("pre-start cancellation confused: %+v %v", r, err)
				}
			}
			raw, _ := json.Marshal(r)
			if strings.Contains(string(raw), os.Args[0]) || strings.Contains(string(raw), "-test.run") || !strings.Contains(string(raw), `"ns":"`) {
				t.Fatal("unsafe/imprecise command evidence")
			}
		})
	}
}

func TestD133_CommandLifecycleDisabledAndDetached(t *testing.T) {
	t.Setenv("FVPN_TEST_COMMAND_HELPER", "ok")
	ctx := t.Context()
	var records []CommandLifecycle
	if beginCommandMeasurement(ctx, &records) != ctx || commandPhase(ctx, "install") != ctx {
		t.Fatal("disabled measurement allocated context")
	}
	if err := runCommand(ctx, helperCommand(ctx), "other"); err != nil || len(records) != 0 {
		t.Fatalf("disabled record: %v %+v", err, records)
	}
	ev := ServingEvidence{timing: &TimingEvidence{Commands: []CommandLifecycle{{Before: CommandResources{Cgroups: []CgroupCPU{{Max: ResourceText{true, "max 100000"}}}}, After: CommandResources{Cgroups: []CgroupCPU{{Weight: ResourceText{true, "100"}}}}}}}}
	v := ev.TimingEvidence()
	v.Commands[0].Before.Cgroups[0].Max.Value = "changed"
	v.Commands[0].After.Cgroups[0].Weight.Value = "changed"
	if ev.TimingEvidence().Commands[0].Before.Cgroups[0].Max.Value != "max 100000" || ev.TimingEvidence().Commands[0].After.Cgroups[0].Weight.Value != "100" {
		t.Fatal("snapshot aliases command resources")
	}
	raw, _ := json.Marshal(newTiming("apply", 1))
	if strings.Contains(string(raw), `"commands"`) {
		t.Fatal("disabled measurement changed existing JSON")
	}
}

type lifecycleApplyExec struct {
	*fakeExec
	phases      []string
	failInstall bool
}

func (e *lifecycleApplyExec) NftBatch(ctx context.Context, script string) error {
	scope, ok := ctx.Value(commandScopeKey{}).(commandScope)
	if !ok {
		return errors.New("missing test measurement scope")
	}
	e.phases = append(e.phases, scope.phase)
	if scope.phase == "install" {
		if _, err := (System{}).Output(ctx, os.Args[0], "-test.run=^TestLifecycleHelper$"); err != nil {
			return err
		}
		if e.failInstall {
			return errors.New("TEST ONLY install rejection")
		}
	}
	return e.fakeExec.NftBatch(ctx, script)
}
func (e *lifecycleApplyExec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	scope, ok := ctx.Value(commandScopeKey{}).(commandScope)
	if !ok {
		return nil, errors.New("missing scope")
	}
	e.phases = append(e.phases, scope.phase)
	return e.fakeExec.Output(ctx, name, args...)
}
func TestD133_AT02_CommandLifecycleJoinsApplyAndRollback(t *testing.T) {
	t.Setenv("FVPN_TEST_COMMAND_HELPER", "ok")
	for _, fail := range []bool{false, true} {
		m := smallModel()
		e := &lifecycleApplyExec{fakeExec: servingExec(m), failInstall: fail}
		res, err := Apply(WithCommandLifecycleForTest(t.Context()), e, m, nil)
		v := res.Evidence.TimingEvidence()
		if (err != nil) != fail || len(v.Commands) != 1 || v.Commands[0].Phase != "install" || v.Commands[0].StartAttempt.NS < v.Deadline.InstallStart || v.Commands[0].WaitReturn.NS > v.Deadline.InstalledBy {
			t.Fatalf("lost Apply join: %+v %v", v, err)
		}
		phases := " " + strings.Join(e.phases, " ") + " "
		if !strings.Contains(phases, " prepare ") || strings.Contains(phases, " rollback ") != fail || strings.Contains(phases, " release ") == fail || strings.Contains(phases, " verify ") == fail {
			t.Fatalf("wrong phases: %s", phases)
		}
		if fail && (v.Deadline.GuardReleased || len(v.Verifications) != 0) {
			t.Fatal("invented verification after failed install")
		}
	}
}
