package isolation

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// This explicit offline review combines two independently captured records from
// one live stack. It never changes a blocked interface or peer result.
func TestAT10_ReviewRecoveryWindow(t *testing.T) {
	dir := os.Getenv("FVPN_AT_REVIEW_RECOVERY_DIR")
	if dir == "" {
		t.Skip("set FVPN_AT_REVIEW_RECOVERY_DIR to the completed run")
	}
	read := func(name string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	original := read("AT10.json")
	var rec, peers Record
	if err := json.Unmarshal(original, &rec); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(read("AT12.json"), &peers); err != nil {
		t.Fatal(err)
	}
	rawRecovery := read("provisional-recovery.json")
	var capture struct {
		RunID        string            `json:"run_id"`
		ImageBuild   string            `json:"image_build"`
		SamplerBuild string            `json:"sampler_build"`
		AdmittedAt   string            `json:"forwarding_admitted_at"`
		Samples      []map[string]any  `json:"samples"`
		Result       ProvisionalResult `json:"result"`
	}
	if err := json.Unmarshal(rawRecovery, &capture); err != nil {
		t.Fatal(err)
	}
	if rec.RunID != capture.RunID || rec.RunID != peers.RunID || rec.Build != capture.ImageBuild || rec.Build != peers.Build {
		t.Fatal("recovery and counted control are not the same live run/build")
	}
	result := EvaluateProvisional(capture.Samples)
	if result.Status != StatusPass || !reflect.DeepEqual(result, capture.Result) {
		t.Fatalf("recovery sample evaluation differs: %+v", result)
	}
	admitted := false
	for _, row := range capture.Samples {
		n, _ := row["forward_member_accept"].(float64)
		if row["at"] == capture.AdmittedAt && row["forwarding"] == true && n > 0 && row["lease_generation"] == result.Generation && row["observation_generation"] == result.Generation {
			admitted = true
		}
	}
	if !admitted {
		t.Fatal("no matching-generation admitted sample")
	}
	populated, err := time.Parse(time.RFC3339Nano, result.PopulatedAt)
	if err != nil {
		t.Fatal(err)
	}
	var positive Positive
	for _, c := range peers.Checks {
		if c.Name != "node-management-22" || !c.Positive.Valid() {
			continue
		}
		for _, o := range c.Positive.Observations {
			if o.Name != "heartbeat" {
				continue
			}
			fields := strings.Fields(o.Detail)
			values := map[string]string{}
			for i := 1; i+1 < len(fields); i += 2 {
				values[fields[i]] = fields[i+1]
			}
			ns, e := strconv.ParseInt(values["wall_unix_ns"], 10, 64)
			if e == nil && values["generation"] == result.Generation && values["serving"] == "1" && time.Unix(0, ns).After(populated) {
				positive = c.Positive
			}
		}
	}
	if !positive.Valid() {
		t.Fatal("no counted same-generation control after populated lease")
	}
	archive := filepath.Join(dir, "history/follow-up-6-bootstrap-only")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, raw := range map[string][]byte{"AT10.json": original, "AT10.md": read("AT10.md"), "provisional-recovery.raw.json": rawRecovery} {
		target := filepath.Join(archive, name)
		// Repeated reviews must not replace the original assembled aggregation.
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if err := os.WriteFile(target, raw, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	body, _ := json.Marshal(result)
	found := false
	for i := range rec.Checks {
		c := &rec.Checks[i]
		if c.Name != "provisional lease reports AWAITING_MEMBER_GRANTS" {
			continue
		}
		found = true
		for j := range c.Observations {
			if c.Observations[j].Name == "provisional_window" {
				c.Observations[j].Name = "bootstrap_provisional_window"
			}
		}
		c.Status, c.Owner, c.Positive, c.At = StatusPass, "", positive, capture.AdmittedAt
		c.Actual = result.Detail + "; counted node-management-22 control on this generation after recovery. Initial bootstrap did not enter the condition; the recovery followed the recorded TC_FOREIGN_FILTER node restart."
		c.Observations = append(c.Observations, Observation{Name: "provisional_window", Detail: string(body)}, Observation{Name: "recovery_capture", Detail: "provisional-recovery.json; recovery-action.json; image build=" + capture.ImageBuild + "; read-only sampler build=" + capture.SamplerBuild})
	}
	if !found {
		t.Fatal("no provisional check to review")
	}
	rec.Status = ChecksStatus(rec.Checks)
	rec.Commands = append(rec.Commands, "FVPN_AT_REVIEW_RECOVERY_DIR=<completed-evidence-dir> go test -count=1 -run '^TestAT10_ReviewRecoveryWindow$' ./tests/e2e/at/isolation")
	build, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	rec.Observations = append(rec.Observations, Observation{Name: "recovery_review", Detail: "Offline same-run correlation; original bootstrap-only AT10 preserved under history/follow-up-6-bootstrap-only; evaluator build=" + strings.TrimSpace(string(build))})
	if err := Write(dir, rec); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(rawRecovery, &metadata); err != nil {
		t.Fatal(err)
	}
	metadata["positive_reference"], _ = json.Marshal("AT12 node-management-22, same run and generation, counted response followed by a serving heartbeat after populated lease issuance; copied into AT10 by offline review")
	corrected, _ := json.MarshalIndent(metadata, "", "  ")
	if err := os.WriteFile(filepath.Join(dir, "provisional-recovery.json"), append(corrected, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("provisional proof pass: %d samples, %d ms, generation %s; AT10 remains %s", len(result.SampleTimes), result.ReadyToPopulatedMS, result.Generation, rec.Status)
}
