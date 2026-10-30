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
func (c *Client) DiscoverNodePeer(ctx context.Context, enrollmentID string, n NodePeer) (result Peer, resultErr error) {
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
	bindingKey := "node:" + n.NodeID + ":" + n.IncarnationID
	wroteBinding := false
	defer func() {
		// A prior delivery may have stored its binding before a transient
		// cleanup failure. A conflict on THIS delivery must remove it too,
		// before the worker acknowledges terminal classification (D-055).
		if !IsKind(resultErr, Conflict) && !(wroteBinding && IsKind(resultErr, Unconfirmed)) {
			return
		}
		bound, exists, err := c.store.Get(ctx, c.deployment, "peers", bindingKey)
		if err != nil {
			resultErr = failure(Unconfirmed)
			return
		}
		if !exists {
			return
		}
		if err = c.withdrawalProof(ctx, enrollmentID, bound.ExternalID); err != nil {
			resultErr = failure(Unconfirmed)
			return
		}
		if err = c.store.Delete(ctx, c.deployment, "peers", bindingKey); err != nil {
			resultErr = failure(Unconfirmed)
		}
	}()
	key, err := c.locate(ctx, "setup-keys", enrollmentID, stableName("enroll", enrollmentID))
	if err != nil {
		return Peer{}, err
	}
	if key == nil {
		return Peer{}, failure(NotFound)
	}
	bound, exists, err := c.store.Get(ctx, c.deployment, "peers", bindingKey)
	if err != nil {
		return Peer{}, err
	}
	group, err := c.locate(ctx, "groups", enrollmentGroupKey(enrollmentID), stableName("enroll-peer", enrollmentID))
	if err != nil {
		return Peer{}, err
	}
	if group != nil {
		if err = c.checkEnrollmentGroupIsolation(ctx, stringField(group, "id")); err != nil {
			return Peer{}, err
		}
	}
	candidate := ""
	if exists {
		candidate = bound.ExternalID
	} else {
		proof, ok, err := c.store.Get(ctx, c.deployment, "enrollment-withdrawal", enrollmentID)
		if err != nil {
			return Peer{}, err
		}
		if ok {
			candidate = proof.ExternalID
		}
	}
	key, group, candidate, err = c.discoverySnapshot(ctx, enrollmentID, key, group, candidate, false)
	if err != nil {
		return Peer{}, err
	}
	o, err := c.get(ctx, "peers", candidate)
	if err != nil {
		return Peer{}, err
	}
	if err = machinePeer(o); err != nil {
		return Peer{}, err
	}
	// Management may commit registration between GETs. Re-read BOTH sides after
	// the peer read, and once more inside cleanup; a stale 0-use read is retryable.
	key, group, _, err = c.discoverySnapshot(ctx, enrollmentID, key, group, candidate, false)
	if err != nil {
		return Peer{}, err
	}
	if err = checkDiscoveryFence(ctx); err != nil {
		return Peer{}, err
	}
	n.PeerID = candidate
	peer := Peer{ID: candidate, Name: stringField(o, "name"), IP: stringField(o, "ip"), Connected: o["connected"] == true}
	if err = c.recordPeer(ctx, bindingKey, peer, n); err != nil {
		return Peer{}, err
	}
	wroteBinding = true
	if err = c.removeEnrollmentGroup(ctx, enrollmentID, key, group, candidate, false); err != nil {
		return Peer{}, err
	}
	return peer, nil
}

const reenrollReason = "correlation_unavailable_revoke_and_re_enroll"

func machinePeer(o object) error {
	if user, ok := o["user_id"].(string); !ok || user != "" {
		return discoveryConflict("user_owned_peer")
	}
	if ephemeral, ok := o["ephemeral"].(bool); !ok || ephemeral {
		return discoveryConflict("ephemeral_peer")
	}
	return nil
}

// Counts are optional in older management responses, but when supplied they
// must agree with the complete arrays. Never treat a partial list as ownership.
func groupMembers(g object) ([]string, error) {
	for _, field := range []string{"peers", "resources"} {
		members, ok := g[field].([]any)
		if g[field] != nil && !ok {
			return nil, failure(Unconfirmed)
		}
		if count, present := g[field+"_count"]; present {
			if n, ok := count.(float64); !ok || n != float64(len(members)) {
				return nil, failure(Unconfirmed)
			}
		}
		if field == "resources" && len(members) != 0 {
			return nil, discoveryConflict("correlation_group_has_resources")
		}
	}
	members := ids(g["peers"])
	array, _ := g["peers"].([]any)
	if len(array) != len(members) || slices.Contains(members, "") {
		return nil, failure(Unconfirmed)
	}
	if len(members) > 1 {
		return nil, discoveryConflict("multiple_group_peers")
	}
	return members, nil
}

