package entitlement

import (
	"context"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// AuthorizedDeviceIDsSQL is the common F-01 slot predicate for reconciliation,
// NetBird synchronization and desired-state readers. Oldest authorizations win;
// UUID order breaks timestamp ties consistently. Callers retain their own time
// and identity eligibility checks.
const AuthorizedDeviceIDsSQL = `SELECT ranked.id FROM (
 SELECT id,member_id,row_number() OVER (PARTITION BY member_id ORDER BY authorized_at ASC,id ASC) AS slot
 FROM member_devices WHERE status='authorized'
) ranked JOIN entitlements e ON e.member_id=ranked.member_id WHERE ranked.slot<=e.device_limit`

// reconcileDevices runs only under Recompute's member lock, after publishing
// the current entitlement. Historical binding intervals are never reopened.
func reconcileDevices(ctx context.Context, tx pgx.Tx, member uuid.UUID, result Result, now time.Time) error {
	rows, err := tx.Query(ctx, `UPDATE member_devices SET status='pending',resource_version=resource_version+1
 WHERE member_id=$1 AND status='authorized' AND id NOT IN (`+AuthorizedDeviceIDsSQL+` AND ranked.member_id=$1) RETURNING id`, member)
	if err != nil {
		return err
	}
	var withdrawn []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		withdrawn = append(withdrawn, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range withdrawn {
		if err = Audit(ctx, tx, Actor{ID: audit.SystemActor.ID, Role: string(audit.SystemActor.Type)}, "device.revoke", api.ResourceTypeMemberDevice, id, map[string]any{"status": "authorized"}, map[string]any{"status": "pending", "reason": "device_limit_reached"}); err != nil {
			return err
		}
	}
	// F-06: suspension, expiration and grant revocation release overlay ownership.
	if _, err = tx.Exec(ctx, `UPDATE device_bindings b SET valid_until=greatest(b.valid_from+interval '1 microsecond',$2)
 WHERE b.member_id=$1 AND b.valid_until>$2 AND (NOT $3 OR b.device_id NOT IN (`+AuthorizedDeviceIDsSQL+` AND ranked.member_id=$1))`, member, now, result.Allows(now)); err != nil {
		return err
	}
	if !result.Allows(now) {
		return nil
	}
	// Re-admit previously authorized (never revoked) pending devices oldest first
	// into newly available slots. New/unverified pending devices cannot qualify.
	rows, err = tx.Query(ctx, `SELECT d.id FROM member_devices d WHERE d.member_id=$1 AND d.status='pending' AND d.authorized_at IS NOT NULL AND d.revoked_at IS NULL AND d.closed_at IS NULL
 AND EXISTS(SELECT 1 FROM device_bindings b WHERE b.device_id=d.id)
 AND EXISTS(SELECT 1 FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id WHERE n.member_id=d.member_id AND n.deployment_id=d.deployment_id AND i.subject=n.external_user_id AND i.stable_claim_ref='sub' AND i.binding_method='oidc' AND n.mapping_proof='shared_issuer_public_subject:'||i.issuer AND n.revision>0 AND n.verified_at<=clock_timestamp() AND i.verified_at<=clock_timestamp())
 ORDER BY d.authorized_at ASC,d.id ASC LIMIT greatest(0,$2-(SELECT count(*) FROM member_devices WHERE member_id=$1 AND status='authorized'))`, member, result.DeviceLimit)
	if err != nil {
		return err
	}
	var resumed []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		resumed = append(resumed, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range resumed {
		if _, err = tx.Exec(ctx, `UPDATE member_devices SET status='authorized',resource_version=resource_version+1 WHERE id=$1`, id); err != nil {
			return err
		}
		if err = Audit(ctx, tx, Actor{ID: audit.SystemActor.ID, Role: string(audit.SystemActor.Type)}, "device.authorize", api.ResourceTypeMemberDevice, id, map[string]any{"status": "pending"}, map[string]any{"status": "authorized", "reason": "entitlement_restored"}); err != nil {
			return err
		}
	}
	// Renew only an interval which is still open now, never any historical rows.
	if _, err = tx.Exec(ctx, `UPDATE device_bindings b SET valid_until=$3 WHERE b.member_id=$1 AND b.valid_from<=$2 AND b.valid_until>greatest($2,b.valid_from+interval '1 microsecond') AND b.valid_until<>$3 AND b.device_id IN (`+AuthorizedDeviceIDsSQL+` AND ranked.member_id=$1)`, member, now, result.EffectiveServiceEnd); err != nil {
		return err
	}
	// A resumed entitlement needs a new revision. A former IP reused by someone
	// else is not reclaimed; SyncDevice will withdraw until identity is reobserved.
	_, err = tx.Exec(ctx, `INSERT INTO device_bindings(device_id,member_id,overlay_scope,overlay_ip,binding_revision,valid_from,valid_until,peer_ref)
 SELECT b.device_id,b.member_id,b.overlay_scope,b.overlay_ip,b.binding_revision+1,greatest($2,b.valid_until),$3,b.peer_ref
 FROM (SELECT DISTINCT ON (device_id) * FROM device_bindings WHERE member_id=$1 ORDER BY device_id,binding_revision DESC) b
 WHERE b.valid_until<=greatest($2,b.valid_from+interval '1 microsecond') AND greatest($2,b.valid_until)<$3 AND b.device_id IN (`+AuthorizedDeviceIDsSQL+` AND ranked.member_id=$1)
 AND NOT EXISTS(SELECT 1 FROM device_bindings occupied WHERE occupied.overlay_scope=b.overlay_scope AND occupied.overlay_ip=b.overlay_ip AND occupied.valid_from<$3 AND occupied.valid_until>greatest($2,b.valid_until))`, member, now, result.EffectiveServiceEnd)
	return err
}
