//go:build integration

package operations

import (
	"github.com/TED-H-com/Freedom-VPN/internal/auth"
	"github.com/TED-H-com/Freedom-VPN/internal/db/dbtest"
	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"testing"
)

func TestAPI01_OperationGeneratedEnumsAndErrorFallback(t *testing.T) {
	pool := dbtest.New(t)
	store := Store{Pool: pool}
	request := CreateRequest{Kind: api.OperationKindMemberUpdate, Principal: auth.Principal{Role: auth.Member, ID: uuid.New()}, Resource: api.ResourceRef{Type: api.ResourceTypeMember, Id: uuid.New()}, RequestID: uuid.New(), Steps: map[string]Step{"apply": {Required: true}}}
	spec, err := gen.GetSpec()
	if err != nil {
		t.Fatal(err)
	}
	var id uuid.UUID
	for _, kind := range spec.Components.Schemas["OperationKind"].Value.Enum {
		request.Kind = api.OperationKind(kind.(string))
		err = store.InTx(t.Context(), func(tx pgx.Tx) error { var e error; id, e = Create(t.Context(), tx, request); return e })
		if err != nil {
			t.Fatalf("contract kind %s: %v", kind, err)
		}
	}
	for _, kind := range []api.OperationKind{"node.enroll", "member.device_authorize", "future.kind"} {
		request.Kind = kind
		if _, err = Create(t.Context(), nil, request); err == nil {
			t.Fatal("non-contract kind accepted", kind)
		}
	}
	request.Kind = api.OperationKindMemberUpdate
	request.Resource.Type = "device"
	if _, err = Create(t.Context(), nil, request); err == nil {
		t.Fatal("invalid resource accepted")
	}
	code := api.ErrorCodeDEPENDENCYUNAVAILABLE
	if err = store.InTx(t.Context(), func(tx pgx.Tx) error { return Transition(t.Context(), tx, id, "apply", "failed", nil, &code, nil) }); err != nil {
		t.Fatal(err)
	}
	var stored, step, message string
	if err = pool.QueryRow(t.Context(), `SELECT error_code,error_message,steps->'apply'->>'error_code' FROM operations WHERE id=$1`, id).Scan(&stored, &message, &step); err != nil || stored != string(code) || step != stored || message == "" {
		t.Fatal(stored, message, step, err)
	}
	// Domain compatibility: top-level error must survive when a producer omits
	// the step error code. This is a reader fallback, not a principal relaxation.
	if _, err = pool.Exec(t.Context(), `UPDATE operations SET steps=steps #- '{apply,error_code}',error_message='Dependency read-back failed.' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	op, err := store.Get(t.Context(), id, request.Principal)
	if err != nil {
		t.Fatal(err)
	}
	failure, getErr := op.Error.Get()
	if getErr != nil || failure.Code != code || failure.Message != "Dependency read-back failed." {
		t.Fatal(op.Error)
	}
	if err = store.InTx(t.Context(), func(tx pgx.Tx) error { return Transition(t.Context(), tx, id, "apply", "running", nil, nil, nil) }); err != nil {
		t.Fatal(err)
	}
	var cleared bool
	if err = pool.QueryRow(t.Context(), `SELECT error_code IS NULL AND error_message IS NULL FROM operations WHERE id=$1`, id).Scan(&cleared); err != nil || !cleared {
		t.Fatal("retry retained failure", err)
	}
}
