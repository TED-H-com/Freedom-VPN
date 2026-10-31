package enforce

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/contracts"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/getkin/kin-openapi/openapi3"
)

func validateTCEvidence(t *testing.T, evidence *api.TCReadbackEvidence) {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(contracts.OpenAPI())
	if err != nil {
		t.Fatal(err)
	}
	var v any
	raw, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	if err := doc.Components.Schemas["TCReadbackEvidence"].Value.VisitJSON(v); err != nil {
		t.Fatal(err)
	}
	var round api.TCReadbackEvidence
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&round); err != nil {
		t.Fatal(err)
	}
	if tcJSON(round) != tcJSON(evidence) {
		t.Fatal("generated Go round trip changed evidence")
	}
	projection := struct {
		Reads     int                    `json:"reads"`
		Objects   int                    `json:"objects"`
		Truncated bool                   `json:"truncated"`
		TC        []api.TCReadAttachment `json:"tc"`
	}{round.ReadsTotal, round.ObjectsTotal, round.Truncated, round.Attachments}
	hash, err := canonjson.HashHex([]byte(tcJSON(projection)))
	if err != nil || hash != round.Sha256 {
		t.Fatalf("contract projection digest mismatch: %s %v", hash, err)
	}
}

func TestD106_ContractVectorsAndObservationRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/vectors/tc-readback-evidence.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors struct {
		Cases []struct {
			Name      string                 `json:"name"`
			Canonical string                 `json:"canonical_projection"`
			Evidence  api.TCReadbackEvidence `json:"evidence"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &vectors); err != nil {
		t.Fatal(err)
	}
	for _, c := range vectors.Cases {
		t.Run(c.Name, func(t *testing.T) {
			// Reconstruct the input read, including the seventeenth omitted row.
			var reads []TCReadEvidence
			if err := json.Unmarshal([]byte(tcJSON(c.Evidence.Attachments)), &reads); err != nil {
				t.Fatal(err)
			}
			for len(reads) < c.Evidence.ReadsTotal {
				reads = append(reads, reads[0])
			}
			ev := ServingEvidence{TC: reads}
			var verdict error
			if c.Name == "rejected" {
				verdict = tcCheck(reads[0], &reads[0].Objects[0])
			}
			ev.retainTC(uint64(c.Evidence.CandidateGeneration), "verify", verdict)
			actual := ev.TCReadbackEvidence()
			validateTCEvidence(t, actual)
			if actual.Sha256 != c.Evidence.Sha256 {
				t.Fatal("netlab digest differs from contract")
			}
			// Only the actual run time differs from a dated vector.
			actual.ReadAt = c.Evidence.ReadAt
			if tcJSON(actual) != tcJSON(c.Evidence) {
				t.Fatalf("vector mismatch:\n%s\n%s", tcJSON(actual), tcJSON(c.Evidence))
			}
			var projection map[string]json.RawMessage
			if err := json.Unmarshal([]byte(c.Canonical), &projection); err != nil {
				t.Fatal(err)
			}
			projection["tc"] = json.RawMessage(tcJSON(actual.Attachments))
			canonical, err := canonjson.Canonicalize([]byte(tcJSON(projection)))
			if err != nil || string(canonical) != c.Canonical {
				t.Fatalf("canonical bytes differ: %v", err)
			}

			raw, err := os.ReadFile("../../contracts/examples/valid/ObservationReport.tc-" + c.Name + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var report api.ObservationReport
			if err := json.Unmarshal(raw, &report); err != nil {
				t.Fatal(err)
			}
			expected, err := report.TcEvidence.Get()
			if err != nil || tcJSON(expected) != tcJSON(actual) {
				t.Fatalf("observation example differs: %v", err)
			}
			report.TcEvidence.Set(*actual)
			body := tcJSON(report)
			var round api.ObservationReport
			if err := json.Unmarshal([]byte(body), &round); err != nil || tcJSON(round) != body {
				t.Fatalf("observation round trip: %v", err)
			}
			t.Logf("netlab = contract = observation %s sha256=%s", c.Name, actual.Sha256)
		})
	}
	if (ServingEvidence{}).TCReadbackEvidence() != nil {
		t.Fatal("invented read for empty evidence")
	}
}

func TestD106_ReadTimeCompleteOutcomeAndImmutableSnapshot(t *testing.T) {
	f := newQuarantineExec(t)
	f.foreign = false
	start := time.Now().UTC()
	ev, err := VerifyServing(context.Background(), f, f.m, f.plan.Assignments)
	end := time.Now().UTC()
	if err != nil {
		t.Fatal(err)
	}
	saved := ev.TCReadbackEvidence()
	if saved == nil || !saved.Verified || !saved.ErrorCode.IsNull() || !saved.Rejected.IsNull() || saved.ReadAt.Before(start) || saved.ReadAt.After(end) || saved.ReadAt.Location() != time.UTC {
		t.Fatalf("bad complete outcome: %+v", saved)
	}
	validateTCEvidence(t, saved)
	expected := tcJSON(saved)
	saved.Attachments[0].Objects[0].Kind = "changed"
	saved.ErrorCode.Set(api.ErrorCodeTCFOREIGNFILTER)
	ev.TC[0].Objects[0].Kind = "changed"
	ev.TCSummary.Generation++
	if tcJSON(ev.TCReadbackEvidence()) != expected {
		t.Fatal("conversion aliases mutable snapshot or refreshes time")
	}

	// Same clean tc, but nft no longer serves. Never promote a clean partial scan.
	f.baseline = true
	ev, err = VerifyServing(context.Background(), f, f.m, f.plan.Assignments)
	if err == nil || ev.TCReadbackEvidence().Verified || !ev.TCReadbackEvidence().Rejected.IsNull() {
		t.Fatalf("clean tc promoted failed nft: %v", err)
	}
	if code, _ := ev.TCReadbackEvidence().ErrorCode.Get(); string(code) != string(ReasonCode(err)) {
		t.Fatal("non-tc verifier reason lost")
	}
	validateTCEvidence(t, ev.TCReadbackEvidence())

	// Unknown executor errors remain nullable and never expose stderr.
	f.fault = "nft_read"
	ev, err = VerifyServing(context.Background(), f, f.m, f.plan.Assignments)
	if err == nil || ev.TCReadbackEvidence().Verified || !ev.TCReadbackEvidence().ErrorCode.IsNull() || strings.Contains(tcJSON(ev.TCReadbackEvidence()), "TEST ONLY") {
		t.Fatal("unknown error invented or leaked")
	}
	validateTCEvidence(t, ev.TCReadbackEvidence())
}

func TestD106_BoundsRedactionAndFailurePhases(t *testing.T) {
	reads := make([]TCReadEvidence, 20)
	for i := range reads {
		reads[i] = TCReadEvidence{Interface: "fv-test", Attachment: fmt.Sprintf("1:%d", i)}
		for range 6 {
			reads[i].Objects = append(reads[i].Objects, TCObject{Kind: "flower", Handle: "https://example.invalid/private", Parent: "gh" + "p_test", Actions: []string{"mirred", "", "192.0.2.1", "sk_" + "test", "extra"}})
		}
	}
	ev := ServingEvidence{TC: reads}
	ev.retainTC(^uint64(0), "verify", tcCheck(reads[19], &reads[19].Objects[5]))
	actual := ev.TCReadbackEvidence()
	validateTCEvidence(t, actual)
	rejected, _ := actual.Rejected.Get()
	if len(actual.Attachments) != 16 || len(rejected.Objects) != 1 || !actual.Truncated || actual.ObjectsTotal != 120 || actual.ReadsTotal != 20 {
		t.Fatal("lost bounds/counts")
	}
	for _, forbidden := range []string{"https://", "192.0.2.1", "gh" + "p_", "sk_"} {
		if strings.Contains(tcJSON(actual), forbidden) {
			t.Fatalf("unredacted %q", forbidden)
		}
	}
	original := tcJSON(actual)
	rejected.Objects[0].Actions = nil
	actual.Rejected.Set(rejected)
	if tcJSON(ev.TCReadbackEvidence()) != original {
		t.Fatal("rejection aliases snapshot")
	}
	for _, phase := range []string{"load", "pre_apply"} {
		r := TCReadEvidence{Interface: "fv-test", Attachment: "class", Error: "read_failed"}
		ev := ServingEvidence{TC: []TCReadEvidence{r}}
		ev.retainTC(32, phase, tcCheck(r, nil))
		actual := ev.TCReadbackEvidence()
		validateTCEvidence(t, actual)
		if actual.Verified || actual.Phase != api.TCReadbackEvidencePhase(phase) {
			t.Fatal("partial scan promoted")
		}
	}
	// Defensive sanitization cannot emit empty required tokens or raw status text.
	r := TCReadEvidence{Objects: []TCObject{{Actions: []string{""}}}, Error: "private stderr"}
	ev = ServingEvidence{TC: []TCReadEvidence{r}}
	ev.retainTC(1, "load", errors.New("private stderr"))
	validateTCEvidence(t, ev.TCReadbackEvidence())
}

func TestISO_F6_01_CleanNewGenerationRereadsTC(t *testing.T) {
	f := newQuarantineExec(t)
	f.m.Generation = 31
	rejected, err := Apply(context.Background(), f, f.m, nil)
	requireCheck(t, err, ReasonTCForeignFilter)
	saved := tcJSON(rejected.Evidence.TCReadbackEvidence())
	f.foreign = false
	// A fresh audit sees the clean TC even while nft/quarantine still deny serving.
	ev, err := VerifyServing(context.Background(), f, f.m, f.plan.Assignments)
	if err == nil || ReasonCode(err) == ReasonTCForeignFilter || ev.TCReadbackEvidence().Verified || !ev.TCReadbackEvidence().Rejected.IsNull() {
		t.Fatalf("stale verdict after removal: %v", err)
	}
	auditReason := ReasonCode(err)
	f.m.Generation = 32
	f.ops = nil
	clean, err := Apply(context.Background(), f, f.m, nil)
	if err != nil || !clean.Verified || clean.Evidence.Quarantined || f.marker || !f.up {
		t.Fatalf("clean newer Apply failed: %+v %v", clean, err)
	}
	current := clean.Evidence.TCReadbackEvidence()
	if !current.Verified || current.CandidateGeneration != 32 || !current.ErrorCode.IsNull() || !current.Rejected.IsNull() || !current.ReadAt.After(rejected.Evidence.TCReadbackEvidence().ReadAt) {
		t.Fatalf("stale transport: %+v", current)
	}
	count := 0
	for _, op := range f.ops {
		if op == "tc -j filter show dev fv-fh ingress" {
			count++
		}
	}
	if count < 3 {
		t.Fatalf("ownership, staged and raised-link verification did not reread: %d", count)
	}
	if tcJSON(rejected.Evidence.TCReadbackEvidence()) != saved {
		t.Fatal("recovery rewrote rejection")
	}
	t.Logf("generation 31 foreign -> removed, blocked audit=%s -> Apply 32 verified, ingress reads=%d", auditReason, count)
}
