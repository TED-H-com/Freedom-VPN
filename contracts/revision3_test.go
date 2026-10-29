package contracts

import (
	"encoding/json"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
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

func TestUPD05_AppliedPolicyGenerationHash(t *testing.T) {
	raw := readFile(t, "examples/valid/UpdateResult.applied-policy-generation.json")
	var result api.UpdateResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.PolicyGeneration.String() != "9007199254740993" || result.PolicyGeneration == result.Generation {
		t.Fatal("applied generation lost precision or confused with job generation")
	}
	var v struct {
		Result json.RawMessage `json:"result"`
		Hash   string          `json:"payload_hash"`
	}
	readJSON(t, "vectors/update-result-policy-generation-hash.json", &v)
	got, err := canonjson.HashHex(v.Result, "payload_hash")
	if err != nil || got != v.Hash {
		t.Fatalf("hash=%s err=%v", got, err)
	}
}

func TestAT02_Checkout201Only(t *testing.T) {
	doc := loadDoc(t)
	op := doc.Paths.Value("/member/checkout-sessions").Post
	if op.Responses.Status(202) != nil {
		t.Fatal("checkout must not return an asynchronous success")
	}
	success := op.Responses.Status(201)
	if success == nil || success.Value.Content.Get("application/json").Schema.Ref != "#/components/schemas/CheckoutSession" {
		t.Fatal("checkout success must carry checkout_url")
	}
	failure := op.Responses.Status(503)
	if failure == nil || failure.Ref != "#/components/responses/ServiceUnavailable" {
		t.Fatal("provider failure must use 503")
	}
	if apierr.DependencyUnavailable.HTTPStatus() != 503 || !apierr.DependencyUnavailable.DefaultRetryable() {
		t.Fatal("provider failure must be retryable")
	}
}
