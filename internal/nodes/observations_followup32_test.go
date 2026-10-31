package nodes

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// D-126 / §5.1–§5.2: cumulative reasons, per-check failures and stage causes
// are independent evidence. In particular, awaiting grants cannot mask a
// safety failure or replace TC_FOREIGN_FILTER on the three serving checks.
func TestD126_MixedObservationEvidenceUnchanged(t *testing.T) {
	raw, err := os.ReadFile("testdata/d126-mixed-observation.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "e2bb60034ff6ac95d8c30f35ad4a2fff3e824abfe47abde77c0737661408b53f" {
		t.Fatalf("captured report bytes changed: %s", got)
	}
	var report api.ObservationReport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report.ForwardingBlockReasons, []api.ForwardingBlockReason{
		api.ForwardingBlockReasonSafetyCheckFailed, api.ForwardingBlockReasonAwaitingMemberGrants,
	}) {
		t.Fatal("fixture lost its ordered cumulative reasons")
	}
	for _, name := range []api.NodeCheckName{api.NodeCheckNameLease, api.NodeCheckNameQos, api.NodeCheckNameSourceBinding} {
		found := false
		for _, check := range report.Checks {
			if check.Name == name {
				code, err := check.ErrorCode.Get()
				if err != nil || code != api.ErrorCodeTCFOREIGNFILTER || check.Status != api.HealthCheckStatusFailed {
					t.Fatalf("%s check changed: %+v", name, check)
				}
				found = true
			}
		}
		if !found {
			t.Fatalf("missing %s check", name)
		}
	}
	leaseErrors := 0
	for _, e := range report.Errors {
		if e.Stage == "lease" {
			leaseErrors++
			if e.Code != api.ErrorCodeAWAITINGMEMBERGRANTS {
				t.Fatalf("lease stage cause changed: %+v", e)
			}
		}
	}
	if leaseErrors != 1 {
		t.Fatalf("lease stage errors = %d, want 1", leaseErrors)
	}
	if err := validateDNSPolicyEvidence(report); err != nil {
		t.Fatal(err)
	}
	want, err := canonjson.Canonicalize(raw)
	if err != nil {
		t.Fatal(err)
	}
	canonical, payload, _, _, err := marshalObservationReport(report)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := canonjson.Canonicalize(payload)
	if err != nil || !bytes.Equal(want, canonical) || !bytes.Equal(want, stored) {
		t.Fatalf("report values changed during serialization: %v", err)
	}
	for _, plainSafety := range []bool{false, true} {
		candidate := report
		if plainSafety {
			candidate.ForwardingBlockReasons = []api.ForwardingBlockReason{api.ForwardingBlockReasonSafetyCheckFailed}
		}
		safety, health, forwarding, errs := evaluateObservationEvidence(candidate)
		if safety != report.SafetyStatus || health != report.DataHealth || forwarding || !reflect.DeepEqual(errs, report.Errors) {
			t.Fatalf("mixed/plain=%v evidence rewritten: %s/%s/%v %+v", plainSafety, safety, health, forwarding, errs)
		}
		// Use observation time, while the captured lease is still valid, so an
		// expired fixture cannot accidentally be the reason publication fails.
		for _, empty := range []bool{false, true} {
			if cause := publicationForwardingFailure(candidate, report.ObservedAt, empty); cause != "check_failed" {
				t.Fatalf("plain=%v empty=%v publication cause=%q", plainSafety, empty, cause)
			}
		}
	}
	if after, err := canonjson.Marshal(report); err != nil || !bytes.Equal(want, after) {
		t.Fatalf("evaluation mutated the report: %v", err)
	}
}
