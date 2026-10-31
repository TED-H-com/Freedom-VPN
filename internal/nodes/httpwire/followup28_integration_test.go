//go:build integration

package httpwire_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/enrollment"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/internal/operations"
	"github.com/TED-H-com/Freedom-VPN/internal/pki/pkitest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Validate every response in these scenarios, including rejected retired leaves.
func validateFollowup28Responses(t *testing.T, f *fixture) {
	t.Helper()
	spec, err := openapi3.NewLoader().LoadFromFile("../../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	spec.Servers = nil
	router, err := legacy.NewRouter(spec)
	if err != nil {
		t.Fatal(err)
	}
	wrap := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := httptest.NewRecorder()
			next.ServeHTTP(rec, r)
			request := r.Clone(r.Context())
			request.URL.Path = strings.TrimPrefix(request.URL.Path, "/api/v1")
			route, params, err := router.FindRoute(request)
			if err != nil {
				t.Fatal(err)
			}
			if err = openapi3filter.ValidateResponse(r.Context(), &openapi3filter.ResponseValidationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{Request: request, Route: route, PathParams: params},
				Status:                 rec.Code, Header: rec.Header(), Body: io.NopCloser(bytes.NewReader(rec.Body.Bytes())),
				Options: &openapi3filter.Options{IncludeResponseStatus: true},
			}); err != nil {
				t.Fatalf("%s %s response violates OpenAPI: %v", r.Method, r.URL.Path, err)
			}
			for k, values := range rec.Header() {
				w.Header()[k] = values
			}
			w.WriteHeader(rec.Code)
			_, _ = w.Write(rec.Body.Bytes())
		})
	}
	f.agent, f.public = wrap(f.agent), wrap(f.public)
}

func requireFreezeReceipt(t *testing.T, w *httptest.ResponseRecorder, frozen bool) api.ObservationAccepted {
	t.Helper()
	out := decode[api.ObservationAccepted](t, w, http.StatusAccepted)
	body := decode[map[string]json.RawMessage](t, w, http.StatusAccepted)
	flag, present := body["identity_frozen"]
	if present != frozen || (present && string(flag) != "true") {
		t.Fatalf("identity_frozen must be true or absent: frozen=%v body=%s", frozen, w.Body.String())
	}
	return out
}

func requireSameReceipt(t *testing.T, first, replay api.ObservationAccepted) {
	t.Helper()
	if first.ReportId != replay.ReportId || first.Accepted != replay.Accepted || !first.ReceivedAt.Equal(replay.ReceivedAt) {
		t.Fatalf("replay changed durable receipt: first=%+v replay=%+v", first, replay)
	}
}

