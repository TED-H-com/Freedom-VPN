package isolation

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

//go:embed expected-blocked.json
var expectedBlockedJSON []byte

type ExpectedBlock struct {
	Case          string `json:"case"`
	Row           string `json:"row"`
	Reason        string `json:"reason"`
	Owner         string `json:"owner"`
	EstablishedBy string `json:"established_by"`
}

func ExpectedBlocks() []ExpectedBlock {
	var entries []ExpectedBlock
	if json.Unmarshal(expectedBlockedJSON, &entries) != nil {
		return nil
	}
	return entries
}

// ControlReadinessSample captures one post-restoration control and observation
// poll after a control/worker outage.
type ControlReadinessSample struct {
	LoginOK                 bool      `json:"login_ok"`
	FleetConfigStatus       int       `json:"fleet_config_status"`
	QueueAvailable          bool      `json:"queue_available"`
	QueuePending            int       `json:"queue_pending"`
	ObservedAt              time.Time `json:"observed_at"`
	AppliedGeneration       string    `json:"applied_config_generation"`
	MemberForwardingEnabled bool      `json:"member_forwarding_enabled"`
}

// SampleControlReadiness reports whether a post-restoration sample has a live
// admin control plane, a drained node observation retry queue, and an accepted
// current observation matching desiredGeneration with member forwarding on.
func SampleControlReadiness(startedAt time.Time, desiredGeneration string, s ControlReadinessSample) bool {
	return !startedAt.IsZero() &&
		desiredGeneration != "" &&
		s.LoginOK &&
		s.FleetConfigStatus == 200 &&
		s.QueueAvailable &&
		s.QueuePending == 0 &&
		s.ObservedAt.After(startedAt) &&
		s.AppliedGeneration == desiredGeneration &&
		s.MemberForwardingEnabled
}

// ControlObservationReady checks whether dir/control-observation-readiness.json
// proves readiness for runID at or after afterControlStart. When absent, stale,
// or unready, it invokes wait (if non-nil) and re-reads the persisted record.
func ControlObservationReady(dir, runID string, afterControlStart time.Time, wait func() bool) bool {
	check := func() bool {
		raw, err := os.ReadFile(filepath.Join(dir, "control-observation-readiness.json"))
		if err != nil {
			return false
		}
		var v struct {
			RunID     string    `json:"run_id"`
			StartedAt time.Time `json:"started_at"`
			Ready     bool      `json:"ready"`
		}
		if json.Unmarshal(raw, &v) != nil || !v.Ready {
			return false
		}
		if runID != "" && v.RunID != "" && v.RunID != runID {
			return false
		}
		if !afterControlStart.IsZero() && v.StartedAt.Before(afterControlStart) {
			return false
		}
		return true
	}
	if check() {
		return true
	}
	if wait == nil || !wait() {
		return false
	}
	return check()
}

// ScheduledCaseAcceptable maps only the scheduled job's exit. It never changes
// a Record or Check. Real Go test errors remain errors regardless of row status.
func ScheduledCaseAcceptable(r Record) bool {
	if len(r.Checks) == 0 || (r.Status != StatusPass && r.Status != StatusBlocked) {
		return false
	}
	seen := map[string]bool{}
	for _, c := range r.Checks {
		if seen[c.Name] {
			return false
		}
		seen[c.Name] = true
		if c.Status == StatusPass {
			if !c.Positive.Valid() {
				return false
			}
			continue
		}
		if c.Status != StatusBlocked {
			return false
		}
		matched := false
		for _, e := range ExpectedBlocks() {
			if e.Case == r.ID && e.Row == c.Name && e.Owner != "" && e.Reason != "" && e.EstablishedBy != "" {
				matched = true
			}
		}
		if !matched {
			return false
		}
	}
	return true
}
