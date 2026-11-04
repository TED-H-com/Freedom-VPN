package members

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/node/enforce"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type applyDiagnosis struct {
	CommandLifecycleStatus    string                     `json:"command_lifecycle_status"`
	Commands                  []enforce.CommandLifecycle `json:"commands,omitempty"`
	CommandLifecycleAvailable bool                       `json:"command_lifecycle_available"`
	Run                       string                     `json:"producer_run_id"`
	Sequence                  decstr.U64                 `json:"sequence"`
	FailureStage              string                     `json:"failure_stage"`
	Verification              string                     `json:"verification_status"`
	Label                     string                     `json:"label"`
	Cause                     string                     `json:"cause_status"`
	Idle                      *float64                   `json:"interval_idle_percent"`
	Host                      *agent.MeasurementHost     `json:"host_context"`
	Original                  json.RawMessage            `json:"original_record"`
}

func diagnoseApply(raw json.RawMessage) (applyDiagnosis, error) {
	var m agent.Measurement
	if err := json.Unmarshal(raw, &m); err != nil {
		return applyDiagnosis{}, err
	}
	if m.Timing == nil || m.Timing.Operation != "apply" {
		return applyDiagnosis{}, fmt.Errorf("not an Apply")
	}
	t := m.Timing
	if err := validateCommandLifecycle(t); err != nil {
		return applyDiagnosis{}, err
	}
	d := t.Deadline
	out := applyDiagnosis{Run: m.RunID.String(), Sequence: m.Sequence, FailureStage: "none", Verification: "missing", Cause: "unproven", Host: m.Host, Original: raw, Label: "Apply"}
	out.CommandLifecycleAvailable = len(t.Commands) > 0
	out.CommandLifecycleStatus = "unavailable"
	if out.CommandLifecycleAvailable {
		out.CommandLifecycleStatus = "available"
		out.Commands = t.Commands
	}
	if len(t.Verifications) > 0 {
		out.Verification = "passed"
		for _, v := range t.Verifications {
			if !v.Verified {
				out.Verification = "failed"
			}
		}
	}
	if !t.Succeeded {
		out.Label = "failed Apply"
		switch {
		case d.InstallStart == 0:
			out.FailureStage = "before_install"
		case !d.InstallOK:
			out.FailureStage = "install"
		case !d.GuardReleased:
			out.FailureStage = "guard_release"
		case out.Verification == "failed":
			out.FailureStage = "verification"
		default:
			out.FailureStage = "after_install_unresolved"
		}
	}
	if h := m.Host; h != nil {
		a, b := h.Before, h.After
		if a.OK && b.OK && a.Start > 0 && a.End >= a.Start && a.End <= t.Start && b.Start >= t.End && b.End >= b.Start && b.Total > a.Total && b.Idle >= a.Idle && b.IOWait >= a.IOWait && b.Idle-a.Idle <= b.Total-a.Total {
			idle := 100 * float64(b.Idle-a.Idle) / float64(b.Total-a.Total)
			out.Idle = &idle
			// Deliberately conservative diagnostic rule: both samples runnable and
			// zero aggregate idle ticks. Low-but-nonzero idle is reported, not labeled.
			if !t.Succeeded && !d.InstallOK && d.InstallBound > 0 && d.RenderStart > 0 && d.InstalledBy-d.RenderStart > d.InstallBound && a.Running > 1 && b.Running > 1 && b.Idle == a.Idle {
				out.Label = "fail-closed install overrun with recorded host contention"
			}
		}
	}
	return out, nil
}

