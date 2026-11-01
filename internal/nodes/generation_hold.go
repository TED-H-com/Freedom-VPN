package nodes

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// This diagnostic deadline never grants serving time and is independent of D-075.
const generationHoldAlarmAfter = 30 * time.Second

func reportsGenerationHold(r api.ObservationReport) bool {
	if !hasForwardingBlockReason(r, api.ForwardingBlockReasonPolicyMissing) {
		return false
	}
	for _, c := range r.Checks {
		code, _ := c.ErrorCode.Get()
		if c.Name == api.NodeCheckNameLease && code == api.ErrorCodeSTALEGENERATION {
			return true
		}
	}
	for _, e := range r.Errors {
		if e.Stage == "lease" && e.Code == api.ErrorCodeSTALEGENERATION {
			return true
		}
	}
	return false
}

// Only the reported lease's own persisted signed generation can corroborate a
// hold. Never substitute the latest lease, a summary's number, or another identity.
func reportedLeaseGeneration(ctx context.Context, q Querier, r api.ObservationReport) (decstr.U64, bool, error) {
	id, err := r.Lease.LeaseId.Get()
	if err != nil || id == uuid.Nil {
		return 0, false, nil
	}
	var generation decstr.U64
	err = q.QueryRow(ctx, `SELECT policy_generation::text FROM serving_leases
		WHERE id=$1 AND node_id=$2 AND incarnation_id=$3
		AND EXISTS(SELECT 1 FROM nodes n WHERE n.id=$2 AND n.current_incarnation_id=$3)
		AND envelope#>>'{signed,lease_id}'=id::text
		AND envelope#>>'{signed,node_id}'=node_id::text
		AND envelope#>>'{signed,node_incarnation_id}'=incarnation_id::text
		AND envelope#>>'{signed,config_generation}'=policy_generation::text`, id, r.NodeId, r.NodeIncarnationId).Scan(&generation)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return generation, err == nil, err
}

// D-126: only the known dependent serving checks belong to the hold. A retained
// Apply failure, duplicate failed safety check or independent error still wins.
func generationHoldSafetyFailure(r api.ObservationReport) string {
	checks := map[api.NodeCheckName]api.HealthCheckStatus{}
	for _, c := range r.Checks {
		checks[c.Name] = c.Status
		if c.Status != api.HealthCheckStatusFailed {
			continue
		}
		code, _ := c.ErrorCode.Get()
		if c.Name == api.NodeCheckNameLease && code == api.ErrorCodeSTALEGENERATION {
			continue
		}
		if code == api.ErrorCodeVALIDATIONFAILED && c.Detail != nil &&
			((c.Name == api.NodeCheckNameQos && *c.Detail == "Rate policy could not be applied.") ||
				(c.Name == api.NodeCheckNameSourceBinding && *c.Detail == "No member binding was admitted.")) {
			continue
		}
		return "check_failed"
	}
	for _, e := range r.Errors {
		if e.Code == api.ErrorCodeSTALEGENERATION && e.Stage == "lease" {
			continue
		}
		return "check_failed"
	}
	if r.SafetyStatus != api.SafetyStatusPassed || r.Netbird.Backend == api.ObservationReportNetbirdBackendUserspace ||
		checks[api.NodeCheckNameInternetOnly] != api.HealthCheckStatusPassed || checks[api.NodeCheckNameMeterPath] != api.HealthCheckStatusPassed {
		return "isolation_unconfirmed"
	}
	if hasForwardingBlockReason(r, api.ForwardingBlockReasonSafetyCheckFailed) || hasForwardingBlockReason(r, api.ForwardingBlockReasonMeterPathInvalid) {
		return "isolation_unconfirmed"
	}
	return ""
}

type generationHoldSummary struct {
	Applied, Lease, Steps, ElapsedMS decstr.U64
	Alarm                            bool
}

// Parse the exact documented agent summary, never best-effort Scan/defaults.
// A generic lease-stage error can accompany it; duplicate summaries are invalid.
func parseGenerationHoldSummary(r api.ObservationReport) (generationHoldSummary, bool) {
	const prefix = "Serving lease generation hold: "
	var summary generationHoldSummary
	found := false
	for _, e := range r.Errors {
		if e.Code != api.ErrorCodeSTALEGENERATION || e.Stage != "lease" || !strings.HasPrefix(e.Message, prefix) {
			continue
		}
		if found {
			return generationHoldSummary{}, false
		}
		found = true
		parts := strings.Split(strings.TrimPrefix(e.Message, prefix), " ")
		if len(parts) != 5 {
			return generationHoldSummary{}, false
		}
		for i, field := range []struct {
			key string
			out *decstr.U64
		}{{"applied=", &summary.Applied}, {"lease=", &summary.Lease}, {"steps=", &summary.Steps}, {"elapsed_ms=", &summary.ElapsedMS}} {
			value, ok := strings.CutPrefix(parts[i], field.key)
			var err error
			*field.out, err = decstr.Parse(value)
			if !ok || err != nil {
				return generationHoldSummary{}, false
			}
		}
		switch parts[4] {
		case "alarm=true.":
			summary.Alarm = true
		case "alarm=false.":
		default:
			return generationHoldSummary{}, false
		}
	}
	return summary, found
}