// discoverySnapshot re-GETs the decision inputs. A cleanup receipt permits only
// our already-confirmed detach/empty-group retry, never a new identity mapping.
func (c *Client) discoverySnapshot(ctx context.Context, enrollmentID string, key, group object, candidate string, revoke bool) (object, object, string, error) {
	key, err := c.get(ctx, "setup-keys", stringField(key, "id"))
	if err != nil {
		return nil, nil, "", err
	}
	if stringField(key, "type") != "one-off" || key["usage_limit"] != float64(1) || key["ephemeral"] != false {
		return nil, nil, "", discoveryConflict("setup_key_settings_changed")
	}
	used, ok := key["used_times"].(float64)
	revoked, hasRevoked := key["revoked"].(bool)
	state := stringField(key, "state")
	if !ok || !hasRevoked || used < 0 || used != float64(int64(used)) || !slices.Contains([]string{"valid", "overused", "expired", "revoked"}, state) {
		return nil, nil, "", failure(Unconfirmed)
	}
	if !revoke && (revoked || state == "revoked") {
		return nil, nil, "", discoveryConflict("setup_key_revoked")
	}
	if revoke && !revoked {
		return nil, nil, "", failure(Unconfirmed)
	}
	if used > 1 {
		return nil, nil, "", discoveryConflict("setup_key_reused")
	}
	if group != nil {
		group, err = c.get(ctx, "groups", stringField(group, "id"))
		if IsKind(err, NotFound) {
			group, err = nil, nil
		}
		if err != nil {
			return nil, nil, "", err
		}
	}
	proof, cleanup, err := c.store.Get(ctx, c.deployment, "enrollment-withdrawal", enrollmentID)
	if err != nil {
		return nil, nil, "", err
	}
	members, err := groupMembers(group)
	if err != nil {
		return nil, nil, "", err
	}
	shared, known, err := c.store.Get(ctx, c.deployment, "groups", "unpublished-nodes")
	if err != nil {
		return nil, nil, "", err
	}
	groups, arrayOK := key["auto_groups"].([]any)
	expected := sorted([]string{shared.ExternalID, stringField(group, "id")})
	exact := group != nil && known && arrayOK && len(groups) == 2 && slices.Equal(ids(groups), expected)
	// After a completed discovery, the node binding or withdrawal proof is the
	// durable correlation. A legacy uncorrelated key must explicitly re-enroll.
	_, retiring, err := c.store.Get(ctx, c.deployment, "enrollment-cleanup", enrollmentID)
	if err != nil {
		return nil, nil, "", err
	}
	partial := (cleanup || revoke && retiring) && known && arrayOK && len(groups) == 1 && slices.Equal(ids(groups), []string{shared.ExternalID})
	if !exact && !partial {
		return nil, nil, "", discoveryConflict(reenrollReason)
	}
	if group == nil && !cleanup && !(revoke && retiring) {
		return nil, nil, "", discoveryConflict(reenrollReason)
	}
	if len(members) == 1 {
		if used == 0 {
			return nil, nil, "", failure(Unconfirmed)
		}
		if candidate != "" && candidate != members[0] {
			return nil, nil, "", discoveryConflict("group_binding_changed")
		}
		candidate = members[0]
	} else if !revoke && (!cleanup || candidate == "") {
		return nil, nil, "", failure(NotFound)
	}
	if cleanup && candidate != "" && proof.ExternalID != candidate {
		return nil, nil, "", discoveryConflict("group_binding_changed")
	}
	if !revoke && used != 1 {
		return nil, nil, "", failure(NotFound)
	}
	return key, group, candidate, nil
}

func (c *Client) checkEnrollmentGroupIsolation(ctx context.Context, group string) error {
	for _, kind := range []string{"policies", "routes", "dns/nameservers"} {
		objects, err := c.list(ctx, kind)
		if err != nil {
			return err
		}
		for _, o := range objects {
			if referencesGroup(o, group) {
				reason := "correlation_group_referenced_by_" + kind
				if kind == "dns/nameservers" {
					reason = "correlation_group_referenced_by_nameservers"
				}
				return discoveryConflict(reason)
			}
		}
	}
	return nil
}

