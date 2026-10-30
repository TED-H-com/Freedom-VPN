package devices

import (
	"context"
	"errors"
	"net/netip"
	"strings"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NetBirdAdapter observes native-client-created peers. NetBird REST does not
// create WireGuard identities (§7.3); only verified user mappings can be adopted.
type NetBirdAdapter struct {
	Pool                       *pgxpool.Pool
	Client                     *netbird.Client
	Bindings                   netbird.BindingStore
	DeploymentID, OverlayScope string
	Identity                   *IdentityDiscovery
}

func (a *NetBirdAdapter) ObservePeer(ctx context.Context, deployment, id string) (Peer, error) {
	p := Peer{DeploymentID: deployment, ID: id, OverlayScope: a.OverlayScope, Platform: "unknown", IdentityUnmapped: true, Groups: []uuid.UUID{}}
	if deployment != a.DeploymentID {
		return p, ErrUnmapped
	}
	state, e := a.Client.ReadObservedState(ctx)
	if e != nil {
		return p, e
	}
	found := false
	for _, v := range state.Peers {
		if v.ID == id {
			found = true
			p.Name = v.Name
			p.ExternalUserID = v.UserID
			p.OverlayIP, e = netip.ParseAddr(v.IP)
			if e != nil {
				return p, e
			}
			if v.Ephemeral {
				return p, ErrUnmapped
			}
			break
		}
	}
	if !found {
		// Confirmed absence denies access; do not invent a member identity.
		var known bool
		if e = a.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM member_devices WHERE deployment_id=$1 AND peer_id=$2)`, deployment, id).Scan(&known); e != nil {
			return p, e
		}
		if !known {
			return p, db.ErrNotFound
		}
	}
	if a.Identity != nil && found {
		// Preserve the observed peer for withdrawal even when its identity is
		// no longer eligible. Authorization and read-back must deny it.
		p.IdentityUnmapped = true
		users, err := a.Client.ListUsers(ctx)
		if err != nil {
			return p, err
		}
		for _, user := range users {
			if user.ID == p.ExternalUserID && !user.ServiceUser && !user.Blocked {
				verified, err := a.verifiedIdentity(ctx, user.ID)
				if err != nil {
					return p, err
				}
				p.IdentityUnmapped = !verified
				break
			}
		}
	}
	for _, g := range state.Groups {
		if !strings.HasPrefix(g.Name, "fvpn-paid-") {
			continue
		}
		contains := false
		for _, v := range g.Peers {
			if v.ID == id {
				contains = true
				break
			}
		}
		if !contains {
			continue
		}
		p.Authorized = true
		b, ok, e := a.Bindings.FindExternal(ctx, deployment, "groups", g.ID)
		if e != nil {
			return p, e
		}
		// An unknown managed membership must be removed, never silently ignored.
		group := uuid.Nil
		if ok && strings.HasPrefix(b.InternalID, "paid:") {
			group, _ = uuid.Parse(strings.TrimPrefix(b.InternalID, "paid:"))
		}
		p.Groups = append(p.Groups, group)
	}
	return p, nil
}
func (a *NetBirdAdapter) device(ctx context.Context, deployment, peer string) (netbird.MemberDevice, string, error) {
	var d netbird.MemberDevice
	var member, id uuid.UUID
	var status string
	e := a.Pool.QueryRow(ctx, `SELECT id,member_id,status FROM member_devices WHERE deployment_id=$1 AND peer_id=$2`, deployment, peer).Scan(&id, &member, &status)
	if e != nil {
		return d, status, db.MapError(e)
	}
	d.MemberID = member.String()
	d.DeviceID = id.String()
	d.PeerID = peer
	var identity uuid.UUID
	var revision decstr.U64
	e = a.Pool.QueryRow(ctx, `SELECT n.external_user_id,n.identity_binding_id,n.revision::text FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id WHERE n.member_id=$1 AND n.deployment_id=$2 AND n.mapping_proof='shared_issuer_public_subject:' || i.issuer AND i.subject=n.external_user_id AND n.revision>0 AND i.stable_claim_ref='sub' AND i.binding_method='oidc' AND n.verified_at<=clock_timestamp() AND i.verified_at<=clock_timestamp()`, member, deployment).Scan(&d.UserID, &identity, &revision)
	if e != nil {
		return d, status, db.MapError(e)
	}
	d.IdentityBindingID = identity.String()
	d.MappingRevision = revision.String()
	return d, status, nil
}
func (a *NetBirdAdapter) SetAuthorization(ctx context.Context, deployment, peer string, allow bool, groups []uuid.UUID) error {
	if deployment != a.DeploymentID || (!allow && len(groups) > 0) {
		return entitlement.ErrInvalid
	}
	d, status, e := a.device(ctx, deployment, peer)
	if !allow {
		// Use the immutable previously observed identity to withdraw even if the
		// current identity mapping was removed/changed. Never use this for grants.
		if d.MemberID != "" {
			b, ok, lookupErr := a.Bindings.Get(ctx, deployment, "users", d.MemberID)
			if lookupErr != nil {
				return lookupErr
			}
			if ok {
				d.UserID = b.ExternalID
				e = nil
			}
		}
	}
	if e != nil {
		return e
	}
	observed, e := a.ObservePeer(ctx, deployment, peer)
	if e != nil {
		return e
	}
	if !observed.OverlayIP.IsValid() {
		if allow {
			return ErrUnmapped
		}
		return nil
	}
	if allow && observed.IdentityUnmapped {
		return ErrUnmapped
	}
	targets := []string{}
	if allow {
		if status != "authorized" || len(groups) == 0 {
			return ErrEntitlement
		}
		if _, e = a.Client.EnsureMemberDevice(ctx, d); e != nil {
			return e
		}
		for _, g := range groups {
			if g == uuid.Nil {
				return entitlement.ErrInvalid
			}
			external, e := a.Client.EnsurePaidGroup(ctx, g.String())
			if e != nil {
				return e
			}
			targets = append(targets, external)
		}
	}
	if e = a.Client.ApplyDeviceAuthorization(ctx, d, targets); e != nil {
		return e
	}
	// Revocation removes only owned access memberships. Retain the native peer
	// and unrelated groups; its durable device tombstone prevents reauthorization.
	return nil
}

// Discover creates D-052 public-subject proofs before admitting native peers.
// It never links pairwise subjects, email, name or IP. GET /member/devices stays
// a read of the resulting projections; this worker owns provider I/O and writes.
func (a *NetBirdAdapter) Discover(ctx context.Context, s *Service) error {
	if a.Identity == nil {
		return errors.New("oidc_issuer required for NetBird identity discovery")
	}
	users, e := a.Client.ListUsers(ctx)
	if e != nil {
		return e
	}
	eligible := make(map[string]bool, len(users))
	for _, user := range users {
		result := identityService
		switch {
		case user.ServiceUser:
		case user.Blocked:
			result = identityBlocked
		default:
			result, e = a.bindUser(ctx, user.ID)
			if e != nil {
				return e
			}
		}
		if result == identityService || result == identityBlocked {
			if e = db.InTx(ctx, a.Pool, db.TxOptions{}, func(tx pgx.Tx) error { return auditIdentity(ctx, tx, uuid.Nil, uuid.Nil, result) }); e != nil {
				return e
			}
		}
		eligible[user.ID] = result == identityCreated || result == identityUnchanged || result == identityUpdated
		a.Identity.record(ctx, result)
	}
	state, e := a.Client.ReadObservedState(ctx)
	if e != nil {
		return e
	}
	for _, p := range state.Peers {
		if p.UserID == "" || p.Ephemeral {
			continue
		}
		verified, listed := eligible[p.UserID]
		if !listed {
			a.Identity.record(ctx, identityUnmapped)
		}
		if !verified {
			c := entitlement.Command{Actor: entitlement.Actor{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Role: "system"}}
			if e = s.recordDenial(ctx, c, Peer{DeploymentID: a.DeploymentID, ID: p.ID}, uuid.Nil, ErrUnmapped); e != nil {
				return e
			}
			continue
		}
		// Existing devices reconcile separately; revoked tombstones never rejoin.
		var exists bool
		if e = a.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM member_devices WHERE deployment_id=$1 AND peer_id=$2)`, a.DeploymentID, p.ID).Scan(&exists); e != nil {
			return e
		}
		if exists {
			continue
		}
		key, e := bindingID(a.DeploymentID, p.ID)
		if e != nil {
			return e
		}
		request, e := uuid.NewV7()
		if e != nil {
			return e
		}
		_, e = s.ObserveAndAuthorize(ctx, entitlement.Command{Actor: entitlement.Actor{ID: uuid.MustParse("00000000-0000-0000-0000-000000000003"), Role: "system", RequestID: request}, Key: key}, a.DeploymentID, p.ID)
		if errors.Is(e, ErrUnmapped) {
			a.Identity.record(ctx, identityUnmapped)
		}
		if errors.Is(e, ErrLimit) || errors.Is(e, ErrEntitlement) || errors.Is(e, ErrUnmapped) || errors.Is(e, entitlement.ErrConflict) {
			continue
		}
		if e != nil {
			return e
		}
	}
	return nil
}

var _ Adapter = (*NetBirdAdapter)(nil)
