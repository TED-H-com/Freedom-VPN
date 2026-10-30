package netbird

import (
	"context"
	"errors"
	"slices"
)

// DiscoveryConflict carries only a fixed, non-secret reason for an unsafe
// management-plane correlation. It never includes upstream response text.
type DiscoveryConflict struct{ Reason string }

func (e *DiscoveryConflict) Error() string { return "netbird discovery: " + e.Reason }
func (e *DiscoveryConflict) Unwrap() error { return failure(Conflict) }

func discoveryConflict(reason string) error { return &DiscoveryConflict{Reason: reason} }

type discoveryFenceKey struct{}

// WithDiscoveryFence checks the caller's durable claim and incarnation after
// acquiring the deployment lock and again before binding. It performs no HTTP.
func WithDiscoveryFence(ctx context.Context, check func(context.Context) error) context.Context {
	return context.WithValue(ctx, discoveryFenceKey{}, check)
}

func checkDiscoveryFence(ctx context.Context) error {
	if check, ok := ctx.Value(discoveryFenceKey{}).(func(context.Context) error); ok {
		return check(ctx)
	}
	return nil
}

func enrollmentGroupKey(id string) string { return "enroll-peer:" + id }

// DiscoverNodePeer correlates a machine peer exclusively through the one-off
// enrollment group's membership and setup-key usage (D-055), never a hostname/IP.
func (c *Client) DiscoverNodePeer(ctx context.Context, enrollmentID string, n NodePeer) (Peer, error) {
	if !valid(enrollmentID, n.NodeID, n.IncarnationID) || n.PeerID != "" {
		return Peer{}, failure(Invalid)
	}
	unlock, err := c.lock(ctx)
	if err != nil {
		return Peer{}, err
	}
	defer unlock()
	if err = checkDiscoveryFence(ctx); err != nil {
		return Peer{}, err
	}
	key, err := c.locate(ctx, "setup-keys", enrollmentID, stableName("enroll", enrollmentID))
	if err != nil {
		return Peer{}, err
	}
	if key == nil {
		return Peer{}, failure(NotFound)
	}
	key, err = c.get(ctx, "setup-keys", stringField(key, "id"))
	if err != nil {
		return Peer{}, err
	}
	if stringField(key, "type") != "one-off" || key["usage_limit"] != float64(1) || key["ephemeral"] != false {
		return Peer{}, discoveryConflict("setup_key_settings_changed")
	}
	used, ok := key["used_times"].(float64)
	revoked, hasRevoked := key["revoked"].(bool)
	state := stringField(key, "state")
	if !ok || !hasRevoked || used < 0 || used != float64(int64(used)) || !slices.Contains([]string{"valid", "overused", "expired", "revoked"}, state) {
		return Peer{}, failure(Unconfirmed)
	}
	if revoked || state == "revoked" {
		return Peer{}, discoveryConflict("setup_key_revoked")
	}
	if used > 1 {
		return Peer{}, discoveryConflict("setup_key_reused")
	}
	// A consumed one-off key may be 'overused' or expire before discovery.
	// Exactly one use is the proof; expiry does not revoke an enrolled peer.
	bindingKey := "node:" + n.NodeID + ":" + n.IncarnationID
	bound, exists, err := c.store.Get(ctx, c.deployment, "peers", bindingKey)
	if err != nil {
		return Peer{}, err
	}
	group, err := c.locate(ctx, "groups", enrollmentGroupKey(enrollmentID), stableName("enroll-peer", enrollmentID))
	if err != nil {
		return Peer{}, err
	}
	var candidates []string
	if group != nil {
		group, err = c.get(ctx, "groups", stringField(group, "id"))
		if err != nil {
			return Peer{}, err
		}
		if err = c.checkEnrollmentGroupIsolation(ctx, stringField(group, "id")); err != nil {
			return Peer{}, err
		}
		candidates = ids(group["peers"])
		if members, ok := group["peers"].([]any); group["peers"] != nil && (!ok || len(members) != len(candidates) || slices.Contains(candidates, "")) {
			return Peer{}, failure(Unconfirmed)
		}
		if len(candidates) > 1 {
			return Peer{}, discoveryConflict("multiple_group_peers")
		}
	}
	if exists {
		n.PeerID = bound.ExternalID
		if len(candidates) == 1 && candidates[0] != n.PeerID {
			return Peer{}, discoveryConflict("group_binding_changed")
		}
	} else {
		if group != nil {
			shared, ok, err := c.store.Get(ctx, c.deployment, "groups", "unpublished-nodes")
			if err != nil {
				return Peer{}, err
			}
			autoGroups, groupsOK := key["auto_groups"].([]any)
			if !ok || !groupsOK || len(autoGroups) != 2 || !slices.Equal(ids(key["auto_groups"]), sorted([]string{shared.ExternalID, stringField(group, "id")})) {
				return Peer{}, discoveryConflict("setup_key_groups_changed")
			}
		}
		if len(candidates) == 0 {
			return Peer{}, failure(NotFound)
		}
		n.PeerID = candidates[0]
	}
	if used != 1 {
		if len(candidates) != 0 {
			return Peer{}, discoveryConflict("group_peer_without_key_use")
		}
		return Peer{}, failure(NotFound)
	}
	o, err := c.get(ctx, "peers", n.PeerID)
	if err != nil {
		return Peer{}, err
	}
	if user, ok := o["user_id"].(string); !ok || user != "" {
		return Peer{}, discoveryConflict("user_owned_peer")
	}
	if ephemeral, ok := o["ephemeral"].(bool); !ok || ephemeral {
		return Peer{}, discoveryConflict("ephemeral_peer")
	}
	if err = checkDiscoveryFence(ctx); err != nil {
		return Peer{}, err
	}
	peer := Peer{ID: n.PeerID, Name: stringField(o, "name"), IP: stringField(o, "ip"), Connected: o["connected"] == true}
	if err = c.recordPeer(ctx, bindingKey, peer, n); err != nil {
		return Peer{}, err
	}
	// Binding precedes cleanup so a crash midway can resume from durable proof.
	if err = c.removeEnrollmentGroup(ctx, enrollmentID, key); err != nil {
		return Peer{}, err
	}
	return peer, nil
}

