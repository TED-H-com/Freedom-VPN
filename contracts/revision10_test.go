package contracts

import (
	"strings"
	"testing"
)

func TestAPI01_Revision10EnrollmentDigestIncarnationsAndBlockReason(t *testing.T) {
	doc := loadDoc(t)
	if doc.Info.Version != "1.6.0" {
		t.Fatal("API version must be 1.6.0")
	}
	if n := len(operations(t)); n != 47 {
		t.Fatalf("got %d operations, want 47", n)
	}
	item := doc.Paths.Value("/admin/nodes/{node_id}/incarnations")
	if item == nil || item.Get == nil || item.Get.OperationID != "listNodeIncarnations" {
		t.Fatal("missing listNodeIncarnations")
	}
	if item.Get.Responses.Status(200) == nil || item.Get.Responses.Status(404) == nil || item.Get.Responses.Status(202) != nil {
		t.Fatal("incarnation list must be synchronous 200 and 404 for an unknown node")
	}
	enrollment := doc.Components.Schemas["Enrollment"].Value
	token := enrollment.Properties["token_sha256"].Value
	if token == nil || !token.ReadOnly || token.Pattern != "^[0-9a-f]{64}$" {
		t.Fatal("token_sha256 must be a read-only sha256 hex digest")
	}
	required := map[string]bool{}
	for _, name := range enrollment.Required {
		required[name] = true
	}
	if !required["token_sha256"] {
		t.Fatal("token_sha256 must be required on Enrollment")
	}
	reason := doc.Components.Schemas["ForwardingBlockReason"].Value
	var values []string
	for _, v := range reason.Enum {
		values = append(values, v.(string))
	}
	joined := strings.Join(values, ",")
	if !strings.Contains(joined, "lease_expired,awaiting_member_grants") {
		t.Fatalf("awaiting_member_grants must sit beside lease_expired, got %s", joined)
	}
	if !strings.Contains(reason.Description, "valid node lease") || !strings.Contains(reason.Description, "Do not report `lease_expired`") {
		t.Fatal("block-reason description must distinguish awaiting_member_grants from lease_expired")
	}
	audit := doc.Components.Schemas["AuditEvent"].Value.Description + doc.Components.Schemas["AuditEvent"].Value.Properties["changes"].Value.Description
	for _, phrase := range []string{"token_sha256", "never redacted"} {
		if !strings.Contains(audit, phrase) {
			t.Fatalf("audit description missing %q", phrase)
		}
	}
	inc := doc.Components.Schemas["NodeIncarnation"].Value
	for _, name := range []string{"incarnation_id", "state", "enrollment_id", "created_at", "bound_at", "retired_at", "retire_reason", "bound_public_key_sha256", "current", "usage"} {
		if inc.Properties[name] == nil {
			t.Fatalf("NodeIncarnation missing %s", name)
		}
	}
	usage := doc.Components.Schemas["NodeIncarnationUsage"].Value.Description
	if !strings.Contains(inc.Description, "node_incarnations") || !strings.Contains(usage, "usage_checkpoints") || !strings.Contains(usage, "usage_samples") {
		t.Fatal("incarnation schema must name the columns it is read from")
	}
	raw := readFile(t, "examples/valid/NodeIncarnationList.json")
	if err := validateComponent(t, "NodeIncarnationList", raw); err != nil {
		t.Fatal(err)
	}
	report := readFile(t, "examples/valid/ObservationReport.awaiting-member-grants.json")
	if !strings.Contains(string(report), "awaiting_member_grants") {
		t.Fatal("observation example must carry awaiting_member_grants")
	}
	if err := validateComponent(t, "ObservationReport", report); err != nil {
		t.Fatal(err)
	}
}
