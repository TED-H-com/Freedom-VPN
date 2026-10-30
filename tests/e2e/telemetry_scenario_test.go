//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"time"
)

func (r *runner) recordTelemetry(ctx context.Context, rec *Record) func() {
	stop, err := (&Stack{lab: r.lab}).StartNodeTelemetry(ctx, 5*time.Second)
	return func() {
		note := recPathNote(rec)
		if err != nil {
			note("node_telemetry", "sampler configuration failed")
			rec.Status = StatusFail
			return
		}
		result := stop()
		raw, _ := json.Marshal(result)
		note("node_telemetry", string(raw))
		note("node_telemetry_scope", "5-second cadence; Docker exec reads node cgroup v2 CPU/memory and namespace /proc FDs/conntrack. CPU percent: 100 equals one core. Includes observer overhead; no process arguments or connection tuples.")
		if result.Errors > 0 || len(result.Samples) < 2 {
			note("node_telemetry_gate", "incomplete resource evidence; owner w3-devstack-e2e")
			rec.Status = StatusFail
		}
	}
}
