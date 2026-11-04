//go:build e2e

package e2e

import "context"

func (r *runner) timedSection(ctx context.Context, name string) func() {
	if r.timing == nil {
		return func() {}
	}
	end, err := r.timing.Begin(ctx, name)
	if err != nil {
		r.t.Fatal(err)
	}
	return func() {
		if err := end(); err != nil {
			r.t.Error(err)
		}
	}
}

func (r *runner) measured(ctx context.Context, name string, scenario func()) {
	defer r.timedSection(ctx, name)()
	scenario()
}
