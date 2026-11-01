package agent

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/google/uuid"
)

// Real Apply and VerifyServing, with nft/tc commands emulated by the existing
// rendered executor. The same executor retains installed sets across restart.
type f35RestartPlane struct {
	*memPlane
	ex      renderedExecutor
	offset  time.Duration
	models  []enforce.Model
	results []enforce.ApplyResult
	errors  []error
}

func (p *f35RestartPlane) kernelModel(m enforce.Model) enforce.Model {
	m.NowMono += p.offset
	m.Leases = slices.Clone(m.Leases)
	for i := range m.Leases {
		if m.Leases[i].DeadlineMono != nil {
			deadline := *m.Leases[i].DeadlineMono + p.offset
			m.Leases[i].DeadlineMono = &deadline
		}
	}
	return m
}

func (p *f35RestartPlane) Apply(ctx context.Context, m enforce.Model, prev map[string]enforce.CounterValue) (enforce.ApplyResult, error) {
	if _, err := p.memPlane.Apply(ctx, m, prev); err != nil {
		return enforce.ApplyResult{}, err
	}
	p.models = append(p.models, m)
	p.ex.model = p.kernelModel(m)
	res, err := enforce.Apply(ctx, &p.ex, p.ex.model, prev)
	p.results = append(p.results, res)
	p.errors = append(p.errors, err)
	return res, err
}

func (p *f35RestartPlane) VerifyServing(ctx context.Context, m enforce.Model, assignments map[uuid.UUID]uint16) (enforce.ServingEvidence, error) {
	return enforce.VerifyServing(ctx, &p.ex, p.kernelModel(m), assignments)
}

func (p *f35RestartPlane) Baseline(ctx context.Context, dp enforce.Datapath) error {
	if err := p.memPlane.Baseline(ctx, dp); err != nil {
		return err
	}
	return enforce.ApplyBaseline(ctx, &p.ex, dp)
}

func TestAT13_F35_RestartFirstApplyRendersConfiguredDenyGrowth(t *testing.T) {
	for _, adopt := range []bool{true, false} {
		name := "same_boot_adoption"
		if !adopt {
			name = "missing_heartbeat_baseline"
		}
		t.Run(name, func(t *testing.T) {
			r := newRig(t)
			alignControlClock(r)
			// TEST ONLY: documentation destinations must initially be outside
			// the spec deny set for this to be an actual enlargement.
			r.eng.cfg.TestMode = true
			r.eng.cfg.TestExemptV4 = []string{"203.0.113.0/24", "198.51.100.0/24"}
			r.eng.cfg.ExtraDeniedV4 = []string{"203.0.113.1/32"}
			r.eng.cfg.HostV4 = []string{"198.51.100.1/32"}
			mono, err := enforce.MonotonicNow()
			if err != nil {
				t.Fatal(err)
			}
			p := &f35RestartPlane{memPlane: r.plane, offset: mono - r.clock.Mono()}
			r.eng.plane = p
			serveBoundGeneration(t, r)
			r.manualUsage(t)
			if !r.eng.serving || len(p.results) == 0 || !p.results[len(p.results)-1].Verified {
				t.Fatalf("initial real Apply failed: %v", p.errors)
			}
			if err := r.eng.Close(); err != nil {
				t.Fatal(err)
			}
			if !adopt {
				if err := os.Remove(r.eng.cfg.HeartbeatPath); err != nil {
					t.Fatal(err)
				}
			}
			cfg := r.eng.cfg
			cfg.ExtraDeniedV4 = []string{"203.0.113.1/32", "203.0.113.9/32"}
			cfg.HostV4 = []string{"198.51.100.1/32", "198.51.100.9/32"}
			raw, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(r.dir, "restart-config.json")
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err = LoadConfig(path) // the command's production config reader
			if err != nil {
				t.Fatal(err)
			}
			next, err := Start(Options{Config: cfg, Clock: r.clock, Plane: p, MeterSource: staticSource{boot: r.eng.bootID.String()}, Prober: passProbe{}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = next.Close() })
			r.eng = next
			before := len(p.models)
			if err := next.Boot(t.Context()); err != nil {
				t.Fatal(err)
			}
			if len(p.models) != before {
				t.Fatal("Boot unexpectedly ran a full Apply")
			}
			if next.serving != adopt || mustBeat(t, r).Serving != adopt {
				t.Fatalf("unexpected adoption: serving=%v errors=%v", next.serving, next.decision.Errors)
			}
			if adopt {
				// Known gap: the real verifier accepts the old set contents.
				// This documents current behavior, not a safe-adoption guarantee.
				if slices.Contains(p.ex.table.sets["denied_v4"].elems, "203.0.113.9/32") || slices.Contains(p.ex.table.sets["host_v4"].elems, "198.51.100.9/32") {
					t.Fatal("fixture did not retain old sets across adoption")
				}
				t.Log("known gap: Boot resumed serving on old sets before the first Step")
			}
			stepOK(t, r) // Run calls Step immediately after Boot.
			if len(p.models) <= before || !p.results[before].Verified || !next.serving {
				t.Fatalf("first Step did not apply: models=%d errors=%v", len(p.models), next.decision.Errors)
			}
			first := p.models[before]
			if !slices.Contains(first.ExtraDeniedV4, netip.MustParsePrefix("203.0.113.9/32")) || !slices.Contains(first.HostV4, netip.MustParsePrefix("198.51.100.9/32")) || len(first.Revoke) != 0 {
				t.Fatalf("first Apply lost new config or invented identity revocation: %+v", first)
			}
			for set, want := range map[string][]string{
				"denied_v4": {"203.0.113.1/32", "203.0.113.9/32"},
				"host_v4":   {"198.51.100.1/32", "198.51.100.9/32"},
			} {
				var elements []string
				for _, line := range strings.Split(p.results[before].Plan.NFT, "\n") {
					if strings.HasPrefix(line, "add element inet fvpn "+set+" {") {
						body, err := braceBody(line)
						if err != nil {
							t.Fatal(err)
						}
						elements = append(elements, splitElems(body)...)
					}
				}
				for _, prefix := range want {
					if !slices.Contains(elements, prefix) {
						t.Fatalf("first Apply did not render %s in %s: %v", prefix, set, elements)
					}
				}
			}
			if len(p.ex.deletes) != 0 {
				t.Fatalf("set growth deleted conntrack: %v", p.ex.deletes)
			}
		})
	}
}
