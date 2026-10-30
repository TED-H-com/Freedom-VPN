package nodes

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestD056_DNSPolicyEvidenceDoesNotInventSuccess(t *testing.T) {
	for _, outcome := range []api.DnsPolicyEvidenceOutcome{api.DnsPolicyEvidenceOutcomeApplied, api.DnsPolicyEvidenceOutcomeFailed, api.DnsPolicyEvidenceOutcomePartial, api.DnsPolicyEvidenceOutcomeNotRun} {
		t.Run(string(outcome), func(t *testing.T) {
			report := api.ObservationReport{Checks: []api.NodeCheck{{Name: api.NodeCheckNameDnsPolicy, Status: api.HealthCheckStatusPassed}}}
			report.DnsPolicyEvidence.Set(api.DnsPolicyEvidence{Outcome: outcome})
			if err := validateDNSPolicyEvidence(report); (err == nil) != (outcome == api.DnsPolicyEvidenceOutcomeApplied) {
				t.Fatal("contradictory DNS success accepted", err)
			}
			report.Checks[0].Status = api.HealthCheckStatusFailed
			if err := validateDNSPolicyEvidence(report); err != nil {
				t.Fatal("failure evidence rejected", err)
			}
		})
	}
	report := api.ObservationReport{}
	report.DnsPolicyEvidence.Set(api.DnsPolicyEvidence{Feeds: []api.DnsFeedEvidence{{FeedCode: "test-only", RuleCount: 1}, {FeedCode: "test-only", RuleCount: 2}}})
	if validateDNSPolicyEvidence(report) == nil {
		t.Fatal("duplicate feed_code accepted")
	}
}
