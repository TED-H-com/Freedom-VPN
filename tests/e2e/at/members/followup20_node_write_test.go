//go:build e2e

package members

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

func TestF20_NodeWriteSharedHelperAndReceipts(t *testing.T) {
	for _, mode := range []string{"retry", "already intended", "other 412", "exhausted"} {
		t.Run(mode, func(t *testing.T) {
			writes, reads := 0, 0
			keys, bodies := []string{}, []string{}
			body := map[string]any{"capacity": map[string]any{"down_bps": 200000000, "up_bps": 200000000}}
			payload, _ := json.Marshal(body)
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					reads++
					w.Header().Set("ETag", fmt.Sprintf(`"%d"`, reads))
					if mode == "already intended" && reads > 1 {
						_, _ = w.Write(payload)
					} else {
						_, _ = io.WriteString(w, `{}`)
					}
					return
				}
				if r.Method != "PATCH" || r.Header.Get("X-CSRF-Token") != "TEST ONLY csrf" {
					t.Error("wrong method/CSRF")
				}
				writes++
				b, _ := io.ReadAll(r.Body)
				bodies = append(bodies, string(b))
				keys = append(keys, r.Header.Get("Idempotency-Key"))
				if mode == "other 412" {
					w.WriteHeader(412)
					_, _ = io.WriteString(w, `{"error":{"code":"STATE_CONFLICT"}}`)
					return
				}
				if writes == 1 || mode == "exhausted" {
					w.WriteHeader(412)
					_, _ = io.WriteString(w, `{"error":{"code":"RESOURCE_VERSION_CONFLICT"}}`)
					return
				}
				w.WriteHeader(202)
				_, _ = io.WriteString(w, `{"operation_id":"019a4b00-0000-7000-8000-000000000001"}`)
			}))
			t.Cleanup(server.Close)
			// Open only the shared stack's metadata; no stack, dependency or Docker
			// process starts. The caller transport pins all requests to this stub.
			state := t.TempDir()
			t.Setenv("FVPN_E2E_STATE", state)
			if err := os.WriteFile(filepath.Join(state, "stack.json"), []byte(`{"prefix":"fvpn-test-only","host_ports":{"control":1}}`), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(state, "ca.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0600); err != nil {
				t.Fatal(err)
			}
			stack, err := labstack.Open()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(stack.Close)
			client := server.Client()
			tr := client.Transport.(*http.Transport).Clone()
			tr.TLSClientConfig = tr.TLSClientConfig.Clone()
			tr.TLSClientConfig.ServerName = "example.com"
			tr.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			}
			client.Transport = tr
			t.Cleanup(tr.CloseIdleConnections)
			l := live{ctx: t.Context(), admin: &browser{client: client, csrf: "TEST ONLY csrf"}, hook: stack, node: uuid.New()}
			got := l.nodeMutation("PATCH", "", body)
			want := mode == "retry" || mode == "already intended"
			if got.accepted(202) != want {
				t.Fatalf("acceptance=%t: %s", got.accepted(202), got.detail())
			}
			if mode == "already intended" && (got.status != 412 || !got.write.AlreadyIntended || writes != 1) {
				t.Fatal("invented operation/HTTP success")
			}
			if mode == "other 412" && (writes != 1 || reads != 1) {
				t.Fatal("retried unrelated 412")
			}
			if mode == "exhausted" && (writes != 3 || reads != 4) {
				t.Fatal("retry bound changed")
			}
			rec := record{}
			rec.check("node write", "accepted with exact read-back", got.detail(), "Stack.WriteNode", want)
			var saved struct {
				Receipt labstack.NodeWriteReceipt `json:"node_write"`
			}
			if json.Unmarshal([]byte(rec.Steps[0].Actual), &saved) != nil || len(saved.Receipt.Attempts) != writes {
				t.Fatal("receipt lost from check record")
			}
			for i, a := range saved.Receipt.Attempts {
				if keys[i] == "" || keys[i] != keys[0] || bodies[i] != string(payload) || a.BodySHA256 != fmt.Sprintf("%x", sha256.Sum256(payload)) || a.IfMatch != fmt.Sprintf(`"%d"`, i+1) {
					t.Fatal("key, body, digest or ETag changed", a)
				}
			}
		})
	}
}
func TestF20_NodeWriteExactPredicates(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	groups := []uuid.UUID{a, b}
	for _, intent := range []api.PublicationIntent{api.PublicationIntentPublished, api.PublicationIntentDisabled} {
		pred := nodeMutationPredicate("PUT", "/publication", api.PublicationRequest{PublicationIntent: intent, GroupIds: &groups, Reason: "TEST ONLY"})
		good, _ := json.Marshal(map[string]any{"publication_intent": intent, "group_ids": groups})
		bad, _ := json.Marshal(map[string]any{"publication_intent": intent, "group_ids": []uuid.UUID{a, a}})
		if pred == nil || !pred(good) || pred(bad) {
			t.Fatal("inexact publication predicate")
		}
	}
	if nodeMutationPredicate("POST", "/unknown", nil) != nil {
		t.Fatal("unreviewed command accepted")
	}
}
