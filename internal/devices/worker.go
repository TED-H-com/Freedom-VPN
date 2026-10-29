package devices

import (
	"context"
	"encoding/json"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
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
	tag, err := conn.Exec(ctx, `UPDATE member_devices d SET observed_authorized=$2,observed_at=clock_timestamp(),observed_version=$3::numeric WHERE d.id=$1 AND d.resource_version=$3::numeric AND EXISTS(SELECT 1 FROM entitlements e WHERE e.member_id=d.member_id AND e.revision=$4::numeric)`, id, allow, version.String(), revision.String())
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return entitlement.ErrConflict
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
	return nil
}
