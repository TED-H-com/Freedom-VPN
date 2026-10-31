package devices

import (
	"context"
	"encoding/base64"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// AdminQuery is the operator device page. Nil filters are omitted. Status is
// the public DeviceStatus, not the stored member_devices.status.
type AdminQuery struct {
	Member      *uuid.UUID
	Unconfirmed *bool
	Status      string
	Cursor      string
	Limit       int
}

// ListAdmin pages every member device in ascending (member_id, device_id) order.
// Filters select the page and do not change that order. The cursor is bound to
// the filters, the same way ListMembers binds entitlement_state.
func (s *Service) ListAdmin(ctx context.Context, actor entitlement.Actor, q AdminQuery) (Page, error) {
	out := Page{Items: []Device{}}
	if err := actor.Admin(); err != nil {
		return out, err
	}
	if q.Limit == 0 {
		q.Limit = 50
	}
	if q.Limit < 1 || q.Limit > 100 || (q.Status != "" && !api.DeviceStatus(q.Status).Valid()) {
		return out, entitlement.ErrInvalid
	}
	if q.Member != nil && *q.Member == uuid.Nil {
		return out, entitlement.ErrInvalid
	}
	afterMember, afterDevice, err := decodeAdminCursor(q)
	if err != nil {
		return out, err
	}
	var member, unconfirmed any
	if q.Member != nil {
		member = *q.Member
	}
	if q.Unconfirmed != nil {
		unconfirmed = *q.Unconfirmed
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+deviceColumns+` FROM member_devices
WHERE ($1::uuid IS NULL OR member_id=$1)
 AND ($2::boolean IS NULL OR (EXISTS(SELECT 1 FROM device_authorization_fences f WHERE f.device_id=member_devices.id))=$2)
 AND ($3='' OR (
      ($3='authorized' AND status='authorized' AND `+observedAuthorizedSQL+`)
   OR ($3='pending' AND (status='pending' OR (status='authorized' AND NOT `+observedAuthorizedSQL+`)))
   OR ($3='revoked' AND status IN ('revoked','closed'))))
 AND (member_id,id)>($4,$5)
ORDER BY member_id,id LIMIT $6`, member, unconfirmed, q.Status, afterMember, afterDevice, q.Limit+1)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, d)
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) > q.Limit {
		out.Items = out.Items[:q.Limit]
		last := out.Items[q.Limit-1]
		out.NextCursor = encodeAdminCursor(last.MemberID, last.ID, q)
	}
	return out, nil
}

func adminFilterKey(q AdminQuery) string {
	member := uuid.Nil
	if q.Member != nil {
		member = *q.Member
	}
	flag := "omit"
	if q.Unconfirmed != nil {
		if *q.Unconfirmed {
			flag = "true"
		} else {
			flag = "false"
		}
	}
	return member.String() + "|" + flag + "|" + q.Status
}

