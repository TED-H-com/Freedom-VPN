package agent

import "context"

// Barrier is the metering barrier the usage task implements (§10.7, §13.6).
// The agent does not sample counters and does not upload usage. FinalSample
// and OpenEpoch stay idempotent while member forwarding is blocked.
type Barrier interface {
	// Health reports whether the meter path matches the kernel forward hook
	// (§10.3). ok false keeps member forwarding off.
	Health(ctx context.Context) (ok bool, err error)
}

// NopBarrier is the stand-in until the usage task is wired. The agent then
// trusts only the configured netbird_backend, which must be "kernel".
type NopBarrier struct{}

// Health reports that this process has no meter of its own.
func (NopBarrier) Health(context.Context) (bool, error) { return true, nil }
