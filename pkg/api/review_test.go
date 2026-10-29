package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/TED-H-com/Freedom-VPN/pkg/api/presence"
)

func TestReviewC03_WebhookRetainsOriginalBytes(t *testing.T) {
	raw := []byte(`{ "z":1, "a":2, "amount":9007199254740993 }`)
	var payload ProviderWebhookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, payload) {
		t.Fatalf("webhook bytes changed: %s", payload)
	}
	// Read and verify r.Body before this decode: even RawMessage remarshal may
	// compact whitespace or HTML-escape it. Its stored bytes remain untouched.
	if !bytes.Contains(payload, []byte("9007199254740993")) {
		t.Fatal("precision lost")
	}
}

func TestReviewC04_PlanRateLimitPresence(t *testing.T) {
	raw, err := os.ReadFile(examplesDir + "/PolicyRevisionCreate.plan.json")
	if err != nil {
		t.Fatal(err)
	}
	var request struct {
		Payload map[string]json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"absent", "null", `{"down_bps":100,"up_bps":50}`} {
		t.Run(value, func(t *testing.T) {
			if value == "absent" {
				delete(request.Payload, "rate_limit")
			} else {
				request.Payload["rate_limit"] = json.RawMessage(value)
			}
			input, err := json.Marshal(request.Payload)
			if err != nil {
				t.Fatal(err)
			}
			var plan PlanPayload
			if err := json.Unmarshal(input, &plan); err != nil {
				t.Fatal(err)
			}
			output, err := json.Marshal(plan)
			if value == "absent" {
				if plan.RateLimit.IsSpecified() || !errors.Is(err, presence.ErrUnspecified) {
					t.Fatalf("omission became valid: %s %v", output, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !plan.RateLimit.IsSpecified() || plan.RateLimit.IsNull() != (value == "null") {
				t.Fatal("presence lost")
			}
			var round map[string]json.RawMessage
			if err := json.Unmarshal(output, &round); err != nil {
				t.Fatal(err)
			}
			if string(round["rate_limit"]) != value {
				t.Fatalf("changed value: %s", round["rate_limit"])
			}
		})
	}
}

func TestReviewC06_RecoveryRequiredPresence(t *testing.T) {
	raw, err := os.ReadFile(examplesDir + "/UpdateResult.json")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"absent", "null", "false", "true"} {
		if value == "absent" {
			delete(fields, "recovery_required")
		} else {
			fields["recovery_required"] = json.RawMessage(value)
		}
		input, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		var result UpdateResult
		err = json.Unmarshal(input, &result)
		if value == "null" {
			if !errors.Is(err, presence.ErrNull) {
				t.Fatalf("null: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		output, err := json.Marshal(result)
		if value == "absent" {
			if !errors.Is(err, presence.ErrUnspecified) {
				t.Fatalf("omission accepted: %s %v", output, err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if got, err := result.RecoveryRequired.Get(); err != nil || got != (value == "true") {
			t.Fatalf("value lost: %v %v", got, err)
		}
		var round map[string]json.RawMessage
		if err := json.Unmarshal(output, &round); err != nil {
			t.Fatal(err)
		}
		if string(round["recovery_required"]) != value {
			t.Fatal("required bool lost on remarshal")
		}
	}
}
