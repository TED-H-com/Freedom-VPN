package enforce

import (
	"time"

	"github.com/google/uuid"
)

// ForwardingDeadline is the earliest instant a member may still be forwarded
// (§16.3). A zero time and ok=false means a required input is missing.
func ForwardingDeadline(lease MemberLease) (time.Time, bool) {
	// Every input is required. A zero signed time or a non-positive offline
	// cap must not fall open to the later service or node end (§16.3).
	if lease.MemberID == uuid.Nil || lease.EffectiveServiceEnd.IsZero() || lease.NodeLeaseEnd.IsZero() ||
		lease.SignedServerTime.IsZero() || lease.MaxOfflineSeconds == 0 || lease.MaxOfflineSeconds > uint64((1<<63-1)/int64(time.Second)) {
		return time.Time{}, false
	}
	end := lease.EffectiveServiceEnd
	if lease.NodeLeaseEnd.Before(end) {
		end = lease.NodeLeaseEnd
	}
	offline := lease.SignedServerTime.Add(time.Duration(lease.MaxOfflineSeconds) * time.Second)
	if offline.Before(end) {
		end = offline
	}
	return end, true
}

// elementTimeout truncates only to DeadlineGranularity, always toward earlier.
// The wall limits remain mandatory; DeadlineMono additionally caps them with an
// absolute monotonic deadline. Apply supplies the guarded installation reserve.
func elementTimeout(binding Binding, lease MemberLease, now time.Time) (time.Duration, bool) {
	return memberTimeout(binding, lease, Model{Now: now})
}

func memberTimeout(binding Binding, lease MemberLease, m Model) (time.Duration, bool) {
	deadline, ok := ForwardingDeadline(lease)
	if !ok || m.Now.IsZero() || (!binding.ValidFrom.IsZero() && m.Now.Before(binding.ValidFrom)) {
		return 0, false
	}
	if !binding.ValidUntil.IsZero() && binding.ValidUntil.Before(deadline) {
		deadline = binding.ValidUntil
	}
	remaining := deadline.Sub(m.Now)
	if remaining <= 0 {
		return 0, false
	}
	if lease.TimeoutCeilSeconds > 0 && lease.TimeoutCeilSeconds <= uint64((1<<63-1)/int64(time.Second)) {
		remaining = min(remaining, time.Duration(lease.TimeoutCeilSeconds)*time.Second)
	}
	if lease.DeadlineMono != nil {
		if m.NowMono <= 0 || *lease.DeadlineMono <= m.NowMono {
			return 0, false
		}
		remaining = min(remaining, *lease.DeadlineMono-m.NowMono)
	}
	if remaining <= m.timeoutReserve {
		return 0, false
	}
	remaining = (remaining - m.timeoutReserve).Truncate(DeadlineGranularity)
	return remaining, remaining >= DeadlineGranularity
}

// probeTimeout is the kernel set-element lifetime for a synthetic probe, in
// whole seconds (the legacy synthetic-probe contract, separate from D-104). A positive gap
// strictly below one second is not admitted (§13.9).
func probeTimeout(expires, now time.Time) (uint32, bool) {
	if now.IsZero() || !expires.After(now) {
		return 0, false
	}
	sec := uint64(expires.Sub(now) / time.Second)
	if sec == 0 {
		return 0, false
	}
	if sec > 0xffffffff {
		sec = 0xffffffff
	}
	return uint32(sec), true
}
