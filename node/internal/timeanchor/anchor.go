package timeanchor

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Skew defaults. They are conservative local limits, not a measured NTP
// bound (待實測). A step outside the window is an anomaly (§16.3).
const (
	DefaultBackwardSkew = 2 * time.Second
	DefaultForwardSkew  = 60 * time.Second
)

// ErrUntrusted means there is no anchor for this boot.
var ErrUntrusted = errors.New("timeanchor: no trusted time anchor")

// ErrAnomaly means the wall clock moved relative to the monotonic clock.
var ErrAnomaly = errors.New("timeanchor: clock anomaly")

// Anchor is one observation of signed server time (§16.3).
// Trusted time is Server plus the monotonic elapsed time since Mono.
// Wall is recorded only so a later jump can be detected. It is never added
// to a deadline.
type Anchor struct {
	BootID  uuid.UUID
	Server  time.Time
	Mono    time.Duration
	Wall    time.Time
	Trusted bool
}

// Observe builds an anchor from a signed server time and a simultaneous
// local reading. The result is trusted for this boot only.
func Observe(boot uuid.UUID, server time.Time, c Clock) (Anchor, error) {
	if boot == uuid.Nil || server.IsZero() || c == nil {
		return Anchor{}, ErrUntrusted
	}
	mono := c.Mono()
	if mono <= 0 {
		return Anchor{}, ErrUntrusted
	}
	return Anchor{
		BootID:  boot,
		Server:  server.UTC(),
		Mono:    mono,
		Wall:    c.Wall(),
		Trusted: true,
	}, nil
}

// Check reports whether a is still a safe base for deadlines.
// boot must be the current boot. A different boot is a reboot: the anchor
// is lost and forwarding stays off (§16.3).
func (a Anchor) Check(boot uuid.UUID, c Clock, backward, forward time.Duration) error {
	if !a.Trusted || a.BootID == uuid.Nil || a.Server.IsZero() || a.Mono <= 0 {
		return ErrUntrusted
	}
	if boot != a.BootID {
		return ErrUntrusted
	}
	if c == nil {
		return ErrUntrusted
	}
	if backward <= 0 {
		backward = DefaultBackwardSkew
	}
	if forward <= 0 {
		forward = DefaultForwardSkew
	}
	nowMono := c.Mono()
	elapsed := nowMono - a.Mono
	if nowMono <= 0 || elapsed < 0 {
		return ErrAnomaly
	}
	expectedWall := a.Wall.Add(elapsed)
	wall := c.Wall()
	// A backward step makes wall+skew earlier than the monotonic expectation.
	if wall.Add(backward).Before(expectedWall) {
		return ErrAnomaly
	}
	if wall.After(expectedWall.Add(forward)) {
		return ErrAnomaly
	}
	return nil
}

// TrustedNow is the signed server time plus monotonic elapsed time.
// It does not consult the wall clock except to refuse an anomaly.
func (a Anchor) TrustedNow(boot uuid.UUID, c Clock, backward, forward time.Duration) (time.Time, error) {
	if err := a.Check(boot, c, backward, forward); err != nil {
		return time.Time{}, err
	}
	return a.Server.Add(c.Mono() - a.Mono), nil
}

// MemberDeadline is the monotonic instant at which a member must stop.
// window is lease_end-issued_at, already reduced to the earliest of the
// member lease end and the effective service end (§16.3). Wall time is not
// an input, so a wall-clock jump cannot move the result.
func MemberDeadline(monoSent time.Duration, window time.Duration) (time.Duration, bool) {
	if monoSent <= 0 || window <= 0 {
		return 0, false
	}
	return monoSent + window, true
}

// Remaining is how long until deadline at monoNow. It never increases when
// monoNow increases, and it ignores the wall clock.
func Remaining(deadline, monoNow time.Duration) time.Duration {
	if deadline <= monoNow {
		return 0
	}
	return deadline - monoNow
}

// Window is the signed allowance from issuedAt to the earlier of leaseEnd
// and serviceEnd. The interval is half-open at the end: a zero or negative
// result does not admit the member.
func Window(issuedAt, leaseEnd, serviceEnd time.Time) time.Duration {
	if issuedAt.IsZero() || !leaseEnd.After(issuedAt) || !serviceEnd.After(issuedAt) {
		return 0
	}
	w := leaseEnd.Sub(issuedAt)
	if s := serviceEnd.Sub(issuedAt); s < w {
		w = s
	}
	if w <= 0 {
		return 0
	}
	return w
}
