// Package boundaryplan plans TEST ONLY D-140 added probes. It does no I/O.
package boundaryplan

import (
	"fmt"
	"slices"
	"sort"
)

type Sample struct {
	PacketID int   `json:"packet_id"`
	Send     int64 `json:"send_mono_ns,string"`
	Ingress  int64 `json:"ingress_mono_ns,string"`
	Egress   int64 `json:"egress_mono_ns,string"`
	Latency  int64 `json:"latency_ns,string"`
	Transit  int64 `json:"transit_ns,string"`
}

type Send struct {
	At    int64  `json:"scheduled_mono_ns,string"`
	Phase string `json:"phase"`
}

type Candidate struct {
	At         int64  `json:"scheduled_mono_ns,string"`
	ExcludedBy []Send `json:"excluded_by_original_sends"`
}

// Plan uses exactly the checkpoint's samples and remaining original sends. It
// preserves the maximum-transit spacing rule: no sample is discounted, even a
// far-from-boundary capture-read outlier. This can make every candidate collide
// with the original cadence (F24-01); that is explicitly unqualified, never a
// silent empty schedule. Retained sampled runs show one such case in nine,
// not a predicted failure rate. Reducing spacing below a real transit could
// increase D-142 rejection; it could never waive captured-order qualification.
// A nonempty plan still needs the unchanged review of every adjacent pair from
// the earlier of last positive and first drop through sweep end. No retries.
func Plan(deadline int64, plannedAt map[string]string, samples []Sample, original []Send) map[string]any {
	samples = append([]Sample{}, samples...)
	original = append([]Send{}, original...)
	sort.Slice(samples, func(i, j int) bool { return samples[i].PacketID < samples[j].PacketID })
	sort.Slice(original, func(i, j int) bool { return original[i].At < original[j].At })
	latencies := make([]int64, 0, len(samples))
	var maxTransit int64
	for _, s := range samples {
		latencies = append(latencies, s.Latency)
		maxTransit = max(maxTransit, s.Transit)
	}
	slices.Sort(latencies)
	var lo, hi, median int64
	if len(latencies) > 0 {
		lo, hi, median = latencies[0], latencies[len(latencies)-1], latencies[len(latencies)/2]
	}
	// Include arrival-latency spread: a fast following packet can catch a slow
	// predecessor even when send spacing exceeds node transit. Round up 0.1ms.
	bound := max(int64(500000), hi-lo+2*maxTransit+100000)
	spacing := ((bound + 99999) / 100000) * 100000
	margin := hi - lo + 3*spacing
	anchor := deadline - 100000000
	first := anchor - ((margin+1000000+spacing-1)/spacing)*spacing
	last := deadline + 100000000 + margin
	gaps := []int64{}
	for i := 1; i < len(original); i++ {
		gaps = append(gaps, original[i].At-original[i-1].At)
	}
	slices.Sort(gaps)
	gapStrings := []string{}
	for _, g := range slices.Compact(gaps) {
		gapStrings = append(gapStrings, fmt.Sprint(g))
	}
	added := []string{}
	candidates := []Candidate{}
	if len(samples) >= 20 {
		for predicted := first; predicted <= last; predicted += spacing {
			c := Candidate{At: predicted - median, ExcludedBy: []Send{}}
			for _, old := range original {
				delta := old.At - c.At
				if delta > -spacing && delta < spacing {
					c.ExcludedBy = append(c.ExcludedBy, old)
				}
			}
			candidates = append(candidates, c)
			if len(c.ExcludedBy) == 0 {
				added = append(added, fmt.Sprint(c.At))
			}
		}
	}
	reason := ""
	if len(samples) < 20 {
		reason = "same-run latency calibration unavailable: fewer than 20 samples"
	} else if len(added) == 0 {
		reason = "empty added schedule: every candidate excluded by original cadence sends"
	}
	return map[string]any{
		"planned_at": plannedAt, "deadline_mono_ns": fmt.Sprint(deadline), "qualified": len(added) > 0,
		"unqualified_reason": reason,
		"scope":              "D-140 added schedule only; same-run original-cadence capture-read latency, not exact kernel timestamps",
		"samples":            samples, "discounted_samples": []Sample{},
		"latency_min_ns": fmt.Sprint(lo), "latency_median_ns": fmt.Sprint(median), "latency_max_ns": fmt.Sprint(hi),
		"latency_spread_ns": fmt.Sprint(hi - lo), "max_transit_ns": fmt.Sprint(maxTransit), "spacing_ns": fmt.Sprint(spacing),
		"spacing_unrounded_ns": fmt.Sprint(bound), "cadence_gaps_ns": gapStrings,
		"cadence_gap_rule": "an interior candidate needs gap >= 2 * spacing; equality permits only the midpoint; actual grid exclusions listed for every candidate",
		"candidates":       candidates, "added_scheduled_mono_ns": added,
		"predicted_first_ingress_ns": fmt.Sprint(first), "predicted_last_ingress_ns": fmt.Sprint(last),
		"grid_anchor_offset_ns":                         "-100000000",
		"decision_tradeoff":                             "one whole-node boundary; spacing protects observed-range separation but cannot guarantee a sample in a roughly 3ms positive window; original D-101ms send remains; every attempt counts",
		"original_boundary_predicted_ingress_offset_ns": fmt.Sprint(-101000000 + median),
		"spacing_reason":                                "max(0.5ms, measured latency spread + twice measured maximum transit + 0.1ms), rounded up to 0.1ms; avoid existing cadence sends within one spacing; verify actual ingress/egress order",
	}
}
