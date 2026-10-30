//go:build integration

package httpwire_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/desiredstate"
	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func (f *fixture) revocationCommand(node uuid.UUID, retire bool) api.Operation {
	f.t.Helper()
	path := "/api/v1/admin/nodes/" + node.String()
	detail := f.call("GET", path, nil, nil, nil)
	decode[api.NodeDetail](f.t, detail, 200)
	method, body := "POST", any(map[string]string{"reason": "TEST ONLY withdrawal"})
	if retire {
		method, body = "DELETE", nil
	} else {
		path += "/revoke"
	}
	headers := map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag")}
	op := decode[api.Operation](f.t, f.call(method, path, body, nil, headers), 202)
	replay := decode[api.Operation](f.t, f.call(method, path, body, nil, headers), 202)
	if !reflect.DeepEqual(op, replay) || len(op.Steps) != 4 {
		f.t.Fatal("revocation receipt/steps", op, replay)
	}
	var count int
	if err := f.r.Pool.QueryRow(f.t.Context(), `SELECT count(*) FROM outbox_events WHERE payload->>'operation_id'=$1`, op.OperationId.String()).Scan(&count); err != nil || count != 1 {
		f.t.Fatal("withdrawal must enqueue once", count, err)
	}
	return op
}

func (f *fixture) identityWorker(url string, adapter exits.NetBirdAdapter) (*worker.Dispatcher, worker.Registry) {
	f.t.Helper()
	// A node revoked before peer read-back has no deployment binding yet. The
	// withdrawal bridge must use this configured deployment, not the sentinel.
	f.r.Config.NetBirdDeployment = "TEST ONLY revocation deployment"
	client, err := netbird.New(netbird.Config{BaseURL: url + "/api", DeploymentID: f.r.Config.NetBirdDeployment, Token: testToken{}, Bindings: nodes.NetBirdBindings{Pool: f.r.Pool}, AllowInsecureHTTP: true})
	if err != nil {
		f.t.Fatal(err)
	}
	f.sql(`UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type<>$1`, desiredstate.EventNodePublicationChanged)
	jobs := worker.NodeJobs(f.r, client, nil, 2, adapter)
	d, err := worker.New(f.r.Pool, jobs, worker.Config{Concurrency: 1, Attempts: 2, Lease: time.Minute, JobTimeout: 30 * time.Second, Poll: time.Millisecond, Backoff: time.Millisecond, MaxBackoff: time.Millisecond}, nil, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	return d, jobs
}

func assertIdentityStep(t *testing.T, op api.Operation, want api.OperationStepStatus) {
	t.Helper()
	for _, step := range op.Steps {
		if step.Name != nodes.StepNetBirdIdentityCleanup {
			continue
		}
		if !step.Required || step.Status != want || step.ObservedAt.IsNull() != (want != api.OperationStepStatusSucceeded) {
			t.Fatal("identity evidence/status", step)
		}
		if want == api.OperationStepStatusFailed {
			code, err := step.ErrorCode.Get()
			if err != nil || code != api.ErrorCodeDEPENDENCYUNAVAILABLE {
				t.Fatal("identity failure lost", step)
			}
		}
		return
	}
	t.Fatal("required identity step absent")
}

func TestAT06_AT07_RevocationWorkerWithdrawalRetryAndDeadLetter(t *testing.T) {
	for _, retire := range []bool{false, true} {
		for _, recoverAdapter := range []bool{false, true} {
			name := map[bool]string{false: "revoke", true: "retire"}[retire] + map[bool]string{false: "/exhausted", true: "/recovered"}[recoverAdapter]
			t.Run(name, func(t *testing.T) {
				f := setup(t)
				// D-050: enrollment creates the node/exit/incarnation; no test
				// writes a serving, entitlement or observation projection.
				_, enrolled, _ := f.enroll(f.create(nil))
				op := f.revocationCommand(enrolled.NodeId, retire)
				wantKind := api.OperationKindNodeRevoke
				if retire {
					wantKind = api.OperationKindNodeRetire
				}
				if op.Kind != wantKind {
					t.Fatal(op.Kind)
				}
				var unavailable atomic.Bool
				var reads atomic.Int32
				unavailable.Store(true)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "GET" || r.URL.Path != "/api/setup-keys" {
						t.Errorf("unexpected identity call %s %s", r.Method, r.URL.Path)
						w.WriteHeader(500)
						return
					}
					reads.Add(1)
					if unavailable.Load() {
						w.WriteHeader(503)
						return
					}
					io.WriteString(w, "[]") // Confirm no key was provisioned.
				}))
				defer server.Close()
				nb := exits.NewFakeNetBirdAdapter() // TEST ONLY remote route state.
				var exitID uuid.UUID
				if err := f.r.Pool.QueryRow(t.Context(), `SELECT id FROM exits WHERE node_id=$1`, enrolled.NodeId).Scan(&exitID); err != nil {
					t.Fatal(err)
				}
				nb.InjectRouteState(exitID, exits.ObservedExitState{Exists: true, Enabled: true, ExternalID: "TEST ONLY route"})
				d, _ := f.identityWorker(server.URL, nb)
				f.publicationDispatch(d, op.OperationId)
				failed := f.publicationOperation(op.OperationId)
				if failed.Status != api.OperationStatusRunning {
					t.Fatal("retry lost", failed)
				}
				assertIdentityStep(t, failed, api.OperationStepStatusFailed)
				unavailable.Store(!recoverAdapter)
				f.publicationDispatch(d, op.OperationId)
				final := f.publicationOperation(op.OperationId)
				var attempt int
				var delivered, dead bool
				if err := f.r.Pool.QueryRow(t.Context(), `SELECT attempt,delivered_at IS NOT NULL,dead_lettered_at IS NOT NULL FROM outbox_events WHERE payload->>'operation_id'=$1`, op.OperationId.String()).Scan(&attempt, &delivered, &dead); err != nil || attempt != 2 || delivered != recoverAdapter || dead == recoverAdapter {
					t.Fatal("retry/dead-letter outcome", attempt, delivered, dead, err)
				}
				if !recoverAdapter {
					if final.Status != api.OperationStatusFailed {
						t.Fatal("exhausted operation claims progress", final)
					}
					assertIdentityStep(t, final, api.OperationStepStatusFailed)
					var audits int
					if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE operation_id=$1 AND action='node.identity_withdrawn'`, op.OperationId).Scan(&audits); err != nil || audits != 0 {
						t.Fatal("invented identity evidence", audits, err)
					}
					return
				}
				if final.Status != api.OperationStatusPartiallySucceeded {
					t.Fatal("offline agent must remain unconfirmed", final)
				}
				assertIdentityStep(t, final, api.OperationStepStatusSucceeded)
				for _, step := range final.Steps {
					if step.Name == nodes.StepNetBirdRouteReadback && (step.Status != api.OperationStepStatusSucceeded || step.ObservedAt.IsNull()) {
						t.Fatal("route not confirmed", step)
					}
					if step.Name == nodes.StepAgentPolicy && (step.Status != api.OperationStepStatusFailed || !step.ObservedAt.IsNull()) {
						t.Fatal("invented offline policy acknowledgement", step)
					}
				}
				var observed, receipt string
				var evidence bool
				if err := f.r.Pool.QueryRow(t.Context(), `SELECT e.publication_observed,o.delivery_receipt,EXISTS(SELECT 1 FROM audit_events a WHERE a.operation_id=p.id AND a.action='node.identity_withdrawn' AND a.result='succeeded' AND p.observed_refs->>'netbird_identity_cleanup'='audit_event:'||a.id::text) FROM operations p JOIN exits e ON e.node_id=p.resource_id JOIN outbox_events o ON o.payload->>'operation_id'=p.id::text WHERE p.id=$1`, op.OperationId).Scan(&observed, &receipt, &evidence); err != nil || observed != "withdrawn" || receipt != op.OperationId.String() || !evidence {
					t.Fatal("withdrawal evidence", observed, receipt, evidence, err)
				}
				state, err := nb.ReadExitState(t.Context(), f.r.Config.NetBirdDeployment, exitID, "")
				if err != nil || state.Enabled {
					t.Fatal("remote route remains enabled", state, err)
				}
				// Replay the delivered semantic event. Terminal results must not
				// repeat identity calls, audits or route writes.
				f.sql(`INSERT INTO outbox_events(aggregate_type,aggregate_id,aggregate_version,event_type,schema_version,payload) SELECT aggregate_type,aggregate_id,aggregate_version,event_type,schema_version,payload FROM outbox_events WHERE payload->>'operation_id'=$1`, op.OperationId.String())
				calls, keyReads := nb.WithdrawCalls(), reads.Load()
				f.publicationDispatch(d, op.OperationId)
				if got := f.publicationOperation(op.OperationId); !reflect.DeepEqual(final, got) || calls != nb.WithdrawCalls() || keyReads != reads.Load() {
					t.Fatal("terminal withdrawal replayed external work", got)
				}
			})
		}
	}
}

func TestAT06_RevocationIdentityEvidenceRequiresCurrentClaim(t *testing.T) {
	f := setup(t)
	_, enrolled, _ := f.enroll(f.create(nil))
	op := f.revocationCommand(enrolled.NodeId, false)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/setup-keys" {
			t.Errorf("unexpected identity call %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
			return
		}
		// Fault injection at the outbox boundary after the preflight fence.
		if _, err := f.r.Pool.Exec(r.Context(), `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE payload->>'operation_id'=$1`, op.OperationId.String()); err != nil {
			t.Error(err)
		}
		io.WriteString(w, "[]")
	}))
	defer server.Close()
	_, jobs := f.identityWorker(server.URL, exits.NewFakeNetBirdAdapter())
	events, err := db.ClaimOutbox(t.Context(), f.r.Pool, "TEST ONLY revocation fence", 1, time.Minute)
	if err != nil || len(events) != 1 {
		t.Fatal(events, err)
	}
	if ref, err := jobs[desiredstate.EventNodePublicationChanged](t.Context(), events[0]); err == nil || ref != "" {
		t.Fatal("stale cleanup acknowledged", ref, err)
	}
	assertIdentityStep(t, f.publicationOperation(op.OperationId), api.OperationStepStatusPending)
	var audits int
	if err := f.r.Pool.QueryRow(t.Context(), `SELECT count(*) FROM audit_events WHERE operation_id=$1 AND action='node.identity_withdrawn'`, op.OperationId).Scan(&audits); err != nil || audits != 0 {
		t.Fatal("stale cleanup evidence committed", audits, err)
	}
}
