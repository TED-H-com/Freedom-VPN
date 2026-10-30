package policies_test

import (
	"bytes"
	"encoding/json"
	"os"
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
