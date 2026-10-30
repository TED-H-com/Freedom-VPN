package dnspolicy

import (
	"encoding/json"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// DefaultFeedUpdateIntervalSeconds is the D-058d revision default (待實測).
const DefaultFeedUpdateIntervalSeconds = 21600

// MaterializeFeedDefaults inserts the cadence only when absent. Explicit zero
// and null remain invalid inputs for validation. Check the original JSON before
// decoding so duplicate keys and other noncanonical input cannot be hidden by
// the map conversion. Call before validation, hashing or decoding stored feeds.
func MaterializeFeedDefaults(raw []byte) ([]byte, error) {
	checked, err := canonjson.Canonicalize(raw)
	if err != nil {
		return nil, err
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(checked, &payload); err != nil {
		return nil, err
	}
	if payload == nil {
		return nil, errors.New("feed payload must be an object")
	}
	if _, present := payload["update_interval_seconds"]; !present {
		payload["update_interval_seconds"], err = json.Marshal(DefaultFeedUpdateIntervalSeconds)
		if err != nil {
			return nil, err
		}
	}
	return canonjson.Marshal(payload)
}