func encodeAdminCursor(member, device uuid.UUID, q AdminQuery) string {
	raw := make([]byte, 0, 33)
	raw = append(raw, 1)
	raw = append(raw, member[:]...)
	raw = append(raw, device[:]...)
	raw = append(raw, adminFilterKey(q)...)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeAdminCursor(q AdminQuery) (uuid.UUID, uuid.UUID, error) {
	if q.Cursor == "" {
		return uuid.Nil, uuid.Nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(q.Cursor)
	if err != nil || len(raw) < 33 || raw[0] != 1 || string(raw[33:]) != adminFilterKey(q) {
		return uuid.Nil, uuid.Nil, entitlement.ErrInvalid
	}
	var member, device uuid.UUID
	copy(member[:], raw[1:17])
	copy(device[:], raw[17:33])
	return member, device, nil
}

// AuthorizeMemberDevice is the admin re-grant of a fenced device and the
// ordinary authorization of a pending device. Revoked and closed devices
// conflict. A pending device over the allowance returns ErrLimit and keeps
// its fence. The caller supplies the HTTP idempotency transaction.
func (s *Service) AuthorizeMemberDevice(ctx context.Context, c entitlement.Command, member, id uuid.UUID) (result Change, err error) {
	if err = c.Actor.Admin(); err != nil {
		return result, err
	}
	err = c.InTx(ctx, s.Pool, func(tx pgx.Tx) error {
		record, err := c.Begin(ctx, tx, "POST", "/admin/members/"+member.String()+"/devices/"+id.String()+"/authorize", map[string]any{})
		if err != nil {
			return err
		}
		if !record.New {
			result.DeviceID = id
			if record.ResourceID != nil {
				result.DeviceID = *record.ResourceID
			}
			result.OperationID = record.OperationID
			result.Replayed = true
			return nil
		}
		var owner uuid.UUID
		var status string
		err = tx.QueryRow(ctx, `SELECT member_id,status FROM member_devices WHERE id=$1 FOR UPDATE`, id).Scan(&owner, &status)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && owner != member) {
			return db.ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == "revoked" || status == "closed" {
			return entitlement.ErrConflict
		}
		projection, err := s.Entitlements.Recompute(ctx, tx, member)
		if err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, `SELECT status FROM member_devices WHERE id=$1 AND member_id=$2`, id, member).Scan(&status); err != nil {
			return db.MapError(err)
		}
		if status == "revoked" || status == "closed" {
			return entitlement.ErrConflict
		}
		if status == "authorized" {
			peer, err := loadAuthorizedPeer(ctx, tx, id)
			if err != nil {
				return err
			}
			scoped := c
			scoped.Tx = tx
			result, err = s.AuthorizeDevice(ctx, scoped, peer)
			if err != nil {
				return err
			}
			if result.OperationID == nil {
				return entitlement.ErrConflict
			}
			return entitlement.Complete(ctx, tx, record, api.ResourceTypeMemberDevice, result.DeviceID, result.OperationID, 202)
		}
		if status != "pending" {
			return entitlement.ErrConflict
		}
		if !projection.Allows(s.Entitlements.Time()) {
			return s.denyAdmin(ctx, c, tx, id, member, ErrEntitlement)
		}
		var count int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'`, member).Scan(&count); err != nil {
			return err
		}
		if count >= projection.DeviceLimit {
			return s.denyAdmin(ctx, c, tx, id, member, ErrLimit)
		}
		return s.promotePending(ctx, tx, c, record, member, id, projection, &result)
	})
	return result, err
}

func (s *Service) denyAdmin(ctx context.Context, c entitlement.Command, tx pgx.Tx, id, member uuid.UUID, cause error) error {
	var deployment, peer string
	_ = tx.QueryRow(ctx, `SELECT deployment_id,peer_id FROM member_devices WHERE id=$1`, id).Scan(&deployment, &peer)
	if err := s.recordDenial(ctx, c, Peer{DeploymentID: deployment, ID: peer}, member, cause); err != nil {
		return err
	}
	return cause
}

func loadAuthorizedPeer(ctx context.Context, tx pgx.Tx, id uuid.UUID) (Peer, error) {
	var p Peer
	var ip string
	err := tx.QueryRow(ctx, `SELECT d.deployment_id,d.peer_id,d.name,d.platform,b.overlay_scope,host(b.overlay_ip),n.external_user_id
FROM member_devices d
JOIN device_bindings b ON b.device_id=d.id AND b.member_id=d.member_id AND b.valid_from<=clock_timestamp() AND b.valid_until>clock_timestamp()
JOIN netbird_user_bindings n ON n.member_id=d.member_id AND n.deployment_id=d.deployment_id
JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id
WHERE d.id=$1 AND n.mapping_proof='shared_issuer_public_subject:'||i.issuer AND i.subject=n.external_user_id AND n.revision>0 AND i.stable_claim_ref='sub' AND i.binding_method='oidc' AND n.verified_at<=clock_timestamp() AND i.verified_at<=clock_timestamp()
ORDER BY b.binding_revision DESC LIMIT 1`, id).Scan(&p.DeploymentID, &p.ID, &p.Name, &p.Platform, &p.OverlayScope, &ip, &p.ExternalUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Peer{}, entitlement.ErrConflict
	}
	if err != nil {
		return Peer{}, err
	}
	p.OverlayIP, err = netip.ParseAddr(ip)
	if err != nil {
		return Peer{}, entitlement.ErrInvalid
	}
	return p, nil
}

