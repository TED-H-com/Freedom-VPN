package members

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/agent"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type applyDiagnosis struct {
	Run          string                 `json:"producer_run_id"`
	Sequence     decstr.U64             `json:"sequence"`
	FailureStage string                 `json:"failure_stage"`
	Verification string                 `json:"verification_status"`
	Label        string                 `json:"label"`
	Cause        string                 `json:"cause_status"`
	Idle         *float64               `json:"interval_idle_percent"`
	Host         *agent.MeasurementHost `json:"host_context"`
	Original     json.RawMessage        `json:"original_record"`
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
	d := t.Deadline
	out := applyDiagnosis{Run: m.RunID.String(), Sequence: m.Sequence, FailureStage: "none", Verification: "missing", Cause: "unproven", Host: m.Host, Original: raw, Label: "Apply"}
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
	Run         string        `json:"producer_run_id"`
	Failed      decstr.U64    `json:"failed_apply_sequence"`
	Baseline    decstr.U64    `json:"baseline_sequence"`
	Recovery    decstr.U64    `json:"next_successful_apply_sequence"`
	Start       time.Duration `json:"baseline_start_ns,string"`
	BaselineEnd time.Duration `json:"baseline_end_ns,string"`
	End         time.Duration `json:"recovery_end_ns,string"`
	Duration    time.Duration `json:"baseline_to_recovery_ns,string"`
	Losses      int           `json:"admitted_losses"`
	Recovered   bool          `json:"forwarding_observed_after_next_successful_apply"`
}

func siblingRecovery(trace []json.RawMessage, observed json.RawMessage) (map[string]any, bool, bool) {
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
	for i, m := range rows {
		if m.Timing == nil || m.Timing.Operation != "apply" || m.Timing.Succeeded || m.Timing.End < capture.Start || m.Timing.Start > capture.End {
			continue
		}
		b := recoveryBracket{Run: m.RunID.String(), Failed: m.Sequence}
		for j := i + 1; j < len(rows); j++ {
			n := rows[j]
			if n.RunID != m.RunID || n.BootID != m.BootID {
				continue
			}
			if b.Start == 0 && n.Baseline != nil && n.Baseline.Cause == "apply_rollback" && n.Baseline.Succeeded {
				b.Baseline = n.Sequence
				b.Start = n.Baseline.Start
				b.BaselineEnd = n.Baseline.End
			}
			if n.Timing != nil && n.Timing.Operation == "apply" && n.Timing.Succeeded {
				b.Recovery = n.Sequence
				b.End = n.Timing.End
				b.Duration = b.End - b.Start
				break
			}
		}
		if m.BootID.String() != capture.Boot || b.Start == 0 || b.End <= b.BaselineEnd || b.End > capture.End {
			valid = false
		}
		brackets = append(brackets, b)
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
				if at >= b.End && at <= b.End+time.Second {
					b.Recovered = true
				}
			}
			continue
		}
		losses++
		covered := false
		for j := range brackets {
			b := &brackets[j]
			if at >= b.Start && at <= b.End {
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
		if !covered && !inGuard {
			outside++
		}
	}
	for _, b := range brackets {
		valid = valid && b.Recovered
	}
	// A recorded contention label never enters this acceptance predicate.
	valid = valid && len(ingress) > 0 && len(egress) > 0 && outside == 0
	return map[string]any{"contract": "D-135; baseline start/end to next successful Apply end are command bounds, not kernel commit timestamps; recovery sampled at 20 ms, first egress within 1 s; D-134 gives no fixed guard ceiling", "apply_records": diagnoses, "failed_apply_total": failedTotal, "failed_apply_bracketed": len(brackets), "brackets": brackets, "ingress_packets": len(ingress), "egress_packets": len(egress), "lost_admitted_packets": losses, "losses_in_completed_guards": guardLoss, "losses_outside_baseline_or_completed_guard": outside, "observation": capture, "cause_status": "unproven"}, valid, len(brackets) > 0
}
