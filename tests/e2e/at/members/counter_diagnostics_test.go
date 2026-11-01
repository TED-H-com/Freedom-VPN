//go:build e2e

package members

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/counterread"
	"github.com/TED-H-com/Freedom-VPN/tests/e2e/internal/labstack"
)

func (l *live) recordCounterTiming(r *record) {
	trace, err := l.readEnforcementTiming()
	reads := []counterread.Result{}
	for _, step := range r.Steps {
		if !strings.HasPrefix(step.Name, "persistent withdrawal ") {
			continue
		}
		var v struct {
			Samples []persistentTick `json:"samples"`
		}
		if e := json.Unmarshal([]byte(step.Actual), &v); e != nil {
			l.t.Fatal(e)
		}
		for _, tick := range v.Samples {
			reads = append(reads, tick.Counters)
		}
	}
	joinErr := validateCounterTiming(trace, reads)
	recovery, recoveryOK, observed := siblingRecovery(trace, l.siblingObserved, l.siblingSent)
	recovery["run_id"], recovery["source_commit"] = l.runID, l.build.Source
	recovery["sender"] = l.siblingSent
	evidence, _ := json.MarshalIndent(recovery, "", "  ")
	if e := os.WriteFile(filepath.Join(l.root, "docs/e2e/at/members/F12-AT02-recovery.json"), append(evidence, '\n'), 0644); e != nil {
		l.t.Fatal(e)
	}
	if !observed {
		if recovery["failed_apply_total"].(int) > 0 {
			r.block("F11-03: failed Applies outside the continuous sibling window are retained; their sibling recovery is unmeasured")
		}
		r.block("F11-03 D-135: no failed Apply observed in the continuous sibling window; recovery row blocked, not passed")
	} else {
		r.jsonCheck("F11-03 D-135 admitted sibling recovery", "revoked source dropped; every sibling loss counted; baseline-to-next-successful-Apply bracket reported; forwarding returns after that Apply", map[string]any{"artifact": "F12-AT02-recovery.json", "measurement": recovery}, "same boot/run/sequence producer records and continuous node ingress/egress; diagnostic labels never waive failures", recoveryOK && joinErr == nil && err == nil)
	}

	// A sidecar retains every original operation/renewal/source kind once. The
	// existing shared result serialization retains both diagnoses on every tick.
	artifact := map[string]any{"run_id": l.runID, "source_commit": l.build.Source, "records": trace}
	raw, marshalErr := json.MarshalIndent(artifact, "", "  ")
	if marshalErr != nil {
		l.t.Fatal(marshalErr)
	}
	if e := os.WriteFile(filepath.Join(l.root, "docs/e2e/at/members/AT02-enforcement-timing.json"), append(raw, '\n'), 0644); e != nil {
		l.t.Fatal(e)
	}
	r.jsonCheck("counter reader enforcement attribution", "same-boot nonzero ordered read brackets; every before/after record joins gap-free original producer stream", map[string]any{"trace_file": "AT02-enforcement-timing.json", "trace_records": len(trace), "counter_samples": len(reads), "export_error": safeError(err), "join_error": safeError(joinErr)}, "EnableEnforcementTiming before enrollment; streamed ExportContainerFile after AT02; command brackets, not kernel commit times", err == nil && joinErr == nil)
}

func TestAT02_AT20_SharedDiagnosticSerialization(t *testing.T) {
	// Exercise the same jsonCheck used by the live cases, including a failed
	// counter read and DNS metadata that must never be lost by a local projection.
	var tick persistentTick
	tick.Counters.Diagnostic.Category = "transient_io"
	tick.Counters.Diagnostic.Stage = "nft_read"
	tick.Counters.Diagnostic.MonoStart = 9007199254740993
	tick.Counters.Diagnostic.MonoEnd = 9007199254740995
	tick.Status.ReadDiagnostic.Category = "none"
	tick.Counters.Diagnostic.TimingBefore = counterread.TimingSnapshot{Available: true, Record: json.RawMessage(`{"sequence":"9007199254740993"}`)}
	var dns labstack.FixtureProbeResult
	input := `{"dns_trace":{"responder":"multiple_matching_responses","cache_state":"unknown","trace_complete":true,"authority_serial_after_query":1002,"answer_ttl_remaining":[9],"resolver_events":[],"authority_events":[]}}`
	if err := json.Unmarshal([]byte(input), &dns); err != nil {
		t.Fatal(err)
	}
	r := record{}
	r.jsonCheck("diagnostics", "retain original shared fields", map[string]any{"sample": tick, "query": dns}, "TEST ONLY serialization", false)
	var got struct {
		Sample persistentTick              `json:"sample"`
		Query  labstack.FixtureProbeResult `json:"query"`
	}
	if len(r.Steps) != 1 || r.Steps[0].Status != "fail" || json.Unmarshal([]byte(r.Steps[0].Actual), &got) != nil {
		t.Fatal("failed result lost")
	}
	if got.Sample.ReadOK || got.Sample.Counters.Diagnostic.MonoStart != tick.Counters.Diagnostic.MonoStart || got.Sample.Counters.Diagnostic.Category != "transient_io" || got.Sample.Status.ReadDiagnostic.Category != "none" || string(got.Sample.Counters.Diagnostic.TimingBefore.Record) != string(tick.Counters.Diagnostic.TimingBefore.Record) {
		t.Fatal("reader diagnosis or decimal clock lost")
	}
	before, _ := json.Marshal(dns)
	after, _ := json.Marshal(got.Query)
	if string(before) != string(after) || !strings.Contains(string(after), `"responder":"multiple_matching_responses"`) {
		t.Fatal("DNS trace lost")
	}
}
