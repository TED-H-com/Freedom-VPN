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

// PublicationForwardingEvidence checks the armed member path (§5.2, §16.3,
// SECOPS-11); without an issued desired state it cannot grant the empty-binding exception.
func PublicationForwardingEvidence(report api.ObservationReport, now time.Time) bool {
	return publicationForwardingFailure(report, now, false) == ""
}

// PublicationDataPathFailure returns a fixed cause, or empty on confirmed evidence.
// The zero-binding exception requires the current incarnation's issued published
// generation, never live device counts or an absent/unknown envelope (§5.2).
func PublicationDataPathFailure(ctx context.Context, q Querier, report api.ObservationReport, minimumGeneration uint64, now time.Time) (string, error) {
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
	return publicationForwardingFailure(report, now, empty), nil
}

func publicationForwardingFailure(report api.ObservationReport, now time.Time, emptyBindings bool) string {
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
