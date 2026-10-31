package fakecontrol

import (
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
)

// UpgradeHealthFixture supplies actual signed desired-state/lease responses
// for an unpublished installer health stage. TEST ONLY capacity and TTL. The
// caller must supply a reachable, isolated lab target; a failed TCP probe is
// still a failed check. This does not fabricate an attestation or kernel state.
func UpgradeHealthFixture(release uuid.UUID, target api.EgressProbeTarget) Fixture {
	timeout := 1000
	return Fixture{
		ReleaseID: release, Publication: api.PublicationIntentUnpublished,
		Generation: 1, TTL: time.Hour,
		Capacity: &api.NodeCapacity{UpBps: 200_000_000, DownBps: 200_000_000, MemberUpBpsCap: presence.NewNullNullable[int64](), MemberDownBpsCap: presence.NewNullNullable[int64]()},
		Egress:   &api.EgressProbe{Targets: []api.EgressProbeTarget{target}, TimeoutMs: &timeout},
	}
}