func referencesGroup(o object, group string) bool {
	for _, key := range []string{"sources", "destinations", "authorized_groups", "groups", "peer_groups", "access_control_groups"} {
		if values, ok := o[key].(map[string]any); ok && key == "authorized_groups" {
			if _, hit := values[group]; hit {
				return true
			}
		}
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

// removeEnrollmentGroup retains correlation until peer deletion (revocation) or
// a durable withdrawal target (discovery) has been confirmed. No remote effect
// is considered complete without a GET read-back (§5.2, §14.5).
func (c *Client) removeEnrollmentGroup(ctx context.Context, enrollmentID string, key, group object, candidate string, revoke bool) error {
	if group != nil {
		if err := c.checkEnrollmentGroupIsolation(ctx, stringField(group, "id")); err != nil {
			return err
		}
	}
	key, group, candidate, err := c.discoverySnapshot(ctx, enrollmentID, key, group, candidate, revoke)
	if err != nil {
		return err
	}
	proof, hasProof, err := c.store.Get(ctx, c.deployment, "enrollment-withdrawal", enrollmentID)
	if err != nil {
		return err
	}
	if revoke {
		if candidate == "" && hasProof {
			candidate = proof.ExternalID
		}
		if candidate != "" {
			o, err := c.get(ctx, "peers", candidate)
			if err != nil && !IsKind(err, NotFound) {
				return err
			}
			if err == nil {
				if err = machinePeer(o); err != nil {
					return err
				}
			}
			key, group, _, err = c.discoverySnapshot(ctx, enrollmentID, key, group, candidate, true)
			if err != nil {
				return err
			}
			// Save the deletion target before DELETE so an interrupted response or
			// empty group can be retried, without claiming an identity binding.
			if err = c.withdrawalProof(ctx, enrollmentID, candidate); err != nil {
				return err
			}
			if err = c.deletePeer(ctx, candidate); err != nil {
				return err
			}
		}
	} else if candidate != "" {
		if err = c.withdrawalProof(ctx, enrollmentID, candidate); err != nil {
			return err
		}
	}
	if group == nil {
		return nil
	}
	id := stringField(group, "id")
	if revoke {
		h, err := hash([]string{"enrollment-cleanup", enrollmentID, id})
		if err != nil {
			return err
		}
		if err = c.bind(ctx, "enrollment-cleanup", enrollmentID, id, h); err != nil {
			return err
		}
	}
	if slices.Contains(ids(key["auto_groups"]), id) {
		groups := slices.DeleteFunc(ids(key["auto_groups"]), func(v string) bool { return v == id })
		if err = c.call(ctx, "PUT", endpoint("setup-keys", stringField(key, "id")), object{"revoked": key["revoked"] == true, "auto_groups": groups}, &object{}); err != nil {
			return err
		}
		observed, err := c.get(ctx, "setup-keys", stringField(key, "id"))
		if err != nil {
			return err
		}
		if !slices.Equal(ids(observed["auto_groups"]), groups) || (key["revoked"] == true && observed["revoked"] != true) {
			return failure(Unconfirmed)
		}
		key = observed
	}
	// Recheck immediately before clearing membership; never discard a member
	// that arrived during the preceding HTTP reads/writes.
	var latestCandidate string
	key, group, latestCandidate, err = c.discoverySnapshot(ctx, enrollmentID, key, group, candidate, revoke)
	if err != nil {
		return err
	}
	if latestCandidate != candidate {
		// Registration may finish after revocation of a previously unused key.
		// Retry from the peer-delete decision; never erase this new correlation.
		return failure(Unconfirmed)
	}
	if group == nil {
		return nil
	}
	if err = c.checkEnrollmentGroupIsolation(ctx, id); err != nil {
		return err
	}
	if len(ids(group["peers"])) != 0 {
		if _, err = c.group(ctx, enrollmentGroupKey(enrollmentID), stableName("enroll-peer", enrollmentID), nil, false); err != nil {
			return err
		}
	}
	_, group, _, err = c.discoverySnapshot(ctx, enrollmentID, key, group, candidate, revoke)
	if err != nil {
		return err
	}
	if group != nil && len(ids(group["peers"])) != 0 {
		return discoveryConflict("group_binding_changed")
	}
	if err = c.checkEnrollmentGroupIsolation(ctx, id); err != nil {
		return err
	}
	if err = c.remove(ctx, "groups", id); err != nil {
		return err
	}
	return c.store.Delete(ctx, c.deployment, "groups", enrollmentGroupKey(enrollmentID))
}

func (c *Client) withdrawalProof(ctx context.Context, enrollmentID, peer string) error {
	h, err := hash([]string{"enrollment-withdrawal", enrollmentID, peer})
	if err != nil {
		return err
	}
	return c.bind(ctx, "enrollment-withdrawal", enrollmentID, peer, h)
}

// DiscoveryReason returns a safe diagnostic suitable for audit/operation data.
func DiscoveryReason(err error) string {
	var conflict *DiscoveryConflict
	if errors.As(err, &conflict) {
		switch conflict.Reason {
		case reenrollReason, "setup_key_settings_changed", "setup_key_revoked", "setup_key_reused",
			"user_owned_peer", "ephemeral_peer", "multiple_group_peers", "group_binding_changed",
			"correlation_group_has_resources", "new_enrollment_group_not_empty",
			"correlation_group_referenced_by_policies", "correlation_group_referenced_by_routes",
			"correlation_group_referenced_by_nameservers":
			return conflict.Reason
		}
	}
	if IsKind(err, Conflict) {
		return "management_binding_conflict"
	}
	return "management_peer_unconfirmed"
}
