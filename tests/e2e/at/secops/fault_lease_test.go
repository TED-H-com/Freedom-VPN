//go:build e2e

package secops

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAT29_FaultLease18(t *testing.T) {
	issued := time.Date(2026, 11, 4, 6, 34, 21, 0, time.UTC)
	old := nodeLease16{ID: "TEST ONLY old", Generation: "14", Issued: issued, Expires: issued.Add(time.Minute), Max: 60, Sent: int64(time.Second), Deadline: int64(61 * time.Second)}
	renewed := old
	renewed.ID = "TEST ONLY renewed"
	renewed.Issued = issued.Add(30 * time.Second)
	renewed.Expires = renewed.Issued.Add(time.Minute)
	renewed.Sent = int64(31 * time.Second)
	renewed.Deadline = int64(91 * time.Second)
	hb := map[string]string{"serving": "1", "policy_expires_mono_ns": decimal18(renewed.Deadline)}
	end := issued.Add(time.Hour)
	d, err := faultDeadline18(renewed, true, hb, int64(35*time.Second), end)
	if err != nil || d != renewed.Deadline || d == old.Deadline {
		t.Fatalf("renewal between first read and stop: D=%d err=%v", d, err)
	}
	if _, err = faultDeadline18(old, true, hb, int64(35*time.Second), end); err == nil {
		t.Fatal("stale before-fault lease accepted")
	}
	for _, tc := range []struct {
		name     string
		lease    nodeLease16
		accepted bool
		now      int64
		hb       map[string]string
	}{
		{"expired", renewed, true, renewed.Deadline, hb},
		{"not_local", renewed, false, int64(35 * time.Second), hb},
		{"heartbeat_mismatch", renewed, true, int64(35 * time.Second), map[string]string{"serving": "1", "policy_expires_mono_ns": "61"}},
		{"nonserving", renewed, true, int64(35 * time.Second), map[string]string{"serving": "0", "policy_expires_mono_ns": decimal18(renewed.Deadline)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := faultDeadline18(tc.lease, tc.accepted, tc.hb, tc.now, end); err == nil {
				t.Fatal("must fail closed")
			}
		})
	}
	over := renewed
	over.Max = 86400
	if _, err = faultDeadline18(over, true, hb, int64(35*time.Second), end); err == nil {
		t.Fatal("over-cap lease accepted")
	}
	over = renewed
	over.Expires = over.Issued.Add(61 * time.Second)
	over.Deadline = over.Sent + int64(61*time.Second)
	if _, err = faultDeadline18(over, true, hb, int64(35*time.Second), end); err == nil {
		t.Fatal("over-cap duration accepted")
	}
	d, err = faultDeadline18(renewed, true, hb, int64(35*time.Second), renewed.Issued.Add(10*time.Second))
	if err != nil || d != int64(41*time.Second) {
		t.Fatalf("earlier grant: %d %v", d, err)
	}
}

func TestAT29_HistoricalStaleLease18(t *testing.T) {
	raw, err := os.ReadFile("../../../../docs/e2e/at/secops/history/follow-up-17/followup17-lease-expiry-finding.json")
	if err != nil {
		t.Fatal(err)
	}
	var evidence struct {
		Details struct {
			Lease     nodeLease16 `json:"recorded_lease"`
			Heartbeat string      `json:"heartbeat_deadline_monotonic_ns"`
		} `json:"details"`
	}
	if json.Unmarshal(raw, &evidence) != nil {
		t.Fatal("historical evidence")
	}
	lease := evidence.Details.Lease
	if lease.Generation != "14" || lease.Issued.Format(time.RFC3339Nano) != "2026-11-04T06:34:21.397505Z" {
		t.Fatal("wrong historical row")
	}
	_, err = faultDeadline18(lease, false, map[string]string{"serving": "1", "policy_expires_mono_ns": evidence.Details.Heartbeat}, 97628453277989, lease.Expires.Add(time.Hour))
	if err == nil || !strings.Contains(err.Error(), "generation=14") || !strings.Contains(err.Error(), "heartbeat_D=97658778490379") {
		t.Fatal("historical stale lease must fail with both observations", err)
	}
	t.Log(err)
}
