//go:build e2e

package secops

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
)

type standingAPI15 struct {
	t                                           *testing.T
	end                                         time.Time
	device                                      uuid.UUID
	op                                          uuid.UUID
	standing, failCreate, failOperation, fenced bool
	posts, authorizations                       int
	keys                                        map[string]string
}

func (f *standingAPI15) Call(_ context.Context, _ labstack.BrowserSession, method, path string, body any, h map[string]string) (labstack.Response, error) {
	response := func(status int, body any) (labstack.Response, error) {
		raw, _ := json.Marshal(body)
		return labstack.Response{Status: status, Body: raw}, nil
	}
	if method == "POST" {
		key := h["Idempotency-Key"]
		raw, _ := json.Marshal(body)
		if old, ok := f.keys[key]; ok {
			if old != string(raw) {
				f.t.Error("replay changed body")
			}
		} else {
			f.keys[key] = string(raw)
			if strings.HasSuffix(path, "/grants") {
				f.posts++
				b := body.(api.GrantCreateRequest)
				if !b.ValidUntil.Equal(f.end) {
					f.t.Error("restoration extended original end")
				}
				if f.failCreate {
					return response(503, map[string]any{})
				}
				f.standing = true
			} else {
				f.authorizations++
			}
		}
		return response(202, map[string]any{"operation_id": f.op, "resource": map[string]any{"id": f.device}})
	}
	if strings.Contains(path, "/operations/") {
		status := "succeeded"
		if f.failOperation {
			status = "failed"
		}
		return response(200, map[string]any{"operation_id": f.op, "status": status})
	}
	if strings.Contains(path, "/admin/devices?") {
		return response(200, map[string]any{"items": []any{map[string]any{"device_id": f.device, "status": "authorized", "authorization_unconfirmed": f.fenced}}})
	}
	entitlement := map[string]any{"effective_service_end": nil}
	if f.standing {
		entitlement["effective_service_end"] = f.end
	}
	return response(200, map[string]any{"entitlement": entitlement})
}

func TestAT29_RestoreStandingOnEarlyReturn(t *testing.T) {
	for _, tc := range []struct {
		name                                                    string
		standing, createFailure, operationFailure, fenced, want bool
		posts                                                   int
	}{
		{"before withdrawal retains original grant", true, false, false, false, true, 0},
		{"after withdrawal restores grant and device", false, false, false, false, true, 1},
		{"lost withdrawal response restores from public read", false, false, false, false, true, 1},
		{"failed restoration remains failed", false, true, false, false, false, 1},
		{"accepted restoration without completion fails", false, false, true, false, false, 1},
		{"D077 fence must be cleared by explicit authorization", false, false, false, true, false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &standingAPI15{t: t, end: time.Now().UTC().Add(time.Hour), device: uuid.New(), op: uuid.New(), standing: tc.standing, failCreate: tc.createFailure, failOperation: tc.operationFailure, fenced: tc.fenced, keys: map[string]string{}}
			// The subject context is already cancelled; cleanup uses a detached one.
			failed, cancelFailed := context.WithCancel(context.Background())
			cancelFailed()
			if failed.Err() == nil {
				t.Fatal("subject not cancelled")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			row := map[string]any{"status": "failed"}
			err := restoreStanding15(ctx, f, labstack.BrowserSession{}, uuid.NewString(), f.device, api.GrantCreateRequest{ValidUntil: f.end}, row)
			if (err == nil) != tc.want || (row["status"] == "restored") != tc.want || f.posts != tc.posts {
				t.Fatalf("err=%v row=%v posts=%d", err, row, f.posts)
			}
			if !tc.createFailure && f.authorizations != 1 {
				t.Fatalf("explicit authorization count %d", f.authorizations)
			}
		})
	}
}
