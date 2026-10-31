//go:build netlab

package agent

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
)

func followup28LiveRestart(t *testing.T, r *rig, p *measuredSystemPlane) {
	t.Helper()
	before, applies := p.baselines, len(p.applies)
	deadline, lease, anchor := r.eng.policyExpires, r.eng.lease.LeaseId, r.eng.leaseMono
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := Start(Options{Config: r.eng.cfg, Clock: timeanchor.System{}, Plane: p,
		MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	r.eng = next
	t.Cleanup(func() { _ = next.Close() })
	if err := next.Boot(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !next.serving || p.baselines != before || len(p.applies) != applies || next.policyExpires != deadline || next.lease.LeaseId != lease || next.leaseMono != anchor {
		t.Fatalf("same-boot kernel adoption failed: baselines=%d->%d applies=%d->%d serving=%v errors=%v reasons=%v", before, p.baselines, applies, len(p.applies), next.serving, p.readErrors, next.decision.Reasons)
	}
	r.manualUsage(t)
	t.Logf("ISO-F6-03 live restart: no baseline or Apply; same lease and deadline %d", deadline)
}
