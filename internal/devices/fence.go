package devices

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"
)

// externalWithdrawalAudit is written once when a previously confirmed device
// is left withdrawn. Field names must not contain "authorization": the audit
// redactor drops those keys.
const externalWithdrawalAudit = "device.authorization.external_withdrawal"

// usageLedgerStateID is the usage_ledger_state singleton (§17.3).
const usageLedgerStateID = "00000000-0000-0000-0000-000000000130"

// fenceAccounting counts fence decisions while the usage ledger is frozen or
// reconciling. Nil when the worker did not register a collector.
type fenceAccounting struct {
	decisions prometheus.Counter
}

// RegisterRestoreFences registers the restore-summary counter on the worker
// registry. A nil registry leaves the summary as a log line only.
func (s *Service) RegisterRestoreFences(reg prometheus.Registerer) error {
	if s == nil || reg == nil || s.fences != nil {
		return nil
	}
	c := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "fvpn_device_restore_fence_decisions_total",
		Help: "Devices newly recorded as authorization-unconfirmed while usage_ledger_state.mode is frozen or reconciling.",
	})
	if err := reg.Register(c); err != nil {
		return err
	}
	s.fences = &fenceAccounting{decisions: c}
	return nil
}

func (s *Service) logger() *slog.Logger {
	if s != nil && s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

// peerWithdrawn reports that the peer is not authorized, or that it is
// authorized in none of the member's granted exit groups.
func peerWithdrawn(p Peer, granted []uuid.UUID) bool {
	if !p.Authorized {
		return true
	}
	for _, g := range p.Groups {
		if slices.Contains(granted, g) {
			return false
		}
	}
	return true
}

func groupIDs(ids []uuid.UUID) ([]uuid.UUID, []string) {
	out := make([]uuid.UUID, 0, len(ids))
	text := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		out = append(out, id)
		text = append(text, id.String())
	}
	slices.SortFunc(out, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
	slices.Sort(text)
	return out, text
}

// detectRestoredWithdrawal decides whether a previously confirmed device must
// stay withdrawn. block means a newer central revocation explains the
// withdrawal, so the sync must not grant and must not label it external.
// fence means both reads show an unexplained withdrawal. A quota device.revoke
// (status pending) is central but not a permanent revocation: the caller keeps
// the normal grant when the device is authorized again.
func (s *Service) detectRestoredWithdrawal(ctx context.Context, q rowQuery, id uuid.UUID, deployment, peer string, preflight Peer, granted []uuid.UUID) (block, fence bool, groups []uuid.UUID, err error) {
	var observedAuthorized bool
	var observedAt time.Time
	err = q.QueryRow(ctx, `SELECT observed_authorized, observed_at FROM device_authorization_observations WHERE device_id=$1`, id).Scan(&observedAuthorized, &observedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, nil, nil
		}
		return false, false, nil, err
	}
	if !observedAuthorized || !peerWithdrawn(preflight, granted) {
		return false, false, nil, nil
	}
	resource := string(api.ResourceTypeMemberDevice) + ":" + id.String()
	var permanent, explained bool
	err = q.QueryRow(ctx, `SELECT
 EXISTS(SELECT 1 FROM operations WHERE resource_type=$3 AND resource_id=$1 AND kind=$4 AND created_at>$2)
  OR EXISTS(SELECT 1 FROM audit_events WHERE resource=$5 AND action='device.revoke' AND occurred_at>$2 AND after_diff->>'status'='revoked'),
 EXISTS(SELECT 1 FROM operations WHERE resource_type=$3 AND resource_id=$1 AND kind=$4 AND created_at>$2)
  OR EXISTS(SELECT 1 FROM audit_events WHERE resource=$5 AND action='device.revoke' AND occurred_at>$2)`,
		id, observedAt, api.ResourceTypeMemberDevice, api.OperationKindMemberDeviceRevoke, resource).Scan(&permanent, &explained)
	if err != nil {
		return false, false, nil, err
	}
	if permanent {
		return true, false, nil, nil
	}
	if explained {
		return false, false, nil, nil
	}
	second, err := s.Adapter.ObservePeer(ctx, deployment, peer)
	if err != nil {
		return false, false, nil, err
	}
	if second.DeploymentID != deployment || second.ID != peer {
		return false, false, nil, entitlement.ErrConflict
	}
	if !peerWithdrawn(second, granted) {
		return false, false, nil, nil
	}
	groups, _ = groupIDs(second.Groups)
	return false, true, groups, nil
}

