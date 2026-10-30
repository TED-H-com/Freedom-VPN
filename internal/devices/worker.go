package devices

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
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
func (s *Service) SyncDevice(ctx context.Context, id uuid.UUID) (retErr error) {
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
	defer func() {
		// Another worker already holds this device, or the row moved under us.
		// The dispatcher retries; leaving the step pending/running is not a failure.
		if retErr == nil || retryableContention(retErr) {
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pgx.BeginFunc(cleanup, conn, func(tx pgx.Tx) error { return failOperations(cleanup, tx, api.ResourceTypeMemberDevice, id, version) })
	}()
	var r entitlement.Result
	if err = json.Unmarshal(raw, &r); err != nil {
		return err
	}
	var withinLimit bool
	if err = conn.QueryRow(ctx, `SELECT $1::uuid IN (`+entitlement.AuthorizedDeviceIDsSQL+` AND ranked.member_id=$2)`, id, member).Scan(&withinLimit); err != nil {
		return err
	}
	centralAllow := status == "authorized" && withinLimit && r.Allows(s.Entitlements.Time())
	allow := centralAllow
	preflight, err := s.Adapter.ObservePeer(ctx, deployment, peer)
	if err != nil {
		return err
	}
	var mapped bool
	if !preflight.IdentityUnmapped && preflight.DeploymentID == deployment && preflight.ID == peer && preflight.OverlayIP.IsValid() {
		err = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM netbird_user_bindings n JOIN identity_bindings i ON i.id=n.identity_binding_id AND i.member_id=n.member_id JOIN device_bindings b ON b.member_id=n.member_id WHERE n.member_id=$1 AND n.deployment_id=$2 AND n.external_user_id=$3 AND n.mapping_proof='shared_issuer_public_subject:' || i.issuer AND i.subject=n.external_user_id AND n.revision>0 AND i.stable_claim_ref='sub' AND i.binding_method='oidc' AND n.verified_at<=clock_timestamp() AND i.verified_at<=clock_timestamp() AND b.device_id=$4 AND b.overlay_scope=$5 AND b.overlay_ip=$6::inet AND b.valid_from<=clock_timestamp() AND b.valid_until>clock_timestamp())`, member, deployment, preflight.ExternalUserID, id, preflight.OverlayScope, preflight.OverlayIP.String()).Scan(&mapped)
		if err != nil {
			return err
		}
	}
	mappingLost := allow && !mapped
	allow = allow && mapped
	// D-076: a restored observed_authorized row is not a current entitlement.
	// Revoked and closed devices never reach this branch. A fence row, once
	// written, keeps the peer withdrawn until explicit re-authorization.
	var recordFence, recordBlock bool
	var fenceGroups []uuid.UUID
	if status == "authorized" {
		var already bool
		if err = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM device_authorization_fences WHERE device_id=$1)`, id).Scan(&already); err != nil {
			return err
		}
		if already {
			allow = false
		} else if centralAllow {
			block, fence, groups, e := s.detectRestoredWithdrawal(ctx, conn, id, deployment, peer, preflight, r.ExitGroups)
			if e != nil {
				return e
			}
			// A newer central revocation while status is still authorized is not
			// an external withdrawal, and it must not grant on a later sync
			// after the read-back is stored as withdrawn.
			if block || fence {
				allow = false
			}
			recordFence = fence
			recordBlock = block
			fenceGroups = groups
		}
	}

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
	if observed.DeploymentID != deployment || observed.ID != peer || observed.Authorized != allow || !sameGroups(observed.Groups, groups) || allow && (observed.IdentityUnmapped || observed.ExternalUserID != preflight.ExternalUserID || observed.OverlayIP != preflight.OverlayIP || observed.OverlayScope != preflight.OverlayScope) {
		return entitlement.ErrConflict
	}

	if allow && !r.Allows(s.Entitlements.Time()) {
		return entitlement.ErrConflict
	}
	var fencedNow bool
	var ledgerMode string
	err = pgx.BeginFunc(ctx, conn, func(tx pgx.Tx) error {
		var lockedMember uuid.UUID
		if e := tx.QueryRow(ctx, `SELECT id FROM members WHERE id=$1 FOR UPDATE`, member).Scan(&lockedMember); e != nil {
			return e
		}
		tag, e := tx.Exec(ctx, `INSERT INTO device_authorization_observations(device_id,observed_authorized,observed_at,observed_version,observed_entitlement_revision)
 SELECT d.id,$2,clock_timestamp(),$3::numeric,$4::numeric FROM member_devices d
 WHERE d.id=$1 AND d.resource_version=$3::numeric AND EXISTS(SELECT 1 FROM entitlements e WHERE e.member_id=d.member_id AND e.revision=$4::numeric)
 ON CONFLICT(device_id) DO UPDATE SET observed_authorized=excluded.observed_authorized,observed_at=excluded.observed_at,observed_version=excluded.observed_version,observed_entitlement_revision=excluded.observed_entitlement_revision`, id, allow, version.String(), revision.String())
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return entitlement.ErrConflict
		}
		if recordFence {
			inserted, mode, e := recordExternalWithdrawal(ctx, tx, id, member, peer, fenceGroups)
			if e != nil {
				return e
			}
			fencedNow = inserted
			ledgerMode = mode
		} else if recordBlock {
			if _, e = ensureFence(ctx, tx, id, nil); e != nil {
				return e
			}
		}
		if mappingLost {
			return nil
		}
		evidence, e := canonjson.Marshal(map[string]any{"device_id": id, "device_version": version, "entitlement_revision": revision, "authorized": allow, "observed_at": s.Entitlements.Time()})
		if e != nil {
			return e
		}
		return observeOperations(ctx, tx, api.ResourceTypeMemberDevice, id, version, evidence, s.Entitlements.Time())
	})
	if err != nil {
		return err
	}
	if fencedNow {
		s.announceFence(ctx, id, member, peer, ledgerMode)
	}
	if mappingLost {
		return ErrUnmapped
	}
	return nil
}

