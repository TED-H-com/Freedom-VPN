package netbird

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"slices"
	"time"
)

type Peer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UserID    string `json:"user_id"`
	IP        string `json:"ip"`
	Connected bool   `json:"connected"`
	Ephemeral bool   `json:"ephemeral"`
}
type NodePeer struct{ NodeID, IncarnationID, PeerID string }

// MemberDevice contains a previously verified netbird_user_bindings mapping
// (§2, §9.1, §15.2). The caller must verify issuer/client-scoped subject proof;
// email and display name are deliberately absent from the mapping interface.
type MemberDevice struct{ MemberID, DeviceID, PeerID, UserID, IdentityBindingID, MappingRevision string }
type Exit struct {
	ExitID, NodeID, IncarnationID, PeerID, Description string
	PaidGroupIDs                                       []string
}
type Route struct {
	ID            string   `json:"id"`
	NetworkID     string   `json:"network_id"`
	Network       string   `json:"network"`
	Peer          string   `json:"peer"`
	Groups        []string `json:"groups"`
	Enabled       bool     `json:"enabled"`
	Masquerade    bool     `json:"masquerade"`
	SkipAutoApply bool     `json:"skip_auto_apply"`
}
type Group struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Peers []Peer `json:"peers"`
}
type ObservedState struct {
	ObservedAt time.Time
	Peers      []Peer
	Groups     []Group
	Routes     []Route
	// Raw policy/nameserver responses contain no keys/tokens. They are provided
	// for reconciliation, not logging; callers apply their own privacy filter.
	Policies    []json.RawMessage
	Nameservers []json.RawMessage
}

func convert[T any](o object) T {
	var v T
	b, _ := json.Marshal(o)
	_ = json.Unmarshal(b, &v)
	return v
}
func valid(values ...string) bool {
	for _, s := range values {
		if s == "" || len(s) > 256 {
			return false
		}
	}
	return true
}
func (c *Client) recordPeer(ctx context.Context, key string, p Peer, identity any) error {
	h, e := hash(identity)
	if e != nil {
		return e
	}
	b, ok, e := c.store.Get(ctx, c.deployment, "peers", key)
	if e != nil {
		return e
	}
	if ok && (b.ExternalID != p.ID || b.DesiredHash != h) {
		return failure(Conflict)
	}
	return c.bind(ctx, "peers", key, p.ID, h)
}
func (c *Client) ensureNode(ctx context.Context, n NodePeer) (Peer, error) {
	if !valid(n.NodeID, n.IncarnationID, n.PeerID) {
		return Peer{}, failure(Invalid)
	}
	o, e := c.get(ctx, "peers", n.PeerID)
	if e != nil {
		return Peer{}, e
	}
	p := convert[Peer](o)
	if p.UserID != "" || p.Ephemeral {
		return Peer{}, failure(Conflict)
	}
	if e = c.recordPeer(ctx, "node:"+n.NodeID+":"+n.IncarnationID, p, n); e != nil {
		return Peer{}, e
	}
	return p, nil
}

// EnsureNodePeer verifies an already enrolled peer reported through the node's
// authenticated incarnation channel (§6.1). REST cannot create WireGuard peers.
func (c *Client) EnsureNodePeer(ctx context.Context, n NodePeer) (Peer, error) {
	unlock, e := c.lock(ctx)
	if e != nil {
		return Peer{}, e
	}
	defer unlock()
	return c.ensureNode(ctx, n)
}
func (c *Client) ensureMember(ctx context.Context, d MemberDevice) (Peer, error) {
	if !valid(d.MemberID, d.DeviceID, d.PeerID, d.UserID, d.IdentityBindingID, d.MappingRevision) {
		return Peer{}, failure(Invalid)
	}
	o, e := c.get(ctx, "peers", d.PeerID)
	if e != nil {
		return Peer{}, e
	}
	p := convert[Peer](o)
	if p.UserID != d.UserID || p.Ephemeral {
		return Peer{}, failure(Conflict)
	}
	users, e := c.list(ctx, "users")
	if e != nil {
		return Peer{}, e
	}
	found := false
	for _, u := range users {
		if stringField(u, "id") == d.UserID {
			found = true
			if u["is_service_user"] == true || u["is_blocked"] == true || len(ids(u["auto_groups"])) > 0 {
				return Peer{}, failure(Conflict)
			}
		}
	}
	if !found {
		return Peer{}, failure(NotFound)
	}
	b, ok, e := c.store.Get(ctx, c.deployment, "users", d.MemberID)
	if e != nil {
		return Peer{}, e
	}
	proof, e := hash([]string{d.IdentityBindingID, d.MappingRevision, d.UserID})
	if e != nil {
		return Peer{}, e
	}
	if ok && (b.ExternalID != d.UserID || b.DesiredHash != proof) {
		return Peer{}, failure(Conflict)
	}
	if e = c.bind(ctx, "users", d.MemberID, d.UserID, proof); e != nil {
		return Peer{}, e
	}
	if e = c.recordPeer(ctx, "device:"+d.DeviceID, p, []string{d.MemberID, d.UserID, d.DeviceID}); e != nil {
		return Peer{}, e
	}
	return p, nil
}

