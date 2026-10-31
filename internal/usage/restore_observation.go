package usage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RestoreObservationStatus is the D-095 node-view and operation hold message.
const RestoreObservationStatus = "pending fresh observation after restore"

// RestoreObservationPrincipal identifies durable per-node restore cutoffs. These
// operations survive complete-recovery and audit retention, including for nodes
// the compiler has not visited during recovery.
const RestoreObservationPrincipal = "system:restore-observation"

const RestoreObservationField = "desired_state_restore_observation"

// holdDesiredStateAfterRestore runs while the ledger row is locked, before the
// restored control/worker services start (§17.3, D-095). Lock nodes before their
// operations, as the observation/compiler paths do. Compilers only read ledger
// state, so they never acquire the inverse node -> ledger write/share lock.
func holdDesiredStateAfterRestore(ctx context.Context, tx pgx.Tx, frozenAt time.Time) error {
	rows, err := tx.Query(ctx, `SELECT id FROM nodes
		WHERE lifecycle_state IN ('enrolling', 'registered') ORDER BY id FOR UPDATE`)
	if err != nil {
		return err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	drift, err := json.Marshal(api.StateDrift{
		Field:    RestoreObservationField,
		Desired:  presence.NewNullableWithValue("observation received after " + frozenAt.UTC().Format(time.RFC3339Nano)),
		Observed: presence.NewNullableWithValue(RestoreObservationStatus),
	})
	if err != nil {
		return err
	}
	for _, id := range ids {
		// This step is central receipt evidence, not agent_policy/apply evidence.
		// The observation writer therefore cannot complete it using restored ACKs.
		if _, err := tx.Exec(ctx, `INSERT INTO operations
			(kind, principal, resource_type, resource_id, request_id, resource_version,
			 status, steps, created_at, updated_at)
			VALUES ('node.update', $1, 'node', $2, $3, 1, 'pending',
			 jsonb_build_object('fresh_observation', jsonb_build_object(
			 'order', 1, 'required', true, 'status', 'pending', 'reason', $4::text,
			 'updated_at', $5::timestamptz)), $5, $5)`,
			RestoreObservationPrincipal, id, uuid.New(), RestoreObservationStatus, frozenAt); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE nodes SET drift =
			coalesce((SELECT jsonb_agg(d) FROM jsonb_array_elements(drift) d
			 WHERE d->>'field' <> $2), '[]'::jsonb) || jsonb_build_array($3::jsonb),
			resource_version = resource_version + 1 WHERE id = $1`,
			id, RestoreObservationField, drift); err != nil {
			return err
		}
	}
	return nil
}
