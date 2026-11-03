package main

// Review r20's command proof (Q5: signals), kept by agent-core follow-up 37
// as a passing regression. It also covers the run command's exit-status rule
// used by node/agent's TestR20_M1_* stage proofs.

import (
	"context"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

type r20CloseLatchEngine struct {
	inClose chan struct{}
	release chan struct{}
	closed  atomic.Int32
}

func (e *r20CloseLatchEngine) Boot(context.Context) error { return nil }
func (e *r20CloseLatchEngine) Step(context.Context) error { return nil }
func (e *r20CloseLatchEngine) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}
func (e *r20CloseLatchEngine) Close() error {
	close(e.inClose)
	<-e.release
	e.closed.Add(1)
	return nil
}

type r20StepCancelEngine struct {
	inStep chan struct{}
}

func (e *r20StepCancelEngine) Boot(context.Context) error { return nil }
func (e *r20StepCancelEngine) Run(context.Context) error  { return nil }
func (e *r20StepCancelEngine) Close() error               { return nil }
func (e *r20StepCancelEngine) Step(ctx context.Context) error {
	close(e.inStep)
	<-ctx.Done()
	return ctx.Err()
}

func TestR20_Q5_Held_RepeatedSignalsDuringCloseAndStepExitCodes(t *testing.T) {
	t.Run("three_sigterms_and_sigint_during_close_exit_zero", func(t *testing.T) {
		inClose := make(chan struct{})
		release := make(chan struct{})
		eng := &r20CloseLatchEngine{inClose: inClose, release: release}
		done := make(chan int, 1)
		go func() {
			done <- runWithOpen([]string{"-config", "ignored"}, true, func(string) (runEngine, error) {
				return eng, nil
			})
		}()
		// Give runWithOpen time to install NotifyContext and enter Run.
		time.Sleep(25 * time.Millisecond)
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
			t.Fatal(err)
		}
		select {
		case <-inClose:
		case <-time.After(2 * time.Second):
			t.Fatal("Close did not start after first SIGTERM")
		}
		// Send a second SIGTERM, a third SIGTERM, and a SIGINT while Close is blocked.
		for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGTERM, syscall.SIGINT} {
			if err := syscall.Kill(syscall.Getpid(), sig); err != nil {
				t.Fatal(err)
			}
		}
		select {
		case code := <-done:
			t.Fatalf("runWithOpen returned %d before Close finished", code)
		case <-time.After(50 * time.Millisecond):
		}
		close(release)
		select {
		case code := <-done:
			if code != 0 || eng.closed.Load() != 1 {
				t.Fatalf("code=%d closed=%d, want code=0 closed=1", code, eng.closed.Load())
			}
		case <-time.After(2 * time.Second):
			t.Fatal("runWithOpen did not return after Close released")
		}
	})

	t.Run("step_subcommand_canceled_exits_nonzero", func(t *testing.T) {
		inStep := make(chan struct{})
		eng := &r20StepCancelEngine{inStep: inStep}
		done := make(chan int, 1)
		go func() {
			done <- runWithOpen([]string{"-config", "ignored"}, false, func(string) (runEngine, error) {
				return eng, nil
			})
		}()
		select {
		case <-inStep:
		case <-time.After(2 * time.Second):
			t.Fatal("Step did not start")
		}
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
			t.Fatal(err)
		}
		select {
		case code := <-done:
			if code != 1 {
				t.Fatalf("canceled step returned %d, want 1", code)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("step did not return after SIGTERM")
		}
	})
}
