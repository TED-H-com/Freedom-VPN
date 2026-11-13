package agent

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/watchdog"
	"github.com/google/uuid"
)

// The watchdog and agent share the same modeled kernel generation. A
// watchdog baseline therefore invalidates the successor's real adoption read.
type f49RestartPlane struct {
	*memPlane
	atStartup func()
}

func (p *f49RestartPlane) MissingExecutables() []string {
	if p.atStartup != nil {
		p.atStartup()
	}
	return nil
}
func (p *f49RestartPlane) BaselinePresent(context.Context) (bool, error) { return true, nil }
func (p *f49RestartPlane) VerifyServing(ctx context.Context, m enforce.Model, _ map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	if p.last.Generation != m.Generation {
		return enforce.ServingEvidence{}, &enforce.CheckError{Reason: enforce.ReasonGenerationMismatch}
	}
	return enforce.ServingEvidence{}, ctx.Err()
}

func f49Watch(r *rig, p *f49RestartPlane) *watchdog.Watch {
	c := r.eng.cfg
	return &watchdog.Watch{Log: r.eng.log, Enforce: p, BootID: r.eng.bootID, Clock: r.clock, Config: watchdog.Config{LockPath: c.LockPath, HeartbeatPath: c.HeartbeatPath, WatchdogStaleSeconds: c.WatchdogStaleSeconds, Datapath: watchdog.DatapathConfig{MemberIngress: c.Datapath.MemberIngress, Uplink: c.Datapath.Uplink, SNAT: c.Datapath.SNAT}}}
}

func TestD156_WatchdogBetweenStartupAndAdoption(t *testing.T) {
	for _, scenario := range []string{"adopt", "stall", "refuse"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				r := newRig(t)
				serveBoundGeneration(t, r)
				r.manualUsage(t)
				closeAdoptionControl(t, r)
				// Shorten nothing: D and the original heartbeat age are captured from
				// production state and remain unchanged through this process replacement.
				prior := mustBeat(t, r)
				cfg := r.eng.cfg
				boot := r.eng.bootID
				if err := r.eng.Close(); err != nil {
					t.Fatal(err)
				}
				p := &f49RestartPlane{memPlane: r.plane}
				w := f49Watch(r, p)
				before := p.baselines
				observed := false
				p.atStartup = func() {
					observed = true
					b := mustBeat(t, r)
					if b.Serving || b.BaselineOK {
						t.Error("startup claimed adoption before verification")
					}
					if err := w.Once(context.Background()); err != nil {
						t.Error(err)
					}
					if p.baselines != before {
						t.Error("watchdog withdrew retained generation before adoption")
					}
					if b.Mono != prior.Mono || b.PolicyExpires != prior.PolicyExpires {
						t.Error("startup extended receipt age or deadline")
					}
					if scenario == "stall" {
						r.clock.Advance(time.Duration(cfg.WatchdogStaleSeconds)*time.Second + time.Nanosecond)
						if err := w.Once(context.Background()); err != nil {
							t.Error(err)
						}
						if p.baselines != before+1 {
							t.Error("stalled successor not withdrawn at staleness bound")
						}
					}
				}
				if scenario == "refuse" {
					cfg.HostV4 = []string{"192.0.2.42/32"}
				}
				next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: p, Log: r.eng.log, MeterSource: staticSource{boot: boot.String()}, Prober: passProbe{}})
				if err != nil {
					t.Fatal(err)
				}
				defer next.Close()
				r.eng = next
				if err := next.Boot(context.Background()); err != nil {
					t.Fatal(err)
				}
				if !observed {
					t.Fatal("startup seam not reached")
				}
				if scenario != "adopt" {
					if next.serving || p.last.Generation != 0 || !mustBeat(t, r).BaselineOK {
						t.Fatal("refused adoption did not install baseline")
					}
					return
				}
				if !next.serving || p.baselines != before {
					t.Fatalf("successor failed to adopt: %s", r.log.String())
				}
				// Drive only the independent watchdog, keeping agent/lease renewal idle.
				// The old kernel admission deadline is checked without changing it.
				r.clock.Advance(prior.PolicyExpires - r.clock.Mono() - time.Nanosecond)
				// Refresh liveness as a healthy successor would, keeping exactly D.
				next.beat()
				if err := w.Once(context.Background()); err != nil {
					t.Fatal(err)
				}
				if p.baselines != before || next.servingDeadline() != prior.PolicyExpires {
					t.Fatal("healthy retained rules withdrawn before D")
				}
				r.clock.Advance(time.Nanosecond)
				if err := w.Once(context.Background()); err != nil {
					t.Fatal(err)
				}
				if p.last.Generation != 0 {
					t.Fatal("rules survived D")
				}
			})
		})
	}
}

func TestD156_RepeatedStartupCannotRenewPendingReceipt(t *testing.T) {
	r := newRig(t)
	serveBoundGeneration(t, r)
	prior := mustBeat(t, r)
	cfg, boot := r.eng.cfg, r.eng.bootID
	if err := r.eng.Close(); err != nil {
		t.Fatal(err)
	}
	p := &f49RestartPlane{memPlane: r.plane}
	next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: p, Log: r.eng.log, MeterSource: staticSource{boot: boot.String()}, Prober: passProbe{}})
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	r.eng = next
	if err := next.startupHeartbeat(context.Background(), prior); err != nil {
		t.Fatal(err)
	}
	pending := mustBeat(t, r)
	if pending.Reason != "boot_adopting" || pending.Mono != prior.Mono {
		t.Fatal("missing bounded pending receipt")
	}
	r.clock.Advance(time.Second)
	// Simulate another successor before the first ever completed adoption.
	if err := next.startupHeartbeat(context.Background(), pending); err != nil {
		t.Fatal(err)
	}
	b := mustBeat(t, r)
	if b.Serving || b.BaselineOK || b.PolicyExpires != 0 || b.Reason != "boot_unverified" {
		t.Fatal("a restart renewed an unverified receipt")
	}
	before := p.baselines
	if err := f49Watch(r, p).Once(context.Background()); err != nil {
		t.Fatal(err)
	}
	if p.baselines != before+1 {
		t.Fatal("watchdog trusted a chain of pending startups")
	}
}
