package isolation

import (
	"encoding/json"
	"testing"
)

func provisionalRows(t *testing.T) []map[string]any {
	t.Helper()
	const fixture = `[
 {"at":"2026-10-31T00:00:00.250Z","desired_generation":"9007199254740993","observation_generation":"9007199254740993","desired_bindings":1,"lease_generation":"9007199254740993","lease_id":"TEST ONLY empty","observed_lease_id":"TEST ONLY empty","lease_members":0,"observation_id":"TEST ONLY ready","observation_at":"2026-10-31T00:00:00Z","provisioning_state":"ready","safety_status":"passed","forwarding":false,"forward_present":true,"forward_member_accept":0,"block_reasons":["awaiting_member_grants"],"checks":[{"name":"lease","error_code":"AWAITING_MEMBER_GRANTS"},{"name":"qos","error_code":"AWAITING_MEMBER_GRANTS"},{"name":"source_binding","error_code":"AWAITING_MEMBER_GRANTS"}],"errors":[{"stage":"lease","code":"AWAITING_MEMBER_GRANTS"}]},
 {"lease_generation":"9007199254740993","lease_members":1,"lease_issued_at":"2026-10-31T00:00:05.021Z"}]
 `
	var original []map[string]any
	if err := json.Unmarshal([]byte(fixture), &original); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	for _, at := range []string{"2026-10-31T00:00:00.250Z", "2026-10-31T00:00:00.500Z", "2026-10-31T00:00:00.750Z"} {
		raw, _ := json.Marshal(original[0])
		var row map[string]any
		_ = json.Unmarshal(raw, &row)
		row["at"] = at
		rows = append(rows, row)
	}
	return append(rows, original[1])
}

func TestAT10_ProvisionalSamplingRequiresShapeAndThreeReads(t *testing.T) {
	rows := provisionalRows(t)
	got := EvaluateProvisional(rows)
	if got.Status != StatusPass || got.ReadyToPopulatedMS != 5021 || len(got.SampleTimes) != 3 || got.Generation != "9007199254740993" {
		t.Fatalf("%+v", got)
	}
	for _, tc := range []struct {
		name   string
		mutate func([]map[string]any)
		want   string
	}{
		{"missing check", func(r []map[string]any) { r[0]["checks"] = []any{} }, StatusFail},
		{"duplicate errors", func(r []map[string]any) {
			r[0]["errors"] = []any{map[string]any{"stage": "lease", "code": "AWAITING_MEMBER_GRANTS"}, map[string]any{"stage": "qos", "code": "AWAITING_MEMBER_GRANTS"}}
		}, StatusFail},
		{"forwarding", func(r []map[string]any) { r[0]["forward_member_accept"] = 1 }, StatusFail},
		{"stale observation", func(r []map[string]any) { r[0]["observation_generation"] = "2" }, StatusBlocked},
		{"populated other generation", func(r []map[string]any) { r[3]["lease_generation"] = "2" }, StatusBlocked},
		{"duplicate read", func(r []map[string]any) { r[1]["at"] = r[0]["at"] }, StatusBlocked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := provisionalRows(t)
			tc.mutate(r)
			if got := EvaluateProvisional(r); got.Status != tc.want {
				t.Fatalf("%+v", got)
			}
		})
	}
}

func TestAT10_PopulatedReadsDoNotMultiplyProvisionalSamples(t *testing.T) {
	rows := provisionalRows(t)
	rows[2]["at"] = "2026-10-31T00:00:05.100Z"
	rows = append(rows, rows[3], rows[3])
	if got := EvaluateProvisional(rows); got.Status != StatusBlocked || len(got.SampleTimes) != 2 {
		t.Fatalf("two reads inside the interval must stay blocked: %+v", got)
	}
}
