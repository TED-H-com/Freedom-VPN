package members

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

const awaitingGrantsDetail = "Serving lease has no member grants yet."

// provisionalObservation is shared by the live revision-11 assertion and the
// captured-sequence replay. It validates report composition, not recovery or
// safety success. D-126 retains both the safety cause and the lease-stage cause.
func provisionalObservation(rep api.ObservationReport) (bool, bool, map[string]any) {
	hasAwaiting := slices.Contains(rep.ForwardingBlockReasons, api.ForwardingBlockReasonAwaitingMemberGrants)
	hasSafety := slices.Contains(rep.ForwardingBlockReasons, api.ForwardingBlockReasonSafetyCheckFailed)
	expired := slices.Contains(rep.ForwardingBlockReasons, api.ForwardingBlockReasonLeaseExpired)
	safetyCode := func(code api.ErrorCode) bool {
		if !hasSafety || rep.SafetyStatus != api.SafetyStatusFailed || code == "" || code == api.ErrorCodeLEASEEXPIRED {
			return false
		}
		for _, e := range rep.Errors {
			// These are the producer's active enforcement causes. Historical
			// tc_evidence/quarantine rows cannot justify a current override.
			if e.Code == code && (e.Stage == "apply" || (e.Stage == "preflight" && code == api.ErrorCodeDEPENDENCYUNAVAILABLE)) {
				return true
			}
		}
		return false
	}
	checkCodeOK := func(name api.NodeCheckName) (bool, string, string) {
		for _, c := range rep.Checks {
			if c.Name == name {
				code, err := c.ErrorCode.Get()
				detail := ""
				if c.Detail != nil {
					detail = *c.Detail
				}
				ok := code == api.ErrorCodeAWAITINGMEMBERGRANTS || safetyCode(code)
				// Only a provisional lease check has the provisional detail.
				// An overriding safety check need not repeat that sentence;
				// the independent lease-stage error must still retain it.
				if name == api.NodeCheckNameLease && code == api.ErrorCodeAWAITINGMEMBERGRANTS {
					ok = ok && detail == awaitingGrantsDetail
				}
				return c.Status == api.HealthCheckStatusFailed && err == nil && ok, string(code), detail
			}
		}
		return false, "", ""
	}
	leaseOK, leaseCode, leaseDetail := checkCodeOK(api.NodeCheckNameLease)
	qosOK, qosCode, _ := checkCodeOK(api.NodeCheckNameQos)
	bindOK, bindCode, _ := checkCodeOK(api.NodeCheckNameSourceBinding)
	leaseErrors := 0
	leaseErrorOK := false
	for _, e := range rep.Errors {
		if e.Stage == "lease" {
			leaseErrors++
			leaseErrorOK = e.Code == api.ErrorCodeAWAITINGMEMBERGRANTS && e.Message == awaitingGrantsDetail
		}
	}
	ok := hasAwaiting && !expired && leaseOK && qosOK && bindOK && leaseErrors == 1 && leaseErrorOK
	return ok, expired, map[string]any{
		"applied_config_generation": rep.AppliedConfigGeneration,
		"safety_status":             rep.SafetyStatus,
		"forwarding_block_reasons":  rep.ForwardingBlockReasons,
		"lease_error_code":          leaseCode,
		"lease_detail":              leaseDetail,
		"qos_error_code":            qosCode,
		"source_binding_error_code": bindCode,
		"errors_count":              len(rep.Errors),
		"lease_stage_errors_count":  leaseErrors,
		"errors":                    rep.Errors,
		"matches_d067_revision11":   ok,
	}
}

