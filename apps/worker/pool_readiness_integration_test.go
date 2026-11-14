//go:build integration

package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAPI01_PoolReadinessReserve(t *testing.T) {
	catalog, err := productionRegistration()
	if err != nil {
		t.Fatal(err)
	}
	size, err := PoolSize(4)
	if err != nil {
		t.Fatal(err)
	}
	pool := budgetPool(t, dbtest.New(t), int(size), nil)
	app, err := Open(t.Context(), Dependencies{Config: config.Defaults(), Pool: pool}, Domain{Name: "test", Build: func(context.Context, Dependencies) (Built, error) { return Built{Jobs: MemberJobs()}, nil }})
	if err != nil {
		t.Fatal(err)
	}
	largest, runners := 0, 0
	for _, job := range catalog.Jobs {
		largest = max(largest, job.Peak)
	}
	for _, runner := range catalog.Runners {
		runners += runner.Peak
	}
	held := []*pgxpool.Conn{}
	defer func() {
		for _, conn := range held {
			conn.Release()
		}
	}()
	for range 4*largest + runners {
		conn, err := pool.Acquire(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, conn)
	}
	response := httptest.NewRecorder()
	app.server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("readiness with all work reservations held: status=%d", response.Code)
	}
	t.Logf("readiness succeeded with %d work connections held in a %d-connection pool", len(held), size)
}

func TestAPI01_PoolPeakCoverage(t *testing.T) {
	catalog, err := productionRegistration()
	if err != nil {
		t.Fatal(err)
	}
	fixtures := map[string]bool{}
	for _, name := range append(memberPeakEvents(), domainPeakEvents()...) {
		fixtures[name] = true
	}
	for name := range nodes().Describe().Jobs {
		fixtures[name] = true
	}
	for name := range catalog.Jobs {
		if !fixtures[name] {
			t.Errorf("registered event lacks a peak fixture: %s", name)
		}
	}
	for name := range fixtures {
		if catalog.Jobs[name].Peak < 1 {
			t.Errorf("peak fixture lacks a registration: %s", name)
		}
	}
	t.Logf("peak fixtures cover all %d event types and %d runners", len(catalog.Jobs), len(catalog.Runners))
}
