package httpwire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PublicationAdapter translates the exits workflow to the management adapter;
// production never substitutes the in-memory FakeNetBirdAdapter (§5.2, §7.2).
type PublicationAdapter struct {
	runtime *Runtime
	client  *netbird.Client
}

func NewPublicationAdapter(r *Runtime, client *netbird.Client) *PublicationAdapter {
	return &PublicationAdapter{runtime: r, client: client}
}

func (a *PublicationAdapter) check(deployment string) error {
	if a.client == nil || deployment == "" || deployment != a.runtime.Config.NetBirdDeployment {
		return errors.New("publication NetBird deployment unavailable")
	}
	return nil
}

// SQL/provider errors must not become public operation error messages.
func publicationError(err *error) {
	if *err == nil {
		return
	}
	var provider *netbird.Error
	var classified exits.PublicationEvidenceError
	if errors.As(*err, &provider) {
		return
	} // error contains only kind/status
	if errors.As(*err, &classified) {
		*err = exits.PublicationEvidenceError(exits.PublicationFailureCause(classified, "readback"))
		return
	}
	*err = exits.PublicationEvidenceError("adapter_unavailable")
}

func (a *PublicationAdapter) PublishExit(ctx context.Context, p exits.PublishExitSpec) (out exits.PublishedRouteEvidence, err error) {
	defer publicationError(&err)
	if len(p.MemberGroupIDs) == 0 {
		return out, exits.PublicationEvidenceError("no_distribution_groups")
	}
	if err = a.check(p.DeploymentID); err != nil {
		return out, err
	}
	var incarnation uuid.UUID
	err = a.runtime.Pool.QueryRow(ctx, `SELECT i.id FROM nodes n JOIN node_incarnations i ON i.id=n.current_incarnation_id JOIN exits e ON e.node_id=n.id WHERE n.id=$1 AND e.id=$2 AND i.peer_id=$3 AND i.deployment_id=$4 AND n.lifecycle_state='registered' AND i.revoked_at IS NULL AND NOT EXISTS(SELECT 1 FROM node_identity_endpoints x WHERE x.incarnation_id=i.id AND x.mapping_frozen)`, p.NodeID, p.ExitID, p.PeerID, p.DeploymentID).Scan(&incarnation)
	if err != nil {
		return out, err
	}
	// NetBird distribution groups are external IDs, not our member-group UUIDs.
	groups := make([]string, 0, len(p.MemberGroupIDs))
	for _, group := range p.MemberGroupIDs {
		external, e := a.client.EnsurePaidGroup(ctx, group.String())
		if e != nil {
			return out, e
		}
		groups = append(groups, external)
	}
	route, err := a.client.PublishExit(ctx, netbird.Exit{ExitID: p.ExitID.String(), NodeID: p.NodeID.String(), IncarnationID: incarnation.String(), PeerID: p.PeerID, PaidGroupIDs: groups, Description: p.DisplayName})
	if err != nil {
		return out, err
	}
	return exits.PublishedRouteEvidence{ExternalID: route.ID, PeerID: route.Peer, Enabled: route.Enabled, GroupIDs: route.Groups, ObservedRef: "netbird:route:" + route.ID}, nil
}

func (a *PublicationAdapter) WithdrawExit(ctx context.Context, deployment string, exitID uuid.UUID, _ string) (ref string, err error) {
	defer publicationError(&err)
	if err = a.check(deployment); err != nil {
		return "", err
	}
	// WithdrawExit locates stable bindings/names, deletes route and access policy,
	// and GETs them to confirm absence. It does not delete the node identity.
	if err = a.client.WithdrawExit(ctx, exitID.String()); err != nil {
		return "", err
	}
	return "netbird:exit:" + exitID.String() + ":withdrawn", nil
}

func (a *PublicationAdapter) ReadExitState(ctx context.Context, deployment string, exitID uuid.UUID, externalID string) (out exits.ObservedExitState, err error) {
	defer publicationError(&err)
	if err = a.check(deployment); err != nil {
		return out, err
	}
	observed, err := a.client.ReadObservedState(ctx)
	if err != nil {
		return out, err
	}
	// Match the adapter's stable name even after withdrawal deletes its binding.
	// This is the reserved naming format in netbird.stableName, verified in the
	// integration test against a real Client. The input is a UUID, not JSON.
	digest := sha256.Sum256([]byte(exitID.String()))
	name := "fvpn-exit-" + hex.EncodeToString(digest[:12])
	var route *netbird.Route
	for _, candidate := range observed.Routes {
		if candidate.NetworkID == name || externalID != "" && candidate.ID == externalID {
			if route != nil || candidate.NetworkID != name {
				return out, exits.PublicationEvidenceError("route_ambiguous")
			}
			copy := candidate
			route = &copy
		}
	}
	if route == nil {
		return exits.ObservedExitState{Exists: false, ExternalID: externalID, ObservedRef: "netbird:exit:" + exitID.String() + ":absent"}, nil
	}
	if route.Network != "0.0.0.0/0" || route.Masquerade || !route.SkipAutoApply {
		return out, exits.PublicationEvidenceError("route_contract_mismatch")
	}
	var peer string
	if err = a.runtime.Pool.QueryRow(ctx, `SELECT i.peer_id FROM exits e JOIN nodes n ON n.id=e.node_id JOIN node_incarnations i ON i.id=n.current_incarnation_id WHERE e.id=$1 AND i.deployment_id=$2`, exitID, deployment).Scan(&peer); err != nil || route.Peer != peer {
		return out, exits.PublicationEvidenceError("route_peer_mismatch")
	}
	bindings := nodes.NetBirdBindings{Pool: a.runtime.Pool}
	groups := make([]string, 0, len(route.Groups))
	for _, group := range route.Groups {
		binding, ok, e := bindings.FindExternal(ctx, deployment, "groups", group)
		if e != nil || !ok || !strings.HasPrefix(binding.InternalID, "paid:") {
			return out, exits.PublicationEvidenceError("group_evidence_unavailable")
		}
		id, e := uuid.Parse(strings.TrimPrefix(binding.InternalID, "paid:"))
		if e != nil {
			return out, e
		}
		groups = append(groups, id.String())
	}
	slices.Sort(groups)
	rows, err := a.runtime.Pool.Query(ctx, `SELECT group_id::text FROM exit_groups WHERE exit_id=$1 ORDER BY group_id`, exitID)
	if err != nil {
		return out, err
	}
	expected, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return out, err
	}
	if route.Enabled && (len(groups) == 0 || !slices.Equal(groups, expected)) {
		return out, exits.PublicationEvidenceError("route_groups_mismatch")
	}
	// Hash actual read-back in the exits service's projection, never copy its
	// desired hash as evidence. Policy/group writes were also read back by Client.
	raw, err := canonjson.Marshal(map[string]any{"exit_id": exitID.String(), "peer_id": route.Peer, "enabled": route.Enabled, "group_ids": groups})
	if err != nil {
		return out, err
	}
	hash, err := canonjson.Hash(raw)
	if err != nil {
		return out, err
	}
	return exits.ObservedExitState{Exists: true, ExternalID: route.ID, PeerID: route.Peer, Enabled: route.Enabled, GroupIDs: route.Groups, ObservedHash: hash[:], ObservedRef: "netbird:route:" + route.ID}, nil
}

var _ exits.NetBirdAdapter = (*PublicationAdapter)(nil)
