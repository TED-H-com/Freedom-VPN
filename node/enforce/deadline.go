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
		lease.SignedServerTime.IsZero() || lease.MaxOfflineSeconds == 0 {
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

// elementTimeout returns the kernel set timeout in whole seconds, rounded
// down. A positive value strictly below one second yields 0 so the element is
// left out rather than rounded up (§16.3). TimeoutCeilSeconds caps the result
// so a wall-clock step cannot lengthen a lease the agent already counted down.
func elementTimeout(binding Binding, lease MemberLease, now time.Time) (seconds uint32, admit bool) {
	deadline, ok := ForwardingDeadline(lease)
	if !ok || now.IsZero() {
		return 0, false
	}
	if !binding.ValidFrom.IsZero() && now.Before(binding.ValidFrom) {
		return 0, false
	}
	if !binding.ValidUntil.IsZero() && binding.ValidUntil.Before(deadline) {
		deadline = binding.ValidUntil
	}
	if !deadline.After(now) {
		return 0, false
	}
	sec := uint64(deadline.Sub(now) / time.Second)
	if sec == 0 {
		return 0, false
	}
	if lease.TimeoutCeilSeconds > 0 && sec > lease.TimeoutCeilSeconds {
		sec = lease.TimeoutCeilSeconds
	}
	if sec > 0xffffffff {
		sec = 0xffffffff
	}
	return uint32(sec), true
}
