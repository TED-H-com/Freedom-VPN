package main

// Adversarial security review r21 proofs for cmd/fvpn-agent follow-up 37
// (05b2c2f2..7b3d7bf8).
// TEST ONLY fixtures; no test touches host networking or the kernel.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
)

type r21SignalDuringFailedBootEngine struct {
	bootErr   error
	closeRuns int
}

func (e *r21SignalDuringFailedBootEngine) Run(ctx context.Context) error {
	if err := e.Boot(ctx); err != nil {
		return err
	}
	return ctx.Err()
}

func (e *r21SignalDuringFailedBootEngine) Boot(ctx context.Context) error {
	// Deliver SIGTERM while Boot is running, then fail Boot (e.g. loadRuntime
	// failure or unestablished baseline).
	_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	<-ctx.Done()
	return e.bootErr
}

func (e *r21SignalDuringFailedBootEngine) Step(ctx context.Context) error {
	return ctx.Err()
}

func (e *r21SignalDuringFailedBootEngine) Close() error {
	e.closeRuns++
	return nil
}

// M-2: Follow-up 37 (docs/node/followup37-r20-adoption-review-2026-11-03.md)
// states that on SIGTERM during Boot, fvpn-agent run exits 0 only after the
// fail-closed baseline is verified and the non-serving heartbeat is written.
// However:
//  1. When loadRuntime(load) fails in Boot (engine.go:441-445) while SIGTERM
//     is pending (ctx.Err() != nil), Boot returns the store error without
//     writing a heartbeat, and main.go:121 (`if err := eng.Run(ctx);
//     err != nil && ctx.Err() == nil`) ignores the error because
//     ctx.Err() != nil and returns exit status 0.
//  2. When e.plane.Baseline(ctx, dp) fails in Boot (engine.go:469-492,
//     e.baselineOK = false, leaving the previous process's unverified serving
//     ruleset active in the kernel), Boot returns nil, Run returns ctx.Err()
//     on the first Step, and main.go:121 returns exit status 0.
func TestR21_M2_RunExitsZeroWhenCanceledBootFailsBaselineOrLoadRuntime(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "agent.json")
	eng := &r21SignalDuringFailedBootEngine{
		bootErr: errors.New("boot failed to load runtime or establish fail-closed baseline"),
	}
	code := runWithOpen([]string{"-config", cfgPath}, true, func(got string) (runEngine, error) {
		return eng, nil
	})
	if eng.closeRuns != 1 {
		t.Fatalf("expected Close to run once, got %d", eng.closeRuns)
	}
	// TODO(review r21): M-2 — when Boot fails to establish a verified baseline
	// or returns a non-cancellation error during SIGTERM, fvpn-agent run must
	// exit non-zero (1), not 0.
	if code == 0 {
		t.Fatalf("M-2: runWithOpen returned exit code 0 when Boot failed with %v during SIGTERM; want exit code 1", eng.bootErr)
	}
}

func TestR21_M2_RunCanceledBootExitStatusMatrix(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		code int
	}{
		{"runtime_failure", errors.New("runtime unavailable"), 1},
		{"baseline_failure", errors.New("baseline unavailable"), 1},
		{"boot_budget_expired", errors.Join(agent.ErrBoot, context.DeadlineExceeded), 1},
		{"boot_local_canceled", errors.Join(agent.ErrBoot, context.Canceled), 1},
		{"baseline_completed", nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			eng := &r21SignalDuringFailedBootEngine{bootErr: tc.err}
			code := runWithOpen([]string{"-config", "test.json"}, true, func(string) (runEngine, error) { return eng, nil })
			if code != tc.code || eng.closeRuns != 1 {
				t.Fatalf("code=%d close=%d", code, eng.closeRuns)
			}
		})
	}
}
