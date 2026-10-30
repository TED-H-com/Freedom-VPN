package healthattest

import (
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// MaxClockSkew tolerates timestamp rounding between the two local processes.
// The challenge's strict issue/expiry bounds still apply.
const MaxClockSkew = 2 * time.Second

// Validate accepts an attestation only when it is bound to challenge, its
// clock falls inside the challenge window, and the attested agent and
// NetBird versions equal manifestVersions. now is the updater's clock.
// An empty reason means the binding held. ReasonNotRun means the binding
// held and at least one check was not_run; the updater treats that check
// as failed. Any other reason means the attestation is not evidence.
func Validate(challenge Challenge, attestation Attestation, now time.Time, manifestVersions Components) Reason {
	if challenge.Consumed {
		return ReasonConsumed
	}
	if challenge.TxnID == "" || challenge.Nonce == "" || now.IsZero() || challenge.IssuedAt.IsZero() || challenge.ExpiresAt.IsZero() {
		return ReasonMalformed
	}
	if !nonceOK(challenge.Nonce) {
		return ReasonMalformed
	}
	if attestation.TxnID == "" || attestation.Nonce == "" || attestation.GeneratedAt.IsZero() {
		return ReasonMalformed
	}
	if !sameID(attestation.TxnID, challenge.TxnID) {
		return ReasonTxnMismatch
	}
	if attestation.Nonce != challenge.Nonce {
		return ReasonNonceMismatch
	}
	if !sameID(attestation.ReleaseID, challenge.ReleaseID) {
		return ReasonReleaseMismatch
	}
	if !sameID(attestation.NodeID, challenge.NodeID) {
		return ReasonNodeMismatch
	}
	if !sameID(attestation.IncarnationID, challenge.IncarnationID) {
		return ReasonIncarnationMismatch
	}
	if !sameID(attestation.BootID, challenge.BootID) {
		return ReasonBootMismatch
	}
	if !attestation.GeneratedAt.After(challenge.IssuedAt) {
		return ReasonGeneratedBeforeIssue
	}
	if !attestation.GeneratedAt.Before(challenge.ExpiresAt) || !now.Before(challenge.ExpiresAt) {
		return ReasonExpired
	}
	if attestation.GeneratedAt.After(now.Add(MaxClockSkew)) {
		return ReasonGeneratedInFuture
	}
	if !attestation.Published.Valid() {
		return ReasonBadPublication
	}
	if manifestVersions.NodeAgent == "" || attestation.AgentVersion != manifestVersions.NodeAgent {
		return ReasonAgentVersion
	}
	if manifestVersions.Netbird == "" || attestation.NetbirdVersion != manifestVersions.Netbird {
		return ReasonNetbirdVersion
	}
	for _, c := range []Check{attestation.Checks.InternetOnly, attestation.Checks.DNSPolicy, attestation.Checks.TunnelProbe} {
		if !c.Status.Valid() {
			return ReasonUnknownStatus
		}
		if reason := checkReason(c.Reason); reason != "" {
			return reason
		}
	}
	if attestation.Checks.InternetOnly.Status == api.HealthCheckStatusNotRun ||
		attestation.Checks.DNSPolicy.Status == api.HealthCheckStatusNotRun ||
		attestation.Checks.TunnelProbe.Status == api.HealthCheckStatusNotRun {
		return ReasonNotRun
	}
	return ""
}

func nonceOK(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
		default:
			return false
		}
	}
	return true
}

func canonicalID(s string) bool {
	return uuidShape(s) && s == strings.ToLower(s)
}

func sameID(got, want string) bool {
	if !uuidShape(got) || !uuidShape(want) {
		return false
	}
	return strings.EqualFold(got, want)
}

func uuidShape(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

func checkReason(s string) Reason {
	if len(s) > MaxReasonBytes {
		return ReasonBadReason
	}
	// An allowlist, not a character filter: even an ASCII token can contain
	// member data. Keep the producer's fixed reasons and the original D-053
	// contract examples; unknown reasons never reach the journal (SEC02).
	switch s {
	case "", "baseline held", "baseline missing", "policy loaded",
		"policy in force", "policy not in force", "not run", "probe not run",
		"overlay up", "probe failed", "overlay_up", "overlay_down", "overlay_partial",
		"probe reached the lab endpoint", "probe skipped", "probe refused":
		return ""
	default:
		return ReasonBadReason
	}
}
