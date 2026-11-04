package agent

import (
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

// TestF53_F1701_StaleLeaseHoldReportQosDetail proves the exact 5-step mechanism
// behind Isolation F17-01 without changing product behavior:
//  1. Decide() itself sets qos.Detail = "Rate policy could not be applied." and
//     source_binding.Detail = "No member binding was admitted." (with
//     internet_only Status = "passed" and Detail = nil) on a stale-lease step.
//  2. Because the node previously served a populated generation in the same
//     Engine process, e.servingRead is non-nil (retaining the prior generation's
//     verified live read) and is not cleared when the node withdraws to baseline.
//  3. Engine.report() -> attachServingRead() unconditionally overwrites the
//     failed qos check's Detail with "read_at=... source=live candidate=<G> verified=true ...".
//  4. Only when e.servingRead is nil (an Engine process that has never run Apply
//     or verifyServingModel) does Engine.report() preserve Decide()'s
//     "Rate policy could not be applied." detail on qos.
func TestF53_F1701_StaleLeaseHoldReportQosDetail(t *testing.T) {
	r := newRig(t)
	keep, _ := servePair(t, r)
	if r.eng.servingRead == nil || !r.eng.servingRead.Verified || r.eng.servingRead.Source != "live" {
		t.Fatalf("expected retained verified live servingRead after serving populated generation, got %+v", r.eng.servingRead)
	}
	servedGen := r.eng.servingRead.Generation

	// Hold the lease at generation 3 while Desired State advances to generation 4.
	cacheStickyLease(t, r)
	r.srv.SetFixture(fixtureWithBindings(4, []api.DeviceBinding{keep}))
	stepOK(t, r)

	if r.eng.serving {
		t.Fatal("expected engine to withdraw serving on stale-generation lease")
	}
	if r.eng.servingRead == nil || r.eng.servingRead.Generation != servedGen || !r.eng.servingRead.Verified {
		t.Fatalf("expected servingRead from generation %d to remain retained on baseline withdrawal, got %+v", servedGen, r.eng.servingRead)
	}

	// Step 1: Decide() produces both exact dependent details and internet_only passed with nil Detail.
	d := Decide(r.eng.input())
	if d.Serve || !hasReason(d.Reasons, api.ForwardingBlockReasonPolicyMissing) {
		t.Fatalf("Decide: Serve=%v Reasons=%v, want Serve=false with policy_missing", d.Serve, d.Reasons)
	}
	iso, ok := checkBy(d.Checks, api.NodeCheckNameInternetOnly)
	if !ok || iso.Status != api.HealthCheckStatusPassed || !iso.ErrorCode.IsNull() || iso.Detail != nil {
		t.Fatalf("Decide internet_only check: %+v, want status=passed error_code=null detail=nil", iso)
	}
	leaseChk, ok := checkBy(d.Checks, api.NodeCheckNameLease)
	if !ok || leaseChk.Status != api.HealthCheckStatusFailed || codeOf(t, leaseChk) != api.ErrorCodeSTALEGENERATION || detailValue(leaseChk) != "Serving lease is not for the applied generation." {
		t.Fatalf("Decide lease check: status=%s code=%s detail=%q", leaseChk.Status, codeOf(t, leaseChk), detailValue(leaseChk))
	}
	qosDecide, ok := checkBy(d.Checks, api.NodeCheckNameQos)
	if !ok || qosDecide.Status != api.HealthCheckStatusFailed || codeOf(t, qosDecide) != api.ErrorCodeVALIDATIONFAILED || detailValue(qosDecide) != "Rate policy could not be applied." {
		t.Fatalf("Decide qos check: status=%s code=%s detail=%q", qosDecide.Status, codeOf(t, qosDecide), detailValue(qosDecide))
	}
	bindDecide, ok := checkBy(d.Checks, api.NodeCheckNameSourceBinding)
	if !ok || bindDecide.Status != api.HealthCheckStatusFailed || codeOf(t, bindDecide) != api.ErrorCodeVALIDATIONFAILED || detailValue(bindDecide) != "No member binding was admitted." {
		t.Fatalf("Decide source_binding check: status=%s code=%s detail=%q", bindDecide.Status, codeOf(t, bindDecide), detailValue(bindDecide))
	}

	// Step 2: Engine.report() -> attachServingRead() overwrites qos.Detail with the
	// retained servingRead diagnostic whenever e.servingRead != nil.
	rep := mustReport(t, r, t.Context())
	requireObservationSchema(t, rep)
	qosReport, ok := checkBy(rep.Checks, api.NodeCheckNameQos)
	if !ok || qosReport.Status != api.HealthCheckStatusFailed || codeOf(t, qosReport) != api.ErrorCodeVALIDATIONFAILED {
		t.Fatalf("Report qos check: %+v", qosReport)
	}
	if !strings.HasPrefix(detailValue(qosReport), "read_at=") || !strings.Contains(detailValue(qosReport), "source=live") || !strings.Contains(detailValue(qosReport), "verified=true") {
		t.Fatalf("Report qos detail: got %q, want retained read_at=... source=live ... verified=true ...", detailValue(qosReport))
	}
	bindReport, ok := checkBy(rep.Checks, api.NodeCheckNameSourceBinding)
	if !ok || detailValue(bindReport) != "No member binding was admitted." {
		t.Fatalf("Report source_binding detail: got %q, want %q", detailValue(bindReport), "No member binding was admitted.")
	}

	// Step 3: Only when e.servingRead is nil does Engine.report() preserve Decide()'s
	// "Rate policy could not be applied." detail on qos.
	savedRead := r.eng.servingRead
	r.eng.servingRead = nil
	t.Cleanup(func() { r.eng.servingRead = savedRead })

	repNoRead := mustReport(t, r, t.Context())
	requireObservationSchema(t, repNoRead)
	qosNoRead, ok := checkBy(repNoRead.Checks, api.NodeCheckNameQos)
	if !ok || detailValue(qosNoRead) != "Rate policy could not be applied." {
		t.Fatalf("Report qos detail with nil servingRead: got %q, want %q", detailValue(qosNoRead), "Rate policy could not be applied.")
	}
}
