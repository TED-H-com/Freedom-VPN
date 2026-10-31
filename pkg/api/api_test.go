package api

// This file is hand-written (not generated). It proves that the generated Go
// types accept every valid contract example without unknown members and
// encode them back to semantically identical JSON.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

var registry = map[string]func() any{
	"TCReadbackEvidence":  func() any { return new(TCReadbackEvidence) },
	"AuditEvent":          func() any { return new(AuditEvent) },
	"ResolverDocument":    func() any { return new(ResolverDocument) },
	"MemberGroup":         func() any { return new(MemberGroup) },
	"MemberGroupCreate":   func() any { return new(MemberGroupCreate) },
	"MemberGroupList":     func() any { return new(MemberGroupList) },
	"DnsPolicyEvidence":   func() any { return new(DnsPolicyEvidence) },
	"EgressProbe":         func() any { return new(EgressProbe) },
	"NodeCheck":           func() any { return new(NodeCheck) },
	"ThreatFeedStatus":    func() any { return new(ThreatFeedStatus) },
	"ThreatFeedPayload":   func() any { return new(ThreatFeedPayload) },
	"ResolverDocumentRef": func() any { return new(ResolverDocumentRef) },

	"NodeError":                   func() any { return new(NodeError) },
	"AgentBootstrap":              func() any { return new(AgentBootstrap) },
	"UpdateTarget":                func() any { return new(UpdateTarget) },
	"AuditEventList":              func() any { return new(AuditEventList) },
	"BillingPortalSession":        func() any { return new(BillingPortalSession) },
	"BillingPortalSessionRequest": func() any { return new(BillingPortalSessionRequest) },
	"CertificateRenewRequest":     func() any { return new(CertificateRenewRequest) },
	"CertificateRenewResponse":    func() any { return new(CertificateRenewResponse) },
	"CheckoutSession":             func() any { return new(CheckoutSession) },
	"CheckoutSessionRequest":      func() any { return new(CheckoutSessionRequest) },
	"DesiredStateEnvelope":        func() any { return new(DesiredStateEnvelope) },
	"FleetConfig":                 func() any { return new(FleetConfig) },
	"FleetConfigUpdate":           func() any { return new(FleetConfigUpdate) },
	"Enrollment":                  func() any { return new(Enrollment) },
	"EnrollmentCreated":           func() any { return new(EnrollmentCreated) },
	"EnrollmentCreateRequest":     func() any { return new(EnrollmentCreateRequest) },
	"EnrollRequest":               func() any { return new(EnrollRequest) },
	"EnrollResponse":              func() any { return new(EnrollResponse) },
	"ErrorEnvelope":               func() any { return new(ErrorEnvelope) },
	"GrantCreateRequest":          func() any { return new(GrantCreateRequest) },
	"LeaseRenewalRequest":         func() any { return new(LeaseRenewalRequest) },
	"MemberDetail":                func() any { return new(MemberDetail) },
	"AdminDeviceList":             func() any { return new(AdminDeviceList) },
	"MemberDeviceList":            func() any { return new(MemberDeviceList) },
	"NodeUsageOutbox":             func() any { return new(NodeUsageOutbox) },
	"MemberExitList":              func() any { return new(MemberExitList) },
	"MemberList":                  func() any { return new(MemberList) },
	"MemberPatch":                 func() any { return new(MemberPatch) },
	"MemberSelf":                  func() any { return new(MemberSelf) },
	"MemberUsage":                 func() any { return new(MemberUsage) },
	"NodeDetail":                  func() any { return new(NodeDetail) },
	"NodeIncarnationList":         func() any { return new(NodeIncarnationList) },
	"NodeList":                    func() any { return new(NodeList) },
	"NodePatch":                   func() any { return new(NodePatch) },
	"ObservationAccepted":         func() any { return new(ObservationAccepted) },
	"ObservationReport":           func() any { return new(ObservationReport) },
	"Operation":                   func() any { return new(Operation) },
	"PolicyActivateRequest":       func() any { return new(PolicyActivateRequest) },
	"PolicyDefinitionList":        func() any { return new(PolicyDefinitionList) },
	"PolicyRevision":              func() any { return new(PolicyRevision) },
	"PolicyRevisionCreate":        func() any { return new(PolicyRevisionCreate) },
	"PublicationRequest":          func() any { return new(PublicationRequest) },
	"ReasonRequest":               func() any { return new(ReasonRequest) },
	"Release":                     func() any { return new(Release) },
	"ReleaseList":                 func() any { return new(ReleaseList) },
	"ReleaseManifestEnvelope":     func() any { return new(ReleaseManifestEnvelope) },
	"ReleaseRegisterRequest":      func() any { return new(ReleaseRegisterRequest) },
	"ServingLeaseEnvelope":        func() any { return new(ServingLeaseEnvelope) },
	"UpdateBatch":                 func() any { return new(UpdateBatch) },
	"UpdateBatchAccepted":         func() any { return new(UpdateBatchAccepted) },
	"UpdateBatchCreateRequest":    func() any { return new(UpdateBatchCreateRequest) },
	"UpdateBatchRetryRequest":     func() any { return new(UpdateBatchRetryRequest) },
	"UpdateResult":                func() any { return new(UpdateResult) },
	"UpdateResultAck":             func() any { return new(UpdateResultAck) },
	"UsageBatchRequest":           func() any { return new(UsageBatchRequest) },
	"UsageBatchResult":            func() any { return new(UsageBatchResult) },
	"UsageCheckpointEnvelope":     func() any { return new(UsageCheckpointEnvelope) },
	"UsageEvent":                  func() any { return new(UsageEvent) },
	"WebhookAck":                  func() any { return new(WebhookAck) },
}

