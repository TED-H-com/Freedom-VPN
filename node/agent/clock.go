package agent

import (
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/internal/timeanchor"
)

// systemClock is the production clock. Mono is CLOCK_MONOTONIC, shared with
// the watchdog (§16.3). A failure returns 0 so the anchor stays untrusted.
type systemClock struct{}

func (systemClock) Wall() time.Time { return time.Now().UTC() }

func (systemClock) Mono() time.Duration {
	d, err := timeanchor.Mono()
	if err != nil {
		return 0
	}
	return d
}
