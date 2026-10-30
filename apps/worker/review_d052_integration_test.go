//go:build integration

package worker

import (
	"context"
	"encoding/json"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

type multiPeerFake struct {
	peers map[string]devices.Peer
}

func (m *multiPeerFake) ObservePeer(_ context.Context, _, id string) (devices.Peer, error) {
	return m.peers[id], nil
}

func (m *multiPeerFake) SetAuthorization(_ context.Context, _, id string, allow bool, groups []uuid.UUID) error {
	p := m.peers[id]
	p.Authorized = allow
	p.Groups = groups
	m.peers[id] = p
	return nil
}

// TestReview_F02_ExitGroupRemovalLeavesStaleDeviceBindingOnRemovedGroupNode proves
// that when a member is removed from an exit group (e.g. Group B is removed via
// PATCH /api/v1/admin/members/{id} leaving Group A), Entitlements.Recompute emits
// member.entitlement_changed carrying only the NEW exit_group_ids ([Group A]).
// Because desiredstate.Compiler.RecompileBatch (internal/desiredstate/compiler.go:882-904)
// only recompiles nodes serving the groups in the event payload when len(rawGroups) > 0,
// Node B (serving Group B) is never recompiled and retains the member's DeviceBindings
// and MemberRatePolicies in its active DesiredStateEnvelope.
func TestReview_F02_ExitGroupRemovalLeavesStaleDeviceBindingOnRemovedGroupNode(t *testing.T) {
	f := newMemberPolicyFixture(t, true)
	ctx := t.Context()

	// Seed a second exit group (groupB) and a second published node (nodeB) serving groupB.
	groupA := uuid.New()
	if err := f.pool.QueryRow(ctx, `SELECT id FROM member_groups LIMIT 1`).Scan(&groupA); err != nil {
		t.Fatal(err)
	}
	groupB, nodeB, incB, exitB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	fpB := make([]byte, 32)
	fpB[0] = 2
	f.sql(`INSERT INTO member_groups(id,code,name) VALUES($1,'TEST ONLY group B','TEST ONLY group B')`, groupB)
	f.sql(`INSERT INTO nodes(id,lifecycle_state,provisioning_state,capacity) VALUES($1,'registered','ready','{"down_bps":1000000000,"up_bps":1000000000,"member_down_bps_cap":60000000,"member_up_bps_cap":30000000}')`, nodeB)
	f.sql(`INSERT INTO node_incarnations(id,node_id,public_key_fingerprint,deployment_id,peer_id,os,os_version,arch) VALUES($1,$2,$3,$4,'TEST ONLY node B peer','ubuntu','24.04','amd64')`, incB, nodeB, fpB, f.nodes.NetBirdDeployment)
	f.sql(`UPDATE nodes SET current_incarnation_id=$2,resource_version=resource_version+1 WHERE id=$1`, nodeB, incB)
	f.sql(`INSERT INTO exits(id,node_id,display_name,region,publication_intent) VALUES($1,$2,'TEST ONLY exit B','TEST ONLY region B','published')`, exitB, nodeB)
	f.sql(`INSERT INTO exit_groups(exit_id,group_id) VALUES($1,$2)`, exitB, groupB)

	app, err := Open(ctx, Dependencies{Config: f.config, Pool: f.pool}, f.memberDomain(), nodes())
	if err != nil {
		t.Fatal(err)
	}

	// Assign the member to both groupA and groupB via the real members.Service.Patch API.
	var ver decstr.U64
	if err := f.pool.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id=$1`, f.member).Scan(&ver); err != nil {
		t.Fatal(err)
	}
	bothGroups := []uuid.UUID{groupA, groupB}
	if _, err := f.services.HTTP.Members.Patch(ctx, memberPolicyCommand("admin", uuid.New()), f.member, &ver, api.MemberPatch{MemberGroupIds: &bothGroups}); err != nil {
		t.Fatal(err)
	}

	// Authorize the member's device and compile desired state across both nodes.
	if _, err := f.services.HTTP.Devices.AuthorizeDevice(ctx, memberPolicyCommand("system", uuid.New()), f.peer.peer); err != nil {
		t.Fatal(err)
	}
	// Drain all pending outbox events so both Node A and Node B compile generation >= 1.
	for range 10 {
		worked, err := app.Once(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !worked {
			break
		}
	}

	readNodeBindings := func(nodeID uuid.UUID) (decstr.U64, int, int) {
		t.Helper()
		var gen decstr.U64
		var raw []byte
		if err := f.pool.QueryRow(ctx, `SELECT n.desired_config_generation::text, e.envelope FROM nodes n JOIN desired_state_envelopes e ON e.node_id=n.id WHERE n.id=$1`, nodeID).Scan(&gen, &raw); err != nil {
			t.Fatal(err)
		}
		var env api.DesiredStateEnvelope
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Fatal(err)
		}
		return gen, len(env.Signed.DeviceBindings), len(env.Signed.MemberRatePolicies)
	}

	genA1, bindingsA1, policiesA1 := readNodeBindings(f.node)
	genB1, bindingsB1, policiesB1 := readNodeBindings(nodeB)
	if bindingsA1 != 1 || bindingsB1 != 1 || policiesA1 != 1 || policiesB1 != 1 {
		t.Fatalf("initial compile: nodeA=(gen=%d,bindings=%d,policies=%d) nodeB=(gen=%d,bindings=%d,policies=%d); want 1 binding and 1 policy on both",
			genA1, bindingsA1, policiesA1, genB1, bindingsB1, policiesB1)
	}

	// Now remove groupB from the member via PATCH (leaving only groupA) and drain worker events.
	if err := f.pool.QueryRow(ctx, `SELECT resource_version::text FROM members WHERE id=$1`, f.member).Scan(&ver); err != nil {
		t.Fatal(err)
	}
	onlyGroupA := []uuid.UUID{groupA}
	if _, err := f.services.HTTP.Members.Patch(ctx, memberPolicyCommand("admin", uuid.New()), f.member, &ver, api.MemberPatch{MemberGroupIds: &onlyGroupA}); err != nil {
		t.Fatal(err)
	}
	t.Run("event includes previous and new groups", func(t *testing.T) {
		var raw []byte
		if err := f.pool.QueryRow(ctx, `SELECT payload->'exit_group_ids' FROM outbox_events WHERE aggregate_id=$1 AND event_type='member.entitlement_changed' ORDER BY created_at DESC,id DESC LIMIT 1`, f.member).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		var groups []uuid.UUID
		if err := json.Unmarshal(raw, &groups); err != nil {
			t.Fatal(err)
		}
		if len(groups) != 2 || !slices.Contains(groups, groupA) || !slices.Contains(groups, groupB) {
			t.Fatal("event lost previous or current group", groups)
		}
		if err := f.pool.QueryRow(ctx, `SELECT projection->'exit_group_ids' FROM entitlements WHERE member_id=$1`, f.member).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &groups); err != nil {
			t.Fatal(err)
		}
		if len(groups) != 1 || groups[0] != groupA {
			t.Fatal("event union contaminated current entitlement", groups)
		}
	})
	for range 10 {
		worked, err := app.Once(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !worked {
			break
		}
	}

	t.Run("compiler withdraws previous group", func(t *testing.T) {
		genB2, bindingsB2, policiesB2 := readNodeBindings(nodeB)
		if bindingsB2 != 0 || policiesB2 != 0 {
			t.Errorf("[F-02]: after removing Group B from member %s, Node B (serving Group B) was not recompiled (gen %d -> %d) and still authorizes %d DeviceBindings and %d MemberRatePolicies",
				f.member, genB1, genB2, bindingsB2, policiesB2)
		}
	})
}

// TestReview_F01_CompilerEmitsExcessDeviceBindingsAfterPlanDowngrade proves that
// desiredstate.Compiler (via DBEntitlementReader.ListEntitledMembersForGroups)
// emits all 3 DeviceBindings into the node's signed DesiredStateEnvelope even after
// the member's entitlement is downgraded to device_limit=1.
func TestReview_F01_CompilerEmitsExcessDeviceBindingsAfterPlanDowngrade(t *testing.T) {
	f := newMemberPolicyFixture(t, true)
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Second)

	var groupID, planID, grant3ID uuid.UUID
	if err := f.pool.QueryRow(ctx, `SELECT id FROM member_groups LIMIT 1`).Scan(&groupID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT id FROM plans LIMIT 1`).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT id FROM entitlement_grants WHERE member_id=$1 LIMIT 1`, f.member).Scan(&grant3ID); err != nil {
		t.Fatal(err)
	}
	rev1ID := uuid.New()
	f.sql(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,2,1,$3,1,'{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`,
		rev1ID, planID, []uuid.UUID{groupID})

	// Admit 3 devices on the initial 3-slot grant.
	mp := &multiPeerFake{peers: map[string]devices.Peer{}}
	f.services.HTTP.Devices.Adapter = mp
	for i, ip := range []string{"100.64.0.111", "100.64.0.112", "100.64.0.113"} {
		peer := devices.Peer{
			DeploymentID:   f.nodes.NetBirdDeployment,
			ID:             "peer-f01-" + string(rune('a'+i)),
			ExternalUserID: f.peer.peer.ExternalUserID,
			OverlayScope:   "TEST ONLY",
			OverlayIP:      netip.MustParseAddr(ip),
			Platform:       "linux",
		}
		mp.peers[peer.ID] = peer
		if _, err := f.services.HTTP.Devices.AuthorizeDevice(ctx, memberPolicyCommand("system", uuid.New()), peer); err != nil {
			t.Fatal(err)
		}
	}

	// Downgrade f.member from 3 slots to 1 slot by revoking grant3ID and issuing a 1-slot grant.
	if err := f.services.HTTP.Members.RevokeGrant(ctx, memberPolicyCommand("admin", uuid.New()), f.member, grant3ID, "TEST ONLY downgrade"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.services.HTTP.Members.Grant(ctx, memberPolicyCommand("admin", uuid.New()), f.member, api.GrantCreateRequest{
		Source:         api.GrantSourceAdminGrant,
		PlanRevisionId: &rev1ID,
		ValidFrom:      &now,
		ValidUntil:     now.Add(2 * time.Hour),
		Reason:         "TEST ONLY 1-slot",
	}); err != nil {
		t.Fatal(err)
	}

	app, err := Open(ctx, Dependencies{Config: f.config, Pool: f.pool}, f.memberDomain(), nodes())
	if err != nil {
		t.Fatal(err)
	}
	for range 20 {
		worked, err := app.Once(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !worked {
			break
		}
	}

	var raw []byte
	if err := f.pool.QueryRow(ctx, `SELECT envelope FROM desired_state_envelopes WHERE node_id=$1`, f.node).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var env api.DesiredStateEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	memberBindings := 0
	for _, b := range env.Signed.DeviceBindings {
		if b.MemberId == f.member {
			memberBindings++
		}
	}
	if memberBindings > 1 {
		t.Errorf("[F-01]: DesiredStateEnvelope on node %s contains %d DeviceBindings for downgraded member %s; want <= 1 (plan device_limit=1)",
			f.node, memberBindings, f.member)
	}
}
