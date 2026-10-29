package timeanchor

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAT29_WallJumpDoesNotExtendDeadline(t *testing.T) {
	boot := uuid.Must(uuid.NewV7())
	start := time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC)
	clock := NewManual(start, 10*time.Hour)
	anchor, err := Observe(boot, start, clock)
	if err != nil {
		t.Fatal(err)
	}
	issued := start
	leaseEnd := issued.Add(time.Hour)
	serviceEnd := issued.Add(30 * time.Minute)
	window := Window(issued, leaseEnd, serviceEnd)
	if window != 30*time.Minute {
		t.Fatalf("window %s, want 30m (earliest of lease and service end)", window)
	}
	monoSent := clock.Mono()
	deadline, ok := MemberDeadline(monoSent, window)
	if !ok {
		t.Fatal("deadline")
	}
	clock.Advance(10 * time.Minute)
	before := Remaining(deadline, clock.Mono())
	if before != 20*time.Minute {
		t.Fatalf("remaining %s, want 20m", before)
	}
	// A one-hour step backward must not put those 60 minutes back on the lease.
	clock.JumpWall(-time.Hour)
	after := Remaining(deadline, clock.Mono())
	if after != before {
		t.Fatalf("wall jump changed remaining from %s to %s", before, after)
	}
	if _, err := anchor.TrustedNow(boot, clock, DefaultBackwardSkew, DefaultForwardSkew); err != ErrAnomaly {
		t.Fatalf("backward jump err = %v, want anomaly", err)
	}
	// A forward step is also an anomaly and still does not extend the deadline.
	clock.JumpWall(time.Hour + 2*time.Minute)
	if _, err := anchor.TrustedNow(boot, clock, DefaultBackwardSkew, DefaultForwardSkew); err != ErrAnomaly {
		t.Fatalf("forward jump err = %v, want anomaly", err)
	}
	if got := Remaining(deadline, clock.Mono()); got != before {
		t.Fatalf("forward jump changed remaining to %s", got)
	}
}

func TestRebootLosesAnchor(t *testing.T) {
	boot := uuid.Must(uuid.NewV7())
	clock := NewManual(time.Date(2026, 10, 28, 15, 0, 0, 0, time.UTC), time.Minute)
	anchor, err := Observe(boot, clock.Wall(), clock)
	if err != nil {
		t.Fatal(err)
	}
	other := uuid.Must(uuid.NewV7())
	if err := anchor.Check(other, clock, 0, 0); err != ErrUntrusted {
		t.Fatalf("reboot err = %v, want untrusted", err)
	}
}

func TestReplayDoesNotRestartWindow(t *testing.T) {
	monoSent := 5 * time.Hour
	window := 24 * time.Hour
	deadline, ok := MemberDeadline(monoSent, window)
	if !ok {
		t.Fatal("deadline")
	}
	// A second delivery of the same lease keeps monoSent. Using "now" as a
	// new monoSent would add the full 24h again.
	now := monoSent + 3*time.Hour
	replayed, ok := MemberDeadline(monoSent, window)
	if !ok || replayed != deadline {
		t.Fatalf("replay deadline %s, want %s", replayed, deadline)
	}
	if Remaining(replayed, now) != 21*time.Hour {
		t.Fatalf("remaining %s", Remaining(replayed, now))
	}
	restarted, _ := MemberDeadline(now, window)
	if restarted <= deadline {
		t.Fatal("test setup: a restarted TTL should be later than the original")
	}
	if Remaining(restarted, now) != 24*time.Hour {
		t.Fatal("restarted TTL would grant a full day")
	}
}
