//go:build integration

package httpwire_test

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func (f *fixture) publicationCommand(node uuid.UUID, intent api.PublicationIntent) api.Operation {
	f.t.Helper()
	path := "/api/v1/admin/nodes/" + node.String()
	detail := f.call("GET", path, nil, nil, nil)
	decode[api.NodeDetail](f.t, detail, 200)
	return decode[api.Operation](f.t, f.call("PUT", path+"/publication", api.PublicationRequest{PublicationIntent: intent, Reason: "TEST ONLY worker publication"}, nil, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag")}), 202)
}

func (f *fixture) acknowledgePublication(enrolled api.EnrollResponse, cert *x509.Certificate, forwarding bool) {
	f.t.Helper()
	state := decode[api.DesiredStateEnvelope](f.t, f.call("GET", "/api/v1/agent/desired-state", nil, cert, nil), 200)
	raw, err := os.ReadFile("../../../contracts/examples/valid/ObservationReport.json")
	if err != nil {
		f.t.Fatal(err)
	}
	var report api.ObservationReport
	if err = json.Unmarshal(raw, &report); err != nil {
		f.t.Fatal(err)
	}
	report.NodeId, report.NodeIncarnationId, report.ReportId = enrolled.NodeId, enrolled.NodeIncarnationId, uuid.New()
	report.ObservedAt = time.Now().UTC()
	report.AppliedConfigGeneration.Set(state.Signed.ConfigGeneration)
	report.CurrentReleaseId.SetNull()
	report.LastGoodReleaseId.SetNull()
	report.Lease.LeaseId.SetNull()
	report.Netbird.PeerId.SetNull()
	report.MemberForwardingEnabled = forwarding
	meter := report.Checks[0]
	meter.Name = api.NodeCheckNameMeterPath
	report.Checks = append(report.Checks, meter)
	for i := range report.Checks {
		report.Checks[i].Status = api.HealthCheckStatusPassed
		report.Checks[i].ObservedAt = report.ObservedAt
	}
	decode[api.ObservationAccepted](f.t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
}

func (f *fixture) publicationDispatcher(nb exits.NetBirdAdapter, attempts int) *worker.Dispatcher {
	f.t.Helper()
	// Keep unrelated provisioning/reconciliation fixtures pending for their own tests.
	f.sql(`UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type<>$1`, desiredstate.EventNodePublicationChanged)
	d, err := worker.New(f.r.Pool, worker.NodeJobs(f.r, nil, nil, nb), worker.Config{Concurrency: 1, Attempts: attempts, Lease: time.Minute, JobTimeout: 30 * time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Millisecond}, nil, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	return d
}

func (f *fixture) publicationDispatch(d *worker.Dispatcher, op uuid.UUID) {
	f.t.Helper()
	f.sql(`UPDATE outbox_events SET available_at=clock_timestamp() WHERE event_type=$1 AND payload->>'operation_id'=$2 AND delivered_at IS NULL AND dead_lettered_at IS NULL`, desiredstate.EventNodePublicationChanged, op.String())
	if worked, err := d.Once(f.t.Context()); err != nil || !worked {
		f.t.Fatal("publication not dispatched", worked, err)
	}
}

func (f *fixture) publicationOperation(id uuid.UUID) api.Operation {
	f.t.Helper()
	op, err := nodes.GetOperation(f.t.Context(), f.r.Pool, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return op
}

func TestAT08_HTTPPublicationWorkerRetryReadbackAndReplay(t *testing.T) {
	f, enrolled, cert, _, _ := publicationFixture(t)
	nb := exits.NewFakeNetBirdAdapter() // TEST ONLY; production uses the real Client bridge.
	op := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)
	f.acknowledgePublication(enrolled, cert, true)
	d := f.publicationDispatcher(nb, 3)
	// The route write succeeds but its independent read-back fails. No success
	// can be acknowledged; retry must reuse the same route.
	nb.SetReadbackError(errors.New("TEST ONLY read-back unavailable"))
	f.publicationDispatch(d, op.OperationId)
	failed := f.publicationOperation(op.OperationId)
	if failed.Status != api.OperationStatusRunning || nb.RouteCount() != 1 {
		t.Fatalf("failure not retryable: %+v routes=%d", failed, nb.RouteCount())
	}
	for _, step := range failed.Steps {
		if step.Name == nodes.StepNetBirdRouteReadback && (step.Status != api.OperationStepStatusFailed || !step.ObservedAt.IsNull()) {
			t.Fatal("invented read-back", step)
		}
	}
	nb.SetReadbackError(nil)
	f.publicationDispatch(d, op.OperationId)
	success := f.publicationOperation(op.OperationId)
	if success.Status != api.OperationStatusSucceeded || len(success.Steps) != 3 || nb.RouteCount() != 1 {
		t.Fatal("publication incomplete or duplicated", success, nb.RouteCount())
	}
	for _, step := range success.Steps {
		if !step.Required || step.Status != api.OperationStepStatusSucceeded || step.ObservedAt.IsNull() {
			t.Fatal("missing observed step", step)
		}
	}
	var observed, receipt string
	var evidence bool
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT e.publication_observed,o.delivery_receipt,p.observed_refs->>'agent_policy'<>'' AND p.observed_refs->>'netbird_route_readback'<>'' AND p.observed_refs->>'data_path_probe'<>'' FROM exits e JOIN operations p ON p.resource_id=e.node_id JOIN outbox_events o ON o.payload->>'operation_id'=p.id::text WHERE p.id=$1 AND o.delivered_at IS NOT NULL`, op.OperationId).Scan(&observed, &receipt, &evidence); err != nil || observed != "active" || receipt != op.OperationId.String() || !evidence {
		t.Fatal(observed, receipt, evidence, err)
	}
	// A new claim of the same semantic event after success is a no-op.
	f.sql(`INSERT INTO outbox_events(aggregate_type,aggregate_id,aggregate_version,event_type,schema_version,payload) SELECT aggregate_type,aggregate_id,aggregate_version,event_type,schema_version,payload FROM outbox_events WHERE payload->>'operation_id'=$1`, op.OperationId.String())
	calls := nb.PublishCalls()
	before := f.publicationOperation(op.OperationId)
	f.publicationDispatch(d, op.OperationId)
	after := f.publicationOperation(op.OperationId)
	if nb.PublishCalls() != calls || !before.UpdatedAt.Equal(after.UpdatedAt) {
		t.Fatal("terminal operation executed again")
	}
	var delivered int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM outbox_events WHERE payload->>'operation_id'=$1 AND delivered_at IS NOT NULL AND delivery_receipt=$1`, op.OperationId.String()).Scan(&delivered); err != nil || delivered != 2 {
		t.Fatal("replay receipt", delivered, err)
	}
}

func TestAT13_HTTPWithdrawalWorkerOnlineAndOffline(t *testing.T) {
	for _, intent := range []api.PublicationIntent{api.PublicationIntentUnpublished, api.PublicationIntentDisabled} {
		for _, offline := range []bool{false, true} {
			t.Run(string(intent)+map[bool]string{false: "/online", true: "/offline"}[offline], func(t *testing.T) {
				f, enrolled, cert, _, _ := servingFixture(t)
				nb := exits.NewFakeNetBirdAdapter()
				var exitID uuid.UUID
				if err := f.r.Pool.QueryRow(t.Context(), `SELECT id FROM exits WHERE node_id=$1`, enrolled.NodeId).Scan(&exitID); err != nil {
					t.Fatal(err)
				}
				nb.InjectRouteState(exitID, exits.ObservedExitState{Exists: true, Enabled: true, ExternalID: "TEST ONLY route"})
				op := f.publicationCommand(enrolled.NodeId, intent)
				if offline {
					f.sql(`UPDATE nodes SET control_status='offline',member_forwarding_observed=true,resource_version=resource_version+1 WHERE id=$1`, enrolled.NodeId)
				} else {
					f.acknowledgePublication(enrolled, cert, false)
				}
				d := f.publicationDispatcher(nb, 2)
				f.publicationDispatch(d, op.OperationId)
				got := f.publicationOperation(op.OperationId)
				want := api.OperationStatusSucceeded
				if offline {
					want = api.OperationStatusPartiallySucceeded
				}
				if got.Status != want {
					t.Fatal(got)
				}
				for _, step := range got.Steps {
					if step.Name == nodes.StepNetBirdRouteReadback && (step.Status != api.OperationStepStatusSucceeded || step.ObservedAt.IsNull()) {
						t.Fatal("withdrawal unconfirmed", step)
					}
					if offline && step.Name == nodes.StepAgentPolicy && (step.Status != api.OperationStepStatusFailed || !step.ObservedAt.IsNull()) {
						t.Fatal("offline node falsely confirmed", step)
					}
				}
				state, err := nb.ReadExitState(t.Context(), "TEST ONLY", exitID, "")
				if err != nil || state.Enabled {
					t.Fatal(state, err)
				}
				var observed string
				if err := f.r.Pool.QueryRow(t.Context(), `SELECT publication_observed FROM exits WHERE id=$1`, exitID).Scan(&observed); err != nil || observed != "withdrawn" {
					t.Fatal(observed, err)
				}
			})
		}
	}
}

func TestAT08_PublicationWorkerFailureRetriesAndDeadLetters(t *testing.T) {
	f, enrolled, cert, _, _ := publicationFixture(t)
	nb := exits.NewFakeNetBirdAdapter()
	nb.SetPublishError(errors.New("TEST ONLY unavailable"))
	op := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)
	f.acknowledgePublication(enrolled, cert, true)
	d := f.publicationDispatcher(nb, 2)
	for attempt := 1; attempt <= 2; attempt++ {
		f.publicationDispatch(d, op.OperationId)
		var count int
		var delivered, dead bool
		if err := f.r.Pool.QueryRow(t.Context(), `SELECT attempt,delivered_at IS NOT NULL,dead_lettered_at IS NOT NULL FROM outbox_events WHERE payload->>'operation_id'=$1`, op.OperationId.String()).Scan(&count, &delivered, &dead); err != nil || count != attempt || delivered || dead != (attempt == 2) {
			t.Fatal(count, delivered, dead, err)
		}
		got := f.publicationOperation(op.OperationId)
		if got.Status != api.OperationStatusRunning || got.Error.IsNull() || nb.RouteCount() != 0 {
			t.Fatal("failure lost or acknowledged", got)
		}
	}
	if worked, err := d.Once(t.Context()); worked || err != nil {
		t.Fatal("dead letter was reclaimed", worked, err)
	}
}
