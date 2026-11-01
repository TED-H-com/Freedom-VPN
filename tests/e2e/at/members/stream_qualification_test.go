package members

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestAT04_F13MonitorPastDiagnosticLimit(t *testing.T) {
	const run = "4182dd3d-7f1f-453e-8c1f-265d097c3abc"
	const archive = "follow-up-13-stream/" + run
	verifyArchive(t, archive, run, 2)
	root := "../../../../docs/e2e/at/members/history/" + archive + "/"
	raw, err := os.ReadFile(root + "primary.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	rows, err := parseTimingStream(bytes.NewReader(raw))
	if err != nil || len(rows) == 0 {
		t.Fatal("complete exported stream unavailable", err)
	}
	type identity struct {
		Run      string `json:"run_id"`
		Sequence string `json:"sequence"`
	}
	offsets := map[identity]int{}
	end := 0
	for _, line := range bytes.SplitAfter(raw, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var row identity
		if err = json.Unmarshal(line, &row); err != nil {
			t.Fatal(err)
		}
		end += len(line)
		offsets[row] = end
	}
	raw, err = os.ReadFile(root + "monitor.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Check struct {
			Status string `json:"status"`
			Actual string `json:"actual"`
		} `json:"original_check"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Check.Status != "fail" {
		t.Fatal("original counter-parse failure relabeled")
	}
	var capture struct {
		Samples []struct {
			ReadOK   bool `json:"read_ok"`
			Counters struct {
				Diagnostic struct {
					Before struct {
						Available bool     `json:"available"`
						Record    identity `json:"record"`
					} `json:"enforcement_before"`
					After struct {
						Available bool     `json:"available"`
						Record    identity `json:"record"`
					} `json:"enforcement_after"`
				} `json:"read_diagnostic"`
			} `json:"counters"`
		} `json:"samples"`
	}
	if err = json.Unmarshal([]byte(fixture.Check.Actual), &capture); err != nil {
		t.Fatal(err)
	}
	joined, bad := 0, 0
	for _, sample := range capture.Samples {
		if !sample.ReadOK {
			bad++
		}
		d := sample.Counters.Diagnostic
		if !d.Before.Available || !d.After.Available || offsets[d.Before.Record] <= 1<<20 || offsets[d.After.Record] <= 1<<20 {
			t.Fatal("monitor did not consume the stream beyond 1 MiB")
		}
		joined += 2
	}
	if joined != 392 || bad != 1 {
		t.Fatalf("original joins/failures changed: %d/%d", joined, bad)
	}
}
