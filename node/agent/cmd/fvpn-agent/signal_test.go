package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TEST ONLY gates in the injected opener and Close make both signal windows
// deterministic. Signals target only this test's child, never the test runner.
type signalEngine struct{}

func signalGate(stage string) {
	fmt.Println(stage)
	var b [1]byte
	if _, err := os.Stdin.Read(b[:]); err != nil {
		os.Exit(91)
	}
}

func (signalEngine) Run(ctx context.Context) error {
	fmt.Println("running")
	<-ctx.Done()
	return ctx.Err()
}
func (signalEngine) Boot(context.Context) error { return nil }
func (signalEngine) Step(context.Context) error { return nil }
func (signalEngine) Close() error {
	signalGate("closing")
	fmt.Println("closed")
	return nil
}

func TestAT13_F36_SignalLifetime(t *testing.T) {
	if os.Getenv("FVPN_TEST_ONLY_SIGNAL_CHILD") == "1" {
		os.Exit(runWithOpen([]string{"-config", "test-only"}, true, func(string) (runEngine, error) {
			signalGate("opening")
			return signalEngine{}, nil
		}))
	}
	for _, phase := range []string{"startup", "close"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAT13_F36_SignalLifetime$")
			cmd.Env = append(os.Environ(), "FVPN_TEST_ONLY_SIGNAL_CHILD=1")
			in, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			defer in.Close()
			out, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			scan := bufio.NewScanner(out)
			expect := func(want string) {
				t.Helper()
				if !scan.Scan() || scan.Text() != want {
					err := cmd.Wait()
					t.Fatalf("want %s; got %q; wait=%v stderr=%s", want, scan.Text(), err, stderr.String())
				}
			}
			term := func() {
				t.Helper()
				if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
					t.Fatal(err)
				}
				// Let Go dispatch the signal before releasing the blocked phase.
				time.Sleep(40 * time.Millisecond)
			}
			expect("opening")
			if phase == "startup" {
				term()
			}
			_, _ = in.Write([]byte{1})
			expect("running")
			if phase == "close" {
				term()
			}
			expect("closing")
			term() // supervisor's additional TERM must not interrupt Close
			_, _ = in.Write([]byte{1})
			expect("closed")
			if err := cmd.Wait(); err != nil {
				t.Fatalf("wait=%v stderr=%s", err, stderr.String())
			}
		})
	}
}
