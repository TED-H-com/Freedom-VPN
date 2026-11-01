package enforce

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

// LifecycleStamp is an observation, never a kernel event timestamp. Unavailable
// values are explicit; a zero must not be interpreted as a successful sample.
type LifecycleStamp struct {
	NS        time.Duration `json:"ns,string"`
	Available bool          `json:"available"`
}

type LifecycleBracket struct {
	Before LifecycleStamp `json:"before"`
	After  LifecycleStamp `json:"after"`
}

// CommandLifecycle is TEST ONLY, local diagnostic data, outside D-106. It never
// contains argv, stdin, output, paths, selectors, or the raw execution error.
// StartAttempt..ChildStartedBy brackets exec.Start, not the child's first run.
// CancelObserved is entry to exec's cancellation callback, not deadline firing.
// WaitReturn is sampled immediately after Wait (absent when Start fails).
type CommandLifecycle struct {
	Kind                string           `json:"kind"`
	Phase               string           `json:"phase"`
	ContextDeadline     LifecycleBracket `json:"context_deadline"`
	StartAttempt        LifecycleStamp   `json:"start_attempt"`
	StartReturn         LifecycleStamp   `json:"start_return"`
	ChildStartedBy      LifecycleStamp   `json:"child_started_by"`
	ChildStartExact     LifecycleStamp   `json:"child_start_exact"`
	CancelObserved      LifecycleStamp   `json:"cancel_observed"`
	WaitReturn          LifecycleStamp   `json:"wait_return"`
	ContextBefore       string           `json:"context_before"`
	ContextAfter        string           `json:"context_after"`
	ContextAtCancel     string           `json:"context_at_cancel"`
	Started             bool             `json:"started"`
	OK                  bool             `json:"ok"`
	ExitAvailable       bool             `json:"exit_available"`
	ExitCode            int              `json:"exit_code"`
	SignalAvailable     bool             `json:"signal_available"`
	Signal              int              `json:"signal"`
	CPUAvailable        bool             `json:"cpu_available"`
	UserCPU             time.Duration    `json:"user_cpu_ns,string"`
	SystemCPU           time.Duration    `json:"system_cpu_ns,string"`
	SwitchesAvailable   bool             `json:"switches_available"`
	VoluntarySwitches   int64            `json:"voluntary_switches,string"`
	InvoluntarySwitches int64            `json:"involuntary_switches,string"`
	Before              CommandResources `json:"before"`
	After               CommandResources `json:"after"`
}

// ResourceText is a bounded, fixed-name kernel numeric file. Missing/unsupported
// files are unavailable, not an empty successful counter or unlimited quota.
type ResourceText struct {
	Available bool   `json:"available"`
	Value     string `json:"value"`
}

type CgroupCPU struct {
	Depth    int          `json:"depth"` // zero is the current cgroup; no cgroup names exported
	Max      ResourceText `json:"cpu_max"`
	Weight   ResourceText `json:"cpu_weight"`
	Stat     ResourceText `json:"cpu_stat"`
	Pressure ResourceText `json:"cpu_pressure"`
}

type CommandResources struct {
	Bracket                  LifecycleBracket `json:"bracket"`
	CgroupV2Available        bool             `json:"cgroup_v2_available"`
	VisibleAncestorsComplete bool             `json:"visible_ancestors_complete"`
	// A mount/cgroup namespace cannot establish visibility of all host ancestors.
	HostAncestorsAvailable bool        `json:"host_ancestors_available"`
	Cgroups                []CgroupCPU `json:"cgroups"`
}

type commandMeasurementKey struct{}
type commandScopeKey struct{}
type commandScope struct {
	phase   string
	records *[]CommandLifecycle
}

// WithCommandLifecycleForTest opts Apply into local command evidence. It does
// not enable logging, change deadlines, or authorize use with production data.
// The agent owner can pass this context only when its measurement gate is on;
// its existing TimingEvidence producer then retains the additive records.
func WithCommandLifecycleForTest(ctx context.Context) context.Context {
	return context.WithValue(ctx, commandMeasurementKey{}, true)
}

func beginCommandMeasurement(ctx context.Context, records *[]CommandLifecycle) context.Context {
	if enabled, _ := ctx.Value(commandMeasurementKey{}).(bool); !enabled {
		return ctx
	}
	return context.WithValue(ctx, commandScopeKey{}, commandScope{"prepare", records})
}

func commandPhase(ctx context.Context, phase string) context.Context {
	scope, ok := ctx.Value(commandScopeKey{}).(commandScope)
	if !ok {
		return ctx
	}
	scope.phase = phase
	return context.WithValue(ctx, commandScopeKey{}, scope)
}

func lifecycleNow() LifecycleStamp {
	n, err := MonotonicNow()
	return LifecycleStamp{n, err == nil}
}

func contextState(ctx context.Context) string {
	switch err := ctx.Err(); {
	case err == nil:
		return "none"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	default:
		return "other"
	}
}

func contextDeadline(ctx context.Context) LifecycleBracket {
	deadline, ok := ctx.Deadline()
	if !ok {
		return LifecycleBracket{}
	}
	// time.Until uses Go's monotonic component when the context has one. The
	// CLOCK_MONOTONIC samples enclose that conversion; no exact epoch is invented.
	before := lifecycleNow()
	remaining := time.Until(deadline)
	after := lifecycleNow()
	if !before.Available || !after.Available {
		return LifecycleBracket{}
	}
	before.NS += remaining
	after.NS += remaining
	return LifecycleBracket{before, after}
}

func runCommand(ctx context.Context, cmd *exec.Cmd, name string) error {
	scope, enabled := ctx.Value(commandScopeKey{}).(commandScope)
	if !enabled {
		return cmd.Run()
	} // no samples, proc reads, allocations or goroutines
	r := CommandLifecycle{Kind: "other", Phase: scope.phase, ContextAtCancel: "unavailable"}
	if name == "nft" || name == "tc" || name == "conntrack" || name == "ip" {
		r.Kind = name
	}
	r.ContextDeadline = contextDeadline(ctx)
	r.Before = commandResources()
	r.ContextBefore = contextState(ctx)
	cancel := cmd.Cancel
	if cancel != nil {
		cmd.Cancel = func() error {
			r.CancelObserved = lifecycleNow()
			r.ContextAtCancel = contextState(ctx)
			return cancel() // preserve CommandContext's original kill and error semantics
		}
	}
	r.StartAttempt = lifecycleNow()
	err := cmd.Start()
	r.StartReturn = lifecycleNow()
	if err == nil {
		r.Started = true
		r.ChildStartedBy = r.StartReturn
		err = cmd.Wait() // Wait joins exec's cancellation watcher before we read r
		r.WaitReturn = lifecycleNow()
		if ps := cmd.ProcessState; ps != nil {
			r.ExitCode, r.ExitAvailable = ps.ExitCode(), true
			r.UserCPU, r.SystemCPU, r.CPUAvailable = ps.UserTime(), ps.SystemTime(), true
			commandProcessState(ps, &r)
		}
	}
	r.ContextAfter, r.OK = contextState(ctx), err == nil
	r.After = commandResources()
	*scope.records = append(*scope.records, r)
	return err
}
