package devices

import (
	"context"
	"encoding/json"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"slices"
	"time"
)

// SyncDevice uses a session advisory lock, NOT a transaction, across external
// calls. It reads current desired state rather than replaying an obsolete event.
// Success means read-back evidence matches both device and entitlement versions.
func (s *Service) SyncDevice(ctx context.Context, id uuid.UUID) error {
	if s.Adapter == nil {
		return entitlement.ErrInvalid
	}
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,120))`, id.String()).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return db.ErrLeaseLost
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, e := conn.Exec(cleanup, `SELECT pg_advisory_unlock(hashtextextended($1,120))`, id.String())
		if e != nil {
			_ = conn.Conn().Close(cleanup)
		}
	}()
	var deployment, peer, status string
	var member uuid.UUID
	var version, revision decstr.U64
	var raw []byte
	err = conn.QueryRow(ctx, `SELECT d.deployment_id,d.peer_id,d.status,d.member_id,d.resource_version::text,e.revision::text,e.projection FROM member_devices d JOIN entitlements e ON e.member_id=d.member_id WHERE d.id=$1`, id).Scan(&deployment, &peer, &status, &member, &version, &revision, &raw)
	if err != nil {
		return db.MapError(err)
	}
	var r entitlement.Result
	if err = json.Unmarshal(raw, &r); err != nil {
		return err
	}
	allow := status == "authorized" && r.Allows(s.Entitlements.Time())
	preflight, err := s.Adapter.ObservePeer(ctx, deployment, peer)
	if err != nil {
		return err
	}
	var mapped bool
	if preflight.DeploymentID == deployment && preflight.ID == peer && preflight.OverlayIP.IsValid() {
		err = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id JOIN device_bindings b ON b.member_id=n.member_id WHERE n.member_id=$1 AND n.deployment_id=$2 AND n.external_user_id=$3 AND n.mapping_proof<>'' AND i.stable_claim_ref<>'' AND b.device_id=$4 AND b.overlay_scope=$5 AND b.overlay_ip=$6::inet AND b.valid_until='infinity')`, member, deployment, preflight.ExternalUserID, id, preflight.OverlayScope, preflight.OverlayIP.String()).Scan(&mapped)
		if err != nil {
			return err
		}
	}
	mappingLost := allow && !mapped
	allow = allow && mapped

	groups := []uuid.UUID{}
	if allow {
		groups = r.ExitGroups
	}
	if err = s.Adapter.SetAuthorization(ctx, deployment, peer, allow, groups); err != nil {
		return err
	}
	observed, err := s.Adapter.ObservePeer(ctx, deployment, peer)
	if err != nil {
		return err
	}
	if observed.DeploymentID != deployment || observed.ID != peer || observed.Authorized != allow || !sameGroups(observed.Groups, groups) {
		return entitlement.ErrConflict
	}

	if allow && !r.Allows(s.Entitlements.Time()) {
		return entitlement.ErrConflict
	}
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		var lockedMember uuid.UUID
		if e := tx.QueryRow(ctx, `SELECT id FROM members WHERE id=$1 FOR UPDATE`, member).Scan(&lockedMember); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `UPDATE member_devices d SET observed_authorized=$2,observed_at=clock_timestamp(),observed_version=$3::numeric,observed_entitlement_revision=$4::numeric WHERE d.id=$1 AND d.resource_version=$3::numeric AND EXISTS(SELECT 1 FROM entitlements e WHERE e.member_id=d.member_id AND e.revision=$4::numeric)`, id, allow, version.String(), revision.String())
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return entitlement.ErrConflict
		}
		if mappingLost {
			return nil
		}
		evidence, e := canonjson.Marshal(map[string]any{"device_id": id, "device_version": version, "entitlement_revision": revision, "authorized": allow, "observed_at": s.Entitlements.Time()})
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE operations SET status='running',steps=jsonb_set(steps,'{netbird}','{"required":true,"status":"succeeded","observed_ref":"netbird"}'),observed_refs=jsonb_set(observed_refs,'{netbird}',$3::jsonb) WHERE resource_type='device' AND resource_id=$1 AND desired_generation=$2::numeric AND status IN ('pending','running')`, id, version.String(), evidence)
		return e
	})
	if err != nil {
		return err
	}
	if mappingLost {
		return ErrUnmapped
	}
	return nil

}
func sameGroups(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	for _, id := range a {
		if !slices.Contains(b, id) {
			return false
		}
	}
	return true
}

// SyncMember handles member.entitlement_changed, including suspension, expiry,
// and closure; device statuses remain distinct from temporary entitlement loss.
func (s *Service) SyncMember(ctx context.Context, member uuid.UUID) error {
	rows, err := s.Pool.Query(ctx, `SELECT id FROM member_devices WHERE member_id=$1 ORDER BY id`, member)
	if err != nil {
		return err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err = s.SyncDevice(ctx, id); err != nil {
			return err
		}
	}

	return db.InTx(ctx, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		var locked uuid.UUID
		if e := tx.QueryRow(ctx, `SELECT id FROM members WHERE id=$1 FOR UPDATE`, member).Scan(&locked); e != nil {
			return e
		}
		var revision decstr.U64
		if e := tx.QueryRow(ctx, `SELECT revision::text FROM entitlements WHERE member_id=$1`, member).Scan(&revision); e != nil {
			return e
		}
		var stale bool
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM member_devices WHERE member_id=$1 AND (observed_version<>resource_version OR observed_entitlement_revision<>$2::numeric))`, member, revision.String()).Scan(&stale); e != nil {
			return e
		}
		if stale {
			return entitlement.ErrConflict
		}
		evidence, e := canonjson.Marshal(map[string]any{"member_id": member, "entitlement_revision": revision, "device_count": len(ids), "observed_at": s.Entitlements.Time()})
		if e != nil {
			return e
		}
		_, e = tx.Exec(ctx, `UPDATE operations SET status='running',steps=jsonb_set(steps,'{netbird}','{"required":true,"status":"succeeded","observed_ref":"netbird"}'),observed_refs=jsonb_set(observed_refs,'{netbird}',$3::jsonb) WHERE resource_type='member' AND resource_id=$1 AND desired_generation=$2::numeric AND status IN ('pending','running')`, member, revision.String(), evidence)
		return e
	})
}
