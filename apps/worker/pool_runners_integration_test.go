//go:build integration

package worker

import (
	"context"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAPI01_RunnerPoolPeaks(t *testing.T) {
	r, err := productionRegistration()
	if err != nil {
		t.Fatal(err)
	}
	for _, runner := range r.Runners {
		t.Run(runner.Name, func(t *testing.T) {
			runBudgetPair(t, runner.Name, runner.Peak, func(t *testing.T, size int) (*pgxpool.Pool, func(context.Context) (string, error), *budgetTrace) {
				switch runner.Name {
				case "dns.schedule":
					return dnsBudgetCase(t, runner.Name, size)
				case "members.schedule":
					f := newMemberPolicyFixture(t, true)
					tr := &budgetTrace{}
					pool := budgetPool(t, f.pool, max(1, size), tr)
					f.services.HTTP.Members.Pool = pool
					return pool, func(ctx context.Context) (string, error) {
						err := memberScheduleCycle(ctx, f.services)
						return "cycle", err
					}, tr
				case "nodes.schedule":
					base, _, _ := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
					tr := &budgetTrace{}
					pool := budgetPool(t, base.Pool, max(1, size), tr)
					runtime, err := httpwire.Open(t.Context(), pool, base.Config, nil)
					if err != nil {
						t.Fatal(err)
					}
					return pool, func(ctx context.Context) (string, error) { err := nodeScheduleCycle(ctx, runtime); return "cycle", err }, tr
				default:
					t.Fatalf("runner %s has no peak fixture", runner.Name)
					return nil, nil, nil
				}
			})
		})
	}
}
