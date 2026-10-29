package timeanchor

import (
	"sync"
	"time"
)

// Clock is the wall clock and a monotonic elapsed clock. The two readings
// are taken together by the caller; implementations must not let wall time
// change the monotonic reading.
type Clock interface {
	Wall() time.Time
	Mono() time.Duration
}

// Manual is a test clock. Wall and Mono move only when the test says so.
type Manual struct {
	mu   sync.Mutex
	wall time.Time
	mono time.Duration
}

// NewManual returns a clock frozen at wall and mono.
func NewManual(wall time.Time, mono time.Duration) *Manual {
	return &Manual{wall: wall, mono: mono}
}

// Wall returns the frozen wall time.
func (m *Manual) Wall() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.wall
}

// Mono returns the frozen monotonic reading.
func (m *Manual) Mono() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mono
}

// Set replaces both readings.
func (m *Manual) Set(wall time.Time, mono time.Duration) {
	m.mu.Lock()
	m.wall = wall
	m.mono = mono
	m.mu.Unlock()
}

// Advance adds d to the monotonic clock and to the wall clock.
func (m *Manual) Advance(d time.Duration) {
	m.mu.Lock()
	m.mono += d
	m.wall = m.wall.Add(d)
	m.mu.Unlock()
}

// JumpWall moves only the wall clock. A negative d is a step backward (§16.3).
func (m *Manual) JumpWall(d time.Duration) {
	m.mu.Lock()
	m.wall = m.wall.Add(d)
	m.mu.Unlock()
}
