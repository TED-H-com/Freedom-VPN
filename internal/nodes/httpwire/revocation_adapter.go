package httpwire

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RevocationAdapter adds required identity cleanup before the exits executor
// finalizes withdrawal (§5.2). All external calls remain outside transactions.
type RevocationAdapter struct {
	exits.NetBirdAdapter
	runtime   *Runtime
	client    *netbird.Client
	operation api.Operation
}

func NewRevocationAdapter(r *Runtime, client *netbird.Client, base exits.NetBirdAdapter, op api.Operation) *RevocationAdapter {
	return &RevocationAdapter{NetBirdAdapter: base, runtime: r, client: client, operation: op}
}

func (a *RevocationAdapter) WithdrawExit(ctx context.Context, deployment string, exitID uuid.UUID, externalID string) (ref string, err error) {
	defer publicationError(&err)
	deployment, err = a.withdrawalDeployment(ctx, deployment)
	if err != nil {
		return "", err
	}
	ref, err = a.NetBirdAdapter.WithdrawExit(ctx, deployment, exitID, externalID)
	if err != nil {
		return "", err
	}
	done := false
	err = commandtx.InTx(ctx, a.runtime.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		step, e := a.lockCleanupStep(ctx, tx)
		done = step.Status == string(api.OperationStepStatusSucceeded)
		return e
	})
	if err != nil || done {
		return ref, err
	}
	keys, peers, cleanupErr := a.cleanup(ctx)
	if err = a.recordCleanup(ctx, keys, peers, cleanupErr); err != nil {
		return "", err
	}
	return ref, cleanupErr
}

func (a *RevocationAdapter) ReadExitState(ctx context.Context, deployment string, exitID uuid.UUID, externalID string) (out exits.ObservedExitState, err error) {
	defer publicationError(&err)
	deployment, err = a.withdrawalDeployment(ctx, deployment)
	if err != nil {
		return out, err
	}
	return a.NetBirdAdapter.ReadExitState(ctx, deployment, exitID, externalID)
}

func (a *RevocationAdapter) withdrawalDeployment(ctx context.Context, deployment string) (string, error) {
	// Enrollment can be revoked before any peer has been observed. The exits
	// executor uses a sentinel then; use the configured deployment for read-back.
	if deployment == exits.DefaultDeploymentID {
		var bound *string
		if err := a.runtime.Pool.QueryRow(ctx, `SELECT i.deployment_id FROM nodes n LEFT JOIN node_incarnations i ON i.id=n.current_incarnation_id WHERE n.id=$1`, a.operation.Resource.Id).Scan(&bound); err != nil {
			return "", err
		}
		if bound == nil {
			return a.runtime.Config.NetBirdDeployment, nil
		}
	}
	return deployment, nil
}

func (a *RevocationAdapter) cleanup(ctx context.Context) ([]uuid.UUID, []uuid.UUID, error) {
	if a.client == nil {
		return nil, nil, errors.New("identity cleanup adapter required")
	}
	keyRows, err := a.runtime.Pool.Query(ctx, `SELECT e.operation_id FROM node_enrollments e JOIN node_incarnations i ON i.id=e.incarnation_id WHERE e.node_id=$1 AND i.revoked_at IS NOT NULL ORDER BY e.operation_id`, a.operation.Resource.Id)
	if err != nil {
		return nil, nil, err
	}
	keys, err := pgx.CollectRows(keyRows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, nil, err
	}
	for _, key := range keys {
		// The client locates the stable enrollment key and reads back revoked
		// or absent, including recovery after a lost creation response.
		if err = a.client.RevokeNodeSetupKey(ctx, key.String()); err != nil {
			return nil, nil, err
		}
	}
	type peer struct {
		id         uuid.UUID
		remote     string
		deployment string
	}
	rows, err := a.runtime.Pool.Query(ctx, `SELECT id,peer_id,deployment_id FROM node_incarnations WHERE node_id=$1 AND revoked_at IS NOT NULL AND peer_id IS NOT NULL ORDER BY id`, a.operation.Resource.Id)
	if err != nil {
		return nil, nil, err
	}
	peers, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (p peer, err error) {
		err = row.Scan(&p.id, &p.remote, &p.deployment)
		return
	})
	if err != nil {
		return nil, nil, err
	}
	confirmed := []uuid.UUID{}
	for _, peer := range peers {
		if peer.deployment != a.runtime.Config.NetBirdDeployment {
			return nil, nil, errors.New("identity cleanup deployment mismatch")
		}
		binding := "node:" + a.operation.Resource.Id.String() + ":" + peer.id.String()
		if _, err = a.client.EnsureNodePeer(ctx, netbird.NodePeer{NodeID: a.operation.Resource.Id.String(), IncarnationID: peer.id.String(), PeerID: peer.remote}); err != nil && !netbird.IsKind(err, netbird.NotFound) {
			return nil, nil, err
		}
		if err = a.client.DeletePeer(ctx, binding); err != nil {
			return nil, nil, err
		}
		confirmed = append(confirmed, peer.id)
	}
	return keys, confirmed, nil
}

