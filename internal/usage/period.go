package usage

import (
	"errors"
	"fmt"
	"math/bits"
	"time"
)

// Granularity is a rollup bucket size. Periods are UTC calendar months
// (§10.4, a pending business decision that the contract fixes for v1 as
// `YYYY-MM`) and UTC days for the member day view.
type Granularity string

// Rollup granularities.
const (
	GranularityMonth Granularity = "month"
	GranularityDay   Granularity = "day"
)

// Valid reports whether g is a known granularity.
func (g Granularity) Valid() bool { return g == GranularityMonth || g == GranularityDay }

const (
	monthLayout = "2006-01"
	dayLayout   = "2006-01-02"
)

// ErrPeriod reports a malformed period ID.
var ErrPeriod = errors.New("usage: invalid period")

// Period is one half-open UTC bucket [Start, End).
type Period struct {
	ID          string
	Granularity Granularity
	Start, End  time.Time
}

// MonthOf returns the UTC calendar month containing t.
func MonthOf(t time.Time) Period {
	t = t.UTC()
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return Period{ID: start.Format(monthLayout), Granularity: GranularityMonth, Start: start, End: start.AddDate(0, 1, 0)}
}

// DayOf returns the UTC day containing t.
func DayOf(t time.Time) Period {
	t = t.UTC()
	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return Period{ID: start.Format(dayLayout), Granularity: GranularityDay, Start: start, End: start.AddDate(0, 0, 1)}
}

// PeriodOf returns the bucket of granularity g containing t.
func PeriodOf(g Granularity, t time.Time) Period {
	if g == GranularityDay {
		return DayOf(t)
	}
	return MonthOf(t)
}

// ParseMonth parses a `YYYY-MM` period ID (contract PeriodId).
func ParseMonth(id string) (Period, error) {
	if len(id) != len(monthLayout) {
		return Period{}, fmt.Errorf("%w: %q", ErrPeriod, id)
	}
	t, err := time.Parse(monthLayout, id)
	if err != nil || t.Format(monthLayout) != id {
		return Period{}, fmt.Errorf("%w: %q", ErrPeriod, id)
	}
	return MonthOf(t), nil
}

// Segment is the part of a sample that falls into one period.
type Segment struct {
	Period
	Upload, Download uint64
	// Estimated is true when the sample crossed a boundary of this
	// granularity and the bytes were split by overlap duration (§10.9).
	Estimated bool
}

// Split allocates a sample interval [start, end) and its byte counts to the
// periods of granularity g it overlaps (§10.9).
//
// A sample inside one period is exact. A sample that crosses a boundary is
// split in proportion to the overlap duration in nanoseconds using exact
// 128-bit integer arithmetic: every segment except the last gets
// floor(bytes*overlap/duration) and the last (later) segment gets the
// remainder, so the parts always sum to the original value. All segments of
// a split sample are marked Estimated (boundary_estimated); a time split is
// never presented as an exact measurement.
func Split(start, end time.Time, upload, download uint64, g Granularity) ([]Segment, error) {
	start, end = start.UTC(), end.UTC()
	if !end.After(start) {
		return nil, errors.New("usage: interval_end must be after interval_start")
	}
	total := end.Sub(start)
	if total <= 0 || total == time.Duration(1<<63-1) {
		return nil, errors.New("usage: interval duration out of range")
	}
	var periods []Period
	for p := PeriodOf(g, start); p.Start.Before(end); p = PeriodOf(g, p.End) {
		periods = append(periods, p)
	}
	if len(periods) == 1 {
		return []Segment{{Period: periods[0], Upload: upload, Download: download}}, nil
	}
	segs := make([]Segment, len(periods))
	var upSum, downSum uint64
	for i, p := range periods {
		segs[i] = Segment{Period: p, Estimated: true}
		if i == len(periods)-1 {
			segs[i].Upload = upload - upSum
			segs[i].Download = download - downSum
			break
		}
		overlap := minTime(p.End, end).Sub(maxTime(p.Start, start))
		segs[i].Upload = mulDiv(upload, uint64(overlap), uint64(total))
		segs[i].Download = mulDiv(download, uint64(overlap), uint64(total))
		upSum += segs[i].Upload
		downSum += segs[i].Download
	}
	return segs, nil
}

// mulDiv returns floor(a*b/c) for b <= c, c > 0, without overflow.
func mulDiv(a, b, c uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	q, _ := bits.Div64(hi, lo, c) // hi < c because b <= c
	return q
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// MonthsOverlapping lists the month IDs that overlap [from, until).
func MonthsOverlapping(from, until time.Time) []string {
	var ids []string
	for p := MonthOf(from); p.Start.Before(until); p = MonthOf(p.End) {
		ids = append(ids, p.ID)
	}
	return ids
}