func TestAT09_D126CapturedProvisionalSequence(t *testing.T) {
	raw, err := os.ReadFile("../../../../docs/e2e/at/members/history/follow-up-9/AT09-restart.json")
	if err != nil {
		t.Fatal(err)
	}
	var capture struct {
		Capture struct {
			Observations []struct {
				Report json.RawMessage `json:"report"`
			} `json:"observations"`
		} `json:"capture"`
	}
	if err := json.Unmarshal(raw, &capture); err != nil {
		t.Fatal(err)
	}
	provisional, mixed, serving := 0, 0, 0
	var mixedRaw json.RawMessage
	for _, row := range capture.Capture.Observations {
		var rep api.ObservationReport
		if err := json.Unmarshal(row.Report, &rep); err != nil {
			t.Fatal(err)
		}
		before, _ := json.Marshal(rep)
		if rep.MemberForwardingEnabled {
			serving++
			continue // Same selection as the live post-restart SQL query.
		}
		provisional++
		ok, expired, _ := provisionalObservation(rep)
		if !ok || expired {
			t.Fatalf("captured provisional report %s rejected", rep.ReportId)
		}
		for _, c := range rep.Checks {
			code, _ := c.ErrorCode.Get()
			if c.Name == api.NodeCheckNameLease && code == api.ErrorCodeTCFOREIGNFILTER {
				mixed++
				mixedRaw = row.Report
			}
		}
		after, _ := json.Marshal(rep)
		beforeJCS, beforeErr := canonjson.Canonicalize(before)
		afterJCS, afterErr := canonjson.Canonicalize(after)
		if beforeErr != nil || afterErr != nil || !bytes.Equal(beforeJCS, afterJCS) {
			t.Fatal("predicate rewrote report values")
		}
		// D-128: whitespace/key order are not a storage contract. Replay a
		// canonical value too; do not compare JSONB output to raw wire bytes.
		canonical, err := canonjson.Canonicalize(row.Report)
		if err != nil || json.Unmarshal(canonical, &rep) != nil {
			t.Fatal("canonical replay failed", err)
		}
		if ok, expired, _ := provisionalObservation(rep); !ok || expired {
			t.Fatal("canonical value changed the verdict")
		}
	}
	if provisional != 3 || mixed != 1 || serving != 3 {
		t.Fatalf("sequence: provisional=%d mixed=%d serving=%d", provisional, mixed, serving)
	}
	for _, tc := range []struct {
		name   string
		mutate func(*api.ObservationReport)
	}{
		{"bare safety code without reason", func(r *api.ObservationReport) {
			r.ForwardingBlockReasons = slices.DeleteFunc(r.ForwardingBlockReasons, func(reason api.ForwardingBlockReason) bool {
				return reason == api.ForwardingBlockReasonSafetyCheckFailed
			})
		}},
		{"historical tc alone", func(r *api.ObservationReport) {
			r.Errors = slices.DeleteFunc(r.Errors, func(e api.NodeError) bool { return e.Stage == "apply" })
		}},
		{"lease cause overwritten", func(r *api.ObservationReport) {
			for i := range r.Errors {
				if r.Errors[i].Stage == "lease" {
					r.Errors[i].Code = api.ErrorCodeTCFOREIGNFILTER
				}
			}
		}},
		{"duplicate lease error", func(r *api.ObservationReport) { r.Errors = append(r.Errors, r.Errors[0]) }},
		{"expired provisional", func(r *api.ObservationReport) {
			r.ForwardingBlockReasons = append(r.ForwardingBlockReasons, api.ForwardingBlockReasonLeaseExpired)
		}},
		{"passed dependent check", func(r *api.ObservationReport) {
			for i := range r.Checks {
				if r.Checks[i].Name == api.NodeCheckNameQos {
					r.Checks[i].Status = api.HealthCheckStatusPassed
				}
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rep api.ObservationReport
			if err := json.Unmarshal(mixedRaw, &rep); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&rep)
			if ok, _, _ := provisionalObservation(rep); ok {
				t.Fatal("invalid provisional report accepted")
			}
		})
	}
	t.Log("replayed 3 provisional reports (1 mixed) and retained 3 serving reports; historical tc evidence unchanged")
}
