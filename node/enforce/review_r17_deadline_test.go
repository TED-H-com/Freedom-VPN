package enforce

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func reviewLease(now time.Time, end time.Time) MemberLease {
	return MemberLease{
		MemberID: uuid.New(), EffectiveServiceEnd: end, NodeLeaseEnd: now.Add(24 * time.Hour),
		SignedServerTime: now.Add(-time.Hour), MaxOfflineSeconds: 86400,
	}
}

func TestReviewR17_DeadlineUsesMillisecondQuantumAndInstallGuard(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	binding := Binding{ValidFrom: now.Add(-24 * time.Hour), ValidUntil: now.Add(24 * time.Hour)}
	m := Model{Now: now}
	lease := reviewLease(now, now.Add(1500*time.Millisecond))
	got, ok := memberTimeout(binding, lease, m)
	if !ok || got != 1500*time.Millisecond {
		t.Fatalf("whole-second floor: %s ok=%v", got, ok)
	}
	m.timeoutReserve = DeadlineInstallBound
	lease = reviewLease(now, now.Add(2*time.Second))
	got, ok = memberTimeout(binding, lease, m)
	if !ok || got != 2*time.Second-DeadlineInstallBound {
		t.Fatalf("guard reserve: %s ok=%v", got, ok)
	}
	lease = reviewLease(now, now.Add(DeadlineInstallBound+DeadlineGranularity))
	got, ok = memberTimeout(binding, lease, m)
	if !ok || got != DeadlineGranularity {
		t.Fatalf("1ms quantum: %s ok=%v", got, ok)
	}
	lease = reviewLease(now, now.Add(DeadlineInstallBound))
	if _, ok = memberTimeout(binding, lease, m); ok {
		t.Fatal("a renewal inside the install guard was admitted")
	}

	monoNow := 5 * time.Second
	monoDead := monoNow + time.Second
	lease = reviewLease(now, now.Add(time.Hour))
	lease.DeadlineMono = &monoDead
	m.Now = now.Add(-time.Hour)
	m.NowMono = monoNow
	got, ok = memberTimeout(binding, lease, m)
	want := time.Second - DeadlineInstallBound
	if !ok || got != want {
		t.Fatalf("wall step changed the monotonic deadline: %s want %s ok=%v", got, want, ok)
	}
}
