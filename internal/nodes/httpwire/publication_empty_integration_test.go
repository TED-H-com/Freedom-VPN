//go:build integration

package httpwire_test

import (
	"context"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

type publicationSigner struct{ signing.Provider }

func (s publicationSigner) Sign(ctx context.Context, purpose envelope.Purpose, value any) (envelope.Envelope, error) {
	return s.SignEnvelope(ctx, signing.Purpose(purpose), value)
}

// D-039/D-050: real enrollment, observation, publication, compiler and dispatcher;
// only the entitlement input and NetBird read-back use TEST ONLY adapters. The
// binding count is proven in the compiler's signed output, never patched in SQL.
func TestAT08_PublicationWorkerEmptyBindingEvidence(t *testing.T) {
	for _, tc := range []struct {
		name     string
		bindings bool
		failed   bool
		stale    bool
		want     string
	}{
		{"empty_not_run", false, false, false, ""},
		{"bound_not_run", true, false, false, "serving_checks_unconfirmed"},
		{"empty_failed_lease", false, true, false, "check_failed"},
		{"bound_failed_lease", true, true, false, "check_failed"},
		{"older_generation", false, false, true, "stale_generation"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
			f := setupRuntime(t, r)
			cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
			group := f.group("publication-probe")
			path := "/api/v1/admin/nodes/" + report.NodeId.String()
			detail := f.call("GET", path, nil, nil, nil)
			op := decode[api.Operation](t, f.call("PUT", path+"/publication", api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, Reason: "TEST ONLY probe evidence", GroupIds: &[]uuid.UUID{group.GroupId}}, nil, map[string]string{"If-Match": detail.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()}), 202)
			reader := desiredstate.NewFakeEntitlementReader()
			now := time.Now().UTC()
			if tc.bindings {
				reader.SetMembers(desiredstate.MemberEntitlement{MemberID: uuid.New(), GroupIDs: []uuid.UUID{group.GroupId}, ValidFrom: now.Add(-time.Hour), EffectiveServiceEnd: now.Add(time.Hour), PlanDownBps: 100000000, PlanUpBps: 50000000,
					Bindings: []desiredstate.MemberDeviceBinding{{DeviceID: uuid.New(), NetbirdPeerID: "TEST ONLY member peer", AllowedIPv4: []string{"100.64.0.2/32"}, BindingRevision: 1, ValidFrom: now.Add(-time.Hour), ValidUntil: now.Add(time.Hour)}}})
			}
			compiler := desiredstate.NewCompiler(r.Pool, publicationSigner{r.Provider}, desiredstate.WithEntitlementReader(reader), desiredstate.WithPolicyReader(desiredstate.NewFakePolicyReader(desiredstate.PolicySnapshot{})))
			compiled, err := compiler.Compile(t.Context(), report.NodeId)
			if err != nil {
				t.Fatal(err)
			}
			wantBindings := 0
			if tc.bindings {
				wantBindings = 1
			}
			if len(compiled.Envelope.Signed.DeviceBindings) != wantBindings || compiled.Envelope.Signed.PublicationIntent != api.PublicationIntentPublished {
				t.Fatal("wrong signed input", compiled.Envelope.Signed)
			}
			report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
			report.AppliedConfigGeneration.Set(compiled.Envelope.Signed.ConfigGeneration)
			report.MemberForwardingEnabled, report.DataHealth = true, api.DataHealthHealthy
			for i := range report.Checks {
				report.Checks[i].ObservedAt = report.ObservedAt
				if report.Checks[i].Name == api.NodeCheckNameTunnelProbe {
					report.Checks[i].Status = api.HealthCheckStatusPassed
				}
				if tc.failed && report.Checks[i].Name == api.NodeCheckNameLease {
					report.Checks[i].Status = api.HealthCheckStatusFailed
				}
			}
			if tc.stale {
				report.AppliedConfigGeneration.Set(compiled.Envelope.Signed.ConfigGeneration - 1)
			}
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
			cause, err := nodes.PublicationDataPathFailure(t.Context(), r.Pool, report, uint64(mustValue(t, op.DesiredGeneration)), time.Now())
			if err != nil || cause != tc.want {
				t.Fatal("gate", cause, err)
			}
			// The observation path must agree with the worker's gate.
			before := f.publicationOperation(op.OperationId)
			for _, step := range before.Steps {
				if step.Name == nodes.StepDataPathProbe && (step.Status == api.OperationStepStatusSucceeded) != (tc.want == "") {
					t.Fatal("observation gate diverged", step)
				}
			}
			nb := exits.NewFakeNetBirdAdapter()
			dispatcher := f.publicationDispatcher(nb, 2)
			f.publicationDispatch(dispatcher, op.OperationId)
			after := f.publicationOperation(op.OperationId)
			wantStatus := api.OperationStatusSucceeded
			if tc.want != "" {
				wantStatus = api.OperationStatusRunning
			}
			if after.Status != wantStatus || (tc.want != "" && !after.Error.IsNull()) {
				t.Fatal("worker outcome", after)
			}
			var observed string
			if err = r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id=$1`, report.NodeId).Scan(&observed); err != nil {
				t.Fatal("publication_observed query", err)
			}
			if tc.want == "" {
				if observed != "active" {
					t.Fatal("active projection", observed)
				}
				for _, step := range after.Steps {
					if step.Status != api.OperationStepStatusSucceeded || step.ObservedAt.IsNull() {
						t.Fatal("missing evidence", step)
					}
				}
				if nb.RouteCount() != 1 {
					t.Fatal("route not confirmed")
				}
			} else {
				if observed != "absent" {
					t.Fatalf("gated publish changed publication_observed to %q before exhaustion, want absent (unchanged)", observed)
				}
				stepKey := nodes.StepDataPathProbe
				if tc.stale {
					stepKey = nodes.StepAgentPolicy
				}
				for _, step := range after.Steps {
					if step.Name == stepKey {
						code, cErr := step.ErrorCode.Get()
						msg, mErr := step.Message.Get()
						if step.Status != api.OperationStepStatusPending || cErr != nil || code != api.ErrorCodeNODENOTSERVICEABLE || mErr != nil || msg != "member_forwarding_unconfirmed" {
							t.Fatalf("gated step %s = %+v, want pending with NODE_NOT_SERVICEABLE / member_forwarding_unconfirmed", stepKey, step)
						}
					}
				}
				var reason, message string
				if err = r.Pool.QueryRow(t.Context(), `SELECT steps->$2->>'reason',steps->$2->>'error_message' FROM operations WHERE id=$1`, op.OperationId, stepKey).Scan(&reason, &message); err != nil || reason != tc.want || message != "member_forwarding_unconfirmed" {
					t.Fatal("bounded reason lost", stepKey, reason, message, err)
				}
				f.publicationDispatch(dispatcher, op.OperationId)
				exhausted := f.publicationOperation(op.OperationId)
				if exhausted.Status != api.OperationStatusFailed || exhausted.Error.IsNull() {
					t.Fatal("exhausted outcome", exhausted)
				}
				if err = r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id=$1`, report.NodeId).Scan(&observed); err != nil || observed != "error" {
					t.Fatal("exhausted publication_observed", observed, err)
				}
			}
		})
	}
}