func TestD088_AT07_AT29_HTTPObservationFreezeLiveReceipt(t *testing.T) {
	for _, triggerReplay := range []bool{false, true} {
		t.Run(map[bool]string{false: "new_report_detects_freeze", true: "report_replay_detects_freeze"}[triggerReplay], func(t *testing.T) {
			runtime, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
			f := setupRuntime(t, runtime)
			validateFollowup28Responses(t, f)
			cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
			headers := map[string]string{"Idempotency-Key": uuid.NewString()}
			post := func(r api.ObservationReport, remote string, key map[string]string, frozen bool) api.ObservationAccepted {
				t.Helper()
				return requireFreezeReceipt(t, f.callFrom("POST", "/api/v1/agent/observations", r, cert, remote, key), frozen)
			}
			first := post(report, reviewDomainX, headers, false)
			// Neither receipt replay nor ordinary X -> Y roaming announces a freeze.
			requireSameReceipt(t, first, post(report, reviewDomainX, headers, false))
			requireSameReceipt(t, first, post(report, reviewDomainX, nil, false))
			roaming := report
			roaming.ReportId = uuid.New()
			post(roaming, reviewDomainY, nil, false)
			trigger := report
			triggerKey := headers
			if !triggerReplay {
				trigger.ReportId = uuid.New()
				triggerKey = map[string]string{"Idempotency-Key": uuid.NewString()}
			}
			// A report-ID replay still observes endpoints. HTTP-key replay is read-only.
			key := triggerKey
			if triggerReplay {
				key = nil
			}
			detecting := post(trigger, reviewDomainX, key, true)
			if triggerReplay {
				requireSameReceipt(t, first, detecting)
			}
			// Rebuild listeners against durable state, resetting the fixture's
			// five-request observation burst before testing replay after restart.
			f = setupRuntime(t, runtime)
			validateFollowup28Responses(t, f)
			for _, replayKey := range []map[string]string{nil, triggerKey} {
				requireSameReceipt(t, detecting, post(trigger, reviewDomainX, replayKey, true))
			}
			// A receipt originally issued before freezing must acquire the live flag.
			for _, replayKey := range []map[string]string{nil, headers} {
				requireSameReceipt(t, first, post(report, reviewDomainX, replayKey, true))
			}
			f = setupRuntime(t, runtime)
			validateFollowup28Responses(t, f)
			for _, delta := range []time.Duration{-time.Minute, time.Minute} {
				later := report
				later.ReportId, later.ObservedAt = uuid.New(), report.ObservedAt.Add(delta)
				post(later, reviewDomainY, nil, true)
			}
			var count int
			if err := runtime.Pool.QueryRow(t.Context(), `SELECT count(*) FROM node_observations WHERE report_id=$1`, trigger.ReportId).Scan(&count); err != nil || count != 1 {
				t.Fatalf("replay inserted another observation: count=%d err=%v", count, err)
			}
			if !reviewFrozen(t, f, enrolled.Response.NodeIncarnationId) || reviewConflictCount(t, f, enrolled.Response.NodeIncarnationId) != 1 {
				t.Fatal("freeze/replay did not retain exactly one conflict")
			}
		})
	}
}

