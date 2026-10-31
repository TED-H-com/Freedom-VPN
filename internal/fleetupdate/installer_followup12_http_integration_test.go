//go:build integration

package fleetupdate_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/config"
	"github.com/TED-H-com/Freedom-VPN/internal/fleetupdate"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi"
	"github.com/TED-H-com/Freedom-VPN/internal/nodes/httpwire"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestD127_CorrectionHTTPReplaysBothStoredAcks(t *testing.T) {
	for _, seed := range []bool{true, false} {
		t.Run(map[bool]string{true: "seed_success", false: "success_rollback"}[seed], func(t *testing.T) {
			fleet, nodes, pool, results := fleetupdate.NewInstallerCorrectionFixture(t, seed)
			composite, err := httpapi.Compose(&httpwire.Handlers{Runtime: &httpwire.Runtime{Pool: pool, Nodes: nodes, Fleet: fleet}})
			if err != nil {
				t.Fatal(err)
			}
			c := config.Defaults()
			c.PublicOrigin = "https://portal.example.invalid"
			identity := auth.Principal{Role: auth.Agent, ID: results[0].NodeId, NodeID: results[0].NodeId, IncarnationID: results[0].IncarnationId}
			options := httpapi.Options{Config: c, AgentListener: true, Commands: &httpapi.Commands{Pool: pool, Retention: time.Hour}, Agents: auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) { return identity, nil })}
			bodies := make([][]byte, 2)
			acks := make([][]byte, 2)
			keys := []string{uuid.NewString(), uuid.NewString()}
			var version uint64
			for j := range results {
				bodies[j], err = json.Marshal(results[j])
				if err != nil {
					t.Fatal(err)
				}
			}
			for attempt, j := range []int{0, 1, 0, 1, 0, 1} {
				// Recreate the HTTP stack and exercise both the existing keyed receipt
				// reader and domain replay with a fresh idempotency key for each outcome.
				handler, err := httpapi.New(composite, options)
				if err != nil {
					t.Fatal(err)
				}
				key := keys[j]
				if attempt >= 4 {
					key = uuid.NewString()
				}
				request := httptest.NewRequest(http.MethodPost, "https://control.example.invalid/api/v1/agent/update-results", bytes.NewReader(bodies[j]))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Idempotency-Key", key)
				request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{Raw: []byte("TEST ONLY installer certificate")}}}}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusAccepted {
					t.Fatal(response.Code, response.Body.String())
				}
				var ack api.UpdateResultAck
				if err = json.Unmarshal(response.Body.Bytes(), &ack); err != nil || !ack.Accepted || !ack.AppliedToCurrentObservation || ack.ResultId != results[j].ResultId {
					t.Fatal(ack, err)
				}
				if attempt < 2 {
					acks[j] = append([]byte(nil), response.Body.Bytes()...)
				} else if !bytes.Equal(acks[j], response.Body.Bytes()) {
					t.Fatal("HTTP replay returned another outcome's ACK")
				}
				detail, _, err := nodes.GetNode(t.Context(), identity.NodeID)
				if err != nil {
					t.Fatal(err)
				}
				if attempt < 2 {
					version = detail.ResourceVersion.Uint64()
				} else if detail.ResourceVersion.Uint64() != version {
					t.Fatal("HTTP replay changed ETag")
				}
				latest := j
				if attempt >= 2 {
					latest = 1
				}
				want, _ := results[latest].CurrentReleaseId.Get()
				current, _ := detail.Releases.CurrentReleaseId.Get()
				good, _ := detail.Releases.LastGoodReleaseId.Get()
				summary, e := detail.Update.Get()
				if e != nil || current != want || good != want || string(summary.State) != string(results[latest].State) {
					t.Fatal("HTTP node projection", detail.Releases, summary, e)
				}
			}
			var rows, audits int
			err = pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM installer_update_results),(SELECT count(*) FROM audit_events WHERE action='update_result.accept')`).Scan(&rows, &audits)
			if err != nil || rows != 2 || audits != 2 {
				t.Fatal("HTTP replay amplified receipts", rows, audits, err)
			}
		})
	}
}
