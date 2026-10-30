//go:build integration

package nodestest

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// BaselineObservation uses enrollment and the real compiler to create an
// unpublished node, then returns its TEST ONLY baseline acknowledgement. It
// never writes node/desired-state projections directly (D-050).
func BaselineObservation(t testing.TB, examplePath string) (*httpwire.Runtime, enrollment.EnrollResult, api.ObservationReport) {
	t.Helper()
	cfg := New(t, NetBird{})
	r, err := httpwire.Open(t.Context(), dbtest.New(t), cfg.Nodes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Provider.(*signing.FileProvider).Rotate(t.Context(), signing.DesiredState, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	created, err := r.Enrollment.CreateEnrollment(t.Context(), enrollment.Command{Principal: "admin:" + uuid.NewString(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller})
	if err != nil {
		t.Fatal(err)
	}
	token, err := created.Token.Get()
	if err != nil {
		t.Fatal(err)
	}
	request, _ := pkitest.Request(t)
	enrolled, err := r.Enrollment.Enroll(t.Context(), token, request)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := r.DesiredStateCompiler().Compile(t.Context(), enrolled.Response.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatal(err)
	}
	var report api.ObservationReport
	if err = json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	report.NodeId, report.NodeIncarnationId, report.ReportId = enrolled.Response.NodeId, enrolled.Response.NodeIncarnationId, uuid.New()
	report.ObservedAt = time.Now().UTC().Truncate(time.Microsecond)
	report.CurrentReleaseId.SetNull()
	report.LastGoodReleaseId.SetNull()
	report.Netbird.PeerId.SetNull()
	report.Lease.LeaseId.SetNull()
	report.Lease.ExpiresAt.SetNull()
	report.AppliedConfigGeneration.Set(compiled.Envelope.Signed.ConfigGeneration)
	report.MemberForwardingEnabled = false
	report.ProvisioningState, report.SafetyStatus, report.DataHealth = api.ProvisioningStateReady, api.SafetyStatusPassed, api.DataHealthDegraded
	report.Checks = nil
	for _, name := range []api.NodeCheckName{api.NodeCheckNameInternetOnly, api.NodeCheckNameMeterPath, api.NodeCheckNameTunnelProbe, api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		status := api.HealthCheckStatusNotRun
		if name == api.NodeCheckNameInternetOnly || name == api.NodeCheckNameMeterPath {
			status = api.HealthCheckStatusPassed
		}
		check := api.NodeCheck{Name: name, Status: status, ObservedAt: report.ObservedAt}
		check.ErrorCode.SetNull()
		report.Checks = append(report.Checks, check)
	}
	return r, enrolled, report
}