type siblingPacket struct {
	Sequence  decstr.U64    `json:"sequence"`
	At        time.Duration `json:"mono_ns,string"`
	Interface string        `json:"interface"`
}
type siblingObservation struct {
	Boot           string                 `json:"boot_id"`
	Start          time.Duration          `json:"start_ns,string"`
	End            time.Duration          `json:"end_ns,string"`
	Packets        []siblingPacket        `json:"packets"`
	Errors         int                    `json:"read_errors"`
	StatsAvailable bool                   `json:"stats_available"`
	Stats          struct{ Drops uint32 } `json:"capture_stats"`
}
type recoveryBracket struct {
	LiveRecord          json.RawMessage `json:"later_live_verification,omitempty"`
	HeartbeatRecord     json.RawMessage `json:"later_heartbeat,omitempty"`
	MemberPath          string          `json:"member_path_evidence"`
	RecoveryRun         string          `json:"recovery_producer_run_id"`
	RecoveryRecord      json.RawMessage `json:"original_recovery_record"`
	FirstEgress         time.Duration   `json:"first_egress_after_recovery_ns,string"`
	Cause               string          `json:"baseline_cause"`
	BaselineSucceeded   bool            `json:"baseline_succeeded"`
	Coverage            string          `json:"sibling_coverage"`
	LossesInGuards      int             `json:"losses_in_completed_guards"`
	LossesOutsideGuards int             `json:"losses_outside_completed_guards"`
	Original            json.RawMessage `json:"original_baseline_record"`
	Run                 string          `json:"producer_run_id"`
	Failed              decstr.U64      `json:"failed_apply_sequence"`
	Baseline            decstr.U64      `json:"baseline_sequence"`
	Recovery            decstr.U64      `json:"next_successful_apply_sequence"`
	Start               time.Duration   `json:"baseline_start_ns,string"`
	BaselineEnd         time.Duration   `json:"baseline_end_ns,string"`
	End                 time.Duration   `json:"recovery_end_ns,string"`
	Duration            time.Duration   `json:"baseline_to_recovery_ns,string"`
	Losses              int             `json:"admitted_losses"`
	Recovered           bool            `json:"forwarding_observed_after_next_successful_apply"`
}

type recoveryStep struct {
	InstallDuration time.Duration   `json:"install_duration_ns,string"`
	DenyAllDuration time.Duration   `json:"deny_all_after_apply_ns,string"`
	InstallLosses   int             `json:"losses_during_apply"`
	DenyAllLosses   int             `json:"losses_after_deny_all_apply"`
	DownVerifiedAt  time.Duration   `json:"down_verified_at_ns,string,omitempty"`
	Run             string          `json:"producer_run_id"`
	Sequence        decstr.U64      `json:"sequence"`
	Kind            string          `json:"kind"`
	Start           time.Duration   `json:"start_ns,string"`
	End             time.Duration   `json:"end_ns,string"`
	Until           time.Duration   `json:"until_serving_apply_ns,string"`
	Duration        time.Duration   `json:"duration_ns,string"`
	Losses          int             `json:"admitted_losses"`
	Original        json.RawMessage `json:"original_record"`
}
type siblingLoss struct {
	Sequence       decstr.U64    `json:"sequence"`
	Before         time.Duration `json:"send_or_ingress_start_ns,string"`
	After          time.Duration `json:"send_or_ingress_end_ns,string"`
	MissingIngress bool          `json:"not_observed_at_node_ingress"`
	InterfaceDown  bool          `json:"interface_administratively_down"`
	Bracket        int           `json:"bracket_index"`
}

// servingApply requires the verified candidate, its actual populated lease,
// and a written serving heartbeat before any subsequent baseline or Apply.
func servingApply(rows []agent.Measurement, i int) (int, bool) {
	m := rows[i]
	t := m.Timing
	if t == nil || t.Operation != "apply" || !t.Succeeded || !t.Verified {
		return 0, false
	}
	lease := m.Lease
	for j := i - 1; lease == nil && j >= 0; j-- {
		if rows[j].RunID == m.RunID && rows[j].BootID == m.BootID {
			lease = rows[j].Lease
		}
	}
	if lease == nil || lease.Members <= 0 || lease.Generation != t.CandidateGeneration {
		return 0, false
	}
	for j := i + 1; j < len(rows); j++ {
		n := rows[j]
		if n.BootID != m.BootID || n.RunID != m.RunID {
			continue
		}
		if n.Baseline != nil || (n.Timing != nil && n.Timing.Operation == "apply") {
			break
		}
		if h := n.Heartbeat; h != nil {
			return j, h.Written && h.Serving && h.Generation == t.CandidateGeneration && h.Mono >= t.End
		}
	}
	return 0, false
}

