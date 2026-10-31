package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
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
	Kind  string     `json:"kind"`
	Start traceStamp `json:"start"`
	End   traceStamp `json:"end"`
	OK    bool       `json:"ok"`
}
type traceExecutor struct {
	enforce.System
	Batches []traceBatch
	// TEST ONLY: fail one real verifier read after a completed release. The
	// candidate, guard and rollback still execute in the real kernel.
	FailVerificationRead bool
	FaultInjected        bool
}

func (e *traceExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if e.FailVerificationRead && !e.FaultInjected && name == "nft" && len(e.Batches) > 0 && e.Batches[len(e.Batches)-1].Kind == "guard_release" {
		e.FaultInjected = true
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
	e.Batches = append(e.Batches, traceBatch{Kind: kind, Start: start, End: traceNow(), OK: err == nil})
	return err
}
func tracedApply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue, dst io.Writer) (enforce.ApplyResult, error) {
	return tracedApplyExecutor(ctx, &traceExecutor{}, m, prev, dst)
}

func tracedApplyExecutor(ctx context.Context, ex *traceExecutor, m enforce.Model, prev map[string]enforce.CounterValue, dst io.Writer) (enforce.ApplyResult, error) {
	start := traceNow()
	res, err := enforce.Apply(ctx, ex, m, prev)
	end := traceNow()
	var reason enforce.Reason
	var check *enforce.CheckError
	if errors.As(err, &check) {
		reason = check.Reason
	}
	row := struct {
		Event         string                   `json:"event"`
		Generation    uint64                   `json:"generation,string"`
		Start         traceStamp               `json:"start"`
		End           traceStamp               `json:"end"`
		Verified      bool                     `json:"verified"`
		Error         enforce.Reason           `json:"error,omitempty"`
		Deadline      enforce.DeadlineEvidence `json:"deadline"`
		Batches       []traceBatch             `json:"batches"`
		Timing        *enforce.TimingEvidence  `json:"timing"`
		FaultInjected bool                     `json:"test_read_fault_injected,omitempty"`
	}{"apply", m.Generation, start, end, res.Verified, reason, res.Deadline, ex.Batches, res.Evidence.TimingEvidence(), ex.FaultInjected}
	if logErr := json.NewEncoder(dst).Encode(row); err == nil && logErr != nil {
		return res, logErr
	}
	return res, err
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
