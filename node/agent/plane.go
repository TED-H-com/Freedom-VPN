package agent

import (
	"context"
	"os/exec"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// Plane applies a verified model. Tests substitute a fake. Production uses
// SystemPlane, which talks to nft and tc inside the node network namespace.
type Plane interface {
	Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error)
	// Baseline is the watchdog and steady fail-closed path. It has no probe.
	Baseline(ctx context.Context, dp enforce.Datapath) error
	// BaselineModel is ApplyBaselineModel. A non-nil Probe is the §13.9
	// exception for one health check. The same call with a nil Probe removes it.
	BaselineModel(ctx context.Context, m enforce.Model) error
}

// Executor is the agent's enforcement boundary, including executable preflight.
type Executor interface {
	enforce.Executor
	LookPath(string) (string, error)
}

type systemExecutor struct{ enforce.System }

func (systemExecutor) LookPath(name string) (string, error) { return exec.LookPath(name) }

// SystemPlane is the production data plane. A nil Executor uses the system tools.
type SystemPlane struct{ Executor Executor }

func (p SystemPlane) executor() Executor {
	if p.Executor != nil {
		return p.Executor
	}
	return systemExecutor{}
}

// MissingExecutables checks all tools before the first enforcement attempt.
// Return names only: PATH and lookup errors can contain local configuration.
func (p SystemPlane) MissingExecutables() []string {
	var missing []string
	for _, name := range []string{"nft", "tc", "ip", "conntrack"} {
		if _, err := p.executor().LookPath(name); err != nil {
			missing = append(missing, name)
		}
	}
	return missing
}

// Apply installs the model. On a verification failure enforce itself rolls
// back to the baseline (§8.2, §10.3).
func (p SystemPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	return enforce.Apply(ctx, p.executor(), m, prev)
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
