package agent

import (
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// D-157 is one opportunity per verified closed Apply, not per empty lease or
// report. It is process-local: restart does not recreate a closed Apply receipt.
type recoveryRenewal struct {
	armed    bool
	reportID uuid.UUID
	accepted bool
}

func (e *Engine) recoveryRenewalEligible() bool {
	return e.closedRecoveryHeld && !e.serving && e.applyError == nil &&
		!e.revoked && !e.identityFrozen && !e.freezeUncertain && e.storageOK &&
		!e.quarantinePending && !e.memberPathPending && !e.dependencyMissing &&
		e.appliedGen > 0 && e.desiredOK && !e.desiredExpired && e.desired != nil &&
		uint64(e.desired.ConfigGeneration) == e.appliedGen && e.leaseGrantless() &&
		!e.leaseBehindApplied()
}

// Bind to a newly queued report from this recovery. An old queued report, even
// at the same generation, cannot arm a later recovery's opportunity.
func (e *Engine) noteRecoveryReport(rep api.ObservationReport) {
	if e.canBindRecoveryReport(rep) {
		e.closedRecoveryRenewal.reportID = rep.ReportId
	}
}

// The pending-outbox exception and the binding use this exact predicate. An
// armed recovery can remain unbindable; it must not disable ordinary dedup.
func (e *Engine) canBindRecoveryReport(rep api.ObservationReport) bool {
	r := &e.closedRecoveryRenewal
	if !r.armed || r.reportID != uuid.Nil || !e.recoveryRenewalEligible() {
		return false
	}
	gen, err := rep.AppliedConfigGeneration.Get()
	if err != nil || uint64(gen) != e.appliedGen || rep.MemberForwardingEnabled ||
		rep.SafetyStatus != api.SafetyStatusPassed || rep.ProvisioningState != api.ProvisioningStateReady ||
		rep.Netbird.Backend != api.ObservationReportNetbirdBackendKernel || !rep.Netbird.PeerId.IsNull() {
		return false
	}
	// These are central's synchronous readiness evidence requirements. A failed
	// meter path is never the D-067 awaiting-grants exception.
	internet, awaiting := false, false
	for _, reason := range rep.ForwardingBlockReasons {
		awaiting = awaiting || reason == api.ForwardingBlockReasonAwaitingMemberGrants
	}
	for _, check := range rep.Checks {
		if check.Name == api.NodeCheckNameInternetOnly {
			internet = check.Status == api.HealthCheckStatusPassed
		}
		if check.Name == api.NodeCheckNameMeterPath && check.Status == api.HealthCheckStatusFailed {
			return false
		}
	}
	for _, err := range rep.Errors {
		switch err.Code {
		case api.ErrorCodeISOLATIONPOLICYINVALID, api.ErrorCodeISOLATIONBASELINEMISSING, api.ErrorCodeMETERPATHINVALID:
			return false
		}
	}
	return internet && awaiting
}

func (e *Engine) acceptRecoveryReport(rep api.ObservationReport, accepted api.ObservationAccepted) {
	r := &e.closedRecoveryRenewal
	if r.armed && r.reportID != uuid.Nil && rep.ReportId == r.reportID &&
		accepted.ReportId == rep.ReportId && bool(accepted.Accepted) &&
		(accepted.IdentityFrozen == nil || !*accepted.IdentityFrozen) && e.recoveryRenewalEligible() {
		r.accepted = true
	}
}

func (e *Engine) earlyRecoveryRenewal() bool {
	return e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.accepted && e.recoveryRenewalEligible()
}
