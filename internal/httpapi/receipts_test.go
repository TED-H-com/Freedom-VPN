package httpapi

import (
	"context"
	"github.com/TED-H-com/Freedom-VPN/internal/db"
	"github.com/google/uuid"
	"testing"
)

func TestAPI01_SetResultOperatorAndReceiptRules(t *testing.T) {
	id := uuid.New()
	for _, receipt := range []bool{false, true} {
		ctx := context.WithValue(t.Context(), commandKey{}, &commandState{agentReceipt: receipt})
		err := SetResult(ctx, db.IdempotencyResult{ResourceType: "node", ResourceID: &id, ResponseStatus: 202})
		if (err == nil) != receipt {
			t.Fatal("202 operation requirement", receipt, err)
		}
	}
	ctx := context.WithValue(t.Context(), commandKey{}, &commandState{agentReceipt: true})
	if err := SetResult(ctx, db.IdempotencyResult{ResponseStatus: 202}); err == nil {
		t.Fatal("receipt without durable reference accepted")
	}
}
