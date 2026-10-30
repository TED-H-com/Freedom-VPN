package isolation

import (
	"testing"
	"time"
)

func TestWithinOneTick(t *testing.T) {
	deadline := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	tick := 5 * time.Second
	if WithinOneTick(deadline, deadline.Add(-time.Millisecond), tick) {
		t.Fatal("early baseline counted")
	}
	if !WithinOneTick(deadline, deadline, tick) || !WithinOneTick(deadline, deadline.Add(tick), tick) {
		t.Fatal("deadline and one tick must count")
	}
	if WithinOneTick(deadline, deadline.Add(tick+time.Millisecond), tick) {
		t.Fatal("past one tick counted")
	}
	if WithinOneTick(time.Time{}, deadline, tick) || WithinOneTick(deadline, time.Time{}, tick) || WithinOneTick(deadline, deadline, 0) {
		t.Fatal("missing input counted")
	}
}