func (s *Service) promotePending(ctx context.Context, tx pgx.Tx, c entitlement.Command, record db.IdempotencyRecord, member, id uuid.UUID, projection entitlement.Projection, result *Change) error {
	if projection.EffectiveServiceEnd == nil {
		return ErrEntitlement
	}
	now := s.Entitlements.Time()
	var version decstr.U64
	var deployment, peer string
	err := tx.QueryRow(ctx, `UPDATE member_devices SET status='authorized',authorized_at=coalesce(authorized_at,$2),resource_version=resource_version+1 WHERE id=$1 AND member_id=$3 AND status='pending' RETURNING resource_version::text,deployment_id,peer_id`, id, now, member).Scan(&version, &deployment, &peer)
	if errors.Is(err, pgx.ErrNoRows) {
		return entitlement.ErrConflict
	}
	if err != nil {
		return authorizationLimit(err)
	}
	if err = ensureOpenBinding(ctx, tx, id, member, now, *projection.EffectiveServiceEnd); err != nil {
		return err
	}
	var binding decstr.U64
	if err = tx.QueryRow(ctx, `SELECT coalesce(max(binding_revision),0)::text FROM device_bindings WHERE device_id=$1`, id).Scan(&binding); err != nil {
		return err
	}
	op, err := entitlement.Operation(ctx, tx, entitlement.Actor{ID: member, Role: string(api.AuditEventActorTypeMember), RequestID: c.Actor.RequestID}, api.OperationKindMemberDeviceAuthorize, api.ResourceTypeMemberDevice, id, version)
	if err != nil {
		return err
	}
	if err = enqueue(ctx, tx, "device.authorized", Event{MemberID: member, DeviceID: id, DeviceVersion: version, EntitlementRevision: projection.Revision, DeploymentID: deployment, PeerID: peer, BindingRevision: binding, OperationID: &op}); err != nil {
		return err
	}
	if err = entitlement.Audit(ctx, tx, c.Actor, "device.authorize", api.ResourceTypeMemberDevice, id, map[string]any{"status": "pending"}, map[string]any{"member_id": member, "status": "authorized", "reason": "admin_authorized"}); err != nil {
		return err
	}
	if _, err = clearAuthorizationFence(ctx, tx, id); err != nil {
		return err
	}
	*result = Change{DeviceID: id, MemberID: member, OperationID: &op, Revision: projection.Revision}
	return entitlement.Complete(ctx, tx, record, api.ResourceTypeMemberDevice, id, &op, 202)
}

func ensureOpenBinding(ctx context.Context, tx pgx.Tx, device, member uuid.UUID, now, until time.Time) error {
	tag, err := tx.Exec(ctx, `UPDATE device_bindings SET valid_until=$4 WHERE device_id=$1 AND member_id=$2 AND valid_from<=$3 AND valid_until>$3 AND valid_until<>$4`, device, member, now, until)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	var open bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device_bindings WHERE device_id=$1 AND member_id=$2 AND valid_from<=$3 AND valid_until>$3)`, device, member, now).Scan(&open); err != nil {
		return err
	}
	if open {
		return nil
	}
	tag, err = tx.Exec(ctx, `INSERT INTO device_bindings(device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref)
SELECT b.device_id,b.member_id,b.overlay_scope,b.overlay_ip,b.binding_revision+1,greatest($3,b.valid_until),$4,b.peer_ref
FROM (SELECT * FROM device_bindings WHERE device_id=$1 AND member_id=$2 ORDER BY binding_revision DESC LIMIT 1) b
WHERE greatest($3,b.valid_until)<$4
 AND NOT EXISTS(SELECT 1 FROM device_bindings occupied WHERE occupied.overlay_scope=b.overlay_scope AND occupied.overlay_ip=b.overlay_ip AND occupied.valid_from<$4 AND occupied.valid_until>greatest($3,b.valid_until))`, device, member, now, until)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return entitlement.ErrConflict
	}
	return nil
}

func authorizationLimit(err error) error {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23514" && strings.Contains(pgerr.Message, "device authorization exceeds") {
		return ErrLimit
	}
	return err
}
