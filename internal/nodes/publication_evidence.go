package nodes

import (
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"time"
)

// PublicationForwardingEvidence distinguishes a baseline ACK from an armed
// member data path (§5.2, §16.3, SECOPS-11). It is agent evidence, not a traffic
// measurement: callers must also confirm the route and requested generation.
func PublicationForwardingEvidence(report api.ObservationReport, now time.Time) bool {
	generation, err := report.AppliedConfigGeneration.Get()
	if err != nil || generation == 0 || !report.MemberForwardingEnabled {
		return false
	}
	lease, err := report.Lease.LeaseId.Get()
	if err != nil || lease == uuid.Nil {
		return false
	}
	expiry, err := report.Lease.ExpiresAt.Get()
	if err != nil || !expiry.After(now) {
		return false
	}
	passed := make(map[api.NodeCheckName]bool)
	for _, check := range report.Checks {
		passed[check.Name] = check.Status == api.HealthCheckStatusPassed
	}
	for _, name := range []api.NodeCheckName{api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		if !passed[name] {
			return false
		}
	}
	return report.DataHealth == api.DataHealthHealthy && report.SafetyStatus == api.SafetyStatusPassed
}
