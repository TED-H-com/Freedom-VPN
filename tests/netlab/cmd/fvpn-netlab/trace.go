package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// TEST ONLY timing decorator. All I/O is deferred until Apply returns, so
// logging cannot insert an extra disk write between guard install and release.
// Each kernel transaction lies somewhere inside its command bracket, not at
// either endpoint. The installed-by -> release-start interval is definitely
// guarded; install-start -> release-end is only a conservative outer bound.
type traceStamp struct {
	WallNS int64 `json:"wall_ns,string"`
	MonoNS int64 `json:"mono_ns,string"`
}

func traceNow() traceStamp {
	mono, err := enforce.MonotonicNow()
	if err != nil {
		panic(err)
	} // Linux is a prerequisite for this lab command.
	return traceStamp{WallNS: time.Now().UnixNano(), MonoNS: int64(mono)}
}

type traceBatch struct {
	Kind   string     `json:"kind"`
	Start  traceStamp `json:"start"`
	End    traceStamp `json:"end"`
	OK     bool       `json:"ok"`
	Script string     `json:"test_only_script,omitempty"`
	SHA256 string     `json:"test_only_sha256,omitempty"`
}
type traceExecutor struct {
	enforce.System
	Batches    []traceBatch
	ObserveTCP bool
	// TEST ONLY: fail one real verifier read after a completed release. The
	// candidate, guard and rollback still execute in the real kernel.
	FailVerificationRead bool
	FaultInjected        bool
	FaultObserved        *traceStamp
}

func (e *traceExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if e.FailVerificationRead && !e.FaultInjected && name == "nft" && len(e.Batches) > 0 && e.Batches[len(e.Batches)-1].Kind == "guard_release" {
		e.FaultInjected = true
		observed := traceNow()
		e.FaultObserved = &observed
		return nil, errors.New("TEST ONLY injected post-release verification read failure")
	}
	return e.System.Output(ctx, name, args...)
}

func (e *traceExecutor) NftBatch(ctx context.Context, script string) error {
	kind := "other"
	if strings.Contains(script, "add chain inet fvpn fv_deadline_guard ") {
		kind = "guard_install"
	}
	if strings.TrimSpace(script) == "delete chain inet fvpn fv_deadline_guard" {
		kind = "guard_release"
	}
	if strings.Contains(script, "fvpn_generation=0") {
		kind = "baseline"
	}
	start := traceNow()
	err := e.System.NftBatch(ctx, script)
	e.Batches = append(e.Batches, traceBatch{Kind: kind, Start: start, End: traceNow(), OK: err == nil, Script: script})
	return err
}
func tracedApply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue, dst io.Writer) (enforce.ApplyResult, error) {
	return tracedApplyExecutor(ctx, &traceExecutor{}, m, prev, dst)
}

