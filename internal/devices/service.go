// Package devices authorizes verified NetBird peers under the member row lock.
package devices

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"net/netip"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrLimit       = errors.New("device limit reached")
	ErrEntitlement = errors.New("entitlement expired or unavailable")
	ErrUnmapped    = errors.New("peer identity has no verified member mapping")
)

// Peer is observed through a trusted adapter, never accepted from a member body.
// Setup-key/server peers have no ExternalUserID and cannot be authorized here.
type Peer struct {
	DeploymentID, ID, ExternalUserID, Name, Platform, OverlayScope string
	OverlayIP                                                      netip.Addr
	Authorized                                                     bool
	Groups                                                         []uuid.UUID
}

// Adapter is the narrow seam for the Wave 3 internal/netbird integration. Implementations preserve all groups outside platform ownership.
type Adapter interface {
	ObservePeer(context.Context, string, string) (Peer, error)
	SetAuthorization(context.Context, string, string, bool, []uuid.UUID) error
}
type Service struct {
	Pool         *pgxpool.Pool
	Entitlements *entitlement.Store
	Adapter      Adapter
}
type Device struct {
	ID                                  uuid.UUID `json:"device_id"`
	MemberID                            uuid.UUID `json:"-"`
	Name                                string    `json:"name"`
	Platform                            string    `json:"platform"`
	Status                              string    `json:"status"`
	AuthorizedAt, RevokedAt, LastSeenAt *time.Time
	CreatedAt                           time.Time
	Version                             decstr.U64 `json:"resource_version"`
	ObservedAuthorized                  bool       `json:"observed_authorized"`
}
type Change struct {
	DeviceID, MemberID uuid.UUID
	OperationID        *uuid.UUID
	Revision           decstr.U64
	Replayed           bool
}
type Event struct {
	MemberID            uuid.UUID  `json:"member_id"`
	DeviceID            uuid.UUID  `json:"device_id"`
	DeviceVersion       decstr.U64 `json:"device_version"`
	EntitlementRevision decstr.U64 `json:"entitlement_revision"`
	DeploymentID        string     `json:"deployment_id"`
	PeerID              string     `json:"peer_id"`
	BindingRevision     decstr.U64 `json:"binding_revision"`
	OperationID         *uuid.UUID `json:"operation_id"`
}

func enqueue(ctx context.Context, tx pgx.Tx, typ string, event Event) error {
	raw, err := canonjson.Marshal(event)
	if err != nil {
		return err
	}
	for _, name := range []string{typ, "netbird.device_sync"} {
		if err = db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateID: event.DeviceID, AggregateType: "device", AggregateVersion: event.DeviceVersion.String(), EventType: name, SchemaVersion: 1, Payload: raw}); err != nil {
			return err
		}
	}
	return nil
}

