package exits

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) publicationGateCause(ctx context.Context, nodeID uuid.UUID, generation uint64) (string, error) {
	return publicationGateCause(ctx, s.pool, nodeID, generation, s.now().UTC())
}

func publicationGateCause(ctx context.Context, q nodes.Querier, nodeID uuid.UUID, generation uint64, now time.Time) (string, error) {
	var raw []byte
	err := q.QueryRow(ctx, `SELECT o.report_payload FROM node_observations o JOIN nodes n ON n.id=o.node_id AND n.current_incarnation_id=o.incarnation_id AND n.last_observed_at=o.observed_at WHERE n.id=$1 ORDER BY o.received_at DESC,o.id DESC LIMIT 1`, nodeID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "stale_generation", nil
	}
	if err != nil {
		return "", err
	}
	var report api.ObservationReport
	if json.Unmarshal(raw, &report) != nil {
		return "readiness_unconfirmed", nil
	}
	return nodes.PublicationDataPathFailure(ctx, q, report, generation, now)
}

// failGatedPublish confirms route withdrawal before recording a closed gate.
// A failed external write/read-back leaves error, never an invented withdrawn state.
func (s *Service) failGatedPublish(ctx context.Context, opID, nodeID uuid.UUID, step, code, message, cause string) (api.Operation, error) {
	empty, err := nodes.PublicationAttachmentEmpty(ctx, s.pool, nodeID)
	if err != nil {
		return api.Operation{}, err
	}
	var populated bool
	if empty {
		if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM nodes n JOIN desired_state_envelopes e ON e.node_id=n.id AND e.incarnation_id=n.current_incarnation_id WHERE n.id=$1 AND e.generation=n.desired_config_generation AND jsonb_array_length(e.envelope->'signed'->'device_bindings')>0)`, nodeID).Scan(&populated); err != nil {
			return api.Operation{}, err
		}
	}
	if !empty || !populated {
		// Established populated routes retain pre-r11 failure semantics. Node-side
		// generation-scoped enforcement covers policy/device changes during apply.
		return s.failPublishStep(ctx, opID, nodeID, step, code, message, cause)
	}
	var exitID uuid.UUID
	if err := s.pool.QueryRow(ctx, `SELECT id FROM exits WHERE node_id=$1`, nodeID).Scan(&exitID); err != nil {
		return api.Operation{}, err
	}
	deployment, err := s.lookupDeploymentID(ctx, nodeID)
	if err != nil {
		return api.Operation{}, err
	}
	ref, err := s.lookupExternalBinding(ctx, exitID, deployment)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return api.Operation{}, err
	}
	withdrawn := false
	if ref != "" {
		if s.netbird == nil {
			return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), "adapter_unavailable", cause)
		}
		state, err := s.netbird.ReadExitState(ctx, deployment, exitID, ref)
		if err != nil {
			return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), PublicationFailureCause(err, "readback"), cause)
		}
		if state.Exists && state.Enabled {
			if _, err = s.netbird.WithdrawExit(ctx, deployment, exitID, ref); err != nil {
				return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), PublicationFailureCause(err, "readback"), cause)
			}
			state, err = s.netbird.ReadExitState(ctx, deployment, exitID, ref)
			if err != nil {
				return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), PublicationFailureCause(err, "readback"), cause)
			}
			if state.Exists && state.Enabled {
				return s.failPublishStep(ctx, opID, nodeID, nodes.StepNetBirdRouteReadback, string(apierr.DependencyUnavailable), "route_contract_mismatch", cause)
			}
		}
		withdrawn = true
	}
	if _, err = s.failPublishStepOutcome(ctx, opID, nodeID, step, code, message, withdrawn, cause); err != nil {
		return api.Operation{}, err
	}
	if withdrawn {
		err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
			// Invalidate the old attached-route evidence so a later observation
			// cannot complete this operation without a new route read-back.
			_, err := tx.Exec(ctx, `UPDATE operations SET steps=jsonb_set(steps,'{netbird_route_readback}',jsonb_build_object('order',2,'required',true,'status','pending','reason',$2::text,'updated_at',$3::timestamptz)),observed_refs=(observed_refs-'netbird_route_readback')||jsonb_build_object('publication_gate_withdrawal',$4::text),resource_version=coalesce(resource_version,0)+1 WHERE id=$1 AND status='failed'`, opID, cause, s.now().UTC(), fmt.Sprintf("netbird:route:%s:withdrawn", ref))
			if err != nil {
				return err
			}
			return s.publicationObservedTx(ctx, tx, opID, nodeID, "withdrawn", cause)
		})
		if err != nil {
			return api.Operation{}, err
		}
	}
	return nodes.GetOperation(ctx, s.pool, opID)
}

// publicationObservedTx audits only confirmed state transitions, with bounded
// fields and the existing node.publish action; it never edits publication intent.
func (s *Service) publicationObservedTx(ctx context.Context, tx pgx.Tx, opID, nodeID uuid.UUID, observed, reason string) error {
	var exitID uuid.UUID
	var before, intent string
	if err := tx.QueryRow(ctx, `SELECT id,publication_observed,publication_intent FROM exits WHERE node_id=$1 FOR UPDATE`, nodeID).Scan(&exitID, &before, &intent); err != nil {
		return err
	}
	if before == observed {
		return nil
	}
	if _, err := tx.Exec(ctx, `UPDATE exits SET publication_observed=$2,resource_version=resource_version+1 WHERE id=$1`, exitID, observed); err != nil {
		return err
	}
	event, err := audit.NewEvent(audit.Entry{Actor: audit.SystemActor, Action: "node.publish", Resource: api.ResourceRef{Type: api.ResourceTypeExit, Id: exitID}, OperationID: &opID, Result: api.AuditEventResultSucceeded,
		Before: map[string]any{"publication_observed": before, "publication_intent": intent}, After: map[string]any{"publication_observed": observed, "publication_intent": intent, "reason": reason}, Fields: []string{"publication_observed", "publication_intent", "reason"}})
	if err != nil {
		return err
	}
	return audit.AppendEvent(ctx, tx, event)
}

// lockPublication serializes external route changes without holding an SQL
// transaction across management I/O. A competing job retries under its claim.
func (s *Service) lockPublication(ctx context.Context, nodeID uuid.UUID) (func(), error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	key := "fvpn:publication:" + nodeID.String()
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1,0))`, key).Scan(&locked); err != nil || !locked {
		conn.Release()
		if err != nil {
			return nil, err
		}
		return nil, apierr.New(apierr.DependencyUnavailable, "Publication reconciliation in progress.")
	}
	return func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if _, err := conn.Exec(cleanup, `SELECT pg_advisory_unlock(hashtextextended($1,0))`, key); err != nil {
			_ = conn.Conn().Close(cleanup)
		}
		conn.Release()
	}, nil
}
