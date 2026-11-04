//go:build e2e

package secops

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type nodeLease16 struct {
	Signed     json.RawMessage `json:"signed"`
	Sent       int64           `json:"request_monotonic_ns,string"`
	Deadline   int64           `json:"deadline_monotonic_ns,string"`
	ID         string          `json:"lease_id"`
	Generation string          `json:"generation"`
	Issued     time.Time       `json:"issued_at"`
	Expires    time.Time       `json:"expires_at"`
	Max        int             `json:"max_offline_seconds"`
	Members    []struct {
		ID  string    `json:"member_id"`
		End time.Time `json:"lease_end"`
	} `json:"members"`
}

// shortLeaseFields17 rejects pre-cap issuance even when its duration is short.
// The caller obtains this record only from the node's accepted local store.
func shortLeaseFields17(lease nodeLease16, capChangedAt time.Time) error {
	duration := lease.Expires.Sub(lease.Issued)
	if lease.ID == "" || capChangedAt.IsZero() || !lease.Issued.After(capChangedAt) || lease.Max != 60 || duration <= 0 || duration > 60*time.Second || lease.Sent <= 0 || lease.Deadline != lease.Sent+int64(duration) {
		return fmt.Errorf("lease %s issued %s cap=%ds lifetime=%s: require issuance after %s, cap=60s, lifetime in (0,60s], accepted monotonic deadline", lease.ID, lease.Issued.Format(time.RFC3339Nano), lease.Max, duration, capChangedAt.Format(time.RFC3339Nano))
	}
	return nil
}

func shortLeaseSetup17(lease nodeLease16, capChangedAt time.Time, memberID string, now int64, remaining time.Duration, heartbeat map[string]string) error {
	if err := shortLeaseFields17(lease, capChangedAt); err != nil {
		return err
	}
	member := false
	for _, v := range lease.Members {
		member = member || (v.ID == memberID && !v.End.Before(lease.Expires))
	}
	if !member || lease.Deadline-now <= int64(remaining) || heartbeat["serving"] != "1" || heartbeat["policy_expires_mono_ns"] != strconv.FormatInt(lease.Deadline, 10) {
		return fmt.Errorf("lease %s: member coverage, remaining continuity window or serving heartbeat deadline not qualified", lease.ID)
	}
	return nil
}
