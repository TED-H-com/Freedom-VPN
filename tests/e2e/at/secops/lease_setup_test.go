//go:build e2e

package secops

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAT29_PostCapLeaseSetup17(t *testing.T) {
	// Replay the actual failed setup, including the public cap write time.
	f, err := os.Open("../../../../docs/e2e/at/secops/followup16-transports-live.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	var old struct {
		Cases struct {
			Live struct {
				Lease nodeLease16 `json:"lease_before_fault"`
				Fleet struct {
					Readback struct {
						Updated time.Time `json:"updated_at"`
					} `json:"readback"`
				} `json:"fleet_set_60"`
			} `json:"control_lost_agent_alive"`
		} `json:"transport_cases"`
	}
	if err := json.NewDecoder(f).Decode(&old); err != nil {
		t.Fatal(err)
	}
	lease, capAt := old.Cases.Live.Lease, old.Cases.Live.Fleet.Readback.Updated
	if lease.ID != "f6b7e639-c508-44f6-af68-b25ac251a9d3" || lease.Max != 86400 {
		t.Fatal("historical receipt changed")
	}
	err = shortLeaseFields17(lease, capAt)
	if err == nil || !strings.Contains(err.Error(), lease.ID) || !strings.Contains(err.Error(), "cap=86400s lifetime=24h0m0s") {
		t.Fatalf("historical 86400s lease accepted or unnamed: %v", err)
	}
	t.Logf("run 1bfd5cff still rejected: %v", err)
	lease.Max = 60
	lease.Issued = capAt.Add(time.Second)
	lease.Expires = lease.Issued.Add(time.Minute)
	lease.Deadline = lease.Sent + int64(time.Minute)
	member := lease.Members[len(lease.Members)-1].ID
	hb := map[string]string{"serving": "1", "policy_expires_mono_ns": strconv.FormatInt(lease.Deadline, 10)}
	check := func(l nodeLease16, beat map[string]string) error {
		return shortLeaseSetup17(l, capAt, member, l.Sent+int64(time.Second), 20*time.Second, beat)
	}
	if err := check(lease, hb); err != nil {
		t.Fatal("post-cap accepted short lease:", err)
	}
	for _, name := range []string{"before cap", "at cap", "long lifetime", "missing member", "expired window", "missing stamp", "heartbeat mismatch", "not serving"} {
		t.Run(name, func(t *testing.T) {
			bad := lease
			beat := map[string]string{"serving": hb["serving"], "policy_expires_mono_ns": hb["policy_expires_mono_ns"]}
			switch name {
			case "before cap":
				bad.Issued = capAt.Add(-time.Second)
				bad.Expires = bad.Issued.Add(time.Minute)
			case "at cap":
				bad.Issued = capAt
				bad.Expires = bad.Issued.Add(time.Minute)
			case "long lifetime":
				bad.Expires = bad.Issued.Add(61 * time.Second)
				bad.Deadline = bad.Sent + int64(61*time.Second)
			case "missing member":
				bad.Members = nil
			case "expired window":
				bad.Expires = bad.Issued.Add(10 * time.Second)
				bad.Deadline = bad.Sent + int64(10*time.Second)
				beat["policy_expires_mono_ns"] = strconv.FormatInt(bad.Deadline, 10)
			case "missing stamp":
				bad.Sent = 0
			case "heartbeat mismatch":
				beat["policy_expires_mono_ns"] = "1"
			case "not serving":
				beat["serving"] = "0"
			}
			if err := check(bad, beat); err == nil {
				t.Fatal("invalid setup accepted")
			}
		})
	}
}
