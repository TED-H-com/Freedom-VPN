package nodes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/audit"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/commandtx"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Recommended default heartbeat and staleness thresholds (§5.1).
const (
	DefaultHeartbeatInterval = 30 * time.Second
	DefaultStaleTimeout      = 90 * time.Second
	DefaultOfflineTimeout    = 180 * time.Second
)

// SubmitObservationTx joins observation ingestion to the caller's command transaction (§14.5).
func (s *Service) SubmitObservationTx(ctx context.Context, tx pgx.Tx, nodeID, incarnationID uuid.UUID, report api.ObservationReport) (api.ObservationAccepted, error) {
	return s.SubmitObservation(commandtx.WithTransaction(ctx, tx), nodeID, incarnationID, report)
}

// SubmitObservation ingests an agent observation report (`POST /agent/observations`, §5.1, §14.4, §15.3).
//
// Invariants enforced:
//   - Authenticated mTLS identity (authNodeID, authIncarnationID) must match the report body and active incarnation.
//   - Revoked or retired identities are rejected with 403 IDENTITY_REVOKED.
//   - Reports are deduplicated by (node_id, incarnation_id, report_id): same payload hash returns 202 with original received_at; different payload hash returns 409 ACK_CONFLICT.
//   - received_at (central time) is stored separately from observed_at (node time).
//   - An older observed_at never overwrites newer observed state on the node.
//   - safety_status and operation step progress require concrete evidence; a 202 observation alone never completes an operation step.
func (s *Service) SubmitObservation(
	ctx context.Context,
	authNodeID uuid.UUID,
	authIncarnationID uuid.UUID,
	report api.ObservationReport,
) (api.ObservationAccepted, error) {
	if authNodeID == uuid.Nil || authIncarnationID == uuid.Nil {
		return api.ObservationAccepted{}, apierr.New(apierr.Unauthenticated, "authenticated node and incarnation identity required")
	}
	if report.NodeId != authNodeID || report.NodeIncarnationId != authIncarnationID {
		return api.ObservationAccepted{}, apierr.New(apierr.IncarnationMismatch, "observation node_id or node_incarnation_id does not match mTLS identity")
	}
	if report.ReportId == uuid.Nil {
		return api.ObservationAccepted{}, apierr.New(apierr.MalformedRequest, "report_id is required")
	}
	if err := validateDNSPolicyEvidence(report); err != nil {
		return api.ObservationAccepted{}, err
	}

	canonBytes, err := canonjson.Marshal(report)
	if err != nil {
		return api.ObservationAccepted{}, apierr.Newf(apierr.MalformedRequest, "invalid observation payload for canonical hash: %v", err)
	}
	payloadHashArr := sha256.Sum256(canonBytes)
	payloadHash := payloadHashArr[:]

	var accepted api.ObservationAccepted
	err = commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		// 1. Lock the node row and verify identity and lifecycle state.
		var (
			lifecycleState       string
			currentIncarnationID *uuid.UUID
			lastObservedAt       *time.Time
		)
		err := tx.QueryRow(ctx, `
			SELECT lifecycle_state, current_incarnation_id, last_observed_at
			FROM nodes
			WHERE id = $1
			FOR UPDATE`, authNodeID).Scan(&lifecycleState, &currentIncarnationID, &lastObservedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierr.Newf(apierr.ResourceNotFound, "node %s not found", authNodeID)
			}
			return err
		}

		if lifecycleState == string(api.LifecycleStateRevoked) || lifecycleState == string(api.LifecycleStateRetired) {
			return apierr.Newf(apierr.IdentityRevoked, "node %s is %s", authNodeID, lifecycleState)
		}

		var (
			incNodeID uuid.UUID
			revokedAt *time.Time
		)
		err = tx.QueryRow(ctx, `
			SELECT node_id, revoked_at
			FROM node_incarnations
			WHERE id = $1`, authIncarnationID).Scan(&incNodeID, &revokedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apierr.New(apierr.IncarnationMismatch, "incarnation is not registered")
			}
			return err
		}
		if incNodeID != authNodeID {
			return apierr.New(apierr.IncarnationMismatch, "incarnation belongs to a different node")
		}
		if revokedAt != nil {
			return apierr.New(apierr.IdentityRevoked, "node incarnation has been revoked")
		}
		if currentIncarnationID != nil && *currentIncarnationID != authIncarnationID {
			return apierr.New(apierr.IncarnationMismatch, "node incarnation has been superseded by a newer incarnation")
		}

		var mappingFrozen bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM node_identity_endpoints WHERE incarnation_id=$1 AND mapping_frozen)`, authIncarnationID).Scan(&mappingFrozen); err != nil {
			return err
		}

		// 2. Check deduplication by (node_id, incarnation_id, report_id).
		var (
			existingHash       []byte
			existingReceivedAt time.Time
		)
		err = tx.QueryRow(ctx, `
			SELECT payload_hash, received_at
			FROM node_observations
			WHERE node_id = $1 AND incarnation_id = $2 AND report_id = $3`,
			authNodeID, authIncarnationID, report.ReportId,
		).Scan(&existingHash, &existingReceivedAt)
		if err == nil {
			if !bytes.Equal(existingHash, payloadHash) {
				return apierr.New(apierr.AckConflict, "report_id already submitted with a different payload")
			}
			accepted = api.ObservationAccepted{
				Accepted:   api.ObservationAcceptedAcceptedTrue,
				ReportId:   report.ReportId,
				ReceivedAt: existingReceivedAt.UTC(),
			}
			return nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		// 3. Evaluate evidence in the report (fail-closed safety and metering checks, §5.1, §8, §10.3).
		effectiveSafety, effectiveHealth, effectiveForwarding, effectiveErrors := evaluateObservationEvidence(report)

		checksMap := make(map[string]any, len(report.Checks))
		for _, ch := range report.Checks {
			entry := map[string]any{
				"status":      string(ch.Status),
				"observed_at": ch.ObservedAt.UTC().Format(time.RFC3339Nano),
			}
			if code, err := ch.ErrorCode.Get(); err == nil {
				entry["error_code"] = string(code)
			}
			if ch.Detail != nil {
				entry["detail"] = *ch.Detail
			}
			checksMap[string(ch.Name)] = entry
		}
		checksObjJSON, err := json.Marshal(checksMap)
		if err != nil {
			return err
		}
		checksArrayJSON, err := json.Marshal(report.Checks)
		if err != nil {
			return err
		}
		errorsArrayJSON, err := json.Marshal(effectiveErrors)
		if err != nil {
			return err
		}
		reportPayloadJSON, err := json.Marshal(report)
		if err != nil {
			return err
		}

		receivedAt := s.now().UTC().Truncate(time.Microsecond)
		observedAt := report.ObservedAt.UTC()
		appliedGenStr := "0"
		if gen, err := report.AppliedConfigGeneration.Get(); err == nil {
			appliedGenStr = gen.String()
		}

		// pgx receives native nullable UUIDs, not the JSON presence wrappers.
		var currentReleaseID, lastGoodReleaseID *uuid.UUID
		if id, err := report.CurrentReleaseId.Get(); err == nil {
			currentReleaseID = &id
		}
		if id, err := report.LastGoodReleaseId.Get(); err == nil {
			lastGoodReleaseID = &id
		}
		const insertObs = `
			INSERT INTO node_observations (
				node_id, incarnation_id, report_id, payload_hash,
				observed_at, received_at, applied_generation,
				actual_release_id, schema_version,
				checks, control_status, data_health, safety_status,
				member_forwarding_enabled, provisioning_state, report_payload
			) VALUES (
				$1, $2, $3, $4,
				$5, $6, $7::numeric,
				$8, 1,
				$9::jsonb, 'online', $10, $11,
				$12, $13, $14::jsonb
			)`
		if _, err := tx.Exec(ctx, insertObs,
			authNodeID, authIncarnationID, report.ReportId, payloadHash,
			observedAt, receivedAt, appliedGenStr,
			currentReleaseID,
			checksObjJSON, string(effectiveHealth), string(effectiveSafety),
			effectiveForwarding, string(report.ProvisioningState), reportPayloadJSON,
		); err != nil {
			return err
		}

		if s.observeVersion != nil {
			if err := s.observeVersion(ctx, tx, report); err != nil {
				return err
			}
			// The fleet ledger owns version ordering; node wall time alone cannot replace it.
			currentReleaseID = nil
			lastGoodReleaseID = nil
		}
		provisioning := report.ProvisioningState
		if provisioning == api.ProvisioningStateReady {
			// §5.1 / §6.1: baseline readiness needs safety and a generation ACK,
			// not a management peer ID. A reported ID still requires read-back;
			// absent IDs must not block unpublished/disabled baseline readiness.
			// Once the incarnation has reached provisioning_state='ready', an in-flight
			// observation for an already-applied generation with safety_status='passed'
			// does not downgrade provisioning_state back to 'applying' when central
			// concurrently advances desired_config_generation.
			var acknowledged, bound bool
			if err := tx.QueryRow(ctx, `SELECT ($2::numeric>0 AND ($2::numeric>=n.desired_config_generation OR n.provisioning_state='ready')),i.peer_id IS NOT NULL FROM nodes n JOIN node_incarnations i ON i.id=$3 WHERE n.id=$1`, authNodeID, appliedGenStr, authIncarnationID).Scan(&acknowledged, &bound); err != nil {
				return err
			}
			_, peerErr := report.Netbird.PeerId.Get()
			if effectiveSafety != api.SafetyStatusPassed || !acknowledged || (s.observeVersion != nil && peerErr == nil && !bound) {
				provisioning = api.ProvisioningStateApplying
			}
		}
		// D-055: an ambiguous management correlation is terminal for this
		// incarnation. Heartbeats remain accepted but cannot clear its failure.
		if mappingFrozen {
			provisioning = api.ProvisioningStateFailed
			effectiveSafety, effectiveForwarding = api.SafetyStatusFailed, false
		}
		// 4. Update node state. An older observed_at NEVER overrides newer observed state!
		isNewerOrEqual := lastObservedAt == nil || !observedAt.Before(lastObservedAt.UTC())
		if isNewerOrEqual {
			const updateNode = `
				UPDATE nodes
				SET
					control_status = 'online',
					last_seen_at = greatest(coalesce(last_seen_at, $2), $2),
					last_observed_at = $3,
					applied_config_generation = greatest(applied_config_generation, $4::numeric),
					provisioning_state = $5,
					data_health = $6,
					safety_status = $7,
					member_forwarding_observed = $8,
					current_release_id = coalesce($9, current_release_id),
					last_good_release_id = coalesce($10, last_good_release_id),
					latest_checks = $11::jsonb,
					recent_errors = CASE WHEN $13 THEN recent_errors ELSE $12::jsonb END,
					resource_version = resource_version + 1
				WHERE id = $1`
			if _, err := tx.Exec(ctx, updateNode,
				authNodeID, receivedAt, observedAt,
				appliedGenStr,
				string(provisioning),
				string(effectiveHealth),
				string(effectiveSafety),
				effectiveForwarding,
				currentReleaseID,
				lastGoodReleaseID,
				checksArrayJSON,
				errorsArrayJSON, mappingFrozen,
			); err != nil {
				return err
			}

			obsAuditID := uuid.Must(uuid.NewV7())
			afterObsMap := map[string]any{
				"report_id":                 report.ReportId.String(),
				"applied_config_generation": appliedGenStr,
				"safety_status":             string(effectiveSafety),
				"data_health":               string(effectiveHealth),
				"provisioning_state":        string(provisioning),
			}
			if _, ok := isPendingIsolationApplyError(report, effectiveForwarding); ok {
				afterObsMap["isolation_error_code"] = string(api.ErrorCodeISOLATIONPOLICYINVALID)
				afterObsMap["isolation_error_reason"] = "conntrack_delete_pending"
			}
			afterObsAudit, _ := json.Marshal(afterObsMap)
			if err := db.AppendAudit(ctx, tx, db.AuditEvent{
				ID:            obsAuditID,
				Actor:         authNodeID.String(),
				ActorType:     string(api.AuditEventActorTypeNode),
				Action:        "node.observation_applied",
				Resource:      "node:" + authNodeID.String(),
				Result:        string(api.AuditEventResultSucceeded),
				SchemaVersion: 1,
				Before:        []byte(`{}`),
				After:         afterObsAudit,
			}); err != nil {
				return err
			}

			var publication, observed string
			var generation decstr.U64
			if err := tx.QueryRow(ctx, `SELECT e.publication_intent,e.publication_observed,n.desired_config_generation::text FROM nodes n JOIN exits e ON e.node_id=n.id WHERE n.id=$1`, authNodeID).Scan(&publication, &observed, &generation); err != nil {
				return err
			}
			if publication == "published" {
				cause, err := PublicationDataPathFailure(ctx, tx, report, uint64(generation), receivedAt)
				if err != nil {
					return err
				}
				// Only recover a withdrawn empty-exception attachment once the
				// populated desired generation has full serving evidence.
				if cause == "" && observed == "withdrawn" {
					if err := QueuePublicationGateTx(ctx, tx, authNodeID, receivedAt, false); err != nil {
						return err
					}
				}
			}

			// 5. Feed pending/running operation steps ONLY if concrete evidence is satisfied.
			if err := advanceNodeOperationsOnObservation(ctx, tx, authNodeID, report, effectiveSafety, effectiveHealth, effectiveForwarding, receivedAt, obsAuditID); err != nil {
				return err
			}
		} else {
			// Older observed_at: record heartbeat arrival time (last_seen_at) and control_status=online,
			// but never regress observed state fields on nodes.
			if _, err := tx.Exec(ctx, `
				UPDATE nodes
				SET
					control_status = 'online',
					last_seen_at = greatest(coalesce(last_seen_at, $2), $2),
					resource_version = resource_version + 1
				WHERE id = $1`, authNodeID, receivedAt); err != nil {
				return err
			}
		}

		accepted = api.ObservationAccepted{
			Accepted:   api.ObservationAcceptedAcceptedTrue,
			ReportId:   report.ReportId,
			ReceivedAt: receivedAt,
		}
		return nil
	})
	if err != nil {
		return api.ObservationAccepted{}, err
	}
	return accepted, nil
}

// evaluateObservationEvidence enforces fail-closed validation on a node's reported checks (§5.1, §8, §10.3).
func evaluateObservationEvidence(report api.ObservationReport) (api.SafetyStatus, api.DataHealth, bool, []api.NodeError) {
	errs := make([]api.NodeError, 0, len(report.Errors))
	errs = append(errs, report.Errors...)

	checkStatus := make(map[api.NodeCheckName]api.HealthCheckStatus, len(report.Checks))
	for _, ch := range report.Checks {
		checkStatus[ch.Name] = ch.Status
	}

	safety := report.SafetyStatus
	health := report.DataHealth
	forwarding := report.MemberForwardingEnabled

	// §10.3: Userspace NetBird backend cannot provide reliable kernel per-peer metering.
	if report.Netbird.Backend == api.ObservationReportNetbirdBackendUserspace {
		safety = api.SafetyStatusFailed
		errs = append(errs, api.NodeError{
			Code:       api.ErrorCode(apierr.MeterPathInvalid),
			Message:    "netbird userspace backend cannot satisfy kernel metering requirement",
			ObservedAt: report.ObservedAt.UTC(),
			Stage:      "metering",
		})
	}

	// §8.1 / §5.1: safety_status=passed requires internet_only and meter_path checks to pass.
	if safety == api.SafetyStatusPassed {
		if checkStatus[api.NodeCheckNameInternetOnly] != api.HealthCheckStatusPassed ||
			checkStatus[api.NodeCheckNameMeterPath] != api.HealthCheckStatusPassed {
			safety = api.SafetyStatusFailed
			errs = append(errs, api.NodeError{
				Code:       api.ErrorCode(apierr.NodeNotServiceable),
				Message:    "safety_status=passed rejected because required isolation/metering checks did not pass",
				ObservedAt: report.ObservedAt.UTC(),
				Stage:      "safety",
			})
		}
	}

	if checkStatus[api.NodeCheckNameInternetOnly] == api.HealthCheckStatusFailed ||
		checkStatus[api.NodeCheckNameMeterPath] == api.HealthCheckStatusFailed {
		safety = api.SafetyStatusFailed
	}

	if checkStatus[api.NodeCheckNameTunnelProbe] == api.HealthCheckStatusFailed && health == api.DataHealthHealthy {
		health = api.DataHealthDegraded
	}

	if _, ok := hasIsolationApplyError(report); ok {
		if health == api.DataHealthHealthy {
			health = api.DataHealthDegraded
		}
		if !forwarding {
			safety = api.SafetyStatusFailed
		}
	}
	for _, e := range report.Errors {
		if e.Code == api.ErrorCodeISOLATIONBASELINEMISSING || e.Code == api.ErrorCodeMETERPATHINVALID {
			safety = api.SafetyStatusFailed
		}
	}

	return safety, health, forwarding, errs
}

func isRevokeMemberOperation(kind, memberState string) bool {
	switch kind {
	case string(api.OperationKindMemberDeviceRevoke),
		string(api.OperationKindMemberGrantRevoke),
		string(api.OperationKindMemberSuspend),
		"member.revoke",
		"member_device.revoke":
		return true
	case string(api.OperationKindMemberUpdate):
		return memberState == "closed" || memberState == "suspended"
	default:
		return false
	}
}

// advanceNodeOperationsOnObservation inspects active node operations and updates
// observation-driven steps (`agent_policy`, `data_path_probe`) ONLY when the report's
// concrete evidence satisfies the step criteria (§5.2, §14.4, §15.4).
func advanceNodeOperationsOnObservation(
	ctx context.Context,
	tx pgx.Tx,
	nodeID uuid.UUID,
	report api.ObservationReport,
	effectiveSafety api.SafetyStatus,
	effectiveHealth api.DataHealth,
	effectiveForwarding bool,
	now time.Time,
	obsAuditID uuid.UUID,
) error {
	rows, err := tx.Query(ctx, `
		SELECT id, kind, status, desired_generation::text, steps, observed_refs
		FROM operations
		WHERE resource_type = 'node'
		  AND resource_id = $1
		  AND status IN ('pending', 'running', 'partially_succeeded')
		ORDER BY created_at ASC
		FOR UPDATE`, nodeID)
	if err != nil {
		return err
	}
	type opRow struct {
		id         uuid.UUID
		kind       string
		status     string
		desiredGen *string
		stepsBytes []byte
		refsBytes  []byte
	}
	var ops []opRow
	for rows.Next() {
		var r opRow
		if err := rows.Scan(&r.id, &r.kind, &r.status, &r.desiredGen, &r.stepsBytes, &r.refsBytes); err != nil {
			rows.Close()
			return err
		}
		ops = append(ops, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	var appliedGen uint64
	if gen, err := report.AppliedConfigGeneration.Get(); err == nil {
		appliedGen = uint64(gen)
	}
	checkStatus := make(map[api.NodeCheckName]api.HealthCheckStatus, len(report.Checks))
	for _, ch := range report.Checks {
		checkStatus[ch.Name] = ch.Status
	}
	_, hasIsoErr := hasIsolationApplyError(report)
	_, pendingIso := isPendingIsolationApplyError(report, effectiveForwarding)

	for _, op := range ops {
		var desiredGen uint64
		if op.desiredGen != nil {
			desiredGen, _ = strconv.ParseUint(*op.desiredGen, 10, 64)
		}
		// Never advance an operation step if the node has not yet applied the desired config generation!
		if desiredGen > 0 && (!report.AppliedConfigGeneration.IsSpecified() || report.AppliedConfigGeneration.IsNull() || appliedGen < desiredGen) {
			continue
		}

		var steps map[string]StoredStep
		if err := json.Unmarshal(op.stepsBytes, &steps); err != nil || len(steps) == 0 {
			continue
		}
		var refs map[string]string
		if len(op.refsBytes) > 0 {
			_ = json.Unmarshal(op.refsBytes, &refs)
		}
		if refs == nil {
			refs = map[string]string{}
		}

		changed := false
		switch op.kind {
		case OpKindPublish, OpKindUpdate:
			// Requires applied_config_generation >= desired_generation AND safety_status == passed AND provisioning_state == ready without an isolation apply error.
			if step, ok := steps[StepAgentPolicy]; ok && step.Status != string(api.OperationStepStatusSucceeded) {
				if effectiveSafety == api.SafetyStatusPassed && report.ProvisioningState == api.ProvisioningStateReady && !hasIsoErr {
					step.Status = string(api.OperationStepStatusSucceeded)
					step.ObservedRef = StepAgentPolicy
					step.ObservedAt = &now
					step.ErrorCode = nil
					step.ErrorMessage = nil
					step.Reason = ""
					step.UpdatedAt = now
					steps[StepAgentPolicy] = step
					refs[StepAgentPolicy] = fmt.Sprintf("report:%s:gen:%d:safety:passed", report.ReportId, appliedGen)
					changed = true
				}
			}
			if step, ok := steps[StepDataPathProbe]; ok && step.Required && step.Status != string(api.OperationStepStatusSucceeded) {
				publicationConfirmed := true
				if op.kind == OpKindPublish {
					cause, err := PublicationDataPathFailure(ctx, tx, report, desiredGen, now)
					if err != nil {
						return err
					}
					publicationConfirmed = effectiveForwarding && cause == ""
				}
				if steps[StepAgentPolicy].Status == string(api.OperationStepStatusSucceeded) &&
					checkStatus[api.NodeCheckNameTunnelProbe] == api.HealthCheckStatusPassed &&
					effectiveHealth == api.DataHealthHealthy &&
					!hasIsoErr &&
					publicationConfirmed {
					step.Status = string(api.OperationStepStatusSucceeded)
					step.ObservedRef = StepDataPathProbe
					step.ObservedAt = &now
					step.ErrorCode = nil
					step.ErrorMessage = nil
					step.Reason = ""
					step.UpdatedAt = now
					steps[StepDataPathProbe] = step
					refs[StepDataPathProbe] = fmt.Sprintf("report:%s:tunnel_probe:passed", report.ReportId)
					changed = true
				}
			}

		case OpKindUnpublish, OpKindDisable, OpKindRevoke, OpKindRetire:
			// Requires applied_config_generation >= desired_generation AND member_forwarding_enabled == false.
			if step, ok := steps[StepAgentPolicy]; ok && step.Status != string(api.OperationStepStatusSucceeded) {
				if !effectiveForwarding {
					step.Status = string(api.OperationStepStatusSucceeded)
					step.ObservedRef = StepAgentPolicy
					step.ObservedAt = &now
					step.ErrorCode = nil
					step.ErrorMessage = nil
					step.Reason = ""
					step.UpdatedAt = now
					steps[StepAgentPolicy] = step
					refs[StepAgentPolicy] = fmt.Sprintf("report:%s:gen:%d:forwarding:false", report.ReportId, appliedGen)
					changed = true
				}
			}
		}

		if !changed {
			continue
		}

		stepsJSON, err := json.Marshal(steps)
		if err != nil {
			return err
		}
		refsJSON, err := json.Marshal(refs)
		if err != nil {
			return err
		}

		allRequiredSucceeded := true
		for _, st := range steps {
			if st.Required && st.Status != string(api.OperationStepStatusSucceeded) {
				allRequiredSucceeded = false
				break
			}
		}

		obsGenStr := strconv.FormatUint(appliedGen, 10)
		// Respect guard_operation() state machine:
		// pending -> running -> succeeded
		// partially_succeeded -> running -> succeeded
		if op.status == string(api.OperationStatusPending) || op.status == string(api.OperationStatusPartiallySucceeded) {
			if _, err := tx.Exec(ctx, `
				UPDATE operations
				SET status = 'running',
				    observed_generation = $2::numeric,
				    steps = $3::jsonb,
				    observed_refs = $4::jsonb,
				    resource_version = coalesce(resource_version, 0) + 1
				WHERE id = $1`, op.id, obsGenStr, stepsJSON, refsJSON); err != nil {
				return err
			}
			op.status = string(api.OperationStatusRunning)
		} else {
			if _, err := tx.Exec(ctx, `
				UPDATE operations
				SET observed_generation = $2::numeric,
				    steps = $3::jsonb,
				    observed_refs = $4::jsonb,
				    resource_version = coalesce(resource_version, 0) + 1
				WHERE id = $1`, op.id, obsGenStr, stepsJSON, refsJSON); err != nil {
				return err
			}
		}

		if allRequiredSucceeded && op.status == string(api.OperationStatusRunning) {
			if op.kind == OpKindPublish {
				if err := RecordPublicationAttachmentTx(ctx, tx, op.id, obsGenStr); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, `
				UPDATE operations
				SET status = 'succeeded',
				    completed_at = $2,
				    error_code = NULL,
				    error_message = NULL,
				    resource_version = coalesce(resource_version, 0) + 1
				WHERE id = $1`, op.id, now); err != nil {
				return err
			}
			// If this was a publish or unpublish/disable operation that now has all required steps succeeded,
			// update exits.publication_observed accordingly.
			switch op.kind {
			case OpKindPublish:
				if _, err := tx.Exec(ctx, `
					UPDATE exits
					SET publication_observed = 'active', resource_version = resource_version + 1
					WHERE node_id = $1`, nodeID); err != nil {
					return err
				}
			case OpKindUnpublish, OpKindDisable, OpKindRevoke, OpKindRetire:
				if _, err := tx.Exec(ctx, `
					UPDATE exits
					SET publication_observed = 'withdrawn', resource_version = resource_version + 1
					WHERE node_id = $1`, nodeID); err != nil {
					return err
				}
			}
		}
	}

	// Also advance or hold the "nodes" step on member, entitlement_grant (via entitlement_grants.member_id),
	// and member_device (via member_devices.member_id) operations (§5.2, §8.1, r13 M-3).
	var desiredNodeGenStr, dbAppliedNodeGenStr string
	if err := tx.QueryRow(ctx, `SELECT desired_config_generation::text, applied_config_generation::text FROM nodes WHERE id = $1`, nodeID).Scan(&desiredNodeGenStr, &dbAppliedNodeGenStr); err != nil {
		return err
	}
	desiredNodeGen, _ := strconv.ParseUint(desiredNodeGenStr, 10, 64)
	dbAppliedNodeGen, _ := strconv.ParseUint(dbAppliedNodeGenStr, 10, 64)

	reportAcked := report.AppliedConfigGeneration.IsSpecified() &&
		!report.AppliedConfigGeneration.IsNull() &&
		appliedGen > 0 &&
		appliedGen >= desiredNodeGen
	nodeAcked := desiredNodeGen > 0 && (reportAcked || dbAppliedNodeGen >= desiredNodeGen)
	cleanReady := reportAcked &&
		effectiveSafety == api.SafetyStatusPassed &&
		report.ProvisioningState == api.ProvisioningStateReady &&
		!hasIsoErr
	failedClosed := !effectiveForwarding &&
		!pendingIso &&
		checkStatus[api.NodeCheckNameInternetOnly] == api.HealthCheckStatusPassed &&
		(nodeAcked || hasIsoErr)

	if cleanReady || (reportAcked && pendingIso) || failedClosed {
		type memberOpCandidate struct {
			id                  uuid.UUID
			kind                string
			resourceType        string
			resourceID          uuid.UUID
			requestID           *uuid.UUID
			memberState         string
			prevStatus          string
			prevErrCode         string
			otherNodePendingIso bool
		}
		mRows, err := tx.Query(ctx, `
			SELECT
				o.id,
				o.kind,
				o.resource_type,
				o.resource_id,
				o.request_id,
				coalesce(m.status, ''),
				coalesce(o.steps->'nodes'->>'status', ''),
				coalesce(o.steps->'nodes'->>'error_code', ''),
				EXISTS (
					SELECT 1
					FROM nodes n2
					JOIN exits ex2 ON ex2.node_id = n2.id
					WHERE n2.id <> $1
					  AND n2.lifecycle_state = 'registered'
					  AND coalesce(n2.member_forwarding_observed, false)
					  AND n2.recent_errors @> '[{"code":"ISOLATION_POLICY_INVALID"}]'::jsonb
					  AND (
						NOT EXISTS (SELECT 1 FROM exit_groups eg2 WHERE eg2.exit_id = ex2.id)
						OR jsonb_array_length(coalesce(e.projection->'exit_group_ids', '[]'::jsonb)) = 0
						OR EXISTS (
							SELECT 1
							FROM exit_groups eg2
							WHERE eg2.exit_id = ex2.id
							  AND (
								(e.projection->'exit_group_ids') ? eg2.group_id::text
								OR eg2.group_id = ANY(m.member_group_ids)
							  )
						)
					  )
				)
			FROM operations o
			LEFT JOIN entitlement_grants g ON o.resource_type = 'entitlement_grant' AND g.id = o.resource_id
			LEFT JOIN member_devices d ON o.resource_type = 'member_device' AND d.id = o.resource_id
			LEFT JOIN members m ON m.id = CASE
				WHEN o.resource_type = 'member' THEN o.resource_id
				WHEN o.resource_type = 'entitlement_grant' THEN g.member_id
				WHEN o.resource_type = 'member_device' THEN d.member_id
			END
			LEFT JOIN entitlements e ON e.member_id = m.id
			WHERE o.resource_type IN ('member', 'entitlement_grant', 'member_device')
			  AND m.id IS NOT NULL
			  AND o.status IN ('pending', 'running', 'failed', 'partially_succeeded')
			  AND o.steps->'nodes'->>'status' IN ('pending', 'running', 'failed')
			  AND (
				(o.resource_type IN ('member', 'entitlement_grant') AND (o.desired_generation IS NULL OR (e.revision IS NOT NULL AND e.revision >= o.desired_generation)))
				OR (o.resource_type = 'member_device' AND (o.desired_generation IS NULL OR (d.resource_version IS NOT NULL AND d.resource_version >= o.desired_generation)))
			  )
			  AND (
				o.kind <> 'member.grant'
				OR NOT EXISTS (
					SELECT 1 FROM entitlement_grants g2
					WHERE (g2.operation_id = o.id OR (o.resource_type = 'entitlement_grant' AND g2.id = o.resource_id))
					  AND g2.valid_from > $2
				)
			  )
			  AND (
				NOT EXISTS (SELECT 1 FROM exits ex JOIN exit_groups eg ON eg.exit_id = ex.id WHERE ex.node_id = $1)
				OR jsonb_array_length(coalesce(e.projection->'exit_group_ids', '[]'::jsonb)) = 0
				OR EXISTS (
					SELECT 1
					FROM exits ex
					JOIN exit_groups eg ON eg.exit_id = ex.id
					WHERE ex.node_id = $1
					  AND (
						(e.projection->'exit_group_ids') ? eg.group_id::text
						OR eg.group_id = ANY(m.member_group_ids)
					  )
				)
			  )
			ORDER BY o.id ASC
			FOR UPDATE OF o`, nodeID, now)
		if err != nil {
			return err
		}
		var memberOps []memberOpCandidate
		for mRows.Next() {
			var c memberOpCandidate
			if err := mRows.Scan(
				&c.id, &c.kind, &c.resourceType, &c.resourceID, &c.requestID,
				&c.memberState, &c.prevStatus, &c.prevErrCode, &c.otherNodePendingIso,
			); err != nil {
				mRows.Close()
				return err
			}
			if c.requestID != nil && *c.requestID == uuid.Nil {
				c.requestID = nil
			}
			memberOps = append(memberOps, c)
		}
		if err := mRows.Err(); err != nil {
			mRows.Close()
			return err
		}
		mRows.Close()

		for _, op := range memberOps {
			if reportAcked && pendingIso {
				// Keep the "nodes" step pending while a conntrack flow delete retry is pending
				// on the applied generation, surfacing ISOLATION_POLICY_INVALID through the operation read.
				var stepsBytes, refsBytes []byte
				if err := tx.QueryRow(ctx, `SELECT steps, coalesce(observed_refs, '{}'::jsonb) FROM operations WHERE id = $1`, op.id).Scan(&stepsBytes, &refsBytes); err != nil {
					return err
				}
				var steps map[string]StoredStep
				if err := json.Unmarshal(stepsBytes, &steps); err != nil {
					return err
				}
				var refs map[string]any
				if len(refsBytes) > 0 {
					_ = json.Unmarshal(refsBytes, &refs)
				}
				if refs == nil {
					refs = map[string]any{}
				}
				nodeStep := steps["nodes"]
				errCode := string(api.ErrorCodeISOLATIONPOLICYINVALID)
				errMsg := "member_forwarding_unconfirmed"
				nodeStep.Status = string(api.OperationStepStatusPending)
				nodeStep.ErrorCode = &errCode
				nodeStep.ErrorMessage = &errMsg
				nodeStep.Reason = "conntrack_delete_pending"
				nodeStep.ObservedAt = &now
				nodeStep.UpdatedAt = now
				steps["nodes"] = nodeStep
				refs["nodes_isolation_pending"] = fmt.Sprintf("report:%s:gen:%d:error:ISOLATION_POLICY_INVALID", report.ReportId, appliedGen)

				newStepsJSON, err := json.Marshal(steps)
				if err != nil {
					return err
				}
				newRefsJSON, err := json.Marshal(refs)
				if err != nil {
					return err
				}
				if _, err := tx.Exec(ctx, `
					UPDATE operations
					SET steps = $2::jsonb,
					    observed_refs = $3::jsonb,
					    updated_at = $4,
					    resource_version = coalesce(resource_version, 0) + 1
					WHERE id = $1`, op.id, newStepsJSON, newRefsJSON, now); err != nil {
					return err
				}
				if op.prevErrCode != string(api.ErrorCodeISOLATIONPOLICYINVALID) {
					opIDCopy := op.id
					ev, err := audit.NewEvent(audit.Entry{
						Actor:       audit.Actor{Type: api.AuditEventActorTypeNode, ID: nodeID},
						Action:      "node.revoke_isolation_pending",
						Resource:    api.ResourceRef{Type: api.ResourceType(op.resourceType), Id: op.resourceID},
						Result:      api.AuditEventResultFailed,
						OperationID: &opIDCopy,
						RequestID:   op.requestID,
						Before:      map[string]any{"nodes_step_status": op.prevStatus},
						After: map[string]any{
							"nodes_step_status":         string(api.OperationStepStatusPending),
							"error_code":                string(api.ErrorCodeISOLATIONPOLICYINVALID),
							"reason":                    "conntrack_delete_pending",
							"applied_config_generation": strconv.FormatUint(appliedGen, 10),
							"report_id":                 report.ReportId.String(),
						},
						Fields: []string{"nodes_step_status", "error_code", "reason", "applied_config_generation", "report_id"},
					})
					if err != nil {
						return err
					}
					if err := audit.AppendEvent(ctx, tx, ev); err != nil {
						return err
					}
				}
				continue
			}

			canCompleteOnFailClosed := !cleanReady && failedClosed && (op.prevErrCode == string(api.ErrorCodeISOLATIONPOLICYINVALID) || (nodeAcked && isRevokeMemberOperation(op.kind, op.memberState)))
			if !cleanReady && !canCompleteOnFailClosed {
				continue
			}
			if op.otherNodePendingIso {
				continue
			}

			if op.prevStatus == "failed" {
				if err := entitlement.TransitionOperation(ctx, tx, op.id, "nodes", api.OperationStepStatusRunning, nil); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, `
				UPDATE operations
				SET steps = jsonb_set(steps, '{nodes}', (steps->'nodes') - 'error_code' - 'error_message' - 'reason'),
				    observed_refs = coalesce(observed_refs, '{}'::jsonb) - 'nodes_isolation_pending'
				WHERE id = $1`, op.id); err != nil {
				return err
			}
			if err := entitlement.TransitionOperation(ctx, tx, op.id, "nodes", api.OperationStepStatusSucceeded, &operations.Evidence{
				Reference:  "audit_event:" + obsAuditID.String(),
				ObservedAt: now,
			}); err != nil {
				return err
			}
			if op.prevErrCode == string(api.ErrorCodeISOLATIONPOLICYINVALID) || canCompleteOnFailClosed {
				resolution := "conntrack_delete_succeeded"
				if !cleanReady && canCompleteOnFailClosed {
					resolution = "node_failed_closed"
				}
				genVal := appliedGen
				if genVal == 0 {
					genVal = dbAppliedNodeGen
				}
				opIDCopy := op.id
				ev, err := audit.NewEvent(audit.Entry{
					Actor:       audit.Actor{Type: api.AuditEventActorTypeNode, ID: nodeID},
					Action:      "node.revoke_isolation_resolved",
					Resource:    api.ResourceRef{Type: api.ResourceType(op.resourceType), Id: op.resourceID},
					Result:      api.AuditEventResultSucceeded,
					OperationID: &opIDCopy,
					RequestID:   op.requestID,
					Before: map[string]any{
						"nodes_step_status": op.prevStatus,
						"error_code":        op.prevErrCode,
					},
					After: map[string]any{
						"nodes_step_status":         string(api.OperationStepStatusSucceeded),
						"error_code":                "",
						"resolution":                resolution,
						"applied_config_generation": strconv.FormatUint(genVal, 10),
						"report_id":                 report.ReportId.String(),
					},
					Fields: []string{"nodes_step_status", "error_code", "resolution", "applied_config_generation", "report_id"},
				})
				if err != nil {
					return err
				}
				if err := audit.AppendEvent(ctx, tx, ev); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// StalenessSweepResult summarizes the transitions performed by SweepStaleNodes.
type StalenessSweepResult struct {
	MarkedStale   int
	MarkedOffline int
}

// SweepStaleNodes transitions nodes' control_status to stale and offline based on
// configurable timeout thresholds since last_seen_at (§5.1, §14.4).
// It never alters exits.publication_intent (EXIT03).
func (s *Service) SweepStaleNodes(
	ctx context.Context,
	staleAfter time.Duration,
	offlineAfter time.Duration,
) (StalenessSweepResult, error) {
	if staleAfter <= 0 {
		staleAfter = DefaultStaleTimeout
	}
	if offlineAfter <= 0 {
		offlineAfter = DefaultOfflineTimeout
	}
	if offlineAfter < staleAfter {
		offlineAfter = staleAfter * 2
	}

	now := s.now().UTC()
	staleCutoff := now.Add(-staleAfter)
	offlineCutoff := now.Add(-offlineAfter)

	var res StalenessSweepResult
	err := commandtx.InTx(ctx, s.pool, db.TxOptions{}, func(tx pgx.Tx) error {
		// 1. Move online/stale nodes past offlineCutoff to offline.
		tagOffline, err := tx.Exec(ctx, `
			UPDATE nodes
			SET control_status = 'offline',
			    resource_version = resource_version + 1
			WHERE control_status IN ('online', 'stale')
			  AND (last_seen_at IS NULL OR last_seen_at <= $1)`, offlineCutoff)
		if err != nil {
			return err
		}
		res.MarkedOffline = int(tagOffline.RowsAffected())

		// 2. Move online nodes between offlineCutoff and staleCutoff to stale.
		tagStale, err := tx.Exec(ctx, `
			UPDATE nodes
			SET control_status = 'stale',
			    resource_version = resource_version + 1
			WHERE control_status = 'online'
			  AND last_seen_at <= $1
			  AND last_seen_at > $2`, staleCutoff, offlineCutoff)
		if err != nil {
			return err
		}
		res.MarkedStale = int(tagStale.RowsAffected())
		return nil
	})
	return res, err
}
