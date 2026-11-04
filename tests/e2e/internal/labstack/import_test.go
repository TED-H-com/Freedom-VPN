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

func TestAPI01_NodeWriteImportSurface(t *testing.T) {
	var stack *labstack.Stack
	result := stack.WriteNode(t.Context(), labstack.BrowserSession{}, "PATCH", "/api/v1/admin/nodes/test-only", "/api/v1/admin/nodes/test-only", nil, `"1"`, labstack.NodeRevoked, 202)
	var receipt labstack.NodeWriteReceipt = result.Receipt
	if result.Err == nil || result.Accepted(202) || len(receipt.Attempts) != 0 {
		t.Fatal("invalid caller accepted")
	}
	if !labstack.NodeFieldsMatch(map[string]string{"lifecycle_state": "retired"})([]byte(`{"lifecycle_state":"retired"}`)) || labstack.NodePublished(uuid.New())([]byte(`{}`)) {
		t.Fatal("exported predicate mismatch")
	}
}
