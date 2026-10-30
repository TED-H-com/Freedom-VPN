package isolation

import "time"

// WithinOneTick reports whether observed is at or after deadline and no later
// than one agent tick after it. An earlier observation means forwarding stopped
// before the signed deadline.
func WithinOneTick(deadline, observed time.Time, tick time.Duration) bool {
	if deadline.IsZero() || observed.IsZero() || tick <= 0 {
		return false
	}
	if observed.Before(deadline) {
		return false
	}
	return !observed.After(deadline.Add(tick))
}
