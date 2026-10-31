//go:build e2e

package enrollment

import (
	"testing"
	"time"
)

func TestAT07_FreezeRequiresLiveGrantsAndExplicitWithdrawal(t *testing.T) {
	now := time.Now().UTC()
	positive := freezeSample{Start: now, End: now, NodeRead: true, CentralRead: true, HeartbeatWall: now, HeartbeatServing: true, ForwardDrop: true, MemberAccept: 2, LeaseMembers: 1, LeaseExpires: now.Add(time.Minute)}
	if !positive.serving() {
		t.Fatal("complete serving control rejected")
	}
	for _, alter := range []func(*freezeSample){
		func(s *freezeSample) { s.LeaseMembers = 0 },
		func(s *freezeSample) { s.MemberAccept = 0 },
		func(s *freezeSample) { s.HeartbeatServing = false },
		func(s *freezeSample) { s.HeartbeatWall = now.Add(-time.Minute) },
		func(s *freezeSample) { s.LeaseExpires = now },
		func(s *freezeSample) { s.CentralRead = false },
	} {
		s := positive
		alter(&s)
		if s.serving() {
			t.Fatal("incomplete serving control accepted")
		}
	}
	withdrawn := positive
	withdrawn.Frozen = true
	withdrawn.HeartbeatServing = false
	withdrawn.MemberAccept = 0
	withdrawn.Reason = "identity_frozen"
	if !withdrawn.withdrawn() {
		t.Fatal("explicit freeze withdrawal rejected")
	}
	for _, alter := range []func(*freezeSample){
		func(s *freezeSample) { s.MemberAccept = 1 },
		func(s *freezeSample) { s.HeartbeatServing = true },
		func(s *freezeSample) { s.Reason = "awaiting_member_grants" },
		func(s *freezeSample) { s.ForwardDrop = false },
		func(s *freezeSample) { s.NodeRead = false },
	} {
		s := withdrawn
		alter(&s)
		if s.withdrawn() {
			t.Fatal("missing withdrawal evidence accepted")
		}
	}
}
