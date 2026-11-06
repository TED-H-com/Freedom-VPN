package enforce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestD106_TCReadFailureProcess(t *testing.T) {
	switch os.Getenv("FVPN_TC_READ_HELPER") {
	case "exit":
		os.Exit(7)
	case "wait":
		fmt.Println("ready")
		time.Sleep(time.Hour)
	}
}

type tcReadReadyWriter struct {
	ready chan struct{}
	once  sync.Once
}

func (w *tcReadReadyWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { close(w.ready) })
	return len(p), nil
}

func TestD106_TCReadFailureRunningCommandCancellation(t *testing.T) {
	t.Setenv("FVPN_TC_READ_HELPER", "wait")
	for _, measured := range []bool{false, true} {
		t.Run(fmt.Sprintf("lifecycle_%t", measured), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			var commands []CommandLifecycle
			if measured {
				ctx = commandPhase(beginCommandMeasurement(WithCommandLifecycleForTest(ctx), &commands), "install")
			}
			ready := &tcReadReadyWriter{ready: make(chan struct{})}
			joined := make(chan struct{})
			go func() {
				defer close(joined)
				select {
				case <-ready.ready:
					cancel()
				case <-ctx.Done():
				}
			}()
			f := &fakeExec{handler: func(string, []string) ([]byte, error) {
				cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestD106_TCReadFailureProcess$")
				cmd.Stdout = ready
				cause := runCommand(ctx, cmd, "tc")
				return nil, &CmdError{Cmd: "TEST ONLY private argv", Err: cause, Out: "TEST ONLY private output"}
			}}
			_, _, err := readServingTC(ctx, f, smallModel())
			<-joined
			select {
			case <-ready.ready:
			default:
				t.Fatal("command never reached the running handshake")
			}
			d, ok := ReadTCFailure(err)
			if !ok || d.Classification != "command_exit" || d.ContextState != "canceled" || d.ExitCode != nil || errors.Is(err, context.Canceled) {
				t.Fatalf("signaled exit mislabeled: %+v %v", d, err)
			}
			if measured && (len(commands) != 1 || !commands[0].Started || !commands[0].CancelObserved.Available || commands[0].ExitCode >= 0) {
				t.Fatalf("missing killed-running-command evidence: %+v", commands)
			}
		})
	}
}

func TestD106_TCReadFailureOriginalCauseAndDetachedMetadata(t *testing.T) {
	t.Setenv("FVPN_TC_READ_HELPER", "exit")
	for _, measured := range []bool{false, true} {
		t.Run(fmt.Sprintf("lifecycle_%t", measured), func(t *testing.T) {
			for _, kind := range []string{"deadline", "canceled", "command_exit", "read_error", "typed_nil_exit"} {
				t.Run(kind, func(t *testing.T) {
					ctx := context.Background()
					if kind == "deadline" {
						var cancel context.CancelFunc
						ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-time.Second))
						defer cancel()
					} else if kind == "canceled" {
						var cancel context.CancelFunc
						ctx, cancel = context.WithCancel(ctx)
						cancel()
					}
					var commands []CommandLifecycle
					if measured {
						ctx = commandPhase(beginCommandMeasurement(WithCommandLifecycleForTest(ctx), &commands), "install")
					}
					var cause error
					switch kind {
					case "read_error":
						cause = errors.New("TEST ONLY private command output")
					case "typed_nil_exit":
						cause = (*exec.ExitError)(nil)
					default:
						_, cause = (System{}).Output(ctx, os.Args[0], "-test.run=^TestD106_TCReadFailureProcess$")
						if cause == nil {
							t.Fatal("expected real command failure")
						}
						if measured && len(commands) != 1 {
							t.Fatalf("missing lifecycle record: %+v", commands)
						}
					}
					wrapped := &CmdError{Cmd: "TEST ONLY private argv", Out: "TEST ONLY private stderr", Err: fmt.Errorf("wrapped: %w", cause)}
					r := TCReadEvidence{Interface: "fvup0", Attachment: "7560:", Error: "read_failed"}
					got := tcReadError(ctx.Err(), r, wrapped)
					want := tcCheck(r, nil)
					if got.Error() != want.Error() {
						t.Fatalf("error contract changed: %q / %q", got, want)
					}
					var check *CheckError
					if !errors.As(got, &check) || check.Reason != ReasonTCForeignFilter || errors.Unwrap(got) != nil || errors.Is(got, cause) {
						t.Fatalf("error matching changed: %v", got)
					}
					d, ok := ReadTCFailure(fmt.Errorf("outer: %w", got))
					wantKind, wantContext := kind, "none"
					if kind == "typed_nil_exit" {
						wantKind = "read_error"
					}
					if kind == "deadline" || kind == "canceled" {
						wantContext = kind
					}
					if !ok || d.Classification != wantKind || d.ContextState != wantContext || d.Interface != "fvup0" || d.Attachment != "7560:" {
						t.Fatalf("diagnostic=%+v available=%v", d, ok)
					}
					if kind == "command_exit" {
						if d.ExitCode == nil || *d.ExitCode != 7 {
							t.Fatalf("missing observed exit: %+v", d)
						}
						*d.ExitCode = 99
					} else if d.ExitCode != nil {
						t.Fatalf("invented exit code: %+v", d)
					}
					d.Classification, d.Interface = "changed", "changed"
					again, _ := ReadTCFailure(got)
					if again.Classification != wantKind || again.Interface != "fvup0" || (again.ExitCode != nil && *again.ExitCode != 7) {
						t.Fatal("accessor exposed retained state")
					}
					oldEvidence, newEvidence := ServingEvidence{TC: []TCReadEvidence{r}}, ServingEvidence{TC: []TCReadEvidence{r}}
					oldEvidence.retainTC(42, "verify", want)
					newEvidence.retainTC(42, "verify", got)
					if !reflect.DeepEqual(oldEvidence.TC, newEvidence.TC) || !reflect.DeepEqual(oldEvidence.TCSummary, newEvidence.TCSummary) {
						t.Fatal("public TC rows or digest changed")
					}
					raw, err := json.Marshal(struct {
						Evidence   ServingEvidence
						Diagnostic TCReadFailure
						Error      error
					}{newEvidence, again, got})
					if err != nil || strings.Contains(string(raw), "private") || strings.Contains(string(raw), "wrapped") {
						t.Fatalf("unsafe diagnostic serialization: %s (%v)", raw, err)
					}
				})
			}
		})
	}
	for _, err := range []error{nil, (*CheckError)(nil), errors.New("ordinary"), &CheckError{Reason: ReasonQoS}} {
		if _, ok := ReadTCFailure(err); ok {
			t.Fatalf("invented diagnostics for %T", err)
		}
	}
}

