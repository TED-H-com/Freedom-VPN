package agent

import (
	"context"

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

// SystemPlane is the production data plane.
type SystemPlane struct{}

// Apply installs the model. On a verification failure enforce itself rolls
// back to the baseline (§8.2, §10.3).
func (SystemPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	return enforce.Apply(ctx, enforce.System{}, m, prev)
}

// Baseline installs the fail-closed boot ruleset. The probe exception is not
// included. The watchdog calls enforce.ApplyBaseline directly.
func (SystemPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	return enforce.ApplyBaseline(ctx, enforce.System{}, dp)
}

// BaselineModel installs the fail-closed ruleset, including a probe when the
// model has one. VerifyBaseline runs on this same model.
func (SystemPlane) BaselineModel(ctx context.Context, m enforce.Model) error {
	return enforce.ApplyBaselineModel(ctx, enforce.System{}, m)
}
