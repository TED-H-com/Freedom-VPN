//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

// NodeTelemetry contains container-wide resource samples, not a hardware
// capacity certification. CPUPercent uses 100% for one fully busy CPU.
type NodeTelemetry struct {
	CadenceSeconds float64               `json:"cadence_seconds"`
	Errors         int                   `json:"errors"`
	Samples        []NodeTelemetrySample `json:"samples"`
}
type NodeTelemetrySample struct {
	At          time.Time `json:"at"`
	CPUUsec     uint64    `json:"cpu_usage_usec"`
	CPUPercent  *float64  `json:"cpu_percent_one_core,omitempty"`
	MemoryBytes uint64    `json:"memory_bytes"`
	OpenFDs     int       `json:"open_fds"`
	Processes   int       `json:"processes"`
	Conntrack   uint64    `json:"conntrack_entries"`
}

// StartNodeTelemetry samples only the enrolled node container. Stop and collect
// with the returned function; it is safe to call more than once. No host /proc
// or product state is read. Cgroup v2 and namespace procfs must be available.
func (s *Stack) StartNodeTelemetry(ctx context.Context, cadence time.Duration) (func() NodeTelemetry, error) {
	if cadence < time.Second || cadence > time.Minute {
		return nil, fmt.Errorf("telemetry cadence must be 1s..1m")
	}
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	result := NodeTelemetry{CadenceSeconds: cadence.Seconds()}
	go func() {
		defer close(done)
		tick := time.NewTicker(cadence)
		defer tick.Stop()
		for {
			probe, stop := context.WithTimeout(ctx, 4*time.Second)
			raw, err := exec.CommandContext(probe, "docker", "exec", s.lab.node, "fvpn-e2e-nodeconfig", "telemetry").Output()
			stop()
			var sample NodeTelemetrySample
			if err == nil && json.Unmarshal(raw, &sample) == nil {
				sample.At = time.Now().UTC()
				if n := len(result.Samples); n > 0 {
					previous := result.Samples[n-1]
					if sample.CPUUsec >= previous.CPUUsec && sample.At.After(previous.At) {
						percent := float64(sample.CPUUsec-previous.CPUUsec) / 1e6 / sample.At.Sub(previous.At).Seconds() * 100
						sample.CPUPercent = &percent
					}
				}
				result.Samples = append(result.Samples, sample)
			} else if ctx.Err() == nil {
				result.Errors++
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
	return func() NodeTelemetry { cancel(); <-done; return result }, nil
}
