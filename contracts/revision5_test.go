package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/netip"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

// §12.3, §14.7 / D-056 and D-058: omission and null cannot fabricate evidence.
func TestDNS03_Revision5NullablePresence(t *testing.T) {
	for _, tc := range []struct {
		component, field, goField string
		model, nullable           any
	}{
		{"DnsServiceConfig", "resolver_document", "ResolverDocument", api.DnsServiceConfig{}, presence.Nullable[api.ResolverDocumentRef]{}},
		{"DesiredState", "egress_probe", "EgressProbe", api.DesiredState{}, presence.Nullable[api.EgressProbe]{}},
		{"ObservationReport", "dns_policy_evidence", "DnsPolicyEvidence", api.ObservationReport{}, presence.Nullable[api.DnsPolicyEvidence]{}},
		{"PolicyDefinition", "threat_feed_status", "ThreatFeedStatus", api.PolicyDefinition{}, presence.Nullable[api.ThreatFeedStatus]{}},
	} {
		t.Run(tc.component, func(t *testing.T) {
			schema := loadDoc(t).Components.Schemas[tc.component].Value
			if slices.Contains(schema.Required, tc.field) || !schema.Properties[tc.field].Value.Nullable {
				t.Fatal("member must be optional and nullable")
			}
			f, ok := reflect.TypeOf(tc.model).FieldByName(tc.goField)
			if !ok || f.Type != reflect.TypeOf(tc.nullable) || !strings.Contains(f.Tag.Get("json"), "omitzero") {
				t.Fatalf("incorrect presence type/tag: %v", f)
			}
		})
	}
	for _, tc := range []struct {
		file            string
		specified, null bool
	}{
		{"DesiredStateEnvelope.json", false, false},
		{"DesiredStateEnvelope.resolver-probe.json", true, false},
		{"DesiredStateEnvelope.no-profile-probe.json", true, true},
	} {
		var state api.DesiredStateEnvelope
		readJSON(t, "examples/valid/"+tc.file, &state)
		r, p := state.Signed.Dns.ResolverDocument, state.Signed.EgressProbe
		if r.IsSpecified() != tc.specified || p.IsSpecified() != tc.specified || r.IsNull() != tc.null || p.IsNull() != tc.null {
			t.Fatalf("presence lost: %s", tc.file)
		}
	}
}

// D-056: the signed reference binds exact external bytes, not re-encoded JSON.
func TestDNS03_ResolverDocumentVectorBinding(t *testing.T) {
	var state api.DesiredStateEnvelope
	readJSON(t, "examples/valid/DesiredStateEnvelope.resolver-probe.json", &state)
	ref, err := state.Signed.Dns.ResolverDocument.Get()
	if err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, "examples/valid/ResolverDocument.json")
	canonical, err := canonjson.Canonicalize(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		t.Fatal("resolver bytes must already be canonical")
	}
	sum := sha256.Sum256(raw)
	if hex.EncodeToString(sum[:]) != ref.ContentSha256 || len(raw) != ref.SizeBytes || ref.Generation != decstr.MustParse("9007199254740993") {
		t.Fatal("signed reference does not bind document bytes/generation")
	}
	changed := sha256.Sum256(append(bytes.Clone(raw), '\n'))
	if hex.EncodeToString(changed[:]) == ref.ContentSha256 {
		t.Fatal("newline must change the digest")
	}
}

