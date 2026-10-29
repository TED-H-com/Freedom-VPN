package agent

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// Plane applies a verified model. Tests substitute a fake. Production uses
// SystemPlane, which talks to nft and tc inside the node network namespace.
type Plane interface {
	Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error)
	Baseline(ctx context.Context, dp enforce.Datapath) error
}

// SystemPlane is the production data plane.
type SystemPlane struct{}

// Apply installs the model. On a verification failure enforce itself rolls
// back to the baseline (§8.2, §10.3).
func (SystemPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	return enforce.Apply(ctx, enforce.System{}, m, prev)
}

// Baseline installs the fail-closed boot ruleset.
func (SystemPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	return enforce.ApplyBaseline(ctx, enforce.System{}, dp)
}
