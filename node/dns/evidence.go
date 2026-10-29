package dns

import (
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// DNSPolicyCheckName is the observation check the agent fills from Evidence.
// Contract name: dns_policy. Verified is the only dns_ok signal (§12.5, §17.1).
const DNSPolicyCheckName = api.NodeCheckNameDnsPolicy

// Evidence is the read-back the agent copies into its observation report.
// policy_generation is the decimal generation stored beside the document.
// document_hash is the lowercase SHA-256 of the RFC 8785 resolver document.
type Evidence struct {
	PolicyGeneration string    `json:"policy_generation"`
	DocumentHash     string    `json:"document_hash"`
	AppliedAt        time.Time `json:"applied_at"`
	Verified         bool      `json:"verified"`
}

// DNSOK reports whether the node may advertise dns_ok. The agent maps this to
// checks[name=dns_policy].status and must leave that check failed when false.
func (e Evidence) DNSOK() bool { return e.Verified }

// DNSPolicyStatus is the observation check status for DNSPolicyCheckName.
func (e Evidence) DNSPolicyStatus() api.HealthCheckStatus {
	if e.Verified {
		return api.HealthCheckStatusPassed
	}
	return api.HealthCheckStatusFailed
}
