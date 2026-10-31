//go:build integration

package httpwire_test

import (
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/devices"
	"github.com/TED-H-com/Freedom-VPN/internal/entitlement"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/members"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
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

	if empty, err := nodes.PublicationAttachmentEmpty(t.Context(), r.Pool, report.NodeId); err != nil || !empty {
		t.Fatal("empty attachment basis", empty, err)
	}
	// Another empty generation must neither queue a gate nor erase the basis
	// of the historical successful generation when the latest cache advances.
	if _, err := compiler.Compile(t.Context(), report.NodeId); err != nil {
		t.Fatal(err)
	}
	if publicationGateCount(t, f, report.NodeId) != 0 {
		t.Fatal("empty->empty compile queued a gate")
	}
	if empty, err := nodes.PublicationAttachmentEmpty(t.Context(), r.Pool, report.NodeId); err != nil || !empty {
		t.Fatal("cache replacement erased empty basis", empty, err)
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

	// Once a populated generation succeeds (including a system recovery), its
	// attachment remains across future generations and failed/not-run reports.
	if empty, err := nodes.PublicationAttachmentEmpty(t.Context(), r.Pool, report.NodeId); err != nil || empty {
		t.Fatal("recovery did not establish populated basis", empty, err)
	}
	beforeGateCount := publicationGateCount(t, f, report.NodeId)
	further, err := compiler.Compile(t.Context(), report.NodeId)
	if err != nil || further.Generation <= bound.Generation {
		t.Fatal("further compile", further.Generation, err)
	}
	if _, err = compiler.GetDesiredState(t.Context(), report.NodeId, report.NodeIncarnationId, ""); err != nil {
		t.Fatal("populated recovery blocked later delivery", err)
	}
	acked.ReportId, acked.ObservedAt = uuid.New(), time.Now().UTC()
	acked.AppliedConfigGeneration.Set(further.Envelope.Signed.ConfigGeneration)
	for i := range acked.Checks {
		if acked.Checks[i].Name == api.NodeCheckNameLease {
			acked.Checks[i].Status = api.HealthCheckStatusFailed
		}
	}
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", acked, cert, nil), 202)
	if publicationGateCount(t, f, report.NodeId) != beforeGateCount || nb.WithdrawCalls() != withdraws {
		t.Fatal("populated recovery basis queued/withdrew on a later generation")
	}
}

func isUnavailable(err error) bool {
	var e *apierr.Error
	return errors.As(err, &e) && e.Code == apierr.DependencyUnavailable
}

// D-050: only identity/plan source rows are seeded; grants, device bindings,
// observations, envelopes and operations are produced by their owning services.
func TestAT08_PopulatedPublicationGenerationDelivery(t *testing.T) {
	for _, remove := range []bool{false, true} {
		name := "add_second_binding"
		if remove {
			name = "remove_binding_netbird_unavailable"
		}
		t.Run(name, func(t *testing.T) {
			r, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
			f := setupRuntime(t, r)
			cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
			group := f.group("populated-gate")
			// TEST ONLY policy source; no serving projection is fabricated.
			f.activate("release_support", uuid.Nil, 1, `{}`)
			path := "/api/v1/admin/nodes/" + report.NodeId.String()
			detail := f.call("GET", path, nil, nil, nil)
			op := decode[api.Operation](t, f.call("PUT", path+"/publication", api.PublicationRequest{PublicationIntent: api.PublicationIntentPublished, Reason: "TEST ONLY populated publication", GroupIds: &[]uuid.UUID{group.GroupId}}, nil, map[string]string{"If-Match": detail.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()}), 202)
			store, err := entitlement.NewStore(entitlement.TestConfig())
			if err != nil {
				t.Fatal(err)
			}
			member, identity, plan, revision := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			now := time.Now().UTC()
			f.sql(`INSERT INTO members(id,status) VALUES($1,'active')`, member)
			f.sql(`INSERT INTO identity_bindings(id,member_id,issuer,client_scope,subject,stable_claim_ref,binding_method,verified_at) VALUES($1,$2,'https://id.example.invalid','portal','TEST ONLY user','sub','oidc',$3)`, identity, member, now)
			f.sql(`INSERT INTO netbird_user_bindings(id,member_id,deployment_id,external_user_id,identity_binding_id,mapping_proof,revision,verified_at) VALUES($1,$2,$3,'TEST ONLY user',$4,'shared_issuer_public_subject:https://id.example.invalid',1,$5)`, uuid.New(), member, r.Config.NetBirdDeployment, identity, now)
			f.sql(`INSERT INTO plans(id,code) VALUES($1,'TEST ONLY plan')`, plan)
			f.sql(`INSERT INTO plan_revisions(id,plan_id,revision,device_limit,exit_group_ids,schema_version,payload) VALUES($1,$2,1,3,$3,1,'{"rate_limit":{"down_bps":100000000,"up_bps":50000000}}')`, revision, plan, []uuid.UUID{group.GroupId})
			command := func(role string, actor uuid.UUID) entitlement.Command {
				return entitlement.Command{Actor: entitlement.Actor{Role: role, ID: actor, RequestID: uuid.New()}, Key: uuid.New()}
			}
			memberService := &members.Service{Pool: r.Pool, Entitlements: store}
			start := now.Add(-time.Hour)
			if _, err := memberService.Grant(t.Context(), command("admin", f.p.ID), member, api.GrantCreateRequest{Source: api.GrantSourceAdminGrant, ValidFrom: &start, ValidUntil: now.Add(12 * time.Hour), Reason: "TEST ONLY finite grant", PlanRevisionId: &revision}); err != nil {
				t.Fatal(err)
			}
			deviceService := &devices.Service{Pool: r.Pool, Entitlements: store}
			authorize := func(ip string) uuid.UUID {
				t.Helper()
				result, err := deviceService.AuthorizeDevice(t.Context(), command("system", nodes.SystemActorUUID), devices.Peer{DeploymentID: r.Config.NetBirdDeployment, ID: "TEST ONLY " + ip, ExternalUserID: "TEST ONLY user", OverlayScope: "TEST ONLY", OverlayIP: netip.MustParseAddr(ip), Platform: "linux"})
				if err != nil {
					t.Fatal(err)
				}
				return result.DeviceID
			}
			authorize("100.64.0.2")
			var second uuid.UUID
			if remove {
				second = authorize("100.64.0.3")
			}
			initial := decode[api.DesiredStateEnvelope](t, f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil), 200)
			wantInitial := 1
			if remove {
				wantInitial = 2
			}
			if len(initial.Signed.DeviceBindings) != wantInitial {
				t.Fatal("missing initial bindings", initial.Signed.DeviceBindings)
			}
			if _, err = r.Provider.(*signing.FileProvider).Rotate(t.Context(), signing.Lease, 48*time.Hour, time.Hour); err != nil {
				t.Fatal(err)
			}
			lease := decode[api.ServingLeaseEnvelope](t, f.call("POST", "/api/v1/agent/lease-renewals", api.LeaseRenewalRequest{RenewalRequestId: uuid.New(), NodeId: report.NodeId, NodeIncarnationId: report.NodeIncarnationId, CurrentLeaseId: presence.NewNullNullable[uuid.UUID](), AppliedConfigGeneration: presence.NewNullableWithValue(initial.Signed.ConfigGeneration)}, cert, nil), 200)
			report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
			report.AppliedConfigGeneration.Set(initial.Signed.ConfigGeneration)
			report.MemberForwardingEnabled, report.DataHealth = true, api.DataHealthHealthy
			report.Lease.LeaseId.Set(lease.Signed.LeaseId)
			report.Lease.ExpiresAt.Set(lease.Signed.ExpiresAt)
			for i := range report.Checks {
				report.Checks[i].Status, report.Checks[i].ObservedAt = api.HealthCheckStatusPassed, report.ObservedAt
			}
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
			nb := exits.NewFakeNetBirdAdapter()
			dispatcher := f.publicationDispatcher(nb, 1)
			f.publicationDispatch(dispatcher, op.OperationId)
			if f.publicationOperation(op.OperationId).Status != api.OperationStatusSucceeded {
				t.Fatal("initial populated publication failed")
			}
			if remove {
				nb.SetReadbackError(errors.New("TEST ONLY unavailable"))
				nb.SetWithdrawError(errors.New("TEST ONLY unavailable"))
				if _, err := deviceService.RevokeDevice(t.Context(), command("member", member), second); err != nil {
					t.Fatal(err)
				}
			} else {
				authorize("100.64.0.3")
			}
			compiled, err := r.CompileControlState(t.Context(), report.NodeId, report.NodeIncarnationId)
			if err != nil {
				t.Fatal(err)
			}
			wantBindings := 2
			if remove {
				wantBindings = 1
			}
			if compiled.Envelope.Signed.ConfigGeneration <= initial.Signed.ConfigGeneration || len(compiled.Envelope.Signed.DeviceBindings) != wantBindings {
				t.Fatal("new generation missing", compiled.Envelope.Signed)
			}
			var gates int
			if err = r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM operations WHERE resource_id=$1 AND principal=$2`, report.NodeId, nodes.PublicationGatePrincipal).Scan(&gates); err != nil {
				t.Fatal(err)
			}
			if gates != 0 {
				t.Errorf("populated attachment queued %d gate operations", gates)
			}
			w := f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil)
			if w.Code != 200 {
				t.Errorf("populated generation GET = %d, want 200: %s", w.Code, w.Body.String())
			} else {
				got := decode[api.DesiredStateEnvelope](t, w, 200)
				if got.Signed.ConfigGeneration != compiled.Envelope.Signed.ConfigGeneration || len(got.Signed.DeviceBindings) != wantBindings {
					t.Error("GET did not deliver changed bindings")
				}
			}
			if gates != 0 {
				var id uuid.UUID
				if err := r.Pool.QueryRow(t.Context(), `SELECT id FROM operations WHERE resource_id=$1 AND principal=$2 ORDER BY created_at LIMIT 1`, report.NodeId, nodes.PublicationGatePrincipal).Scan(&id); err != nil {
					t.Fatal(err)
				}
				f.publicationDispatch(dispatcher, id)
			}
			if nb.WithdrawCalls() != 0 {
				t.Errorf("populated route withdrawn %d times", nb.WithdrawCalls())
			}
			// An ordinary current observation does not queue a recovery gate.
			report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
			report.AppliedConfigGeneration.Set(compiled.Envelope.Signed.ConfigGeneration)
			for i := range report.Checks {
				if report.Checks[i].Name == api.NodeCheckNameLease {
					report.Checks[i].Status = api.HealthCheckStatusFailed
				}
			}
			decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
			if publicationGateCount(t, f, report.NodeId) != 0 {
				t.Error("populated failed observation queued recovery")
			}
			// A requested publish execution waits in pending/running while retries
			// remain without changing publication_observed or withdrawing the route,
			// and records error only when the retry budget exhausts (D-074).
			retry := f.publicationCommand(report.NodeId, api.PublicationIntentPublished)
			dispatcher = f.publicationDispatcher(nb, 2)
			f.publicationDispatch(dispatcher, retry.OperationId)
			waiting := f.publicationOperation(retry.OperationId)
			if waiting.Status != api.OperationStatusRunning || !waiting.Error.IsNull() {
				t.Fatalf("populated gated publish did not wait: %+v", waiting)
			}
			var observed string
			if err := r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id=$1`, report.NodeId).Scan(&observed); err != nil {
				t.Fatal(err)
			}
			if observed != "active" || nb.WithdrawCalls() != 0 {
				t.Errorf("populated waiting gate changed route semantics: observed=%s withdraws=%d", observed, nb.WithdrawCalls())
			}
			f.publicationDispatch(dispatcher, retry.OperationId)
			exhausted := f.publicationOperation(retry.OperationId)
			if exhausted.Status != api.OperationStatusFailed || exhausted.Error.IsNull() {
				t.Fatalf("populated exhausted gate did not fail: %+v", exhausted)
			}
			if err := r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id=$1`, report.NodeId).Scan(&observed); err != nil {
				t.Fatal(err)
			}
			if observed != "error" || nb.WithdrawCalls() != 0 {
				t.Error("populated failure changed route semantics", observed, nb.WithdrawCalls())
			}

		})
	}
}

func TestD074_GatedNodePublishWaitsThenCompletesOrExhausts(t *testing.T) {
	t.Run("a_stale_generation_waits_and_completes_on_next_observation", func(t *testing.T) {
		f, enrolled, cert, _, _ := publicationFixture(t)
		nb := exits.NewFakeNetBirdAdapter()
		op := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)
		d := f.publicationDispatcher(nb, 3)

		// Attempt 1 runs while applied_config_generation < desired_config_generation.
		f.publicationDispatch(d, op.OperationId)
		waiting := f.publicationOperation(op.OperationId)
		if waiting.Status != api.OperationStatusRunning || !waiting.Error.IsNull() {
			t.Fatalf("attempt 1 op = (status=%s, error=%+v), want running with null error", waiting.Status, waiting.Error)
		}
		var stepStatus, errCode, errMsg, reason, pubObserved string
		if err := f.r.Pool.QueryRow(t.Context(), `
			SELECT o.steps->'agent_policy'->>'status',
			       o.steps->'agent_policy'->>'error_code',
			       o.steps->'agent_policy'->>'error_message',
			       o.steps->'agent_policy'->>'reason',
			       e.publication_observed
			FROM operations o
			JOIN exits e ON e.node_id = o.resource_id
			WHERE o.id = $1`, op.OperationId).Scan(&stepStatus, &errCode, &errMsg, &reason, &pubObserved); err != nil {
			t.Fatal(err)
		}
		if stepStatus != "pending" || errCode != string(apierr.NodeNotServiceable) || errMsg != "member_forwarding_unconfirmed" || reason != "stale_generation" || pubObserved != "absent" {
			t.Fatalf("attempt 1 gate state = (step=%s, code=%s, msg=%s, reason=%s, observed=%s), want (pending, NODE_NOT_SERVICEABLE, member_forwarding_unconfirmed, stale_generation, absent)",
				stepStatus, errCode, errMsg, reason, pubObserved)
		}
		for _, st := range waiting.Steps {
			if st.Name == nodes.StepAgentPolicy {
				code, cErr := st.ErrorCode.Get()
				msg, mErr := st.Message.Get()
				if st.Status != api.OperationStepStatusPending || cErr != nil || code != api.ErrorCodeNODENOTSERVICEABLE || mErr != nil || msg != "member_forwarding_unconfirmed" {
					t.Fatalf("API agent_policy step = %+v, want pending with NODE_NOT_SERVICEABLE / member_forwarding_unconfirmed", st)
				}
			}
			if st.Name == nodes.StepDataPathProbe && st.Status != api.OperationStepStatusPending {
				t.Fatalf("API data_path_probe step = %+v, want pending", st)
			}
		}

		// Next observation acknowledging the generation satisfies agent_policy and data_path_probe,
		// wakes the outbox event, and completes the operation on the woken worker retry.
		f.acknowledgePublication(enrolled, cert, true)
		f.sql(`UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type<>$1`, desiredstate.EventNodePublicationChanged)
		var availableNow bool
		if err := f.r.Pool.QueryRow(t.Context(), `
			SELECT available_at <= clock_timestamp()
			FROM outbox_events
			WHERE event_type = $1 AND payload->>'operation_id' = $2 AND delivered_at IS NULL`,
			desiredstate.EventNodePublicationChanged, op.OperationId.String()).Scan(&availableNow); err != nil || !availableNow {
			t.Fatalf("acknowledging observation did not wake outbox event: availableNow=%v err=%v", availableNow, err)
		}
		if worked, err := d.Once(t.Context()); err != nil || !worked {
			t.Fatalf("woken worker retry did not run: worked=%v err=%v", worked, err)
		}
		done := f.publicationOperation(op.OperationId)
		if done.Status != api.OperationStatusSucceeded || !done.Error.IsNull() || nb.RouteCount() != 1 {
			t.Fatalf("completed op = %+v routes=%d, want succeeded with 1 route", done, nb.RouteCount())
		}
		if err := f.r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id = $1`, enrolled.NodeId).Scan(&pubObserved); err != nil || pubObserved != "active" {
			t.Fatalf("publication_observed = %q (%v), want active", pubObserved, err)
		}
	})

	t.Run("b_member_forwarding_unconfirmed_on_data_path_probe_waits_and_completes", func(t *testing.T) {
		f, enrolled, cert, _, _ := publicationFixture(t)
		nb := exits.NewFakeNetBirdAdapter()
		op := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)
		// Acknowledge the generation with member_forwarding_enabled=false so agent_policy passes
		// and data_path_probe is gated with forwarding_disabled.
		f.acknowledgePublication(enrolled, cert, false)
		d := f.publicationDispatcher(nb, 3)

		f.publicationDispatch(d, op.OperationId)
		waiting := f.publicationOperation(op.OperationId)
		if waiting.Status != api.OperationStatusRunning || !waiting.Error.IsNull() {
			t.Fatalf("attempt 1 op = (status=%s, error=%+v), want running with null error", waiting.Status, waiting.Error)
		}
		var policyStatus, probeStatus, errCode, errMsg, reason, pubObserved string
		if err := f.r.Pool.QueryRow(t.Context(), `
			SELECT o.steps->'agent_policy'->>'status',
			       o.steps->'data_path_probe'->>'status',
			       o.steps->'data_path_probe'->>'error_code',
			       o.steps->'data_path_probe'->>'error_message',
			       o.steps->'data_path_probe'->>'reason',
			       e.publication_observed
			FROM operations o
			JOIN exits e ON e.node_id = o.resource_id
			WHERE o.id = $1`, op.OperationId).Scan(&policyStatus, &probeStatus, &errCode, &errMsg, &reason, &pubObserved); err != nil {
			t.Fatal(err)
		}
		if policyStatus != "succeeded" || probeStatus != "pending" || errCode != string(apierr.NodeNotServiceable) || errMsg != "member_forwarding_unconfirmed" || reason != "forwarding_disabled" || pubObserved != "absent" {
			t.Fatalf("attempt 1 probe gate = (policy=%s, probe=%s, code=%s, msg=%s, reason=%s, observed=%s)",
				policyStatus, probeStatus, errCode, errMsg, reason, pubObserved)
		}

		// Next observation with armed forwarding satisfies data_path_probe and wakes the worker retry.
		f.acknowledgePublication(enrolled, cert, true)
		f.sql(`UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type<>$1`, desiredstate.EventNodePublicationChanged)
		if worked, err := d.Once(t.Context()); err != nil || !worked {
			t.Fatalf("woken worker retry did not run: worked=%v err=%v", worked, err)
		}
		done := f.publicationOperation(op.OperationId)
		if done.Status != api.OperationStatusSucceeded || !done.Error.IsNull() || nb.RouteCount() != 1 {
			t.Fatalf("completed op = %+v routes=%d, want succeeded with 1 route", done, nb.RouteCount())
		}
		if err := f.r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id = $1`, enrolled.NodeId).Scan(&pubObserved); err != nil || pubObserved != "active" {
			t.Fatalf("publication_observed = %q (%v), want active", pubObserved, err)
		}
	})

	t.Run("c_exhaustion_marks_step_and_operation_failed_and_sets_error", func(t *testing.T) {
		f, enrolled, _, _, _ := publicationFixture(t)
		nb := exits.NewFakeNetBirdAdapter()
		op := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)
		d := f.publicationDispatcher(nb, 2)

		// Attempt 1 of 2: waiting.
		f.publicationDispatch(d, op.OperationId)
		if got := f.publicationOperation(op.OperationId); got.Status != api.OperationStatusRunning || !got.Error.IsNull() {
			t.Fatalf("attempt 1 op = %+v, want running with null error", got)
		}
		// Attempt 2 of 2: exhausted -> failed step, failed operation, publication_observed = 'error'.
		f.publicationDispatch(d, op.OperationId)
		exhausted := f.publicationOperation(op.OperationId)
		if exhausted.Status != api.OperationStatusFailed || exhausted.Error.IsNull() {
			t.Fatalf("attempt 2 exhausted op = %+v, want failed with error", exhausted)
		}
		opErr, err := exhausted.Error.Get()
		if err != nil || opErr.Code != api.ErrorCodeNODENOTSERVICEABLE {
			t.Fatalf("exhausted op.Error = (%+v, %v), want NODE_NOT_SERVICEABLE", opErr, err)
		}
		for _, st := range exhausted.Steps {
			if st.Name == nodes.StepAgentPolicy {
				code, cErr := st.ErrorCode.Get()
				msg, mErr := st.Message.Get()
				if st.Status != api.OperationStepStatusFailed || cErr != nil || code != api.ErrorCodeNODENOTSERVICEABLE || mErr != nil || msg != "member_forwarding_unconfirmed" {
					t.Fatalf("exhausted agent_policy step = %+v, want failed with NODE_NOT_SERVICEABLE / member_forwarding_unconfirmed", st)
				}
			}
		}
		var stepStatus, reason, pubObserved string
		if err := f.r.Pool.QueryRow(t.Context(), `
			SELECT o.steps->'agent_policy'->>'status',
			       o.steps->'agent_policy'->>'reason',
			       e.publication_observed
			FROM operations o
			JOIN exits e ON e.node_id = o.resource_id
			WHERE o.id = $1`, op.OperationId).Scan(&stepStatus, &reason, &pubObserved); err != nil {
			t.Fatal(err)
		}
		if stepStatus != "failed" || reason != "stale_generation" || pubObserved != "error" {
			t.Fatalf("exhausted state = (step=%s, reason=%s, observed=%s), want (failed, stale_generation, error)", stepStatus, reason, pubObserved)
		}
	})

	t.Run("d_adapter_failure_fails_step_and_sets_error_immediately", func(t *testing.T) {
		f, enrolled, cert, _, _ := publicationFixture(t)
		nb := exits.NewFakeNetBirdAdapter()
		nb.SetPublishError(errors.New("TEST ONLY adapter 503"))
		op := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)
		f.acknowledgePublication(enrolled, cert, true)
		d := f.publicationDispatcher(nb, 3)

		// Even on attempt 1 of 3, an adapter failure marks netbird_route_readback failed and sets publication_observed = 'error'.
		f.publicationDispatch(d, op.OperationId)
		after := f.publicationOperation(op.OperationId)
		assertPublicationFailure(t, after, api.OperationStatusRunning)
		var pubObserved string
		if err := f.r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE node_id = $1`, enrolled.NodeId).Scan(&pubObserved); err != nil || pubObserved != "error" {
			t.Fatalf("adapter failure publication_observed = %q (%v), want error", pubObserved, err)
		}
	})

	t.Run("e_r11_empty_exception_withdraws_while_pending_and_keeps_withdrawn_on_exhaustion", func(t *testing.T) {
		r, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
		f := setupRuntime(t, r)
		cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
		decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
		group := f.group("d074-r11")
		path := "/api/v1/admin/nodes/" + report.NodeId.String()
		detail := f.call("GET", path, nil, nil, nil)
		op := decode[api.Operation](t, f.call("PUT", path+"/publication", api.PublicationRequest{
			PublicationIntent: api.PublicationIntentPublished,
			Reason:            "TEST ONLY r11 empty publish",
			GroupIds:          &[]uuid.UUID{group.GroupId},
		}, nil, map[string]string{"If-Match": detail.Header().Get("ETag"), "Idempotency-Key": uuid.NewString()}), 202)

		reader := desiredstate.NewFakeEntitlementReader()
		compiler := desiredstate.NewCompiler(r.Pool, publicationSigner{r.Provider}, desiredstate.WithEntitlementReader(reader), desiredstate.WithPolicyReader(desiredstate.NewFakePolicyReader(desiredstate.PolicySnapshot{})))
		compiled, err := compiler.Compile(t.Context(), report.NodeId)
		if err != nil {
			t.Fatal(err)
		}
		report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
		report.AppliedConfigGeneration.Set(compiled.Envelope.Signed.ConfigGeneration)
		report.MemberForwardingEnabled, report.DataHealth = true, api.DataHealthHealthy
		for i := range report.Checks {
			report.Checks[i].ObservedAt = report.ObservedAt
			if report.Checks[i].Name == api.NodeCheckNameTunnelProbe {
				report.Checks[i].Status = api.HealthCheckStatusPassed
			}
		}
		decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
		nb := exits.NewFakeNetBirdAdapter()
		d := f.publicationDispatcher(nb, 2)
		f.publicationDispatch(d, op.OperationId)

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
		acked := report
		acked.ReportId, acked.ObservedAt = uuid.New(), time.Now().UTC()
		acked.AppliedConfigGeneration.Set(bound.Envelope.Signed.ConfigGeneration)
		decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", acked, cert, nil), 202)

		var gate uuid.UUID
		if err := r.Pool.QueryRow(t.Context(), `SELECT id FROM operations WHERE resource_id=$1 AND principal=$2 ORDER BY created_at DESC,id DESC LIMIT 1`, report.NodeId, nodes.PublicationGatePrincipal).Scan(&gate); err != nil {
			t.Fatal(err)
		}
		var exitID uuid.UUID
		if err := r.Pool.QueryRow(t.Context(), `SELECT id FROM exits WHERE node_id=$1`, report.NodeId).Scan(&exitID); err != nil {
			t.Fatal(err)
		}

		// Attempt 1 of 2: withdraws route and sets publication_observed='withdrawn', while step stays pending and operation stays running.
		d = f.publicationDispatcher(nb, 2)
		f.publicationDispatch(d, gate)
		waiting := f.publicationOperation(gate)
		if waiting.Status != api.OperationStatusRunning || !waiting.Error.IsNull() {
			t.Fatalf("r11 gate attempt 1 op = %+v, want running with null error", waiting)
		}
		remote, err := nb.ReadExitState(t.Context(), r.Config.NetBirdDeployment, exitID, "")
		if err != nil || remote.Enabled {
			t.Fatalf("r11 gate attempt 1 route still enabled: %+v (%v)", remote, err)
		}
		var probeStatus, reason, observed string
		if err := r.Pool.QueryRow(t.Context(), `
			SELECT o.steps->'data_path_probe'->>'status',
			       o.steps->'data_path_probe'->>'reason',
			       e.publication_observed
			FROM operations o
			JOIN exits e ON e.node_id = o.resource_id
			WHERE o.id = $1`, gate).Scan(&probeStatus, &reason, &observed); err != nil {
			t.Fatal(err)
		}
		if probeStatus != "pending" || reason != "serving_checks_unconfirmed" || observed != "withdrawn" {
			t.Fatalf("r11 gate attempt 1 = (probe=%s, reason=%s, observed=%s), want (pending, serving_checks_unconfirmed, withdrawn)", probeStatus, reason, observed)
		}

		// Attempt 2 of 2 (exhaustion): marks step and operation failed while keeping publication_observed='withdrawn'.
		f.publicationDispatch(d, gate)
		exhausted := f.publicationOperation(gate)
		if exhausted.Status != api.OperationStatusFailed || exhausted.Error.IsNull() {
			t.Fatalf("r11 gate attempt 2 exhausted op = %+v, want failed", exhausted)
		}
		if err := r.Pool.QueryRow(t.Context(), `
			SELECT o.steps->'data_path_probe'->>'status',
			       e.publication_observed
			FROM operations o
			JOIN exits e ON e.node_id = o.resource_id
			WHERE o.id = $1`, gate).Scan(&probeStatus, &observed); err != nil {
			t.Fatal(err)
		}
		if probeStatus != "failed" || observed != "withdrawn" {
			t.Fatalf("r11 gate attempt 2 = (probe=%s, observed=%s), want (failed, withdrawn)", probeStatus, observed)
		}
	})
}

func publicationGateCount(t *testing.T, f *fixture, node uuid.UUID) int {
	t.Helper()
	var count int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM operations WHERE resource_id=$1 AND principal=$2`, node, nodes.PublicationGatePrincipal).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
