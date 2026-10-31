package agent

import (
	"context"
	"errors"
	"net/netip"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

// Plane applies a verified model. Tests substitute a fake. Production uses
// SystemPlane, which talks to nft and tc inside the node network namespace.
type Plane interface {
	Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error)
	// VerifyServing reads live state against the last applied model, not a
	// newly desired model. Implementations must honor ctx's read deadline.
	VerifyServing(ctx context.Context, m enforce.Model, assignments map[uuid.UUID]uint16) (enforce.ServingEvidence, error)
	Quarantine(ctx context.Context, m enforce.Model) (enforce.QuarantineEvidence, error)
	MemberPath(ctx context.Context, m enforce.Model, expected *MemberPath) (MemberPath, error)
	// Baseline is the watchdog and steady fail-closed path. It has no probe.
	Baseline(ctx context.Context, dp enforce.Datapath) error
	// BaselineModel is ApplyBaselineModel. A non-nil Probe is the §13.9
	// exception for one health check. The same call with a nil Probe removes it.
	BaselineModel(ctx context.Context, m enforce.Model) error
}

// SystemPlane is the production data plane. A nil Executor uses the system tools.
type SystemPlane struct{ Executor enforce.Executor }

func (p SystemPlane) executor() enforce.Executor {
	if p.Executor != nil {
		return p.Executor
	}
	return enforce.System{}
}

// MissingExecutables checks all tools before the first enforcement attempt.
// Return names only: PATH and lookup errors can contain local configuration.
func (p SystemPlane) MissingExecutables() []string {
	var missing []string
	for _, name := range []string{"nft", "tc", "ip"} {
		if _, err := p.executor().LookPath(name); err != nil {
			missing = append(missing, name)
		}
	}
	if err := enforce.RequireConntrack(p.executor()); err != nil {
		missing = append(missing, "conntrack")
	}
	return missing
}

// ConntrackLookPath reports whether conntrack is still installed. The agent
// calls it after the boot preflight. A later failure fails closed at once.
func (p SystemPlane) ConntrackLookPath() error {
	return enforce.RequireConntrack(p.executor())
}

// FlushConntrack deletes conntrack entries for addrs without installing a
// serving ruleset. Failed is the subset whose -D did not succeed. The
// serving Apply path persists that same subset from ConntrackDeleteError.Failed.
func (p SystemPlane) FlushConntrack(ctx context.Context, addrs []netip.Addr) ([]netip.Addr, error) {
	if err := enforce.RequireConntrack(p.executor()); err != nil {
		return append([]netip.Addr(nil), addrs...), err
	}
	var failed []netip.Addr
	for _, addr := range addrs {
		if err := flushOneConntrack(ctx, p.executor(), addr); err != nil {
			failed = append(failed, addr)
		}
	}
	return failed, nil
}

func flushOneConntrack(ctx context.Context, ex enforce.Executor, addr netip.Addr) error {
	if !addr.IsValid() {
		return errors.New("agent: revoke address is invalid")
	}
	for _, flag := range []string{"-s", "-d"} {
		_, err := ex.Output(ctx, "conntrack", "-D", flag, addr.String())
		if err == nil || strings.Contains(err.Error(), "flow entries") {
			continue
		}
		return err
	}
	return nil
}

// Apply installs the model. On a verification failure enforce itself rolls
// back to the baseline (§8.2, §10.3).
func (p SystemPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	ex := p.executor()
	if timing, ok := ctx.Value(applyReadKey{}).(*applyReadTiming); ok {
		ex = timedApplyExecutor{ex, timing}
	}
	return enforce.Apply(ctx, ex, m, prev)
}

// VerifyServing checks live nft and tc state without changing networking.
func (p SystemPlane) VerifyServing(ctx context.Context, m enforce.Model, assignments map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	return enforce.VerifyServing(ctx, p.executor(), m, assignments)
}

// Baseline installs the fail-closed boot ruleset. The probe exception is not
// included. The watchdog calls enforce.ApplyBaseline directly.
func (p SystemPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	return enforce.ApplyBaseline(ctx, p.executor(), dp)
}

// BaselineModel installs the fail-closed ruleset, including a probe when the
// model has one. VerifyBaseline runs on this same model.
func (p SystemPlane) BaselineModel(ctx context.Context, m enforce.Model) error {
	return enforce.ApplyBaselineModel(ctx, p.executor(), m)
}

// Quarantine uses exactly the executor (and namespace) used by Apply.
func (p SystemPlane) Quarantine(ctx context.Context, m enforce.Model) (enforce.QuarantineEvidence, error) {
	return enforce.Quarantine(ctx, p.executor(), m)
}
