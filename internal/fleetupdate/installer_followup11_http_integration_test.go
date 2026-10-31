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

func TestD111_InstallerNullBaselineHTTPStoredAck(t *testing.T) {
	for _, state := range []api.UpdateResultState{api.UpdateResultStateFailed, api.UpdateResultStateRolledBack, api.UpdateResultStateSucceeded} {
		t.Run(string(state), func(t *testing.T) {
			fleet, nodes, pool, result := fleetupdate.NewInstallerNullBaselineFixture(t, state)
			composite, err := httpapi.Compose(&httpwire.Handlers{Runtime: &httpwire.Runtime{Pool: pool, Nodes: nodes, Fleet: fleet}})
			if err != nil {
				t.Fatal(err)
			}
			c := config.Defaults()
			c.PublicOrigin = "https://portal.example.invalid"
			identity := auth.Principal{Role: auth.Agent, ID: result.NodeId, NodeID: result.NodeId, IncarnationID: result.IncarnationId}
			options := httpapi.Options{Config: c, AgentListener: true, Commands: &httpapi.Commands{Pool: pool, Retention: time.Hour}, Agents: auth.AgentAuthenticatorFunc(func(context.Context, *x509.Certificate) (auth.Principal, error) { return identity, nil })}
			raw, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			key := uuid.NewString()
			var first []byte
			var version uint64
			for attempt := range 3 {
				// Reconstruct the HTTP stack: the stored receipt, rather than a
				// process cache, supplies both keyed and domain replay.
				handler, err := httpapi.New(composite, options)
				if err != nil {
					t.Fatal(err)
				}
				request := httptest.NewRequest(http.MethodPost, "https://control.example.invalid/api/v1/agent/update-results", bytes.NewReader(raw))
				request.Header.Set("Content-Type", "application/json")
				if attempt == 2 {
					key = uuid.NewString()
				}
				request.Header.Set("Idempotency-Key", key)
				// The HTTP boundary receives a verified certificate; authentication
				// supplies the fixture's bound identity, then fleet authenticates it.
				request.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{Raw: []byte("TEST ONLY installer certificate")}}}}
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusAccepted {
					t.Fatal(response.Code, response.Body.String())
				}
				var ack api.UpdateResultAck
				if err := json.Unmarshal(response.Body.Bytes(), &ack); err != nil || !ack.Accepted || !ack.AppliedToCurrentObservation || ack.ResultId != result.ResultId {
					t.Fatal(ack, err)
				}
				detail, _, err := nodes.GetNode(t.Context(), result.NodeId)
				if err != nil {
					t.Fatal(err)
				}
				if attempt == 0 {
					first = append([]byte(nil), response.Body.Bytes()...)
					version = detail.ResourceVersion.Uint64()
				} else if !bytes.Equal(first, response.Body.Bytes()) || detail.ResourceVersion.Uint64() != version {
					t.Fatal("replay changed stored ACK or ETag")
				}
				current, _ := detail.Releases.CurrentReleaseId.Get()
				good, _ := detail.Releases.LastGoodReleaseId.Get()
				want, _ := result.CurrentReleaseId.Get()
				if current != want || good != want {
					t.Fatal("HTTP receipt did not establish actual baseline", current, good, want)
				}
			}
			var rows, audits int
			if err := pool.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM installer_update_results),(SELECT count(*) FROM audit_events WHERE action='update_result.accept')`).Scan(&rows, &audits); err != nil || rows != 1 || audits != 1 {
				t.Fatal("replay amplified evidence", rows, audits, err)
			}
		})
	}
}