func siblingRecovery(trace []json.RawMessage, observed json.RawMessage, sent ...json.RawMessage) (map[string]any, bool, bool) {
	var capture siblingObservation
	valid := json.Unmarshal(observed, &capture) == nil && capture.Start > 0 && capture.End > capture.Start && capture.StatsAvailable && capture.Stats.Drops == 0 && capture.Errors == 0
	rows := make([]agent.Measurement, len(trace))
	diagnoses := []applyDiagnosis{}
	failedTotal, failedInCapture := 0, 0
	lastSequence := map[string]decstr.U64{}
	for i, raw := range trace {
		if json.Unmarshal(raw, &rows[i]) != nil {
			valid = false
		}
		m := rows[i]
		key := m.BootID.String() + "/" + m.RunID.String()
		if m.Sequence != lastSequence[key]+1 {
			valid = false
		}
		lastSequence[key] = m.Sequence
		if m.Timing != nil && m.Timing.Operation == "apply" {
			v, e := diagnoseApply(raw)
			if e != nil {
				valid = false
			} else {
				diagnoses = append(diagnoses, v)
			}
			if !m.Timing.Succeeded {
				failedTotal++
				if m.BootID.String() == capture.Boot && m.Timing.End >= capture.Start && m.Timing.Start < capture.End {
					failedInCapture++
				}
			}
		}
	}
	// Keep every original baseline writer call. Merge overlapping intervals for
	// packet accounting so repeated quarantine calls cannot multiply losses.
	calls := []recoveryBracket{}
	for i, m := range rows {
		if m.Baseline == nil || m.BootID.String() != capture.Boot {
			continue
		}
		b := recoveryBracket{Run: m.RunID.String(), Baseline: m.Sequence, Cause: m.Baseline.Cause, BaselineSucceeded: m.Baseline.Succeeded, Start: m.Baseline.Start, BaselineEnd: m.Baseline.End, Original: trace[i], Coverage: "outside continuous sibling window"}
		if b.Cause == "apply_rollback" || b.Cause == "apply_quarantine" {
			for j := i - 1; j >= 0; j-- {
				n := rows[j]
				if n.RunID == m.RunID && n.BootID == m.BootID && n.Timing != nil && n.Timing.Operation == "apply" {
					if !n.Timing.Succeeded {
						b.Failed = n.Sequence
					}
					break
				}
			}
		}
		for j := i + 1; j < len(rows); j++ {
			n := rows[j]
			if n.BootID != m.BootID || n.Timing == nil || n.Timing.Start < b.BaselineEnd {
				continue
			}
			hb, ok := servingApply(rows, j)
			if !ok {
				continue
			}
			b.RecoveryRun, b.RecoveryRecord, b.Recovery = n.RunID.String(), trace[j], n.Sequence
			b.HeartbeatRecord = trace[hb]
			b.MemberPath = "verified Apply with populated lease and written serving heartbeat; captured sibling egress checked separately"
			b.End, b.Duration = n.Timing.End, n.Timing.End-b.Start
			break
		}
		if b.Start < capture.End && (b.End == 0 || b.End >= capture.Start) {
			b.Coverage = "complete"
			if b.Start < capture.Start || b.End == 0 || b.End > capture.End {
				b.Coverage = "partial"
				valid = false
			}
			if b.Start <= 0 || b.BaselineEnd < b.Start {
				valid = false
			}
		}
		calls = append(calls, b)
	}
	failedBracketed := 0
	for _, m := range rows {
		if m.Timing == nil || m.Timing.Operation != "apply" || m.Timing.Succeeded || m.BootID.String() != capture.Boot || m.Timing.End < capture.Start || m.Timing.Start >= capture.End {
			continue
		}
		for _, b := range calls {
			if b.Run == m.RunID.String() && b.Failed == m.Sequence && b.Coverage == "complete" {
				failedBracketed++
				break
			}
		}
	}
	sort.SliceStable(calls, func(i, j int) bool { return calls[i].Start < calls[j].Start })
	brackets := []recoveryBracket{}
	for _, b := range calls {
		if b.Coverage == "outside continuous sibling window" {
			continue
		}
		if len(brackets) > 0 {
			prev := &brackets[len(brackets)-1]
			if prev.End == 0 || b.Start < prev.End {
				if b.End == 0 || b.End > prev.End {
					prev.End, prev.Recovery, prev.RecoveryRun, prev.RecoveryRecord, prev.HeartbeatRecord = b.End, b.Recovery, b.RecoveryRun, b.RecoveryRecord, b.HeartbeatRecord
				}
				prev.Duration = prev.End - prev.Start
				if b.Coverage != "complete" {
					prev.Coverage = "partial"
				}
				continue
			}
		}
		brackets = append(brackets, b)
	}
	// Successful foreign_tc_quarantine wraps Plane.Quarantine, whose success
	// requires read-back of every member link DOWN. The interval starts with
	// that lowering command; DownVerifiedAt preserves its completion bound,
	// rather than inventing the exact kernel transition time. apply_quarantine alone is
	// only an nft baseline callback, so cannot prove administrative link state.
	down := []recoveryStep{}
	closed := []recoveryStep{}
	for i, m := range rows {
		if m.BootID.String() != capture.Boot {
			continue
		}
		if b := m.Baseline; b != nil && b.Cause == "foreign_tc_quarantine" && b.Succeeded {
			step := recoveryStep{Run: m.RunID.String(), Sequence: m.Sequence, Kind: "verified quarantine: member interface down (command bounds)", Start: b.Start, DownVerifiedAt: b.End, Original: trace[i]}
			for j := i + 1; j < len(rows); j++ {
				n := rows[j]
				if n.BootID == m.BootID && n.Timing != nil && n.Timing.Operation == "apply" && n.Timing.Succeeded && n.Timing.Verified && n.Timing.Start >= b.End {
					step.End = n.Timing.End
					break
				}
			}
			if step.End == 0 {
				step.End = capture.End
			}
			step.Duration = step.End - step.Start
			down = append(down, step)
		}
		if t := m.Timing; t != nil && t.Operation == "apply" && t.Succeeded && t.Verified && m.Lease != nil && m.Lease.Members == 0 {
			step := recoveryStep{Run: m.RunID.String(), Sequence: m.Sequence, Kind: "D-136 verified deny-all candidate Apply", Start: t.Start, End: t.End, Original: trace[i]}
			for _, b := range brackets {
				if t.Start >= b.Start && (b.End == 0 || t.End <= b.End) {
					step.Until = b.End
					step.Duration = b.End - t.Start
					step.InstallDuration = t.End - t.Start
					step.DenyAllDuration = b.End - t.End
					closed = append(closed, step)
					break
				}
			}
		}
	}
	ingress, egress := map[decstr.U64]time.Duration{}, map[decstr.U64]time.Duration{}
	for _, p := range capture.Packets {
		if p.At < capture.Start || p.At >= capture.End {
			valid = false
		}
		switch p.Interface {
		case "fvmb0":
			ingress[p.Sequence] = p.At
		case "fvup0":
			egress[p.Sequence] = p.At
		default:
			valid = false
		}
	}
	attempts, sendErrors := len(ingress), 0
	sends := map[decstr.U64]sendAttempt{}
	hasSender := len(sent) > 0
	if hasSender {
		var sender struct {
			Attempts int           `json:"attempts"`
			Errors   int           `json:"send_errors"`
			Sends    []sendAttempt `json:"sends"`
		}
		if json.Unmarshal(sent[0], &sender) != nil || validateSendTimes(sent[0], observed) != nil {
			valid = false
		}
		attempts, sendErrors = sender.Attempts, sender.Errors
		// Preserve packet facts from old aggregate-only ledgers. Their missing
		// send bounds still make the measurement invalid under today's predicate.
		if len(sender.Sends) == 0 && attempts == len(ingress) {
			for seq, at := range ingress {
				sends[seq] = sendAttempt{Sequence: seq, Before: at, After: at, OK: true}
			}
		}
		for _, a := range sender.Sends {
			sends[a.Sequence] = a
		}
	} else {
		// Legacy synthetic/archived callers without a sender can place ingress
		// losses only. The live path always supplies the complete send ledger.
		for seq, at := range ingress {
			sends[seq] = sendAttempt{Sequence: seq, Before: at, After: at, OK: true}
		}
	}
	for seq := range ingress {
		if _, ok := sends[seq]; !ok {
			valid = false
		}
	}
	for seq := range egress {
		if _, ok := sends[seq]; !ok {
			valid = false
		}
		if _, ok := ingress[seq]; !ok {
			valid = false
		}
	}
	losses := []siblingLoss{}
	missingIngress, nodeLoss, outside, guardLoss, unplaced, downLoss := 0, 0, 0, 0, 0, 0
	for seq, a := range sends {
		in, seenIn := ingress[seq]
		out, seenOut := egress[seq]
		if seenOut {
			if !seenIn || out < in {
				valid = false
			}
			for j := range brackets {
				b := &brackets[j]
				limit := capture.End
				if j+1 < len(brackets) {
					limit = brackets[j+1].Start
				}
				if b.End > 0 && in >= b.End && out < limit {
					b.Recovered = true
					if b.FirstEgress == 0 || out < b.FirstEgress {
						b.FirstEgress = out
					}
				}
			}
			continue
		}
		loss := siblingLoss{Sequence: seq, Before: a.Before, After: a.After, MissingIngress: !seenIn, Bracket: -1}
		if !seenIn {
			missingIngress++
		} else {
			nodeLoss++
		}
		if a.Before < capture.Start || a.After < a.Before || a.After >= capture.End {
			unplaced++
			valid = false
		}
		inGuard := false
		for _, m := range rows {
			if m.BootID.String() != capture.Boot || m.Timing == nil {
				continue
			}
			d := m.Timing.Deadline
			at := in
			if !seenIn {
				at = a.Before
			}
			if d.GuardReleased && at >= d.InstallStart && at < d.ReleaseEnd {
				inGuard = true
			}
		}
		if inGuard {
			guardLoss++
		}
		for _, q := range down {
			if a.Before >= q.Start && a.After < q.End {
				loss.InterfaceDown = true
				break
			}
		}
		if loss.MissingIngress && loss.InterfaceDown {
			downLoss++
		}
		for j := range brackets {
			b := &brackets[j]
			if a.Before >= b.Start && b.End > 0 && a.After < b.End {
				loss.Bracket = j
				b.Losses++
				if inGuard {
					b.LossesInGuards++
				} else {
					b.LossesOutsideGuards++
				}
				break
			}
			// A send call straddling a boundary cannot be placed without guessing.
			if a.Before < b.Start && a.After >= b.Start || b.End > 0 && a.Before < b.End && a.After >= b.End {
				unplaced++
				valid = false
			}
		}
		if loss.Bracket < 0 {
			outside++
		}
		for j := range closed {
			if a.Before >= closed[j].Start && a.After < closed[j].Until {
				closed[j].Losses++
				if a.After < closed[j].End {
					closed[j].InstallLosses++
				} else if a.Before >= closed[j].End {
					closed[j].DenyAllLosses++
				}
			}
		}
		losses = append(losses, loss)
	}
	sort.Slice(losses, func(i, j int) bool { return losses[i].Sequence < losses[j].Sequence })
	recovered := len(brackets) > 0
	for _, b := range brackets {
		recovered = recovered && b.Coverage == "complete" && b.Recovered
	}
	measurementValid := valid && len(ingress) > 0 && len(egress) > 0 && len(sends) == attempts
	accountingOK := measurementValid && outside == 0 && failedBracketed == failedInCapture
	recoveryOK := measurementValid && recovered
	return map[string]any{
		"contract":      "D-135/D-136: first baseline or quarantine through first verified populated-lease Apply with serving heartbeat; every loss counted, no outage ceiling; 100 ms is the install budget only; intervals are command bounds [start,end), not kernel commit timestamps",
		"apply_records": diagnoses, "failed_apply_total": failedTotal, "failed_apply_in_capture": failedInCapture, "failed_apply_bracketed": failedBracketed,
		"baseline_transition_count": len(calls), "baseline_calls": calls, "brackets": brackets, "closed_recovery_steps": closed, "interface_down_intervals": down,
		"measurement_valid": measurementValid, "bracket_accounting_ok": accountingOK, "sibling_recovery_ok": recoveryOK,
		"ingress_packets": len(ingress), "egress_packets": len(egress), "lost_admitted_packets": attempts - len(egress), "losses": losses,
		"node_ingress_without_egress": nodeLoss, "sender_attempts": attempts, "send_errors": sendErrors, "not_observed_at_node_ingress": missingIngress, "not_observed_at_node_ingress_interface_down": downLoss,
		"unplaced_packets": unplaced, "losses_in_completed_guards": guardLoss, "losses_outside_completed_guards": len(losses) - guardLoss, "losses_outside_baseline_or_completed_guard": outside,
		"observation": capture, "cause_status": "unproven", "sender": sent,
	}, accountingOK && recoveryOK, failedInCapture > 0
}
