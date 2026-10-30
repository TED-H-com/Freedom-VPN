package netbird

import (
	"net/http"
	"strings"
	"testing"
)

func reviewNode() NodePeer {
	return NodePeer{NodeID: "TEST-node-id", IncarnationID: "TEST-incarnation"}
}

func reviewBound(t *testing.T, c *Client) bool {
	t.Helper()
	_, exists, err := c.store.Get(t.Context(), c.deployment, "peers", "node:TEST-node-id:TEST-incarnation")
	require(t, err)
	return exists
}

func reviewUse(f *fake, key SetupKey, correlation string) {
	f.objects["setup-keys"][key.ID]["used_times"] = float64(1)
	f.objects["setup-keys"][key.ID]["state"] = "overused"
	f.objects["groups"][correlation]["peers"] = []any{"TEST-node"}
}

// TODO(review): membership and used_times are snapshotted on separate GETs.
// A peer that joins, and the key that moves 0 → 1, between those reads is a
// terminal group_peer_without_key_use conflict. The safe result is a re-read
// that binds this single use, or a retryable NotFound/Unconfirmed.
func TestAT06_D055_Review_StaleKeyUseIsNotTerminal(t *testing.T) {
	f, c, key, _, correlation := discoveryFixture(t)
	f.beforeGet = func(kind, id string, _ http.ResponseWriter) bool {
		if kind == "groups" && id == correlation {
			f.beforeGet = nil
			f.objects["groups"][correlation]["peers"] = []any{"TEST-node"}
			f.objects["setup-keys"][key.ID]["used_times"] = float64(1)
			f.objects["setup-keys"][key.ID]["state"] = "overused"
		}
		return false
	}
	_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
	if IsKind(err, Conflict) {
		t.Fatalf("live enrollment frozen by a stale used_times snapshot: %s", DiscoveryReason(err))
	}
	if err == nil || IsKind(err, NotFound) || IsKind(err, Unconfirmed) {
		return
	}
	t.Fatal(err)
}

// TODO(review): the single-member decision is not repeated after GET /peers/{id}.
// A second member, or used_times moving 1 → 2, in that window still binds the
// first peer and deletes the group.
func TestAT06_D055_Review_PeerJoiningAfterSnapshotDoesNotBind(t *testing.T) {
	f, c, key, _, correlation := discoveryFixture(t)
	reviewUse(f, key, correlation)
	f.beforeGet = func(kind, id string, _ http.ResponseWriter) bool {
		if kind == "peers" && id == "TEST-node" {
			f.beforeGet = nil
			f.objects["groups"][correlation]["peers"] = []any{"TEST-node", "TEST-device"}
			f.objects["setup-keys"][key.ID]["used_times"] = float64(2)
		}
		return false
	}
	_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
	bound := reviewBound(t, c)
	if IsKind(err, Conflict) && !bound && f.objects["groups"][correlation] != nil {
		return
	}
	t.Fatalf("first peer bound after the group changed: err=%v bound=%v group_remains=%v", err, bound, f.objects["groups"][correlation] != nil)
}

// TODO(review): PolicyRule.authorized_groups is a map whose keys are group IDs.
// ids() only accepts arrays, so a policy that names the correlation group only
// there does not conflict. Discovery binds and deletes the group.
func TestAT06_D055_Review_AuthorizedGroupsMapIsIsolation(t *testing.T) {
	f, c, key, _, correlation := discoveryFixture(t)
	reviewUse(f, key, correlation)
	f.objects["policies"]["TEST-policy"] = object{
		"id": "TEST-policy",
		"rules": []any{map[string]any{
			"authorized_groups": map[string]any{correlation: []any{"root"}},
		}},
	}
	_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
	bound := reviewBound(t, c)
	if IsKind(err, Conflict) && DiscoveryReason(err) == "correlation_group_referenced_by_policies" && !bound && f.objects["groups"][correlation] != nil {
		return
	}
	t.Fatalf("authorized_groups map was ignored: err=%v bound=%v group_remains=%v", err, bound, f.objects["groups"][correlation] != nil)
}

// TODO(review): isolation lists policies and routes only. A nameserver whose
// groups array contains the correlation group does not conflict.
func TestAT06_D055_Review_NameserverReferenceBlocksDiscovery(t *testing.T) {
	f, c, key, _, correlation := discoveryFixture(t)
	reviewUse(f, key, correlation)
	f.objects["dns/nameservers"]["TEST-ns"] = object{
		"id":     "TEST-ns",
		"name":   "TEST ONLY nameserver",
		"groups": []any{correlation},
	}
	_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
	bound := reviewBound(t, c)
	if IsKind(err, Conflict) && !bound && f.objects["groups"][correlation] != nil {
		return
	}
	t.Fatalf("nameserver group reference was ignored: err=%v bound=%v group_remains=%v", err, bound, f.objects["groups"][correlation] != nil)
}

