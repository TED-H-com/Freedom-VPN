//go:build integration

package nodes_test

import (
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
	"testing"
)

func TestMemberGrantOperation_NodeEnforcementACKAndLookup(t *testing.T) {
	w := newD165World(t, 1, false)
	var opID uuid.UUID
	if err := w.r.Pool.QueryRow(t.Context(), `SELECT operation_id FROM entitlement_grants WHERE id=$1`, w.grant).Scan(&opID); err != nil {
		t.Fatal(err)
	}
	ops, err := w.r.Nodes.ListMemberOperations(t.Context(), w.member)
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 {
		t.Fatalf("member operations=%d, want grant and device authorization", len(ops))
	}
	found := false
	for _, op := range ops {
		if op.OperationId == opID {
			found = true
			if op.Resource.Type != api.ResourceTypeEntitlementGrant || op.Resource.Id != w.grant {
				t.Fatalf("grant resource=%+v", op.Resource)
			}
		}
	}
	if !found {
		t.Fatal("grant missing from member lookup")
	}
	got, err := w.r.Nodes.GetMemberOperation(t.Context(), w.member, opID)
	if err != nil || got.OperationId != opID {
		t.Fatalf("GetMemberOperation=%+v err=%v", got, err)
	}
	if _, err = w.r.Nodes.GetMemberOperation(t.Context(), uuid.New(), opID); err == nil {
		t.Fatal("unrelated member read the grant")
	}
	if got.Status != api.OperationStatusRunning {
		t.Fatalf("netbird-only status=%s, want running", got.Status)
	}
	w.observe(w.report(w.reports[0], "clean"))
	got, err = w.r.Nodes.GetMemberOperation(t.Context(), w.member, opID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != api.OperationStatusSucceeded {
		t.Fatalf("finalOp.Status=%s, want succeeded", got.Status)
	}
	for _, st := range got.Steps {
		if st.Status != api.OperationStepStatusSucceeded || st.ObservedAt.IsNull() {
			t.Fatalf("step %s=%+v, want succeeded with observation", st.Name, st)
		}
	}
}
