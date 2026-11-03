package agent

import (
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/node/enforce"
)

// TEST ONLY pure clock boundaries; no elapsed-time measurement.
func TestAT13_F41_PresenceAllowsOnlyTheExistingEarlyExpiryMargin(t *testing.T) {
	in := withAdmittedMember(servingInput())
	m, _, err := buildModel(in)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"nftables":[]}`)
	deadline := *m.Leases[0].DeadlineMono
	for _, tc := range []struct {
		left    time.Duration
		missing bool
	}{
		{time.Hour, true}, {102 * time.Millisecond, true}, {101 * time.Millisecond, false}, {time.Millisecond, false}, {0, false},
	} {
		err := verifyAllowedPresence(raw, m, deadline-tc.left)
		if (err != nil) != tc.missing {
			t.Fatalf("remaining=%s missing=%v err=%v", tc.left, tc.missing, err)
		}
	}
	// A prior-boot/otherwise incompatible monotonic sample can never verify.
	if err := verifyAllowedPresence(raw, m, m.NowMono-enforce.DeadlineGranularity); err == nil {
		t.Fatal("backwards anchor accepted")
	}
}
