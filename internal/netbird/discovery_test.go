package netbird

import (
	"context"
	"errors"
	"testing"
)

func discoveryFixture(t *testing.T) (*fake, *Client, SetupKey, string, string) {
	t.Helper()
	f, c := newFake(t)
	key, err := c.CreateNodeSetupKey(t.Context(), "TEST-enrollment")
	require(t, err)
	groups := ids(f.objects["setup-keys"][key.ID]["auto_groups"])
	if len(groups) != 2 {
		t.Fatal("setup key must have exactly two groups", groups)
	}
	var shared, correlation string
	for _, id := range groups {
		switch stringField(f.objects["groups"][id], "name") {
		case "fvpn-unpublished-nodes":
			shared = id
		case stableName("enroll-peer", "TEST-enrollment"):
			correlation = id
		}
		// Tests mutate membership directly; omit optional response counts unless
		// a case explicitly supplies them (including the adversarial mismatch).
		delete(f.objects["groups"][id], "peers_count")
		delete(f.objects["groups"][id], "resources_count")
		if len(ids(f.objects["groups"][id]["peers"])) != 0 {
			t.Fatal("new enrollment group was not empty")
		}
	}
	if shared == "" || correlation == "" {
		t.Fatal("missing required group")
	}
	return f, c, key, shared, correlation
}

func TestAT06_DiscoverNodePeerManagementCorrelation(t *testing.T) {
	f, c, key, shared, correlation := discoveryFixture(t)
	ctx := t.Context()
	node := NodePeer{NodeID: "TEST-node-id", IncarnationID: "TEST-incarnation"}
	_, err := c.DiscoverNodePeer(ctx, "TEST-enrollment", node)
	if !IsKind(err, NotFound) {
		t.Fatal("unused key", err)
	}
	f.objects["setup-keys"][key.ID]["used_times"] = float64(1)
	f.objects["setup-keys"][key.ID]["state"] = "overused"
	for _, id := range []string{shared, correlation} {
		f.objects["groups"][id]["peers"] = []any{"TEST-node"}
	}
	peer, err := c.DiscoverNodePeer(ctx, "TEST-enrollment", node)
	require(t, err)
	if peer.ID != "TEST-node" || f.objects["groups"][correlation] != nil || len(ids(f.objects["groups"][shared]["peers"])) != 1 {
		t.Fatal("discovery/cleanup", peer)
	}
	bound, exists, err := c.store.Get(ctx, c.deployment, "peers", "node:TEST-node-id:TEST-incarnation")
	require(t, err)
	if !exists || bound.ExternalID != peer.ID || bound.LastObservedAt.IsZero() {
		t.Fatal("missing read-back binding", bound)
	}
	writes := f.writes
	again, err := c.DiscoverNodePeer(ctx, "TEST-enrollment", node)
	require(t, err)
	if again != peer || f.writes != writes {
		t.Fatal("discovery replay mutated management")
	}
}

func TestAT06_DiscoveryConflictNeverBinds(t *testing.T) {
	for _, scenario := range []string{"two_peers", "reused_key", "user_peer", "ephemeral_peer", "policy_group", "route_group", "revoked", "changed_groups", "duplicate_groups"} {
		t.Run(scenario, func(t *testing.T) {
			f, c, key, _, correlation := discoveryFixture(t)
			f.objects["setup-keys"][key.ID]["used_times"] = float64(1)
			f.objects["groups"][correlation]["peers"] = []any{"TEST-node"}
			switch scenario {
			case "two_peers":
				f.objects["groups"][correlation]["peers"] = []any{"TEST-node", "TEST-device"}
			case "reused_key":
				f.objects["setup-keys"][key.ID]["used_times"] = float64(2)
			case "user_peer":
				f.objects["peers"]["TEST-node"]["user_id"] = "TEST-user"
			case "ephemeral_peer":
				f.objects["peers"]["TEST-node"]["ephemeral"] = true
			case "policy_group":
				f.objects["policies"]["TEST-policy"] = decodeObject(object{"rules": []object{{"sources": []string{correlation}}}})
			case "route_group":
				f.objects["routes"]["TEST-route"] = decodeObject(object{"peer_groups": []string{correlation}})
			case "revoked":
				f.objects["setup-keys"][key.ID]["revoked"] = true
			case "changed_groups":
				f.objects["setup-keys"][key.ID]["auto_groups"] = []any{correlation}
			case "duplicate_groups":
				f.objects["setup-keys"][key.ID]["auto_groups"] = append(f.objects["setup-keys"][key.ID]["auto_groups"].([]any), correlation)
			}
			_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", NodePeer{NodeID: "TEST-node-id", IncarnationID: "TEST-incarnation"})
			if !IsKind(err, Conflict) || DiscoveryReason(err) == "management_peer_unconfirmed" {
				t.Fatal("expected classified conflict", err)
			}
			_, exists, err := c.store.Get(t.Context(), c.deployment, "peers", "node:TEST-node-id:TEST-incarnation")
			require(t, err)
			if exists || f.objects["groups"][correlation] == nil {
				t.Fatal("unsafe correlation was bound or discarded")
			}
		})
	}
}

func TestAT06_DiscoveryCleanupRetryAndRevoke(t *testing.T) {
	for _, revoke := range []bool{false, true} {
		t.Run(map[bool]string{false: "discover", true: "revoke"}[revoke], func(t *testing.T) {
			f, c, key, shared, correlation := discoveryFixture(t)
			f.objects["setup-keys"][key.ID]["used_times"] = float64(1)
			f.objects["groups"][correlation]["peers"] = []any{"TEST-node"}
			f.lose = "setup-keys"
			run := func() error {
				if revoke {
					return c.RevokeNodeSetupKey(t.Context(), "TEST-enrollment")
				}
				_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", NodePeer{NodeID: "TEST-node-id", IncarnationID: "TEST-incarnation"})
				return err
			}
			if run() == nil {
				t.Fatal("lost mutation response was acknowledged")
			}
			require(t, run())
			require(t, run())
			if f.objects["groups"][correlation] != nil || f.objects["groups"][shared] == nil || (revoke && f.objects["setup-keys"][key.ID]["revoked"] != true) {
				t.Fatal("correlation group cleanup incomplete")
			}
		})
	}
}

func TestAT06_DiscoveryFenceRejectsBeforeBinding(t *testing.T) {
	f, c, key, _, correlation := discoveryFixture(t)
	f.objects["setup-keys"][key.ID]["used_times"] = float64(1)
	f.objects["groups"][correlation]["peers"] = []any{"TEST-node"}
	lost := errors.New("TEST ONLY lost claim")
	checks := 0
	ctx := WithDiscoveryFence(t.Context(), func(context.Context) error {
		checks++
		if checks == 2 {
			return lost
		}
		return nil
	})
	_, err := c.DiscoverNodePeer(ctx, "TEST-enrollment", NodePeer{NodeID: "TEST-node-id", IncarnationID: "TEST-incarnation"})
	if !errors.Is(err, lost) || checks != 2 {
		t.Fatal(err, checks)
	}
	_, exists, err := c.store.Get(t.Context(), c.deployment, "peers", "node:TEST-node-id:TEST-incarnation")
	require(t, err)
	if exists {
		t.Fatal("stale claim recorded a binding")
	}
}
