//go:build integration

package httpwire_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/apps/worker"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/releases"
	"github.com/TED-H-com/Freedom-VPN/internal/releases/releasetest"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type testToken struct{}

func (testToken) Token(context.Context) (string, error) { return "TEST ONLY adapter token", nil }
func TestAT06_WorkerBootstrapReadbackRetryAndFence(t *testing.T) {
	f := setup(t)
	c := f.create(nil)
	_, enrolled, cert := f.enroll(c)
	ctx := context.Background()
	var mu sync.Mutex
	var key map[string]any
	groups := map[string]map[string]any{}
	creates, readbacks := 0, 0
	peerExists := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/routes", "GET /api/policies", "GET /api/peers", "GET /api/dns/nameservers":
			io.WriteString(w, "[]")
		case "GET /api/peers/test-peer":
			if !peerExists {
				w.WriteHeader(404)
				io.WriteString(w, `{"message":"not found"}`)
			} else {
				io.WriteString(w, `{"id":"test-peer","ip":"100.64.1.2","user_id":"","ephemeral":false}`)
			}
		case "DELETE /api/peers/test-peer":
			peerExists = false
			io.WriteString(w, "{}")
		case "PUT /api/setup-keys/test-key":
			var changes map[string]any
			json.NewDecoder(r.Body).Decode(&changes)
			for k, v := range changes {
				key[k] = v
			}
			json.NewEncoder(w).Encode(key)
		case "GET /api/groups":
			list := []any{}
			for _, group := range groups {
				list = append(list, group)
			}
			json.NewEncoder(w).Encode(list)
		case "POST /api/groups":
			var group map[string]any
			json.NewDecoder(r.Body).Decode(&group)
			id := fmt.Sprintf("test-group-%d", len(groups)+1)
			group["id"] = id
			groups[id] = group
			json.NewEncoder(w).Encode(group)
		case "GET /api/groups/test-group-1", "GET /api/groups/test-group-2":
			group := groups[strings.TrimPrefix(r.URL.Path, "/api/groups/")]
			if group == nil {
				w.WriteHeader(404)
				return
			}
			json.NewEncoder(w).Encode(group)
		case "DELETE /api/groups/test-group-2":
			delete(groups, "test-group-2")
			io.WriteString(w, "{}")
		case "GET /api/setup-keys":
			if key == nil {
				io.WriteString(w, "[]")
			} else {
				json.NewEncoder(w).Encode([]any{key})
			}
		case "POST /api/setup-keys":
			creates++
			json.NewDecoder(r.Body).Decode(&key)
			key["id"] = "test-key"
			key["key"] = "TEST ONLY one-time setup"
			key["expires"] = time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
			json.NewEncoder(w).Encode(key)
			delete(key, "key")
		case "GET /api/setup-keys/test-key":
			readbacks++
			json.NewEncoder(w).Encode(key)
		default:
			t.Errorf("unexpected adapter call %s %s", r.Method, r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	defer server.Close()
	nb, e := netbird.New(netbird.Config{BaseURL: server.URL + "/api", DeploymentID: "TEST ONLY deployment", Token: testToken{}, Bindings: nodes.NetBirdBindings{Pool: f.r.Pool}, AllowInsecureHTTP: true})
	if e != nil {
		t.Fatal(e)
	}
	f.r.Config.NetBirdDeployment = "TEST ONLY deployment"
	f.r.Config.BootstrapSeconds = 60
	f.r.Config.ReconcileSeconds = 60
	f.r.Config.DesiredStateSeconds = 600
	f.r.Config.DNSResolverIPv4 = []string{"100.64.0.53"}
	if _, e = f.r.Provider.(*signing.FileProvider).Rotate(ctx, signing.DesiredState, 48*time.Hour, time.Hour); e != nil {
		t.Fatal(e)
	}
	compiled, e := f.r.DesiredStateCompiler().Compile(ctx, enrolled.NodeId)
	if e != nil {
		t.Fatal(e)
	}
	events, e := db.ClaimOutbox(ctx, f.r.Pool, "TEST ONLY worker", 1, time.Minute)
	if e != nil || len(events) != 1 {
		t.Fatal(e, len(events))
	}
	event := events[0]
	jobs := worker.NodeJobs(f.r, nb, nil, 3)
	for range 2 {
		receipt, e := jobs[event.EventType](ctx, event)
		if e != nil || !strings.HasPrefix(receipt, "bootstrap:") {
			t.Fatal(receipt, e)
		}
	}
	if creates != 1 || readbacks < 1 {
		t.Fatal("external creation/read-back", creates, readbacks)
	}
	var n int
	if e = f.r.Pool.QueryRow(ctx, `SELECT count(*) FROM node_bootstrap_secrets`).Scan(&n); e != nil || n != 1 {
		t.Fatal(e, n)
	}
	if _, e = f.r.Pool.Exec(ctx, `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, event.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = jobs[event.EventType](ctx, event); !errors.Is(e, db.ErrLeaseLost) {
		t.Fatal("stale worker committed", e)
	}
	// A successor can finish the same provisioning intent without another key.
	events, e = db.ClaimOutbox(ctx, f.r.Pool, "TEST ONLY successor", 1, time.Minute)
	if e != nil || len(events) != 1 {
		t.Fatal(e, len(events))
	}
	complete := func(event db.ClaimedEvent) {
		t.Helper()
		receipt, e := jobs[event.EventType](ctx, event)
		if e != nil {
			t.Fatal(event.EventType, e)
		}
		if _, e = f.r.Pool.Exec(ctx, `UPDATE outbox_events SET delivery_receipt=$2 WHERE id=$1`, event.ID, receipt); e != nil {
			t.Fatal(e)
		}
		if e = db.CompleteOutbox(ctx, f.r.Pool, event); e != nil {
			t.Fatal(e)
		}
	}
	complete(events[0])
	// Discovery has its own dispatcher tests; this case exercises the existing
	// reported-peer path and subsequent durable revocation cleanup.
	if _, err := f.r.Pool.Exec(ctx, `UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE event_type='node.peer.discover'`); err != nil {
		t.Fatal(err)
	}
	// The observation transaction queues read-back; reporting ready cannot bypass it.
	f.r.Nodes = nodes.NewService(f.r.Pool, nodes.WithVersionObserver(func(ctx context.Context, tx pgx.Tx, report api.ObservationReport) error {
		raw, e := canonjson.Marshal(map[string]any{"report_id": report.ReportId})
		if e != nil {
			return e
		}
		return db.EnqueueOutbox(ctx, tx, db.OutboxEvent{AggregateType: "node-incarnation", AggregateID: report.NodeIncarnationId, AggregateVersion: "1", EventType: "node.peer.observe", SchemaVersion: 1, Payload: raw})
	}))
	raw, e := os.ReadFile("../../../contracts/examples/valid/ObservationReport.json")
	if e != nil {
		t.Fatal(e)
	}
	var report api.ObservationReport
	if e = json.Unmarshal(raw, &report); e != nil {
		t.Fatal(e)
	}
	report.NodeId = enrolled.NodeId
	report.NodeIncarnationId = enrolled.NodeIncarnationId
	report.ReportId = uuid.New()
	report.ObservedAt = time.Now().UTC().Truncate(time.Microsecond)
	report.AppliedConfigGeneration.Set(compiled.Envelope.Signed.ConfigGeneration)
	report.CurrentReleaseId.SetNull()
	report.LastGoodReleaseId.SetNull()
	report.Lease.LeaseId.SetNull()
	report.Netbird.PeerId.Set("test-peer")
	check := report.Checks[0]
	check.Name = api.NodeCheckNameMeterPath
	report.Checks = append(report.Checks, check)
	decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), 202)
	var state string
	if e = f.r.Pool.QueryRow(ctx, `SELECT provisioning_state FROM nodes WHERE id=$1`, enrolled.NodeId).Scan(&state); e != nil || state != "applying" {
		t.Fatal(state, e)
	}
	events, e = db.ClaimOutbox(ctx, f.r.Pool, "TEST ONLY peer worker", 1, time.Minute)
	if e != nil || len(events) != 1 || events[0].EventType != "node.peer.observe" {
		t.Fatal(events, e)
	}
	complete(events[0])
	if e = f.r.Pool.QueryRow(ctx, `SELECT provisioning_state FROM nodes WHERE id=$1`, enrolled.NodeId).Scan(&state); e != nil || state != "ready" {
		t.Fatal(state, e)
	}
	// Revoke removes the known key and peer only after remote absence/read-back.
	path := "/api/v1/admin/nodes/" + enrolled.NodeId.String()
	detail := f.call("GET", path, nil, nil, nil)
	w := f.call("POST", path+"/revoke", map[string]string{"reason": "TEST ONLY worker withdrawal"}, nil, map[string]string{"Idempotency-Key": uuid.NewString(), "If-Match": detail.Header().Get("ETag")})
	if w.Code != 202 {
		t.Fatal(w.Code, w.Body.String())
	}
	events, e = db.ClaimOutbox(ctx, f.r.Pool, "TEST ONLY withdrawal worker", 2, time.Minute)
	if e != nil || len(events) != 1 || events[0].EventType != "node.publication_changed" {
		t.Fatal(events, e)
	}
	for _, event := range events {
		complete(event)
	}
	op := decode[api.Operation](t, w, 202)
	observed := f.publicationOperation(op.OperationId)
	if observed.Status != api.OperationStatusPartiallySucceeded {
		t.Fatal("revoked agent must remain unconfirmed", observed)
	}
	for _, step := range observed.Steps {
		if step.Name == nodes.StepNetBirdIdentityCleanup || step.Name == nodes.StepNetBirdRouteReadback {
			if step.Status != api.OperationStepStatusSucceeded || step.ObservedAt.IsNull() {
				t.Fatal("missing identity/route read-back", step)
			}
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if creates != 1 || key["revoked"] != true || peerExists {
		t.Fatal("unconfirmed external revocation", creates, key["revoked"], peerExists)
	}
}

type invalidArtifact struct{}

func (invalidArtifact) Open(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("TEST ONLY wrong artifact")), nil
}
func TestUPD02_WorkerReleaseVerificationFence(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	rf := releasetest.New(t)
	f.r.Releases = &releases.Service{Pool: f.r.Pool, Validator: rf.Validator, VerifyArtifacts: true}
	registration, e := f.r.Releases.Register(ctx, releasetest.Command(), rf.Request(t, rf.Manifest))
	if e != nil {
		t.Fatal(e)
	}
	events, e := db.ClaimOutbox(ctx, f.r.Pool, "TEST ONLY release worker", 1, time.Minute)
	if e != nil || len(events) != 1 {
		t.Fatal(e)
	}
	event := events[0]
	if _, e = f.r.Pool.Exec(ctx, `UPDATE outbox_events SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1`, event.ID); e != nil {
		t.Fatal(e)
	}
	jobs := worker.NodeJobs(f.r, nil, invalidArtifact{}, 3)
	if _, e = jobs[event.EventType](ctx, event); !errors.Is(e, db.ErrLeaseLost) {
		t.Fatal("unfenced verification", e)
	}
	var state string
	if e = f.r.Pool.QueryRow(ctx, `SELECT state FROM release_verifications WHERE release_id=$1`, registration.ReleaseID).Scan(&state); e != nil || state != "pending" {
		t.Fatal(state, e)
	}
	events, e = db.ClaimOutbox(ctx, f.r.Pool, "TEST ONLY current worker", 1, time.Minute)
	if e != nil || len(events) != 1 {
		t.Fatal(e)
	}
	if _, e = jobs[event.EventType](ctx, events[0]); e != nil {
		t.Fatal(e)
	}
	if e = f.r.Pool.QueryRow(ctx, `SELECT steps->'verify'->>'status' FROM operations WHERE id=$1`, registration.OperationID).Scan(&state); e != nil || state != "failed" {
		t.Fatal("failed verification reported success", state, e)
	}
}
