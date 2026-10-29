package leases

import (
	"testing"
	"time"
)

func TestComputeMemberLeaseEnd(t *testing.T) {
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	maxOffline := 24 * time.Hour
	nodeEnd := now.Add(24 * time.Hour)

	// Case 1: effective_service_end is sooner than 24h (now + 2h).
	serviceSooner := now.Add(2 * time.Hour)
	got1 := ComputeMemberLeaseEnd(now, maxOffline, serviceSooner, nodeEnd)
	if !got1.Equal(serviceSooner) {
		t.Errorf("sooner service end: got %v, want %v", got1, serviceSooner)
	}

	// Case 2: effective_service_end is later than 24h (now + 14d).
	serviceLater := now.Add(14 * 24 * time.Hour)
	got2 := ComputeMemberLeaseEnd(now, maxOffline, serviceLater, nodeEnd)
	if !got2.Equal(nodeEnd) {
		t.Errorf("later service end: got %v, want %v", got2, nodeEnd)
	}

	// Case 3: node_lease_end is shorter than maxOffline (now + 6h).
	shortNodeEnd := now.Add(6 * time.Hour)
	got3 := ComputeMemberLeaseEnd(now, maxOffline, serviceLater, shortNodeEnd)
	if !got3.Equal(shortNodeEnd) {
		t.Errorf("short node lease end: got %v, want %v", got3, shortNodeEnd)
	}
}
