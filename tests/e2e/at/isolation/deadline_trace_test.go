package isolation

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestAT13_D104StrictBoundaryAndMissingEvidence(t *testing.T) {
	const d = int64(10000000000)
	for _, tc := range []struct {
		name          string
		early, late   int64
		ingress, pass bool
	}{
		{"inclusive", -101000000, 100000000, true, true},
		{"one ns too early", -101000001, 100000000, true, false},
		{"one ns too late", -101000000, 100000001, true, false},
		{"unobserved is not denial", -101000000, 0, false, false},
		{"postdeadline positive", 0, 100000000, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stamp := func(n int64) map[string]string { return map[string]string{"mono_ns": fmt.Sprint(n)} }
			probes := []map[string]any{{"packet_id": 1, "send": stamp(d + tc.early - 1000), "read_completion": stamp(d + tc.early + 1000), "result": "ok"}, {"packet_id": 2, "send": stamp(d + tc.late - 1000), "read_completion": stamp(d + tc.late + 80000000), "result": "timeout"}}
			packets := []map[string]any{{"packet_id": 1, "interface": "fvmb0", "capture_read": stamp(d + tc.early - 500)}, {"packet_id": 1, "interface": "fvup0", "capture_read": stamp(d + tc.early)}, {"event": "complete", "statistics_available": true, "drops": 0}}
			if tc.ingress {
				packets = append(packets, map[string]any{"packet_id": 2, "interface": "fvmb0", "capture_read": stamp(d + tc.late)})
			}
			p, _ := json.Marshal(probes)
			c, _ := json.Marshal(packets)
			v := EvaluateDeadline(p, c, d)
			if v.Passed != tc.pass {
				t.Fatalf("%+v", v)
			}
		})
	}
}