// TODO(review): a key issued with only the shared staging group never matches
// by name, address, or time, but the miss is NotFound. The worker then polls
// until the retry budget and reports management_peer_unconfirmed. The operator
// message has to say to revoke this attempt and re-enroll.
func TestAT06_D055_Review_LegacyOneGroupKeyAsksForReenroll(t *testing.T) {
	f, c, key, shared, correlation := discoveryFixture(t)
	delete(f.objects["groups"], correlation)
	f.objects["setup-keys"][key.ID]["auto_groups"] = []any{shared}
	f.objects["setup-keys"][key.ID]["used_times"] = float64(1)
	f.objects["setup-keys"][key.ID]["state"] = "overused"
	f.objects["groups"][shared]["peers"] = []any{"TEST-node"}
	_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
	if reviewBound(t, c) {
		t.Fatal("legacy key was correlated onto TEST-node")
	}
	reason := strings.ToLower(DiscoveryReason(err))
	if IsKind(err, Conflict) && strings.Contains(reason, "revoke") && (strings.Contains(reason, "re-enroll") || strings.Contains(reason, "re_enroll") || strings.Contains(reason, "reenroll")) {
		return
	}
	t.Fatalf("legacy key must be a terminal revoke-and-re-enroll conflict, got %v (%s)", err, DiscoveryReason(err))
}

// TODO(review): peers_count is not compared with the peers array. One visible
// peer and peers_count 2 still binds candidate zero.
func TestAT06_D055_Review_PeerCountMismatchDoesNotBind(t *testing.T) {
	f, c, key, _, correlation := discoveryFixture(t)
	reviewUse(f, key, correlation)
	f.objects["groups"][correlation]["peers_count"] = float64(2)
	_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
	bound := reviewBound(t, c)
	if (IsKind(err, Unconfirmed) || IsKind(err, Conflict)) && !bound {
		return
	}
	t.Fatalf("peers_count disagreement bound a peer: err=%v bound=%v", err, bound)
}

// TODO(review): recordPeer runs before cleanup. A policy that appears on the
// second isolation list returns Conflict and leaves the new adapter binding.
// The worker treats that Conflict as terminal and does not retry cleanup.
func TestAT06_D055_Review_ConflictAfterBindRollsBack(t *testing.T) {
	f, c, key, _, correlation := discoveryFixture(t)
	reviewUse(f, key, correlation)
	lists := 0
	f.beforeGet = func(kind, id string, _ http.ResponseWriter) bool {
		if kind != "policies" || id != "" {
			return false
		}
		lists++
		if lists == 2 {
			f.objects["policies"]["TEST-late-policy"] = object{
				"id":    "TEST-late-policy",
				"rules": []any{map[string]any{"sources": []any{correlation}}},
			}
		}
		return false
	}
	_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
	bound := reviewBound(t, c)
	if IsKind(err, Conflict) && !bound && f.objects["groups"][correlation] != nil {
		return
	}
	t.Fatalf("binding survived a cleanup conflict: err=%v bound=%v group_remains=%v policy_lists=%d", err, bound, f.objects["groups"][correlation] != nil, lists)
}

// TODO(review): RevokeNodeSetupKey deletes the correlation group and does not
// delete its peer. DeletePeer then no-ops because no node binding exists, so
// the management peer stays in the shared staging group.
func TestAT07_D055_Review_RevokeBeforeBindWithdrawsPeer(t *testing.T) {
	f, c, key, shared, correlation := discoveryFixture(t)
	reviewUse(f, key, correlation)
	f.objects["groups"][shared]["peers"] = []any{"TEST-node"}
	revokeErr := c.RevokeNodeSetupKey(t.Context(), "TEST-enrollment")
	deleteErr := c.DeletePeer(t.Context(), "node:TEST-node-id:TEST-incarnation")
	peerRemains := f.objects["peers"]["TEST-node"] != nil
	if revokeErr != nil || deleteErr != nil || peerRemains {
		t.Fatalf("enrolled peer was not withdrawn: revoke=%v delete=%v peer_remains=%v group_gone=%v key_revoked=%v shared_peers=%v",
			revokeErr, deleteErr, peerRemains, f.objects["groups"][correlation] == nil, f.objects["setup-keys"][key.ID]["revoked"], f.objects["groups"][shared]["peers"])
	}
}

func TestAT06_D055_Review_MalformedReadsDoNotBind(t *testing.T) {
	t.Run("group_page_wrapper", func(t *testing.T) {
		f, c, key, _, correlation := discoveryFixture(t)
		reviewUse(f, key, correlation)
		f.beforeGet = func(kind, id string, w http.ResponseWriter) bool {
			if kind == "groups" && id == "" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"items":[{"id":"` + correlation + `","name":"fvpn-enroll-peer","peers":["TEST-node"]}],"next":"TEST-page"}`))
				return true
			}
			return false
		}
		_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
		if err == nil || reviewBound(t, c) || IsKind(err, Conflict) {
			t.Fatal("page wrapper bound or conflicted", err)
		}
	})
	t.Run("peer_5xx", func(t *testing.T) {
		f, c, key, _, correlation := discoveryFixture(t)
		reviewUse(f, key, correlation)
		f.beforeGet = func(kind, id string, w http.ResponseWriter) bool {
			if kind == "peers" && id == "TEST-node" {
				w.WriteHeader(http.StatusInternalServerError)
				return true
			}
			return false
		}
		_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
		if !IsKind(err, Upstream) || reviewBound(t, c) {
			t.Fatal(err)
		}
	})
	t.Run("peer_truncated_json", func(t *testing.T) {
		f, c, key, _, correlation := discoveryFixture(t)
		reviewUse(f, key, correlation)
		f.beforeGet = func(kind, id string, w http.ResponseWriter) bool {
			if kind == "peers" && id == "TEST-node" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"TEST-node","user_id":`))
				return true
			}
			return false
		}
		_, err := c.DiscoverNodePeer(t.Context(), "TEST-enrollment", reviewNode())
		if !IsKind(err, Unconfirmed) || reviewBound(t, c) {
			t.Fatal(err)
		}
	})
}
