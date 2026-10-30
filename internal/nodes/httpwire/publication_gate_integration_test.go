//go:build integration

package httpwire_test

import (
	"errors"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
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

func TestAT08_R11_PublicationGateWithdrawsAndReattaches(t *testing.T) {
	// The exact r11 reproduction is retained in testdata: its final assertion
	// fails unconditionally. This companion runs the durable worker after the
	// compile/observation and asserts the safe property in both directions.
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

	if _, err := compiler.GetDesiredState(t.Context(), report.NodeId, report.NodeIncarnationId, ""); !isUnavailable(err) {
		t.Fatal("bound generation escaped before withdrawal", err)
	}
	if _, err := compiler.GetDesiredState(t.Context(), report.NodeId, report.NodeIncarnationId, bound.ETag); !isUnavailable(err) {
		t.Fatal("conditional read bypassed pending withdrawal", err)
	}
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", acked, cert, nil), 202)
	latestGate := func() uuid.UUID {
		t.Helper()
		var id uuid.UUID
		if err := r.Pool.QueryRow(t.Context(), `SELECT id FROM operations WHERE resource_id=$1 AND principal=$2 ORDER BY created_at DESC,id DESC LIMIT 1`, report.NodeId, nodes.PublicationGatePrincipal).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	gate := latestGate()
	if gate == op.OperationId {
		t.Fatal("terminal publication reused")
	}
	// A lost external withdrawal must leave error, never invented read-back.
	nb.SetWithdrawError(errors.New("TEST ONLY unavailable"))
	dispatcher = f.publicationDispatcher(nb, 1)
	f.publicationDispatch(dispatcher, gate)
	var observed, intent string
	read := func() {
		t.Helper()
		if err := r.Pool.QueryRow(t.Context(), `SELECT publication_intent,publication_observed FROM exits WHERE node_id=$1`, report.NodeId).Scan(&intent, &observed); err != nil {
			t.Fatal(err)
		}
		if intent != "published" {
			t.Fatal("gate changed intent", intent)
		}
	}
	read()
	if observed != "error" {
		t.Fatal("unconfirmed withdrawal", observed)
	}
	if _, err := compiler.GetDesiredState(t.Context(), report.NodeId, report.NodeIncarnationId, ""); !isUnavailable(err) {
		t.Fatal("failed withdrawal released binding", err)
	}
	nb.SetWithdrawError(nil)
	// Semantic redelivery gets its own claim; failure evidence is retried with
	// independent read-back and does not modify the successful empty operation.
	replay := func(id uuid.UUID) {
		t.Helper()
		f.sql(`INSERT INTO outbox_events(aggregate_type,aggregate_id,aggregate_version,event_type,schema_version,payload) SELECT aggregate_type,aggregate_id,aggregate_version,event_type,schema_version,payload FROM outbox_events WHERE payload->>'operation_id'=$1 ORDER BY created_at LIMIT 1`, id.String())
		f.publicationDispatch(dispatcher, id)
	}
	replay(gate)
	read()
	if observed != "withdrawn" {
		t.Fatal("gate did not withdraw", observed)
	}
	var exitID uuid.UUID
	if err := r.Pool.QueryRow(t.Context(), `SELECT id FROM exits WHERE node_id=$1`, report.NodeId).Scan(&exitID); err != nil {
		t.Fatal(err)
	}
	remote, err := nb.ReadExitState(t.Context(), r.Config.NetBirdDeployment, exitID, "")
	if err != nil || remote.Enabled {
		t.Fatal("route still attached", remote, err)
	}
	var reason string
	if err := r.Pool.QueryRow(t.Context(), `SELECT steps->'data_path_probe'->>'reason' FROM operations WHERE id=$1`, gate).Scan(&reason); err != nil || reason != "serving_checks_unconfirmed" {
		t.Fatal(reason, err)
	}
	if _, err := compiler.GetDesiredState(t.Context(), report.NodeId, report.NodeIncarnationId, ""); err != nil {
		t.Fatal("withdrawal did not release desired state", err)
	}
	auditCount := func(state string) int {
		t.Helper()
		var count int
		if err := r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE action='node.publish' AND resource=$1 AND after_diff->>'publication_observed'=$2`, "exit:"+exitID.String(), state).Scan(&count); err != nil {
			t.Fatal(err)
		}
		return count
	}
	if auditCount("withdrawn") != 1 {
		t.Fatal("missing withdrawal audit")
	}
	withdraws := nb.WithdrawCalls()
	replay(gate)
	if nb.WithdrawCalls() != withdraws || auditCount("withdrawn") != 1 {
		t.Fatal("withdrawal replay was not idempotent")
	}
	if _, err = r.Provider.(*signing.FileProvider).Rotate(t.Context(), signing.Lease, 48*time.Hour, time.Hour); err != nil {
		t.Fatal(err)
	}
	lease := decode[api.ServingLeaseEnvelope](t, f.call("POST", "/api/v1/agent/lease-renewals", api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: report.NodeId, NodeIncarnationId: report.NodeIncarnationId, CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullableWithValue(decstr.U64(boundGen))}, cert, nil), 200)
	acked.ReportId, acked.ObservedAt = uuid.New(), time.Now().UTC()
	acked.Lease.LeaseId.Set(lease.Signed.LeaseId)
	acked.Lease.ExpiresAt.Set(lease.Signed.ExpiresAt)
	for i := range acked.Checks {
		acked.Checks[i].Status = api.HealthCheckStatusPassed
		acked.Checks[i].ObservedAt = acked.ObservedAt
	}
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", acked, cert, nil), 202)
	dispatcher = f.publicationDispatcher(nb, 1)
	recovery := latestGate()
	if recovery == gate {
		t.Fatal("exhausted gate was reset")
	}
	nb.SetReadbackError(errors.New("TEST ONLY uncertain reattachment"))
	f.publicationDispatch(dispatcher, recovery)
	read()
	if observed != "error" {
		t.Fatal("uncertain reattachment claimed withdrawn", observed)
	}
	nb.SetReadbackError(nil)
	replay(recovery)
	read()
	remote, err = nb.ReadExitState(t.Context(), r.Config.NetBirdDeployment, exitID, "")
	if observed != "active" || err != nil || !remote.Enabled || f.publicationOperation(recovery).Status != api.OperationStatusSucceeded {
		t.Fatal("route not reattached", observed, remote, err)
	}
	if auditCount("active") != 2 {
		t.Fatal("missing reattachment audit")
	}
	calls := nb.PublishCalls()
	replay(recovery)
	if nb.PublishCalls() != calls || auditCount("active") != 2 {
		t.Fatal("reattachment replay duplicated effect")
	}
	if f.publicationOperation(op.OperationId).Status != api.OperationStatusSucceeded {
		t.Fatal("historical publication changed")
	}
}

func isUnavailable(err error) bool {
	var e *apierr.Error
	return errors.As(err, &e) && e.Code == apierr.DependencyUnavailable
}
