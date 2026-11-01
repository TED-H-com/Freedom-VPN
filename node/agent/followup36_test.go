package agent

import (
	"context"
	"encoding/json"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/node/internal/nodelock"
)

func TestAT13_F36_RestartSetChanges(t *testing.T) {
	for _, field := range []string{"denied", "extra", "host"} {
		for _, change := range []string{"grow", "shrink", "replace", "unchanged", "reorder"} {
			t.Run(field+"/"+change, func(t *testing.T) {
				r := newRig(t)
				alignControlClock(r)
				r.eng.cfg.TestMode = true
				r.eng.cfg.TestExemptV4 = []string{"203.0.113.0/24", "198.51.100.0/24"}
				r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32", "203.0.113.2/32"}
				r.eng.cfg.HostV4 = []string{"198.51.100.1/32", "198.51.100.2/32"}
				serveBoundGeneration(t, r)
				r.manualUsage(t)
				if err := r.eng.Close(); err != nil {
					t.Fatal(err)
				}
				cfg := r.eng.cfg
				var set *[]string
				switch field {
				case "denied":
					set = &cfg.TestExemptV4
				case "extra":
					set = &cfg.ExtraDeniedV4
				case "host":
					set = &cfg.HostV4
				}
				*set = slices.Clone(*set)
				switch change {
				case "grow":
					*set = append(*set, "192.0.2.0/24")
				case "shrink":
					*set = (*set)[:1]
				case "replace":
					(*set)[1] = "192.0.2.0/24"
				case "reorder":
					slices.Reverse(*set)
					*set = append(*set, (*set)[0])
				}
				// DeniedV4 is built as spec minus TestExemptV4, so grow/shrink
				// of that config slice exercise the opposite model-set change.
				next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: r.plane, MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = next.Close() })
				r.eng = next
				baselines, applies := r.plane.baselines, r.plane.applies
				if err := next.Boot(t.Context()); err != nil {
					t.Fatal(err)
				}
				adopt := change == "unchanged" || change == "reorder"
				if next.serving != adopt || mustBeat(t, r).Serving != adopt || r.plane.applies != applies {
					t.Fatalf("adoption=%v want=%v applies=%d/%d", next.serving, adopt, r.plane.applies, applies)
				}
				if adopt {
					if r.plane.baselines != baselines {
						t.Fatal("unchanged sets withdrew service")
					}
					return
				}
				if r.plane.baselines != baselines+1 || !next.baselineOK {
					t.Fatal("changed sets missed baseline")
				}
				// Block the first Step's Apply after Boot has refused adoption.
				lock, err := nodelock.Acquire(t.Context(), cfg.LockPath, 1, 0)
				if err != nil {
					t.Fatal(err)
				}
				defer lock.Release()
				stepOK(t, r)
				if next.serving || mustBeat(t, r).Serving || r.plane.applies != applies || len(r.plane.last.Bindings) != 0 {
					t.Fatal("busy first Apply escaped baseline")
				}
				lock.Release()
				stepOK(t, r)
				if !next.serving || r.plane.applies != applies+1 {
					t.Fatal("released first Apply did not serve")
				}
			})
		}
	}
}

func TestAT13_F36_ChangedSetsControlTimeout(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	serveBoundGeneration(t, r)
	r.manualUsage(t)
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.9/32"}
	restartEngine(t, r, r.plane)
	if r.eng.serving || mustBeat(t, r).Serving {
		t.Fatal("changed sets adopted")
	}
	applies := r.plane.applies
	r.srv.HoldLeases()
	defer r.srv.ReleaseLeases()
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	_ = r.eng.Step(ctx)
	if r.eng.serving || mustBeat(t, r).Serving || r.plane.applies != applies || len(r.plane.last.Bindings) != 0 {
		t.Fatal("control timeout served retained sets")
	}
}

// Persisted model contents, not a rebuilt candidate, are the adoption input.
func TestAT13_F36_RetainedModelContainsSets(t *testing.T) {
	r := newRig(t)
	alignControlClock(r)
	r.eng.cfg.TestMode = true
	r.eng.cfg.TestExemptV4 = []string{"203.0.113.0/24"}
	r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
	r.eng.cfg.HostV4 = []string{"198.51.100.1/32"}
	serveBoundGeneration(t, r)
	raw, ok, err := r.eng.store.Meta(t.Context(), "retained_applied")
	var got retainedApplied
	if err != nil || !ok || json.Unmarshal([]byte(raw), &got) != nil {
		t.Fatal("missing retained model", err)
	}
	if !slices.Equal(got.Model.DeniedV4, enforce.Without(enforce.SpecDeniedV4(), netip.MustParsePrefix("203.0.113.0/24"))) ||
		!slices.Equal(got.Model.ExtraDeniedV4, []netip.Prefix{netip.MustParsePrefix("203.0.113.1/32")}) ||
		!slices.Equal(got.Model.HostV4, []netip.Prefix{netip.MustParsePrefix("198.51.100.1/32")}) {
		t.Fatal("retained model lost configured sets")
	}
}