func (a *RevocationAdapter) lockCleanupStep(ctx context.Context, tx pgx.Tx) (step nodes.StoredStep, err error) {
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT steps->$2 FROM operations WHERE id=$1 AND resource_type='node' AND resource_id=$3 AND kind IN ('node.revoke','node.retire') AND status NOT IN ('succeeded','superseded') FOR UPDATE`, a.operation.OperationId, nodes.StepNetBirdIdentityCleanup, a.operation.Resource.Id).Scan(&raw)
	if err != nil {
		return step, err
	}
	if err = json.Unmarshal(raw, &step); err != nil {
		return step, err
	}
	if !step.Required {
		return step, errors.New("required identity cleanup step missing")
	}
	return step, nil
}

func (a *RevocationAdapter) recordCleanup(ctx context.Context, keys, peers []uuid.UUID, cleanupErr error) error {
	return commandtx.InTx(ctx, a.runtime.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
		step, err := a.lockCleanupStep(ctx, tx)
		if err != nil || step.Status == string(api.OperationStepStatusSucceeded) {
			return err
		}
		now := time.Now().UTC()
		step.UpdatedAt = now
		step.ObservedRef, step.ObservedAt = "", nil
		step.ErrorCode, step.ErrorMessage = nil, nil
		var ref string
		if cleanupErr != nil {
			code, message := string(api.ErrorCodeDEPENDENCYUNAVAILABLE), "NetBird identity cleanup unconfirmed"
			step.Status, step.ErrorCode, step.ErrorMessage = string(api.OperationStepStatusFailed), &code, &message
		} else {
			evidence := uuid.Must(uuid.NewV7())
			err = audit.Append(ctx, tx, db.AuditEvent{ID: evidence, Actor: nodes.SystemActorUUID.String(), ActorType: string(api.AuditEventActorTypeSystem), Action: "node.identity_withdrawn", Resource: "node:" + a.operation.Resource.Id.String(), OperationID: &a.operation.OperationId, Result: string(api.AuditEventResultSucceeded)}, nil,
				map[string]any{"enrollment_operations": keys, "peer_incarnations": peers}, "enrollment_operations", "peer_incarnations")
			if err != nil {
				return err
			}
			step.Status, step.ObservedRef, step.ObservedAt = string(api.OperationStepStatusSucceeded), nodes.StepNetBirdIdentityCleanup, &now
			ref = "audit_event:" + evidence.String()
		}
		raw, err := canonjson.Marshal(step)
		if err != nil {
			return err
		}
		// Preserve the executor's step and reference representation. Only this
		// required step changes; the executor owns final operation status.
		_, err = tx.Exec(ctx, `UPDATE operations SET steps=jsonb_set(steps,ARRAY[$2],$3::jsonb),observed_refs=CASE WHEN $4::text='' THEN observed_refs-$2 ELSE jsonb_set(observed_refs,ARRAY[$2],to_jsonb($4::text)) END,resource_version=coalesce(resource_version,0)+1 WHERE id=$1`, a.operation.OperationId, nodes.StepNetBirdIdentityCleanup, raw, ref)
		return err
	})
}