type rowQuery interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func ensureFence(ctx context.Context, tx pgx.Tx, id uuid.UUID, groups []uuid.UUID) (bool, error) {
	if groups == nil {
		groups = []uuid.UUID{}
	}
	tag, err := tx.Exec(ctx, `INSERT INTO device_authorization_fences(device_id,fenced_at,observed_group_ids) VALUES($1,clock_timestamp(),$2) ON CONFLICT(device_id) DO NOTHING`, id, groups)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// recordExternalWithdrawal inserts the fence and audits once. Repeated calls
// for an existing row do not append another audit. mode is the usage ledger
// mode read in the same transaction; callers announce it after commit.
func recordExternalWithdrawal(ctx context.Context, tx pgx.Tx, id, member uuid.UUID, peer string, groups []uuid.UUID) (inserted bool, mode string, err error) {
	inserted, err = ensureFence(ctx, tx, id, groups)
	if err != nil || !inserted {
		return false, "", err
	}
	_, text := groupIDs(groups)
	if text == nil {
		text = []string{}
	}
	if err = entitlement.Audit(ctx, tx, entitlement.Actor{ID: audit.SystemActor.ID, Role: string(audit.SystemActor.Type)}, externalWithdrawalAudit, api.ResourceTypeMemberDevice, id, nil, map[string]any{
		"member_id":       member,
		"peer_id":         peer,
		"observed_groups": text,
		"reason":          "external_withdrawal",
	}); err != nil {
		return false, "", err
	}
	if err = tx.QueryRow(ctx, `SELECT mode FROM usage_ledger_state WHERE id=$1`, usageLedgerStateID).Scan(&mode); err != nil {
		return false, "", err
	}
	return true, mode, nil
}

func (s *Service) announceFence(ctx context.Context, id, member uuid.UUID, peer, mode string) {
	if mode == "frozen" || mode == "reconciling" {
		s.logger().WarnContext(ctx, "device authorization unconfirmed during ledger restore", "device_id", id, "member_id", member, "peer_id", peer, "ledger_mode", mode)
		if s.fences != nil && s.fences.decisions != nil {
			s.fences.decisions.Inc()
		}
		return
	}
	s.logger().InfoContext(ctx, "device authorization unconfirmed", "device_id", id, "member_id", member, "peer_id", peer, "ledger_mode", mode)
}

// deleteAuthorizationFence removes the fence without touching the read-back.
// Revocation uses this. Re-authorization also refreshes the observation.
func deleteAuthorizationFence(ctx context.Context, tx pgx.Tx, device uuid.UUID) error {
	_, err := tx.Exec(ctx, `DELETE FROM device_authorization_fences WHERE device_id=$1`, device)
	return err
}

// clearAuthorizationFence drops the fence and marks the stored read-back
// unconfirmed so the next sync grants instead of fencing again. cleared is
// false when the device was not fenced.
func clearAuthorizationFence(ctx context.Context, tx pgx.Tx, device uuid.UUID) (bool, error) {
	tag, err := tx.Exec(ctx, `DELETE FROM device_authorization_fences WHERE device_id=$1`, device)
	if err != nil || tag.RowsAffected() != 1 {
		return false, err
	}
	_, err = tx.Exec(ctx, `UPDATE device_authorization_observations SET observed_authorized=false, observed_at=clock_timestamp() WHERE device_id=$1`, device)
	return true, err
}
