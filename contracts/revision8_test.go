package contracts

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestAPI01_Revision8VersionAndEnforcementReasons(t *testing.T) {
	doc := loadDoc(t)
	// Revision 12 advances info.version to 1.8.0. This test still pins revision 8's decisions.
	if doc.Info.Version != "1.8.0" {
		t.Fatal("API version must be 1.8.0")
	}
	// The twelve verification reasons. Literal list; see node/enforce/reason.go.
	// This test does not read that source.
	reasons := []string{
		"METER_PATH_INVALID",
		"ISOLATION_BASELINE_MISSING",
		"GENERATION_MISMATCH",
		"HOOK_PRIORITY_MISMATCH",
		"COUNTER_ORDER_INVALID",
		"FLOWTABLE_BYPASS",
		"DEVICE_CLASS_MISSING",
		"METER_EPOCH_CHANGE",
		"QOS_CLASS_MISSING",
		"MARK_SEMANTICS_VIOLATED",
		"ISOLATION_POLICY_INVALID",
		"PROBE_UNBOUNDED",
	}
	var enum []string
	for _, v := range doc.Components.Schemas["ErrorCode"].Value.Enum {
		code, ok := v.(string)
		if !ok {
			t.Fatalf("ErrorCode enum value %v is not a string", v)
		}
		enum = append(enum, code)
	}
	for _, reason := range reasons {
		if !contains(enum, reason) {
			t.Fatalf("%s is not an accepted ErrorCode", reason)
		}
		if !api.ErrorCode(reason).Valid() {
			t.Fatalf("generated ErrorCode %s is not valid", reason)
		}
	}
	// The nine additions follow the three enforcement codes already in the enum.
	added := []string{
		"ISOLATION_POLICY_INVALID",
		"GENERATION_MISMATCH",
		"HOOK_PRIORITY_MISMATCH",
		"COUNTER_ORDER_INVALID",
		"FLOWTABLE_BYPASS",
		"DEVICE_CLASS_MISSING",
		"QOS_CLASS_MISSING",
		"MARK_SEMANTICS_VIOLATED",
		"PROBE_UNBOUNDED",
	}
	at := map[string]int{}
	for i, code := range enum {
		at[code] = i
	}
	if _, ok := at["METER_EPOCH_CHANGE"]; !ok || at["METER_EPOCH_CHANGE"]+1 != at[added[0]] {
		t.Fatal("new enforcement codes must follow METER_EPOCH_CHANGE")
	}
	for i := 1; i < len(added); i++ {
		if at[added[i]] != at[added[0]]+i {
			t.Fatalf("enforcement code order: %s", strings.Join(added, ","))
		}
	}
	desc := doc.Components.Schemas["ErrorCode"].Value.Description
	if !strings.Contains(desc, "Node enforcement verification") || !strings.Contains(desc, "browser-facing") {
		t.Fatal("ErrorCode description must name the node enforcement verification group")
	}
}

func TestAPI01_Revision8ObservationReportEnforcementCodes(t *testing.T) {
	// Kept inline. examples_test.go lists the examples directory for request
	// and response coverage; it does not expect one example file per schema.
	raw := readFile(t, "examples/valid/ObservationReport.json")
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	body["checks"] = []any{map[string]any{
		"name":        "qos",
		"status":      "failed",
		"observed_at": "2026-10-28T15:00:30Z",
		"error_code":  "QOS_CLASS_MISSING",
	}}
	body["errors"] = []any{map[string]any{
		"code":        "ISOLATION_POLICY_INVALID",
		"stage":       "apply",
		"message":     "ISOLATION_POLICY_INVALID",
		"observed_at": "2026-10-28T15:00:30Z",
	}}
	example := mustMarshal(t, body)
	if err := validateComponent(t, "ObservationReport", example); err != nil {
		t.Fatal(err)
	}
	var report api.ObservationReport
	if err := json.Unmarshal(example, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Errors) != 1 || report.Errors[0].Code != api.ErrorCode("ISOLATION_POLICY_INVALID") || report.Errors[0].Stage != "apply" {
		t.Fatalf("errors: %+v", report.Errors)
	}
	code, err := report.Checks[0].ErrorCode.Get()
	if len(report.Checks) != 1 || report.Checks[0].Name != api.NodeCheckNameQos || report.Checks[0].Status != api.HealthCheckStatusFailed || err != nil || code != api.ErrorCode("QOS_CLASS_MISSING") {
		t.Fatalf("qos check: %+v code %s err %v", report.Checks, code, err)
	}
	assertObservationRoundTrip(t, example)

	errors := body["errors"].([]any)
	errors[0].(map[string]any)["code"] = "NOT_A_CODE"
	rejected := mustMarshal(t, body)
	if err := validateComponent(t, "ObservationReport", rejected); err == nil {
		t.Fatal("NOT_A_CODE was accepted")
	}
}

func assertObservationRoundTrip(t *testing.T, raw []byte) {
	t.Helper()
	var model api.ObservationReport
	if err := json.Unmarshal(raw, &model); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	want, err := canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := canonjson.Canonicalize(encoded)
	if err != nil || !bytes.Equal(want, got) {
		t.Fatalf("round trip changed JSON\n got %s\nwant %s", got, want)
	}
}
