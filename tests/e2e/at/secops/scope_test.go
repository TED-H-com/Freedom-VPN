//go:build e2e

package secops

import (
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/httpapi/gen"
)

// §19 AT32 also requires three-source E2E evidence. This check is deliberately
// restricted to the HTTP schema surface; it cannot prove absence of code branches.
func TestAT32_UnifiedNodeContractSurface(t *testing.T) {
	doc, err := gen.GetSwagger()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"EnrollmentCreateRequest", "NodePatch", "EnrollRequest", "UpdateBatchCreateRequest"} {
		schema := doc.Components.Schemas[name]
		if schema == nil {
			t.Fatalf("missing contract schema %s", name)
		}
		for _, field := range []string{"source_trust", "trust_level", "partner_id", "owner_role", "approval_status", "maintenance_ticket", "shell_command", "source"} {
			if _, ok := schema.Value.Properties[field]; ok {
				t.Errorf("%s adds source privilege/workflow field %s", name, field)
			}
		}
	}
	for path := range doc.Paths.Map() {
		for _, forbidden := range []string{"partner", "approval", "maintenance", "shell"} {
			if strings.Contains(path, forbidden) {
				t.Errorf("out-of-scope API route %s", path)
			}
		}
	}
	t.Log("reviewed enrollment, node patch, agent enrollment and update batch input surfaces; no source privilege or partner workflow field")
}
