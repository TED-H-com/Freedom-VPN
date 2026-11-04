package boundaryplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

const history = "../../../../../docs/e2e/at/isolation/history"

func TestAT13_F24_01PlannerReplay(t *testing.T) {
	for _, tc := range []struct {
		name, dir string
		spacing   string
		count     int
	}{
		{"df9dbad7", "follow-up-24-run", "33200000", 0},
		{"328ab9e0", "follow-up-23-run", "6100000", 31},
		{"runner-1104", "follow-up-25-runner-replay", "4000000", 49},
	} {
		t.Run(tc.name, func(t *testing.T) {
			read := func(name string, value any) {
				t.Helper()
				raw, err := os.ReadFile(filepath.Join(history, tc.dir, name))
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(raw, value); err != nil {
					t.Fatal(err)
				}
			}
			var old struct {
				Deadline int64             `json:"deadline_mono_ns,string"`
				Planned  map[string]string `json:"planned_at"`
				Samples  []Sample          `json:"samples"`
				Added    []string          `json:"added_scheduled_mono_ns"`
			}
			read("boundary-schedule.json", &old)
			var capture []struct {
				ID        int    `json:"packet_id"`
				Interface string `json:"interface"`
				Read      struct {
					Mono int64 `json:"mono_ns,string"`
				} `json:"capture_read"`
			}
			read("boundary-calibration-packets.json", &capture)
			// Verify replay inputs really are the checkpoint's captured samples.
			for _, sample := range old.Samples {
				seen := map[string]int64{}
				for _, row := range capture {
					if row.ID == sample.PacketID && seen[row.Interface] == 0 {
						seen[row.Interface] = row.Read.Mono
					}
				}
				if seen["fvmb0"] != sample.Ingress || seen["fvup0"] != sample.Egress || sample.Latency != sample.Ingress-sample.Send || sample.Transit != sample.Egress-sample.Ingress {
					t.Fatal("calibration input differs", sample)
				}
			}
			var verdict struct {
				Packets []struct {
					Probe Send `json:"probe"`
				} `json:"packets"`
			}
			read("deadline-verdict.json", &verdict)
			original := []Send{}
			for _, row := range verdict.Packets {
				if row.Probe.Phase != "dense_boundary" && row.Probe.At > old.Deadline-1000000000 {
					original = append(original, row.Probe)
				}
			}
			p := Plan(old.Deadline, old.Planned, old.Samples, original)
			added := p["added_scheduled_mono_ns"].([]string)
			if p["qualified"] != (tc.count > 0) || p["spacing_ns"] != tc.spacing || len(added) != tc.count {
				t.Fatalf("wrong plan: %+v", p)
			}
			if tc.count > 0 {
				before, _ := json.Marshal(old.Added)
				after, _ := json.Marshal(added)
				if !bytes.Equal(before, after) {
					t.Fatal("added schedule bytes changed")
				}
				t.Logf("added schedule JSON bytes identical; spacing=%s count=%d; diagnostic fields added", tc.spacing, tc.count)
			} else {
				if len(p["candidates"].([]Candidate)) != 14 || p["unqualified_reason"] != "empty added schedule: every candidate excluded by original cadence sends" {
					t.Fatal("empty schedule not explained")
				}
				for _, c := range p["candidates"].([]Candidate) {
					if len(c.ExcludedBy) == 0 {
						t.Fatal("missing exclusion", c)
					}
				}
				t.Log("F24-01: qualified=false; added=[]; 14 candidates with all excluding original sends; no discounts")
			}
			// Every retained candidate obeys the same exclusion and separation rule.
			spacing, _ := strconv.ParseInt(tc.spacing, 10, 64)
			for i, a := range added {
				at, _ := strconv.ParseInt(a, 10, 64)
				for _, old := range original {
					if delta := old.At - at; delta > -spacing && delta < spacing {
						t.Fatal("original cadence collision")
					}
				}
				if i > 0 {
					prev, _ := strconv.ParseInt(added[i-1], 10, 64)
					if at-prev < spacing {
						t.Fatal("dense probes too close")
					}
				}
			}
			if !reflect.DeepEqual(p["cadence_gaps_ns"], []string{"37000000", "63000000"}) {
				t.Fatal("cadence gaps missing")
			}
			if dir := os.Getenv("FVPN_AT_BOUNDARY_REPLAY_DIR"); dir != "" {
				raw, err := json.MarshalIndent(p, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, tc.name+"-plan.json"), append(raw, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestAT13_F24_01MissingCalibration(t *testing.T) {
	p := Plan(10_000_000_000, map[string]string{"mono_ns": "9000000000", "wall": "2026-11-04T00:00:00Z"}, nil, nil)
	raw, err := json.Marshal(p)
	if err != nil || bytes.Contains(raw, []byte(":null")) || p["qualified"] != false || p["unqualified_reason"] != "same-run latency calibration unavailable: fewer than 20 samples" {
		t.Fatalf("missing calibration must be explicit: %s %v", raw, err)
	}
}

func TestD098_Followup24AndRunnerReplayPins(t *testing.T) {
	for _, dir := range []string{"follow-up-24", "follow-up-25-runner-replay"} {
		raw, err := os.ReadFile(filepath.Join(history, dir, "archive.json"))
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Files []struct{ Path, SHA256 string }
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatal(err)
		}
		want := 213
		if dir == "follow-up-25-runner-replay" {
			want = 6
		}
		if len(manifest.Files) != want {
			t.Fatal("archive scope changed")
		}
		for _, file := range manifest.Files {
			raw, err := os.ReadFile(filepath.Join(history, dir, file.Path))
			if err != nil {
				t.Fatal(err)
			}
			hash := sha256.Sum256(raw)
			if hex.EncodeToString(hash[:]) != file.SHA256 {
				t.Fatal("archive changed", file.Path)
			}
		}
	}
}