const examplesDir = "../../contracts/examples/valid"

func TestGeneratedTypesRoundTripValidExamples(t *testing.T) {
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		comp, _, _ := strings.Cut(strings.TrimSuffix(e.Name(), ".json"), ".")
		t.Run(e.Name(), func(t *testing.T) {
			newV, ok := registry[comp]
			if !ok {
				t.Fatalf("component %s has no entry in the round-trip registry", comp)
			}
			raw, err := os.ReadFile(filepath.Join(examplesDir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			v := newV()
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if err := dec.Decode(v); err != nil {
				t.Fatalf("decode into %T: %v", v, err)
			}
			out, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			want, err := canonjson.Canonicalize(raw)
			if err != nil {
				t.Fatal(err)
			}
			got, err := canonjson.Canonicalize(out)
			if err != nil {
				t.Fatalf("re-encoded JSON is outside the canonical subset: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("round trip changed the document\n got  %s\n want %s", got, want)
			}
		})
	}
}

// TestDecimalFieldsUseDecstr checks that a 2^53+1 byte count survives the
// generated types (no float64 anywhere on the path, §10.4).
func TestDecimalFieldsUseDecstr(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(examplesDir, "UsageEvent.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ev UsageEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Records[1].UploadBytes != decstr.MustParse("9007199254740993") {
		t.Fatalf("upload_bytes = %s", ev.Records[1].UploadBytes)
	}
	if ev.EventSeq != 1234 {
		t.Fatalf("event_seq = %s", ev.EventSeq)
	}
	// A JSON number is rejected by the generated type.
	bad := bytes.Replace(raw, []byte(`"event_seq": "1234"`), []byte(`"event_seq": 1234`), 1)
	if bytes.Equal(bad, raw) {
		t.Fatal("test fixture did not change")
	}
	if err := json.Unmarshal(bad, &ev); err == nil {
		t.Fatal("numeric event_seq accepted")
	}
}

func TestPolicyRevisionCreateDiscriminator(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(examplesDir, "PolicyRevisionCreate.fair-use.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p PolicyRevisionCreate
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	d, err := p.Discriminator()
	if err != nil || d != "fair_use" {
		t.Fatalf("discriminator %q %v", d, err)
	}
	v, err := p.ValueByDiscriminator()
	if err != nil {
		t.Fatal(err)
	}
	fu, ok := v.(FairUsePolicyRevisionCreate)
	if !ok {
		t.Fatalf("got %T", v)
	}
	if fu.Payload.Tiers[0].ThresholdBytes != 0 || fu.Payload.Mode != FairUsePayloadModeShadow {
		t.Fatalf("payload %+v", fu.Payload)
	}
}