// retryableContention reports overlap with another worker. The outbox
// dispatcher retries these; they must not mark an operation step failed.
func retryableContention(err error) bool {
	if err == nil {
		return false
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		saw := false
		for _, part := range joined.Unwrap() {
			if part == nil {
				continue
			}
			saw = true
			if !retryableContention(part) {
				return false
			}
		}
		return saw
	}
	if inner := errors.Unwrap(err); inner != nil {
		return retryableContention(inner)
	}
	return errors.Is(err, db.ErrLeaseLost) || errors.Is(err, entitlement.ErrConflict)
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
// and closure; every device is attempted even if another adapter call fails.
// A non-blocking member lock (same namespace as the per-device lock, distinct
// key) keeps two workers from interleaving one member's devices. Losing that
// lock is contention, not an operation failure.
func (s *Service) SyncMember(ctx context.Context, member uuid.UUID) (retErr error) {
	var initial decstr.U64
	if err := s.Pool.QueryRow(ctx, `SELECT revision::text FROM entitlements WHERE member_id=$1`, member).Scan(&initial); err != nil {
		return db.MapError(err)
	}
	defer func() {
		if retErr == nil || retryableContention(retErr) {
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = db.InTx(cleanup, s.Pool, db.TxOptions{}, func(tx pgx.Tx) error { return failOperations(cleanup, tx, api.ResourceTypeMember, member, initial) })
	}()
	conn, err := s.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	lockKey := "member:" + member.String()
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,120))`, lockKey).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return db.ErrLeaseLost
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, e := conn.Exec(cleanup, `SELECT pg_advisory_unlock(hashtextextended($1,120))`, lockKey)
		if e != nil {
			_ = conn.Conn().Close(cleanup)
		}
	}()
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
		err = errors.Join(err, s.SyncDevice(ctx, id))
	}
	if err != nil {
		return err
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
		if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM member_devices d LEFT JOIN device_authorization_observations o ON o.device_id=d.id WHERE d.member_id=$1 AND (o.observed_version IS DISTINCT FROM d.resource_version OR o.observed_entitlement_revision IS DISTINCT FROM $2::numeric))`, member, revision.String()).Scan(&stale); e != nil {
			return e
		}
		if stale {
			return entitlement.ErrConflict
		}
		evidence, e := canonjson.Marshal(map[string]any{"member_id": member, "entitlement_revision": revision, "device_count": len(ids), "observed_at": s.Entitlements.Time()})
		if e != nil {
			return e
		}
		return observeOperations(ctx, tx, api.ResourceTypeMember, member, revision, evidence, s.Entitlements.Time())
	})
}

