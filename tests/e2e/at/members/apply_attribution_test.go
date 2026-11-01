package members

import (
	"encoding/json"
	"fmt"
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

func siblingRecovery(trace []json.RawMessage, observed json.RawMessage, sent ...json.RawMessage) (map[string]any, bool, bool) {
	var capture siblingObservation
	valid := json.Unmarshal(observed, &capture) == nil && capture.Start > 0 && capture.End > capture.Start && capture.StatsAvailable && capture.Stats.Drops == 0 && capture.Errors == 0
	rows := make([]agent.Measurement, len(trace))
	diagnoses := []applyDiagnosis{}
	failedTotal := 0
	brackets := []recoveryBracket{}
	for i, raw := range trace {
		if json.Unmarshal(raw, &rows[i]) != nil {
			valid = false
		}
		if rows[i].Timing != nil && rows[i].Timing.Operation == "apply" {
			v, e := diagnoseApply(raw)
			if e != nil {
				valid = false
			} else {
				diagnoses = append(diagnoses, v)
				if !rows[i].Timing.Succeeded {
					failedTotal++
				}
			}
		}
	}
	failedInCapture := 0
	for i, m := range rows {
		if m.Timing != nil && m.Timing.Operation == "apply" && !m.Timing.Succeeded && m.BootID.String() == capture.Boot && m.Timing.End >= capture.Start && m.Timing.Start <= capture.End {
			failedInCapture++
		}
		if m.Baseline == nil {
			continue
		}
		b := recoveryBracket{Run: m.RunID.String(), Baseline: m.Sequence, Cause: m.Baseline.Cause, BaselineSucceeded: m.Baseline.Succeeded, Start: m.Baseline.Start, BaselineEnd: m.Baseline.End, Original: trace[i], Coverage: "outside continuous sibling window"}
		// Failed Apply alone is not proof of successful baseline installation.
		// Preserve every writer call, including repeated or failed calls.
		if m.Baseline.Cause == "apply_rollback" {
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
		for offset, n := range rows[i+1:] {
			if n.BootID == m.BootID && n.Timing != nil && n.Timing.Operation == "apply" && n.Timing.Succeeded && n.Timing.Start >= b.BaselineEnd {
				b.RecoveryRun = n.RunID.String()
				b.RecoveryRecord = trace[i+1+offset]
				b.Recovery = n.Sequence
				b.MemberPath = "unavailable; no joined member-path sample"
				for k, following := range rows[i+2+offset:] {
					if following.BootID != m.BootID {
						continue
					}
					original := trace[i+2+offset+k]
					if following.Baseline != nil {
						break
					}
					if b.LiveRecord == nil && following.Source == "live" && following.Timing != nil {
						b.LiveRecord = original
					}
					if following.Heartbeat != nil {
						b.HeartbeatRecord = original
						if following.Heartbeat.Serving {
							b.MemberPath = "serving heartbeat after local live verification; separate from captured egress"
						}
						break
					}
				}
				b.End = n.Timing.End
				b.Duration = b.End - b.Start
				break
			}
		}
		if m.BootID.String() == capture.Boot && b.Start <= capture.End && (b.End == 0 || b.End >= capture.Start) {
			b.Coverage = "complete"
			if b.Start < capture.Start || b.End == 0 || b.End > capture.End {
				b.Coverage = "partial"
				valid = false
			}
			if !b.BaselineSucceeded || b.End <= b.BaselineEnd {
				valid = false
			}
		}
		brackets = append(brackets, b)
	}
	failedBracketed := 0
	for _, m := range rows {
		if m.Timing == nil || m.Timing.Operation != "apply" || m.Timing.Succeeded || m.BootID.String() != capture.Boot || m.Timing.End < capture.Start || m.Timing.Start > capture.End {
			continue
		}
		found := false
		for _, b := range brackets {
			if b.Run == m.RunID.String() && b.Failed == m.Sequence && b.Coverage == "complete" {
				found = true
			}
		}
		if found {
			failedBracketed++
		} else {
			valid = false
		}
	}
	ingress, egress := map[decstr.U64]time.Duration{}, map[decstr.U64]time.Duration{}
	for _, p := range capture.Packets {
		if p.At < capture.Start || p.At > capture.End {
			valid = false
		}
		switch p.Interface {
		case "fvmb0":
			ingress[p.Sequence] = p.At
		case "fvup0":
			egress[p.Sequence] = p.At
		}
	}
	losses, outside, guardLoss := 0, 0, 0
	for seq, at := range ingress {
		if _, ok := egress[seq]; ok {
			for j := range brackets {
				b := &brackets[j]
				if b.Coverage == "complete" && b.End > 0 && at >= b.End && at <= b.End+time.Second {
					b.Recovered = true
					if b.FirstEgress == 0 || egress[seq] < b.FirstEgress {
						b.FirstEgress = egress[seq]
					}
				}
			}
			continue
		}
		losses++
		covered := false
		for j := range brackets {
			b := &brackets[j]
			if b.Coverage != "outside continuous sibling window" && at >= b.Start && (b.End == 0 || at <= b.End) {
				b.Losses++
				covered = true
			}
		}
		inGuard := false
		for _, m := range rows {
			if m.Timing != nil && m.BootID.String() == capture.Boot {
				d := m.Timing.Deadline
				if d.GuardReleased && at >= d.InstallStart && at <= d.ReleaseEnd {
					inGuard = true
				}
			}
		}
		if inGuard {
			guardLoss++
		}
		for j := range brackets {
			b := &brackets[j]
			if b.Coverage != "outside continuous sibling window" && at >= b.Start && (b.End == 0 || at <= b.End) {
				if inGuard {
					b.LossesInGuards++
				} else {
					b.LossesOutsideGuards++
				}
			}
		}
		if !covered && !inGuard {
			outside++
		}
	}
	for _, b := range brackets {
		if b.Coverage != "outside continuous sibling window" {
			valid = valid && b.Recovered
		}
	}

	attempts := len(ingress)
	sendErrors := 0
	if len(sent) > 0 {
		var sender struct {
			Attempts int `json:"attempts"`
			Errors   int `json:"send_errors"`
		}
		if json.Unmarshal(sent[0], &sender) != nil || sender.Attempts < len(ingress) || sender.Attempts < len(egress) {
			valid = false
		} else {
			attempts, sendErrors = sender.Attempts, sender.Errors
		}
	}
	missingIngress := attempts - len(ingress)
	valid = valid && missingIngress == 0 && sendErrors == 0
	// Bracket/guard membership explains location; it never excludes a loss.
	measurementValid := valid && len(ingress) > 0 && len(egress) > 0
	valid = measurementValid && attempts == len(egress)
	return map[string]any{"contract": "provisional D-135 + D-136; every node-wide baseline call; overlapping brackets are not summed; baseline start/end to next successful Apply end are command bounds, not kernel commit timestamps; recovery sampled at 20 ms, first egress within 1 s; D-134 gives no fixed guard ceiling; corrected D-136 ordinary stale lease uses baseline then ACK/renew/Apply/verify/heartbeat, with no finite outage ceiling; verified deny-all candidate only when an Apply error is retained", "apply_records": diagnoses, "failed_apply_total": failedTotal, "failed_apply_in_capture": failedInCapture, "failed_apply_bracketed": failedBracketed, "baseline_transition_count": len(brackets), "measurement_valid": measurementValid, "brackets": brackets, "ingress_packets": len(ingress), "egress_packets": len(egress), "lost_admitted_packets": attempts - len(egress), "node_ingress_without_egress": losses, "sender_attempts": attempts, "send_errors": sendErrors, "not_observed_at_node_ingress": missingIngress, "losses_in_completed_guards": guardLoss, "losses_outside_completed_guards": losses - guardLoss, "losses_outside_baseline_or_completed_guard": outside, "observation": capture, "cause_status": "unproven"}, valid, failedInCapture > 0
}
