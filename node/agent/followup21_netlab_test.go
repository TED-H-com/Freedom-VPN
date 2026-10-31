//go:build netlab

package agent

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

type measuredSystemPlane struct {
	SystemPlane
	reads []time.Duration
}

func (p *measuredSystemPlane) VerifyServing(ctx context.Context, model enforce.Model, assignments map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	start := time.Now()
	ev, err := p.SystemPlane.VerifyServing(ctx, model, assignments)
	p.reads = append(p.reads, time.Since(start))
	return ev, err
}

// Invoked only by the foreground container launcher in tests/netlab/agent.
// The engine and TLS fake control run in this process; nft/tc are real. No
// traffic generator or external endpoint is used for the latency measurement.
func TestD085_LiveReadNetlab(t *testing.T) {
	if os.Getenv("FVPN_AGENT_LIVE_NETLAB") != "1" {
		t.Skip("requires tests/netlab/agent/run-live.sh foreground privileged container")
	}
	if _, err := os.Stat("/.dockerenv"); err != nil {
		t.Fatal("refusing to change networking outside the lab container")
	}
	run := func(name string, args ...string) {
		t.Helper()
		if out, err := exec.CommandContext(t.Context(), name, args...).CombinedOutput(); err != nil {
			t.Fatalf("lab %s: %v %s", name, err, out)
		}
	}
	for _, pair := range [][2]string{{"fv-fh", "fv-fa"}, {"fv-ih", "fv-ia"}} {
		run("ip", "link", "add", pair[0], "type", "veth", "peer", "name", pair[1])
		for _, iface := range pair {
			run("ip", "link", "set", iface, "up")
		}
	}
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	p := &measuredSystemPlane{}
	r.eng.plane = p
	r.eng.cfg.HeartbeatSeconds, r.eng.cfg.PollSeconds = 1, 1
	r.eng.withdraw(t.Context())
	stepOK(t, r)
	if !r.eng.serving {
		t.Fatalf("real Apply failed: %+v %s", r.eng.decision.Errors, r.log.String())
	}
	p.reads = nil
	const steps = 30
	started := time.Now()
	for range steps {
		stepOK(t, r)
		if !mustBeat(t, r).Serving {
			t.Fatalf("live read failed: %+v", r.eng.decision.Errors)
		}
	}
	stepMean := time.Since(started) / steps
	if len(p.reads) != 3*steps {
		t.Fatalf("live reads=%d want=%d", len(p.reads), 3*steps)
	}
	var total time.Duration
	for _, d := range p.reads {
		total += d
	}
	slices.Sort(p.reads)
	t.Logf("real nft/tc: steps=%d reads=%d mean_read=%s p50=%s p95=%s max=%s added_live_reads_per_step=%s total_step_mean=%s timeout=%s",
		steps, len(p.reads), total/time.Duration(len(p.reads)), p.reads[len(p.reads)/2], p.reads[len(p.reads)*95/100], p.reads[len(p.reads)-1], total/steps, stepMean, r.eng.servingReadTimeout())

	run("tc", "qdisc", "add", "dev", "fv-fh", "clsact")
	run("tc", "filter", "add", "dev", "fv-fh", "ingress", "pref", "1", "matchall", "action", "mirred", "egress", "redirect", "dev", "fv-ih")
	stepOK(t, r)
	assertLiveWithdrawn(t, r, 2, "TC_FOREIGN_FILTER")
	// Baseline intentionally leaves this foreign filter. D-090 will supply
	// quarantine; remove the test injection explicitly before recovery.
	run("tc", "filter", "del", "dev", "fv-fh", "ingress", "pref", "1")
	run("tc", "qdisc", "del", "dev", "fv-fh", "clsact")
	stepOK(t, r)
	if !mustBeat(t, r).Serving || r.eng.applyError != nil {
		t.Fatalf("real full Apply recovery failed: %+v", r.eng.decision.Errors)
	}
	t.Log("real engine: serving -> injected ingress redirect -> next Step non-serving with TC_FOREIGN_FILTER/control report accepted -> remove injection -> clean full Apply serving")
}
