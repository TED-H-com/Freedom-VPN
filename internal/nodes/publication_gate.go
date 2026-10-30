package nodes

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const PublicationGatePrincipal = "system:publication-gate"

// QueuePublicationGateTx rechecks an existing route after compilation or a new
// observation (§5.2). The caller holds the node lock; terminal evidence is never reset.
func QueuePublicationGateTx(ctx context.Context, tx pgx.Tx, nodeID uuid.UUID, now time.Time, newGeneration bool) error {
	var generation decstr.U64
	var eligible bool
	if err := tx.QueryRow(ctx, `SELECT n.desired_config_generation::text,
		n.lifecycle_state='registered' AND e.publication_intent='published' AND EXISTS(
		SELECT 1 FROM external_bindings b WHERE b.resource_type='exit' AND b.resource_id=e.id AND b.provider='netbird')
		FROM nodes n JOIN exits e ON e.node_id=n.id WHERE n.id=$1`, nodeID).Scan(&generation, &eligible); err != nil {
		return err
	}
	if !eligible {
		return nil
	}
	var pending bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM operations WHERE resource_type='node' AND resource_id=$1 AND kind='node.publish' AND ((principal=$2 AND desired_generation=$3::numeric) OR (NOT $4::boolean AND principal<>$2)) AND status IN ('pending','running'))`, nodeID, PublicationGatePrincipal, generation.String(), newGeneration).Scan(&pending); err != nil {
		return err
	}
	if pending {
		return nil
	}
	// Obsolete queued generations cannot re-attach a route after a newer one.
	if _, err := tx.Exec(ctx, `UPDATE operations SET status='superseded',completed_at=$3,resource_version=coalesce(resource_version,0)+1 WHERE resource_type='node' AND resource_id=$1 AND principal=$2 AND status IN ('pending','running') AND desired_generation<$4::numeric`, nodeID, PublicationGatePrincipal, now, generation.String()); err != nil {
		return err
	}
	gen := uint64(generation)
	op, err := CreateOperation(ctx, tx, CreateOperationParams{Kind: OpKindPublish, Principal: PublicationGatePrincipal, ResourceID: nodeID, DesiredGeneration: &gen, Now: now})
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"operation_id": op.OperationId, "node_id": nodeID, "desired_generation": generation})
	if err != nil {
		return err
	}
	return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateType: "node", AggregateID: nodeID, AggregateVersion: generation.String(), EventType: "node.publication_changed", SchemaVersion: 1, Payload: payload})
}

// CheckPublicationDelivery prevents a new admitted binding from reaching a node
// while its older route is awaiting confirmed withdrawal. No external call runs
// in the read transaction; the durable publication job supplies the read-back.
func CheckPublicationDelivery(ctx context.Context, q Querier, state api.DesiredState) error {
	if state.PublicationIntent != api.PublicationIntentPublished || len(state.DeviceBindings) == 0 {
		return nil
	}
	var blocked bool
	if err := q.QueryRow(ctx, `SELECT e.publication_observed<>'withdrawn' AND EXISTS(
		SELECT 1 FROM operations o WHERE o.resource_type='node' AND o.resource_id=$1 AND o.principal=$2 AND o.desired_generation=$3::numeric AND o.status<>'succeeded' AND NOT EXISTS(SELECT 1 FROM operations done WHERE done.resource_type='node' AND done.resource_id=$1 AND done.principal=$2 AND done.desired_generation=$3::numeric AND done.status='succeeded'))
		FROM exits e WHERE e.node_id=$1`, state.NodeId, PublicationGatePrincipal, state.ConfigGeneration.String()).Scan(&blocked); err != nil {
		return err
	}
	if blocked {
		return apierr.New(apierr.DependencyUnavailable, "Publication route verification pending.")
	}
	return nil
}
