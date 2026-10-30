package netbird

import (
	"net/http"
	"testing"
)

func TestAT06_D055_DiscoveryRevalidatesCleanup(t *testing.T) {
	for _, scenario := range []string{"no_use", "resources_count", "resource_member", "late_second_use", "late_second_peer"} {
		t.Run(scenario, func(t *testing.T) {
			f, c, key, _, group := discoveryFixture(t)
			reviewUse(f, key, group)
			switch scenario {
			case "no_use":
				f.objects["setup-keys"][key.ID]["used_times"] = float64(0)
			case "resources_count":
				f.objects["groups"][group]["resources_count"] = float64(1)
			case "resource_member":
				f.objects["groups"][group]["resources"] = []any{map[string]any{"id": "TEST-resource"}}
			default:
				f.beforeGet = func(kind, id string, _ http.ResponseWriter) bool {
					if kind == "policies" && id == "" && reviewBound(t, c) {
						f.beforeGet = nil
						if scenario == "late_second_use" {
							f.objects["setup-keys"][key.ID]["used_times"] = float64(2)
						} else {
							f.objects["groups"][group]["peers"] = []any{"TEST-node", "TEST-device"}
						}
					}
					return false
				}
			}
			_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
			if err == nil || reviewBound(t, c) || f.objects["groups"][group] == nil {
				t.Fatal("unsafe discovery", err)
			}
			if scenario == "no_use" && !IsKind(err, Unconfirmed) {
				t.Fatal("unused snapshot must retry", err)
			}
		})
	}
}

func TestAT07_D055_WithdrawalRetainsAmbiguousCorrelation(t *testing.T) {
	for _, scenario := range []string{"two_peers", "user_peer", "ephemeral_peer", "unconfirmed_delete"} {
		t.Run(scenario, func(t *testing.T) {
			f, c, key, _, group := discoveryFixture(t)
			reviewUse(f, key, group)
			switch scenario {
			case "two_peers":
				f.objects["groups"][group]["peers"] = []any{"TEST-node", "TEST-device"}
			case "user_peer":
				f.objects["peers"]["TEST-node"]["user_id"] = "TEST-user"
			case "ephemeral_peer":
				f.objects["peers"]["TEST-node"]["ephemeral"] = true
			case "unconfirmed_delete":
				f.beforeGet = func(kind, id string, w http.ResponseWriter) bool {
					if kind == "peers" && id == "TEST-node" && f.objects["peers"][id] == nil {
						w.WriteHeader(503)
						return true
					}
					return false
				}
			}
			err := c.RevokeNodeSetupKey(t.Context(), "TEST-enrollment")
			if err == nil || f.objects["groups"][group] == nil {
				t.Fatal("ambiguous withdrawal discarded correlation", err)
			}
		})
	}
}

func TestAT07_D055_DiscoveryReceiptSurvivesLostCentralCommit(t *testing.T) {
	f, c, key, _, group := discoveryFixture(t)
	reviewUse(f, key, group)
	_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
	require(t, err)
	// The caller never committed peer_id; the durable withdrawal receipt is
	// independent of the node mapping and survives removal of that mapping.
	require(t, c.store.Delete(t.Context(), c.deployment, "peers", "node:TEST-node-id:TEST-incarnation"))
	require(t, c.RevokeNodeSetupKey(t.Context(), "TEST-enrollment"))
	if f.objects["peers"]["TEST-node"] != nil {
		t.Fatal("orphaned peer")
	}
	require(t, c.RevokeNodeSetupKey(t.Context(), "TEST-enrollment"))
}

func TestAT08_NoDistributionGroupsRejectedBeforeManagement(t *testing.T) {
	f, c := newFake(t)
	exit := testExit("")
	exit.PaidGroupIDs = nil
	_, err := c.PublishExit(t.Context(), exit)
	if !IsKind(err, Invalid) || f.writes != 0 {
		t.Fatal("empty distribution reached management", err, f.writes)
	}
}

func TestAT06_D055_ConflictOnCleanupRedeliveryRemovesPendingBinding(t *testing.T) {
	f, c, key, _, group := discoveryFixture(t)
	reviewUse(f, key, group)
	lists := 0
	f.beforeGet = func(kind, id string, w http.ResponseWriter) bool {
		if kind == "policies" && id == "" {
			lists++
			if lists == 2 {
				w.WriteHeader(503)
				return true
			}
		}
		return false
	}
	if _, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode()); !IsKind(err, Upstream) || !reviewBound(t, c) {
		t.Fatal("did not interrupt cleanup after binding", err)
	}
	f.beforeGet = nil
	f.objects["policies"]["TEST-late"] = decodeObject(object{"rules": []object{{"sources": []string{group}}}})
	if _, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode()); !IsKind(err, Conflict) || reviewBound(t, c) || f.objects["groups"][group] == nil {
		t.Fatal("terminal redelivery retained tentative identity", err)
	}
	delete(f.objects["policies"], "TEST-late")
	require(t, c.RevokeNodeSetupKey(t.Context(), "TEST-enrollment"))
	if f.objects["peers"]["TEST-node"] != nil {
		t.Fatal("rollback lost the withdrawal target")
	}
}

func TestAT07_D055_RegistrationCompletingDuringRevocationIsNotOrphaned(t *testing.T) {
	f, c, key, _, group := discoveryFixture(t)
	f.beforeGet = func(kind, id string, _ http.ResponseWriter) bool {
		if kind == "setup-keys" && id == key.ID && f.objects[kind][id]["revoked"] == true && len(ids(f.objects[kind][id]["auto_groups"])) == 1 {
			f.beforeGet = nil
			reviewUse(f, key, group)
		}
		return false
	}
	if err := c.RevokeNodeSetupKey(t.Context(), "TEST-enrollment"); !IsKind(err, Unconfirmed) || f.objects["groups"][group] == nil {
		t.Fatal("newly committed correlation discarded", err)
	}
	require(t, c.RevokeNodeSetupKey(t.Context(), "TEST-enrollment"))
	if f.objects["peers"]["TEST-node"] != nil || f.objects["groups"][group] != nil {
		t.Fatal("retry did not confirm withdrawal")
	}
}
