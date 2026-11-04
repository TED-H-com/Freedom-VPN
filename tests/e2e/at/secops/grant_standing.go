//go:build e2e

package secops

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
	"strings"
	"time"
)

type grantAPI15 interface {
	Call(context.Context, labstack.BrowserSession, string, string, any, map[string]string) (labstack.Response, error)
}

// All standing changes and operation reads use the public contract. In
// particular cleanup has its own context: a failed/cancelled test cannot skip it.
func grantCommand15(ctx context.Context, stack grantAPI15, admin labstack.BrowserSession, path string, body any, row map[string]any) (api.Operation, error) {
	if !strings.HasPrefix(path, "/api/v1/admin/members/") {
		return api.Operation{}, fmt.Errorf("member command path required")
	}
	row["started_at"] = time.Now().UTC()
	key := uuid.NewString()
	r, err := stack.Call(ctx, admin, "POST", path, body, map[string]string{"Idempotency-Key": key})
	row["http_status"], row["error_code"] = r.Status, r.Code
	row["returned_at"] = time.Now().UTC()
	var op api.Operation
	if err != nil || r.Status != 202 || json.Unmarshal(r.Body, &op) != nil || op.OperationId == uuid.Nil {
		return op, fmt.Errorf("grant command not accepted: status=%d code=%s", r.Status, r.Code)
	}
	row["accepted_operation"] = op
	replay, replayErr := stack.Call(ctx, admin, "POST", path, body, map[string]string{"Idempotency-Key": key})
	var repeated api.Operation
	replayOK := replayErr == nil && replay.Status == 202 && json.Unmarshal(replay.Body, &repeated) == nil && repeated.OperationId == op.OperationId && repeated.Resource == op.Resource
	row["same_key_replay"] = map[string]any{"http_status": replay.Status, "same_operation_and_resource": replayOK}
	if !replayOK {
		return op, fmt.Errorf("same-key grant command replay not confirmed")
	}
	return op, nil
}

func readOperation15(ctx context.Context, stack grantAPI15, admin labstack.BrowserSession, id uuid.UUID) (api.Operation, bool) {
	r, err := stack.Call(ctx, admin, "GET", "/api/v1/admin/operations/"+id.String(), nil, nil)
	var op api.Operation
	ok := err == nil && r.Status == 200 && json.Unmarshal(r.Body, &op) == nil && op.OperationId == id
	return op, ok
}

func memberStanding15(ctx context.Context, stack grantAPI15, admin labstack.BrowserSession, member string, end time.Time) (api.MemberDetail, bool) {
	r, err := stack.Call(ctx, admin, "GET", "/api/v1/admin/members/"+member, nil, nil)
	var detail api.MemberDetail
	if err != nil || r.Status != 200 || json.Unmarshal(r.Body, &detail) != nil {
		return detail, false
	}
	actual, err := detail.Entitlement.EffectiveServiceEnd.Get()
	return detail, err == nil && actual.Equal(end) && !detail.Entitlement.AdminSuspended
}

// Restore the same finite standing on normal and early-return paths. This is
// deliberately detached from testing.T and from the failed subject's context.
func restoreStanding15(ctx context.Context, stack grantAPI15, admin labstack.BrowserSession, member string, device uuid.UUID, body api.GrantCreateRequest, row map[string]any) error {
	detail, standing := memberStanding15(ctx, stack, admin, member, body.ValidUntil)
	var op api.Operation
	if !standing {
		var err error
		op, err = grantCommand15(ctx, stack, admin, "/api/v1/admin/members/"+member+"/grants", body, row)
		if err != nil {
			return err
		}
	} else {
		row["original_standing_retained"] = true
	}
	for ctx.Err() == nil {
		detail, standing = memberStanding15(ctx, stack, admin, member, body.ValidUntil)
		if standing {
			break
		}
		if !pauseStanding15(ctx) {
			break
		}
	}
	row["entitlement"] = detail.Entitlement
	if !standing {
		return fmt.Errorf("restored entitlement not observed")
	}
	if device != uuid.Nil {
		authorization := map[string]any{}
		row["explicit_device_authorization"] = authorization
		_, err := grantCommand15(ctx, stack, admin, "/api/v1/admin/members/"+member+"/devices/"+device.String()+"/authorize", nil, authorization)
		if err != nil {
			return err
		}
	}
	for ctx.Err() == nil {
		operationOK := op.OperationId == uuid.Nil
		if !operationOK {
			actual, ok := readOperation15(ctx, stack, admin, op.OperationId)
			row["completed_operation"] = actual
			operationOK = ok && actual.Status == api.OperationStatusSucceeded
		}
		deviceOK := device == uuid.Nil
		if !deviceOK {
			r, err := stack.Call(ctx, admin, "GET", "/api/v1/admin/devices?member_id="+member, nil, nil)
			var page struct {
				Items []api.MemberDevice `json:"items"`
			}
			if err == nil && r.Status == 200 && json.Unmarshal(r.Body, &page) == nil {
				for _, d := range page.Items {
					if d.DeviceId == device {
						row["device"] = map[string]any{"device_id": d.DeviceId, "status": d.Status, "authorization_unconfirmed": d.AuthorizationUnconfirmed}
						deviceOK = d.Status == api.DeviceStatusAuthorized && d.AuthorizationUnconfirmed != nil && !*d.AuthorizationUnconfirmed
					}
				}
			}
		}
		if operationOK && deviceOK {
			row["status"] = "restored"
			return nil
		}
		if !pauseStanding15(ctx) {
			break
		}
	}
	return fmt.Errorf("restoration operation/device read-back timeout")
}

func pauseStanding15(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Second):
		return true
	}
}
