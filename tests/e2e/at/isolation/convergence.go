package isolation

import (
	"encoding/json"
	"fmt"
	"time"
)

// ProvisionalResult keeps the measurement tied to one accepted observation and
// the same desired generation, rather than first-ever node readiness.
type ProvisionalResult struct {
	Status             string   `json:"status"`
	Detail             string   `json:"detail"`
	Generation         string   `json:"generation,omitempty"`
	ObservationID      string   `json:"observation_id,omitempty"`
	LeaseID            string   `json:"provisional_lease_id,omitempty"`
	SampleTimes        []string `json:"sample_times,omitempty"`
	ReadyAt            string   `json:"ready_at,omitempty"`
	PopulatedAt        string   `json:"populated_at,omitempty"`
	ReadyToPopulatedMS int64    `json:"ready_to_populated_ms"`
}

type convergenceSample struct {
	At              string   `json:"at"`
	Desired         string   `json:"desired_generation"`
	Observed        string   `json:"observation_generation"`
	Bindings        int      `json:"desired_bindings"`
	LeaseGeneration string   `json:"lease_generation"`
	LeaseID         string   `json:"lease_id"`
	ObservedLeaseID string   `json:"observed_lease_id"`
	Members         int      `json:"lease_members"`
	IssuedAt        string   `json:"lease_issued_at"`
	ObservationID   string   `json:"observation_id"`
	ObservedAt      string   `json:"observation_at"`
	Provisioning    string   `json:"provisioning_state"`
	Safety          string   `json:"safety_status"`
	Forwarding      bool     `json:"forwarding"`
	ForwardPresent  bool     `json:"forward_present"`
	ForwardAccept   int      `json:"forward_member_accept"`
	Reasons         []string `json:"block_reasons"`
	Checks          []struct {
		Name, Status string
		Code         string `json:"error_code"`
	} `json:"checks"`
	Errors []struct{ Stage, Code string } `json:"errors"`
}

// EvaluateProvisional requires three separate reads of the report shape,
// withheld kernel forwarding, and a later populated lease on that generation.
func EvaluateProvisional(rows []map[string]any) ProvisionalResult {
	r := ProvisionalResult{Status: StatusBlocked, Detail: "fewer than three coherent provisional samples; 250 ms target cadence against the #250 approximately 5 s window", ReadyToPopulatedMS: -1}
	var samples []convergenceSample
	for _, row := range rows {
		raw, _ := json.Marshal(row)
		var s convergenceSample
		if json.Unmarshal(raw, &s) == nil {
			samples = append(samples, s)
		}
	}
	groups := map[string][]convergenceSample{}
	var order []string
	for _, s := range samples {
		if s.Bindings <= 0 || s.Members != 0 || s.Desired == "" || s.LeaseGeneration != s.Desired || s.Observed != s.Desired || s.LeaseID == "" || s.ObservedLeaseID != s.LeaseID || s.Provisioning != "ready" {
			continue
		}
		checks := map[string]bool{}
		for _, c := range s.Checks {
			if c.Code == "AWAITING_MEMBER_GRANTS" {
				checks[c.Name] = true
			}
		}
		reasonsOK := len(s.Reasons) == 1 && s.Reasons[0] == "awaiting_member_grants"
		errorsOK := len(s.Errors) == 1 && s.Errors[0].Stage == "lease" && s.Errors[0].Code == "AWAITING_MEMBER_GRANTS"
		if !checks["lease"] || !checks["qos"] || !checks["source_binding"] || !reasonsOK || !errorsOK || s.Safety != "passed" || s.Forwarding || !s.ForwardPresent || s.ForwardAccept != 0 {
			r.Status, r.Detail = StatusFail, "coherent provisional sample violates checks/errors/withheld forwarding: "+s.At+" observation="+s.ObservationID
			return r
		}
		if _, ok := groups[s.LeaseID]; !ok {
			order = append(order, s.LeaseID)
		}
		g := groups[s.LeaseID]
		if len(g) == 0 || g[len(g)-1].At != s.At {
			groups[s.LeaseID] = append(g, s)
		}
	}
	for _, key := range order {
		g := groups[key]
		if len(g) < 3 {
			continue
		}
		first := g[0]
		ready, err := time.Parse(time.RFC3339Nano, first.ObservedAt)
		if err != nil {
			continue
		}
		for _, s := range samples {
			issued, err := time.Parse(time.RFC3339Nano, s.IssuedAt)
			if err != nil || s.Members <= 0 || s.LeaseGeneration != first.Desired || !issued.After(ready) {
				continue
			}
			r.Status, r.Generation, r.ObservationID, r.LeaseID = StatusPass, first.Desired, first.ObservationID, key
			r.ReadyAt, r.PopulatedAt, r.ReadyToPopulatedMS = first.ObservedAt, s.IssuedAt, issued.Sub(ready).Milliseconds()
			for _, sample := range g {
				at, err := time.Parse(time.RFC3339Nano, sample.At)
				if err == nil && !at.Before(ready) && at.Before(issued) {
					r.SampleTimes = append(r.SampleTimes, sample.At)
				}
			}
			if len(r.SampleTimes) < 3 {
				r.Status = StatusBlocked
				continue
			}
			r.Detail = fmt.Sprintf("%d reads inside one grant-less same-generation window: lease/qos/source_binding AWAITING_MEMBER_GRANTS, exactly one lease error, ready/passed, forwarding withheld; ready to populated lease %d ms (usage follow-up 3: 5021 / 5025 ms)", len(r.SampleTimes), r.ReadyToPopulatedMS)
			return r
		}
	}
	return r
}
