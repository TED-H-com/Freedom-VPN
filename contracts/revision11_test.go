package contracts

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
)

func TestAPI01_Revision11AwaitingMemberGrantsAndIdentityConflict(t *testing.T) {
	doc := loadDoc(t)
	// Revision 13 advances info.version to 1.9.0 with the operation count still 49. This test still pins revision 11's decisions.
	if doc.Info.Version != "1.9.0" {
		t.Fatal("API version must be 1.9.0")
	}
	if n := len(operations(t)); n != 49 {
		t.Fatalf("got %d operations, want 49", n)
	}

	var enum []string
	for _, v := range doc.Components.Schemas["ErrorCode"].Value.Enum {
		enum = append(enum, v.(string))
	}
	at := map[string]int{}
	for i, code := range enum {
		at[code] = i
	}
	if at["AWAITING_MEMBER_GRANTS"] != at["LEASE_EXPIRED"]+1 {
		t.Fatal("AWAITING_MEMBER_GRANTS must follow LEASE_EXPIRED")
	}
	if !api.ErrorCodeAWAITINGMEMBERGRANTS.Valid() {
		t.Fatal("generated ErrorCode AWAITING_MEMBER_GRANTS is not valid")
	}
	desc := doc.Components.Schemas["ErrorCode"].Value.Description
	for _, phrase := range []string{
		"inside its anchored deadline",
		"D-067 schedule",
		"not a deadline failure",
		"not a malformed policy",
	} {
		if !strings.Contains(desc, phrase) {
			t.Fatalf("ErrorCode description missing %q", phrase)
		}
	}

	summary := doc.Components.Schemas["NodeSummary"].Value
	if summary.Properties["identity_conflict"] != nil {
		t.Fatal("identity_conflict must not be on NodeSummary")
	}
	detail := doc.Components.Schemas["NodeDetail"].Value.AllOf[1].Value
	prop := detail.Properties["identity_conflict"].Value
	if prop == nil || !prop.Nullable || contains(detail.Required, "identity_conflict") {
		t.Fatal("NodeDetail.identity_conflict must be optional and nullable")
	}
	if prop.AllOf[0].Ref != "#/components/schemas/NodeIdentityConflict" {
		t.Fatal("identity_conflict must reference NodeIdentityConflict")
	}
	conflict := doc.Components.Schemas["NodeIdentityConflict"].Value
	for _, name := range []string{"mapping_frozen", "conflict_count", "last_conflict_at"} {
		if conflict.Properties[name] == nil || !contains(conflict.Required, name) {
			t.Fatalf("NodeIdentityConflict.%s must be required", name)
		}
	}
	count := conflict.Properties["conflict_count"].Value
	if count.Min == nil || *count.Min != 0 || count.Max == nil || *count.Max != 9007199254740991 {
		t.Fatal("conflict_count must be bounded by the JSON-number ceiling")
	}
	when := conflict.Properties["last_conflict_at"].Value
	if !when.Nullable || when.Format != "date-time" || when.Pattern != utcPattern {
		t.Fatal("last_conflict_at must be a nullable UTC date-time")
	}
	if !strings.Contains(prop.Description, "pre-revision-11") || !strings.Contains(prop.Description, "never exposed") {
		t.Fatal("identity_conflict description must distinguish absent from null and forbid endpoint exposure")
	}

	obs := doc.Components.Schemas["ObservationReport"].Value.Description
	check := doc.Components.Schemas["NodeCheck"].Value.Description
	for _, phrase := range []string{
		"AWAITING_MEMBER_GRANTS",
		"Serving lease has no member grants yet.",
		"Rate policy could not be applied.",
		"No member binding was admitted.",
		"exactly one",
		"stage=lease",
		"safety_status",
		"passed",
		"not `VALIDATION_FAILED`",
		"LEASE_EXPIRED",
		"Serving lease is not in force.",
		"policy_missing",
		"STALE_GENERATION",
		"device_bindings",
	} {
		if !strings.Contains(obs, phrase) {
			t.Fatalf("ObservationReport description missing %q", phrase)
		}
	}
	for _, phrase := range []string{
		"name=lease",
		"name=qos",
		"name=source_binding",
		"AWAITING_MEMBER_GRANTS",
		"not `VALIDATION_FAILED`",
		"Serving lease has no member grants yet.",
		"Rate policy could not be applied.",
		"No member binding was admitted.",
		"LEASE_EXPIRED",
		"STALE_GENERATION",
	} {
		if !strings.Contains(check, phrase) {
			t.Fatalf("NodeCheck description missing %q", phrase)
		}
	}
	reason := doc.Components.Schemas["ForwardingBlockReason"].Value
	var values []string
	for _, v := range reason.Enum {
		values = append(values, v.(string))
	}
	joined := strings.Join(values, ",")
	if !strings.Contains(joined, "lease_expired,awaiting_member_grants") || strings.Contains(joined, "AWAITING_MEMBER_GRANTS") {
		t.Fatalf("forwarding_block_reasons vocabulary changed: %s", joined)
	}

	raw := readFile(t, "examples/valid/ObservationReport.awaiting-member-grants.json")
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	body["safety_status"] = "passed"
	body["forwarding_block_reasons"] = []any{"awaiting_member_grants"}
	body["checks"] = []any{
		map[string]any{
			"name": "lease", "status": "failed", "observed_at": "2026-10-28T15:00:30Z",
			"error_code": "AWAITING_MEMBER_GRANTS", "detail": "Serving lease has no member grants yet.",
		},
		map[string]any{
			"name": "qos", "status": "failed", "observed_at": "2026-10-28T15:00:30Z",
			"error_code": "AWAITING_MEMBER_GRANTS", "detail": "Rate policy could not be applied.",
		},
		map[string]any{
			"name": "source_binding", "status": "failed", "observed_at": "2026-10-28T15:00:30Z",
			"error_code": "AWAITING_MEMBER_GRANTS", "detail": "No member binding was admitted.",
		},
	}
	body["errors"] = []any{map[string]any{
		"code": "AWAITING_MEMBER_GRANTS", "stage": "lease",
		"message": "Serving lease has no member grants yet.", "observed_at": "2026-10-28T15:00:30Z",
	}}
	example := mustMarshal(t, body)
	if err := validateComponent(t, "ObservationReport", example); err != nil {
		t.Fatal(err)
	}
	var report api.ObservationReport
	if err := json.Unmarshal(example, &report); err != nil {
		t.Fatal(err)
	}
	if report.SafetyStatus != api.SafetyStatusPassed || len(report.ForwardingBlockReasons) != 1 ||
		report.ForwardingBlockReasons[0] != api.ForwardingBlockReasonAwaitingMemberGrants {
		t.Fatalf("provisional forwarding: %+v %s", report.ForwardingBlockReasons, report.SafetyStatus)
	}
	if len(report.Errors) != 1 || report.Errors[0].Code != api.ErrorCodeAWAITINGMEMBERGRANTS || report.Errors[0].Stage != "lease" ||
		report.Errors[0].Message != "Serving lease has no member grants yet." {
		t.Fatalf("errors: %+v", report.Errors)
	}
	wantChecks := []struct {
		name   api.NodeCheckName
		detail string
	}{
		{api.NodeCheckNameLease, "Serving lease has no member grants yet."},
		{api.NodeCheckNameQos, "Rate policy could not be applied."},
		{api.NodeCheckNameSourceBinding, "No member binding was admitted."},
	}
	if len(report.Checks) != len(wantChecks) {
		t.Fatalf("checks: %+v", report.Checks)
	}
	for i, want := range wantChecks {
		code, err := report.Checks[i].ErrorCode.Get()
		if err != nil || code != api.ErrorCodeAWAITINGMEMBERGRANTS || report.Checks[i].Name != want.name ||
			report.Checks[i].Status != api.HealthCheckStatusFailed || report.Checks[i].Detail == nil || *report.Checks[i].Detail != want.detail {
			t.Fatalf("check %d: %+v code %s err %v", i, report.Checks[i], code, err)
		}
	}
	assertObservationRoundTrip(t, example)

	nodeRaw := readFile(t, "examples/valid/NodeDetail.json")
	if err := validateComponent(t, "NodeDetail", nodeRaw); err != nil {
		t.Fatal(err)
	}
	var absent api.NodeDetail
	if err := json.Unmarshal(nodeRaw, &absent); err != nil {
		t.Fatal(err)
	}
	if absent.IdentityConflict.IsSpecified() {
		t.Fatal("omitted identity_conflict must stay unspecified")
	}
	var node map[string]any
	if err := json.Unmarshal(nodeRaw, &node); err != nil {
		t.Fatal(err)
	}
	node["identity_conflict"] = nil
	nullNode := mustMarshal(t, node)
	if err := validateComponent(t, "NodeDetail", nullNode); err != nil {
		t.Fatal(err)
	}
	var nulled api.NodeDetail
	if err := json.Unmarshal(nullNode, &nulled); err != nil || !nulled.IdentityConflict.IsNull() {
		t.Fatalf("explicit null: err %v specified %v null %v", err, nulled.IdentityConflict.IsSpecified(), nulled.IdentityConflict.IsNull())
	}
	node["identity_conflict"] = map[string]any{
		"mapping_frozen":   true,
		"conflict_count":   1,
		"last_conflict_at": "2026-10-30T20:28:00Z",
	}
	present := mustMarshal(t, node)
	if err := validateComponent(t, "NodeDetail", present); err != nil {
		t.Fatal(err)
	}
	var frozen api.NodeDetail
	if err := json.Unmarshal(present, &frozen); err != nil {
		t.Fatal(err)
	}
	got, err := frozen.IdentityConflict.Get()
	seen, seenErr := got.LastConflictAt.Get()
	if err != nil || seenErr != nil || !got.MappingFrozen || got.ConflictCount != 1 || seen.IsZero() {
		t.Fatalf("identity_conflict: %+v err %v when %v", got, err, seenErr)
	}
}
