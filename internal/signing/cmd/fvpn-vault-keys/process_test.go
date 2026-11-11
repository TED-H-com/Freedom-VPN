package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
)

func buildCommand(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "fvpn-vault-keys")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build command: %v\n%s", err, out)
	}
	return binary
}
func processCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatal(err)
	}
	status := exit.Sys().(syscall.WaitStatus)
	if status.Signaled() {
		t.Fatalf("child died from signal %v", status.Signal())
	}
	return exit.ExitCode()
}
func TestSEC01_CommandProcessConcurrency(t *testing.T) {
	binary := buildCommand(t)
	for _, raceRevoke := range []bool{false, true} {
		f := setup(t, signing.Lease)
		n := 6
		if raceRevoke {
			n = 2
		}
		lock, err := os.OpenFile(filepath.Join(f.dir, "lock"), os.O_RDWR, 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		commands := make([]*exec.Cmd, n)
		stdout := make([]bytes.Buffer, n)
		stderr := make([]bytes.Buffer, n)
		errs := make([]error, n)
		for i := range n {
			op := "rotate"
			if raceRevoke && i == 0 {
				op = "revoke"
			}
			commands[i] = exec.CommandContext(ctx, binary, f.args(op)...)
			commands[i].Stdout = &stdout[i]
			commands[i].Stderr = &stderr[i]
			if err := commands[i].Start(); err != nil {
				t.Fatal(err)
			}
		}
		if err := lock.Close(); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		for i := range n {
			wg.Add(1)
			go func() { defer wg.Done(); errs[i] = commands[i].Wait() }()
		}
		wg.Wait()
		applied := 0
		for i := range n {
			code := processCode(t, errs[i])
			outcome := "precondition_refused"
			message := ""
			switch code {
			case exitOK:
				applied++
				outcome = "applied"
				if raceRevoke && i == 0 {
					outcome = "revoked"
				}
			case exitRefused:
			case exitTimeout:
				outcome = ""
				message = "fvpn-vault-keys: vault lock deadline exceeded\n"
			default:
				t.Fatalf("unexpected process code: %d", code)
			}
			assertOutput(t, code, stdout[i].String(), stderr[i].String(), code, outcome, message)
		}
		s := getSnapshot(t, f.p)
		if applied != 1 || s.Revision != 2 {
			t.Fatal("more than one change")
		}
		files, err := filepath.Glob(filepath.Join(f.dir, "*.key"))
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != len(s.Keys) || (!raceRevoke && len(files) != 2) {
			t.Fatal("extra key file")
		}
	}
}
func TestSEC01_CommandProcessClosedPipe(t *testing.T) {
	binary := buildCommand(t)
	for _, op := range []string{"rotate", "revoke"} {
		f := setup(t, signing.Lease)
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Close(); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, f.args(op)...)
		cmd.Stdout = writer
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err = cmd.Run()
		if closeErr := writer.Close(); closeErr != nil {
			t.Fatal(closeErr)
		}
		code := processCode(t, err)
		outcome := "applied"
		if op == "revoke" {
			outcome = "revoked"
		}
		assertOutput(t, code, "", stderr.String(), exitOutput, "", "fvpn-vault-keys: result output failed: "+outcome+"; run fvpn-vault inspect; do not repeat blindly\n")
		if getSnapshot(t, f.p).Revision != 2 {
			t.Fatal("lost or duplicated mutation")
		}
		before := manifest(t, f.dir)
		repeat := exec.CommandContext(ctx, binary, f.args(op)...)
		var stdout bytes.Buffer
		stderr.Reset()
		repeat.Stdout = &stdout
		repeat.Stderr = &stderr
		code = processCode(t, repeat.Run())
		want, result := exitRefused, "precondition_refused"
		if op == "revoke" {
			want, result = 0, "already_revoked"
		}
		assertOutput(t, code, stdout.String(), stderr.String(), want, result, "")
		unchanged(t, f.dir, before)
	}
}
func TestSEC01_CommandProcessFIFO(t *testing.T) {
	binary := buildCommand(t)
	for _, op := range []string{"rotate", "revoke"} {
		t.Run(op, func(t *testing.T) {
			f := setup(t, signing.Lease)
			path := filepath.Join(f.dir, "lock")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := syscall.Mkfifo(path, 0600); err != nil {
				t.Fatal(err)
			}
			before := manifest(t, f.dir)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, f.args(op)...)
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			if ctx.Err() != nil {
				t.Fatal("FIFO command exceeded 2s process deadline")
			}
			code := processCode(t, err)
			assertOutput(t, code, stdout.String(), stderr.String(), exitUnavailable, "", "fvpn-vault-keys: vault unavailable\n")
			unchanged(t, f.dir, before)
		})
	}
}