func tracedApplyExecutor(ctx context.Context, ex *traceExecutor, m enforce.Model, prev map[string]enforce.CounterValue, dst io.Writer) (enforce.ApplyResult, error) {
	input, encodeErr := json.Marshal(m)
	if encodeErr != nil {
		return enforce.ApplyResult{}, encodeErr
	}
	var capture enforce.ApplyCaptureForTest
	ctx = enforce.WithApplyCaptureForTest(enforce.WithCommandLifecycleForTest(ctx), &capture)
	var tcpSnapshots []tcpSnapshot
	if ex.ObserveTCP {
		tcpSnapshots = append(tcpSnapshots, readTCPState("before_apply"))
	}
	start := traceNow()
	res, err := enforce.Apply(ctx, ex, m, prev)
	end := traceNow()
	if ex.ObserveTCP {
		tcpSnapshots = append(tcpSnapshots, readTCPState("after_apply"))
	}
	// Hash/encode only after the operation, never inside an installed guard.
	for i := range ex.Batches {
		ex.Batches[i].SHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(ex.Batches[i].Script)))
	}
	rendered, encodeErr := json.Marshal(capture)
	if encodeErr != nil {
		return res, encodeErr
	}
	var reason enforce.Reason
	var check *enforce.CheckError
	if errors.As(err, &check) {
		reason = check.Reason
	}
	row := struct {
		Event             string                   `json:"event"`
		Generation        uint64                   `json:"generation,string"`
		Start             traceStamp               `json:"start"`
		End               traceStamp               `json:"end"`
		Verified          bool                     `json:"verified"`
		Error             enforce.Reason           `json:"error,omitempty"`
		Deadline          enforce.DeadlineEvidence `json:"deadline"`
		Batches           []traceBatch             `json:"batches"`
		Timing            *enforce.TimingEvidence  `json:"timing"`
		FaultInjected     bool                     `json:"test_read_fault_injected,omitempty"`
		TestInput         json.RawMessage          `json:"test_only_input_model"`
		TestInputSHA256   string                   `json:"test_only_input_sha256"`
		TestRender        json.RawMessage          `json:"test_only_render"`
		TestFaultObserved *traceStamp              `json:"test_only_fault_observed,omitempty"`
		TestRenderSHA256  string                   `json:"test_only_render_sha256"`
		TCPSnapshots      []tcpSnapshot            `json:"test_only_tcp_snapshots,omitempty"`
	}{"apply", m.Generation, start, end, res.Verified, reason, res.Deadline, ex.Batches, res.Evidence.TimingEvidence(), ex.FaultInjected, input, fmt.Sprintf("%x", sha256.Sum256(input)), rendered, ex.FaultObserved, fmt.Sprintf("%x", sha256.Sum256(rendered)), tcpSnapshots}
	if logErr := json.NewEncoder(dst).Encode(row); err == nil && logErr != nil {
		return res, logErr
	}
	return res, err
}

// TEST ONLY read-only snapshots enclose Apply; no extra command executes
// between guard installation and release. Empty stdout means absence only
// when OK is true. Errors remain explicit diagnostic evidence.
type tcpRead struct {
	Start  traceStamp `json:"start"`
	End    traceStamp `json:"end"`
	OK     bool       `json:"ok"`
	Output string     `json:"output"`
	Error  string     `json:"error,omitempty"`
}
type tcpSnapshot struct {
	Phase     string  `json:"phase"`
	Conntrack tcpRead `json:"conntrack"`
	Socket    tcpRead `json:"socket"`
}

func readTCPCommand(name string, args ...string) tcpRead {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	r := tcpRead{Start: traceNow()}
	out, err := exec.CommandContext(ctx, name, args...).Output()
	r.End, r.OK, r.Output = traceNow(), err == nil, string(out)
	if err != nil {
		r.Error = err.Error()
	}
	return r
}
func readTCPState(phase string) tcpSnapshot {
	return tcpSnapshot{Phase: phase,
		Conntrack: readTCPCommand("conntrack", "-L", "-f", "ipv4", "-p", "tcp", "--orig-src", "100.64.1.1", "--orig-dst", "203.0.113.10", "--dport", "9001", "-o", "extended,id"),
		Socket:    readTCPCommand("ip", "netns", "exec", "fv-a1", "ss", "-H", "-O", "-n", "-t", "-i", "-o", "dst", "203.0.113.10:9001"),
	}
}

type tracePlane struct {
	agent.SystemPlane
	log io.Writer
}

func (p tracePlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	return tracedApply(ctx, m, prev, p.log)
}

// Runs the real agent Engine and SystemPlane; only Apply's executor is decorated.
// Enrollment still uses the ordinary fvpn-agent binary. No stub serving result,
// lease override, bypassed verifier, or modified production agent is involved.
func agentTraceCmd(args []string) error {
	fs := flag.NewFlagSet("agent-trace", flag.ContinueOnError)
	path := fs.String("config", "", "agent configuration")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := agent.LoadConfig(*path)
	if err != nil {
		return err
	}
	log := slog.New(agent.RedactHandler{Next: slog.NewTextHandler(os.Stderr, nil)})
	eng, err := agent.Start(agent.Options{Config: cfg, Log: log, Plane: tracePlane{log: os.Stderr}})
	if err != nil {
		return err
	}
	defer eng.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	err = eng.Run(ctx)
	if ctx.Err() != nil {
		return nil
	}
	return err
}
