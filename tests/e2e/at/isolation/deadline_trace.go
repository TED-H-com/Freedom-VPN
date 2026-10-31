package isolation

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type DeadlineVerdict struct {
	Passed             bool             `json:"passed"`
	Deadline           string           `json:"deadline_mono_ns"`
	LastPositive       string           `json:"last_positive_mono_ns"`
	FirstDrop          string           `json:"first_drop_mono_ns"`
	LastPositiveOffset string           `json:"last_positive_offset_ns"`
	FirstDropOffset    string           `json:"first_drop_offset_ns"`
	Positives          int              `json:"positives"`
	Drops              int              `json:"drops"`
	Errors             []string         `json:"errors"`
	Packets            []map[string]any `json:"packets"`
}

// EvaluateDeadline requires independent ingress and egress observations, complete
// zero-drop capture and every send/read. Missing evidence never proves denial.
func EvaluateDeadline(probes, capture []byte, deadline int64) DeadlineVerdict {
	v := DeadlineVerdict{Deadline: fmt.Sprint(deadline), Errors: []string{}, Packets: []map[string]any{}}
	var ps, cs []map[string]any
	if json.Unmarshal(probes, &ps) != nil || json.Unmarshal(capture, &cs) != nil || deadline <= 0 {
		v.Errors = append(v.Errors, "invalid trace")
		return v
	}
	stamp := func(row map[string]any, key string) int64 {
		m, _ := row[key].(map[string]any)
		s, _ := m["mono_ns"].(string)
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
	}
	var complete bool
	ingress, egress := map[int]int64{}, map[int]int64{}
	for _, r := range cs {
		if r["event"] == "complete" {
			complete = r["statistics_available"] == true && r["drops"] == float64(0)
		}
		id, _ := r["packet_id"].(float64)
		at := stamp(r, "capture_read")
		switch r["interface"] {
		case "fvmb0":
			if ingress[int(id)] == 0 {
				ingress[int(id)] = at
			}
		case "fvup0":
			if egress[int(id)] == 0 {
				egress[int(id)] = at
			}
		}
	}
	if !complete {
		v.Errors = append(v.Errors, "capture incomplete or dropped packets")
	}
	var last, first int64
	for _, r := range ps {
		id, _ := r["packet_id"].(float64)
		in, out := ingress[int(id)], egress[int(id)]
		row := map[string]any{"probe": r, "ingress_mono_ns": fmt.Sprint(in), "egress_mono_ns": fmt.Sprint(out)}
		v.Packets = append(v.Packets, row)
		if in == 0 || stamp(r, "send") == 0 || stamp(r, "read_completion") == 0 {
			v.Errors = append(v.Errors, fmt.Sprintf("packet %d missing send/ingress/read", int(id)))
			continue
		}
		if out > 0 {
			v.Positives++
			if out > last {
				last = out
			}
			if out >= deadline {
				v.Errors = append(v.Errors, fmt.Sprintf("packet %d post-deadline egress", int(id)))
			}
		} else {
			v.Drops++
			if first == 0 || in < first {
				first = in
			}
			if r["result"] != "timeout" {
				v.Errors = append(v.Errors, fmt.Sprintf("packet %d no egress but result=%v", int(id), r["result"]))
			}
		}
	}
	v.LastPositive = fmt.Sprint(last)
	v.FirstDrop = fmt.Sprint(first)
	v.LastPositiveOffset = fmt.Sprint(last - deadline)
	v.FirstDropOffset = fmt.Sprint(first - deadline)
	if last == 0 || last < deadline-101000000 {
		v.Errors = append(v.Errors, "last positive earlier than D-101ms")
	}
	if first == 0 || first > deadline+100000000 {
		v.Errors = append(v.Errors, "first drop later than D+100ms")
	}
	if first != 0 && last >= first {
		v.Errors = append(v.Errors, "later positive after first drop")
	}
	v.Passed = len(v.Errors) == 0
	return v
}
