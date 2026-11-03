package members

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type sendAttempt struct {
	Sequence  decstr.U64    `json:"sequence"`
	Before    time.Duration `json:"mono_before_ns,string"`
	After     time.Duration `json:"mono_after_ns,string"`
	UTCBefore time.Time     `json:"utc_before"`
	UTCAfter  time.Time     `json:"utc_after"`
	Bytes     int           `json:"bytes"`
	OK        bool          `json:"ok"`
}

func validateSendTimes(raw, observed json.RawMessage) error {
	var s struct {
		Boot     string        `json:"boot_id"`
		Clock    string        `json:"clock"`
		Attempts int           `json:"attempts"`
		Errors   int           `json:"send_errors"`
		Sends    []sendAttempt `json:"sends"`
	}
	var o siblingObservation
	if json.Unmarshal(raw, &s) != nil || json.Unmarshal(observed, &o) != nil || s.Boot == "" || s.Boot != o.Boot || s.Clock != "CLOCK_MONOTONIC" || s.Attempts == 0 || len(s.Sends) != s.Attempts {
		return fmt.Errorf("missing complete same-boot send ledger")
	}
	errors := 0
	for i, a := range s.Sends {
		if a.Sequence != decstr.U64(i) || a.Before < o.Start || a.After < a.Before || a.After > o.End || a.UTCBefore.IsZero() || a.UTCAfter.Before(a.UTCBefore) || (i > 0 && a.Before < s.Sends[i-1].After) {
			return fmt.Errorf("invalid send bounds at %d", i)
		}
		if !a.OK {
			errors++
		}
	}
	if errors != s.Errors {
		return fmt.Errorf("send error count mismatch")
	}
	return nil
}
func TestAT02_EverySendHasBoundsEvenOnWriteFailure(t *testing.T) {
	raw := json.RawMessage(`{"boot_id":"TEST ONLY boot","clock":"CLOCK_MONOTONIC","attempts":2,"send_errors":1,"sends":[{"sequence":"0","mono_before_ns":"10","mono_after_ns":"11","utc_before":"2026-11-01T00:00:00Z","utc_after":"2026-11-01T00:00:00Z","ok":true},{"sequence":"1","mono_before_ns":"20","mono_after_ns":"21","utc_before":"2026-11-01T00:00:01Z","utc_after":"2026-11-01T00:00:01Z","ok":false}]}`)
	obs := json.RawMessage(`{"boot_id":"TEST ONLY boot","start_ns":"1","end_ns":"30"}`)
	if err := validateSendTimes(raw, obs); err != nil {
		t.Fatal(err)
	}
	var s map[string]any
	_ = json.Unmarshal(raw, &s)
	for _, field := range []string{"sends", "boot_id", "attempts", "clock"} {
		original := s[field]
		delete(s, field)
		bad, _ := json.Marshal(s)
		if validateSendTimes(bad, obs) == nil {
			t.Fatal("accepted absent", field)
		}
		s[field] = original
	}
	s["send_errors"] = 0
	bad, _ := json.Marshal(s)
	if validateSendTimes(bad, obs) == nil {
		t.Fatal("hid write failure")
	}
}
