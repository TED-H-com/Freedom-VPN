//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

// GrantRecoveryOperation records acceptance separately from worker completion.
// It excludes response bodies and transport error text.
type GrantRecoveryOperation struct {
	GrantID     uuid.UUID `json:"grant_id"`
	OperationID uuid.UUID `json:"operation_id"`
	Status      int       `json:"http_status"`
	ErrorCode   string    `json:"error_code"`
	Accepted    bool      `json:"accepted"`
	Outcome     string    `json:"outcome"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
}

type GrantRestoration struct {
	RevokedGrantID uuid.UUID `json:"revoked_grant_id"`
	PlanRevisionID uuid.UUID `json:"plan_revision_id"`
	ValidFrom      time.Time `json:"valid_from"`
	ValidUntil     time.Time `json:"valid_until"`
	GrantRecoveryOperation
}

type finiteGrant struct {
	ID, Plan    uuid.UUID
	From, Until time.Time
}

type finiteGrantSteps struct {
	revoke  func(context.Context, uuid.UUID, uuid.UUID) error
	call    func(context.Context, string, string, any, map[string]string) (reply, error)
	created func(context.Context, string) ([]uuid.UUID, error)
	project func(context.Context, *GrantTransition) bool
}

// The production path and stub harness execute this same transition, including
// every early return. Recovery has its own bounded context so caller cancellation
// cannot skip restoring already revoked grants.
func replaceFiniteGrants(ctx context.Context, out GrantTransition, grants []finiteGrant, steps finiteGrantSteps) (result GrantTransition, err error) {
	var revoked []finiteGrant
	var createKey string
	defer func() {
		if err != nil && len(revoked) != 0 {
			clean, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
			defer cancel()
			if recoveryErr := restoreFiniteGrants(clean, &out, revoked, createKey, steps); recoveryErr != nil {
				err = errors.Join(err, recoveryErr)
			}
		}
		result = out
	}()
	// The deferred copy retains recovery receipts on every early return.
	return runFiniteGrantTransition(ctx, &out, grants, &revoked, &createKey, steps)
}

func runFiniteGrantTransition(ctx context.Context, out *GrantTransition, grants []finiteGrant, revoked *[]finiteGrant, createKey *string, steps finiteGrantSteps) (GrantTransition, error) {
	for _, grant := range grants {
		key := uuid.New()
		if err := steps.revoke(ctx, grant.ID, key); err != nil {
			return *out, fmt.Errorf("finite grant: production revoke failed: %w", err)
		}
		*revoked = append(*revoked, grant)
		out.Revoked = append(out.Revoked, grant.ID)
		if err := steps.revoke(ctx, grant.ID, key); err != nil {
			return *out, fmt.Errorf("finite grant: revoke replay failed: %w", err)
		}
	}
	out.RevokeReplayVerified = true
	*createKey = uuid.NewString()
	body := api.GrantCreateRequest{PlanRevisionId: &out.PlanRevisionID, Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY finite plan transition", ValidUntil: out.ValidUntil}
	path := "/api/v1/admin/members/" + out.MemberID.String() + "/grants"
	resp, err := steps.call(ctx, "POST", path, body, map[string]string{"Idempotency-Key": *createKey})
	if err != nil || resp.Status != 202 {
		return *out, fmt.Errorf("finite grant: create HTTP %d", resp.Status)
	}
	out.Accepted = true
	var accepted struct {
		ID uuid.UUID `json:"operation_id"`
	}
	if json.Unmarshal(resp.Body, &accepted) != nil || accepted.ID == uuid.Nil {
		return *out, fmt.Errorf("finite grant: operation absent")
	}
	out.OperationID = accepted.ID
	replay, e := steps.call(ctx, "POST", path, body, map[string]string{"Idempotency-Key": *createKey})
	var replayOp struct {
		ID uuid.UUID `json:"operation_id"`
	}
	out.CreateReplayVerified = e == nil && replay.Status == 202 && json.Unmarshal(replay.Body, &replayOp) == nil && replayOp.ID == out.OperationID
	if !out.CreateReplayVerified {
		return *out, fmt.Errorf("finite grant: create replay did not return the same operation")
	}
	wait, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	out.Projected = waitUntil(wait, time.Second, func() bool { return steps.project(wait, out) })
	if !out.Projected {
		return *out, fmt.Errorf("finite grant: worker operation/projection deadline")
	}
	return *out, nil
}

func restoreFiniteGrants(ctx context.Context, out *GrantTransition, revoked []finiteGrant, createKey string, steps finiteGrantSteps) error {
	var errs []error
	path := "/api/v1/admin/members/" + out.MemberID.String()
	post := func(path string, body any) GrantRecoveryOperation {
		row := GrantRecoveryOperation{StartedAt: time.Now().UTC(), Outcome: "request_failed"}
		resp, err := steps.call(ctx, "POST", path, body, map[string]string{"Idempotency-Key": uuid.NewString()})
		row.FinishedAt, row.Status, row.ErrorCode = time.Now().UTC(), resp.Status, resp.Code
		var op api.Operation
		if err == nil && resp.Status == 202 && json.Unmarshal(resp.Body, &op) == nil && op.OperationId != uuid.Nil {
			row.OperationID, row.GrantID, row.Accepted, row.Outcome = op.OperationId, op.Resource.Id, true, "accepted"
		}
		return row
	}
	if createKey != "" {
		ids, err := steps.created(ctx, createKey)
		if err != nil {
			errs = append(errs, err)
		}
		for _, id := range ids {
			row := post(path+"/grants/"+id.String()+"/revoke", api.ReasonRequest{Reason: "TEST ONLY rollback finite plan transition"})
			row.GrantID = id
			out.ReplacementRevocations = append(out.ReplacementRevocations, row)
			if !row.Accepted {
				errs = append(errs, fmt.Errorf("finite grant: replacement withdrawal failed for %s", id))
			}
		}
	}
	// Attempt every restoration once before waiting for operations; a failed
	// withdrawal or restoration must not skip the remaining original grants.
	for _, grant := range revoked {
		row := GrantRestoration{RevokedGrantID: grant.ID, PlanRevisionID: grant.Plan, ValidFrom: grant.From, ValidUntil: grant.Until}
		row.GrantRecoveryOperation = post(path+"/grants", api.GrantCreateRequest{PlanRevisionId: &grant.Plan, ValidFrom: &grant.From, ValidUntil: grant.Until, Source: api.GrantSourceAdminGrant, Reason: "TEST ONLY restore revoked finite grant"})
		out.Restorations = append(out.Restorations, row)
		if !row.Accepted {
			errs = append(errs, fmt.Errorf("finite grant: restoration failed for %s (HTTP %d)", grant.ID, row.Status))
		}
	}
	// Read back live grant identity/plan/validity and worker completion. Acceptance
	// alone never reports a successful restoration.
	for i := range out.Restorations {
		row := &out.Restorations[i]
		if !row.Accepted {
			continue
		}
		ok := waitUntil(ctx, time.Second, func() bool {
			resp, err := steps.call(ctx, "GET", "/api/v1/admin/operations/"+row.OperationID.String(), nil, nil)
			var op api.Operation
			if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &op) != nil || op.Status != "succeeded" {
				return false
			}
			if row.GrantID == uuid.Nil {
				row.GrantID = op.Resource.Id
			}
			resp, err = steps.call(ctx, "GET", path, nil, nil)
			var member api.MemberDetail
			if err != nil || resp.Status != 200 || json.Unmarshal(resp.Body, &member) != nil {
				return false
			}
			for _, grant := range member.Grants {
				plan, e := grant.PlanRevisionId.Get()
				if grant.GrantId == row.GrantID && grant.RevokedAt.IsNull() && e == nil && plan == row.PlanRevisionID && grant.ValidFrom.Equal(row.ValidFrom) && grant.ValidUntil.Equal(row.ValidUntil) && grant.ValidUntil.After(time.Now()) {
					return true
				}
			}
			return false
		})
		row.FinishedAt = time.Now().UTC()
		if ok {
			row.Outcome = "restored"
		} else {
			row.Outcome = "verification_failed"
			errs = append(errs, fmt.Errorf("finite grant: restoration verification failed for %s", row.RevokedGrantID))
		}
	}
	return errors.Join(errs...)
}