func TestD106_TCReadFailureAllReadSitesAndContextObservation(t *testing.T) {
	t.Setenv("FVPN_TC_READ_HELPER", "exit")
	_, exit := (System{}).Output(context.Background(), os.Args[0], "-test.run=^TestD106_TCReadFailureProcess$")
	if exit == nil {
		t.Fatal("expected real exit")
	}
	for _, site := range []string{"verify_qdisc", "verify_class", "verify_parent", "load_qdisc", "load_class"} {
		t.Run(site, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			m := smallModel()
			attachment := "qdisc"
			if strings.HasSuffix(site, "class") {
				attachment = "class"
			}
			if strings.HasSuffix(site, "parent") {
				attachment = "7560:"
			}
			f := &fakeExec{handler: func(_ string, args []string) ([]byte, error) {
				key := strings.Join(args, " ")
				fail := (attachment == "qdisc" && args[1] == "qdisc") || (attachment == "class" && args[1] == "class") || strings.HasSuffix(key, "parent 7560:")
				if fail {
					cancel() // A canceled ambient context must not relabel this real exit.
					return nil, exit
				}
				if args[1] == "qdisc" {
					return []byte(`[{"kind":"fq_codel","handle":"7560:","parent":"1:20"}]`), nil
				}
				return []byte(`[]`), nil
			}}
			var err error
			var rows []TCReadEvidence
			if strings.HasPrefix(site, "load_") {
				_, rows, err = loadTC(ctx, f, []string{m.Datapath.MemberIngress})
			} else {
				_, rows, err = readServingTC(ctx, f, m)
			}
			d, ok := ReadTCFailure(err)
			if !ok || d.Classification != "command_exit" || d.ContextState != "canceled" || d.ExitCode == nil || *d.ExitCode != 7 || d.Attachment != attachment || d.Interface != m.Datapath.MemberIngress {
				t.Fatalf("diagnostic=%+v err=%v", d, err)
			}
			if len(rows) == 0 || rows[len(rows)-1].Error != "read_failed" || errors.Is(err, context.Canceled) {
				t.Fatalf("changed existing rejection: %+v %v", rows, err)
			}
		})
	}
}

func TestSEC02_TCReadFailureTokenAndParseBoundaries(t *testing.T) {
	err := tcReadError(nil, TCReadEvidence{Interface: "unsafe/interface", Attachment: "unsafe selector", Error: "read_failed"}, errors.New("TEST ONLY private stderr"))
	d, _ := ReadTCFailure(err)
	if d.Interface != "redacted" || d.Attachment != "redacted" {
		t.Fatalf("unsafe tokens: %+v", d)
	}
	f := &fakeExec{handler: func(string, []string) ([]byte, error) { return []byte("not JSON"), nil }}
	_, rows, err := readServingTC(context.Background(), f, smallModel())
	if _, ok := ReadTCFailure(err); ok || len(rows) != 1 || rows[0].Error != "invalid_json" {
		t.Fatalf("parse failure was relabeled: %+v %v", rows, err)
	}
}