// §13.9: reject hostnames, zones, CIDRs, ports and malformed IPs even when
// an OpenAPI consumer does not implement format validators.
func TestISO01_SignedEgressProbeBounds(t *testing.T) {
	valid := []string{"192.0.2.1", "2001:db8::1", "::", "::1", "2001:db8::", "1:2:3:4:5:6:7:8", "::ffff:192.0.2.1", "1:2:3:4:5:6:192.0.2.1", "2001:db8::192.0.2.1"}
	invalid := []string{"probe.example.invalid", "192.0.2.256", "192.000.2.1", "192.0.2.1:443", "[2001:db8::1]", "fe80::1%eth0", "2001:db8::1/64", "1:2:3:4:5:6:7:8:9", "1::2::3", "::ffff:192.0.2.999", ""}
	schema := loadDoc(t).Components.Schemas["EgressProbeTarget"].Value
	for _, address := range append(valid, invalid...) {
		t.Run(address, func(t *testing.T) {
			parsed, parseErr := netip.ParseAddr(address)
			want := slices.Contains(valid, address)
			if (parseErr == nil && parsed.Zone() == "") != want {
				t.Fatal("bad IP test expectation")
			}
			v := map[string]any{"address": address, "port": float64(443), "protocol": "tcp"}
			if (schema.VisitJSON(v) == nil) != want {
				t.Fatalf("IP literal validation: %q", address)
			}
		})
	}
	// Exercise every compression position, including dotted-quad suffixes.
	for left := 0; left < 8; left++ {
		for right := 0; left+right < 8; right++ {
			addr := strings.TrimSuffix(strings.Repeat("abcd:", left), ":") + "::" + strings.TrimSuffix(strings.Repeat("1234:", right), ":")
			v := map[string]any{"address": addr, "port": float64(443), "protocol": "tcp"}
			if err := schema.VisitJSON(v); err != nil {
				t.Fatalf("compressed IPv6 %s: %v", addr, err)
			}
		}
	}
	var probe map[string]any
	readJSON(t, "examples/valid/EgressProbe.json", &probe)
	for _, tc := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"empty", func(p map[string]any) { p["targets"] = []any{} }},
		{"duplicate", func(p map[string]any) { x := p["targets"].([]any)[0]; p["targets"] = []any{x, x} }},
		{"five", func(p map[string]any) {
			var v []any
			for i := 1; i <= 5; i++ {
				v = append(v, map[string]any{"address": fmt.Sprintf("192.0.2.%d", i), "port": 443, "protocol": "tcp"})
			}
			p["targets"] = v
		}},
		{"timeout-low", func(p map[string]any) { p["timeout_ms"] = 99 }},
		{"timeout-high", func(p map[string]any) { p["timeout_ms"] = 10001 }},
		{"port-zero", func(p map[string]any) { p["targets"].([]any)[0].(map[string]any)["port"] = 0 }},
		{"port-high", func(p map[string]any) { p["targets"].([]any)[0].(map[string]any)["port"] = 65536 }},
		{"udp", func(p map[string]any) { p["targets"].([]any)[0].(map[string]any)["protocol"] = "udp" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			readJSON(t, "examples/valid/EgressProbe.json", &probe)
			tc.edit(probe)
			raw, _ := json.Marshal(probe)
			if validateComponent(t, "EgressProbe", raw) == nil {
				t.Fatal("accepted invalid probe")
			}
		})
	}
}

func TestAPI01_GroupMembershipOmittedVersusEmpty(t *testing.T) {
	for _, tc := range []struct {
		file        string
		wantPointer bool
		wantLen     int
	}{
		{"PublicationRequest.json", false, 0}, {"PublicationRequest.groups.json", true, 1}, {"PublicationRequest.clear-groups.json", true, 0},
	} {
		var req api.PublicationRequest
		readJSON(t, "examples/valid/"+tc.file, &req)
		if (req.GroupIds != nil) != tc.wantPointer {
			t.Fatal("membership presence lost")
		}
		if req.GroupIds != nil && len(*req.GroupIds) != tc.wantLen {
			t.Fatal("membership content changed")
		}
	}
}

func TestDNS03_FeedDefaultAndNullableAttribution(t *testing.T) {
	schema := loadDoc(t).Components.Schemas["ThreatFeedPayload"].Value
	if slices.Contains(schema.Required, "update_interval_seconds") || fmt.Sprint(schema.Properties["update_interval_seconds"].Value.Default) != "21600" {
		t.Fatal("wrong feed default/optionality")
	}
	raw := readFile(t, "examples/valid/ThreatFeedPayload.default-null-attribution.json")
	if err := validateComponent(t, "ThreatFeedPayload", raw); err != nil {
		t.Fatal(err)
	}
	if err := validateJSONSchema(t, "policies/threat-feed-payload.schema.json", raw); err != nil {
		t.Fatal(err)
	}
	var feed api.ThreatFeedPayload
	if err := json.Unmarshal(raw, &feed); err != nil || !feed.Attribution.IsNull() {
		t.Fatalf("nullable attribution: %v", err)
	}
	// Defaults are annotations, not generated JSON-decoder behavior. The policy
	// owner must materialize 21600 before invoking existing domain validation.
	if feed.UpdateIntervalSeconds != 0 {
		t.Fatal("unexpected decoder default")
	}
}

