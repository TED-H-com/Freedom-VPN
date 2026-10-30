//go:build integration

package httpwire_test

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestReviewR11_ZeroBindingRouteBeforeFirstBindingChecks(t *testing.T) {
	// TODO(review): once a zero-binding publication has attached the NetBird
	// route, the first device binding does not withdraw that route or require
	// lease, qos and source_binding for the generation that carries the binding.
	r, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, r)
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
	group := f.group("publication-probe")
	path := "/api/v1/admin/nodes/" + report.NodeId.String()
	detail := f.call("GET", path, nil, nil, nil)
	op := decode[api.Operation](t, f.call("PUT", path+"/publication", api.PublicationRequest{
		PublicationIntent: api.PublicationIntentPublished, Reason: "TEST ONLY probe evidence",
		GroupIds: &[]uuid.UUID{group.GroupId},
	}, nil, map[string]string{"If-Match": detail.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()}), 202)
	reader := desiredstate.NewFakeEntitlementReader()
	compiler := desiredstate.NewCompiler(r.Pool, publicationSigner{r.Provider}, desiredstate.WithEntitlementReader(reader), desiredstate.WithPolicyReader(desiredstate.NewFakePolicyReader(desiredstate.PolicySnapshot{})))
	compiled, err := compiler.Compile(t.Context(), report.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled.Envelope.Signed.DeviceBindings) != 0 || compiled.Envelope.Signed.PublicationIntent != api.PublicationIntentPublished {
		t.Fatal("empty publish did not compile zero bindings", compiled.Envelope.Signed)
	}
	emptyGen := compiled.Envelope.Signed.ConfigGeneration
	report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
	report.AppliedConfigGeneration.Set(emptyGen)
	report.MemberForwardingEnabled, report.DataHealth = true, api.DataHealthHealthy
	for i := range report.Checks {
		report.Checks[i].ObservedAt = report.ObservedAt
		if report.Checks[i].Name == api.NodeCheckNameTunnelProbe {
			report.Checks[i].Status = api.HealthCheckStatusPassed
		}
	}
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
	cause, err := nodes.PublicationDataPathFailure(t.Context(), r.Pool, report, uint64(mustValue(t, op.DesiredGeneration)), time.Now())
	if err != nil || cause != "" {
		t.Fatal("zero-binding gate", cause, err)
	}
	nb := exits.NewFakeNetBirdAdapter()
	dispatcher := f.publicationDispatcher(nb, 1)
	f.publicationDispatch(dispatcher, op.OperationId)
	if nb.RouteCount() != 1 || nb.WithdrawCalls() != 0 {
		t.Fatalf("zero-binding publish route %d withdraws %d", nb.RouteCount(), nb.WithdrawCalls())
	}

	now := time.Now().UTC()
	reader.SetMembers(desiredstate.MemberEntitlement{
		MemberID: uuid.New(), GroupIDs: []uuid.UUID{group.GroupId},
		ValidFrom: now.Add(-time.Hour), EffectiveServiceEnd: now.Add(time.Hour),
		PlanDownBps: 100000000, PlanUpBps: 50000000,
		Bindings: []desiredstate.MemberDeviceBinding{{
			DeviceID: uuid.New(), NetbirdPeerID: "TEST ONLY member peer",
			AllowedIPv4: []string{"100.64.0.2/32"}, BindingRevision: 1,
			ValidFrom: now.Add(-time.Hour), ValidUntil: now.Add(time.Hour),
		}},
	})
	bound, err := compiler.Compile(t.Context(), report.NodeId)
	if err != nil {
		t.Fatal(err)
	}
	boundGen := bound.Envelope.Signed.ConfigGeneration
	if boundGen <= emptyGen || len(bound.Envelope.Signed.DeviceBindings) != 1 {
		t.Fatalf("bound compile gen %d bindings %d", boundGen, len(bound.Envelope.Signed.DeviceBindings))
	}
	// (a) the observation is still the empty generation. The gate must not
	// treat that ack as permission for the generation that now has a binding.
	stale, err := nodes.PublicationDataPathFailure(t.Context(), r.Pool, report, uint64(boundGen), time.Now())
	if err != nil || stale != "stale_generation" {
		t.Fatalf("stale ack of generation %d against desired %d returned %q (%v)", emptyGen, boundGen, stale, err)
	}
	acked := report
	acked.ReportId, acked.ObservedAt = uuid.New(), time.Now().UTC()
	acked.AppliedConfigGeneration.Set(boundGen)
	cause, err = nodes.PublicationDataPathFailure(t.Context(), r.Pool, acked, uint64(boundGen), time.Now())
	if err != nil || cause != "serving_checks_unconfirmed" {
		t.Fatalf("proof inconclusive: bound generation gate %q %v", cause, err)
	}
	var observed string
	if err = r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id=$1`, report.NodeId).Scan(&observed); err != nil {
		t.Fatal(err)
	}
	var intent string
	if err = r.Pool.QueryRow(t.Context(), `SELECT publication_intent FROM exits WHERE node_id=$1`, report.NodeId).Scan(&intent); err != nil {
		t.Fatal(err)
	}
	if nb.RouteCount() == 0 || nb.WithdrawCalls() != 0 {
		t.Fatalf("route was re-gated: count %d withdraws %d", nb.RouteCount(), nb.WithdrawCalls())
	}
	t.Fatalf("generation %d carries peer %q while lease/qos/source_binding are not verified (gate %s). NetBird route count %d, withdraws %d, publication_intent %s, publication_observed %s. The zero-binding publish attached the route for group %s and nothing withdrew it",
		boundGen, bound.Envelope.Signed.DeviceBindings[0].PeerId, cause, nb.RouteCount(), nb.WithdrawCalls(), intent, observed, group.GroupId)
}
