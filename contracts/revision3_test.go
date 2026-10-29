package contracts

import (
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/getkin/kin-openapi/openapi3"
)

func TestAPI01_Revision3Enums(t *testing.T) {
	for _, kind := range []api.OperationKind{api.OperationKindNodeEnroll, api.OperationKindMemberDeviceAuthorize, api.OperationKindMemberGrantRevoke} {
		if !kind.Valid() {
			t.Fatalf("invalid generated kind %s", kind)
		}
	}
	if !api.ResourceTypeAdminIdentity.Valid() {
		t.Fatal("admin_identity missing")
	}
	if api.OperationKind("node.provision").Valid() || api.ErrorCode("LEDGER_MODE_CONFLICT").Valid() {
		t.Fatal("unapproved enum accepted")
	}
}

func TestAT01_BrowserRoleSelectorOutsideAPI(t *testing.T) {
	doc := loadDoc(t)
	for _, path := range []string{"/auth/session", "/auth/logout"} {
		if doc.Paths.Value(path) != nil {
			t.Fatalf("browser route %s added to API", path)
		}
	}
	ext := doc.Extensions["x-fvpn-browser-auth"].(map[string]any)
	param := ext["parameters"].([]any)[0].(map[string]any)
	if param["name"] != "role" || param["in"] != "query" || param["required"] != false {
		t.Fatal(param)
	}
	// Assert the documented query enum using the same schema validator as API inputs.
	raw := param["schema"].(map[string]any)
	schema := &openapi3.Schema{Type: &openapi3.Types{"string"}, Enum: raw["enum"].([]any)}
	for _, role := range []string{"admin", "member", "", "administrator", "node"} {
		err := schema.VisitJSON(role)
		if (err == nil) != (role == "admin" || role == "member") {
			t.Fatalf("role=%q: %v", role, err)
		}
	}
}