// ObserveAndAuthorize always re-observes by external ID, then verifies the
// explicit cross-client identity binding. Email is never read (§2, §15.2).
func (s *Service) ObserveAndAuthorize(ctx context.Context, c entitlement.Command, deployment, peerID string) (Change, error) {
	if s.Adapter == nil {
		return Change{}, entitlement.ErrInvalid
	}
	p, err := s.Adapter.ObservePeer(ctx, deployment, peerID)
	if err != nil {
		return Change{}, err
	}
	if p.DeploymentID != deployment || p.ID != peerID {
		return Change{}, ErrUnmapped
	}
	return s.AuthorizeDevice(ctx, c, p)
}
func (s *Service) AuthorizeDevice(ctx context.Context, c entitlement.Command, p Peer) (Change, error) {
	var result Change
	if c.Actor.ID == uuid.Nil || (c.Actor.Role != "system" && c.Actor.Role != "admin") {
		return result, entitlement.ErrForbidden
	}
	if p.ExternalUserID == "" {
		return result, ErrUnmapped
	}
	if !entitlement.Reason(p.DeploymentID, 1000) || !entitlement.Reason(p.ID, 1000) || !entitlement.Reason(p.ExternalUserID, 1000) || !entitlement.Reason(p.OverlayScope, 1000) || !netip.MustParsePrefix("100.64.0.0/10").Contains(p.OverlayIP) || len([]rune(p.Name)) > 100 {
		return result, entitlement.ErrInvalid
	}
	switch p.Platform {
	case "windows", "macos", "ios", "android", "linux", "unknown":
	default:
		p.Platform = "unknown"
	}
	err := c.InTx(ctx, s.Pool, func(tx pgx.Tx) error {
		record, err := c.Begin(ctx, tx, "POST", "device.authorize", map[string]any{"deployment_id": p.DeploymentID, "peer_id": p.ID, "external_user_id": p.ExternalUserID, "overlay_scope": p.OverlayScope, "overlay_ip": p.OverlayIP.String()})
		if err != nil {
			return err
		}
		if !record.New {
			result.DeviceID = *record.ResourceID
			result.OperationID = record.OperationID
			result.Replayed = true
			return nil
		}
		var member uuid.UUID
		err = tx.QueryRow(ctx, `SELECT b.member_id FROM netbird_user_bindings b JOIN identity_bindings i ON i.id=b.identity_binding_id AND i.member_id=b.member_id WHERE b.deployment_id=$1 AND b.external_user_id=$2 AND b.mapping_proof<>'' AND i.stable_claim_ref<>'' AND i.binding_method<>'' AND b.verified_at<=clock_timestamp() AND i.verified_at<=clock_timestamp()`, p.DeploymentID, p.ExternalUserID).Scan(&member)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUnmapped
		}
		if err != nil {
			return err
		}
		projection, err := s.Entitlements.Recompute(ctx, tx, member)
		if err != nil {
			return err
		}
		if !projection.Allows(s.Entitlements.Time()) {
			return ErrEntitlement
		}
		var existing uuid.UUID
		var owner uuid.UUID
		var status string
		err = tx.QueryRow(ctx, `SELECT id,member_id,status FROM member_devices WHERE deployment_id=$1 AND peer_id=$2`, p.DeploymentID, p.ID).Scan(&existing, &owner, &status)
		if err == nil {
			if owner != member {
				return ErrUnmapped
			}
			if status != "authorized" {
				return entitlement.ErrConflict
			}
			var bindingMatches bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device_bindings WHERE device_id=$1 AND overlay_scope=$2 AND overlay_ip=$3::inet AND valid_from<=clock_timestamp() AND valid_until>clock_timestamp())`, existing, p.OverlayScope, p.OverlayIP.String()).Scan(&bindingMatches); err != nil {
				return err
			}
			if !bindingMatches {
				return entitlement.ErrConflict
			}

			op, err := s.targetOperation(ctx, tx, c.Actor, existing, member, api.OperationKindMemberDeviceAuthorize, "device.authorized", projection.Revision)
			if err != nil {
				return err
			}
			result = Change{DeviceID: existing, MemberID: member, Revision: projection.Revision, OperationID: &op, Replayed: true}
			return entitlement.Complete(ctx, tx, record, api.ResourceTypeMemberDevice, existing, &op, 202)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var count int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, member).Scan(&count); err != nil {
			return err
		}
		if count >= projection.DeviceLimit {
			return ErrLimit
		}
		device, err := uuid.NewV7()
		if err != nil {
			return err
		}
		binding, err := uuid.NewV7()
		if err != nil {
			return err
		}
		now := s.Entitlements.Time()
		_, err = tx.Exec(ctx, `INSERT INTO member_devices(id,member_id,deployment_id,peer_id,name,status,authorized_at,platform,last_seen_at) VALUES($1,$2,$3,$4,$5,'authorized',$6,$7,$6)`, device, member, p.DeploymentID, p.ID, p.Name, now, p.Platform)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO device_bindings(id,device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref) VALUES($1,$2,$3,$4,$5::inet,1,$6,$8,$7)`, binding, device, member, p.OverlayScope, p.OverlayIP.String(), now, p.ID, projection.EffectiveServiceEnd)
		if err != nil {
			return err
		}
		opActor := entitlement.Actor{ID: member, Role: string(api.AuditEventActorTypeMember), RequestID: c.Actor.RequestID}
		op, err := entitlement.Operation(ctx, tx, opActor, api.OperationKindMemberDeviceAuthorize, api.ResourceTypeMemberDevice, device, 1)
		if err != nil {
			return err
		}
		evt := Event{MemberID: member, DeviceID: device, DeviceVersion: 1, EntitlementRevision: projection.Revision, DeploymentID: p.DeploymentID, PeerID: p.ID, BindingRevision: 1, OperationID: &op}
		if err = enqueue(ctx, tx, "device.authorized", evt); err != nil {
			return err
		}
		if err = entitlement.Audit(ctx, tx, c.Actor, "device.authorize", api.ResourceTypeMemberDevice, device, nil, map[string]any{"member_id": member, "status": "authorized"}); err != nil {
			return err
		}
		result = Change{DeviceID: device, MemberID: member, OperationID: &op, Revision: projection.Revision}
		return entitlement.Complete(ctx, tx, record, api.ResourceTypeMemberDevice, device, &op, 202)
	})
	return result, err
}

