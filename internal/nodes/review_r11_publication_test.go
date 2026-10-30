package nodes

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// documentedPublicationReasons is the step reason vocabulary in
// docs/nodes/WIRING.md (Publication serving evidence and failure causes).
func documentedPublicationReasons() map[string]bool {
	return map[string]bool{
		"forwarding_disabled":        true,
		"stale_generation":           true,
		"tunnel_not_passed":          true,
		"check_failed":               true,
		"serving_checks_unconfirmed": true,
		"isolation_unconfirmed":      true,
		"readiness_unconfirmed":      true,
		"lease_unconfirmed":          true,
	}
}

func reviewReport(empty bool) api.ObservationReport {
	r := api.ObservationReport{
		ProvisioningState: api.ProvisioningStateReady, SafetyStatus: api.SafetyStatusPassed,
		DataHealth: api.DataHealthHealthy, MemberForwardingEnabled: true,
	}
	r.AppliedConfigGeneration.Set(4)
	for i, name := range []api.NodeCheckName{
		api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath, api.NodeCheckNameTunnelProbe,
		api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding,
	} {
		status := api.HealthCheckStatusPassed
		if empty && i >= 3 {
			status = api.HealthCheckStatusNotRun
		}
		r.Checks = append(r.Checks, api.NodeCheck{Name: name, Status: status})
	}
	if !empty {
		r.Lease.LeaseId.Set(uuid.Must(uuid.NewV7()))
		r.Lease.ExpiresAt.Set(time.Now().UTC().Add(time.Minute))
	}
	return r
}

func TestReviewR11_PublicationReasonVocabulary(t *testing.T) {
	now := time.Now().UTC()
	reasons := map[string]string{}
	record := func(name, got string) {
		t.Helper()
		reasons[name] = got
		if got != "" && !documentedPublicationReasons()[got] {
			t.Fatalf("reason %q is outside the WIRING.md vocabulary", got)
		}
	}
	empty := reviewReport(true)
	record("empty_not_run", publicationForwardingFailure(empty, now, true))
	record("bound_not_run", publicationForwardingFailure(reviewReport(true), now, false))
	failed := reviewReport(true)
	failed.Checks[0].Status = api.HealthCheckStatusFailed
	record("check_failed", publicationForwardingFailure(failed, now, true))
	off := reviewReport(true)
	off.MemberForwardingEnabled = false
	record("forwarding_disabled", publicationForwardingFailure(off, now, true))
	tunnel := reviewReport(true)
	tunnel.Checks[2].Status = api.HealthCheckStatusNotRun
	record("tunnel_not_passed", publicationForwardingFailure(tunnel, now, true))
	isolated := reviewReport(true)
	isolated.SafetyStatus = api.SafetyStatusFailed
	record("isolation_unconfirmed", publicationForwardingFailure(isolated, now, true))
	degraded := reviewReport(true)
	degraded.DataHealth = api.DataHealthDegraded
	record("readiness_unconfirmed", publicationForwardingFailure(degraded, now, true))
	unleased := reviewReport(false)
	unleased.Lease.LeaseId.SetNull()
	record("lease_unconfirmed", publicationForwardingFailure(unleased, now, false))
	stale := reviewReport(true)
	stale.AppliedConfigGeneration.Set(0)
	record("stale_generation", publicationForwardingFailure(stale, now, true))
	for reason := range documentedPublicationReasons() {
		seen := false
		for _, got := range reasons {
			if got == reason {
				seen = true
			}
		}
		if !seen {
			t.Fatalf("vocabulary entry %s was not produced", reason)
		}
	}
	if reasons["empty_not_run"] != "" || reasons["bound_not_run"] != "serving_checks_unconfirmed" || reasons["lease_unconfirmed"] != "lease_unconfirmed" {
		t.Fatalf("reasons %+v", reasons)
	}
}
