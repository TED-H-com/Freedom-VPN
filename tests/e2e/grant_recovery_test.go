//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

func TestENT01_FiniteGrantRecovery(t *testing.T) {
	for _, mode := range []string{"create", "lost_create_response", "replay", "projection", "revoke_replay", "later_revoke", "restoration_failure", "success"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			member, oldPlan, plan, replacement, operation := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
			from, until := time.Now().UTC().Add(-time.Hour), time.Now().UTC().Add(time.Hour)
			grants := []finiteGrant{{uuid.New(), oldPlan, from, until}}
			if mode == "later_revoke" || mode == "restoration_failure" {
				grants = append(grants, finiteGrant{uuid.New(), oldPlan, from.Add(time.Minute), until.Add(time.Minute)})
			}
			initial := GrantTransition{MemberID: member, PlanRevisionID: plan, ValidUntil: until, Revoked: []uuid.UUID{}, BeforeRevision: "4", DeviceLimit: 2}
			var requests, restores, withdrawals int
			var createKey, createBody string
			var restoredGrants []map[string]any
			var restoreOps []uuid.UUID
			path := "/api/v1/admin/members/" + member.String()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					if r.URL.Path == path {
						_ = json.NewEncoder(w).Encode(map[string]any{"grants": restoredGrants})
					} else {
						_ = json.NewEncoder(w).Encode(map[string]any{"status": "succeeded"})
					}
					return
				}
				if r.URL.Path == path+"/grants/"+replacement.String()+"/revoke" {
					withdrawals++
					w.WriteHeader(202)
					_ = json.NewEncoder(w).Encode(map[string]any{"operation_id": uuid.New(), "resource": map[string]any{"id": replacement}})
					return
				}
				if r.URL.Path != path+"/grants" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				var body api.GrantCreateRequest
				if json.NewDecoder(r.Body).Decode(&body) != nil {
					t.Error("malformed grant request")
				}
				if body.Reason == "TEST ONLY finite plan transition" {
					requests++
					raw, _ := json.Marshal(body)
					if requests == 1 {
						createKey, createBody = r.Header.Get("Idempotency-Key"), string(raw)
					} else if createKey != r.Header.Get("Idempotency-Key") || createBody != string(raw) {
						t.Error("changed create replay key/body")
					}
					if mode == "create" || mode == "lost_create_response" || mode == "restoration_failure" || (mode == "replay" && requests == 2) {
						w.WriteHeader(503)
						return
					}
					w.WriteHeader(202)
					_ = json.NewEncoder(w).Encode(map[string]any{"operation_id": operation, "resource": map[string]any{"id": replacement}})
					return
				}
				i := restores
				restores++
				if i >= len(grants) {
					t.Error("duplicate restoration")
					w.WriteHeader(500)
					return
				}
				grant := grants[i]
				if body.PlanRevisionId == nil || *body.PlanRevisionId != grant.Plan || body.ValidFrom == nil || !body.ValidFrom.Equal(grant.From) || !body.ValidUntil.Equal(grant.Until) || body.Source != api.GrantSourceAdminGrant || r.Header.Get("Idempotency-Key") == "" {
					t.Errorf("original validity/plan lost: %+v", body)
				}
				if mode == "restoration_failure" && i == 0 {
					w.WriteHeader(503)
					return
				}
				id, op := uuid.New(), uuid.New()
				restoreOps = append(restoreOps, op)
				restoredGrants = append(restoredGrants, map[string]any{"grant_id": id, "plan_revision_id": grant.Plan, "valid_from": grant.From, "valid_until": grant.Until, "revoked_at": nil})
				w.WriteHeader(202)
				_ = json.NewEncoder(w).Encode(map[string]any{"operation_id": op, "resource": map[string]any{"id": id}})
			}))
			defer server.Close()
			stack := &Stack{lab: &lab{public: server.URL}}
			session := BrowserSession{Client: server.Client()}
			revokes := 0
			revokeKeys := map[uuid.UUID]uuid.UUID{}
			sentinel := errors.New("TEST ONLY revoke failure")
			got, err := replaceFiniteGrants(ctx, initial, grants, finiteGrantSteps{
				revoke: func(_ context.Context, id, key uuid.UUID) error {
					revokes++
					if prior, ok := revokeKeys[id]; ok && prior != key {
						t.Error("changed revoke replay key")
					}
					revokeKeys[id] = key
					if mode == "revoke_replay" && revokes == 2 || mode == "later_revoke" && revokes == 3 {
						return sentinel
					}
					return nil
				},
				call: func(ctx context.Context, method, path string, body any, headers map[string]string) (reply, error) {
					if ctx.Err() != nil {
						t.Error("recovery reused canceled context")
					}
					return stack.Call(ctx, session, method, path, body, headers)
				},
				created: func(ctx context.Context, key string) ([]uuid.UUID, error) {
					if ctx.Err() != nil || key != createKey {
						t.Error("wrong recovery context/key")
					}
					if mode == "create" || mode == "restoration_failure" {
						return nil, nil
					}
					return []uuid.UUID{replacement}, nil
				},
				project: func(_ context.Context, out *GrantTransition) bool {
					if mode == "projection" {
						cancel()
						return false
					}
					out.AfterRevision, out.Authorized, out.Pending = "5", 2, 1
					return true
				},
			})
			if mode == "success" {
				if err != nil || !got.Accepted || !got.Projected || !got.CreateReplayVerified || !got.RevokeReplayVerified || got.AfterRevision != "5" || restores != 0 || withdrawals != 0 || len(got.Restorations) != 0 {
					t.Fatalf("normal path changed: %+v %v", got, err)
				}
				return
			}
			if err == nil || restores != len(got.Revoked) || len(got.Restorations) != restores {
				t.Fatalf("recovery absent/duplicated: %+v %v", got, err)
			}
			if mode == "revoke_replay" || mode == "later_revoke" {
				if !errors.Is(err, sentinel) {
					t.Fatal("original error lost")
				}
			}
			wantOriginal := map[string]string{"create": "create HTTP 503", "lost_create_response": "create HTTP 503", "replay": "create replay did not return the same operation", "projection": "worker operation/projection deadline", "restoration_failure": "create HTTP 503"}[mode]
			if !strings.Contains(err.Error(), wantOriginal) {
				t.Fatalf("original error lost: %v", err)
			}
			for i, row := range got.Restorations {
				if row.RevokedGrantID != grants[i].ID || row.PlanRevisionID != grants[i].Plan || !row.ValidFrom.Equal(grants[i].From) || !row.ValidUntil.Equal(grants[i].Until) || row.StartedAt.Location() != time.UTC || row.FinishedAt.Before(row.StartedAt) {
					t.Fatalf("receipt=%+v", row)
				}
				if mode == "restoration_failure" && i == 0 {
					if row.Accepted || row.Outcome != "request_failed" || !strings.Contains(err.Error(), "restoration failed") {
						t.Fatalf("restoration failure missing: %+v %v", row, err)
					}
				} else if row.Outcome != "restored" || row.OperationID == uuid.Nil || row.GrantID == uuid.Nil {
					t.Fatalf("restoration not verified: %+v", row)
				}
			}
			if mode == "replay" || mode == "projection" || mode == "lost_create_response" {
				if withdrawals != 1 || len(got.ReplacementRevocations) != 1 || !got.ReplacementRevocations[0].Accepted {
					t.Fatal("own replacement not withdrawn")
				}
			}
		})
	}
}

func TestENT01_ExpiredFiniteGrantHasNoEffects(t *testing.T) {
	got, err := (&Stack{}).ReplaceFiniteGrant(t.Context(), BrowserSession{}, uuid.New(), uuid.New(), time.Now().Add(-time.Second))
	if err == nil || len(got.Revoked) != 0 || len(got.Restorations) != 0 || got.BeforeRevision != "" || got.AfterRevision != "" {
		t.Fatal(fmt.Sprint(got, err))
	}
}
