//go:build !linux

package timeanchor

import (
	"errors"
	"time"
)

// System is unusable off Linux. Member forwarding stays off.
type System struct{}

// Wall returns the current wall time.
func (System) Wall() time.Time { return time.Now() }

// Mono returns zero. The anchor treats that as untrusted.
func (System) Mono() time.Duration { return 0 }

// Mono reports that this OS has no shared monotonic clock the node trusts.
func Mono() (time.Duration, error) {
	return 0, errors.New("timeanchor: CLOCK_MONOTONIC is only available on linux")
}
