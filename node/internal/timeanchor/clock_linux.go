//go:build linux

package timeanchor

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// System uses time.Now for the wall clock and CLOCK_MONOTONIC for elapsed
// time. CLOCK_MONOTONIC is shared by every process on the node and resets
// on reboot, so the agent and the watchdog see the same age (§16.3).
// Suspend behaviour of this clock versus nftables jiffies is 待實測.
type System struct{}

// Wall returns the current wall time.
func (System) Wall() time.Time { return time.Now() }

// Mono returns time since an arbitrary boot epoch.
func (System) Mono() time.Duration {
	d, err := Mono()
	if err != nil {
		return 0
	}
	return d
}

// Mono reads CLOCK_MONOTONIC. A zero duration with a non-nil error means the
// clock cannot be trusted and the caller must fail closed.
func Mono() (time.Duration, error) {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &ts); err != nil {
		return 0, fmt.Errorf("timeanchor: CLOCK_MONOTONIC: %w", err)
	}
	d, ok := MonotonicDuration(int64(ts.Sec), int64(ts.Nsec))
	if !ok {
		return 0, fmt.Errorf("timeanchor: CLOCK_MONOTONIC: %w", unix.EOVERFLOW)
	}
	return d, nil
}

// MonotonicDuration converts a Linux monotonic sample without allowing either
// the seconds multiplication or the nanoseconds sum to wrap.
func MonotonicDuration(seconds, nanoseconds int64) (time.Duration, bool) {
	const maxInt64 = int64(1<<63 - 1)
	const minInt64 = -maxInt64 - 1
	const unit = int64(time.Second)
	if seconds > maxInt64/unit || seconds < minInt64/unit {
		return 0, false
	}
	scaled := seconds * unit
	if nanoseconds > 0 && scaled > maxInt64-nanoseconds || nanoseconds < 0 && scaled < minInt64-nanoseconds {
		return 0, false
	}
	return time.Duration(scaled) + time.Duration(nanoseconds), true
}
