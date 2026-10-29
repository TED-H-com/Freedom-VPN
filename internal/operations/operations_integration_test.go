//go:build integration

package operations

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestAT28_OperationOwnershipAndObservedEvidence(t *testing.T) {
	pool := dbtest.New(t)
	s := Store{Pool: pool}
	p := auth.Principal{Role: auth.Member, ID: uuid.New()}
	var id uuid.UUID
	e := s.InTx(t.Context(), func(tx pgx.Tx) error {
		var e error
		id, e = Create(t.Context(), tx, CreateRequest{Kind: "member.device_revoke", Principal: p, Resource: api.ResourceRef{Type: "member_device", Id: uuid.New()}, RequestID: uuid.New(), Steps: map[string]Step{"agent": {Required: true}, "netbird": {Required: true}, "optional": {Reason: "Not needed for this command."}}})
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	other := p
	other.ID = uuid.New()
	if _, e = s.Get(t.Context(), id, other); !errors.Is(e, apierr.New(apierr.ResourceNotFound, "")) {
		t.Fatal("cross-member access", e)
	}
	transition := func(step string, status api.OperationStepStatus, evidence *Evidence, code *api.ErrorCode) error {
		return s.InTx(t.Context(), func(tx pgx.Tx) error { return Transition(t.Context(), tx, id, step, status, evidence, code, nil) })
	}
	if e = transition("agent", "succeeded", nil, nil); e == nil {
		t.Fatal("success without evidence")
	}
	evidence := &Evidence{Reference: "observation:" + uuid.NewString(), ObservedAt: time.Now()}
	if e = transition("agent", "succeeded", evidence, nil); e != nil {
		t.Fatal(e)
	}
	op, e := s.Get(t.Context(), id, p)
	if e != nil || op.Status != "running" {
		t.Fatal(op.Status, e)
	}
	code := api.ErrorCodeDEPENDENCYUNAVAILABLE
	if e = transition("netbird", "failed", nil, &code); e != nil {
		t.Fatal(e)
	}
	op, e = s.Get(t.Context(), id, p)
	if e != nil || op.Status != "partially_succeeded" || op.CompletedAt == nil {
		t.Fatal(op.Status, e)
	}
	if e = transition("netbird", "running", nil, nil); e != nil {
		t.Fatal(e)
	}
	if e = transition("netbird", "succeeded", evidence, nil); e != nil {
		t.Fatal(e)
	}
	op, e = s.Get(t.Context(), id, p)
	if e != nil || op.Status != "succeeded" || op.CompletedAt == nil {
		t.Fatal(op.Status, e)
	}
	if e = transition("agent", "running", nil, nil); e == nil {
		t.Fatal("terminal operation changed")
	}
	ctx := auth.WithPrincipal(context.Background(), other)
	if _, e = s.GetMemberOperation(ctx, gen.GetMemberOperationRequestObject{OperationId: id}); !errors.Is(e, apierr.New(apierr.ResourceNotFound, "")) {
		t.Fatal(e)
	}
}
