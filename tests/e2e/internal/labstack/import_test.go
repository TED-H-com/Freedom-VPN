//go:build e2e

package labstack_test

import (
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
	"github.com/google/uuid"
	"testing"
)

func TestHarnessImportAndRollbackOptions(t *testing.T) {
	id := uuid.New()
	options := labstack.BaselineOptions{Candidates: true, IncludeBootInRollback: true, Rollback: labstack.ReleaseOptions{AllowedReleaseIDs: labstack.RollbackAllowlist(id)}}
	if len(options.Rollback.AllowedReleaseIDs) != 1 || options.Rollback.AllowedReleaseIDs[0] != id {
		t.Fatal("last-good lost")
	}
	var stack *labstack.Stack
	if _, err := stack.ScaleNativeMembers(t.Context(), labstack.BrowserSession{}, id, "scale", 21, 3); err == nil {
		t.Fatal("unbounded scale accepted")
	}
}
