//go:build integration

package worker

import (
	"strings"
	"testing"

	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
)

// Regression proof: the worker freezes the incarnation on the first attempt when
// management commits the one-off use between the setup-key GET and the
// correlation-group GET. A consistent re-read would bind TEST-peer.
func TestAT06_D055_Review_StaleKeyUseFreezesLiveEnrollment(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.fake.mu.Lock()
	f.fake.injectPeerOnGroupRead = "TEST-peer"
	f.fake.mu.Unlock()
	f.dispatch(t, "node.peer.discover")
	frozen, peer, reason := reviewDiscoveryState(t, f)
	f.fake.mu.Lock()
	used := f.fake.key["used_times"]
	f.fake.mu.Unlock()
	if used != 1 && used != float64(1) {
		t.Fatal("review harness did not inject the enrollment")
	}
	if frozen {
		t.Fatalf("stale used_times snapshot froze a live enrollment: %s", reason)
	}
	if peer == nil || *peer != "TEST-peer" {
		f.dispatch(t, "node.peer.discover")
		frozen, peer, reason = reviewDiscoveryState(t, f)
	}
	if frozen || peer == nil || *peer != "TEST-peer" {
		t.Fatalf("enrollment that committed between reads was not discovered: frozen=%v peer=%v reason=%s", frozen, peer, reason)
	}
}

// Regression proof: a second group member and a second key use that appear on the
// peer GET, after the single-member check, still commit that first peer.
func TestAT06_D055_Review_PeerJoinsAfterSnapshotBindsFirst(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.fake.useKey("TEST-peer")
	joined := false
	f.fake.beforePeerRead = func() {
		joined = true
		for _, g := range f.fake.groups {
			name, _ := g["name"].(string)
			if strings.HasPrefix(name, "fvpn-enroll-peer-") {
				g["peers"] = []any{"TEST-peer", "TEST-second-peer"}
			}
		}
		f.fake.key["used_times"] = 2
		f.fake.key["state"] = "overused"
	}
	f.dispatch(t, "node.peer.discover")
	if !joined {
		t.Fatal("review harness did not observe the peer read")
	}
	_, peer, _ := reviewDiscoveryState(t, f)
	_, bound, err := (nodelib.NetBirdBindings{Pool: f.r.Pool}).Get(t.Context(), f.r.Config.NetBirdDeployment, "peers", "node:"+f.report.NodeId.String()+":"+f.report.NodeIncarnationId.String())
	if err != nil {
		t.Fatal(err)
	}
	got := ""
	if peer != nil {
		got = *peer
	}
	if bound || got != "" {
		t.Fatalf("bound the first peer after another member appeared: peer=%s bound=%v", got, bound)
	}
}

// Regression proof: revoke before recordPeer deletes the correlation group, leaves
// the management peer, and node.identity_withdrawn still lists the incarnation.
func TestAT07_D055_Review_RevokeBeforeDiscoveryLeavesPeer(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.fake.useKey("TEST-peer")
	f.revoke(t)
	f.dispatch(t, "node.peer.discover")
	_, peer, _ := reviewDiscoveryState(t, f)
	if peer != nil && *peer != "" {
		t.Fatal("revoked discovery committed a peer", *peer)
	}
	f.dispatch(t, "node.publication_changed")
	var after, cleanup string
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT COALESCE((SELECT after_diff::text FROM audit_events WHERE action='node.identity_withdrawn' ORDER BY occurred_at DESC LIMIT 1), ''), COALESCE((SELECT steps->'netbird_identity_cleanup'->>'status' FROM operations WHERE kind IN ('node.revoke','node.retire') ORDER BY created_at DESC LIMIT 1), '')`).Scan(&after, &cleanup); err != nil {
		t.Fatal(err)
	}
	f.fake.mu.Lock()
	deleted := f.fake.peerDeleted
	var names []string
	var sharedPeers any
	for _, g := range f.fake.groups {
		names = append(names, g["name"].(string))
		if g["name"] == "fvpn-unpublished-nodes" {
			sharedPeers = g["peers"]
		}
	}
	f.fake.mu.Unlock()
	listed := strings.Contains(after, f.report.NodeIncarnationId.String())
	if !deleted || !listed || cleanup != "succeeded" {
		t.Fatalf("withdrawal left the peer: deleted=%v incarnation_listed=%v cleanup=%s groups=%v shared_peers=%v", deleted, listed, cleanup, names, sharedPeers)
	}
}

func TestAT07_D055_Review_RevokeAfterCentralCommitDeletesPeer(t *testing.T) {
	f := newDiscoveryWorker(t)
	f.fake.useKey("TEST-peer")
	f.dispatch(t, "node.peer.discover")
	_, peer, _ := reviewDiscoveryState(t, f)
	if peer == nil || *peer != "TEST-peer" {
		t.Fatal("discovery did not commit", peer)
	}
	f.revoke(t)
	f.dispatch(t, "node.publication_changed")
	f.fake.mu.Lock()
	deleted := f.fake.peerDeleted
	f.fake.mu.Unlock()
	if !deleted {
		t.Fatal("peer bound in central state was not deleted")
	}
}

func reviewDiscoveryState(t *testing.T, f *discoveryWorker) (frozen bool, peer *string, reason string) {
	t.Helper()
	err := f.r.Pool.QueryRow(t.Context(), `SELECT COALESCE((SELECT mapping_frozen FROM node_identity_endpoints WHERE incarnation_id=$1), false), (SELECT peer_id FROM node_incarnations WHERE id=$1), COALESCE((SELECT error_message FROM operations WHERE id=$2), '')`, f.report.NodeIncarnationId, f.enrolled.OperationID).Scan(&frozen, &peer, &reason)
	if err != nil {
		t.Fatal(err)
	}
	return frozen, peer, reason
}
