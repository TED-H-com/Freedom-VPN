//go:build e2e

package usage

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
	"github.com/TED-H-com/Freedom-VPN/tests/e2e"
	"github.com/google/uuid"
)

func TestNodeWrite_AdoptedCallShapesAndPredicates(t *testing.T) {
	groupA := uuid.MustParse("019a4b00-0000-7000-8000-0000000000a1")
	groupB := uuid.MustParse("019a4b00-0000-7000-8000-0000000000b2")

	t.Run("patch_capacity", func(t *testing.T) {
		body := map[string]any{
			"capacity": map[string]any{
				"up_bps":              int64(1000000000),
				"down_bps":            int64(1000000000),
				"member_up_bps_cap":   int64(30000000),
				"member_down_bps_cap": int64(60000000),
			},
		}
		pred := nodeMutationPredicate(http.MethodPatch, "", body)
		if pred == nil {
			t.Fatal("expected non-nil predicate for PATCH capacity")
		}
		good, _ := json.Marshal(map[string]any{
			"id":              "019a4b00-0000-7000-8000-000000000001",
			"lifecycle_state": "registered",
			"capacity": map[string]any{
				"up_bps":              int64(1000000000),
				"down_bps":            int64(1000000000),
				"member_up_bps_cap":   int64(30000000),
				"member_down_bps_cap": int64(60000000),
			},
		})
		wrongRate, _ := json.Marshal(map[string]any{
			"capacity": map[string]any{
				"up_bps":              int64(1000000000),
				"down_bps":            int64(1000000000),
				"member_up_bps_cap":   int64(30000000),
				"member_down_bps_cap": int64(15000000),
			},
		})
		missingField, _ := json.Marshal(map[string]any{
			"capacity": map[string]any{
				"up_bps":            int64(1000000000),
				"down_bps":          int64(1000000000),
				"member_up_bps_cap": int64(30000000),
			},
		})
		if !pred(good) {
			t.Fatal("expected predicate to accept matching capacity")
		}
		if pred(wrongRate) || pred(missingField) || pred([]byte(`{}`)) {
			t.Fatal("expected predicate to reject wrong or incomplete capacity")
		}
		if nodeMutationPredicate(http.MethodPatch, "", map[string]any{}) != nil {
			t.Fatal("expected empty PATCH body to be rejected")
		}
		if nodeMutationPredicate(http.MethodPatch, "", map[string]any{"capacity": map[string]any{}}) != nil {
			t.Fatal("expected empty capacity map to be rejected")
		}
		if nodeMutationPredicate(http.MethodPatch, "", map[string]any{"reason": "TEST ONLY"}) != nil {
			t.Fatal("expected command-only field in PATCH body to be rejected")
		}
	})

	t.Run("put_publication_published", func(t *testing.T) {
		body := map[string]any{
			"publication_intent": "published",
			"group_ids":          []string{groupA.String(), groupB.String()},
			"reason":             "TEST ONLY publish node",
		}
		pred := nodeMutationPredicate(http.MethodPut, "/publication", body)
		if pred == nil {
			t.Fatal("expected non-nil predicate for PUT /publication published")
		}
		goodReordered, _ := json.Marshal(map[string]any{
			"publication_intent": "published",
			"group_ids":          []string{groupB.String(), groupA.String()},
		})
		wrongIntent, _ := json.Marshal(map[string]any{
			"publication_intent": "disabled",
			"group_ids":          []string{groupA.String(), groupB.String()},
		})
		dupGroups, _ := json.Marshal(map[string]any{
			"publication_intent": "published",
			"group_ids":          []string{groupA.String(), groupA.String()},
		})
		missingGroup, _ := json.Marshal(map[string]any{
			"publication_intent": "published",
			"group_ids":          []string{groupA.String()},
		})
		if !pred(goodReordered) {
			t.Fatal("expected published predicate to accept exact groups regardless of order")
		}
		if pred(wrongIntent) || pred(dupGroups) || pred(missingGroup) {
			t.Fatal("expected published predicate to reject wrong intent, duplicate groups, or missing group")
		}
		if nodeMutationPredicate(http.MethodPut, "/publication", map[string]any{
			"publication_intent": "published",
			"group_ids":          []string{},
			"reason":             "TEST ONLY empty groups",
		}) != nil {
			t.Fatal("expected published intent with empty group_ids to be rejected")
		}
		if nodeMutationPredicate(http.MethodPut, "/publication", map[string]any{
			"publication_intent": "published",
			"group_ids":          []string{groupA.String(), groupA.String()},
			"reason":             "TEST ONLY duplicate groups",
		}) != nil {
			t.Fatal("expected duplicate request group_ids to be rejected")
		}
	})

	t.Run("put_publication_disabled_with_group_and_empty", func(t *testing.T) {
		withGroup := map[string]any{
			"publication_intent": "disabled",
			"group_ids":          []string{groupA.String()},
			"reason":             "TEST ONLY reset publication before publish",
		}
		predGroup := nodeMutationPredicate(http.MethodPut, "/publication", withGroup)
		if predGroup == nil {
			t.Fatal("expected non-nil predicate for disabled publication with group")
		}
		goodWithGroup, _ := json.Marshal(map[string]any{
			"publication_intent": "disabled",
			"group_ids":          []string{groupA.String()},
		})
		goodEmpty, _ := json.Marshal(map[string]any{
			"publication_intent": "disabled",
			"group_ids":          []string{},
		})
		publishedWithGroup, _ := json.Marshal(map[string]any{
			"publication_intent": "published",
			"group_ids":          []string{groupA.String()},
		})
		if !predGroup(goodWithGroup) || predGroup(goodEmpty) || predGroup(publishedWithGroup) {
			t.Fatal("disabled-with-group predicate mismatch")
		}

		emptyGroups := map[string]any{
			"publication_intent": "disabled",
			"group_ids":          []string{},
			"reason":             "TEST ONLY cleanup node publication",
		}
		predEmpty := nodeMutationPredicate(http.MethodPut, "/publication", emptyGroups)
		if predEmpty == nil {
			t.Fatal("expected non-nil predicate for disabled publication with empty groups")
		}
		if !predEmpty(goodEmpty) || predEmpty(goodWithGroup) || predEmpty(publishedWithGroup) {
			t.Fatal("disabled-empty-groups predicate mismatch")
		}
	})

	t.Run("post_revoke_and_delete_retire", func(t *testing.T) {
		revPred := nodeMutationPredicate(http.MethodPost, "/revoke", map[string]any{"reason": "TEST ONLY revoke node"})
		retPred := nodeMutationPredicate(http.MethodDelete, "", map[string]any{"reason": "TEST ONLY retire node"})
		if revPred == nil || retPred == nil {
			t.Fatal("expected non-nil predicates for POST /revoke and DELETE node")
		}
		revokedBody := []byte(`{"lifecycle_state":"revoked"}`)
		retiredBody := []byte(`{"lifecycle_state":"retired"}`)
		registeredBody := []byte(`{"lifecycle_state":"registered"}`)

		if !revPred(revokedBody) || revPred(retiredBody) || revPred(registeredBody) {
			t.Fatal("revoke predicate must match only lifecycle_state=revoked")
		}
		if !retPred(retiredBody) || retPred(revokedBody) || retPred(registeredBody) {
			t.Fatal("retire predicate must match only lifecycle_state=retired")
		}
		if nodeMutationPredicate(http.MethodPost, "/revoke", map[string]any{"reason": "   "}) != nil {
			t.Fatal("revoke without reason must be rejected")
		}
		if nodeMutationPredicate(http.MethodDelete, "", map[string]any{}) != nil {
			t.Fatal("retire without reason must be rejected")
		}
		if nodeMutationPredicate(http.MethodPost, "/unknown", map[string]any{"reason": "TEST ONLY"}) != nil {
			t.Fatal("unknown command must return nil predicate")
		}
		if got := nodeMutationSuccessStatuses(http.MethodPatch, ""); len(got) != 2 || got[0] != http.StatusOK || got[1] != http.StatusAccepted {
			t.Fatalf("PATCH success statuses = %v, want [200 202]", got)
		}
		for _, tc := range []struct {
			method string
			suffix string
		}{
			{http.MethodPut, "/publication"},
			{http.MethodPost, "/revoke"},
			{http.MethodDelete, ""},
		} {
			if got := nodeMutationSuccessStatuses(tc.method, tc.suffix); len(got) != 1 || got[0] != http.StatusAccepted {
				t.Fatalf("%s %s success statuses = %v, want [202]", tc.method, tc.suffix, got)
			}
		}
		if got := nodeMutationSuccessStatuses(http.MethodPost, "/unknown"); got != nil {
			t.Fatalf("unknown command success statuses = %v, want nil", got)
		}
	})
}

