package contracts

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestAT28_BootstrapMTLSAndOneTimeDelivery(t *testing.T) {
	doc := loadDoc(t)
	op := doc.Paths.Value("/agent/bootstrap").Post
	if op == nil || op.RequestBody != nil || op.Extensions["x-fvpn-auth"] != "agent-mtls" || op.Security == nil || len(*op.Security) != 0 {
		t.Fatal("bootstrap must be mTLS without a body")
	}
	for _, status := range []int{200, 409, 410} {
		response := op.Responses.Status(status)
		if response == nil || response.Value.Headers["Cache-Control"].Ref != "#/components/headers/Cache-Control-No-Store" {
			t.Fatalf("%d missing no-store", status)
		}
	}
	for _, code := range []string{"BOOTSTRAP_CONSUMED", "BOOTSTRAP_NOT_READY", "BOOTSTRAP_EXPIRED"} {
		if !strings.Contains(op.Description, code) {
			t.Errorf("missing %s", code)
		}
	}
	s := doc.Components.Schemas["AgentBootstrap"].Value
	if len(s.Properties) != 3 || len(s.Required) != 3 || s.AdditionalProperties.Has == nil || *s.AdditionalProperties.Has {
		t.Fatal("bootstrap must contain exactly three required members")
	}
	for _, field := range []string{"external_key_id", "setup_key", "expires_at"} {
		if !slices.Contains(s.Required, field) {
			t.Errorf("missing %s", field)
		}
	}
	for _, name := range []string{"EnrollResponse", "Enrollment"} {
		if _, ok := doc.Components.Schemas[name].Value.Properties["setup_key"]; ok {
			t.Errorf("%s exposes setup key", name)
		}
	}
}

func TestAT07_ReinstallationBindsExistingNode(t *testing.T) {
	s := loadDoc(t).Components.Schemas["EnrollmentCreateRequest"].Value
	field := s.Properties["node_id"]
	if field == nil || field.Value.Format != "uuid" || field.Value.Nullable || slices.Contains(s.Required, "node_id") {
		t.Fatal("node_id must be optional UUID, non-null")
	}
	for _, name := range []string{"EnrollmentCreateRequest.installer.json", "EnrollmentCreateRequest.reinstall.json"} {
		if err := validateComponent(t, "EnrollmentCreateRequest", readFile(t, "examples/valid/"+name)); err != nil {
			t.Fatal(err)
		}
	}
	op := loadDoc(t).Paths.Value("/admin/enrollments").Post
	for _, phrase := range []string{"NODE_INCARNATION_ACTIVE", "NODE_RETIRED", "never clear a revoked lifecycle"} {
		if !strings.Contains(op.Description, phrase) {
			t.Errorf("missing precondition %s", phrase)
		}
	}
}

func TestUPD05_UpdateTargetAttentionAndConfirmation(t *testing.T) {
	raw := readFile(t, "examples/valid/UpdateTarget.json")
	for _, field := range []string{"needs_attention", "unconfirmed"} {
		var v map[string]any
		if err := json.Unmarshal(raw, &v); err != nil {
			t.Fatal(err)
		}
		for _, value := range []any{true, false} {
			v[field] = value
			encoded, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateComponent(t, "UpdateTarget", encoded); err != nil {
				t.Fatal(err)
			}
		}
		v[field] = nil
		encoded, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateComponent(t, "UpdateTarget", encoded); err == nil {
			t.Fatalf("%s accepted null", field)
		}
	}
}