const deviceColumns = `id,member_id,name,platform,status,authorized_at,coalesce(revoked_at,closed_at),last_seen_at,created_at,resource_version::text,coalesce((SELECT o.observed_authorized AND o.observed_version=member_devices.resource_version AND o.observed_entitlement_revision=e.revision AND e.valid_from<=clock_timestamp() AND e.effective_service_end>clock_timestamp() AND NOT e.admin_suspended FROM device_authorization_observations o JOIN entitlements e ON e.member_id=member_devices.member_id WHERE o.device_id=member_devices.id),false)`

func scan(row pgx.Row) (Device, error) {
	var d Device
	err := row.Scan(&d.ID, &d.MemberID, &d.Name, &d.Platform, &d.Status, &d.AuthorizedAt, &d.RevokedAt, &d.LastSeenAt, &d.CreatedAt, &d.Version, &d.ObservedAuthorized)
	if d.Status == "closed" {
		d.Status = "revoked"
	}
	return d, db.MapError(err)
}
func (s *Service) Get(ctx context.Context, actor entitlement.Actor, id uuid.UUID) (Device, error) {
	if actor.ID == uuid.Nil || actor.Role != "member" && actor.Role != "admin" {
		return Device{}, db.ErrNotFound
	}
	return scan(s.Pool.QueryRow(ctx, `SELECT `+deviceColumns+` FROM member_devices WHERE id=$1 AND ($2='admin' OR member_id=$3)`, id, actor.Role, actor.ID))
}

type Page struct {
	Items                        []Device
	NextCursor                   string
	AuthorizedCount, DeviceLimit int
}