func TestNodeWrite_SharedHelperConflictPathsAndReceipts(t *testing.T) {
	groupID := uuid.MustParse("019a4b00-0000-7000-8000-0000000000f1")
	commands := []struct {
		name           string
		method         string
		suffix         string
		acceptStatuses []int
		body           map[string]any
		intendedBody   map[string]any
	}{
		{
			name:           "patch_capacity",
			method:         http.MethodPatch,
			suffix:         "",
			acceptStatuses: []int{http.StatusOK, http.StatusAccepted},
			body: map[string]any{
				"capacity": map[string]any{
					"up_bps":              int64(1000000000),
					"down_bps":            int64(1000000000),
					"member_up_bps_cap":   int64(30000000),
					"member_down_bps_cap": int64(60000000),
				},
			},
			intendedBody: map[string]any{
				"capacity": map[string]any{
					"up_bps":              int64(1000000000),
					"down_bps":            int64(1000000000),
					"member_up_bps_cap":   int64(30000000),
					"member_down_bps_cap": int64(60000000),
				},
			},
		},
		{
			name:           "put_publication_published",
			method:         http.MethodPut,
			suffix:         "/publication",
			acceptStatuses: []int{http.StatusAccepted},
			body: map[string]any{
				"publication_intent": "published",
				"group_ids":          []string{groupID.String()},
				"reason":             "TEST ONLY publish node",
			},
			intendedBody: map[string]any{
				"publication_intent": "published",
				"group_ids":          []string{groupID.String()},
			},
		},
		{
			name:           "put_publication_disabled",
			method:         http.MethodPut,
			suffix:         "/publication",
			acceptStatuses: []int{http.StatusAccepted},
			body: map[string]any{
				"publication_intent": "disabled",
				"group_ids":          []string{groupID.String()},
				"reason":             "TEST ONLY disable node",
			},
			intendedBody: map[string]any{
				"publication_intent": "disabled",
				"group_ids":          []string{groupID.String()},
			},
		},
		{
			name:           "post_revoke",
			method:         http.MethodPost,
			suffix:         "/revoke",
			acceptStatuses: []int{http.StatusAccepted},
			body: map[string]any{
				"reason": "TEST ONLY revoke node",
			},
			intendedBody: map[string]any{
				"lifecycle_state": string(api.LifecycleStateRevoked),
			},
		},
		{
			name:           "delete_retire",
			method:         http.MethodDelete,
			suffix:         "",
			acceptStatuses: []int{http.StatusAccepted},
			body: map[string]any{
				"reason": "TEST ONLY retire node",
			},
			intendedBody: map[string]any{
				"lifecycle_state": string(api.LifecycleStateRetired),
			},
		},
	}

	modes := []string{"retry", "already_intended", "status_200", "other_412", "exhausted", "initial_get_failed"}

	for _, cmdSpec := range commands {
		for _, mode := range modes {
			t.Run(cmdSpec.name+"/"+mode, func(t *testing.T) {
				writes, reads := 0, 0
				keys, bodies := []string{}, []string{}
				payload, _ := json.Marshal(cmdSpec.body)
				intendedJSON, _ := json.Marshal(cmdSpec.intendedBody)

				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet {
						reads++
						if mode == "initial_get_failed" && reads == 1 {
							w.WriteHeader(http.StatusServiceUnavailable)
							_, _ = io.WriteString(w, `{"error":{"code":"DEPENDENCY_UNAVAILABLE"}}`)
							return
						}
						w.Header().Set("ETag", fmt.Sprintf(`"%d"`, reads))
						if mode == "already_intended" && reads > 1 {
							_, _ = w.Write(intendedJSON)
						} else {
							_, _ = io.WriteString(w, `{"lifecycle_state":"registered","publication_intent":"unpublished","group_ids":[]}`)
						}
						return
					}
					if r.Method != cmdSpec.method || r.Header.Get("X-CSRF-Token") != "TEST ONLY csrf" {
						t.Errorf("unexpected method=%s or CSRF=%q", r.Method, r.Header.Get("X-CSRF-Token"))
					}
					writes++
					b, _ := io.ReadAll(r.Body)
					bodies = append(bodies, string(b))
					keys = append(keys, r.Header.Get("Idempotency-Key"))
					if mode == "status_200" {
						w.Header().Set("ETag", `"2"`)
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write(intendedJSON)
						return
					}
					if mode == "other_412" {
						w.WriteHeader(http.StatusPreconditionFailed)
						_, _ = io.WriteString(w, `{"error":{"code":"STATE_CONFLICT"}}`)
						return
					}
					if writes == 1 || mode == "exhausted" {
						w.WriteHeader(http.StatusPreconditionFailed)
						_, _ = fmt.Fprintf(w, `{"error":{"code":"RESOURCE_VERSION_CONFLICT","details":{"current_resource_version":"%d"}}}`, writes+1)
						return
					}
					w.WriteHeader(http.StatusAccepted)
					_, _ = io.WriteString(w, `{"operation_id":"019a4b00-0000-7000-8000-000000000001"}`)
				}))
				t.Cleanup(server.Close)

				state := t.TempDir()
				t.Setenv("FVPN_E2E_STATE", state)
				if err := os.WriteFile(filepath.Join(state, "stack.json"), []byte(`{"prefix":"fvpn-test-only","host_ports":{"control":1}}`), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(state, "ca.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), 0o600); err != nil {
					t.Fatal(err)
				}
				stack, err := e2e.OpenStack()
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

				nodeID := uuid.MustParse("019a4b00-0000-7000-8000-000000000099")
				s := &liveSuite{
					t:              t,
					ctx:            t.Context(),
					hook:           stack,
					admin:          &browserSession{client: client, csrf: "TEST ONLY csrf", subject: "admin"},
					nodeID:         nodeID,
					groupID:        groupID,
					runID:          "testrun1",
					build:          BuildMetadata{GitCommit: "0123456"},
					imageDigests:   map[string]string{"control": "sha256:0000"},
					activePolicies: map[string]string{},
				}

				got := s.nodeMutation(nodeID, cmdSpec.method, cmdSpec.suffix, cmdSpec.body)
				wantAccepted := mode == "retry" || mode == "already_intended" || (mode == "status_200" && cmdSpec.method == http.MethodPatch)
				if got.accepted(cmdSpec.acceptStatuses...) != wantAccepted {
					t.Fatalf("accepted(%v)=%t want=%t detail=%s", cmdSpec.acceptStatuses, got.accepted(cmdSpec.acceptStatuses...), wantAccepted, got.detail())
				}

				switch mode {
				case "retry":
					if writes != 2 || reads != 2 || got.status != http.StatusAccepted || got.write.Receipt.Outcome != "accepted" {
						t.Fatalf("retry mismatch: writes=%d reads=%d status=%d outcome=%s", writes, reads, got.status, got.write.Receipt.Outcome)
					}
				case "already_intended":
					if writes != 1 || reads != 2 || got.status != http.StatusPreconditionFailed || !got.write.AlreadyIntended || got.write.Receipt.Outcome != "already_intended" {
						t.Fatalf("already_intended must preserve 412 without fabricating an operation: writes=%d reads=%d status=%d receipt=%+v", writes, reads, got.status, got.write.Receipt)
					}
				case "status_200":
					wantOutcome := "non_conflict_response"
					wantStatus := e2e.StatusBlocked
					if cmdSpec.method == http.MethodPatch {
						wantOutcome = "accepted"
						wantStatus = e2e.StatusPass
					}
					if writes != 1 || reads != 1 || got.status != http.StatusOK || len(got.write.Receipt.Attempts) != 1 ||
						got.write.Receipt.Attempts[0].Status != http.StatusOK || got.write.Receipt.Status != wantStatus ||
						got.write.Receipt.Outcome != wantOutcome || got.write.AlreadyIntended {
						t.Fatalf("status_200 mismatch: writes=%d reads=%d status=%d receipt=%+v", writes, reads, got.status, got.write.Receipt)
					}
				case "other_412":
					if writes != 1 || reads != 1 || got.write.Receipt.Outcome != "non_conflict_response" {
						t.Fatalf("other_412 must not reread or retry: writes=%d reads=%d outcome=%s", writes, reads, got.write.Receipt.Outcome)
					}
				case "exhausted":
					if writes != 3 || reads != 4 || got.write.Receipt.Outcome != "attempt_limit" {
						t.Fatalf("exhausted must stop after 3 writes and 3 rereads: writes=%d reads=%d outcome=%s", writes, reads, got.write.Receipt.Outcome)
					}
				case "initial_get_failed":
					if writes != 0 || reads != 1 {
						t.Fatalf("initial_get_failed must refuse blind write: writes=%d reads=%d", writes, reads)
					}
				}

				for i, a := range got.write.Receipt.Attempts {
					wantDigest := fmt.Sprintf("%x", sha256.Sum256(payload))
					if keys[i] == "" || keys[i] != keys[0] || bodies[i] != string(payload) || a.BodySHA256 != wantDigest || a.IfMatch != fmt.Sprintf(`"%d"`, i+1) {
						t.Fatalf("attempt %d key/body/digest/ETag mismatch: attempt=%+v key=%s", i, a, keys[i])
					}
				}

				// Verify receipt embedding in suite records and check details.
				rec := s.baseRecord("AT16", "Node Write Receipt Embedding Test", "§14.5", "Node writes embed receipts.", []string{"API01"})
				rec.Commands = append(rec.Commands, CommandRecord{
					Command:  cmdSpec.method + " /api/v1/admin/nodes/{id}" + cmdSpec.suffix,
					ExitCode: 0,
					Output:   got.detail(),
					At:       "2026-11-03T00:00:00Z",
				})
				rec.MechanismChecks = append(rec.MechanismChecks, MechanismCheck{
					Name:    "NodeWriteReceiptEmbedded",
					SpecRef: "§14.5",
					Status:  StatusPass,
					Summary: "Node write receipt embedded in check details and observations.",
					Details: map[string]any{
						"node_write": got.write.Receipt,
					},
				})
				rec.Actual = "Offline node write test completed."

				outDir := t.TempDir()
				if err := WriteEvidence(outDir, rec, "TEST ONLY csrf"); err != nil {
					t.Fatalf("WriteEvidence failed: %v", err)
				}
				rawJSON, err := os.ReadFile(filepath.Join(outDir, "AT16.json"))
				if err != nil {
					t.Fatalf("read AT16.json: %v", err)
				}
				var saved struct {
					Observations struct {
						NodeWrites []e2e.NodeWriteReceipt `json:"node_writes"`
					} `json:"observations"`
					MechanismChecks []struct {
						Details struct {
							NodeWrite e2e.NodeWriteReceipt `json:"node_write"`
						} `json:"details"`
					} `json:"mechanism_checks"`
				}
				if err := json.Unmarshal(rawJSON, &saved); err != nil {
					t.Fatalf("unmarshal saved record: %v", err)
				}
				if len(saved.Observations.NodeWrites) != 1 || len(saved.Observations.NodeWrites[0].Attempts) != writes ||
					len(saved.MechanismChecks) != 1 || len(saved.MechanismChecks[0].Details.NodeWrite.Attempts) != writes {
					t.Fatalf("embedded receipt lost from saved JSON: %+v", saved)
				}
			})
		}
	}
}
