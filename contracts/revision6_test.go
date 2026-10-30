package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/dnspublish"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestAPI01_Revision6VersionAndDecisions(t *testing.T) {
	doc := loadDoc(t)
	if doc.Info.Version != "1.2.0" {
		t.Fatal("API version must be 1.2.0")
	}
	if n := len(doc.Paths.Map()); n == 0 {
		t.Fatal("paths missing")
	}
	detail := doc.Components.Schemas["NodeDetail"].Value.AllOf[1].Value
	prop := detail.Properties["dns_policy_evidence"].Value
	if prop == nil || !prop.Nullable || contains(detail.Required, "dns_policy_evidence") || prop.AllOf[0].Ref != "#/components/schemas/DnsPolicyEvidence" {
		t.Fatal("NodeDetail.dns_policy_evidence must be optional nullable DnsPolicyEvidence")
	}
	if !strings.Contains(prop.Description, "not central verification") {
		t.Fatal("node evidence must be described as a report, not central verification")
	}
	result := doc.Components.Schemas["UpdateResult"].Value
	health := result.Properties["candidate_health"].Value
	if health == nil || !health.Nullable || contains(result.Required, "candidate_health") || health.AllOf[0].Ref != "#/components/schemas/UpdateHealth" {
		t.Fatal("candidate_health must be optional nullable UpdateHealth")
	}
	for _, sentence := range []string{"precedes checking", "restored release", "candidate's failure", "previous_release_id"} {
		if !strings.Contains(result.Description, sentence) {
			t.Fatalf("UpdateResult description missing %q", sentence)
		}
	}
	observation := doc.Components.Schemas["ObservationReport"].Value.Properties["update"].Value
	if !strings.Contains(observation.Description, "durable updater state") || !strings.Contains(observation.Description, "generation 0") {
		t.Fatal("observation update description missing D-062e")
	}
	allow := doc.Components.Schemas["ManifestRollback"].Value.Properties["allowed_release_ids"].Value
	if !strings.Contains(allow.Description, "no additional restriction") || !strings.Contains(allow.Description, "fail-closed") {
		t.Fatal("allowed_release_ids description missing D-034e")
	}
	resolver := doc.Components.Schemas["ResolverDocument"].Value.Description
	if !strings.Contains(resolver, "bootstrap_dns is non-empty") {
		t.Fatal("resolver document must require non-empty bootstrap_dns when upstreams is non-empty")
	}
}

