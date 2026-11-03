//go:build netlab

package netlab

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

func startReloadTrace(t *testing.T) {
	t.Helper()
	mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "start", "/scratch/at10")
	t.Cleanup(func() {
		// Also retain an aborted Apply, not only successful measurement runs.
		if !scratchExists("/scratch/at10-trace.done") {
			finishReloadTrace(t)
		}
	})
	_, _, _ = dockerExec("rm", "-f", "/scratch/at10-trace.done")
}
func finishReloadTrace(t *testing.T) {
	t.Helper()
	if scratchExists("/scratch/stream-a1.log.pid") {
		out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/trace_capture.py", "finish-stream", "/scratch/stream-a1.log", "/scratch/at10-apply.jsonl")
		if err != nil {
			// Keep the incomplete trace and let the unchanged analyzer fail it.
			t.Errorf("stream end: %v %s%s", err, out, errOut)
		}
	}
	mustExec(t, "python3", "/opt/netlab/scripts/trace_capture.py", "stop", "/scratch/at10")
	mustExec(t, "touch", "/scratch/at10-trace.done")
	for _, name := range []string{"at10-origin.txt", "at10-origin.err", "at10-node.txt", "at10-node.err", "at10-responder.txt", "at10-responder.err", "at10-apply.jsonl", "stream-a1.log", "stream-a1.log.jsonl", "stream-a1.log.trace-finish.json", "stream-a1.log.trace-done"} {
		if scratchExists("/scratch/" + name) {
			evidenceName := name
			if strings.HasSuffix(name, ".log") {
				evidenceName = strings.TrimSuffix(name, ".log") + ".txt"
			}
			save(t, evidenceName, readScratch(t, "/scratch/"+name))
		}
	}
}

// Mandatory acceptance input: a missing trace, sidecar, or verified Apply
// cannot silently fall back to the legacy stream-only verdict.
func requireD120(t *testing.T) {
	t.Helper()
	requireProducerTiming(t)
	out, errOut, err := dockerExec("python3", "/opt/netlab/scripts/reload_analysis.py",
		"--stream", "/scratch/stream-a1.log.jsonl", "--stream-log", "/scratch/stream-a1.log",
		"--apply", "/scratch/at10-apply.jsonl", "--trace-base", "/scratch/at10",
		"--output", "/scratch/at10-analysis.json", "--d120")
	save(t, "d120-verdict.txt", out+errOut)
	if scratchExists("/scratch/at10-analysis.json") {
		save(t, "at10-analysis.json", readScratch(t, "/scratch/at10-analysis.json"))
	}
	t.Log("\n" + out)
	if err != nil {
		t.Fatalf("AT10 FAIL: D-120 verdict or evidence check: %v %s", err, errOut)
	}
}

// The independently sampled lab decorator must be inside the producer's
// executor brackets. Historical D-120 inputs remain usable by the analyzer;
// fresh runs additionally require the ISO-F6-03 producer contract.
func requireProducerTiming(t *testing.T) {
	t.Helper()
	count := 0
	for _, line := range strings.Split(readScratch(t, "/scratch/at10-apply.jsonl"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var row struct {
			Timing  *enforce.TimingEvidence `json:"timing"`
			Batches []struct {
				Kind  string `json:"kind"`
				Start struct {
					Mono time.Duration `json:"mono_ns,string"`
				} `json:"start"`
				End struct {
					Mono time.Duration `json:"mono_ns,string"`
				} `json:"end"`
			} `json:"batches"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		requireApplyTiming(t, row.Timing)
		d := row.Timing.Deadline
		for _, batch := range row.Batches {
			var start, end time.Duration
			switch batch.Kind {
			case "guard_install":
				start, end = d.InstallStart, d.InstalledBy
			case "guard_release":
				start, end = d.ReleaseStart, d.ReleaseEnd
			default:
				continue
			}
			if start > batch.Start.Mono || batch.End.Mono > end {
				t.Fatalf("%s producer does not enclose executor", batch.Kind)
			}
		}
		count++
	}
	if count != 101 {
		t.Fatalf("producer timing samples=%d want=101", count)
	}
	t.Logf("ISO-F6-03: %d Apply/guard/verifier/precision producer snapshots match executor brackets", count)
}

func requireApplyTiming(t *testing.T, v *enforce.TimingEvidence) {
	t.Helper()
	if v == nil || v.Version != 1 || v.Clock != "CLOCK_MONOTONIC" || v.Operation != "apply" || !v.Succeeded || !v.Verified || len(v.Verifications) == 0 {
		t.Fatalf("missing successful producer timing: %+v", v)
	}
	d := v.Deadline
	if d.ApplyStart != v.Start || v.Start <= 0 || d.RenderStart < v.Start || d.InstallStart < d.RenderStart || d.InstalledBy < d.InstallStart || !d.InstallOK || d.ReleaseStart < d.InstalledBy || d.ReleaseEnd < d.ReleaseStart || !d.GuardReleased || v.End < d.ReleaseEnd {
		t.Fatalf("invalid Apply timeline: %+v", v)
	}
	previous := d.ReleaseEnd
	for _, r := range v.Verifications {
		p := r.PrecisionRead
		if r.Start < previous || r.NFTRead.Start < r.Start || r.NFTRead.End < r.NFTRead.Start || !r.NFTRead.OK || p.Start < r.NFTRead.End || p.End < p.Start || !p.ReadOK || !p.Verified || r.TCRead.Start < p.End || r.TCRead.End < r.TCRead.Start || !r.TCRead.OK || r.End < r.TCRead.End || r.End > v.End || !r.Verified || len(r.SHA256) != 64 {
			t.Fatalf("invalid verifier/precision timing: %+v", r)
		}
		previous = r.End
	}
}
