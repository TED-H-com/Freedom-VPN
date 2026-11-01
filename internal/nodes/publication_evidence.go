package nodes

import (
	"context"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PublicationForwardingEvidence checks the reported armed member path (§5.2,
// §16.3, SECOPS-11). Central decisions must use PublicationDataPathFailure to
// also check the stored lease deadline; this helper only validates observations.
func PublicationForwardingEvidence(report api.ObservationReport, now time.Time) bool {
	return publicationForwardingFailure(report, now, false) == ""
}

// PublicationDataPathFailure returns a fixed cause, or empty on confirmed evidence.
// The zero-binding exception requires the current incarnation's issued published
// generation, never live device counts or an absent/unknown envelope (§5.2).
func PublicationDataPathFailure(ctx context.Context, q Querier, report api.ObservationReport, minimumGeneration uint64, now time.Time) (string, error) {
	if reportsGenerationHold(report) {
		if cause := generationHoldSafetyFailure(report); cause != "" {
			return cause, nil
		}
		generation, known, err := reportedLeaseGeneration(ctx, q, report)
		if err != nil {
			return "", err
		}
		applied, appliedErr := report.AppliedConfigGeneration.Get()
		if !known || appliedErr != nil || generation == applied {
			return "lease_unconfirmed", nil
		}
		return "stale_generation", nil
	}
	applied, err := report.AppliedConfigGeneration.Get()
	if err != nil || applied == 0 || uint64(applied) < minimumGeneration {
		return "stale_generation", nil
	}
	var desired decstr.U64
	var empty, published bool
	err = q.QueryRow(ctx, `SELECT n.desired_config_generation::text,
		coalesce(e.generation=n.desired_config_generation AND e.generation=$3::text::numeric
		AND e.envelope#>>'{signed,config_generation}'=$3::text
		AND e.envelope#>>'{signed,publication_intent}'='published'
		AND e.envelope#>'{signed,device_bindings}'='[]'::jsonb,false), n.lifecycle_state='registered' AND x.publication_intent='published'
		FROM nodes n JOIN exits x ON x.node_id=n.id LEFT JOIN desired_state_envelopes e
		ON e.node_id=n.id AND e.incarnation_id=n.current_incarnation_id
		WHERE n.id=$1 AND n.current_incarnation_id=$2`, report.NodeId, report.NodeIncarnationId, applied.String()).Scan(&desired, &empty, &published)
	if errors.Is(err, pgx.ErrNoRows) {
		return "stale_generation", nil
	}
	if err != nil {
		return "", err
	}
	if !published {
		return "readiness_unconfirmed", nil
	}
	if applied < desired {
		return "stale_generation", nil
	}
	if cause := publicationForwardingFailure(report, now, empty); cause != "" {
		return cause, nil
	}
	if !empty {
		// D-088: an agent may still report the original signed expiry after
		// a central clamp. Its claim cannot extend the stored lease deadline.
		leaseID, _ := report.Lease.LeaseId.Get()
		var valid bool
		if err := q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM serving_leases
			WHERE id=$1 AND node_id=$2 AND incarnation_id=$3
			AND issued_at <= $4 AND expires_at > $4
			AND policy_generation=$5::text::numeric)`, leaseID, report.NodeId, report.NodeIncarnationId, now, applied.String()).Scan(&valid); err != nil {
			return "", err
		}
		if !valid {
			return "lease_unconfirmed", nil
		}
	}
	return "", nil
}

func publicationForwardingFailure(report api.ObservationReport, now time.Time, emptyBindings bool) string {
	if reportsGenerationHold(report) {
		if cause := generationHoldSafetyFailure(report); cause != "" {
			return cause
		}
		return "stale_generation"
	}
	generation, err := report.AppliedConfigGeneration.Get()
	if err != nil || generation == 0 {
		return "stale_generation"
	}
	checks := make(map[api.NodeCheckName]api.HealthCheckStatus, len(report.Checks))
	for _, check := range report.Checks {
		// Never let another check (including a duplicate name) hide a failure.
		if check.Status == api.HealthCheckStatusFailed {
			return "check_failed"
		}
		checks[check.Name] = check.Status
	}
	if !report.MemberForwardingEnabled {
		return "forwarding_disabled"
	}
	if checks[api.NodeCheckNameTunnelProbe] != api.HealthCheckStatusPassed {
		return "tunnel_not_passed"
	}
	if report.SafetyStatus != api.SafetyStatusPassed || checks[api.NodeCheckNameInternetOnly] != api.HealthCheckStatusPassed || checks[api.NodeCheckNameMeterPath] != api.HealthCheckStatusPassed {
		return "isolation_unconfirmed"
	}
	if _, ok := hasIsolationApplyError(report); ok {
		return "isolation_unconfirmed"
	}
	if report.DataHealth != api.DataHealthHealthy || report.ProvisioningState != api.ProvisioningStateReady {
		return "readiness_unconfirmed"
	}
	for _, name := range []api.NodeCheckName{api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		status := checks[name]
		if status != api.HealthCheckStatusPassed && !(emptyBindings && status == api.HealthCheckStatusNotRun) {
			return "serving_checks_unconfirmed"
		}
	}
	// With no admitted binding, the agent has not armed a member lease yet.
	// Once any binding exists the full SECOPS-11 lease evidence remains required.
	if !emptyBindings {
		lease, err := report.Lease.LeaseId.Get()
		if err != nil || lease == uuid.Nil {
			return "lease_unconfirmed"
		}
		expiry, err := report.Lease.ExpiresAt.Get()
		if err != nil || !expiry.After(now) {
			return "lease_unconfirmed"
		}
	}
	return ""
}

// hasIsolationApplyError reports whether report carries an ISOLATION_POLICY_INVALID
// error at stage "apply" from the node's enforcement/conntrack apply path (§5.2, §8.1, D-070).
func hasIsolationApplyError(report api.ObservationReport) (api.NodeError, bool) {
	for _, e := range report.Errors {
		if e.Code == api.ErrorCodeISOLATIONPOLICYINVALID && e.Stage == "apply" {
			return e, true
		}
	}
	return api.NodeError{}, false
}

// isPendingIsolationApplyError reports whether report carries an ISOLATION_POLICY_INVALID
// apply error (keyed on code + stage, D-070) while member forwarding remains active
// or safety_status is still passed (pending conntrack flow delete under the retry bound).
func isPendingIsolationApplyError(report api.ObservationReport, effectiveForwarding bool) (api.NodeError, bool) {
	isoErr, ok := hasIsolationApplyError(report)
	if !ok {
		return api.NodeError{}, false
	}
	if effectiveForwarding || report.SafetyStatus == api.SafetyStatusPassed {
		return isoErr, true
	}
	return api.NodeError{}, false
}

// isFailClosedIsolationApplyError reports whether report matches the agent's
// fail-closed conntrack observation shape (D-070): member_forwarding_enabled=false,
// safety_status=failed, forwarding_block_reasons contains safety_check_failed,
// check internet_only failed with ISOLATION_POLICY_INVALID, and the
// ISOLATION_POLICY_INVALID apply error is present in errors.
func isFailClosedIsolationApplyError(report api.ObservationReport, effectiveSafety api.SafetyStatus, effectiveForwarding bool) bool {
	if effectiveForwarding || effectiveSafety != api.SafetyStatusFailed {
		return false
	}
	if _, ok := isPendingIsolationApplyError(report, effectiveForwarding); ok {
		return false
	}
	if _, ok := hasIsolationApplyError(report); !ok {
		return false
	}
	if !hasForwardingBlockReason(report, api.ForwardingBlockReasonSafetyCheckFailed) {
		return false
	}
	for _, e := range report.Errors {
		if e.Code == api.ErrorCodeISOLATIONBASELINEMISSING {
			return false
		}
	}
	var internetOnlyFailedIso bool
	for _, ch := range report.Checks {
		if ch.Name != api.NodeCheckNameInternetOnly {
			continue
		}
		code, err := ch.ErrorCode.Get()
		if ch.Status != api.HealthCheckStatusFailed || err != nil || code != api.ErrorCodeISOLATIONPOLICYINVALID {
			return false
		}
		internetOnlyFailedIso = true
	}
	return internetOnlyFailedIso
}
