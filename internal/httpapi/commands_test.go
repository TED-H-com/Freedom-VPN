package httpapi

import (
	"context"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/db"
)

func TestAPI01_CommandRecordUnavailableOutsideTransaction(t *testing.T) {
	for _, ctx := range []context.Context{
		t.Context(),
		context.WithValue(t.Context(), commandKey{}, (*commandState)(nil)),
		context.WithValue(t.Context(), commandKey{}, &commandState{}),
	} {
		if record, ok := CommandRecord(ctx); ok || record != (db.IdempotencyRecord{}) {
			t.Fatalf("record outside command transaction: %+v, %v", record, ok)
		}
	}
}
