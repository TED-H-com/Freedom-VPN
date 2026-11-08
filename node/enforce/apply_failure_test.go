package enforce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type applyFailureExec struct {
	*fakeExec
	stage      string
	cause      error
	cancel     context.CancelFunc
	rolledBack bool
}

func (f *applyFailureExec) NftBatch(ctx context.Context, script string) error {
	if strings.Contains(script, "fvpn_baseline=1") {
		f.rolledBack = true
		f.cancel() // Must not replace the context observation from the primary error.
	}
	if (f.stage == "install" && strings.Contains(script, "add chain inet fvpn "+deadlineGuard+" {")) || (f.stage == "release" && script == "delete chain inet fvpn "+deadlineGuard+"\n") {
		return f.cause
	}
	return f.fakeExec.NftBatch(ctx, script)
}
func (f *applyFailureExec) TCBatch(ctx context.Context, script string) error {
	if f.stage == "shape" {
		return f.cause
	}
	return f.fakeExec.TCBatch(ctx, script)
}
func (f *applyFailureExec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := strings.Join(args, " ")
	if (f.stage == "load_nft" && name == "nft" && key == "-j list table inet fvpn") || (f.stage == "verify" && name == "nft" && key == "-j list ruleset") {
		return nil, f.cause
	}
	return f.fakeExec.Output(ctx, name, args...)
}

func TestSEC02_ApplyFailureBoundaryBeforeCleanup(t *testing.T) {
	for _, stage := range []string{"", "load_nft", "shape", "install", "release", "verify"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			cause := &CmdError{Cmd: "TEST ONLY private argv", Out: "TEST ONLY private output", Err: context.DeadlineExceeded}
			m := smallModel()
			f := &applyFailureExec{fakeExec: servingExec(m), stage: stage, cause: cause, cancel: cancel}
			// Freeze Apply's samples in the same clock domain as its verifier.
			// An arbitrary one-hour anchor can be ahead of a freshly booted
			// runner's real monotonic read and correctly fail verification.
			anchor, err := MonotonicNow()
			if err != nil {
				t.Fatal(err)
			}
			res, err := apply(ctx, f, m, nil, func() (time.Duration, error) { return anchor, nil })
			d, ok := ReadApplyFailure(err)
			if stage == "" {
				if err != nil || ok || !res.Verified {
					t.Fatalf("success changed: %+v %v", d, err)
				}
				return
			}
			if !ok || d.Stage != stage || d.Classification != "deadline" || d.ContextState != "none" || d.ExitCode != nil || !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, cause) || res.Verified {
				t.Fatalf("failure stage or original error lost: %+v %v", d, err)
			}
			if stage != "load_nft" && (!f.rolledBack || ctx.Err() != context.Canceled) {
				t.Fatal("fixture failed to exercise cleanup cancellation")
			}
			raw, err := json.Marshal(d)
			if err != nil || strings.Contains(string(raw), "private") {
				t.Fatalf("unsafe diagnosis: %s %v", raw, err)
			}
		})
	}
}

// The precision verifier already accepts an explicit read time. Reproduce the
// former fixture's short-uptime failure without changing clocks or waiting.
func TestSEC02_ApplyFailureFixtureShortUptime(t *testing.T) {
	const listing = "table inet fvpn { set allowed_v4 { type ipv4_addr flags timeout } }"
	for _, readStart := range []time.Duration{time.Second, 2 * time.Hour} {
		t.Run(readStart.String(), func(t *testing.T) {
			m := smallModel()
			m.NowMono = time.Hour // Former unrelated Apply fixture clock.
			_, err := verifyAllowedLifetimes([]byte(listing), m, readStart)
			if readStart < m.NowMono {
				var check *CheckError
				if !errors.As(err, &check) || check.Reason != ReasonIsolation || check.Detail != "monotonic read precedes model" {
					t.Fatalf("future anchor did not fail closed: %v", err)
				}
			} else if err != nil {
				t.Fatalf("long-uptime control failed: %v", err)
			}
			m.NowMono = readStart // Sampled from the verifier's clock domain.
			if _, err := verifyAllowedLifetimes([]byte(listing), m, readStart); err != nil {
				t.Fatalf("same-domain anchor failed: %v", err)
			}
		})
	}
}

func TestSEC02_ApplyFailureErrorIdentityAndClassification(t *testing.T) {
	t.Setenv("FVPN_TC_READ_HELPER", "exit")
	exit := exec.Command(os.Args[0], "-test.run=^TestD106_TCReadFailureProcess$").Run()
	if exit == nil {
		t.Fatal("expected actual process exit")
	}
	check := checkf(ReasonIsolation, "TEST ONLY detail")
	for _, tc := range []struct {
		cause error
		class string
	}{
		{context.DeadlineExceeded, "deadline"}, {context.Canceled, "canceled"}, {exit, "command_exit"}, {check, "check"}, {errors.New("TEST ONLY private text"), "operation_error"},
	} {
		t.Run(tc.class, func(t *testing.T) {
			cause := fmt.Errorf("TEST ONLY wrapper: %w", tc.cause)
			err := applyFailure("install", context.Canceled, cause)
			d, ok := ReadApplyFailure(errors.Join(err, context.DeadlineExceeded))
			if !ok || d.Classification != tc.class || d.ContextState != "canceled" || !errors.Is(err, tc.cause) || err.Error() != cause.Error() {
				t.Fatalf("identity/classification changed: %+v %v", d, err)
			}
			if tc.class == "check" {
				var got *CheckError
				if !errors.As(err, &got) || got != check {
					t.Fatal("lost CheckError")
				}
			}
			if tc.class == "command_exit" {
				if d.ExitCode == nil || *d.ExitCode != 7 {
					t.Fatalf("lost exit code: %+v", d)
				}
				*d.ExitCode = 9
				again, _ := ReadApplyFailure(err)
				if *again.ExitCode != 7 {
					t.Fatal("diagnostic was not detached")
				}
			} else if d.ExitCode != nil {
				t.Fatal("invented exit code")
			}
		})
	}
	if applyFailure("install", nil, nil) != nil {
		t.Fatal("created error on success")
	}
	if _, ok := ReadApplyFailure(errors.New("ordinary")); ok {
		t.Fatal("invented diagnosis")
	}
}
