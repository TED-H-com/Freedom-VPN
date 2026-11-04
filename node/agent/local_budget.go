package agent

import (
	"context"
	"time"
)

// A pass owns one deadline and one quarantine allowance. Keep its context even
// through cancellation-independent helpers. Engine.Step is single-threaded.
type localBudgetKey struct{}
type localBudget struct {
	quarantineAttempted bool
	quarantineBaseline  bool
	ctx                 context.Context
	cancel              context.CancelFunc
	quarantine          context.Context
	cancelQuarantine    context.CancelFunc
}

func newLocalBudget(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), timeout)
	budget := &localBudget{cancel: cancel}
	ctx = context.WithValue(ctx, localBudgetKey{}, budget)
	budget.ctx = ctx
	return ctx, func() {
		if budget.cancelQuarantine != nil {
			budget.cancelQuarantine()
		}
		cancel()
	}
}

func localContext(ctx context.Context) context.Context {
	if budget, ok := ctx.Value(localBudgetKey{}).(*localBudget); ok {
		// Preserve a helper's shorter deadline as well as the enclosing budget.
		if deadline, ok := ctx.Deadline(); ok {
			if outer, _ := budget.ctx.Deadline(); !deadline.After(outer) {
				return ctx
			}
		}
		return budget.ctx
	}
	return context.WithoutCancel(ctx)
}

func cleanupBudget(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Value(localBudgetKey{}).(*localBudget); ok {
		return localContext(ctx), func() {}
	}
	return newLocalBudget(ctx, 5*time.Second)
}

// Boot's adoption work follows TERM, unlike Step's independent local checks.
// Keep that lifetime separate from the budget for control-plane HTTP requests.
type bootCallerKey struct{}

func localWorkContext(ctx context.Context) context.Context {
	if caller, ok := ctx.Value(bootCallerKey{}).(context.Context); ok {
		return caller
	}
	return localContext(ctx)
}
