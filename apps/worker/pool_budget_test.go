package worker

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
)

func TestAPI01_PoolBudgetBound(t *testing.T) {
	for _, tc := range []struct {
		concurrency int
		want        int32
	}{{1, 9}, {4, 24}, {11, 59}} {
		size, err := PoolSize(tc.concurrency)
		if err != nil || size != tc.want {
			t.Fatalf("concurrency=%d size=%d want=%d err=%v", tc.concurrency, size, tc.want, err)
		}
	}
	for _, n := range []int{12, 1000, math.MaxInt} {
		_, err := PoolSize(n)
		var bound *PoolBudgetError
		if !errors.As(err, &bound) || bound.Maximum != 11 || !strings.Contains(err.Error(), "largest accepted worker_concurrency is 11") {
			t.Fatalf("concurrency=%d: %v", n, err)
		}
		t.Log(err)
	}
	for _, n := range []int{0, -1} {
		if _, err := PoolSize(n); err == nil {
			t.Fatal("accepted nonpositive concurrency")
		}
	}
	// Open enforces the same bound before invoking builders or touching the pool.
	c := config.Defaults()
	c.WorkerConcurrency = 12
	_, err := Open(t.Context(), Dependencies{Config: c}, Domain{Name: "test", Build: func(context.Context, Dependencies) (Built, error) { t.Fatal("built above bound"); return Built{}, nil }})
	if err == nil || !strings.Contains(err.Error(), "largest accepted worker_concurrency is 11") {
		t.Fatal(err)
	}
	t.Log("Open:", err)
}

func TestAPI01_PoolBudgetReadiness(t *testing.T) {
	r, err := productionRegistration()
	if err != nil {
		t.Fatal(err)
	}
	largest, runners := 0, 0
	for _, job := range r.Jobs {
		largest = max(largest, job.Peak)
	}
	for _, runner := range r.Runners {
		runners += runner.Peak
	}
	size, err := PoolSize(4)
	if err != nil || int(size)-4*largest-runners != 1 {
		t.Fatalf("readiness reserve: size=%d event=%d runners=%d err=%v", size, largest, runners, err)
	}
}

func TestAPI01_OpenRejectsMissingPeak(t *testing.T) {
	c := config.Defaults()
	h := func(context.Context, db.ClaimedEvent) (string, error) { return "test", nil }
	for _, built := range []Built{
		{Jobs: Registry{"test.without_peak": h}},
		{Runners: []func(context.Context) error{func(context.Context) error { return nil }}},
		{RunnerRegistrations: []Runner{{Name: "members.schedule", Run: func(context.Context) error { return nil }}}},
	} {
		_, err := Open(t.Context(), Dependencies{Config: c}, Domain{Name: "test", Build: func(context.Context, Dependencies) (Built, error) { return built, nil }})
		if err == nil || !strings.Contains(err.Error(), "peak") {
			t.Fatalf("missing peak accepted: %v", err)
		}
		t.Log(err)
	}
	// This assertion also exercises a removed production peak mutation before
	// pool/configuration validation can hide it behind a different startup error.
	r, err := productionRegistration()
	if err != nil {
		t.Fatal(err)
	}
	for name, job := range r.Jobs {
		if job.Peak < 1 {
			t.Fatalf("missing peak: %s", name)
		}
	}
}
