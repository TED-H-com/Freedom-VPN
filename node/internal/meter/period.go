package meter

import "time"

// DefaultSampleInterval is the §10.5 recommended default.
const DefaultSampleInterval = 10 * time.Second

// nextBoundaryAfter is the next UTC month start strictly after t.
// A sample stamped on a month start has already closed that boundary, so
// the following month is next (§10.9).
func nextBoundaryAfter(t time.Time) time.Time {
	t = t.UTC()
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	next := start.AddDate(0, 1, 0)
	if !t.Before(next) {
		return next.AddDate(0, 1, 0)
	}
	return next
}

// stampEnd chooses the interval end. A sample that lands within one sample
// interval after a UTC month boundary is stamped on the boundary, so the
// half-open interval stays inside the month that just ended (§10.9).
// A later sample keeps the real clock and is boundary_estimated: the
// boundary sample did not succeed. The node does not split the bytes.
// The central projector owns the split.
func stampEnd(last, now time.Time, interval time.Duration) (end time.Time, estimated bool) {
	now = now.UTC()
	last = last.UTC()
	if interval <= 0 {
		interval = DefaultSampleInterval
	}
	boundary := nextBoundaryAfter(last)
	if now.Before(boundary) {
		return now, false
	}
	if now.Sub(boundary) <= interval {
		return boundary, false
	}
	return now, true
}

// NextDelay is how long the sampler waits. The wait ends at the earlier of
// the sample interval and the next UTC month boundary (§10.5, §10.9).
func NextDelay(last, now time.Time, interval time.Duration) time.Duration {
	if interval <= 0 {
		interval = DefaultSampleInterval
	}
	if last.IsZero() {
		return 0
	}
	wake := last.UTC().Add(interval)
	if b := nextBoundaryAfter(last); b.Before(wake) {
		wake = b
	}
	d := wake.Sub(now.UTC())
	if d < 0 {
		return 0
	}
	return d
}