// Preserve an immutable audit evidence row before transitioning the operation.
// A successful NetBird step never fabricates the separate node enforcement ACK.
func observeOperations(ctx context.Context, tx pgx.Tx, resource api.ResourceType, id uuid.UUID, revision decstr.U64, evidence []byte, now time.Time) error {
	if !resource.Valid() {
		return entitlement.ErrInvalid
	}
	rows, err := tx.Query(ctx, `SELECT o.id FROM operations o WHERE ((resource_type=$1 AND resource_id=$2) OR ($1=$5 AND resource_type=$6 AND resource_id IN (SELECT id FROM entitlement_grants WHERE member_id=$2))) AND desired_generation=$3::numeric AND status IN ('pending','running','failed','partially_succeeded') AND steps->'netbird'->>'status' IN ('pending','running','failed') AND (kind<>$7 OR NOT EXISTS(SELECT 1 FROM entitlement_grants g WHERE g.operation_id=o.id AND g.valid_from>$4)) ORDER BY o.id FOR UPDATE`, resource, id, revision.String(), now, api.ResourceTypeMember, api.ResourceTypeEntitlementGrant, api.OperationKindMemberGrant)
	if err != nil {
		return err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var op uuid.UUID
		if err = rows.Scan(&op); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, op)
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(ids) == 0 {
		return err
	}
	evidenceID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	if err = db.AppendAudit(ctx, tx, db.AuditEvent{ID: evidenceID, Actor: "00000000-0000-0000-0000-000000000003", ActorType: string(api.AuditEventActorTypeSystem), Action: "netbird.observed", Resource: string(resource) + ":" + id.String(), SchemaVersion: 1, Before: []byte(`{}`), After: evidence, Result: string(api.AuditEventResultSucceeded)}); err != nil {
		return err
	}
	for _, op := range ids {
		var failed bool
		if err = tx.QueryRow(ctx, `SELECT steps->'netbird'->>'status'='failed' FROM operations WHERE id=$1`, op).Scan(&failed); err != nil {
			return err
		}
		if failed {
			if err = entitlement.TransitionOperation(ctx, tx, op, "netbird", api.OperationStepStatusRunning, nil); err != nil {
				return err
			}
		}
		if err = entitlement.TransitionOperation(ctx, tx, op, "netbird", api.OperationStepStatusSucceeded, &operations.Evidence{Reference: "audit_event:" + evidenceID.String(), ObservedAt: now}); err != nil {
			return err
		}
	}
	return nil
}

func failOperations(ctx context.Context, tx pgx.Tx, resource api.ResourceType, id uuid.UUID, revision decstr.U64) error {
	if !resource.Valid() {
		return entitlement.ErrInvalid
	}
	rows, err := tx.Query(ctx, `SELECT id FROM operations WHERE ((resource_type=$1 AND resource_id=$2) OR ($1=$5 AND resource_type=$4 AND resource_id IN (SELECT id FROM entitlement_grants WHERE member_id=$2))) AND desired_generation=$3::numeric AND status IN ('pending','running','partially_succeeded') AND steps->'netbird'->>'status' IN ('pending','running') ORDER BY id FOR UPDATE`, resource, id, revision.String(), api.ResourceTypeEntitlementGrant, api.ResourceTypeMember)
	if err != nil {
		return err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var op uuid.UUID
		if err = rows.Scan(&op); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, op)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, op := range ids {
		if err = entitlement.TransitionOperation(ctx, tx, op, "netbird", api.OperationStepStatusFailed, nil); err != nil {
			return err
		}
	}
	return nil
}
