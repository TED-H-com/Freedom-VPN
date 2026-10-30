//go:build e2e

package e2e

import "context"

func (r *runner) prepareProbe(ctx context.Context, note func(string, string)) bool {
	p, err := (&Stack{lab: r.lab}).SetupUpdaterProbe(ctx, "node", EnrolledNode{r.nodeID, r.incID})
	if err != nil {
		note("probe_setup", err.Error())
		return false
	}
	r.probe = p
	r.probePeer = p.Config
	note("probe_config_scope", "dedicated non-member peer /32 and isolated origin UDP 4443; exported setup")
	return true
}
func (r *runner) startProbeCapture(ctx context.Context, note func(string, string)) func() bool {
	finish, err := r.probe.Observe(ctx)
	if err != nil {
		note("probe_observer", err.Error())
		return func() bool { return false }
	}
	return func() bool {
		out := finish()
		for _, o := range out.Observations {
			note(o.Name, o.Detail)
		}
		return out.Verified
	}
}
