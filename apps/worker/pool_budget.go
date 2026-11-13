package worker

import (
	"context"
	"fmt"
)

const maxWorkerConnections = 60
const readinessConnections = 1

// jobRegistrations is the production catalog. The plain Registry remains the
// callable handler view used by the dispatcher and domain-level callers.
type jobRegistration struct {
	Peak    int
	Handler Handler
}
type jobRegistrations map[string]jobRegistration

func (r jobRegistrations) Register(name string, peak int, handler Handler) {
	if name == "" || handler == nil || r[name].Handler != nil {
		panic("invalid or duplicate worker handler")
	}
	r[name] = jobRegistration{Peak: peak, Handler: handler}
}
func (r jobRegistrations) handlers() Registry {
	out := Registry{}
	for name, job := range r {
		out.Register(name, job.Handler)
	}
	return out
}

// Runner declares the peak alongside the supervised function. Peaks include
// connections waiting for session locks as well as connections executing work.
type Runner struct {
	Name string
	Peak int
	Run  func(context.Context) error
}
type registration struct {
	Jobs    jobRegistrations
	Runners []Runner
}

func (r registration) runnerPeak(name string) int {
	for _, runner := range r.Runners {
		if runner.Name == name {
			return runner.Peak
		}
	}
	return 0
}

func productionRegistration() (registration, error) {
	out := registration{Jobs: jobRegistrations{}}
	for _, domain := range Domains() {
		if domain.Describe == nil {
			return out, fmt.Errorf("worker domain %s: connection peaks required", domain.Name)
		}
		described := domain.Describe()
		for name, job := range described.Jobs {
			if name == "" || job.Handler == nil || job.Peak < 1 {
				return out, fmt.Errorf("worker event %s: connection peak required", name)
			}
			if out.Jobs[name].Handler != nil {
				return out, fmt.Errorf("duplicate worker event: %s", name)
			}
			out.Jobs[name] = job
		}
		for _, runner := range described.Runners {
			if runner.Name == "" || runner.Run == nil || runner.Peak < 1 {
				return out, fmt.Errorf("worker runner %s: connection peak required", runner.Name)
			}
			if out.runnerPeak(runner.Name) != 0 {
				return out, fmt.Errorf("duplicate worker runner: %s", runner.Name)
			}
			out.Runners = append(out.Runners, runner)
		}
	}
	return out, nil
}

// PoolSize uses the same pure registration factories as Open, before any domain
// configuration or pool is opened. All production domains share this budget.
func PoolSize(concurrency int) (int32, error) {
	r, err := productionRegistration()
	if err != nil {
		return 0, err
	}
	return r.poolSize(concurrency)
}

type PoolBudgetError struct{ Concurrency, Maximum int }

func (e *PoolBudgetError) Error() string {
	return fmt.Sprintf("worker_concurrency %d exceeds the database pool limit of %d connections; largest accepted worker_concurrency is %d", e.Concurrency, maxWorkerConnections, e.Maximum)
}
func (e *PoolBudgetError) Setting() string { return "worker_concurrency" }

func (r registration) poolSize(concurrency int) (int32, error) {
	largest, fixed := 0, readinessConnections
	for _, job := range r.Jobs {
		largest = max(largest, job.Peak)
	}
	for _, runner := range r.Runners {
		fixed += runner.Peak
	}
	if largest < 1 || fixed >= maxWorkerConnections {
		return 0, fmt.Errorf("invalid worker connection budget")
	}
	maximum := (maxWorkerConnections - fixed) / largest
	if concurrency > maximum {
		return 0, &PoolBudgetError{Concurrency: concurrency, Maximum: maximum}
	}
	if concurrency < 1 {
		return 0, fmt.Errorf("worker_concurrency must be positive")
	}
	return int32(concurrency*largest + fixed), nil
}
