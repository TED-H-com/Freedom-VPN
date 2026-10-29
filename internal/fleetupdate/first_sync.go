package fleetupdate

import (
	"context"

	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Reserved system actor for internal snapshots, never taken from request JSON.
const firstSyncActor = "00000000-0000-0000-0000-000000000003"

// firstSync preserves §13.1's original all_nodes denominator. A separate system
// batch supplies the same sealed target and composite identity FK as admin work.
func (s *Service) firstSync(ctx context.Context, tx pgx.Tx, n node, release uuid.UUID, generation decstr.U64) error {
	batch, op, target := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	if _, e := tx.Exec(ctx, `INSERT INTO operations(id,kind,principal,resource_type,resource_id,desired_generation,request_id) VALUES($1,'update_batch.create',$2,'update_batch',$3,$4::numeric,$5)`, op, firstSyncActor, batch, generation.String(), commandtx.RequestID(ctx)); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, `INSERT INTO update_batches(id,operation_id,target_release_id,scope,created_by,target_count,generation,reason,online_at_snapshot,first_sync_node_id)
 SELECT $1,$2,$3,'first_sync',$4,1,generation,'first_sync',1,$5 FROM fleet_config`, batch, op, release, firstSyncActor, n.id); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, `INSERT INTO update_targets(id,batch_id,node_id,incarnation_at_snapshot,execution_incarnation_id,target_generation,online_at_snapshot) VALUES($1,$2,$3,$4,$4,$5::numeric,true)`, target, batch, n.id, n.incarnation, generation.String()); e != nil {
		return e
	}
	if e := s.newJob(ctx, tx, n, &target, release, generation, 1); e != nil {
		return e
	}
	return releases.Audit(ctx, tx, firstSyncActor, "system", "fleet.first_sync_created", batch.String(), &op, map[string]any{"node_id": n.id, "release_id": release})
}
