//go:build e2e && integration

package secops

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// beginTimed18 admits before any deadline read. Its idempotent end is also
// deferred, so returns and test failures close the section before restoration.
func (p *liveProbe) beginTimed18(name string, row map[string]any) func() {
	if os.Getenv("FVPN_SECOPS_TIMING_JOURNAL") == "" {
		p.t.Fatal("D-160 requires harness timing/admission journal")
	}
	call := func(action string) map[string]any {
		raw, err := exec.CommandContext(p.t.Context(), "python3", "-B", filepath.Join(p.root, "tests/e2e/at/secops/timing_admission.py"), action, name).Output()
		var event map[string]any
		if err != nil || json.Unmarshal(raw, &event) != nil {
			p.t.Fatalf("D-160 %s %s failed", action, name)
		}
		return event
	}
	section := map[string]any{"start": call("begin")}
	row["timed_section"] = section
	ended := false
	return func() {
		if !ended {
			section["end"] = call("end")
			ended = true
		}
	}
}

func (p *liveProbe) installLeaseProbe18() {
	binary := filepath.Join(p.root, ".agent/tmp/secops-leaseprobe18")
	build := exec.CommandContext(p.t.Context(), "go", "build", "-o", binary, "./tests/e2e/at/secops/leaseprobe")
	build.Dir = p.root
	if build.Run() != nil {
		p.t.Fatal("read-only lease probe build")
	}
	p.t.Cleanup(func() { _ = os.Remove(binary) })
	if exec.CommandContext(p.t.Context(), "docker", "cp", binary, livePrefix+"node:/var/lib/vpn-node/e2e/secops-leaseprobe").Run() != nil {
		p.t.Fatal("read-only lease probe install")
	}
}

func (p *liveProbe) faultLease18(row map[string]any, grantEnd time.Time) (time.Time, bool) {
	started := time.Now()
	lease, err := p.nodeLease16()
	heartbeat := p.heartbeatView()
	now := mono16()
	row["lease_at_fault"] = lease
	row["heartbeat_at_fault"] = heartbeat
	// Store large integers only as decimal strings.
	row["fault_lease_read"] = faultRead18(started.UTC(), time.Now().UTC(), now, err == nil)
	d, qualify := faultDeadline18(lease, err == nil, heartbeat, now, grantEnd)
	if qualify != nil {
		row["fault_lease_error"] = qualify.Error()
		p.t.Error(qualify)
		return time.Time{}, false
	}
	row["fault_lease_qualified"] = true
	row["deadline_monotonic_ns"] = decimal18(d)
	// time.Now carries Go's monotonic clock, preserving the existing sampling
	// loop's relative timing while using the locally accepted CLOCK_MONOTONIC D.
	deadline := time.Now().Add(time.Duration(d - mono16()))
	row["earliest_deadline"] = deadline.UTC()
	return deadline, true
}

func faultRead18(start, end time.Time, now int64, accepted bool) map[string]any {
	return map[string]any{"started_at": start, "completed_at": end, "mono_ns": decimal18(now), "locally_accepted": accepted}
}