func TestD088_AT07_HTTPFrozenRenewalThenObservationAndRetiredIncarnation(t *testing.T) {
	runtime, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
	f := setupRuntime(t, runtime)
	validateFollowup28Responses(t, f)
	cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
	reviewRotateLease(t, f)
	for _, remote := range []string{reviewDomainX, reviewDomainY, reviewDomainX} {
		reviewRenew(t, f, enrolled.Response, cert, remote)
	}
	if !reviewFrozen(t, f, report.NodeIncarnationId) {
		t.Fatal("renewal did not freeze incarnation")
	}
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	requireFreezeReceipt(t, f.callFrom("POST", "/api/v1/agent/observations", report, cert, reviewDomainX, headers), true)
	path := "/api/v1/admin/nodes/" + report.NodeId.String()
	detail := f.call("GET", path, nil, nil, nil)
	decode[api.NodeDetail](t, detail, http.StatusOK)
	decode[api.OperationAccepted](t, f.call("POST", path+"/revoke", map[string]string{"reason": "TEST ONLY frozen incarnation replacement"}, nil,
		map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag")}), http.StatusAccepted)
	created, err := runtime.Enrollment.CreateEnrollment(t.Context(), enrollment.Command{Principal: f.p.Scope(), Key: uuid.New()}, api.EnrollmentCreateRequest{Delivery: api.EnrollmentDeliveryInstaller, NodeId: &report.NodeId})
	if err != nil {
		t.Fatal(err)
	}
	request, _ := pkitest.Request(t)
	replaced, err := runtime.Enrollment.Enroll(t.Context(), mustValue(t, created.Token), request)
	if err != nil {
		t.Fatal(err)
	}
	replacement := replaced.Response
	newCert := pkitest.Certificate(t, replacement.CertificatePem)
	for _, key := range []map[string]string{nil, headers} {
		w := f.callFrom("POST", "/api/v1/agent/observations", report, cert, reviewDomainX, key)
		env := decode[api.ErrorEnvelope](t, w, http.StatusForbidden)
		if env.Error.Code != api.ErrorCodeIDENTITYREVOKED || strings.Contains(w.Body.String(), "identity_frozen") {
			t.Fatalf("retired report changed behavior: %s", w.Body.String())
		}
	}
	report.NodeIncarnationId, report.ReportId = replacement.NodeIncarnationId, uuid.New()
	requireFreezeReceipt(t, f.callFrom("POST", "/api/v1/agent/observations", report, newCert, reviewDomainY, nil), false)
	if !reviewFrozen(t, f, enrolled.Response.NodeIncarnationId) || reviewFrozen(t, f, replacement.NodeIncarnationId) {
		t.Fatal("historical freeze leaked to replacement")
	}
}

func TestD093_ISO02_HTTPTCForeignFilterStoredEvidence(t *testing.T) {
	for _, code := range []api.ErrorCode{api.ErrorCodeHEALTHCHECKFAILED, api.ErrorCodeTCFOREIGNFILTER} {
		t.Run(string(code), func(t *testing.T) {
			spec, ok := apierr.Lookup(apierr.Code(code))
			if !ok || !spec.NodeLocal || spec.Retryable {
				t.Fatalf("expected registered non-retryable node-local error: %+v", spec)
			}
			runtime, enrolled, report := nodestest.BaselineObservation(t, "../../../contracts/examples/valid/ObservationReport.json")
			f := setupRuntime(t, runtime)
			validateFollowup28Responses(t, f)
			cert := pkitest.Certificate(t, enrolled.Response.CertificatePem)
			report.SafetyStatus, report.ProvisioningState = api.SafetyStatusFailed, api.ProvisioningStateFailed
			report.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonSafetyCheckFailed}
			report.Errors = []api.NodeError{{Code: code, Stage: "apply", Message: "TEST ONLY serving verification failed", ObservedAt: report.ObservedAt}}
			report.Checks[0].Status = api.HealthCheckStatusFailed
			report.Checks[0].ErrorCode.Set(code)
			requireFreezeReceipt(t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), false)
			var safety, checkCode string
			var forwarding bool
			var raw []byte
			if err := runtime.Pool.QueryRow(t.Context(), `SELECT safety_status,member_forwarding_enabled,checks->'internet_only'->>'error_code',report_payload FROM node_observations WHERE report_id=$1`, report.ReportId).Scan(&safety, &forwarding, &checkCode, &raw); err != nil {
				t.Fatal(err)
			}
			var stored api.ObservationReport
			if err := json.Unmarshal(raw, &stored); err != nil {
				t.Fatal(err)
			}
			if safety != "failed" || forwarding || checkCode != string(code) || !reflect.DeepEqual(stored.Errors, report.Errors) || !reflect.DeepEqual(stored.Checks, report.Checks) {
				t.Fatal("stored observation lost fail-closed classification or original error code")
			}
			detail := decode[api.NodeDetail](t, f.call("GET", "/api/v1/admin/nodes/"+report.NodeId.String(), nil, nil, nil), http.StatusOK)
			if detail.SafetyStatus != api.SafetyStatusFailed || mustValue(t, detail.MemberForwardingObserved) || !reflect.DeepEqual(detail.RecentErrors, report.Errors) || !reflect.DeepEqual(detail.Checks, report.Checks) {
				t.Fatal("NodeDetail changed local failure evidence")
			}

			// Exercise the existing operation failure writer/allowlist with the
			// stored report's code, then both node-domain and HTTP projections.
			var opID uuid.UUID
			if err := db.InTx(t.Context(), runtime.Pool, db.TxOptions{}, func(tx pgx.Tx) error {
				op, err := nodes.CreateOperation(t.Context(), tx, nodes.CreateOperationParams{Kind: nodes.OpKindUpdate, ResourceID: report.NodeId, Principal: f.p.Scope()})
				if err != nil {
					return err
				}
				opID = op.OperationId
				return operations.Transition(t.Context(), tx, opID, nodes.StepAgentPolicy, api.OperationStepStatusFailed, nil, &stored.Errors[0].Code, nil)
			}); err != nil {
				t.Fatal(err)
			}
			projected, err := runtime.Nodes.GetOperation(t.Context(), opID)
			if err != nil {
				t.Fatal(err)
			}
			wire := decode[api.Operation](t, f.call("GET", "/api/v1/admin/operations/"+opID.String(), nil, nil, nil), http.StatusOK)
			for _, op := range []api.Operation{projected, wire} {
				failure, err := op.Error.Get()
				if err != nil || failure.Code != code || op.Status != api.OperationStatusFailed || len(op.Steps) != 1 || mustValue(t, op.Steps[0].ErrorCode) != code {
					t.Fatalf("operation projection changed stored failure: %+v", op)
				}
			}
		})
	}
}
