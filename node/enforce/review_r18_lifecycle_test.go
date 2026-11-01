package enforce

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestReviewR18_DisabledLifecycleDoesNotAllocate(t *testing.T) {
	ctx := context.Background()
	allocs := testing.AllocsPerRun(200, func() {
		next := beginCommandMeasurement(ctx, nil)
		next = commandPhase(next, "install")
		captureRenderForTest(next, Model{})
		if next != ctx {
			panic("disabled lifecycle replaced the context")
		}
	})
	if allocs != 0 {
		t.Fatalf("TODO(review): with the lifecycle and render-capture keys absent, beginCommandMeasurement, commandPhase and captureRenderForTest allocated %v times per call; D-138 said each site does one context lookup and nothing else", allocs)
	}
}

func BenchmarkReviewR18_DisabledLifecycle(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		next := beginCommandMeasurement(ctx, nil)
		next = commandPhase(next, "install")
		captureRenderForTest(next, Model{})
		if next != ctx {
			b.Fatal("disabled lifecycle replaced the context")
		}
	}
}

func TestReviewR18_DisabledRunCommandMatchesRun(t *testing.T) {
	base := testing.AllocsPerRun(20, func() {
		if err := exec.Command("true").Run(); err != nil {
			panic(err)
		}
	})
	got := testing.AllocsPerRun(20, func() {
		if err := runCommand(context.Background(), exec.Command("true"), "nft"); err != nil {
			panic(err)
		}
	})
	if got-base > 1 {
		t.Fatalf("TODO(review): disabled runCommand averaged %v allocs against %v for exec.Cmd.Run; the disabled branch is one context lookup then Run", got, base)
	}
	before := readProcIO(t)
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		next := beginCommandMeasurement(ctx, nil)
		next = commandPhase(next, "install")
		captureRenderForTest(next, Model{})
	}
	after := readProcIO(t)
	quiet := readProcIO(t)
	// The counter read itself is the only expected syscr/syscw growth.
	if after.syscr-before.syscr > quiet.syscr-after.syscr+2 || after.syscw-before.syscw > quiet.syscw-after.syscw+2 {
		t.Fatalf("TODO(review): disabled call sites moved /proc/self/io by syscr=%d syscw=%d (control read moved syscr=%d syscw=%d)",
			after.syscr-before.syscr, after.syscw-before.syscw, quiet.syscr-after.syscr, quiet.syscw-after.syscw)
	}
}

func TestReviewR18_RenderCaptureStaysInMemory(t *testing.T) {
	var capture ApplyCaptureForTest
	captureRenderForTest(context.Background(), Model{})
	if capture.Rendered {
		t.Fatal("render capture ran without its context key")
	}
	ctx := WithApplyCaptureForTest(context.Background(), &capture)
	captureRenderForTest(ctx, Model{Bindings: []Binding{{PeerID: "peer-secret"}}})
	if !capture.Rendered || len(capture.Model.Bindings) != 1 {
		t.Fatal("keyed capture did not retain the render input")
	}
}

func TestReviewR18_EnabledSampleIgnoresInstallDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), DeadlineInstallBound)
	defer cancel()
	time.Sleep(DeadlineInstallBound + 20*time.Millisecond)
	if ctx.Err() == nil {
		t.Fatal("install context was still live")
	}
	var records []CommandLifecycle
	ctx = commandPhase(beginCommandMeasurement(WithCommandLifecycleForTest(ctx), &records), "install")
	before := readProcIO(t)
	start := time.Now()
	err := runCommand(ctx, exec.Command("true"), "nft")
	elapsed := time.Since(start)
	after := readProcIO(t)
	if err != nil || len(records) != 1 {
		t.Fatalf("enabled observation did not complete: %v records=%d", err, len(records))
	}
	rec := records[0]
	if rec.ContextBefore != "deadline_exceeded" || !rec.Before.Bracket.Before.Available || after.syscr <= before.syscr {
		t.Fatalf("sample did not run against a finished install context: before=%s syscr=%d available=%v",
			rec.ContextBefore, after.syscr-before.syscr, rec.Before.Bracket.Before.Available)
	}
	t.Fatalf("TODO(review): runCommand sampled command resources after the %s install context was already %s (elapsed %s, /proc reads syscr +%d). commandResources ignores ctx and runs before Start, so an enabled Apply can stay inside NftBatch past DeadlineInstallBound",
		DeadlineInstallBound, rec.ContextBefore, elapsed, after.syscr-before.syscr)
}

func TestReviewR18_EnabledCancelReapsChild(t *testing.T) {
	t.Setenv("FVPN_TEST_COMMAND_HELPER", "wait")
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(context.Background())
	var records []CommandLifecycle
	ctx = commandPhase(beginCommandMeasurement(WithCommandLifecycleForTest(ctx), &records), "install")
	cmd := helperCommand(ctx)
	done := make(chan error, 1)
	go func() { done <- runCommand(ctx, cmd, "nft") }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err == nil || len(records) != 1 || !records[0].SignalAvailable || records[0].Signal != 9 {
			t.Fatalf("cancel did not reap with signal 9: %v %+v", err, records)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("enabled runCommand left the child running")
	}
	deadline := time.Now().Add(time.Second)
	for runtime.NumGoroutine() > before+1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if left := runtime.NumGoroutine(); left > before+1 {
		t.Fatalf("goroutines %d, before %d", left, before)
	}
}

type procIO struct{ syscr, syscw uint64 }

func readProcIO(t *testing.T) procIO {
	t.Helper()
	b, err := os.ReadFile("/proc/self/io")
	if err != nil {
		t.Fatal(err)
	}
	var out procIO
	for _, line := range strings.Split(string(b), "\n") {
		key, val, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSpace(val), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		switch key {
		case "syscr":
			out.syscr = n
		case "syscw":
			out.syscw = n
		}
	}
	return out
}
