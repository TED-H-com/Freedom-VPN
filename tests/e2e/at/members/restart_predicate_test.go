package members

import (
	"slices"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/google/uuid"
)

type restartLease struct {
	ID           uuid.UUID `json:"lease_id"`
	IssuedAt     time.Time `json:"issued_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	Generation   string    `json:"generation"`
	MemberGrants int       `json:"member_grants"`
}
type restartObservation struct {
	ReceivedAt time.Time             `json:"received_at"`
	Report     api.ObservationReport `json:"report"`
}
type restartAssessment struct {
	ReportCount             int              `json:"post_restart_report_count"`
	Sequence                string           `json:"lease_sequence"`
	SequenceOK              bool             `json:"lease_sequence_valid"`
	D067Status              string           `json:"d067_status"`
	GrantlessLeases         int              `json:"grantless_leases"`
	ProvisionalObservations int              `json:"provisional_observations"`
	PopulatedObservations   int              `json:"populated_nonserving_observations"`
	NoForeignFilter         bool             `json:"no_foreign_filter"`
	Observations            []map[string]any `json:"observations"`
}

// Join the report's actual lease ID, never the latest central lease at receipt
// time (a queued observation can arrive after another renewal).
func assessRestart(resumedAt time.Time, generation string, leases []restartLease, observations []restartObservation) restartAssessment {
	out := restartAssessment{D067Status: "blocked", NoForeignFilter: true, Sequence: "unresolved"}
	valid, provisionalValid, populated := generation != "", true, false
	post := []restartLease{}
	for _, lease := range leases {
		if lease.IssuedAt.Before(resumedAt) {
			continue
		}
		post = append(post, lease)
	}
	slices.SortFunc(post, func(a, b restartLease) int { return a.IssuedAt.Compare(b.IssuedAt) })
	for _, lease := range post {
		if lease.Generation != generation || lease.ID == uuid.Nil || lease.MemberGrants < 0 {
			valid = false
		}
		if lease.MemberGrants == 0 {
			out.GrantlessLeases++
			if populated {
				valid = false
			} // neither accepted sequence goes back to grant-less
		} else {
			populated = true
		}
	}
	if populated {
		out.Sequence = "populated from first renewal"
		if out.GrantlessLeases > 0 {
			out.Sequence = "grant-less then populated"
		}
	}
	for _, row := range observations {
		rep := row.Report
		if row.ReceivedAt.Before(resumedAt) || rep.ObservedAt.Before(resumedAt) {
			continue
		}
		out.ReportCount++
		for _, e := range rep.Errors {
			if e.Code == api.ErrorCodeTCFOREIGNFILTER {
				out.NoForeignFilter = false
			}
		}
		for _, c := range rep.Checks {
			code, _ := c.ErrorCode.Get()
			if code == api.ErrorCodeTCFOREIGNFILTER {
				out.NoForeignFilter = false
			}
		}
		if rep.MemberForwardingEnabled {
			continue
		}
		id, idErr := rep.Lease.LeaseId.Get()
		var held *restartLease
		for i := range post {
			if post[i].ID == id {
				if held != nil {
					valid = false
				}
				held = &post[i]
			}
		}
		ok := idErr == nil && held != nil && !held.IssuedAt.After(rep.ObservedAt) && held.ExpiresAt.After(rep.ObservedAt) && held.Generation == generation
		sample := map[string]any{"received_at": row.ReceivedAt, "report": rep, "lease_id": id, "lease_joined": ok}
		if ok && held.MemberGrants == 0 {
			out.ProvisionalObservations++
			match, expired, d067 := provisionalObservation(rep)
			ok = match && !expired
			provisionalValid = provisionalValid && ok
			sample["d067"] = d067
		} else if ok {
			out.PopulatedObservations++
			ok = populatedSafetyObservation(rep)
		}
		sample["valid"] = ok
		sample["held_lease"] = held
		out.Observations = append(out.Observations, sample)
		valid = valid && ok
	}
	if out.GrantlessLeases > 0 {
		out.D067Status = "fail"
		if out.ProvisionalObservations > 0 && provisionalValid {
			out.D067Status = "pass"
		}
	}
	out.NoForeignFilter = out.NoForeignFilter && out.ReportCount > 0
	out.SequenceOK = valid && out.ReportCount > 0 && populated && (out.GrantlessLeases == 0 || out.D067Status == "pass")
	return out
}

// A populated lease does not manufacture a provisional lease error. Require
// the non-serving report's safety cause and its corresponding failed checks.
func populatedSafetyObservation(rep api.ObservationReport) bool {
	if rep.MemberForwardingEnabled || rep.SafetyStatus != api.SafetyStatusFailed || len(rep.ForwardingBlockReasons) != 1 || rep.ForwardingBlockReasons[0] != api.ForwardingBlockReasonSafetyCheckFailed {
		return false
	}
	active := func(code api.ErrorCode, dependent bool) bool {
		for _, e := range rep.Errors {
			if e.Code == code && (e.Stage == "apply" || (e.Stage == "preflight" && code == api.ErrorCodeDEPENDENCYUNAVAILABLE) || (!dependent && e.Stage == "probe")) {
				return true
			}
		}
		return false
	}
	for _, e := range rep.Errors {
		if e.Stage == "lease" || e.Code == api.ErrorCodeAWAITINGMEMBERGRANTS || e.Code == api.ErrorCodeLEASEEXPIRED {
			return false
		}
	}
	seen := map[api.NodeCheckName]bool{}
	failed := false
	for _, c := range rep.Checks {
		if seen[c.Name] {
			return false
		}
		seen[c.Name] = true
		code, _ := c.ErrorCode.Get()
		switch c.Status {
		case api.HealthCheckStatusPassed:
			if code != "" {
				return false
			}
		case api.HealthCheckStatusFailed:
			dependent := c.Name == api.NodeCheckNameLease || c.Name == api.NodeCheckNameQos || c.Name == api.NodeCheckNameSourceBinding
			if code == "" || !active(code, dependent) {
				return false
			}
			failed = true
		default:
			return false
		}
	}
	return failed && seen[api.NodeCheckNameLease] && seen[api.NodeCheckNameQos] && seen[api.NodeCheckNameSourceBinding]
}
