package policies_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/internal/policies"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

func TestD058_FeedDefaultBeforeValidationAndHashing(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/ThreatFeedPayload.default-null-attribution.json")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	defaulted, err := policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, raw, policies.ValidationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if defaulted.ThreatFeed.UpdateIntervalSeconds != 21600 || !defaulted.ThreatFeed.Attribution.IsNull() {
		t.Fatalf("default/null lost: %+v", defaulted.ThreatFeed)
	}
	for _, value := range []string{"21600", "300", "604800", "0", "null", "299", "604801"} {
		t.Run(value, func(t *testing.T) {
			fields["update_interval_seconds"] = json.RawMessage(value)
			explicit, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			got, err := policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, explicit, policies.ValidationOptions{})
			valid := value == "21600" || value == "300" || value == "604800"
			if (err == nil) != valid {
				t.Fatalf("accepted=%v want %v: %v", err == nil, valid, err)
			}
			if value == "21600" && (got.PayloadSHA256 != defaulted.PayloadSHA256 || !bytes.Equal(got.CanonicalJSON, defaulted.CanonicalJSON)) {
				t.Fatal("omission and explicit default must persist/hash identically")
			}
		})
	}
	if hash, err := canonjson.Hash(defaulted.CanonicalJSON); err != nil || hash != defaulted.PayloadHash {
		t.Fatalf("persisted bytes/hash differ: %v", err)
	}
	// Do not hide a duplicate/invalid original representation while defaulting.
	for _, prefix := range []string{`"update_interval_seconds":0,"update_interval_seconds":21600,`, `"update_interval_seconds":2.16e4,`} {
		bad := append([]byte("{"+prefix), bytes.TrimSpace(raw)[1:]...)
		if _, err := policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, bad, policies.ValidationOptions{}); err == nil {
			t.Fatal("defaulting hid invalid original JSON")
		}
	}
}

func TestB9_AttributionVector(t *testing.T) {
	raw, err := os.ReadFile("../../contracts/examples/valid/ThreatFeedPayload.default-null-attribution.json")
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		t.Fatal(err)
	}
	valid, err := json.Marshal("TEST ONLY attribution")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		present bool
		value   json.RawMessage
		accept  bool
		blank   bool
	}{
		{name: "null", present: true, value: json.RawMessage("null"), accept: true},
		{name: "missing", accept: false},
		{name: "empty", present: true, value: json.RawMessage(`""`), blank: true},
		{name: "whitespace", present: true, value: json.RawMessage(`"   "`), blank: true},
		{name: "valid", present: true, value: valid, accept: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := make(map[string]json.RawMessage, len(object))
			for key, value := range object {
				fields[key] = value
			}
			if tc.present {
				fields["attribution"] = tc.value
			} else {
				delete(fields, "attribution")
			}
			body, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			got, err := policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, body, policies.ValidationOptions{})
			if tc.accept {
				if err != nil {
					t.Fatal(err)
				}
				if tc.name == "null" && !got.ThreatFeed.Attribution.IsNull() {
					t.Fatal("null attribution was not preserved")
				}
				if tc.name == "valid" {
					note, err := got.ThreatFeed.Attribution.Get()
					if err != nil || note != "TEST ONLY attribution" {
						t.Fatalf("valid note = %q %v", note, err)
					}
				}
				return
			}
			var ve *policies.ValidationError
			if !errors.As(err, &ve) || ve.Reason != policies.ReasonSchemaInvalid {
				t.Fatalf("attribution error = %v", err)
			}
			if tc.blank {
				if ve.Field != "/payload/attribution" || ve.Message != "attribution must be null or a non-blank note" {
					t.Fatalf("blank attribution = field %s message %s", ve.Field, ve.Message)
				}
				return
			}
			if ve.Field != "/payload" || !strings.Contains(ve.Message, "missing property 'attribution'") {
				t.Fatalf("missing attribution = field %s message %s", ve.Field, ve.Message)
			}
		})
	}

	// Null attribution must not skip the other source checks.
	for _, tc := range []struct {
		name   string
		key    string
		value  string
		reason policies.RuleReason
	}{
		{name: "scheme", key: "source_url", value: `"http://feeds.example.com/malware-domains.txt"`, reason: policies.ReasonThreatFeedInvalidURL},
		{name: "port", key: "source_url", value: `"https://feeds.example.com:8443/malware-domains.txt"`, reason: policies.ReasonThreatFeedInvalidURL},
		{name: "host", key: "source_url", value: `"https://127.0.0.1/malware-domains.txt"`, reason: policies.ReasonThreatFeedInvalidURL},
		{name: "size", key: "max_size_bytes", value: `"0"`, reason: policies.ReasonThreatFeedInvalidSize},
	} {
		t.Run("null_keeps_"+tc.name, func(t *testing.T) {
			fields := make(map[string]json.RawMessage, len(object))
			for key, value := range object {
				fields[key] = value
			}
			fields["attribution"] = json.RawMessage("null")
			fields[tc.key] = json.RawMessage(tc.value)
			body, err := json.Marshal(fields)
			if err != nil {
				t.Fatal(err)
			}
			_, err = policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, body, policies.ValidationOptions{})
			if !policies.HasReason(err, tc.reason) {
				t.Fatalf("got %v", err)
			}
		})
	}
	// Schema maxLength is the contract gate. The semantic length check is only
	// reached by ValidateThreatFeed callers that do not re-run the schema.
	long := strings.Repeat("a", 1001)
	encoded, err := json.Marshal(long)
	if err != nil {
		t.Fatal(err)
	}
	fields := make(map[string]json.RawMessage, len(object))
	for key, value := range object {
		fields[key] = value
	}
	fields["attribution"] = encoded
	body, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	_, err = policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, body, policies.ValidationOptions{})
	var ve *policies.ValidationError
	if !errors.As(err, &ve) || ve.Reason != policies.ReasonSchemaInvalid || ve.Field != "/payload" || !strings.Contains(ve.Message, "at '/attribution': maxLength: got 1,001, want 1,000") {
		t.Fatalf("length = %v", err)
	}
	accepted, err := policies.ValidatePayload(api.PolicyTypeThreatFeed, 1, raw, policies.ValidationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	direct := *accepted.ThreatFeed
	direct.Attribution.Set(long)
	err = policies.ValidateThreatFeed(direct)
	if !errors.As(err, &ve) || ve.Field != "/payload/attribution" || ve.Message != "attribution must not exceed 1000 characters" {
		t.Fatalf("direct length = %v", err)
	}
}
