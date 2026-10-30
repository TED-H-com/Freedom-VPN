package e2e

import (
	"github.com/google/uuid"
	"testing"
)

func TestD034e_RollbackAllowlistRetainsLastGood(t *testing.T) {
	last, candidate := uuid.New(), uuid.New()
	input := []uuid.UUID{candidate, last, candidate, uuid.Nil}
	got := RollbackAllowlist(last, input...)
	if len(got) != 2 || got[0] != last || got[1] != candidate {
		t.Fatalf("allowlist: %v", got)
	}
	got[1] = uuid.Nil
	if input[0] != candidate {
		t.Fatal("input aliased")
	}
	if got := RollbackAllowlist(uuid.Nil); got == nil || len(got) != 0 {
		t.Fatal("empty allowlist must encode as []")
	}
}