func TestAPI01_Revision5ResolverOperation(t *testing.T) {
	doc := loadDoc(t)
	if doc.Info.Version != "1.1.0" {
		t.Fatal("API version must be 1.1.0")
	}
	op := doc.Paths.Find("/agent/resolver-documents/{document_id}").Get
	if op.RequestBody != nil || op.Extensions["x-fvpn-auth"] != "agent-mtls" {
		t.Fatal("resolver read requires mTLS and no body")
	}
	headers := op.Responses.Status(200).Value.Headers
	if headers["ETag"].Value.Schema.Value.Pattern != `^"sha256:[0-9a-f]{64}"$` || headers["Cache-Control"].Value.Schema.Value.Enum[0] != "private, max-age=0" {
		t.Fatal("missing digest/cache contract")
	}
	if apierr.NotFound.HTTPStatus() != 404 || apierr.New(apierr.NotFound, "").IsRetryable() {
		t.Fatal("D-056 NOT_FOUND must be non-retryable 404")
	}
}

// §12.3 / D-056, D-058: evidence limits apply independently of signature
// verification. Raw generation numbers and internal-only errors are rejected.
func TestDNS03_ReportedEvidenceBounds(t *testing.T) {
	for _, outcome := range []string{"applied", "partial", "failed", "not_run"} {
		var evidence map[string]any
		readJSON(t, "examples/valid/DnsPolicyEvidence.json", &evidence)
		evidence["outcome"] = outcome
		if outcome != "applied" {
			evidence["applied_at"] = nil
			evidence["error_code"] = "HASH_MISMATCH"
		}
		raw, _ := json.Marshal(evidence)
		if err := validateComponent(t, "DnsPolicyEvidence", raw); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name  string
		edit  func(map[string]any)
		valid bool
	}{
		{"max_uint64", func(e map[string]any) { e["generation"] = "18446744073709551615" }, true},
		{"numeric_generation", func(e map[string]any) { e["generation"] = 1 }, false},
		{"unknown_outcome", func(e map[string]any) { e["outcome"] = "success" }, false},
		{"internal_reason", func(e map[string]any) { e["error_code"] = "fetch_failed" }, false},
		{"missing_error", func(e map[string]any) { delete(e, "error_code") }, false},
		{"missing_feeds", func(e map[string]any) { delete(e, "feeds") }, false},
		{"uppercase_digest", func(e map[string]any) { e["content_sha256"] = strings.Repeat("A", 64) }, false},
		{"33_feeds", func(e map[string]any) {
			var feeds []any
			for i := 0; i < 33; i++ {
				feeds = append(feeds, map[string]any{"feed_code": fmt.Sprintf("feed-%d", i), "content_sha256": strings.Repeat("a", 64), "rule_count": 0})
			}
			e["feeds"] = feeds
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var evidence map[string]any
			readJSON(t, "examples/valid/DnsPolicyEvidence.json", &evidence)
			tc.edit(evidence)
			raw, _ := json.Marshal(evidence)
			if (validateComponent(t, "DnsPolicyEvidence", raw) == nil) != tc.valid {
				t.Fatal("incorrect evidence validation")
			}
		})
	}
	// uint64 overflow is enforced by the generated decstr decoder beyond regex.
	var evidence api.DnsPolicyEvidence
	raw := bytes.Replace(readFile(t, "examples/valid/DnsPolicyEvidence.json"), []byte("9007199254740993"), []byte("18446744073709551616"), 1)
	if json.Unmarshal(raw, &evidence) == nil {
		t.Fatal("accepted uint64 overflow")
	}
	var status api.ThreatFeedStatus
	readJSON(t, "examples/valid/ThreatFeedStatus.bootstrap-empty.json", &status)
	if !status.LastGoodAgeSeconds.IsNull() || !status.LastGoodRuleCount.IsNull() || !status.LastGoodContentSha256.IsNull() {
		t.Fatal("bootstrap unknowns became zero/empty")
	}
}
