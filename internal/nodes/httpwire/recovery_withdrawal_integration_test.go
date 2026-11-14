//go:build integration

package httpwire_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/exits"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes"
	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestRecoveryModeT8_RevocationWithdrawal(t *testing.T) {
	for _, freshObservation := range []bool{true, false} {
		name := "without_post_freeze_observation"
		if freshObservation {
			name = "with_post_freeze_observation"
		}
		t.Run(name, func(t *testing.T) {
			f, enrolled, cert, _, _ := publicationFixture(t)
			adapter := exits.NewFakeNetBirdAdapter()
			publish := f.publicationCommand(enrolled.NodeId, api.PublicationIntentPublished)
			f.acknowledgePublication(enrolled, cert, true)
			publisher := f.publicationDispatcher(adapter, 3)
			f.publicationDispatch(publisher, publish.OperationId)
			if op := f.publicationOperation(publish.OperationId); op.Status != api.OperationStatusSucceeded || adapter.PublishCalls() != 1 {
				t.Fatalf("initial publication: status=%s calls=%d", op.Status, adapter.PublishCalls())
			}
			var reportRaw []byte
			if err := f.r.Pool.QueryRow(t.Context(), `SELECT report_payload FROM node_observations WHERE node_id=$1 ORDER BY received_at DESC,id DESC LIMIT 1`, enrolled.NodeId).Scan(&reportRaw); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || (r.URL.Path != "/api/setup-keys" && r.URL.Path != "/api/groups") {
					t.Errorf("unexpected identity request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				_, _ = io.WriteString(w, "[]")
			}))
			defer server.Close()
			dispatcher, _ := f.identityWorker(server.URL, adapter)
			_, key, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			ledger, err := usage.NewService(usage.Config{Pool: f.r.Pool, LedgerEpochSigner: usage.NewLedgerEpochSigner("TEST ONLY recovery", key)})
			if err != nil {
				t.Fatal(err)
			}
			actor := nodes.SystemActorUUID.String()
			if err := ledger.FreezeIngest(t.Context(), actor, "TEST ONLY restore"); err != nil {
				t.Fatal(err)
			}
			if _, err := ledger.BeginRecovery(t.Context(), usage.RecoveryRequest{Actor: actor, Reason: "TEST ONLY restore", RestoredTo: time.Now().UTC().Add(-time.Hour)}); err != nil {
				t.Fatal(err)
			}
			if freshObservation {
				var report api.ObservationReport
				if err := json.Unmarshal(reportRaw, &report); err != nil {
					t.Fatal(err)
				}
				report.ReportId, report.ObservedAt = uuid.New(), time.Now().UTC()
				decode[api.ObservationAccepted](t, f.call("POST", "/api/v1/agent/observations", report, cert, nil), http.StatusAccepted)
				var fresh bool
				if err := f.r.Pool.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM node_observations o JOIN operations p ON p.resource_id=o.node_id AND p.principal=$2 WHERE o.report_id=$1 AND o.received_at>p.created_at)`, report.ReportId, usage.RestoreObservationPrincipal).Scan(&fresh); err != nil || !fresh {
					t.Fatalf("post-freeze observation not recorded: fresh=%v err=%v", fresh, err)
				}
			}
			revoke := f.revocationCommand(enrolled.NodeId, false)
			withdrawals := adapter.WithdrawCalls()
			drained := false
			for i := 0; i < 100; i++ {
				worked, err := dispatcher.Once(t.Context())
				if err != nil {
					t.Fatal(err)
				}
				if !worked {
					drained = true
					break
				}
			}
			if !drained {
				t.Fatal("dispatcher did not reach an empty claim")
			}
			op := f.publicationOperation(revoke.OperationId)
			if op.Status != api.OperationStatusSucceeded && op.Status != api.OperationStatusPartiallySucceeded {
				t.Errorf("revoke status=%s, want succeeded or partially_succeeded", op.Status)
			}
			if adapter.WithdrawCalls() <= withdrawals {
				t.Error("revoke did not withdraw the published route")
			}
			var mode, observed string
			if err := f.r.Pool.QueryRow(t.Context(), `SELECT s.mode,e.publication_observed FROM usage_ledger_state s JOIN exits e ON e.node_id=$1`, enrolled.NodeId).Scan(&mode, &observed); err != nil {
				t.Fatal(err)
			}
			if mode != "reconciling" || observed != "withdrawn" {
				t.Errorf("mode=%s publication_observed=%s", mode, observed)
			}
		})
	}
}