// GetNode200JSONResponse encodes NodeDetail. The generated dns_policy_evidence
// member must keep a reported value and an explicit null without renaming or
// dropping a field. The test encodes that member directly, so it stays green
// when the additive nodes wrapper is removed.
func TestDNS03_NodeDetailEvidenceDecodesHandlerJSON(t *testing.T) {
	var detail api.NodeDetail
	readJSON(t, "examples/valid/NodeDetail.json", &detail)
	var evidence api.DnsPolicyEvidence
	readJSON(t, "examples/valid/DnsPolicyEvidence.json", &evidence)
	encode := func(body api.NodeDetail) []byte {
		t.Helper()
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	populated := detail
	populated.DnsPolicyEvidence.Set(evidence)
	raw := encode(populated)
	if err := validateComponent(t, "NodeDetail", raw); err != nil {
		t.Fatal(err)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	want, err := canonjson.Canonicalize(mustMarshal(t, evidence))
	if err != nil {
		t.Fatal(err)
	}
	got, err := canonjson.Canonicalize(body["dns_policy_evidence"])
	if err != nil || !bytes.Equal(want, got) {
		t.Fatalf("projected evidence bytes differ\n got %s\nwant %s", got, want)
	}
	var decoded api.NodeDetail
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	reported, err := decoded.DnsPolicyEvidence.Get()
	if err != nil || !reflect.DeepEqual(reported, evidence) {
		t.Fatalf("generated type did not keep reported evidence: %v", err)
	}
	empty := detail
	empty.DnsPolicyEvidence.SetNull()
	raw = encode(empty)
	if err := validateComponent(t, "NodeDetail", raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte(`"dns_policy_evidence":null`)) {
		t.Fatalf("handler null evidence changed shape: %s", raw)
	}
	decoded = api.NodeDetail{}
	if err := json.Unmarshal(raw, &decoded); err != nil || !decoded.DnsPolicyEvidence.IsNull() {
		t.Fatal("explicit null evidence was not preserved", err)
	}
}

func TestUPD02_Revision6UpdateResultEvidence(t *testing.T) {
	var rolled api.UpdateResult
	readJSON(t, "examples/valid/UpdateResult.rolled-back.json", &rolled)
	if rolled.State != api.UpdateResultStateRolledBack || rolled.ErrorCode.IsNull() {
		t.Fatal("rolled_back example lost the candidate failure")
	}
	code, err := rolled.ErrorCode.Get()
	if err != nil || code != api.ErrorCodeHEALTHCHECKFAILED {
		t.Fatal("error_code must be the candidate failure")
	}
	current, err := rolled.CurrentReleaseId.Get()
	if err != nil {
		t.Fatal(err)
	}
	previous, err := rolled.PreviousReleaseId.Get()
	if err != nil || previous != rolled.RequestedReleaseId || previous == current {
		t.Fatal("rolled_back previous_release_id must be the candidate and current the restored release")
	}
	health, err := rolled.Health.Get()
	if err != nil || !healthPassed(health) {
		t.Fatal("rolled_back health must be the restored release's checks")
	}
	candidate, err := rolled.CandidateHealth.Get()
	if err != nil || candidate.TunnelProbe == health.TunnelProbe || candidate.TunnelProbe != api.HealthCheckStatusFailed {
		t.Fatal("candidate_health must keep the candidate failure")
	}
	var preflight api.UpdateResult
	readJSON(t, "examples/valid/UpdateResult.preflight-failed.json", &preflight)
	if preflight.State != api.UpdateResultStateFailed || preflight.Stage.IsNull() || !preflight.Health.IsNull() || !preflight.CandidateHealth.IsNull() {
		t.Fatal("preflight failure must keep health null")
	}
	stage, err := preflight.Stage.Get()
	if err != nil || stage != api.UpdateStageDownload {
		t.Fatal("preflight example must use a stage before checking")
	}
	var explicit api.UpdateResult
	readJSON(t, "examples/valid/UpdateResult.null-candidate-health.json", &explicit)
	if !explicit.CandidateHealth.IsNull() {
		t.Fatal("explicit null candidate_health was lost")
	}
}

func TestDNS03_ResolverBootstrapCopiedFromUpstreams(t *testing.T) {
	raw := readFile(t, "examples/valid/ResolverDocument.json")
	if bytes.HasSuffix(raw, []byte("\n")) {
		t.Fatal("resolver example must be the exact canonical bytes")
	}
	var doc dnspublish.Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if err := dnspublish.Verify(doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Upstreams) == 0 || !reflect.DeepEqual(doc.BootstrapDNS, doc.Upstreams) {
		t.Fatalf("bootstrap %v upstreams %v", doc.BootstrapDNS, doc.Upstreams)
	}
	compiled, err := dnspublish.Compile(dnspublish.Input{
		ProfileCode: doc.ProfileCode, BlockResponse: doc.BlockResponse, BlockedTTLSeconds: doc.BlockedTTLSeconds,
		SafeSearch: doc.SafeSearch, Upstreams: append([]string(nil), doc.Upstreams...),
		Rules: append([]dnspublish.Rule(nil), doc.Rules...), Feeds: append([]dnspublish.Feed(nil), doc.Feeds...),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(compiled.Bytes, raw) {
		t.Fatalf("example is not compiler output\n got %s\nwant %s", compiled.Bytes, raw)
	}
	empty := doc
	empty.BootstrapDNS = []string{}
	if dnspublish.Verify(empty) == nil {
		t.Fatal("empty bootstrap_dns with non-empty upstreams must be rejected")
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	var ref api.ResolverDocumentRef
	readJSON(t, "examples/valid/ResolverDocumentRef.json", &ref)
	if ref.ContentSha256 != digest || ref.SizeBytes != len(raw) {
		t.Fatal("resolver reference does not bind the compiler bytes")
	}
	var evidence api.DnsPolicyEvidence
	readJSON(t, "examples/valid/DnsPolicyEvidence.json", &evidence)
	if evidence.ContentSha256 != digest {
		t.Fatal("reported evidence digest does not match the compiler document")
	}
}

func healthPassed(h api.UpdateHealth) bool {
	return h.Processes == api.HealthCheckStatusPassed && h.IdentityAndState == api.HealthCheckStatusPassed &&
		h.InternetOnly == api.HealthCheckStatusPassed && h.DnsPolicy == api.HealthCheckStatusPassed &&
		h.TunnelProbe == api.HealthCheckStatusPassed
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
