package members

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread"
)

type producerRecord struct {
	Boot     string     `json:"boot_id"`
	Run      string     `json:"run_id"`
	Sequence decstr.U64 `json:"sequence"`
	Anchor   struct {
		Before time.Duration `json:"mono_before_ns,string"`
		After  time.Duration `json:"mono_after_ns,string"`
	} `json:"anchor"`
	Timing *struct {
		Clock string        `json:"clock"`
		Start time.Duration `json:"start_ns,string"`
		End   time.Duration `json:"end_ns,string"`
	} `json:"timing"`
}

// Validate the original producer stream, including renewal/baseline/heartbeat
// rows. A failed read remains failed independently of diagnostic completeness.
// Only operation clocks are required here; unreached command stages may be zero.
func validateCounterTiming(trace []json.RawMessage, reads []counterread.Result) error {
	if len(trace) == 0 || len(reads) == 0 {
		return fmt.Errorf("missing trace or counter samples")
	}
	last := map[string]decstr.U64{}
	index := map[string]json.RawMessage{}
	key := func(v producerRecord) string { return fmt.Sprintf("%s/%s/%s", v.Boot, v.Run, v.Sequence.String()) }
	for i, raw := range trace {
		if err := validateProducerCommands(raw); err != nil {
			return fmt.Errorf("producer row %d: %w", i, err)
		}
		var v producerRecord
		if json.Unmarshal(raw, &v) != nil || v.Boot == "" || v.Run == "" || v.Sequence == 0 || v.Anchor.Before <= 0 || v.Anchor.After < v.Anchor.Before {
			return fmt.Errorf("producer row %d missing identity or ordered clocks", i)
		}
		run := v.Boot + "/" + v.Run
		if v.Sequence != last[run]+1 {
			return fmt.Errorf("producer row %d sequence gap or duplicate", i)
		}
		last[run] = v.Sequence
		if v.Timing != nil && (v.Timing.Clock != "CLOCK_MONOTONIC" || v.Timing.Start <= 0 || v.Timing.End < v.Timing.Start || v.Timing.End > v.Anchor.Before) {
			return fmt.Errorf("producer row %d invalid operation clocks", i)
		}
		index[key(v)] = raw
	}
	for i, read := range reads {
		d := read.Diagnostic
		if d.BootID == "" || d.MonoStart <= 0 || d.MonoEnd < d.MonoStart {
			return fmt.Errorf("counter sample %d missing same-boot read bracket", i)
		}
		for j, snap := range []counterread.TimingSnapshot{d.TimingBefore, d.TimingAfter} {
			var v producerRecord
			if !snap.Available || json.Unmarshal(snap.Record, &v) != nil || v.Timing == nil || v.Boot != d.BootID {
				return fmt.Errorf("counter sample %d side %d missing same-boot operation", i, j)
			}
			original, ok := index[key(v)]
			var a, b any
			// Preserve decimal strings and all fields, including future source kinds.
			if !ok || json.Unmarshal(original, &a) != nil || json.Unmarshal(snap.Record, &b) != nil || !reflect.DeepEqual(a, b) {
				return fmt.Errorf("counter sample %d side %d does not join original record", i, j)
			}
			if j == 0 && v.Timing.End > d.MonoStart {
				return fmt.Errorf("counter sample %d prior operation ends after read start", i)
			}
		}
	}
	return nil
}

func TestAT02_CounterTimingRejectsUnattributableSamples(t *testing.T) {
	row := json.RawMessage(`{"boot_id":"TEST ONLY boot","run_id":"TEST ONLY run","sequence":"1","anchor":{"mono_before_ns":"30","mono_after_ns":"31"},"timing":{"clock":"CLOCK_MONOTONIC","start_ns":"10","end_ns":"20"}}`)
	read := counterread.Result{Diagnostic: counterread.Diagnostic{BootID: "TEST ONLY boot", MonoStart: 40, MonoEnd: 50, Category: "nft_read", TimingBefore: counterread.TimingSnapshot{Available: true, Record: row}, TimingAfter: counterread.TimingSnapshot{Available: true, Record: row}}}
	if err := validateCounterTiming([]json.RawMessage{row}, []counterread.Result{read}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*counterread.Result)
	}{
		{"zero clock", func(r *counterread.Result) { r.Diagnostic.MonoStart = 0 }},
		{"reversed clock", func(r *counterread.Result) { r.Diagnostic.MonoEnd = 1 }},
		{"wrong boot", func(r *counterread.Result) { r.Diagnostic.BootID = "different" }},
		{"missing record", func(r *counterread.Result) { r.Diagnostic.TimingBefore.Available = false }},
		{"changed record", func(r *counterread.Result) {
			r.Diagnostic.TimingAfter.Record = json.RawMessage(`{"boot_id":"TEST ONLY boot","run_id":"TEST ONLY run","sequence":"2","timing":{}}`)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := read
			tc.change(&r)
			if validateCounterTiming([]json.RawMessage{row}, []counterread.Result{r}) == nil {
				t.Fatal("accepted incomplete attribution")
			}
		})
	}
	if validateCounterTiming([]json.RawMessage{row, row}, []counterread.Result{read}) == nil {
		t.Fatal("accepted duplicate sequence")
	}
	var gap map[string]any
	_ = json.Unmarshal(row, &gap)
	gap["sequence"] = "3"
	gapRow, _ := json.Marshal(gap)
	if validateCounterTiming([]json.RawMessage{row, gapRow}, []counterread.Result{read}) == nil {
		t.Fatal("accepted sequence gap")
	}
}