// EnsureMemberDevice maps identity only; it never grants groups (§9.4).
func (c *Client) EnsureMemberDevice(ctx context.Context, d MemberDevice) (Peer, error) {
	unlock, e := c.lock(ctx)
	if e != nil {
		return Peer{}, e
	}
	defer unlock()
	return c.ensureMember(ctx, d)
}
func (c *Client) group(ctx context.Context, key, name string, peers []string, preserve bool) (string, error) {
	if preserve {
		o, e := c.locate(ctx, "groups", key, name)
		if e != nil {
			return "", e
		}
		if o != nil {
			peers = ids(o["peers"])
		}
	}
	o, e := c.ensure(ctx, "groups", key, object{"name": name, "peers": sorted(peers), "resources": []any{}})
	return stringField(o, "id"), e
}

// EnsurePaidGroup creates an empty device group. Subsequent calls preserve its
// members; only ApplyDeviceAuthorization changes paid memberships.
func (c *Client) EnsurePaidGroup(ctx context.Context, groupID string) (string, error) {
	if !valid(groupID) {
		return "", failure(Invalid)
	}
	unlock, e := c.lock(ctx)
	if e != nil {
		return "", e
	}
	defer unlock()
	return c.group(ctx, "paid:"+groupID, stableName("paid", groupID), nil, true)
}
func (c *Client) paid(ctx context.Context, externalID string) (object, error) {
	o, e := c.get(ctx, "groups", externalID)
	if e != nil {
		return nil, e
	}
	// Stable prefix is the adapter's reserved namespace. Do not accept All or an
	// arbitrary user-created group as an entitlement distribution target.
	name := stringField(o, "name")
	if len(name) != len("fvpn-paid-")+24 || name[:len("fvpn-paid-")] != "fvpn-paid-" {
		return nil, failure(Invalid)
	}
	_, known, err := c.store.FindExternal(ctx, c.deployment, "groups", externalID)
	if err != nil {
		return nil, err
	}
	if !known {
		return nil, failure(Conflict)
	}
	return o, nil
}