func (s *Service) List(ctx context.Context, actor entitlement.Actor, member uuid.UUID, cursor string, limit int) (Page, error) {
	out := Page{Items: []Device{}}
	if err := actor.Own(member); err != nil {
		return out, err
	}
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return out, entitlement.ErrInvalid
	}
	after := uuid.Nil
	if cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(cursor)
		if e != nil || len(raw) != 16 {
			return out, entitlement.ErrInvalid
		}
		copy(after[:], raw)
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+deviceColumns+` FROM member_devices WHERE member_id=$1 AND id>$2 ORDER BY id LIMIT $3`, member, after, limit+1)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			rows.Close()
			return out, e
		}
		out.Items = append(out.Items, d)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	if len(out.Items) > limit {
		out.Items = out.Items[:limit]
		id := out.Items[limit-1].ID
		out.NextCursor = base64.RawURLEncoding.EncodeToString(id[:])
	}
	err = s.Pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'),coalesce((SELECT device_limit FROM entitlements WHERE member_id=$1),0)`, member).Scan(&out.AuthorizedCount, &out.DeviceLimit)
	return out, err
}
func (s *Service) RevokeDevice(ctx context.Context, c entitlement.Command, id uuid.UUID) (Change, error) {
	result := Change{DeviceID: id}
	d, err := s.Get(ctx, c.Actor, id)
	if err != nil {
		return result, err
	}
	err = c.InTx(ctx, s.Pool, func(tx pgx.Tx) error {
		record, err := c.Begin(ctx, tx, "DELETE", "/member/devices/"+id.String(), map[string]any{})
		if err != nil {
			return err
		}
		if !record.New {
			result.OperationID = record.OperationID
			result.Replayed = true
			return nil
		}
		projection, err := s.Entitlements.Recompute(ctx, tx, d.MemberID)
		if err != nil {
			return err
		}
		var deployment, peer, status string
		var version decstr.U64
		err = tx.QueryRow(ctx, `SELECT deployment_id,peer_id,status,resource_version::text FROM member_devices WHERE id=$1 AND member_id=$2 FOR UPDATE`, id, d.MemberID).Scan(&deployment, &peer, &status, &version)
		if err != nil {
			return db.MapError(err)
		}
		if status == "revoked" || status == "closed" {
			result.Replayed = true
			op, err := s.targetOperation(ctx, tx, c.Actor, id, d.MemberID, api.OperationKindMemberDeviceRevoke, "device.revoked", projection.Revision)
			if err != nil {
				return err
			}
			result.OperationID = &op
			return entitlement.Complete(ctx, tx, record, api.ResourceTypeMemberDevice, id, &op, 202)
		}
		now := s.Entitlements.Time()
		_, err = tx.Exec(ctx, `UPDATE member_devices SET status='revoked',revoked_at=$2,resource_version=resource_version+1 WHERE id=$1`, id, now)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE device_bindings SET valid_until=greatest(valid_from+interval '1 microsecond',$2) WHERE device_id=$1 AND valid_until>$2`, id, now)
		if err != nil {
			return err
		}
		version, err = version.Add(1)
		if err != nil {
			return err
		}
		op, err := entitlement.Operation(ctx, tx, entitlement.Actor{ID: d.MemberID, Role: string(api.AuditEventActorTypeMember), RequestID: c.Actor.RequestID}, api.OperationKindMemberDeviceRevoke, api.ResourceTypeMemberDevice, id, version)
		if err != nil {
			return err
		}
		var binding decstr.U64
		if err = tx.QueryRow(ctx, `SELECT coalesce(max(binding_revision),0)::text FROM device_bindings WHERE device_id=$1`, id).Scan(&binding); err != nil {
			return err
		}
		if err = enqueue(ctx, tx, "device.revoked", Event{MemberID: d.MemberID, DeviceID: id, DeviceVersion: version, EntitlementRevision: projection.Revision, DeploymentID: deployment, PeerID: peer, BindingRevision: binding, OperationID: &op}); err != nil {
			return err
		}
		if err = entitlement.Audit(ctx, tx, c.Actor, "device.revoke", api.ResourceTypeMemberDevice, id, map[string]any{"status": status}, map[string]any{"status": "revoked"}); err != nil {
			return err
		}
		result.MemberID = d.MemberID
		result.OperationID = &op
		result.Revision = projection.Revision
		return entitlement.Complete(ctx, tx, record, api.ResourceTypeMemberDevice, id, &op, 202)
	})
	return result, err
}

// D-029 / review finding 15: device operations belong to the member; admins can read them too.
// A new key for an achieved state returns its original operation. Legacy rows
// without one enqueue a fresh read-back reconciliation, never an empty 202.
func (s *Service) targetOperation(ctx context.Context, tx pgx.Tx, actor entitlement.Actor, device, member uuid.UUID, kind api.OperationKind, event string, revision decstr.U64) (uuid.UUID, error) {
	if !kind.Valid() {
		return uuid.Nil, entitlement.ErrInvalid
	}
	var op uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM operations WHERE resource_type=$1 AND resource_id=$2 AND kind=$3 AND principal=$4 AND desired_generation=(SELECT resource_version FROM member_devices WHERE id=$2) ORDER BY created_at DESC,id DESC LIMIT 1`, api.ResourceTypeMemberDevice, device, kind, string(api.AuditEventActorTypeMember)+":"+member.String()).Scan(&op)
	if err == nil {
		return op, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, err
	}
	var version, binding decstr.U64
	var deployment, peer string
	if err = tx.QueryRow(ctx, `UPDATE member_devices SET resource_version=resource_version+1 WHERE id=$1 RETURNING resource_version::text,deployment_id,peer_id`, device).Scan(&version, &deployment, &peer); err != nil {
		return uuid.Nil, err
	}
	if err = tx.QueryRow(ctx, `SELECT coalesce(max(binding_revision),0)::text FROM device_bindings WHERE device_id=$1`, device).Scan(&binding); err != nil {
		return uuid.Nil, err
	}
	op, err = entitlement.Operation(ctx, tx, entitlement.Actor{ID: member, Role: string(api.AuditEventActorTypeMember), RequestID: actor.RequestID}, kind, api.ResourceTypeMemberDevice, device, version)
	if err != nil {
		return uuid.Nil, err
	}
	err = enqueue(ctx, tx, event, Event{MemberID: member, DeviceID: device, DeviceVersion: version, EntitlementRevision: revision, DeploymentID: deployment, PeerID: peer, BindingRevision: binding, OperationID: &op})
	return op, err
}
