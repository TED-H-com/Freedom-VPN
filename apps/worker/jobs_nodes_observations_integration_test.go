//go:build integration

package worker

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/netbird"
	nodelib "github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/nodestest"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type observationToken struct{}

func (observationToken) Token(context.Context) (string, error) { return "TEST ONLY token", nil }

func TestAT06_PeerObservationWorkerSeparatesReadinessAndBinding(t *testing.T) {
	for _, name := range []string{"absent_peer", "present_peer", "failed_safety", "behind_generation", "regressed_generation", "stale_ready", "failed_readback"} {
		t.Run(name, func(t *testing.T) {
			r, enrolled, report := nodestest.BaselineObservation(t, "../../contracts/examples/valid/ObservationReport.json")
			if name == "regressed_generation" {
				if _, err := r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
					t.Fatal(err)
				}
				report.ReportId, report.ObservedAt = uuid.New(), report.ObservedAt.Add(time.Second)
			}
			withPeer := name == "present_peer" || name == "failed_readback"
			if withPeer {
				report.Netbird.PeerId.Set("test-peer")
			}
			if name == "failed_safety" {
				report.SafetyStatus = api.SafetyStatusFailed
			}
			if name == "behind_generation" || name == "regressed_generation" {
				report.AppliedConfigGeneration.Set(0)
			}
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				calls.Add(1)
				if !withPeer || req.Method != "GET" || req.URL.Path != "/api/peers/test-peer" {
					t.Errorf("unexpected NetBird request %s %s", req.Method, req.URL.Path)
					w.WriteHeader(500)
					return
				}
				if name == "failed_readback" {
					w.WriteHeader(503)
					return
				}
				io.WriteString(w, `{"id":"test-peer","ip":"100.64.1.2","user_id":"","ephemeral":false}`)
			}))
			defer server.Close()
			nb, err := netbird.New(netbird.Config{BaseURL: server.URL + "/api", DeploymentID: r.Config.NetBirdDeployment, Token: observationToken{}, Bindings: nodelib.NetBirdBindings{Pool: r.Pool}, AllowInsecureHTTP: true})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = r.Nodes.SubmitObservation(t.Context(), report.NodeId, report.NodeIncarnationId, report); err != nil {
				t.Fatal(err)
			}
			if name == "stale_ready" {
				newer := report
				newer.ReportId, newer.ObservedAt = uuid.New(), report.ObservedAt.Add(time.Second)
				newer.ProvisioningState = api.ProvisioningStateApplying
				if _, err := r.Nodes.SubmitObservation(t.Context(), newer.NodeId, newer.NodeIncarnationId, newer); err != nil {
					t.Fatal(err)
				}
			}
			var eventID uuid.UUID
			if err := r.Pool.QueryRow(t.Context(), `SELECT id FROM outbox_events WHERE event_type='node.peer.observe' AND payload->>'report_id'=$1`, report.ReportId.String()).Scan(&eventID); err != nil {
				t.Fatal("runtime did not enqueue observation", err)
			}
			// Dispatch the production node jobs through explicit composition
			// (D-039), deferring unrelated enrollment/reconciliation work.
			if _, err = r.Pool.Exec(t.Context(), `UPDATE outbox_events SET available_at=clock_timestamp()+interval '1 day' WHERE id<>$1`, eventID); err != nil {
				t.Fatal(err)
			}
			cfg := config.Defaults()
			app, err := Open(t.Context(), Dependencies{Pool: r.Pool, Config: cfg}, Domain{Name: "nodes", Build: func(context.Context, Dependencies) (Built, error) {
				return Built{Jobs: NodeJobs(r, nb, nil, cfg.WorkerAttempts)}, nil
			}})
			if err != nil {
				t.Fatal(err)
			}
			if worked, err := app.Once(t.Context()); err != nil || !worked {
				t.Fatal("observation dispatch", worked, err)
			}
			var attempt int
			var delivered, dead bool
			var receipt *string
			if err := r.Pool.QueryRow(t.Context(), `SELECT attempt,delivered_at IS NOT NULL,dead_lettered_at IS NOT NULL,delivery_receipt FROM outbox_events WHERE id=$1`, eventID).Scan(&attempt, &delivered, &dead, &receipt); err != nil || attempt != 1 || dead || delivered != (name != "failed_readback") {
				t.Fatal("observation retry/receipt", attempt, delivered, dead, receipt, err)
			}
			if delivered && (receipt == nil || *receipt != "observation:"+report.ReportId.String()) {
				t.Fatal("observation evidence receipt", receipt)
			}
			detail, _, err := r.Nodes.GetNode(t.Context(), report.NodeId)
			wantReady := name == "absent_peer" || name == "present_peer"
			wantState := api.ProvisioningStateApplying
			if wantReady {
				wantState = api.ProvisioningStateReady
			}
			if err != nil || detail.ProvisioningState != wantState {
				t.Fatal("readiness", detail, err)
			}
			peer, peerErr := detail.NetbirdPeerId.Get()
			if name == "present_peer" {
				if peerErr != nil || peer != "test-peer" || calls.Load() == 0 {
					t.Fatal("reported peer not bound after read-back", peer, peerErr, calls.Load())
				}
			} else if !detail.NetbirdPeerId.IsNull() || (!withPeer && calls.Load() != 0) {
				t.Fatal("invented binding or unnecessary read-back", detail.NetbirdPeerId, calls.Load())
			}
			op, err := nodelib.GetOperation(t.Context(), r.Pool, enrolled.OperationID)
			if err != nil || op.Status == api.OperationStatusSucceeded {
				t.Fatal("unobserved provisioning steps completed", op, err)
			}
			for _, step := range op.Steps {
				switch step.Name {
				case "peer_readback":
					if (step.Status == api.OperationStepStatusSucceeded) != (name == "present_peer") || step.ObservedAt.IsNull() != (name != "present_peer") {
						t.Fatal("peer evidence", step)
					}
				case "node_policy":
					if (step.Status == api.OperationStepStatusSucceeded) != wantReady || step.ObservedAt.IsNull() == wantReady {
						t.Fatal("policy acknowledgement", step)
					}
				}
			}
		})
	}
}