// ApplyDeviceAuthorization replaces this device's paid-group set. Removal is
// performed before additions. An interrupted run is reconciled on retry. Device
// quota and entitlement checks must already have committed in the control DB.
func (c *Client) ApplyDeviceAuthorization(ctx context.Context, d MemberDevice, paidGroupIDs []string) error {
	unlock, e := c.lock(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	if _, e = c.ensureMember(ctx, d); e != nil {
		return e
	}
	return c.authorize(ctx, d.PeerID, paidGroupIDs)
}
func (c *Client) authorize(ctx context.Context, peer string, wanted []string) error {
	wanted = sorted(wanted)
	for _, id := range wanted {
		if _, e := c.paid(ctx, id); e != nil {
			return e
		}
	}
	all, e := c.list(ctx, "groups")
	if e != nil {
		return e
	}
	for _, add := range []bool{false, true} {
		for _, g := range all {
			name := stringField(g, "name")
			if len(name) != len("fvpn-paid-")+24 || name[:len("fvpn-paid-")] != "fvpn-paid-" {
				continue
			}
			id := stringField(g, "id")
			has := slices.Contains(ids(g["peers"]), peer)
			want := slices.Contains(wanted, id)
			if want == has || want != add {
				continue
			}
			peers := ids(g["peers"])
			if want {
				peers = append(peers, peer)
			} else {
				peers = slices.DeleteFunc(peers, func(s string) bool { return s == peer })
			}
			request := object{"name": name, "peers": sorted(peers), "resources": g["resources"]}
			if request["resources"] == nil {
				request["resources"] = []any{}
			}
			// This group already exists; don't invent another binding key for it.
			if e = c.call(ctx, "PUT", endpoint("groups", id), request, &object{}); e != nil {
				return e
			}
			read, e := c.get(ctx, "groups", id)
			if e != nil {
				return e
			}
			wh, _ := hash(projection("groups", request))
			oh, _ := hash(projection("groups", read))
			if wh != oh {
				return failure(Unconfirmed)
			}
			binding, known, err := c.store.FindExternal(ctx, c.deployment, "groups", id)
			if err != nil {
				return err
			}
			if known {
				if err := c.bind(ctx, "groups", binding.InternalID, id, wh); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func rule(name, protocol string, sources, destinations, ports []string, bidirectional bool) object {
	r := object{"name": name, "description": "Freedom VPN managed (§7.1)", "enabled": true, "action": "accept", "bidirectional": bidirectional, "protocol": protocol, "sources": sorted(sources), "destinations": sorted(destinations)}
	if len(ports) > 0 {
		r["ports"] = sorted(ports)
	}
	return r
}
func (c *Client) policy(ctx context.Context, key string, rules []object) (string, error) {
	o, e := c.ensure(ctx, "policies", key, object{"name": stableName("policy", key), "description": "Freedom VPN managed (§7.1)", "enabled": true, "rules": rules, "source_posture_checks": []string{}})
	return stringField(o, "id"), e
}

// PublishExit keeps one stable route per exit, one peer, manual selection and
// member overlay source addresses (§7.2, §8.2). G2 supplies SNAT/enforcement.
func (c *Client) PublishExit(ctx context.Context, x Exit) (Route, error) {
	if !valid(x.ExitID) || len(x.PaidGroupIDs) == 0 {
		return Route{}, failure(Invalid)
	}
	unlock, e := c.lock(ctx)
	if e != nil {
		return Route{}, e
	}
	defer unlock()
	if _, e = c.ensureNode(ctx, NodePeer{x.NodeID, x.IncarnationID, x.PeerID}); e != nil {
		return Route{}, e
	}
	for _, g := range x.PaidGroupIDs {
		if _, e = c.paid(ctx, g); e != nil {
			return Route{}, e
		}
	}
	group, e := c.group(ctx, "exit:"+x.ExitID, stableName("exit", x.ExitID), []string{x.PeerID}, false)
	if e != nil {
		return Route{}, e
	}
	if _, e = c.policy(ctx, "exit:"+x.ExitID, []object{rule("member to exit", "icmp", x.PaidGroupIDs, []string{group}, nil, true)}); e != nil {
		return Route{}, e
	}
	o, e := c.ensure(ctx, "routes", x.ExitID, object{"network_id": stableName("exit", x.ExitID), "description": x.Description, "network": "0.0.0.0/0", "peer": x.PeerID, "groups": sorted(x.PaidGroupIDs), "enabled": true, "masquerade": false, "skip_auto_apply": true, "metric": 9999, "keep_route": false, "access_control_groups": []string{}})
	return convert[Route](o), e
}
func (c *Client) WithdrawExit(ctx context.Context, exitID string) error {
	if !valid(exitID) {
		return failure(Invalid)
	}
	unlock, e := c.lock(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	for _, r := range []struct{ kind, key, name string }{{"routes", exitID, stableName("exit", exitID)}, {"policies", "exit:" + exitID, stableName("policy", "exit:"+exitID)}} {
		o, e := c.locate(ctx, r.kind, r.key, r.name)
		if e != nil {
			return e
		}
		if o != nil {
			if e = c.remove(ctx, r.kind, stringField(o, "id")); e != nil {
				return e
			}
		}
		if e = c.store.Delete(ctx, c.deployment, r.kind, r.key); e != nil {
			return e
		}
	}
	return nil
}

// DeletePeer takes the stable binding key (node:<node>:<incarnation> or
// device:<device>). A peer serving routes must first be withdrawn explicitly.
func (c *Client) DeletePeer(ctx context.Context, bindingKey string) error {
	unlock, e := c.lock(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	b, ok, e := c.store.Get(ctx, c.deployment, "peers", bindingKey)
	if e != nil || !ok {
		return e
	}
	routes, e := c.list(ctx, "routes")
	if e != nil {
		return e
	}
	groups, e := c.list(ctx, "groups")
	if e != nil {
		return e
	}
	peerGroups := []string{}
	for _, group := range groups {
		if slices.Contains(ids(group["peers"]), b.ExternalID) {
			peerGroups = append(peerGroups, stringField(group, "id"))
		}
	}
	for _, r := range routes {
		if stringField(r, "peer") == b.ExternalID || slices.ContainsFunc(ids(r["peer_groups"]), func(id string) bool { return slices.Contains(peerGroups, id) }) {
			return failure(Conflict)
		}
	}
	if e = c.remove(ctx, "peers", b.ExternalID); e != nil {
		return e
	}
	return c.store.Delete(ctx, c.deployment, "peers", bindingKey)
}
func (c *Client) ReadObservedState(ctx context.Context) (ObservedState, error) {
	s := ObservedState{}
	for _, r := range []struct {
		path string
		out  any
	}{{"peers", &s.Peers}, {"groups", &s.Groups}, {"routes", &s.Routes}, {"policies", &s.Policies}, {"dns/nameservers", &s.Nameservers}} {
		if e := c.call(ctx, "GET", endpoint(r.path, ""), nil, r.out); e != nil {
			return ObservedState{}, e
		}
	}
	s.ObservedAt = time.Now().UTC()
	return s, nil
}

// RemoveDefaultPolicy is explicit bootstrap behavior, never a hidden side
// effect of a read. Unknown broad policies cause failure rather than deletion.
func (c *Client) RemoveDefaultPolicy(ctx context.Context) error {
	unlock, e := c.lock(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	groups, e := c.list(ctx, "groups")
	if e != nil {
		return e
	}
	all := ""
	for _, g := range groups {
		if stringField(g, "name") == "All" {
			all = stringField(g, "id")
		}
	}
	policies, e := c.list(ctx, "policies")
	if e != nil {
		return e
	}
	for _, p := range policies {
		rules, _ := p["rules"].([]any)
		for _, r := range rules {
			m, _ := r.(map[string]any)
			if all != "" && (slices.Contains(ids(m["sources"]), all) || slices.Contains(ids(m["destinations"]), all)) {
				if stringField(p, "name") != "Default" {
					return failure(Conflict)
				}
				if e = c.remove(ctx, "policies", stringField(p, "id")); e != nil {
					return e
				}
				break
			}
		}
	}
	return nil
}

// BootstrapAccess creates only exact resolver and admin-ops exceptions. The
// caller supplies enrolled resolver peer IDs and explicit ops ports (§12.2).
type AccessConfig struct {
	PaidGroupIDs, ResolverPeerIDs, ResolverIPs, AdminPeerIDs, OpsPeerIDs []string
	OpsPorts                                                             []string
}

func (c *Client) BootstrapAccess(ctx context.Context, a AccessConfig) error {
	if len(a.PaidGroupIDs) == 0 || len(a.ResolverPeerIDs) == 0 || len(a.ResolverPeerIDs) != len(a.ResolverIPs) || len(a.OpsPorts) == 0 {
		return failure(Invalid)
	}
	for _, ip := range a.ResolverIPs {
		p, e := netip.ParseAddr(ip)
		if e != nil || !netip.MustParsePrefix("100.64.0.0/10").Contains(p) {
			return failure(Invalid)
		}
	}
	unlock, e := c.lock(ctx)
	if e != nil {
		return e
	}
	defer unlock()
	for _, id := range a.PaidGroupIDs {
		if _, e = c.paid(ctx, id); e != nil {
			return e
		}
	}
	for i, id := range a.ResolverPeerIDs {
		o, e := c.get(ctx, "peers", id)
		if e != nil {
			return e
		}
		if stringField(o, "ip") != a.ResolverIPs[i] {
			return failure(Conflict)
		}
	}
	resolver, e := c.group(ctx, "resolvers", stableName("group", "resolvers"), a.ResolverPeerIDs, false)
	if e != nil {
		return e
	}
	admin, e := c.group(ctx, "admins", stableName("group", "admins"), a.AdminPeerIDs, false)
	if e != nil {
		return e
	}
	ops, e := c.group(ctx, "ops", stableName("group", "ops"), a.OpsPeerIDs, false)
	if e != nil {
		return e
	}
	for _, proto := range []string{"tcp", "udp"} {
		if _, e = c.policy(ctx, "dns:"+proto, []object{rule("DNS only", proto, a.PaidGroupIDs, []string{resolver}, []string{"53"}, false)}); e != nil {
			return e
		}
	}
	if _, e = c.policy(ctx, "ops", []object{rule("admin ops", "tcp", []string{admin}, []string{ops}, a.OpsPorts, false)}); e != nil {
		return e
	}
	ns := []object{}
	for _, ip := range a.ResolverIPs {
		ns = append(ns, object{"ip": ip, "ns_type": "udp", "port": 53})
	}
	_, e = c.ensure(ctx, "dns/nameservers", "resolvers", object{"name": stableName("dns", "resolvers"), "description": "Freedom VPN resolvers (§12.2)", "enabled": true, "groups": sorted(a.PaidGroupIDs), "nameservers": ns, "primary": true, "domains": []string{}, "search_domains_enabled": false})
	return e
}

// SetupSecret prints redacted even with fmt %#v. Reveal is deliberately explicit.
type SetupSecret struct{ value string }

func (s SetupSecret) Reveal() string               { return s.value }
func (s SetupSecret) String() string               { return "[REDACTED]" }
func (s SetupSecret) GoString() string             { return "[REDACTED]" }
func (s SetupSecret) MarshalJSON() ([]byte, error) { return []byte(`"[REDACTED]"`), nil }

type SetupKey struct {
	ID              string
	Secret          SetupSecret
	SecretAvailable bool
}

func (s SetupKey) String() string {
	return fmt.Sprintf("setup key (secret_available=%t)", s.SecretAvailable)
}

// CreateNodeSetupKey uses a stable enrollment attempt ID. A lost response is
// recovered as metadata only, never a second key or recovered plaintext (§6.4).
func (c *Client) CreateNodeSetupKey(ctx context.Context, enrollmentID string) (SetupKey, error) {
	if !valid(enrollmentID) {
		return SetupKey{}, failure(Invalid)
	}
	unlock, e := c.lock(ctx)
	if e != nil {
		return SetupKey{}, e
	}
	defer unlock()
	group, e := c.group(ctx, "unpublished-nodes", "fvpn-unpublished-nodes", nil, true)
	if e != nil {
		return SetupKey{}, e
	}
	name := stableName("enroll", enrollmentID)
	o, e := c.locate(ctx, "setup-keys", enrollmentID, name)
	if e != nil {
		return SetupKey{}, e
	}
	result := SetupKey{}
	if o == nil {
		e = c.call(ctx, "POST", "/setup-keys", object{"name": name, "type": "one-off", "expires_in": 86400, "auto_groups": []string{group}, "usage_limit": 1, "ephemeral": false, "allow_extra_dns_labels": false}, &o)
		if e != nil {
			return result, e
		}
		result.Secret = SetupSecret{stringField(o, "key")}
		result.SecretAvailable = result.Secret.value != ""
	}
	result.ID = stringField(o, "id")
	if result.ID == "" {
		return SetupKey{}, failure(Unconfirmed)
	}
	if e = c.bind(ctx, "setup-keys", enrollmentID, result.ID, ""); e != nil {
		return SetupKey{}, e
	}
	observed, e := c.get(ctx, "setup-keys", result.ID)
	if e != nil {
		return SetupKey{}, e
	}
	if stringField(observed, "name") != name || stringField(observed, "type") != "one-off" || observed["usage_limit"] != float64(1) || observed["ephemeral"] != false || !slices.Equal(ids(observed["auto_groups"]), []string{group}) {
		return SetupKey{}, failure(Unconfirmed)
	}
	if result.SecretAvailable {
		expires, err := time.Parse(time.RFC3339Nano, stringField(observed, "expires"))
		remaining := time.Until(expires)
		if err != nil || remaining <= 0 || remaining > 24*time.Hour+time.Minute || observed["revoked"] == true || observed["allow_extra_dns_labels"] == true {
			return SetupKey{}, failure(Unconfirmed)
		}
	}
	h, _ := hash(object{"name": name, "type": "one-off", "usage_limit": 1, "ephemeral": false, "auto_groups": []string{group}})
	if e = c.bind(ctx, "setup-keys", enrollmentID, result.ID, h); e != nil {
		return SetupKey{}, e
	}
	return result, nil
}

// RevokeNodeSetupKey is the explicit recovery path for a lost one-time secret.
// It does not disconnect enrolled peers (§6.4); use WithdrawExit/DeletePeer for
// revocation. A replacement uses a NEW authorized enrollment attempt ID.
func (c *Client) RevokeNodeSetupKey(ctx context.Context, enrollmentID string) error {
	if !valid(enrollmentID) {
		return failure(Invalid)
	}
	unlock, err := c.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	o, err := c.locate(ctx, "setup-keys", enrollmentID, stableName("enroll", enrollmentID))
	if err != nil || o == nil {
		return err
	}
	id := stringField(o, "id")
	if o["revoked"] != true {
		err = c.call(ctx, "PUT", endpoint("setup-keys", id), object{"revoked": true, "auto_groups": ids(o["auto_groups"])}, &object{})
		if err != nil {
			return err
		}
	}
	o, err = c.get(ctx, "setup-keys", id)
	if err != nil {
		return err
	}
	if o["revoked"] != true {
		return failure(Unconfirmed)
	}
	return nil
}
