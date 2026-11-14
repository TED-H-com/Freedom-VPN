package leases

import (
	"context"
	"encoding/json"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// lockUsageLedgerMode holds the recovery fence until the surrounding transaction
// commits. The schema guarantees at most one ledger-state row.
func lockUsageLedgerMode(ctx context.Context, tx pgx.Tx) (usage.LedgerMode, error) {
	var mode usage.LedgerMode
	if err := tx.QueryRow(ctx, `SELECT mode FROM usage_ledger_state FOR SHARE`).Scan(&mode); err != nil {
		return "", apierr.Wrap(err, apierr.DependencyUnavailable, "usage ledger state unavailable")
	}
	switch mode {
	case usage.LedgerModeNormal, usage.LedgerModeFrozen, usage.LedgerModeReconciling:
		return mode, nil
	default:
		return "", apierr.New(apierr.DependencyUnavailable, "usage ledger state unavailable")
	}
}

func allowLeaseReplay(mode usage.LedgerMode, stored api.ServingLeaseEnvelope) error {
	if mode != usage.LedgerModeNormal && len(stored.Signed.Members) != 0 {
		return apierr.New(apierr.DependencyUnavailable, "usage ledger recovery in progress")
	}
	return nil
}

// ReadRenewal returns the immutable envelope for an authenticated identity's renewal without signing or extending it.
// D-088: its signed expiry can exceed the clamped serving_leases.expires_at;
// replay is issuance evidence, not a check of the authoritative central deadline.
func (s *Service) ReadRenewal(ctx context.Context, nodeID, incarnationID, renewalID uuid.UUID) (api.ServingLeaseEnvelope, error) {
	var out api.ServingLeaseEnvelope
	err := db.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		mode, err := lockUsageLedgerMode(ctx, tx)
		if err != nil {
			return err
		}
		var raw []byte
		if err := tx.QueryRow(ctx, `SELECT envelope FROM serving_leases WHERE node_id=$1 AND incarnation_id=$2 AND renewal_request_id=$3`, nodeID, incarnationID, renewalID).Scan(&raw); err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &out); err != nil {
			return err
		}
		return allowLeaseReplay(mode, out)
	})
	if err != nil {
		return api.ServingLeaseEnvelope{}, err
	}
	return out, err
}
