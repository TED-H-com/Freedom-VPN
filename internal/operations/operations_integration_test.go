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
	if e != nil || op.Status != "partially_succeeded" || (!op.CompletedAt.IsSpecified() || op.CompletedAt.IsNull()) {
		t.Fatal(op.Status, e)
	}
	if e = transition("netbird", "running", nil, nil); e != nil {
		t.Fatal(e)
	}
	if e = transition("netbird", "succeeded", evidence, nil); e != nil {
		t.Fatal(e)
	}
	op, e = s.Get(t.Context(), id, p)
	if e != nil || op.Status != "succeeded" || (!op.CompletedAt.IsSpecified() || op.CompletedAt.IsNull()) {
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

func TestAPI01_OperationCreationAndFrozenTerminalRows(t *testing.T) {
	pool := dbtest.New(t)
	s := Store{Pool: pool}
	r := CreateRequest{Kind: "member.update", Principal: auth.Principal{Role: auth.Member, ID: uuid.New()}, Resource: api.ResourceRef{Type: "member", Id: uuid.New()}, RequestID: uuid.New(), Steps: map[string]Step{"optional": {Reason: "No external effect."}}}
	if err := s.InTx(t.Context(), func(tx pgx.Tx) error {
		_, err := Create(t.Context(), tx, r)
		return err
	}); err == nil {
		t.Fatal("operation with no required work was created")
	}
	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM operations`).Scan(&count); err != nil || count != 0 {
		t.Fatal("invalid operation persisted", count, err)
	}
	for _, terminal := range []api.OperationStepStatus{"succeeded", "cancelled"} {
		t.Run(string(terminal), func(t *testing.T) {
			r.Steps["readback"] = Step{Required: true}
			var id uuid.UUID
			if err := s.InTx(t.Context(), func(tx pgx.Tx) error {
				var err error
				id, err = Create(t.Context(), tx, r)
				return err
			}); err != nil {
				t.Fatal(err)
			}
			op, err := s.Get(t.Context(), id, r.Principal)
			if err != nil || op.Status != "pending" || !op.CompletedAt.IsNull() {
				t.Fatal("new operation must be pending", op.Status, err)
			}
			evidence := &Evidence{Reference: "observation:" + uuid.NewString(), ObservedAt: time.Now()}
			if err := s.InTx(t.Context(), func(tx pgx.Tx) error {
				return Transition(t.Context(), tx, id, "readback", terminal, evidence, nil, nil)
			}); err != nil {
				t.Fatal(err)
			}
			var original string
			if err := pool.QueryRow(t.Context(), `SELECT to_jsonb(o)::text FROM operations o WHERE id=$1`, id).Scan(&original); err != nil {
				t.Fatal(err)
			}
			if err := s.InTx(t.Context(), func(tx pgx.Tx) error { return Transition(t.Context(), tx, id, "readback", "running", nil, nil, nil) }); err == nil {
				t.Fatal("terminal transition accepted")
			}
			for _, query := range []string{
				`UPDATE operations SET observed_refs='{"late":"unverified"}' WHERE id=$1`,
				`UPDATE operations SET updated_at=clock_timestamp() WHERE id=$1`,
				`DELETE FROM operations WHERE id=$1`,
			} {
				if _, err := pool.Exec(t.Context(), query, id); err == nil {
					t.Fatal("terminal history changed", query)
				}
			}
			var after string
			if err := pool.QueryRow(t.Context(), `SELECT to_jsonb(o)::text FROM operations o WHERE id=$1`, id).Scan(&after); err != nil || after != original {
				t.Fatal("terminal row was not frozen", err)
			}
		})
	}
}