func (c *Client) checkEnrollmentGroupIsolation(ctx context.Context, group string) error {
	for _, kind := range []string{"policies", "routes"} {
		objects, err := c.list(ctx, kind)
		if err != nil {
			return err
		}
		for _, o := range objects {
			if referencesGroup(o, group) {
				return discoveryConflict("correlation_group_referenced_by_" + kind)
			}
		}
	}
	return nil
}

func referencesGroup(o object, group string) bool {
	for _, key := range []string{"sources", "destinations", "authorized_groups", "groups", "peer_groups", "access_control_groups"} {
		if slices.Contains(ids(o[key]), group) {
			return true
		}
	}
	rules, _ := o["rules"].([]any)
	for _, rule := range rules {
		if r, ok := rule.(map[string]any); ok && referencesGroup(r, group) {
			return true
		}
	}
	return false
}

func (c *Client) removeEnrollmentGroup(ctx context.Context, enrollmentID string, key object) error {
	groupKey := enrollmentGroupKey(enrollmentID)
	g, err := c.locate(ctx, "groups", groupKey, stableName("enroll-peer", enrollmentID))
	if err != nil || g == nil {
		return err
	}
	id := stringField(g, "id")
	if err = c.checkEnrollmentGroupIsolation(ctx, id); err != nil {
		return err
	}
	// Detach the exhausted/revoked key before deleting the group. Preserve the
	// shared unpublished group and every existing revocation flag on read-back.
	if key != nil && slices.Contains(ids(key["auto_groups"]), id) {
		groups := slices.DeleteFunc(ids(key["auto_groups"]), func(v string) bool { return v == id })
		keyID := stringField(key, "id")
		if err = c.call(ctx, "PUT", endpoint("setup-keys", keyID), object{"revoked": key["revoked"] == true, "auto_groups": groups}, &object{}); err != nil {
			return err
		}
		observed, err := c.get(ctx, "setup-keys", keyID)
		if err != nil {
			return err
		}
		if !slices.Equal(ids(observed["auto_groups"]), groups) || (key["revoked"] == true && observed["revoked"] != true) {
			return failure(Unconfirmed)
		}
	}
	if len(ids(g["peers"])) != 0 {
		if _, err = c.group(ctx, groupKey, stableName("enroll-peer", enrollmentID), nil, false); err != nil {
			return err
		}
	}
	if err = c.remove(ctx, "groups", id); err != nil {
		return err
	}
	return c.store.Delete(ctx, c.deployment, "groups", groupKey)
}

// DiscoveryReason returns a safe diagnostic suitable for audit/operation data.
func DiscoveryReason(err error) string {
	var conflict *DiscoveryConflict
	if errors.As(err, &conflict) {
		return conflict.Reason
	}
	if IsKind(err, Conflict) {
		return "management_binding_conflict"
	}
	return "management_peer_unconfirmed"
}
